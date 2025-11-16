package internal

import (
	"context"
	"fmt"
	"maps"

	pool "github.com/utkarsh5026/SourceControl/pkg/common/concurrency"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"golang.org/x/sync/errgroup"
)

// FileInfo represents metadata about a file in a tree or index
type FileInfo struct {
	SHA  objects.ObjectHash
	Mode objects.FileMode
}

// ChangeAnalysis contains the result of comparing two file states
type ChangeAnalysis struct {
	Operations  []Operation
	Summary     ChangeSummary
	TargetFiles map[scpath.RelativePath]FileInfo
}

// Analyzer implements the TreeAnalyzer interface for analyzing Git trees and detecting changes.
// It handles tree walking, file collection, and change detection.
type Analyzer struct {
	repo *sourcerepo.SourceRepository
}

// NewAnalyzer creates a new TreeAnalyzer
func NewAnalyzer(repo *sourcerepo.SourceRepository) *Analyzer {
	return &Analyzer{
		repo: repo,
	}
}

// GetCommitFiles retrieves all files from a commit's tree.
// It reads the commit object, gets the root tree SHA, and recursively walks the tree.
func (a *Analyzer) GetCommitFiles(ctx context.Context, commitSHA objects.ObjectHash) (map[scpath.RelativePath]FileInfo, error) {
	c, err := a.repo.ReadCommitObject(commitSHA)
	if err != nil {
		return nil, err
	}

	if c.TreeSHA == "" {
		return nil, fmt.Errorf("commit %s has no tree", commitSHA.Short())
	}

	return a.getTreeFiles(ctx, c.TreeSHA, scpath.RelativePath(""))
}

// getTreeFiles recursively walks a tree object and collects all files.
// It handles nested trees (subdirectories) and builds the complete file map.
func (a *Analyzer) getTreeFiles(ctx context.Context, treeSHA objects.ObjectHash, basePath scpath.RelativePath) (map[scpath.RelativePath]FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	files := make(map[scpath.RelativePath]FileInfo)
	treeObj, err := a.repo.ReadTreeObject(treeSHA)
	if err != nil {
		return nil, fmt.Errorf("read tree %s: %w", treeSHA.Short(), err)
	}

	type dirTask struct {
		sha  objects.ObjectHash
		path scpath.RelativePath
	}
	var directories []dirTask

	for _, e := range treeObj.Entries() {
		var fullPath scpath.RelativePath
		if basePath == "" {
			fullPath = e.Name()
		} else {
			fullPath = basePath.Join(e.Name().String())
		}

		if e.IsDirectory() {
			directories = append(directories, dirTask{e.SHA(), fullPath})
			continue
		}

		if a.isSupportedFileType(e) {
			files[fullPath] = FileInfo{
				SHA:  e.SHA(),
				Mode: e.Mode(),
			}
		}
	}

	if len(directories) == 0 {
		return files, nil
	}

	if len(directories) == 1 {
		subFiles, err := a.getTreeFiles(ctx, directories[0].sha, directories[0].path)
		if err != nil {
			return nil, err
		}
		maps.Copy(files, subFiles)
		return files, nil
	}

	pool := pool.NewWorkerPool[dirTask, FileMap]()
	processFn := func(ctx context.Context, task dirTask) (FileMap, error) {
		return a.getTreeFiles(ctx, task.sha, task.path)
	}

	results, err := pool.Process(ctx, directories, processFn)
	if err != nil {
		return nil, err
	}

	for _, subFiles := range results {
		maps.Copy(files, subFiles)
	}

	return files, nil
}

// GetIndexFiles extracts file information from the Git index.
// Converts index entries to the FileInfo format used by the analyzer.
func (a *Analyzer) GetIndexFiles(idx *index.Index) FileMap {
	files := make(FileMap)
	for _, entry := range idx.Entries {
		files[entry.Path] = FileInfo{
			SHA:  entry.BlobHash,
			Mode: entry.Mode,
		}
	}
	return files
}

