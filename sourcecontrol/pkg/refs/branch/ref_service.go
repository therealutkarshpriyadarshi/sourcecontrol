package branch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/common/fileops"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/refs"
)

const (
	// BranchDirName is the directory name for branch refs
	BranchDirName = "heads"

	// HeadFile is the name of the HEAD file
	HeadFile = "HEAD"

	// BranchRefPrefix is the prefix for branch references
	BranchRefPrefix = "refs/heads/"
)

// BranchRefManager handles low-level branch reference operations.
// It wraps the RefManager to provide branch-specific functionality.
type BranchRefManager struct {
	refManager *refs.RefManager
}

// NewRefService creates a new branch reference service
func NewBranchRefManager(refMgr *refs.RefManager) *BranchRefManager {
	return &BranchRefManager{
		refManager: refMgr,
	}
}

// Init initializes the branch manager by creating necessary directories.
// This should be called once after creating a new Manager instance.
func (rs *BranchRefManager) Init() error {
	if err := rs.refManager.Init(); err != nil {
		return fmt.Errorf("init ref manager: %w", err)
	}

	branchDir := filepath.Join(rs.refManager.GetRefsPath().String(), BranchDirName)
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		return fmt.Errorf("create branch directory: %w", err)
	}

	return nil
}

// Current returns the name of the current branch, or empty string if detached
func (rs *BranchRefManager) Current() (string, error) {
	headPath := rs.refManager.GetHeadPath().ToAbsolutePath()
	content, err := fileops.ReadStringStrict(headPath)
	if err != nil {
		return "", fmt.Errorf("read HEAD: %w", err)
	}

	if after, ok := strings.CutPrefix(content, refs.SymbolicRefPrefix); ok {
		refPath := strings.TrimSpace(after)
		// Extract branch name from "refs/heads/branch-name"
		if branchName, ok := strings.CutPrefix(refPath, BranchRefPrefix); ok {
			return branchName, nil
		}
		return "", fmt.Errorf("HEAD points to non-branch ref: %s", refPath)
	}

	return "", nil
}

func (rs *BranchRefManager) ValidateExists(name string) error {
	exists, err := rs.Exists(name)
	if err != nil {
		return fmt.Errorf("check branch exists: %w", err)
	}
	if !exists {
		return NewNotFoundError(name)
	}

	return nil
}

// IsDetached checks if HEAD is in detached state
func (rs *BranchRefManager) IsDetached() (bool, error) {
	current, err := rs.Current()
	if err != nil {
		return false, err
	}
	return current == "", nil
}

// Create creates a new branch reference pointing to the given SHA
func (rs *BranchRefManager) Create(name string, sha objects.ObjectHash) error {
	if err := rs.validateBranchName(name); err != nil {
		return err
	}

	refPath := rs.branchRefPath(name)

	exists, err := rs.refManager.Exists(refPath)
	if err != nil {
		return fmt.Errorf("check branch exists: %w", err)
	}
	if exists {
		return NewAlreadyExistsError(name)
	}

	if err := rs.refManager.UpdateRef(refPath, sha); err != nil {
		return fmt.Errorf("create branch ref: %w", err)
	}

	return nil
}

// Update updates an existing branch to point to a new SHA.
// If the branch doesn't exist and force is true, it will be created.
// This is useful for the initial commit which needs to create the branch reference.
func (rs *BranchRefManager) Update(name string, sha objects.ObjectHash, force bool) error {
	if err := rs.validateBranchName(name); err != nil {
		return err
	}

	refPath := rs.branchRefPath(name)
	exists, err := rs.refManager.Exists(refPath)
	if err != nil {
		return fmt.Errorf("check branch exists: %w", err)
	}

	if !exists && !force {
		return NewNotFoundError(name)
	}

	if err := rs.refManager.UpdateRef(refPath, sha); err != nil {
		return fmt.Errorf("update branch ref: %w", err)
	}

	return nil
}

// Delete deletes a branch reference
func (rs *BranchRefManager) Delete(name string) error {
	if err := rs.validateBranchName(name); err != nil {
		return err
	}

	current, err := rs.Current()
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}
	if current == name {
		return NewIsCurrentError(name)
	}

	refPath := rs.branchRefPath(name)
	deleted, err := rs.refManager.DeleteRef(refPath)
	if err != nil {
		return fmt.Errorf("delete branch ref: %w", err)
	}
	if !deleted {
		return NewNotFoundError(name)
	}

	return nil
}

