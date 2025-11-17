package merge

import (
	"context"
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/refs/branch"
	"github.com/utkarsh5026/SourceControl/pkg/repository/refs"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/workdir"
)

// FastForwardMerger implements fast-forward merge strategy
type FastForwardMerger struct {
	repo           *sourcerepo.SourceRepository
	baseCalculator *MergeBaseCalculator
}

// NewFastForwardMerger creates a new fast-forward merger
func NewFastForwardMerger(repo *sourcerepo.SourceRepository) *FastForwardMerger {
	return &FastForwardMerger{
		repo:           repo,
		baseCalculator: NewMergeBaseCalculator(repo),
	}
}

// Name returns the name of this strategy
func (ffm *FastForwardMerger) Name() string {
	return "fast-forward"
}

// CanMerge checks if a fast-forward merge is possible
func (ffm *FastForwardMerger) CanMerge(mergeCtx *MergeContext) bool {
	// Can only fast-forward when merging a single branch
	if len(mergeCtx.TheirCommits) != 1 {
		return false
	}

	ourSHA, err := mergeCtx.OurCommit.Hash()
	if err != nil {
		return false
	}

	theirSHA, err := mergeCtx.TheirCommits[0].Hash()
	if err != nil {
		return false
	}

	// Check if our commit is an ancestor of their commit
	canFF, err := ffm.baseCalculator.CanFastForward(mergeCtx.Ctx, ourSHA, theirSHA)
	if err != nil {
		return false
	}

	return canFF
}

// Merge performs a fast-forward merge
func (ffm *FastForwardMerger) Merge(mergeCtx *MergeContext) (*MergeResult, error) {
	if len(mergeCtx.TheirCommits) != 1 {
		return nil, fmt.Errorf("fast-forward merge requires exactly one commit to merge")
	}

	theirCommit := mergeCtx.TheirCommits[0]
	theirSHA, err := theirCommit.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get their commit hash: %w", err)
	}

	ourSHA, err := mergeCtx.OurCommit.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get our commit hash: %w", err)
	}

	// Verify fast-forward is possible
	canFF, err := ffm.baseCalculator.CanFastForward(mergeCtx.Ctx, ourSHA, theirSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to check fast-forward: %w", err)
	}

	if !canFF {
		return nil, fmt.Errorf("fast-forward is not possible")
	}

	// If already up to date
	if ourSHA.Equal(theirSHA) {
		return &MergeResult{
			Success:     true,
			FastForward: true,
			CommitSHA:   theirSHA,
			Message:     "Already up to date",
		}, nil
	}

	// Update HEAD to point to their commit
	if err := ffm.updateHead(mergeCtx.Ctx, theirSHA); err != nil {
		return nil, fmt.Errorf("failed to update HEAD: %w", err)
	}

	// Update working directory
	workdirMgr := workdir.NewManager(ffm.repo)
	_, err = workdirMgr.UpdateToCommit(mergeCtx.Ctx, theirSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to update working directory: %w", err)
	}

	return &MergeResult{
		Success:     true,
		FastForward: true,
		CommitSHA:   theirSHA,
		Message:     fmt.Sprintf("Fast-forward to %s", theirSHA.Short()),
	}, nil
}

// updateHead updates HEAD to point to the target commit
func (ffm *FastForwardMerger) updateHead(ctx context.Context, targetSHA objects.ObjectHash) error {
	branchMgr := branch.NewManager(ffm.repo)
	currentBranch, err := branchMgr.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	refMgr := refs.NewRefManager(ffm.repo)

	if currentBranch != "" {
		// Update the branch reference
		branchRef := refs.RefPath(fmt.Sprintf("refs/heads/%s", currentBranch))
		if err := refMgr.UpdateRef(branchRef, targetSHA); err != nil {
			return fmt.Errorf("failed to update branch %s: %w", currentBranch, err)
		}
	} else {
		// Detached HEAD - update HEAD directly
		if err := refMgr.UpdateRef(refs.RefPath("HEAD"), targetSHA); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}
	}

	return nil
}
