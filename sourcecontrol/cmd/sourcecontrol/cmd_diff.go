package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/cmd/ui"
	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/blob"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

// DiffOptions contains options for diff generation
type DiffOptions struct {
	ContextLines int
	NoColor      bool
	Cached       bool
	NameOnly     bool
	Stat         bool
}

// FileDiff represents the diff of a single file
type FileDiff struct {
	Path       string
	OldMode    string
	NewMode    string
	OldHash    objects.ObjectHash
	NewHash    objects.ObjectHash
	Status     DiffStatus
	IsBinary   bool
	OldContent []byte
	NewContent []byte
	Hunks      []*DiffHunk
}

// DiffStatus represents the change status of a file
type DiffStatus string

const (
	DiffAdded    DiffStatus = "added"
	DiffDeleted  DiffStatus = "deleted"
	DiffModified DiffStatus = "modified"
	DiffRenamed  DiffStatus = "renamed"
)

// DiffHunk represents a unified diff hunk
type DiffHunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

// DiffLine represents a single line in a diff
type DiffLine struct {
	Type    DiffLineType
	Content string
	OldLine int
	NewLine int
}

// DiffLineType represents the type of a diff line
type DiffLineType int

const (
	DiffLineContext DiffLineType = iota
	DiffLineAdded
	DiffLineDeleted
)

func newDiffCmd() *cobra.Command {
	var opts DiffOptions

	cmd := &cobra.Command{
		Use:   "diff [<commit>] [<commit>] [-- <path>...]",
		Short: "Show changes between commits, branches, and files",
		Long: `Show changes between commits, commit and working tree, etc.

Usage:
  srcc diff                    # Changes in working tree (not staged)
  srcc diff --cached           # Changes between index and HEAD
  srcc diff HEAD               # Changes between HEAD and working tree
  srcc diff <commit>           # Changes between <commit> and working tree
  srcc diff <commit> <commit>  # Changes between two commits
  srcc diff <branch1> <branch2> # Changes between two branches

Options:
  -U<n>, --unified=<n>  Generate diffs with <n> lines of context (default: 3)
  --no-color            Turn off colored diff output
  --cached, --staged    Show changes staged for commit
  --name-only           Show only names of changed files
  --stat                Show diffstat instead of full diff

Examples:
  # Show unstaged changes
  srcc diff

  # Show staged changes (what will be committed)
  srcc diff --cached

  # Show changes between two commits
  srcc diff abc123 def456

  # Show changes in specific file
  srcc diff -- file.txt

  # Show only file names that changed
  srcc diff --name-only HEAD~1 HEAD`,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Parse arguments
			var ref1, ref2 string
			var paths []string

			// Find the "--" separator
			dashDashIdx := -1
			for i, arg := range args {
				if arg == "--" {
					dashDashIdx = i
					break
				}
			}

			if dashDashIdx >= 0 {
				// Arguments before "--" are refs, after are paths
				refs := args[:dashDashIdx]
				paths = args[dashDashIdx+1:]

				if len(refs) > 0 {
					ref1 = refs[0]
				}
				if len(refs) > 1 {
					ref2 = refs[1]
				}
			} else {
				// No "--" separator
				if len(args) > 0 {
					ref1 = args[0]
				}
				if len(args) > 1 {
					ref2 = args[1]
				}
			}

			// Perform diff based on arguments
			return performDiff(ctx, repo, ref1, ref2, paths, opts)
		},
	}

	cmd.Flags().IntVarP(&opts.ContextLines, "unified", "U", 3, "Generate diffs with <n> lines of context")
	cmd.Flags().BoolVar(&opts.NoColor, "no-color", false, "Turn off colored diff")
	cmd.Flags().BoolVar(&opts.Cached, "cached", false, "Show changes staged for commit")
	cmd.Flags().BoolVar(&opts.Cached, "staged", false, "Same as --cached")
	cmd.Flags().BoolVar(&opts.NameOnly, "name-only", false, "Show only names of changed files")
	cmd.Flags().BoolVar(&opts.Stat, "stat", false, "Show diffstat")

	return cmd
}

