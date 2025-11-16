package main

import (
	"context"
	"os"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

func TestCommitCommand(t *testing.T) {
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

	t.Run("commit with staged files", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create and stage a file
		h.WriteFile("test.txt", "hello world")

		// Add file to staging
		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		// Run commit command
		cmd := newCommitCmd()
		cmd.SetArgs([]string{"-m", "Initial commit"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("commit command failed: %v", err)
		}

		// Verify commit was created
		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		history, err := commitMgr.GetHistory(ctx, objects.ObjectHash(""), 10)
		if err != nil {
			t.Fatalf("failed to get history: %v", err)
		}

		if len(history) != 1 {
			t.Errorf("expected 1 commit, got %d", len(history))
		}

		if history[0].Message != "Initial commit" {
			t.Errorf("expected message 'Initial commit', got '%s'", history[0].Message)
		}
	})

	t.Run("commit multiple files", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create and stage multiple files
		h.WriteFile("file1.txt", "content 1")
		h.WriteFile("file2.txt", "content 2")
		h.WriteFile("file3.txt", "content 3")

		// Add files to staging
		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"file1.txt", "file2.txt", "file3.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add files: %v", err)
		}

		// Run commit command
		cmd := newCommitCmd()
		cmd.SetArgs([]string{"-m", "Add multiple files"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("commit command failed: %v", err)
		}

		// Verify commit exists
		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		history, err := commitMgr.GetHistory(ctx, objects.ObjectHash(""), 10)
		if err != nil {
			t.Fatalf("failed to get history: %v", err)
		}

		if len(history) != 1 {
			t.Errorf("expected 1 commit, got %d", len(history))
		}
	})

	t.Run("commit chain with parent", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Set up index manager and object store
		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		// First commit
		h.WriteFile("file1.txt", "first")
		if _, err := indexMgr.Add([]string{"file1.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file1: %v", err)
		}

		cmd1 := newCommitCmd()
		cmd1.SetArgs([]string{"-m", "First commit"})
		if err := cmd1.Execute(); err != nil {
			t.Fatalf("first commit failed: %v", err)
		}

		// Second commit
		h.WriteFile("file2.txt", "second")
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to reinitialize index: %v", err)
		}
		if _, err := indexMgr.Add([]string{"file2.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file2: %v", err)
		}

		cmd2 := newCommitCmd()
		cmd2.SetArgs([]string{"-m", "Second commit"})
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("second commit failed: %v", err)
		}

		// Verify commit chain
		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		history, err := commitMgr.GetHistory(ctx, objects.ObjectHash(""), 10)
		if err != nil {
			t.Fatalf("failed to get history: %v", err)
		}

		if len(history) != 2 {
			t.Errorf("expected 2 commits, got %d", len(history))
		}

		// Verify parent relationship
		if len(history[0].ParentSHAs) != 1 {
			t.Errorf("expected second commit to have 1 parent, got %d", len(history[0].ParentSHAs))
		}
	})

	t.Run("commit without message fails", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Run commit without -m flag
		cmd := newCommitCmd()
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when committing without message")
		}
	})

	t.Run("commit without staged files fails", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Try to commit without staging anything
		cmd := newCommitCmd()
		cmd.SetArgs([]string{"-m", "Empty commit"})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when committing without staged files")
		}
	})

	t.Run("commit without repository fails", func(t *testing.T) {
		h := NewTestHelper(t)
		// Don't initialize repo
		h.Chdir()
		defer os.Chdir(origDir)

		// Try to commit
		cmd := newCommitCmd()
		cmd.SetArgs([]string{"-m", "Test commit"})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when committing outside repository")
		}
	})
}
