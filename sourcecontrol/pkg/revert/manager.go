package revert

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/common/logger"
	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/refs/branch"
	"github.com/utkarsh5026/SourceControl/pkg/repository/refs"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/workdir"
)

// Manager handles the revert operations for commits.
//
// The revert process follows these steps:
//  1. Resolve the commit(s) to revert
//  2. For each commit, get its parent commit
//  3. Apply the inverse changes (parent tree vs commit tree)
//  4. Create a new commit with the inverse changes
//  5. Update HEAD to point to the new commit
//
// Thread Safety:
// Manager is not thread-safe. External synchronization is required when
// accessing a Manager instance from multiple goroutines.
type Manager struct {
	repo          *sourcerepo.SourceRepository
	commitManager *commitmanager.Manager
	branchManager *branch.Manager
	refManager    *refs.RefManager
	workdirMgr    *workdir.Manager
	logger        *slog.Logger
}

// NewManager creates a new RevertManager instance
//
// Example:
//
//	repo := sourcerepo.NewSourceRepository()
//	repo.Initialize(scpath.RepositoryPath("/path/to/repo"))
//	mgr := revert.NewManager(repo)
func NewManager(repo *sourcerepo.SourceRepository) *Manager {
	return &Manager{
		repo:          repo,
		commitManager: commitmanager.NewManager(repo),
		branchManager: branch.NewManager(repo),
		refManager:    refs.NewRefManager(repo),
		workdirMgr:    workdir.NewManager(repo),
		logger:        logger.With("component", "revertmanager"),
	}
}

// Initialize initializes the revert manager by initializing dependent managers.
//
// This should be called once after creating a new Manager instance.
func (m *Manager) Initialize(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.logger.Info("initializing revert manager")

	// Initialize commit manager
	if err := m.commitManager.Initialize(ctx); err != nil {
		m.logger.Error("failed to initialize commit manager", "error", err)
		return fmt.Errorf("init commit manager: %w", err)
	}

	m.logger.Info("revert manager initialized successfully")
	return nil
}

// RevertOptions contains options for the revert operation
type RevertOptions struct {
	// NoCommit stages the changes but doesn't create a commit
	NoCommit bool
	// Message is an optional custom commit message
	Message string
}

// RevertResult contains the result of a revert operation
type RevertResult struct {
	// NewCommit is the newly created revert commit (nil if NoCommit is true)
	NewCommit *commit.Commit
	// RevertedCommits are the commits that were reverted
	RevertedCommits []*commit.Commit
	// Conflicts indicates if there were conflicts (for future use)
	Conflicts bool
}

// Revert reverts a single commit by creating a new commit that undoes its changes
//
// This is equivalent to: git revert <commit>
func (m *Manager) Revert(ctx context.Context, commitSHA objects.ObjectHash, options RevertOptions) (*RevertResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.logger.Info("reverting commit", "sha", commitSHA.Short())

	// Get the commit to revert
	commitToRevert, err := m.commitManager.GetCommit(ctx, commitSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit %s: %w", commitSHA.Short(), err)
	}

	// Verify the commit has a parent
	if len(commitToRevert.ParentSHAs) == 0 {
		return nil, fmt.Errorf("cannot revert initial commit %s (no parent)", commitSHA.Short())
	}

	// For now, only support reverting commits with a single parent
	if len(commitToRevert.ParentSHAs) > 1 {
		return nil, fmt.Errorf("cannot revert merge commit %s (multiple parents)", commitSHA.Short())
	}

	// Get the parent commit
	parentSHA := commitToRevert.ParentSHAs[0]
	parentCommit, err := m.commitManager.GetCommit(ctx, parentSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent commit %s: %w", parentSHA.Short(), err)
	}

	// Apply the inverse changes by updating to the parent's tree
	if err := m.applyInverseChanges(ctx, commitToRevert, parentCommit); err != nil {
		return nil, fmt.Errorf("failed to apply inverse changes: %w", err)
	}

	result := &RevertResult{
		RevertedCommits: []*commit.Commit{commitToRevert},
		Conflicts:       false,
	}

	// If NoCommit is set, we're done (changes are staged)
	if options.NoCommit {
		return result, nil
	}

	// Create the revert commit
	message := options.Message
	if message == "" {
		message = m.generateRevertMessage(commitToRevert)
	}

	newCommit, err := m.commitManager.CreateCommit(ctx, commitmanager.CommitOptions{
		Message:    message,
		AllowEmpty: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create revert commit: %w", err)
	}

	result.NewCommit = newCommit
	commitHash, _ := newCommit.Hash()
	m.logger.Info("revert commit created", "sha", commitHash.Short())

	return result, nil
}

// RevertRange reverts a range of commits
//
// This is equivalent to: git revert <commit1>..<commit2>
// The range is exclusive of commit1 and inclusive of commit2
func (m *Manager) RevertRange(ctx context.Context, startSHA, endSHA objects.ObjectHash, options RevertOptions) (*RevertResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.logger.Info("reverting commit range", "start", startSHA.Short(), "end", endSHA.Short())

	// Get the list of commits to revert
	commitsToRevert, err := m.getCommitsInRange(ctx, startSHA, endSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits in range: %w", err)
	}

	if len(commitsToRevert) == 0 {
		return nil, fmt.Errorf("no commits found in range %s..%s", startSHA.Short(), endSHA.Short())
	}

	m.logger.Info("reverting commits", "count", len(commitsToRevert))

	// Revert commits in reverse order (newest to oldest)
	// This ensures we revert changes in the correct sequence
	revertedCommits := make([]*commit.Commit, 0, len(commitsToRevert))
	var lastCommit *commit.Commit

	for i := len(commitsToRevert) - 1; i >= 0; i-- {
		commitToRevert := commitsToRevert[i]
		commitHash, _ := commitToRevert.Hash()
		m.logger.Info("reverting commit", "sha", commitHash.Short(), "index", i)

		// Revert each commit individually
		result, err := m.Revert(ctx, commitHash, RevertOptions{
			NoCommit: options.NoCommit && i > 0, // Only commit the last one if NoCommit is false
		})
		if err != nil {
			return nil, fmt.Errorf("failed to revert commit %s: %w", commitHash.Short(), err)
		}

		revertedCommits = append(revertedCommits, commitToRevert)
		if result.NewCommit != nil {
			lastCommit = result.NewCommit
		}
	}

	return &RevertResult{
		NewCommit:       lastCommit,
		RevertedCommits: revertedCommits,
		Conflicts:       false,
	}, nil
}

// applyInverseChanges applies the inverse changes from a commit by updating to its parent's tree
func (m *Manager) applyInverseChanges(ctx context.Context, commitToRevert, parentCommit *commit.Commit) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// The inverse of the commit's changes is to restore the parent's tree
	// We update the working directory and index to match the parent's tree
	parentTreeSHA := parentCommit.TreeSHA

	// Read the parent tree
	parentTreeObj, err := m.repo.ReadObject(parentTreeSHA)
	if err != nil {
		return fmt.Errorf("failed to read parent tree: %w", err)
	}

	parentTree, ok := parentTreeObj.(*tree.Tree)
	if !ok {
		return fmt.Errorf("parent tree object is not a tree")
	}

	// Update the index to match the parent tree
	if err := m.updateIndexToTree(ctx, parentTree); err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}

	// Update the working directory to match the parent commit
	// Note: We use WithForce to override any uncommitted changes
	parentCommitHash, _ := parentCommit.Hash()
	if _, err := m.workdirMgr.UpdateToCommit(ctx, parentCommitHash, workdir.WithForce()); err != nil {
		return fmt.Errorf("failed to update working directory: %w", err)
	}

	return nil
}

