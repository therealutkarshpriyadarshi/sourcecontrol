package merge

import (
	"context"
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

// ThreeWayMerger implements three-way merge strategy
type ThreeWayMerger struct {
	repo           *sourcerepo.SourceRepository
	baseCalculator *MergeBaseCalculator
	objectStore    store.ObjectStore
	resolver       *ConflictResolver
}

// NewThreeWayMerger creates a new three-way merger
func NewThreeWayMerger(repo *sourcerepo.SourceRepository) *ThreeWayMerger {
	return &ThreeWayMerger{
		repo:           repo,
		baseCalculator: NewMergeBaseCalculator(repo),
		objectStore:    repo.ObjectStore(),
	}
}

// Name returns the name of this strategy
func (twm *ThreeWayMerger) Name() string {
	return "three-way"
}

// CanMerge checks if three-way merge can be performed
func (twm *ThreeWayMerger) CanMerge(mergeCtx *MergeContext) bool {
	// Three-way merge works for single branch merges with a common ancestor
	return len(mergeCtx.TheirCommits) == 1 && mergeCtx.BaseCommit != nil
}

// Merge performs a three-way merge
func (twm *ThreeWayMerger) Merge(mergeCtx *MergeContext) (*MergeResult, error) {
	if !twm.CanMerge(mergeCtx) {
		return nil, fmt.Errorf("three-way merge requires exactly one commit and a merge base")
	}

	twm.resolver = NewConflictResolver(mergeCtx.Config.ConflictResolution)

	ourCommit := mergeCtx.OurCommit
	theirCommit := mergeCtx.TheirCommits[0]
	baseCommit := mergeCtx.BaseCommit

	// Get trees for all three commits
	ourTree, err := twm.repo.ReadTreeObject(ourCommit.TreeSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to read our tree: %w", err)
	}

	theirTree, err := twm.repo.ReadTreeObject(theirCommit.TreeSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to read their tree: %w", err)
	}

	baseTree, err := twm.repo.ReadTreeObject(baseCommit.TreeSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to read base tree: %w", err)
	}

	// Perform the three-way merge on trees
	mergedTree, conflicts, err := twm.mergeTrees(mergeCtx.Ctx, baseTree, ourTree, theirTree)
	if err != nil {
		return nil, fmt.Errorf("failed to merge trees: %w", err)
	}

	// Add conflicts to resolver
	for _, conflict := range conflicts {
		twm.resolver.AddConflict(conflict)
	}

	// If there are conflicts and strategy is to fail, return error
	if twm.resolver.HasConflicts() && mergeCtx.Config.ConflictResolution == ConflictFail {
		conflictPaths := make([]string, len(conflicts))
		for i, c := range conflicts {
			conflictPaths[i] = string(c.Path)
		}
		return &MergeResult{
			Success:   false,
			Conflicts: conflictPaths,
			Message:   fmt.Sprintf("Merge conflicts in %d file(s)", len(conflicts)),
		}, nil
	}

	result := &MergeResult{
		Success:     !twm.resolver.HasConflicts(),
		FastForward: false,
	}

	if twm.resolver.HasConflicts() {
		result.Conflicts = make([]string, len(conflicts))
		for i, c := range conflicts {
			result.Conflicts[i] = string(c.Path)
		}
	}

	// If mode is no-commit, don't create the merge commit
	if mergeCtx.Config.Mode == ModeNoCommit {
		// Update the index with merged tree
		if err := twm.updateIndex(mergedTree); err != nil {
			return nil, fmt.Errorf("failed to update index: %w", err)
		}
		result.Message = "Changes staged but not committed (--no-commit)"
		return result, nil
	}

	// Create the merge commit
	mergeCommit, err := twm.createMergeCommit(mergeCtx, mergedTree)
	if err != nil {
		return nil, fmt.Errorf("failed to create merge commit: %w", err)
	}

	commitSHA, err := mergeCommit.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit hash: %w", err)
	}

	result.CommitSHA = commitSHA
	result.Message = fmt.Sprintf("Merge commit %s created", commitSHA.Short())

	return result, nil
}

// mergeTrees performs three-way merge on trees
func (twm *ThreeWayMerger) mergeTrees(ctx context.Context, base, ours, theirs *tree.Tree) (*tree.Tree, []Conflict, error) {
	conflicts := make([]Conflict, 0)

	// Get all entries from all three trees
	baseEntries := twm.treeToMap(base)
	ourEntries := twm.treeToMap(ours)
	theirEntries := twm.treeToMap(theirs)

	// Collect all unique paths
	allPaths := make(map[string]bool)
	for path := range baseEntries {
		allPaths[path] = true
	}
	for path := range ourEntries {
		allPaths[path] = true
	}
	for path := range theirEntries {
		allPaths[path] = true
	}

	mergedEntries := make([]*tree.TreeEntry, 0)

	for path := range allPaths {
		baseEntry := baseEntries[path]
		ourEntry := ourEntries[path]
		theirEntry := theirEntries[path]

		// Determine the merged entry
		mergedEntry, conflict, err := twm.mergeEntry(ctx, path, baseEntry, ourEntry, theirEntry)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to merge entry %s: %w", path, err)
		}

		if conflict != nil {
			conflicts = append(conflicts, *conflict)
		}

		if mergedEntry != nil {
			mergedEntries = append(mergedEntries, mergedEntry)
		}
	}

	// Create merged tree with all merged entries
	mergedTree := tree.NewTree(mergedEntries)

	return mergedTree, conflicts, nil
}

