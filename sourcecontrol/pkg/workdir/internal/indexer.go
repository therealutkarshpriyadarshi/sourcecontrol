package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	pool "github.com/utkarsh5026/SourceControl/pkg/common/concurrency"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// IndexUpdater implements the IndexUpdater interface for synchronizing the Git index.
type IndexUpdater struct {
	workDir   string
	indexPath scpath.AbsolutePath
}

// NewUpdater creates a new index updater
func NewUpdater(workDir string, indexPath scpath.AbsolutePath) *IndexUpdater {
	return &IndexUpdater{
		workDir:   workDir,
		indexPath: indexPath,
	}
}

// UpdateToMatch replaces the entire index to match the target files.
// This is used after checking out a commit or branch.
// Uses concurrent processing to create index entries for better performance.
func (u *IndexUpdater) UpdateToMatch(targetFiles map[scpath.RelativePath]FileInfo) (IndexUpdateResult, error) {
	result := IndexUpdateResult{
		Success:        true,
		EntriesUpdated: 0,
		EntriesRemoved: 0,
		Errors:         []error{},
	}

	if len(targetFiles) == 0 {
		newIndex := index.NewIndex()
		if err := newIndex.Write(u.indexPath); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Errorf("index %s failed (%s): %w", "write", u.indexPath.String(), err))
			return result, err
		}
		return result, nil
	}

	newIndex := index.NewIndex()

	// Create entries concurrently
	entries, errors := u.createEntries(targetFiles)

	// Add successful entries to index
	for _, entry := range entries {
		newIndex.Add(entry)
		result.EntriesUpdated++
	}

	// Collect errors
	if len(errors) > 0 {
		result.Success = false
		result.Errors = errors
	}

	if result.Success {
		if err := newIndex.Write(u.indexPath); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Errorf("index %s failed (%s): %w", "write", u.indexPath.String(), err))
			return result, err
		}
	}

	return result, nil
}

// createEntries creates index entries for multiple files concurrently.
// Returns successfully created entries and any errors encountered.
// Uses a worker pool for efficient parallel processing.
// Unlike the worker pool's fail-fast behavior, this collects all results and errors.
func (u *IndexUpdater) createEntries(targetFiles map[scpath.RelativePath]FileInfo) ([]*index.Entry, []error) {
	if len(targetFiles) == 0 {
		return nil, nil
	}

	type task struct {
		path scpath.RelativePath
		info FileInfo
	}

	type result struct {
		entry *index.Entry
		err   error
		path  scpath.RelativePath
	}

	pool := pool.NewWorkerPool[task, result]()

	tasks := make([]task, 0, len(targetFiles))
	for path, info := range targetFiles {
		tasks = append(tasks, task{path: path, info: info})
	}

	processFn := func(ctx context.Context, t task) (result, error) {
		entry, err := u.createIndexEntry(t.path, t.info)
		return result{
			entry: entry,
			err:   err,
			path:  t.path,
		}, nil
	}

	results, _ := pool.Process(context.Background(), tasks, processFn)

	var entries []*index.Entry
	var errors []error

	for _, res := range results {
		if res.err != nil {
			errors = append(errors, fmt.Errorf("create entry for %s: %w", res.path, res.err))
		} else if res.entry != nil {
			entries = append(entries, res.entry)
		}
	}

	return entries, errors
}

// UpdateIncremental applies specific additions and removals to the existing index.
// This is more efficient than replacing the entire index.
// Uses concurrent processing to create new index entries for better performance.
func (u *IndexUpdater) UpdateIncremental(toAdd FileMap, toRemove []scpath.RelativePath) (IndexUpdateResult, error) {
	result := IndexUpdateResult{
		Success:        true,
		EntriesUpdated: 0,
		EntriesRemoved: 0,
		Errors:         []error{},
	}

	idx, err := index.Read(u.indexPath)
	if err != nil {
		return result, fmt.Errorf("index %s failed (%s): %w", "read", u.indexPath.String(), err)
	}

	for _, path := range toRemove {
		if idx.Remove(path) {
			result.EntriesRemoved++
		}
	}

	if len(toAdd) > 0 {
		entries, errors := u.createEntries(toAdd)

		for _, entry := range entries {
			idx.Add(entry)
			result.EntriesUpdated++
		}

		if len(errors) > 0 {
			result.Success = false
			result.Errors = errors
		}
	}

	if result.Success {
		if err := idx.Write(u.indexPath); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Errorf("index %s failed (%s): %w", "write", u.indexPath.String(), err))
			return result, err
		}
	}

	return result, nil
}

// createIndexEntry creates an index entry from file information.
// It stats the file to get metadata and combines it with the provided SHA and mode.
func (u *IndexUpdater) createIndexEntry(path scpath.RelativePath, info FileInfo) (*index.Entry, error) {
	fullPath := filepath.Join(u.workDir, path.String())

	stats, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	entry, err := index.NewEntryFromFileInfo(path, stats, info.SHA)
	if err != nil {
		return nil, fmt.Errorf("create entry: %w", err)
	}

	return entry, nil
}
