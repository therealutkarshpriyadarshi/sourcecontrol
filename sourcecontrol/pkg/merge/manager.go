package merge

import (
	"context"
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/refs/branch"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// Manager orchestrates merge operations and selects appropriate strategies
type Manager struct {
	repo           *sourcerepo.SourceRepository
	branchMgr      *branch.Manager
	commitMgr      *commitmanager.Manager
	baseCalculator *MergeBaseCalculator

	// Available merge strategies
	fastForward *FastForwardMerger
	threeWay    *ThreeWayMerger
	recursive   *RecursiveMerger
	octopus     *OctopusMerger
	squash      *SquashMerger
}

// NewManager creates a new merge manager
func NewManager(repo *sourcerepo.SourceRepository) *Manager {
	return &Manager{
		repo:           repo,
		branchMgr:      branch.NewManager(repo),
		commitMgr:      commitmanager.NewManager(repo),
		baseCalculator: NewMergeBaseCalculator(repo),
		fastForward:    NewFastForwardMerger(repo),
		threeWay:       NewThreeWayMerger(repo),
		recursive:      NewRecursiveMerger(repo),
		octopus:        NewOctopusMerger(repo),
		squash:         NewSquashMerger(repo),
	}
}

// Initialize initializes the merge manager
func (m *Manager) Initialize(ctx context.Context) error {
	if err := m.commitMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize commit manager: %w", err)
	}
	return nil
}

// Merge performs a merge operation with the given configuration
// branches is a list of branch names or commit SHAs to merge into the current branch
func (m *Manager) Merge(ctx context.Context, branches []string, opts ...MergeOption) (*MergeResult, error) {
	if len(branches) == 0 {
		return nil, fmt.Errorf("no branches specified for merge")
	}

	// Build configuration
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	// Initialize
	if err := m.Initialize(ctx); err != nil {
		return nil, err
	}

	// Get current commit
	currentSHA, err := m.branchMgr.CurrentCommit()
	if err != nil {
		return nil, fmt.Errorf("failed to get current commit: %w", err)
	}

	ourCommit, err := m.commitMgr.GetCommit(ctx, currentSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get our commit: %w", err)
	}

	// Resolve branch names to commits
	theirCommits, err := m.resolveBranches(ctx, branches)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve branches: %w", err)
	}

	// Find merge base for the first branch
	var baseCommit *commit.Commit
	if len(theirCommits) > 0 {
		ourSHA, _ := ourCommit.Hash()
		theirSHA, _ := theirCommits[0].Hash()

		base, err := m.baseCalculator.FindMergeBase(ctx, ourSHA, theirSHA)
		if err != nil && !config.AllowUnrelatedHistories {
			return nil, fmt.Errorf("no merge base found: %w", err)
		}
		baseCommit = base
	}

	// Create merge context
	mergeCtx := &MergeContext{
		Ctx:          ctx,
		OurCommit:    ourCommit,
		TheirCommits: theirCommits,
		BaseCommit:   baseCommit,
		Config:       config,
	}

	// Select and execute merge strategy
	return m.executeMerge(mergeCtx)
}

// MergeBranch is a convenience method to merge a single branch
func (m *Manager) MergeBranch(ctx context.Context, branch string, opts ...MergeOption) (*MergeResult, error) {
	return m.Merge(ctx, []string{branch}, opts...)
}

// executeMerge selects the appropriate merge strategy and executes it
func (m *Manager) executeMerge(mergeCtx *MergeContext) (*MergeResult, error) {
	// Select strategy based on configuration and context
	var merger Merger

	// Check for squash merge first
	if mergeCtx.Config.Mode == ModeSquash {
		merger = m.squash
	} else if len(mergeCtx.TheirCommits) > 1 {
		// Multiple branches - use octopus
		if mergeCtx.Config.Strategy == StrategyOctopus {
			merger = m.octopus
		} else {
			return nil, fmt.Errorf("multiple branches require octopus strategy")
		}
	} else {
		// Single branch merge
		switch mergeCtx.Config.Mode {
		case ModeFastForwardOnly:
			// Only allow fast-forward
			if !m.fastForward.CanMerge(mergeCtx) {
				return nil, fmt.Errorf("fast-forward merge not possible")
			}
			merger = m.fastForward

		case ModeNoFastForward:
			// Never fast-forward, always create merge commit
			merger = m.selectNonFastForwardStrategy(mergeCtx)

		default:
			// Try fast-forward first, fall back to recursive
			if m.fastForward.CanMerge(mergeCtx) {
				merger = m.fastForward
			} else {
				merger = m.selectNonFastForwardStrategy(mergeCtx)
			}
		}
	}

	if merger == nil {
		return nil, fmt.Errorf("no suitable merge strategy found")
	}

	// Verify the merger can handle this merge
	if !merger.CanMerge(mergeCtx) {
		return nil, fmt.Errorf("selected strategy %s cannot handle this merge", merger.Name())
	}

	// Execute the merge
	result, err := merger.Merge(mergeCtx)
	if err != nil {
		return nil, fmt.Errorf("merge failed with strategy %s: %w", merger.Name(), err)
	}

	return result, nil
}

