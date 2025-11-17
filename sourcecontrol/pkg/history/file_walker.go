package history

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// FileHistoryWalker walks commit history and filters commits that modified a specific file
type FileHistoryWalker struct {
	repo *sourcerepo.SourceRepository
}

// NewFileHistoryWalker creates a new FileHistoryWalker instance
func NewFileHistoryWalker(repo *sourcerepo.SourceRepository) *FileHistoryWalker {
	return &FileHistoryWalker{repo: repo}
}

// FilterByFile filters commits to only those that modified the specified file
//
// This function walks through the commit history and checks each commit's tree
// to see if the specified file was added, modified, or deleted.
//
// Parameters:
//   - ctx: Context for cancellation
//   - commits: List of commits to filter (typically from GetHistory)
//   - filePath: Path to the file relative to repository root
//
// Returns:
//   - Filtered list of commits that touched the file
//   - Error if there was a problem reading objects
func (w *FileHistoryWalker) FilterByFile(
	ctx context.Context,
	commits []*commit.Commit,
	filePath string,
) ([]*commit.Commit, error) {
	if filePath == "" {
		return commits, nil
	}

	filtered := make([]*commit.Commit, 0, len(commits))

	// Normalize file path (remove leading/trailing slashes, clean up)
	filePath = filepath.Clean(filePath)
	filePath = strings.TrimPrefix(filePath, "/")

	for _, c := range commits {
		select {
		case <-ctx.Done():
			return filtered, ctx.Err()
		default:
		}

		// Check if commit modified the file
		modified, err := w.commitModifiesFile(ctx, c, filePath)
		if err != nil {
			// On error, skip this commit and continue
			continue
		}

		if modified {
			filtered = append(filtered, c)
		}
	}

	return filtered, nil
}

// commitModifiesFile checks if a commit modified a specific file
//
// A file is considered modified if:
// 1. It was added (exists in commit but not in parent)
// 2. Its content changed (different blob hash)
// 3. It was deleted (exists in parent but not in commit)
func (w *FileHistoryWalker) commitModifiesFile(
	ctx context.Context,
	c *commit.Commit,
	filePath string,
) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	// Get file blob hash in this commit
	currentBlob, err := w.findFileInCommit(ctx, c, filePath)
	if err != nil {
		return false, err
	}

	// For initial commit (no parents), include if file exists
	if len(c.ParentSHAs) == 0 {
		return currentBlob != nil, nil
	}

	// Check against all parents (handles merge commits)
	hasChanges := false

	for _, parentSHA := range c.ParentSHAs {
		parentCommit, err := w.repo.ReadCommitObject(parentSHA)
		if err != nil {
			// If we can't read parent, assume file was modified
			return true, nil
		}

		parentBlob, err := w.findFileInCommit(ctx, parentCommit, filePath)
		if err != nil {
			continue
		}

		// File was added in this commit
		if currentBlob != nil && parentBlob == nil {
			hasChanges = true
			break
		}

		// File was deleted in this commit
		if currentBlob == nil && parentBlob != nil {
			hasChanges = true
			break
		}

		// File content changed
		if currentBlob != nil && parentBlob != nil {
			if *currentBlob != *parentBlob {
				hasChanges = true
				break
			}
		}
	}

	return hasChanges, nil
}

// findFileInCommit finds a file's blob hash in a commit's tree
func (w *FileHistoryWalker) findFileInCommit(
	ctx context.Context,
	c *commit.Commit,
	filePath string,
) (*objects.ObjectHash, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	treeObj, err := w.repo.ReadTreeObject(c.TreeSHA)
	if err != nil {
		return nil, fmt.Errorf("read tree object: %w", err)
	}

	return FindFileInTree(ctx, w.repo, treeObj, filePath)
}
