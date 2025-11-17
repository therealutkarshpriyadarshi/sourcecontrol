package history

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// FindFileInTree recursively searches for a file in a tree and returns its blob hash
//
// The function splits the file path by directory separator and walks through
// the tree structure to find the target file.
//
// Parameters:
//   - ctx: Context for cancellation
//   - repo: Repository to read tree objects from
//   - t: Root tree to search in
//   - filePath: Path to file (e.g., "src/main.go" or "README.md")
//
// Returns:
//   - Pointer to blob hash if found, nil if not found
//   - Error if there was a problem reading tree objects
func FindFileInTree(
	ctx context.Context,
	repo *sourcerepo.SourceRepository,
	t *tree.Tree,
	filePath string,
) (*objects.ObjectHash, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Normalize and split path
	filePath = filepath.Clean(filePath)
	filePath = strings.TrimPrefix(filePath, "/")

	if filePath == "" || filePath == "." {
		return nil, nil
	}

	parts := strings.Split(filePath, string(filepath.Separator))
	return findFileInTreeRecursive(ctx, repo, t, parts)
}

// findFileInTreeRecursive performs the recursive search through tree hierarchy
func findFileInTreeRecursive(
	ctx context.Context,
	repo *sourcerepo.SourceRepository,
	t *tree.Tree,
	pathParts []string,
) (*objects.ObjectHash, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if len(pathParts) == 0 {
		return nil, nil
	}

	// Get target name (first path component)
	targetName := pathParts[0]
	if targetName == "" {
		return nil, nil
	}

	// Search through tree entries
	entries := t.Entries()

	for _, entry := range entries {
		if entry.Name().String() == targetName {
			// Found matching entry!

			if len(pathParts) == 1 {
				// This is the final component
				if entry.IsFile() {
					// Found the file - return its blob hash
					hash := entry.SHA()
					return &hash, nil
				}
				// If it's a tree but we expected a file, return nil
				return nil, nil
			}

			// More path components remain - must descend into subtree
			if entry.IsDirectory() {
				// Load the subtree
				subTree, err := repo.ReadTreeObject(entry.SHA())
				if err != nil {
					return nil, err
				}

				// Recursively search in subtree
				return findFileInTreeRecursive(ctx, repo, subTree, pathParts[1:])
			}

			// Entry is a blob but we need to go deeper - not found
			return nil, nil
		}
	}

	// Entry not found in this tree
	return nil, nil
}

// GetAllFilesInTree returns all file paths in a tree (recursively)
//
// This is useful for operations like finding all files that were added in a commit.
func GetAllFilesInTree(
	ctx context.Context,
	repo *sourcerepo.SourceRepository,
	t *tree.Tree,
	prefix string,
) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	files := make([]string, 0)
	entries := t.Entries()

	for _, entry := range entries {
		entryPath := entry.Name().String()
		if prefix != "" {
			entryPath = filepath.Join(prefix, entry.Name().String())
		}

		if entry.IsFile() {
			files = append(files, entryPath)
		} else if entry.IsDirectory() {
			// Recursively get files from subtree
			subTree, err := repo.ReadTreeObject(entry.SHA())
			if err != nil {
				continue // Skip on error
			}

			subFiles, err := GetAllFilesInTree(ctx, repo, subTree, entryPath)
			if err != nil {
				continue // Skip on error
			}

			files = append(files, subFiles...)
		}
	}

	return files, nil
}
