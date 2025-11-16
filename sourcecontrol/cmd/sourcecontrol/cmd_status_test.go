package main

import (
	"os"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

func TestStatusCommand(t *testing.T) {
	// Save and restore current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Set up git config for commits
	os.Setenv("GIT_AUTHOR_NAME", "Test User")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	defer os.Unsetenv("GIT_AUTHOR_NAME")
	defer os.Unsetenv("GIT_AUTHOR_EMAIL")

	t.Run("status on clean repository", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Run status command
		cmd := newStatusCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("status command failed: %v", err)
		}

		// Just verify command succeeds without error
	})

	t.Run("status with untracked files", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create untracked files
		h.WriteFile("file1.txt", "content 1")
		h.WriteFile("file2.txt", "content 2")

		// Run status command
		cmd := newStatusCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("status command failed: %v", err)
		}
	})

	t.Run("status with modified files", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create, add, and commit a file
		h.WriteFile("test.txt", "original content")

		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		// Commit the file
		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// Modify the file
		h.WriteFile("test.txt", "modified content")

		// Run status command
		cmd := newStatusCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("status command failed: %v", err)
		}
	})

	t.Run("status with deleted files", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create, add, and commit a file
		h.WriteFile("test.txt", "content")

		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		// Commit the file
		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// Delete the file
		if err := os.Remove("test.txt"); err != nil {
			t.Fatalf("failed to delete file: %v", err)
		}

		// Run status command
		cmd := newStatusCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("status command failed: %v", err)
		}
	})

	t.Run("status with staged files", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create and stage a file
		h.WriteFile("test.txt", "content")

		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		// Run status command
		cmd := newStatusCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("status command failed: %v", err)
		}
	})

	t.Run("status without repository fails", func(t *testing.T) {
		h := NewTestHelper(t)
		// Don't initialize repo
		h.Chdir()
		defer os.Chdir(origDir)

		// Run status command
		cmd := newStatusCmd()
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when running status outside repository")
		}
	})

	t.Run("status with mixed changes", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create multiple files with different states
		h.WriteFile("committed.txt", "committed content")
		h.WriteFile("modified.txt", "original content")
		h.WriteFile("staged.txt", "staged content")

		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		// Add and commit some files
		if _, err := indexMgr.Add([]string{"committed.txt", "modified.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add files: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// Modify one file
		h.WriteFile("modified.txt", "modified content")

		// Stage a new file
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to reinitialize index: %v", err)
		}
		if _, err := indexMgr.Add([]string{"staged.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add staged file: %v", err)
		}

		// Create an untracked file
		h.WriteFile("untracked.txt", "untracked content")

		// Run status command
		cmd := newStatusCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("status command failed: %v", err)
		}
	})

	t.Run("status in subdirectory", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create nested directory structure
		h.WriteFile("subdir/file.txt", "nested content")

		// Change to subdirectory
		if err := os.Chdir("subdir"); err != nil {
			t.Fatalf("failed to change to subdirectory: %v", err)
		}
		defer os.Chdir(h.TempDir())

		// Run status command from subdirectory
		cmd := newStatusCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("status command failed: %v", err)
		}
	})

	t.Run("status after successful commit shows clean", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create, add, and commit file
		h.WriteFile("test.txt", "content")

		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Test commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// Run status command
		cmd := newStatusCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("status command failed: %v", err)
		}

		// Just verify command succeeds without error
	})
}