// performDiff performs the diff operation
func performDiff(ctx context.Context, repo *sourcerepo.SourceRepository, ref1, ref2 string, paths []string, opts DiffOptions) error {
	objStore := store.NewFileObjectStore()
	if err := objStore.Initialize(repo.WorkingDirectory()); err != nil {
		return fmt.Errorf("failed to initialize object store: %w", err)
	}

	commitMgr := commitmanager.NewManager(repo)
	if err := commitMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize commit manager: %w", err)
	}

	var diffs []*FileDiff
	var err error

	// Determine what to compare based on arguments
	switch {
	case ref1 == "" && ref2 == "":
		// No refs: compare working tree vs index (or HEAD if --cached)
		if opts.Cached {
			diffs, err = diffIndexVsHEAD(ctx, repo, objStore, commitMgr, paths)
		} else {
			diffs, err = diffWorkingTreeVsIndex(ctx, repo, objStore, paths)
		}

	case ref1 != "" && ref2 == "":
		// One ref: compare ref vs working tree (or index if --cached)
		if opts.Cached {
			return fmt.Errorf("--cached cannot be used with commit reference")
		}
		diffs, err = diffCommitVsWorkingTree(ctx, repo, objStore, commitMgr, ref1, paths)

	case ref1 != "" && ref2 != "":
		// Two refs: compare ref1 vs ref2
		diffs, err = diffCommitVsCommit(ctx, repo, objStore, commitMgr, ref1, ref2, paths)

	default:
		return fmt.Errorf("invalid diff arguments")
	}

	if err != nil {
		return err
	}

	// Display the diffs
	if opts.NameOnly {
		displayNameOnly(diffs)
	} else if opts.Stat {
		displayStat(diffs, opts)
	} else {
		displayUnifiedDiff(diffs, opts)
	}

	return nil
}

// diffWorkingTreeVsIndex compares working tree to index
func diffWorkingTreeVsIndex(ctx context.Context, repo *sourcerepo.SourceRepository, objStore *store.FileObjectStore, paths []string) ([]*FileDiff, error) {
	indexMgr := index.NewManager(repo.WorkingDirectory())
	if err := indexMgr.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize index: %w", err)
	}

	idx := indexMgr.GetIndex()
	entries := idx.Entries

	var diffs []*FileDiff

	// Check modified files
	for _, entry := range entries {
		// Skip if paths filter is specified and doesn't match
		if len(paths) > 0 && !matchesPath(entry.Path.String(), paths) {
			continue
		}

		workingPath := filepath.Join(string(repo.WorkingDirectory()), entry.Path.String())

		// Check if file exists in working tree
		fileInfo, err := os.Stat(workingPath)
		if os.IsNotExist(err) {
			// File deleted in working tree
			oldContent, _ := readBlobContent(objStore, entry.BlobHash)
			diff := &FileDiff{
				Path:       entry.Path.String(),
				OldHash:    entry.BlobHash,
				NewHash:    objects.ObjectHash(""),
				Status:     DiffDeleted,
				IsBinary:   isBinary(oldContent),
				OldContent: oldContent,
				NewContent: nil,
			}
			if !diff.IsBinary {
				diff.Hunks = generateHunks(oldContent, nil, 3)
			}
			diffs = append(diffs, diff)
			continue
		}

		// Read working tree file
		workingContent, err := os.ReadFile(workingPath)
		if err != nil {
			continue
		}

		// Create blob from working content to get hash
		workingBlob := blob.NewBlob(workingContent)
		workingHash, _ := workingBlob.Hash()

		// Compare hashes
		if workingHash != entry.BlobHash {
			oldContent, _ := readBlobContent(objStore, entry.BlobHash)
			diff := &FileDiff{
				Path:       entry.Path.String(),
				OldHash:    entry.BlobHash,
				NewHash:    workingHash,
				Status:     DiffModified,
				IsBinary:   isBinary(workingContent) || isBinary(oldContent),
				OldContent: oldContent,
				NewContent: workingContent,
				OldMode:    entry.Mode.ToOctalString(),
				NewMode:    entry.Mode.ToOctalString(),
			}
			if !diff.IsBinary {
				diff.Hunks = generateHunks(oldContent, workingContent, 3)
			}
			diffs = append(diffs, diff)
		}

		_ = fileInfo
	}

	return diffs, nil
}

