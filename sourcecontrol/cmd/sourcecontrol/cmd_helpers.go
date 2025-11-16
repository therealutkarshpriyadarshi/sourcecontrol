package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/utkarsh5026/SourceControl/pkg/refs/branch"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// findRepository finds the repository starting from current directory
func findRepository() (*sourcerepo.SourceRepository, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	dir := cwd
	for {
		sourceDir := filepath.Join(dir, scpath.SourceDir)
		if info, err := os.Stat(sourceDir); err == nil && info.IsDir() {
			repoPath, err := scpath.NewRepositoryPath(dir)
			if err != nil {
				return nil, fmt.Errorf("invalid repository path: %w", err)
			}
			return sourcerepo.Open(repoPath)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("not a sourcecontrol repository (or any parent up to mount point)")
		}
		dir = parent
	}
}

// getCurrentBranchName gets the current branch name or returns detached HEAD info
func getCurrentBranchName(repo *sourcerepo.SourceRepository) (string, error) {
	mgr := branch.NewManager(repo)

	// Check if we're in detached HEAD state
	detached, err := mgr.IsDetached()
	if err != nil {
		return "", fmt.Errorf("check detached state: %w", err)
	}

	if detached {
		// Get current commit SHA
		commitSHA, err := mgr.CurrentCommit()
		if err != nil {
			return "", fmt.Errorf("get current commit: %w", err)
		}
		return fmt.Sprintf("HEAD detached at %s", commitSHA.Short()), nil
	}

	// Get current branch name
	branchName, err := mgr.CurrentBranch()
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}

	return branchName, nil
}
