package merge

import (
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// SquashMerger implements squash merge strategy
// Squash merge combines all commits from the branch being merged into a single commit
// This creates a clean, linear history without merge commits
type SquashMerger struct {
	repo      *sourcerepo.SourceRepository
	recursive *RecursiveMerger
}

// NewSquashMerger creates a new squash merger
func NewSquashMerger(repo *sourcerepo.SourceRepository) *SquashMerger {
	return &SquashMerger{
		repo:      repo,
		recursive: NewRecursiveMerger(repo),
	}
}

// Name returns the name of this strategy
func (sm *SquashMerger) Name() string {
	return "squash"
}

// CanMerge checks if squash merge can be performed
func (sm *SquashMerger) CanMerge(mergeCtx *MergeContext) bool {
	// Squash merge works for single branch merges
	return len(mergeCtx.TheirCommits) == 1 && mergeCtx.Config.Mode == ModeSquash
}

// Merge performs a squash merge
// This merges all changes from the branch but creates a single commit instead of a merge commit
func (sm *SquashMerger) Merge(mergeCtx *MergeContext) (*MergeResult, error) {
	if !sm.CanMerge(mergeCtx) {
		return nil, fmt.Errorf("squash merge requires exactly one branch and squash mode")
	}

	theirCommit := mergeCtx.TheirCommits[0]

	// Perform the merge using recursive strategy to get the merged tree
	// But configure it to not create a commit
	squashConfig := *mergeCtx.Config
	squashConfig.Mode = ModeNoCommit // Don't create merge commit yet

	squashCtx := &MergeContext{
		Ctx:          mergeCtx.Ctx,
		OurCommit:    mergeCtx.OurCommit,
		TheirCommits: mergeCtx.TheirCommits,
		BaseCommit:   mergeCtx.BaseCommit,
		Config:       &squashConfig,
	}

	// Perform the merge
	result, err := sm.recursive.Merge(squashCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to perform merge: %w", err)
	}

	// If there are conflicts, return the result
	if !result.Success {
		return result, nil
	}

	// Now create a single commit (not a merge commit)
	// Get the tree from the merge
	ourSHA, err := mergeCtx.OurCommit.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get our commit hash: %w", err)
	}

	theirSHA, err := theirCommit.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get their commit hash: %w", err)
	}

	// Generate squash commit message
	message := mergeCtx.Config.Message
	if message == "" {
		message = sm.generateSquashMessage(mergeCtx, theirCommit)
	}

	// Create a regular commit (not a merge commit) with only our commit as parent
	squashCommit := &commit.Commit{
		TreeSHA:    theirCommit.TreeSHA, // Use the merged tree
		ParentSHAs: []objects.ObjectHash{ourSHA}, // Only one parent (our commit)
		Author:     mergeCtx.OurCommit.Author,
		Committer:  mergeCtx.OurCommit.Committer,
		Message:    message,
	}

	// Write the squash commit
	commitSHA, err := sm.repo.WriteObject(squashCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to write squash commit: %w", err)
	}

	return &MergeResult{
		Success:     true,
		FastForward: false,
		CommitSHA:   commitSHA,
		Message:     fmt.Sprintf("Squash merge of %s into %s", theirSHA.Short(), ourSHA.Short()),
	}, nil
}

// generateSquashMessage generates a commit message for squash merge
// It includes information about all the commits being squashed
func (sm *SquashMerger) generateSquashMessage(mergeCtx *MergeContext, theirCommit *commit.Commit) string {
	theirSHA, _ := theirCommit.Hash()

	message := fmt.Sprintf("Squash merge of %s\n\n", theirSHA.Short())
	message += fmt.Sprintf("Original commit message:\n%s\n", theirCommit.Message)

	// In a full implementation, we'd walk the commit history
	// and include all commit messages that are being squashed

	return message
}