// AnalyzeChanges compares current and target file states to generate operations.
// It detects files that need to be created, modified, or deleted.
// The analysis runs concurrently for better performance using errgroup.
func (a *Analyzer) AnalyzeChanges(current, target FileMap) ChangeAnalysis {
	var operations []Operation
	summary := ChangeSummary{}

	type analysisResult struct {
		ops     []Operation
		deleted int
		created int
		changed int
	}

	var deleteResult, createModifyResult analysisResult
	g := new(errgroup.Group)

	g.Go(func() error {
		var localSummary ChangeSummary
		ops := findDeletedFiles(current, target, &localSummary)
		deleteResult = analysisResult{
			ops:     ops,
			deleted: localSummary.Deleted,
		}
		return nil
	})

	g.Go(func() error {
		var localSummary ChangeSummary
		ops := a.findCreatedAndModifiedFiles(current, target, &localSummary)
		createModifyResult = analysisResult{
			ops:     ops,
			created: localSummary.Created,
			changed: localSummary.Modified,
		}
		return nil
	})

	_ = g.Wait()

	operations = append(operations, deleteResult.ops...)
	operations = append(operations, createModifyResult.ops...)

	summary.Deleted = deleteResult.deleted
	summary.Created = createModifyResult.created
	summary.Modified = createModifyResult.changed

	return ChangeAnalysis{
		Operations:  operations,
		Summary:     summary,
		TargetFiles: target,
	}
}

func findDeletedFiles(current, target FileMap, summary *ChangeSummary) []Operation {
	var operations []Operation

	for path := range current {
		if _, exists := target[path]; !exists {
			operations = append(operations, Operation{
				Path:   path,
				Action: ActionDelete,
			})
			summary.Deleted++
		}
	}

	return operations
}

// findCreatedAndModifiedFiles identifies new and changed files in target
func (a *Analyzer) findCreatedAndModifiedFiles(current, target FileMap, summary *ChangeSummary) []Operation {
	var operations []Operation

	for path, targetInfo := range target {
		currentInfo, exists := current[path]

		if !exists {
			operations = append(operations, Operation{
				Path:   path,
				Action: ActionCreate,
				SHA:    targetInfo.SHA,
				Mode:   targetInfo.Mode,
			})
			summary.Created++
		} else if a.hasChanged(currentInfo, targetInfo) {
			operations = append(operations, Operation{
				Path:   path,
				Action: ActionModify,
				SHA:    targetInfo.SHA,
				Mode:   targetInfo.Mode,
			})
			summary.Modified++
		}
	}

	return operations
}

// AreTreesIdentical checks if two trees contain exactly the same files.
// This is an optimization to avoid unnecessary operations.
func (a *Analyzer) AreTreesIdentical(ctx context.Context, treeSHA1, treeSHA2 objects.ObjectHash) (bool, error) {
	if treeSHA1 == treeSHA2 {
		return true, nil
	}

	var tree1Files, tree2Files FileMap

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		tree1Files, err = a.getTreeFiles(ctx, treeSHA1, scpath.RelativePath(""))
		if err != nil {
			return fmt.Errorf("read tree1: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		tree2Files, err = a.getTreeFiles(ctx, treeSHA2, scpath.RelativePath(""))
		if err != nil {
			return fmt.Errorf("read tree2: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return false, err
	}

	if len(tree1Files) != len(tree2Files) {
		return false, nil
	}

	for path, info1 := range tree1Files {
		info2, exists := tree2Files[path]
		if !exists || a.hasChanged(info1, info2) {
			return false, nil
		}
	}

	return true, nil
}

// isSupportedFileType checks if a tree entry represents a supported file type.
// We support regular files, executables, and symlinks (but not submodules for now).
func (a *Analyzer) isSupportedFileType(entry *tree.TreeEntry) bool {
	return entry.IsFile() || entry.IsExecutable() || entry.IsSymbolicLink()
}

// hasChanged checks if a file has changed between two states.
// A file is considered changed if either its content (SHA) or mode has changed.
func (a *Analyzer) hasChanged(current, target FileInfo) bool {
	return current.SHA != target.SHA || current.Mode != target.Mode
}