// mergeEntry merges a single tree entry
func (twm *ThreeWayMerger) mergeEntry(ctx context.Context, path string, base, ours, theirs *tree.TreeEntry) (*tree.TreeEntry, *Conflict, error) {
	// Case 1: No change in any version
	if base != nil && ours != nil && theirs != nil {
		if base.SHA().Equal(ours.SHA()) && base.SHA().Equal(theirs.SHA()) {
			return ours, nil, nil
		}
	}

	// Case 2: Only we changed it
	if base != nil && ours != nil && theirs != nil {
		if base.SHA().Equal(theirs.SHA()) && !base.SHA().Equal(ours.SHA()) {
			return ours, nil, nil
		}
	}

	// Case 3: Only they changed it
	if base != nil && ours != nil && theirs != nil {
		if base.SHA().Equal(ours.SHA()) && !base.SHA().Equal(theirs.SHA()) {
			return theirs, nil, nil
		}
	}

	// Case 4: Both changed to the same thing
	if ours != nil && theirs != nil && ours.SHA().Equal(theirs.SHA()) {
		return ours, nil, nil
	}

	// Case 5: File added by both (check content)
	if base == nil && ours != nil && theirs != nil {
		if ours.SHA().Equal(theirs.SHA()) {
			return ours, nil, nil
		}
		// Different additions - conflict
		return ours, twm.createConflict(path, base, ours, theirs), nil
	}

	// Case 6: File deleted by us, unchanged by them
	if base != nil && ours == nil && theirs != nil && base.SHA().Equal(theirs.SHA()) {
		return nil, nil, nil // Accept deletion
	}

	// Case 7: File deleted by them, unchanged by us
	if base != nil && ours != nil && theirs == nil && base.SHA().Equal(ours.SHA()) {
		return nil, nil, nil // Accept deletion
	}

	// Case 8: File added by us only
	if base == nil && ours != nil && theirs == nil {
		return ours, nil, nil
	}

	// Case 9: File added by them only
	if base == nil && ours == nil && theirs != nil {
		return theirs, nil, nil
	}

	// All other cases are conflicts
	return ours, twm.createConflict(path, base, ours, theirs), nil
}

// createConflict creates a conflict object
func (twm *ThreeWayMerger) createConflict(path string, base, ours, theirs *tree.TreeEntry) *Conflict {
	relPath, _ := scpath.NewRelativePath(path)

	conflict := &Conflict{
		Path: relPath,
	}

	if base != nil {
		conflict.BaseSHA = base.SHA()
		conflict.BaseVersion, _ = twm.readBlobContent(base.SHA())
	}

	if ours != nil {
		conflict.OurSHA = ours.SHA()
		conflict.OurVersion, _ = twm.readBlobContent(ours.SHA())
	}

	if theirs != nil {
		conflict.TheirSHA = theirs.SHA()
		conflict.TheirVersion, _ = twm.readBlobContent(theirs.SHA())
	}

	return conflict
}

// readBlobContent reads the content of a blob
func (twm *ThreeWayMerger) readBlobContent(sha objects.ObjectHash) ([]byte, error) {
	blob, err := twm.repo.ReadBlobObject(sha)
	if err != nil {
		return nil, err
	}
	content, err := blob.Content()
	if err != nil {
		return nil, err
	}
	return content.Bytes(), nil
}

// treeToMap converts a tree to a map of path -> entry
func (twm *ThreeWayMerger) treeToMap(t *tree.Tree) map[string]*tree.TreeEntry {
	entries := make(map[string]*tree.TreeEntry)
	for _, entry := range t.Entries() {
		entries[string(entry.Name())] = entry
	}
	return entries
}

// updateIndex updates the index with the merged tree
func (twm *ThreeWayMerger) updateIndex(mergedTree *tree.Tree) error {
	indexMgr := index.NewManager(twm.repo.WorkingDirectory())
	if err := indexMgr.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize index: %w", err)
	}

	// TODO: Update index with merged tree entries
	// This would require adding all entries from the merged tree to the index

	return nil
}

// createMergeCommit creates the merge commit
func (twm *ThreeWayMerger) createMergeCommit(mergeCtx *MergeContext, mergedTree *tree.Tree) (*commit.Commit, error) {
	// Write the merged tree
	treeSHA, err := twm.repo.WriteObject(mergedTree)
	if err != nil {
		return nil, fmt.Errorf("failed to write merged tree: %w", err)
	}

	ourSHA, err := mergeCtx.OurCommit.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get our commit hash: %w", err)
	}

	theirSHA, err := mergeCtx.TheirCommits[0].Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get their commit hash: %w", err)
	}

	// Create merge commit message
	message := mergeCtx.Config.Message
	if message == "" {
		message = fmt.Sprintf("Merge commit %s into HEAD", theirSHA.Short())
	}

	// Create the merge commit
	mergeCommit := &commit.Commit{
		TreeSHA:    treeSHA,
		ParentSHAs: []objects.ObjectHash{ourSHA, theirSHA},
		Author:     mergeCtx.OurCommit.Author,
		Committer:  mergeCtx.OurCommit.Committer,
		Message:    message,
	}

	// Write the commit
	commitSHA, err := twm.repo.WriteObject(mergeCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to write merge commit: %w", err)
	}

	// Update HEAD
	commitMgr := commitmanager.NewManager(twm.repo)
	if err := commitMgr.Initialize(mergeCtx.Ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize commit manager: %w", err)
	}

	// TODO: Update the branch reference to point to the new merge commit

	_ = commitSHA // Use the SHA

	return mergeCommit, nil
}