// diffIndexVsHEAD compares index to HEAD
func diffIndexVsHEAD(ctx context.Context, repo *sourcerepo.SourceRepository, objStore *store.FileObjectStore, commitMgr *commitmanager.Manager, paths []string) ([]*FileDiff, error) {
	// Get HEAD commit
	history, err := commitMgr.GetHistory(ctx, objects.ObjectHash(""), 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	if len(history) == 0 {
		return nil, fmt.Errorf("no commits yet")
	}

	headCommit := history[0]

	// Get HEAD tree
	headTreeObj, err := objStore.ReadObject(headCommit.TreeSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to read HEAD tree: %w", err)
	}
	headTree := headTreeObj.(*tree.Tree)

	// Get index entries
	indexMgr := index.NewManager(repo.WorkingDirectory())
	if err := indexMgr.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize index: %w", err)
	}

	idx := indexMgr.GetIndex()
	entries := idx.Entries

	// Compare trees
	return compareTreeWithIndex(objStore, headTree, entries, "", paths)
}

// diffCommitVsWorkingTree compares a commit to working tree
func diffCommitVsWorkingTree(ctx context.Context, repo *sourcerepo.SourceRepository, objStore *store.FileObjectStore, commitMgr *commitmanager.Manager, ref string, paths []string) ([]*FileDiff, error) {
	// Resolve commit
	commitHash, err := resolveObjectRef(ctx, repo, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", ref, err)
	}

	commitObj, err := objStore.ReadObject(commitHash)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit: %w", err)
	}
	commit := commitObj.(*commit.Commit)

	// Get commit tree
	treeObj, err := objStore.ReadObject(commit.TreeSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to read tree: %w", err)
	}
	commitTree := treeObj.(*tree.Tree)

	// Get working tree files
	// For now, compare against index (simplified)
	indexMgr := index.NewManager(repo.WorkingDirectory())
	if err := indexMgr.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize index: %w", err)
	}

	idx := indexMgr.GetIndex()
	entries := idx.Entries

	return compareTreeWithIndex(objStore, commitTree, entries, "", paths)
}

// diffCommitVsCommit compares two commits
func diffCommitVsCommit(ctx context.Context, repo *sourcerepo.SourceRepository, objStore *store.FileObjectStore, commitMgr *commitmanager.Manager, ref1, ref2 string, paths []string) ([]*FileDiff, error) {
	// Resolve first commit
	commit1Hash, err := resolveObjectRef(ctx, repo, ref1)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", ref1, err)
	}

	commit1Obj, err := objStore.ReadObject(commit1Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit %s: %w", ref1, err)
	}
	commit1 := commit1Obj.(*commit.Commit)

	// Resolve second commit
	commit2Hash, err := resolveObjectRef(ctx, repo, ref2)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", ref2, err)
	}

	commit2Obj, err := objStore.ReadObject(commit2Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit %s: %w", ref2, err)
	}
	commit2 := commit2Obj.(*commit.Commit)

	// Get trees
	tree1Obj, err := objStore.ReadObject(commit1.TreeSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to read tree1: %w", err)
	}
	tree1 := tree1Obj.(*tree.Tree)

	tree2Obj, err := objStore.ReadObject(commit2.TreeSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to read tree2: %w", err)
	}
	tree2 := tree2Obj.(*tree.Tree)

	// Compare trees
	return compareTrees2(objStore, tree1, tree2, "", paths)
}

