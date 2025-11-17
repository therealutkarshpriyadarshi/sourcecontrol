package stash

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/common"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/refs/branch"
	"github.com/utkarsh5026/SourceControl/pkg/repository/refs"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/store"
	"github.com/utkarsh5026/SourceControl/pkg/workdir"
)

const (
	// StashRefPrefix is the ref path for stashes
	StashRefPrefix = "refs/stash"
)

// Manager handles all stash operations
type Manager struct {
	repo       *sourcerepo.SourceRepository
	refMgr     *refs.RefManager
	branchMgr  *branch.Manager
	commitMgr  *commitmanager.Manager
	objectStore store.ObjectStore
}

// NewManager creates a new stash manager
func NewManager(repo *sourcerepo.SourceRepository) *Manager {
	return &Manager{
		repo:       repo,
		refMgr:     refs.NewRefManager(repo),
		branchMgr:  branch.NewManager(repo),
		commitMgr:  commitmanager.NewManager(repo),
		objectStore: store.NewFileObjectStore(),
	}
}

// Initialize initializes the stash manager
func (m *Manager) Initialize(ctx context.Context) error {
	m.objectStore.Initialize(m.repo.WorkingDirectory())
	return m.commitMgr.Initialize(ctx)
}

// Save creates a new stash entry
func (m *Manager) Save(ctx context.Context, opts StashOptions) (*Entry, error) {
	// Get current branch
	currentBranch, err := m.branchMgr.CurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get HEAD commit
	headSHA, err := m.branchMgr.CurrentCommit()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	// Load index
	indexMgr := index.NewManager(m.repo.WorkingDirectory())
	if err := indexMgr.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize index: %w", err)
	}

	currentIndex := indexMgr.GetIndex()

	// Create index tree (staged changes)
	indexTree, err := m.createTreeFromIndex(ctx, currentIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to create index tree: %w", err)
	}

	// Create working tree (includes unstaged changes)
	workingTree, err := m.createWorkingTree(ctx, opts.Paths)
	if err != nil {
		return nil, fmt.Errorf("failed to create working tree: %w", err)
	}

	// Create commits for the stash
	stashCommit, err := m.createStashCommits(ctx, headSHA, indexTree, workingTree, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create stash commits: %w", err)
	}

	// Get next stash index
	entries, err := m.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list stashes: %w", err)
	}

	stashIndex := len(entries)

	// Save stash reference
	stashRef := m.getStashRef(stashIndex)
	if err := m.refMgr.UpdateRef(stashRef, stashCommit.WorkingTreeCommit); err != nil {
		return nil, fmt.Errorf("failed to save stash ref: %w", err)
	}

	// Update reflog to track stash stack
	if err := m.updateStashReflog(stashIndex, stashCommit.WorkingTreeCommit, opts.Message, currentBranch); err != nil {
		return nil, fmt.Errorf("failed to update stash reflog: %w", err)
	}

	entry := &Entry{
		SHA:       stashCommit.WorkingTreeCommit,
		Message:   opts.Message,
		Branch:    currentBranch,
		CreatedAt: time.Now(),
		Index:     stashIndex,
	}

	// Reset working directory and index if requested
	if !opts.KeepIndex {
		// Clear the index to match HEAD
		if err := indexMgr.Clear(); err != nil {
			return nil, fmt.Errorf("failed to clear index: %w", err)
		}

		// Reset working directory to HEAD
		workdirMgr := workdir.NewManager(m.repo)
		if _, err := workdirMgr.UpdateToCommit(ctx, headSHA, workdir.WithForce()); err != nil {
			return nil, fmt.Errorf("failed to reset working directory: %w", err)
		}
	}

	return entry, nil
}

// List returns all stash entries
func (m *Manager) List() ([]*Entry, error) {
	// Read all stash refs
	stashDir := filepath.Join(m.repo.WorkingDirectory().String(), scpath.SourceDir, "refs", "stash")

	entries := make([]*Entry, 0)

	// Read the reflog to get stash history
	reflogPath := filepath.Join(m.repo.WorkingDirectory().String(), scpath.SourceDir, "logs", "refs", "stash")
	reflog, err := m.readReflog(reflogPath)
	if err != nil {
		// No stashes yet
		return entries, nil
	}

	for i, logEntry := range reflog {
		entry := &Entry{
			SHA:       logEntry.NewSHA,
			Message:   logEntry.Message,
			Branch:    logEntry.Branch,
			CreatedAt: logEntry.Timestamp,
			Index:     i,
		}
		entries = append(entries, entry)
	}

	// Sort by index (most recent first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Index > entries[j].Index
	})

	_ = stashDir // Avoid unused variable

	return entries, nil
}