// selectNonFastForwardStrategy selects the appropriate non-fast-forward strategy
func (m *Manager) selectNonFastForwardStrategy(mergeCtx *MergeContext) Merger {
	switch mergeCtx.Config.Strategy {
	case StrategyRecursive:
		return m.recursive
	case StrategyOctopus:
		return m.octopus
	default:
		// Default to recursive
		return m.recursive
	}
}

// resolveBranches resolves branch names or SHAs to commit objects
func (m *Manager) resolveBranches(ctx context.Context, branches []string) ([]*commit.Commit, error) {
	commits := make([]*commit.Commit, 0, len(branches))

	for _, branch := range branches {
		commitSHA, err := m.resolveBranchToSHA(branch)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", branch, err)
		}

		commit, err := m.commitMgr.GetCommit(ctx, commitSHA)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit for %s: %w", branch, err)
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

// resolveBranchToSHA resolves a branch name or SHA to a commit SHA
func (m *Manager) resolveBranchToSHA(branchOrSHA string) (objects.ObjectHash, error) {
	// Try as branch name first
	exists, err := m.branchMgr.BranchExists(branchOrSHA)
	if err == nil && exists {
		branchInfo, err := m.branchMgr.GetBranch(context.Background(), branchOrSHA)
		if err != nil {
			return "", err
		}
		return branchInfo.SHA, nil
	}

	// Try as SHA
	sha, err := objects.NewObjectHashFromString(branchOrSHA)
	if err != nil {
		return "", fmt.Errorf("'%s' is not a valid branch name or commit SHA", branchOrSHA)
	}

	return sha, nil
}

// CanFastForward checks if a fast-forward merge is possible
func (m *Manager) CanFastForward(ctx context.Context, fromBranch, toBranch string) (bool, error) {
	fromSHA, err := m.resolveBranchToSHA(fromBranch)
	if err != nil {
		return false, err
	}

	toSHA, err := m.resolveBranchToSHA(toBranch)
	if err != nil {
		return false, err
	}

	return m.baseCalculator.CanFastForward(ctx, fromSHA, toSHA)
}

// FindMergeBase finds the common ancestor between two branches
func (m *Manager) FindMergeBase(ctx context.Context, branch1, branch2 string) (*commit.Commit, error) {
	sha1, err := m.resolveBranchToSHA(branch1)
	if err != nil {
		return nil, err
	}

	sha2, err := m.resolveBranchToSHA(branch2)
	if err != nil {
		return nil, err
	}

	return m.baseCalculator.FindMergeBase(ctx, sha1, sha2)
}

// AbortMerge aborts an in-progress merge
func (m *Manager) AbortMerge(ctx context.Context) error {
	// Check if a merge is in progress
	mergeState := NewMergeState(m.repo)
	if !mergeState.InProgress() {
		return fmt.Errorf("no merge in progress")
	}

	// Clear merge state files
	if err := mergeState.ClearState(); err != nil {
		return fmt.Errorf("failed to clear merge state: %w", err)
	}

	// TODO: Restore index and working directory to pre-merge state
	// This requires additional repository methods

	return nil
}

// ContinueMerge continues a merge after resolving conflicts
func (m *Manager) ContinueMerge(ctx context.Context) error {
	// Check if a merge is in progress
	mergeState := NewMergeState(m.repo)
	if !mergeState.InProgress() {
		return fmt.Errorf("no merge in progress")
	}

	// Get merge message
	mergeMsg, err := mergeState.GetMergeMessage()
	if err != nil {
		return fmt.Errorf("failed to get merge message: %w", err)
	}

	// Get MERGE_HEAD
	mergeHead, err := mergeState.GetMergeHead()
	if err != nil {
		return fmt.Errorf("failed to get MERGE_HEAD: %w", err)
	}

	// Get current HEAD
	currentHead, err := m.branchMgr.CurrentCommit()
	if err != nil {
		return fmt.Errorf("failed to get current HEAD: %w", err)
	}

	// TODO: Create merge commit with both parents
	// parents := []objects.ObjectHash{currentHead, mergeHead}
	// commitHash, err := m.commitMgr.CreateCommitFromIndex(ctx, mergeMsg, parents)
	// This requires CreateCommitFromIndex method
	_ = mergeMsg
	_ = currentHead
	_ = mergeHead

	// Clear merge state
	if err := mergeState.ClearState(); err != nil {
		return fmt.Errorf("failed to clear merge state: %w", err)
	}

	return nil
}