// compareTreeWithIndex compares a tree with index entries
func compareTreeWithIndex(objStore *store.FileObjectStore, tree1 *tree.Tree, entries []*index.Entry, prefix string, paths []string) ([]*FileDiff, error) {
	var diffs []*FileDiff

	tree1Map := make(map[string]*tree.TreeEntry)
	for _, entry := range tree1.Entries() {
		tree1Map[entry.Name().String()] = entry
	}

	indexMap := make(map[string]*index.Entry)
	for _, entry := range entries {
		indexMap[entry.Path.String()] = entry
	}

	// Find modified and deleted files
	for name, tree1Entry := range tree1Map {
		path := prefix + name

		if len(paths) > 0 && !matchesPath(path, paths) {
			continue
		}

		indexEntry, inIndex := indexMap[path]

		if !inIndex {
			// Deleted
			oldContent, _ := readBlobContent(objStore, tree1Entry.SHA())
			diff := &FileDiff{
				Path:       path,
				OldHash:    tree1Entry.SHA(),
				NewHash:    objects.ObjectHash(""),
				Status:     DiffDeleted,
				IsBinary:   isBinary(oldContent),
				OldContent: oldContent,
				NewContent: nil,
			}
			if !diff.IsBinary {
				diff.Hunks = generateHunks(oldContent, nil, 3)
			}
			diffs = append(diffs, diff)
		} else if tree1Entry.SHA() != indexEntry.BlobHash {
			// Modified
			oldContent, _ := readBlobContent(objStore, tree1Entry.SHA())
			newContent, _ := readBlobContent(objStore, indexEntry.BlobHash)

			diff := &FileDiff{
				Path:       path,
				OldHash:    tree1Entry.SHA(),
				NewHash:    indexEntry.BlobHash,
				Status:     DiffModified,
				IsBinary:   isBinary(oldContent) || isBinary(newContent),
				OldContent: oldContent,
				NewContent: newContent,
			}
			if !diff.IsBinary {
				diff.Hunks = generateHunks(oldContent, newContent, 3)
			}
			diffs = append(diffs, diff)
		}
	}

	// Find added files
	for path, indexEntry := range indexMap {
		if len(paths) > 0 && !matchesPath(path, paths) {
			continue
		}

		if _, inTree := tree1Map[path]; !inTree {
			newContent, _ := readBlobContent(objStore, indexEntry.BlobHash)
			diff := &FileDiff{
				Path:       path,
				OldHash:    objects.ObjectHash(""),
				NewHash:    indexEntry.BlobHash,
				Status:     DiffAdded,
				IsBinary:   isBinary(newContent),
				OldContent: nil,
				NewContent: newContent,
			}
			if !diff.IsBinary {
				diff.Hunks = generateHunks(nil, newContent, 3)
			}
			diffs = append(diffs, diff)
		}
	}

	return diffs, nil
}

