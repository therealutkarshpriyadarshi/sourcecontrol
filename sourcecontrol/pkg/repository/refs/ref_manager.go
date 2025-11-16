package refs

import (
	"fmt"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/common/fileops"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

const (
	// SymbolicRefPrefix is the prefix used to indicate that a reference
	// points to another reference rather than directly to a commit.
	// Example: "ref: refs/heads/master" in the HEAD file.
	SymbolicRefPrefix = "ref: "

	// MaxRefDepth is the maximum number of symbolic references that can
	// be followed when resolving a reference to its final SHA-1 hash.
	// This prevents infinite loops in case of circular references.
	MaxRefDepth = 10
)

// RefManager manages Git references in a repository. It provides operations
// for creating, reading, updating, and deleting references, as well as
// resolving symbolic references to their final commit hashes.
//
// References are stored as files in the .git/refs directory, with the file
// content being either a 40-character SHA-1 hash or a symbolic reference
// starting with "ref: ".
type RefManager struct {
	refsPath scpath.SourcePath // Path to the refs directory (.git/refs)
	headPath scpath.SourcePath // Path to the HEAD file (.git/HEAD)
}

// NewRefManager creates a new reference manager for the given repository.
// It initializes the paths to the refs directory and HEAD file based on
// the repository's source directory structure.
//
// Parameters:
//   - repo: The repository for which to create the reference manager
//
// Returns:
//   - A new RefManager instance configured for the given repository
func NewRefManager(repo sourcerepo.Repository) *RefManager {
	sourceDir := repo.SourceDirectory()
	return &RefManager{
		refsPath: sourceDir.RefsPath(),
		headPath: sourceDir.HeadPath(),
	}
}

// Init initializes the reference manager by creating necessary directory
// structure and files. This includes:
//   - Creating the refs directory (.git/refs)
//   - Creating the HEAD file with default content pointing to master branch
//
// This method should be called once when initializing a new repository.
//
// Returns:
//   - An error if directory or file creation fails, nil otherwise
func (rm *RefManager) Init() error {
	if err := fileops.EnsureDir(rm.refsPath.ToAbsolutePath()); err != nil {
		return fmt.Errorf("failed to create refs directory: %w", err)
	}

	defaultRef := "ref: refs/heads/master\n"
	if err := fileops.WriteConfigString(rm.headPath.ToAbsolutePath(), defaultRef); err != nil {
		return fmt.Errorf("failed to create HEAD file: %w", err)
	}

	return nil
}

// ReadRef reads the content of a reference file. The content can be either:
//   - A 40-character SHA-1 hash pointing directly to a commit
//   - A symbolic reference starting with "ref: " pointing to another reference
//
// Parameters:
//   - ref: The reference path to read (e.g., "HEAD", "refs/heads/master")
//
// Returns:
//   - The raw content of the reference file (including newlines)
//   - An error if the reference doesn't exist or cannot be read
//
// Example:
//
//	content, err := rm.ReadRef("HEAD")
//	// content might be "ref: refs/heads/master\n" or "abc123...\n"
func (rm *RefManager) ReadRef(ref RefPath) (string, error) {
	fullPath := rm.resolveReferencePath(ref)

	content, err := fileops.ReadStringStrict(fullPath.ToAbsolutePath())
	if err != nil {
		return "", fmt.Errorf("error reading ref %s: %w", ref, err)
	}

	return content, nil
}

// UpdateRef updates a reference to point to a new commit. This operation:
//   - Validates the provided SHA-1 hash
//   - Creates parent directories if they don't exist
//   - Writes the hash to the reference file
//
// This is used for operations like committing, branching, or merging.
//
// Parameters:
//   - ref: The reference path to update (e.g., "refs/heads/master")
//   - hash: The SHA-1 hash of the commit to point to
//
// Returns:
//   - An error if validation fails, directory creation fails, or write fails
//   - nil on success
//
// Example:
//
//	hash := objects.ObjectHash("abc123...")
//	err := rm.UpdateRef("refs/heads/master", hash)
func (rm *RefManager) UpdateRef(ref RefPath, hash objects.ObjectHash) error {
	if err := hash.Validate(); err != nil {
		return fmt.Errorf("invalid hash: %w", err)
	}

	fullPath := rm.resolveReferencePath(ref).ToAbsolutePath()

	if err := fileops.EnsureParentDir(fullPath); err != nil {
		return fmt.Errorf("failed to create ref directory: %w", err)
	}

	content := hash.String() + "\n"
	if err := fileops.WriteConfigString(fullPath, content); err != nil {
		return fmt.Errorf("failed to write ref %s: %w", ref, err)
	}

	return nil
}

// ResolveToSHA resolves a reference to its final SHA-1 hash by following
// symbolic references. This process:
//   - Reads the reference content
//   - If it's a symbolic ref (starts with "ref: "), follows the target
//   - Repeats until finding a direct SHA-1 hash
//   - Enforces MaxRefDepth to prevent infinite loops
//
// Parameters:
//   - ref: The reference path to resolve (e.g., "HEAD", "refs/heads/master")
//
// Returns:
//   - The final SHA-1 hash that the reference ultimately points to
//   - An error if the reference doesn't exist, is invalid, or depth is exceeded
//
// Example:
//
//	// If HEAD contains "ref: refs/heads/master"
//	// and refs/heads/master contains "abc123..."
//	hash, err := rm.ResolveToSHA("HEAD")
//	// hash will be "abc123..."
func (rm *RefManager) ResolveToSHA(ref RefPath) (objects.ObjectHash, error) {
	currentRef := ref

	for range MaxRefDepth {
		content, err := rm.ReadRef(currentRef)
		if err != nil {
			return "", fmt.Errorf("error reading ref %s: %w", currentRef, err)
		}

		if after, ok := strings.CutPrefix(content, SymbolicRefPrefix); ok {
			target := after
			currentRef = RefPath(target)
			continue
		}

		hash, err := objects.NewObjectHashFromString(content)
		if err != nil {
			return "", fmt.Errorf("invalid ref content: %s", content)
		}

		return hash, nil
	}

	return "", fmt.Errorf("reference depth exceeded for %s", ref)
}

// DeleteRef deletes a reference file from the repository. This is used
// when deleting branches or cleaning up stale references.
//
// Parameters:
//   - ref: The reference path to delete (e.g., "refs/heads/feature")
//
// Returns:
//   - true if the reference existed and was deleted
//   - false if the reference didn't exist
//   - An error if the deletion operation fails
//
// Example:
//
//	deleted, err := rm.DeleteRef("refs/heads/old-feature")
//	if deleted {
//	    fmt.Println("Branch deleted successfully")
//	}
func (rm *RefManager) DeleteRef(ref RefPath) (bool, error) {
	fullPath := rm.resolveReferencePath(ref).ToAbsolutePath()

	exists, err := fileops.Exists(fullPath)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	if err := fileops.SafeRemove(fullPath); err != nil {
		return false, err
	}

	return true, nil
}

// Exists checks whether a reference exists in the repository.
//
// Parameters:
//   - ref: The reference path to check (e.g., "refs/heads/master")
//
// Returns:
//   - true if the reference exists, false otherwise
//   - An error if the existence check fails
func (rm *RefManager) Exists(ref RefPath) (bool, error) {
	fullPath := rm.resolveReferencePath(ref).ToAbsolutePath()
	return fileops.Exists(fullPath)
}

// GetHeadPath returns the full path to the HEAD file. The HEAD file
// is a special reference that typically points to the current branch
// or directly to a commit (detached HEAD state).
//
// Returns:
//   - The path to the HEAD file (.git/HEAD)
func (rm *RefManager) GetHeadPath() scpath.SourcePath {
	return rm.headPath
}

// GetRefsPath returns the full path to the refs directory where all
// branch and tag references are stored.
//
// Returns:
//   - The path to the refs directory (.git/refs)
func (rm *RefManager) GetRefsPath() scpath.SourcePath {
	return rm.refsPath
}

// resolveReferencePath resolves a RefPath to its full filesystem path.
// This handles special cases like HEAD and properly joins relative paths.
//
// The resolution follows these rules:
//   - "HEAD" maps to .git/HEAD
//   - "refs/heads/master" maps to .git/refs/heads/master
//   - "heads/master" maps to .git/refs/heads/master
//
// Parameters:
//   - ref: The reference path to resolve
//
// Returns:
//   - The full filesystem path to the reference file
func (rm *RefManager) resolveReferencePath(ref RefPath) scpath.SourcePath {
	refStr := strings.TrimSpace(ref.String())

	if refStr == scpath.HeadFile {
		return rm.headPath
	}

	if after, ok := strings.CutPrefix(refStr, scpath.RefsDir+"/"); ok {
		relPath := after
		return rm.refsPath.Join(relPath)
	}

	return rm.refsPath.Join(refStr)
}