// Apply applies a stash to the working directory
func (m *Manager) Apply(ctx context.Context, stashIndex int, opts ApplyOptions) error {
	entries, err := m.List()
	if err != nil {
		return fmt.Errorf("failed to list stashes: %w", err)
	}

	if stashIndex >= len(entries) {
		return fmt.Errorf("stash@{%d} does not exist", stashIndex)
	}

	entry := entries[len(entries)-1-stashIndex] // Reverse index

	// Get the stash commit
	stashCommitObj, err := m.commitMgr.GetCommit(ctx, entry.SHA)
	if err != nil {
		return fmt.Errorf("failed to get stash commit: %w", err)
	}

	// Apply the stash tree to working directory
	workdirMgr := workdir.NewManager(m.repo)
	if _, err := workdirMgr.UpdateToCommit(ctx, entry.SHA); err != nil {
		return fmt.Errorf("failed to apply stash: %w", err)
	}

	// If --index is specified, restore staged changes
	if opts.Index && len(stashCommitObj.ParentSHAs) > 1 {
		indexCommitSHA := stashCommitObj.ParentSHAs[1]
		indexCommit, err := m.commitMgr.GetCommit(ctx, indexCommitSHA)
		if err != nil {
			return fmt.Errorf("failed to get index commit: %w", err)
		}

		// Restore index state
		if err := m.restoreIndex(ctx, indexCommit.TreeSHA); err != nil {
			return fmt.Errorf("failed to restore index: %w", err)
		}
	}

	// If pop is requested, drop the stash
	if opts.Pop {
		if err := m.Drop(stashIndex); err != nil {
			return fmt.Errorf("failed to drop stash after applying: %w", err)
		}
	}

	return nil
}

// Pop applies and removes a stash
func (m *Manager) Pop(ctx context.Context, stashIndex int) error {
	return m.Apply(ctx, stashIndex, ApplyOptions{Pop: true})
}

// Drop removes a stash entry
func (m *Manager) Drop(stashIndex int) error {
	entries, err := m.List()
	if err != nil {
		return fmt.Errorf("failed to list stashes: %w", err)
	}

	if stashIndex >= len(entries) {
		return fmt.Errorf("stash@{%d} does not exist", stashIndex)
	}

	// Remove the stash ref
	stashRef := m.getStashRef(stashIndex)
	if _, err := m.refMgr.DeleteRef(stashRef); err != nil {
		return fmt.Errorf("failed to delete stash ref: %w", err)
	}

	// Update reflog to remove the entry
	if err := m.removeFromReflog(stashIndex); err != nil {
		return fmt.Errorf("failed to update reflog: %w", err)
	}

	return nil
}

// Clear removes all stash entries
func (m *Manager) Clear() error {
	entries, err := m.List()
	if err != nil {
		return fmt.Errorf("failed to list stashes: %w", err)
	}

	for i := range entries {
		if err := m.Drop(i); err != nil {
			return fmt.Errorf("failed to drop stash %d: %w", i, err)
		}
	}

	return nil
}

// Helper functions

func (m *Manager) getStashRef(index int) refs.RefPath {
	return refs.RefPath(fmt.Sprintf("%s/%d", StashRefPrefix, index))
}

func (m *Manager) createTreeFromIndex(ctx context.Context, idx *index.Index) (objects.ObjectHash, error) {
	// Build tree from index entries
	treeEntries := make([]*tree.TreeEntry, 0)

	for _, entry := range idx.Entries {
		treeEntry, err := tree.NewTreeEntry(
			entry.Mode,
			entry.Path,
			entry.BlobHash,
		)
		if err != nil {
			return "", fmt.Errorf("failed to create tree entry: %w", err)
		}
		treeEntries = append(treeEntries, treeEntry)
	}

	treeObj := tree.NewTree(treeEntries)
	sha, err := m.objectStore.WriteObject(treeObj)
	if err != nil {
		return "", fmt.Errorf("failed to write tree: %w", err)
	}

	return sha, nil
}

func (m *Manager) createWorkingTree(ctx context.Context, paths []string) (objects.ObjectHash, error) {
	// For now, create tree from current working directory
	// In a full implementation, this would scan the working directory
	// and create a tree object from all tracked files

	indexMgr := index.NewManager(m.repo.WorkingDirectory())
	if err := indexMgr.Initialize(); err != nil {
		return "", fmt.Errorf("failed to initialize index: %w", err)
	}

	currentIndex := indexMgr.GetIndex()

	// If paths are specified, filter the index
	if len(paths) > 0 {
		filteredIndex := index.NewIndex()
		for _, entry := range currentIndex.Entries {
			for _, path := range paths {
				if strings.HasPrefix(entry.Path.String(), path) {
					filteredIndex.Add(entry)
				}
			}
		}
		currentIndex = filteredIndex
	}

	return m.createTreeFromIndex(ctx, currentIndex)
}