// compareTrees2 compares two trees
func compareTrees2(objStore *store.FileObjectStore, tree1, tree2 *tree.Tree, prefix string, paths []string) ([]*FileDiff, error) {
	var diffs []*FileDiff

	tree1Map := make(map[string]*tree.TreeEntry)
	tree2Map := make(map[string]*tree.TreeEntry)

	for _, entry := range tree1.Entries() {
		tree1Map[entry.Name().String()] = entry
	}

	for _, entry := range tree2.Entries() {
		tree2Map[entry.Name().String()] = entry
	}

	// Find modified and deleted files
	for name, tree1Entry := range tree1Map {
		path := prefix + name

		if len(paths) > 0 && !matchesPath(path, paths) {
			continue
		}

		tree2Entry, inTree2 := tree2Map[name]

		if !inTree2 {
			// Deleted
			oldContent, _ := readBlobContent(objStore, tree1Entry.SHA())
			diff := &FileDiff{
				Path:       path,
				OldHash:    tree1Entry.SHA(),
				NewHash:    objects.ObjectHash(""),
				Status:     DiffDeleted,
				IsBinary:   isBinary(oldContent),
				OldContent: oldContent,
				NewContent: nil,
			}
			if !diff.IsBinary {
				diff.Hunks = generateHunks(oldContent, nil, 3)
			}
			diffs = append(diffs, diff)
		} else if tree1Entry.SHA() != tree2Entry.SHA() {
			// Check if both are directories
			if tree1Entry.IsDirectory() && tree2Entry.IsDirectory() {
				// Recurse into subdirectories
				subTree1, err := loadTree(objStore, tree1Entry.SHA())
				if err != nil {
					continue
				}
				subTree2, err := loadTree(objStore, tree2Entry.SHA())
				if err != nil {
					continue
				}
				subDiffs, err := compareTrees2(objStore, subTree1, subTree2, path+"/", paths)
				if err == nil {
					diffs = append(diffs, subDiffs...)
				}
			} else {
				// Modified file
				oldContent, _ := readBlobContent(objStore, tree1Entry.SHA())
				newContent, _ := readBlobContent(objStore, tree2Entry.SHA())

				diff := &FileDiff{
					Path:       path,
					OldHash:    tree1Entry.SHA(),
					NewHash:    tree2Entry.SHA(),
					Status:     DiffModified,
					IsBinary:   isBinary(oldContent) || isBinary(newContent),
					OldContent: oldContent,
					NewContent: newContent,
				}
				if !diff.IsBinary {
					diff.Hunks = generateHunks(oldContent, newContent, 3)
				}
				diffs = append(diffs, diff)
			}
		}
	}

	// Find added files
	for name, tree2Entry := range tree2Map {
		path := prefix + name

		if len(paths) > 0 && !matchesPath(path, paths) {
			continue
		}

		if _, inTree1 := tree1Map[name]; !inTree1 {
			// Check if it's a directory
			if tree2Entry.IsDirectory() {
				// Recurse into new directory
				subTree2, err := loadTree(objStore, tree2Entry.SHA())
				if err != nil {
					continue
				}
				// Create empty tree for comparison
				emptyTree := tree.NewTree([]*tree.TreeEntry{})
				subDiffs, err := compareTrees2(objStore, emptyTree, subTree2, path+"/", paths)
				if err == nil {
					diffs = append(diffs, subDiffs...)
				}
			} else {
				// Added file
				newContent, _ := readBlobContent(objStore, tree2Entry.SHA())
				diff := &FileDiff{
					Path:       path,
					OldHash:    objects.ObjectHash(""),
					NewHash:    tree2Entry.SHA(),
					Status:     DiffAdded,
					IsBinary:   isBinary(newContent),
					OldContent: nil,
					NewContent: newContent,
				}
				if !diff.IsBinary {
					diff.Hunks = generateHunks(nil, newContent, 3)
				}
				diffs = append(diffs, diff)
			}
		}
	}

	return diffs, nil
}

