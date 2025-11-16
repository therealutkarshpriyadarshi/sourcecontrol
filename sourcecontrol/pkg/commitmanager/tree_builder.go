package commitmanager

import (
	"context"
	"fmt"

	pool "github.com/utkarsh5026/SourceControl/pkg/common/concurrency"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

const (
	// concurrencyThreshold is the minimum number of subdirectories
	// required before using concurrent processing.
	// Below this threshold, sequential processing is more efficient.
	concurrencyThreshold = 3
)

// TreeBuilder builds tree objects from the index (staging area).
//
// It converts a flat list of file paths into a hierarchical tree structure
// that mirrors the directory layout. For example:
//   - src/main.go
//   - src/utils/helper.go
//   - README.md
//
// Becomes:
//
//	root/
//	  ├── README.md (blob)
//	  └── src/ (tree)
//	      ├── main.go (blob)
//	      └── utils/ (tree)
//	          └── helper.go (blob)
type TreeBuilder struct {
	repo *sourcerepo.SourceRepository
}

// NewTreeBuilder creates a new TreeBuilder for the given repository
func NewTreeBuilder(repo *sourcerepo.SourceRepository) *TreeBuilder {
	return &TreeBuilder{
		repo: repo,
	}
}

// BuildFromIndex builds a tree object from the given index (staging area).
//
// Process:
//  1. Creates an in-memory directory tree from flat index entries
//  2. Recursively converts the tree into Git-style tree objects
//  3. Returns the root tree's SHA hash
//
// Returns an empty tree if the index contains no entries.
func (tb *TreeBuilder) BuildFromIndex(ctx context.Context, idx *index.Index) (objects.ObjectHash, error) {
	if err := tb.checkContext(ctx); err != nil {
		return "", err
	}

	if idx.Count() == 0 {
		t := tree.NewEmptyTree()
		return tb.repo.WriteObject(t)
	}

	root := tb.buildDirectoryTree(idx)
	treeSHA, err := tb.buildTree(ctx, root)
	if err != nil {
		return "", fmt.Errorf("build tree: %w", err)
	}

	return treeSHA, nil
}

// buildDirectoryTree constructs an in-memory directory tree from index entries
func (tb *TreeBuilder) buildDirectoryTree(idx *index.Index) *directoryNode {
	root := newDirectoryNode("")
	for _, entry := range idx.Entries {
		root.addEntry(entry.Path.String(), entry.BlobHash, objects.FileModeRegular)
	}
	return root
}

// checkContext checks if the context has been cancelled
func (tb *TreeBuilder) checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// buildTree recursively builds tree objects for a directory node.
//
// It processes the directory in two phases:
//  1. Create entries for all files in this directory
//  2. Recursively process subdirectories and create entries for them
//
// Returns the SHA hash of the created tree object.
func (tb *TreeBuilder) buildTree(ctx context.Context, node *directoryNode) (objects.ObjectHash, error) {
	if err := tb.checkContext(ctx); err != nil {
		return "", err
	}

	entries := make([]*tree.TreeEntry, 0, len(node.files)+len(node.subdirs))

	fileEntries, err := tb.buildFileEntries(node)
	if err != nil {
		return "", err
	}
	entries = append(entries, fileEntries...)

	subdirEntries, err := tb.buildSubdirectoryEntries(ctx, node)
	if err != nil {
		return "", err
	}
	entries = append(entries, subdirEntries...)
	return tb.writeTreeObject(entries)
}

// buildFileEntries creates tree entries for all files in the directory node
func (tb *TreeBuilder) buildFileEntries(node *directoryNode) ([]*tree.TreeEntry, error) {
	entries := make([]*tree.TreeEntry, 0, len(node.files))

	for name, sha := range node.files {
		mode := node.modes[name]
		entry, err := tree.NewTreeEntry(mode, scpath.RelativePath(name), sha)
		if err != nil {
			return nil, fmt.Errorf("create tree entry for file %s: %w", name, err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// buildSubdirectoryEntries recursively builds tree entries for all subdirectories.
//
// Performance optimization:
//   - For directories with >= 3 subdirectories: Uses concurrent processing with worker pool
//   - For directories with < 3 subdirectories: Uses sequential processing (lower overhead)
//
// This approach balances parallelism benefits with goroutine overhead.
func (tb *TreeBuilder) buildSubdirectoryEntries(ctx context.Context, node *directoryNode) ([]*tree.TreeEntry, error) {
	if len(node.subdirs) == 0 {
		return []*tree.TreeEntry{}, nil
	}

	// For small numbers of subdirectories, sequential processing is more efficient
	if len(node.subdirs) < concurrencyThreshold {
		return tb.buildSubdirectoriesSequential(ctx, node)
	}

	// For larger numbers, use concurrent processing
	return tb.buildSubdirectoriesConcurrent(ctx, node)
}

// buildSubdirectoriesSequential processes subdirectories one at a time
func (tb *TreeBuilder) buildSubdirectoriesSequential(ctx context.Context, node *directoryNode) ([]*tree.TreeEntry, error) {
	entries := make([]*tree.TreeEntry, 0, len(node.subdirs))

	for name, subdir := range node.subdirs {
		entry, err := tb.buildSubdirectoryEntry(ctx, name, subdir)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// buildSubdirectoriesConcurrent processes subdirectories in parallel using worker pool
func (tb *TreeBuilder) buildSubdirectoriesConcurrent(ctx context.Context, node *directoryNode) ([]*tree.TreeEntry, error) {
	// Create a worker pool for processing subdirectories
	workerPool := pool.NewWorkerPool[*directoryNode, *tree.TreeEntry]()

	// Process subdirectories concurrently
	entryMap, err := workerPool.ProcessMap(
		ctx,
		node.subdirs,
		func(ctx context.Context, subdir *directoryNode) (*tree.TreeEntry, error) {
			return tb.buildSubdirectoryEntry(ctx, subdir.name, subdir)
		},
	)
	if err != nil {
		return nil, err
	}

	// Convert map to slice
	entries := make([]*tree.TreeEntry, 0, len(entryMap))
	for _, entry := range entryMap {
		entries = append(entries, entry)
	}

	return entries, nil
}

// buildSubdirectoryEntry builds a single subdirectory tree and creates its entry
func (tb *TreeBuilder) buildSubdirectoryEntry(ctx context.Context, name string, subdir *directoryNode) (*tree.TreeEntry, error) {
	subTreeSHA, err := tb.buildTree(ctx, subdir)
	if err != nil {
		return nil, fmt.Errorf("build subdirectory %s: %w", name, err)
	}

	// Create a tree entry for the subdirectory
	entry, err := tree.NewTreeEntry(objects.FileModeDirectory, scpath.RelativePath(name), subTreeSHA)
	if err != nil {
		return nil, fmt.Errorf("create tree entry for directory %s: %w", name, err)
	}

	return entry, nil
}

// writeTreeObject creates a tree object from entries and writes it to the repository
func (tb *TreeBuilder) writeTreeObject(entries []*tree.TreeEntry) (objects.ObjectHash, error) {
	treeObj := tree.NewTree(entries)
	treeSHA, err := tb.repo.WriteObject(treeObj)
	if err != nil {
		return "", fmt.Errorf("write tree: %w", err)
	}
	return treeSHA, nil
}