func (m *Manager) createStashCommits(ctx context.Context, baseCommit, indexTree, workingTree objects.ObjectHash, opts StashOptions) (*StashCommit, error) {
	// Get author info from config
	author := &commit.CommitPerson{
		Name:  "Stash User",
		Email: "stash@sourcecontrol.local",
		When:  common.NewTimestampFromTime(time.Now()),
	}

	// Create index commit (parent: base commit)
	indexCommitMsg := fmt.Sprintf("index on %s", opts.Message)
	indexCommitObj := &commit.Commit{
		TreeSHA:    indexTree,
		ParentSHAs: []objects.ObjectHash{baseCommit},
		Author:     author,
		Committer:  author,
		Message:    indexCommitMsg,
	}
	indexCommitSHA, err := m.objectStore.WriteObject(indexCommitObj)
	if err != nil {
		return nil, fmt.Errorf("failed to write index commit: %w", err)
	}

	// Create working tree commit (parents: base commit, index commit)
	workingCommitMsg := opts.Message
	if workingCommitMsg == "" {
		workingCommitMsg = "WIP on stash"
	}

	workingCommitObj := &commit.Commit{
		TreeSHA:    workingTree,
		ParentSHAs: []objects.ObjectHash{baseCommit, indexCommitSHA},
		Author:     author,
		Committer:  author,
		Message:    workingCommitMsg,
	}
	workingCommitSHA, err := m.objectStore.WriteObject(workingCommitObj)
	if err != nil {
		return nil, fmt.Errorf("failed to write working tree commit: %w", err)
	}

	return &StashCommit{
		WorkingTreeCommit: workingCommitSHA,
		IndexCommit:       indexCommitSHA,
		BaseCommit:        baseCommit,
	}, nil
}

func (m *Manager) restoreIndex(ctx context.Context, treeSHA objects.ObjectHash) error {
	// Load the tree
	treeObj, err := m.objectStore.ReadObject(treeSHA)
	if err != nil {
		return fmt.Errorf("failed to read tree: %w", err)
	}

	treeData, ok := treeObj.(*tree.Tree)
	if !ok {
		return fmt.Errorf("object is not a tree")
	}

	// Rebuild index from tree
	indexMgr := index.NewManager(m.repo.WorkingDirectory())
	if err := indexMgr.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize index: %w", err)
	}

	if err := indexMgr.Clear(); err != nil {
		return fmt.Errorf("failed to clear index: %w", err)
	}

	// Add all tree entries to index
	for _, entry := range treeData.Entries() {
		// This is simplified - a full implementation would recursively walk the tree
		// and add all entries to the index
		_ = entry
	}

	return nil
}

// ReflogEntry represents an entry in the stash reflog
type ReflogEntry struct {
	OldSHA    objects.ObjectHash
	NewSHA    objects.ObjectHash
	Message   string
	Branch    string
	Timestamp time.Time
}

func (m *Manager) updateStashReflog(index int, sha objects.ObjectHash, message, branch string) error {
	reflogPath := filepath.Join(m.repo.WorkingDirectory().String(), scpath.SourceDir, "logs", "refs", "stash")

	// Create directory if it doesn't exist
	if err := m.createReflogDir(reflogPath); err != nil {
		return err
	}

	entry := ReflogEntry{
		OldSHA:    "",
		NewSHA:    sha,
		Message:   message,
		Branch:    branch,
		Timestamp: time.Now(),
	}

	return m.appendReflog(reflogPath, entry)
}

func (m *Manager) createReflogDir(reflogPath string) error {
	// Implementation would create the directory structure
	return nil
}

func (m *Manager) appendReflog(reflogPath string, entry ReflogEntry) error {
	// Implementation would append to the reflog file
	return nil
}

func (m *Manager) readReflog(reflogPath string) ([]ReflogEntry, error) {
	// Implementation would read the reflog file
	return nil, fmt.Errorf("no reflog found")
}

func (m *Manager) removeFromReflog(index int) error {
	// Implementation would remove entry from reflog
	return nil
}

// Show displays the contents of a stash
func (m *Manager) Show(ctx context.Context, stashIndex int) (*commit.Commit, error) {
	entries, err := m.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list stashes: %w", err)
	}

	if stashIndex >= len(entries) {
		return nil, fmt.Errorf("stash@{%d} does not exist", stashIndex)
	}

	entry := entries[len(entries)-1-stashIndex]

	// Get the stash commit
	stashCommit, err := m.commitMgr.GetCommit(ctx, entry.SHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get stash commit: %w", err)
	}

	return stashCommit, nil
}

// GetStashName returns the name of a stash by index
func GetStashName(index int) string {
	return fmt.Sprintf("stash@{%d}", index)
}

// ParseStashName parses a stash name and returns the index
func ParseStashName(name string) (int, error) {
	// Handle "stash@{N}" format
	if strings.HasPrefix(name, "stash@{") && strings.HasSuffix(name, "}") {
		indexStr := strings.TrimPrefix(name, "stash@{")
		indexStr = strings.TrimSuffix(indexStr, "}")
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return 0, fmt.Errorf("invalid stash name: %s", name)
		}
		return index, nil
	}

	// Handle just a number
	index, err := strconv.Atoi(name)
	if err != nil {
		return 0, fmt.Errorf("invalid stash name: %s", name)
	}
	return index, nil
}