// generateHunks generates unified diff hunks
func generateHunks(oldContent, newContent []byte, contextLines int) []*DiffHunk {
	oldLines := splitLinesDiff(oldContent)
	newLines := splitLinesDiff(newContent)

	// Simple line-by-line diff (Myers algorithm would be more sophisticated)
	var hunks []*DiffHunk
	var currentHunk *DiffHunk

	oldIdx := 0
	newIdx := 0

	for oldIdx < len(oldLines) || newIdx < len(newLines) {
		if oldIdx < len(oldLines) && newIdx < len(newLines) && oldLines[oldIdx] == newLines[newIdx] {
			// Lines match
			if currentHunk != nil {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    DiffLineContext,
					Content: oldLines[oldIdx],
					OldLine: oldIdx + 1,
					NewLine: newIdx + 1,
				})
				currentHunk.OldCount++
				currentHunk.NewCount++
			}
			oldIdx++
			newIdx++
		} else {
			// Lines differ - start a new hunk if needed
			if currentHunk == nil {
				currentHunk = &DiffHunk{
					OldStart: oldIdx + 1,
					NewStart: newIdx + 1,
				}
			}

			// Try to determine if line was deleted, added, or modified
			if oldIdx < len(oldLines) && (newIdx >= len(newLines) || oldLines[oldIdx] != newLines[newIdx]) {
				// Line deleted
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    DiffLineDeleted,
					Content: oldLines[oldIdx],
					OldLine: oldIdx + 1,
				})
				currentHunk.OldCount++
				oldIdx++
			}

			if newIdx < len(newLines) && (oldIdx >= len(oldLines) || oldLines[oldIdx-1] != newLines[newIdx]) {
				// Line added
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    DiffLineAdded,
					Content: newLines[newIdx],
					NewLine: newIdx + 1,
				})
				currentHunk.NewCount++
				newIdx++
			}
		}

		// Finalize hunk if we've moved past it
		if currentHunk != nil && len(currentHunk.Lines) > 0 {
			// Check if we should close this hunk
			contextAfter := 0
			for i := oldIdx; i < len(oldLines) && i < oldIdx+contextLines; i++ {
				if i < len(newLines) && oldLines[i] == newLines[i] {
					contextAfter++
				} else {
					break
				}
			}

			if contextAfter >= contextLines || (oldIdx >= len(oldLines) && newIdx >= len(newLines)) {
				hunks = append(hunks, currentHunk)
				currentHunk = nil
			}
		}
	}

	// Add final hunk if exists
	if currentHunk != nil && len(currentHunk.Lines) > 0 {
		hunks = append(hunks, currentHunk)
	}

	return hunks
}