// Exists checks if a branch exists
func (rs *BranchRefManager) Exists(name string) (bool, error) {
	if err := rs.validateBranchName(name); err != nil {
		return false, err
	}

	refPath := rs.branchRefPath(name)
	return rs.refManager.Exists(refPath)
}

// Resolve resolves a branch name to its commit SHA
func (rs *BranchRefManager) Resolve(name string) (objects.ObjectHash, error) {
	if err := rs.validateBranchName(name); err != nil {
		return "", err
	}

	refPath := rs.branchRefPath(name)
	sha, err := rs.refManager.ResolveToSHA(refPath)
	if err != nil {
		return "", NewNotFoundError(name)
	}

	return sha, nil
}

// List returns all branch names in the repository
func (rs *BranchRefManager) List() ([]string, error) {
	branchDir := filepath.Join(rs.refManager.GetRefsPath().String(), BranchDirName)

	if _, err := os.Stat(branchDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	var branches []string

	// Walk the directory tree to find all branch files
	err := filepath.Walk(branchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from branchDir
		relPath, err := filepath.Rel(branchDir, path)
		if err != nil {
			return err
		}

		// Convert to forward slashes for consistency
		branchName := filepath.ToSlash(relPath)
		branches = append(branches, branchName)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk branch directory: %w", err)
	}

	return branches, nil
}

// SetHead updates HEAD to point to the given branch
func (rs *BranchRefManager) SetHead(branchName string) error {
	if err := rs.validateBranchName(branchName); err != nil {
		return err
	}

	exists, err := rs.Exists(branchName)
	if err != nil {
		return fmt.Errorf("check branch exists: %w", err)
	}
	if !exists {
		return NewNotFoundError(branchName)
	}

	headPath := rs.refManager.GetHeadPath().ToAbsolutePath()
	content := fmt.Sprintf("ref: refs/heads/%s\n", branchName)

	if err := fileops.WriteConfigString(headPath, content); err != nil {
		return fmt.Errorf("update HEAD: %w", err)
	}

	return nil
}

// SetHeadDetached sets HEAD to point directly to a commit (detached state)
func (rs *BranchRefManager) SetHeadDetached(sha objects.ObjectHash) error {
	if err := sha.Validate(); err != nil {
		return fmt.Errorf("invalid SHA: %w", err)
	}

	headPath := rs.refManager.GetHeadPath().ToAbsolutePath()
	content := sha.String() + "\n"

	if err := fileops.WriteConfigString(headPath, content); err != nil {
		return fmt.Errorf("update HEAD: %w", err)
	}

	return nil
}

// GetHeadSHA returns the SHA that HEAD points to
func (rs *BranchRefManager) GetHeadSHA() (objects.ObjectHash, error) {
	sha, err := rs.refManager.ResolveToSHA(refs.RefHEAD)
	if err != nil {
		return "", fmt.Errorf("resolve HEAD: %w", err)
	}
	return sha, nil
}

// branchRefPath converts a branch name to its full ref path
func (rs *BranchRefManager) branchRefPath(name string) refs.RefPath {
	refPath, _ := refs.NewBranchRef(name)
	return refPath
}

// ValidateBranchName validates a branch name according to Git rules
func (rs *BranchRefManager) validateBranchName(name string) error {
	if name == "" {
		return NewInvalidNameError(name, "branch name cannot be empty")
	}

	var reasons []string

	// Check for invalid characters
	invalidChars := []string{" ", "~", "^", ":", "?", "*", "[", "\\", "..", "@{"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			reasons = append(reasons, fmt.Sprintf("contains invalid character '%s'", char))
		}
	}

	// Cannot start or end with slash
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		reasons = append(reasons, "cannot start or end with '/'")
	}

	// Cannot start with a dot
	if strings.HasPrefix(name, ".") {
		reasons = append(reasons, "cannot start with '.'")
	}

	// Cannot end with .lock
	if strings.HasSuffix(name, ".lock") {
		reasons = append(reasons, "cannot end with '.lock'")
	}

	// Cannot contain consecutive slashes
	if strings.Contains(name, "//") {
		reasons = append(reasons, "cannot contain consecutive slashes")
	}

	if len(reasons) > 0 {
		return NewInvalidNameError(name, reasons...)
	}

	return nil
}