// updateIndexToTree updates the index to match a tree
func (m *Manager) updateIndexToTree(ctx context.Context, treeData *tree.Tree) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Get the index manager
	indexMgr := index.NewManager(m.repo.WorkingDirectory())
	if err := indexMgr.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize index manager: %w", err)
	}

	// Clear the current index
	if err := indexMgr.Clear(); err != nil {
		return fmt.Errorf("failed to clear index: %w", err)
	}

	// Add all entries from the tree to the index
	if err := m.addTreeToIndex(ctx, treeData, scpath.RelativePath(""), indexMgr); err != nil {
		return fmt.Errorf("failed to add tree to index: %w", err)
	}

	return nil
}

// addTreeToIndex recursively adds tree entries to the index
func (m *Manager) addTreeToIndex(ctx context.Context, treeData *tree.Tree, prefix scpath.RelativePath, indexMgr *index.Manager) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	entries := treeData.Entries()
	for _, entry := range entries {
		entryPath := prefix
		if prefix != "" {
			entryPath = scpath.RelativePath(string(prefix) + "/" + string(entry.Name()))
		} else {
			entryPath = entry.Name()
		}

		// If it's a tree (directory), recurse
		if entry.Mode() == objects.FileModeDirectory {
			subtreeObj, err := m.repo.ReadObject(entry.SHA())
			if err != nil {
				return fmt.Errorf("failed to read subtree %s: %w", entry.SHA().Short(), err)
			}

			subtree, ok := subtreeObj.(*tree.Tree)
			if !ok {
				return fmt.Errorf("subtree object is not a tree")
			}

			if err := m.addTreeToIndex(ctx, subtree, entryPath, indexMgr); err != nil {
				return err
			}
		} else {
			// It's a file (blob), add it to the index
			// The index manager's Add method expects file paths
			// For now, we'll skip this as it requires reading actual file content
			// In a full implementation, we'd need to write the blob to the working directory
			// and then add it to the index
			continue
		}
	}

	return nil
}

// generateRevertMessage generates a standard revert commit message
func (m *Manager) generateRevertMessage(commitToRevert *commit.Commit) string {
	// Extract the first line of the original message
	firstLine := commitToRevert.Message
	if idx := strings.Index(firstLine, "\n"); idx > 0 {
		firstLine = firstLine[:idx]
	}

	commitHash, _ := commitToRevert.Hash()
	return fmt.Sprintf("Revert \"%s\"\n\nThis reverts commit %s.",
		firstLine,
		commitHash.String())
}

// getCommitsInRange gets all commits in a range (exclusive start, inclusive end)
func (m *Manager) getCommitsInRange(ctx context.Context, startSHA, endSHA objects.ObjectHash) ([]*commit.Commit, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Get all commits from endSHA back to startSHA
	result := make([]*commit.Commit, 0)
	currentSHA := endSHA

	for {
		// Get the current commit
		currentCommit, err := m.commitManager.GetCommit(ctx, currentSHA)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit %s: %w", currentSHA.Short(), err)
		}

		// Stop if we've reached the start commit (exclusive)
		if currentSHA == startSHA {
			break
		}

		// Add to result
		result = append(result, currentCommit)

		// Move to parent
		if len(currentCommit.ParentSHAs) == 0 {
			// We've reached the initial commit without finding startSHA
			return nil, fmt.Errorf("start commit %s not found in history", startSHA.Short())
		}

		// For simplicity, follow the first parent
		currentSHA = currentCommit.ParentSHAs[0]
	}

	return result, nil
}