// splitLines splits content into lines
func splitLinesDiff(content []byte) []string {
	if len(content) == 0 {
		return []string{}
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// readBlobContent reads content from a blob
func readBlobContent(objStore *store.FileObjectStore, hash objects.ObjectHash) ([]byte, error) {
	if hash == "" {
		return nil, nil
	}

	obj, err := objStore.ReadObject(hash)
	if err != nil {
		return nil, err
	}

	blobObj, ok := obj.(*blob.Blob)
	if !ok {
		return nil, fmt.Errorf("object is not a blob")
	}

	content, err := blobObj.Content()
	if err != nil {
		return nil, err
	}

	return []byte(content), nil
}

// matchesPath checks if a path matches any of the filter paths
func matchesPath(path string, filters []string) bool {
	for _, filter := range filters {
		if path == filter || strings.HasPrefix(path, filter+"/") {
			return true
		}
	}
	return false
}

// displayUnifiedDiff displays diffs in unified format
func displayUnifiedDiff(diffs []*FileDiff, opts DiffOptions) {
	if len(diffs) == 0 {
		fmt.Println("No changes")
		return
	}

	for _, diff := range diffs {
		displayFileDiff(diff, opts)
	}
}

// displayFileDiff displays a single file diff
func displayFileDiff(diff *FileDiff, opts DiffOptions) {
	// File header
	switch diff.Status {
	case DiffAdded:
		if !opts.NoColor {
			fmt.Printf("%s\n", ui.Green(fmt.Sprintf("diff --srcc a/%s b/%s", diff.Path, diff.Path)))
			fmt.Printf("%s\n", ui.Green("new file"))
		} else {
			fmt.Printf("diff --srcc a/%s b/%s\n", diff.Path, diff.Path)
			fmt.Println("new file")
		}

	case DiffDeleted:
		if !opts.NoColor {
			fmt.Printf("%s\n", ui.Red(fmt.Sprintf("diff --srcc a/%s b/%s", diff.Path, diff.Path)))
			fmt.Printf("%s\n", ui.Red("deleted file"))
		} else {
			fmt.Printf("diff --srcc a/%s b/%s\n", diff.Path, diff.Path)
			fmt.Println("deleted file")
		}

	case DiffModified:
		if !opts.NoColor {
			fmt.Printf("%s\n", ui.Cyan(fmt.Sprintf("diff --srcc a/%s b/%s", diff.Path, diff.Path)))
		} else {
			fmt.Printf("diff --srcc a/%s b/%s\n", diff.Path, diff.Path)
		}
	}

	// Index line
	if diff.OldHash != "" && diff.NewHash != "" {
		fmt.Printf("index %s..%s\n", string(diff.OldHash.Short()), string(diff.NewHash.Short()))
	} else if diff.NewHash != "" {
		fmt.Printf("index 0000000..%s\n", string(diff.NewHash.Short()))
	} else if diff.OldHash != "" {
		fmt.Printf("index %s..0000000\n", string(diff.OldHash.Short()))
	}

	// Binary file check
	if diff.IsBinary {
		if !opts.NoColor {
			fmt.Printf("%s\n", ui.Yellow("Binary files differ"))
		} else {
			fmt.Println("Binary files differ")
		}
		fmt.Println()
		return
	}

	// File names
	fmt.Printf("--- a/%s\n", diff.Path)
	fmt.Printf("+++ b/%s\n", diff.Path)

	// Display hunks
	for _, hunk := range diff.Hunks {
		displayHunk(hunk, opts)
	}

	fmt.Println()
}

// displayHunk displays a single diff hunk
func displayHunk(hunk *DiffHunk, opts DiffOptions) {
	// Hunk header
	if !opts.NoColor {
		fmt.Printf("%s\n", ui.Cyan(fmt.Sprintf("@@ -%d,%d +%d,%d @@",
			hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount)))
	} else {
		fmt.Printf("@@ -%d,%d +%d,%d @@\n",
			hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount)
	}

	// Hunk lines
	for _, line := range hunk.Lines {
		displayDiffLine(line, opts)
	}
}

// displayDiffLine displays a single diff line
func displayDiffLine(line DiffLine, opts DiffOptions) {
	switch line.Type {
	case DiffLineAdded:
		if !opts.NoColor {
			fmt.Printf("%s\n", ui.Green("+"+line.Content))
		} else {
			fmt.Printf("+%s\n", line.Content)
		}

	case DiffLineDeleted:
		if !opts.NoColor {
			fmt.Printf("%s\n", ui.Red("-"+line.Content))
		} else {
			fmt.Printf("-%s\n", line.Content)
		}

	case DiffLineContext:
		fmt.Printf(" %s\n", line.Content)
	}
}

// displayNameOnly displays only file names
func displayNameOnly(diffs []*FileDiff) {
	if len(diffs) == 0 {
		return
	}

	for _, diff := range diffs {
		fmt.Println(diff.Path)
	}
}

// displayStat displays diffstat
func displayStat(diffs []*FileDiff, opts DiffOptions) {
	if len(diffs) == 0 {
		fmt.Println("No changes")
		return
	}

	totalAdded := 0
	totalDeleted := 0

	for _, diff := range diffs {
		added := 0
		deleted := 0

		for _, hunk := range diff.Hunks {
			for _, line := range hunk.Lines {
				switch line.Type {
				case DiffLineAdded:
					added++
				case DiffLineDeleted:
					deleted++
				}
			}
		}

		totalAdded += added
		totalDeleted += deleted

		// Display file stat
		status := " "
		switch diff.Status {
		case DiffAdded:
			status = "A"
		case DiffDeleted:
			status = "D"
		case DiffModified:
			status = "M"
		}

		changes := added + deleted
		var bar string
		if changes > 0 {
			maxBar := 40
			addedBar := (added * maxBar) / changes
			if addedBar > maxBar {
				addedBar = maxBar
			}
			deletedBar := maxBar - addedBar

			if !opts.NoColor {
				bar = ui.Green(strings.Repeat("+", addedBar)) + ui.Red(strings.Repeat("-", deletedBar))
			} else {
				bar = strings.Repeat("+", addedBar) + strings.Repeat("-", deletedBar)
			}
		}

		fmt.Printf(" %s %s | %d %s\n", status, diff.Path, changes, bar)
	}

	// Summary
	fmt.Printf(" %d files changed, %d insertions(+), %d deletions(-)\n",
		len(diffs), totalAdded, totalDeleted)
}
