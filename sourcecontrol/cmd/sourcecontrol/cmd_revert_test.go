package main

import (
	"context"
	"os"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

func TestRevertCommand(t *testing.T) {
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

	t.Run("revert single commit", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		ctx := context.Background()
		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		// Create initial commit
		h.WriteFile("file1.txt", "initial content")
		if _, err := indexMgr.Add([]string{"file1.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("initial commit failed: %v", err)
		}

		// Create second commit that we'll revert
		h.WriteFile("file2.txt", "second file")
		if _, err := indexMgr.Add([]string{"file2.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file2: %v", err)
		}

		commitCmd2 := newCommitCmd()
		commitCmd2.SetArgs([]string{"-m", "Add file2"})
		if err := commitCmd2.Execute(); err != nil {
			t.Fatalf("second commit failed: %v", err)
		}

		// Get the commit SHA of the second commit
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		history, err := commitMgr.GetHistory(ctx, "", 2)
		if err != nil {
			t.Fatalf("failed to get history: %v", err)
		}

		if len(history) < 2 {
			t.Fatalf("expected at least 2 commits, got %d", len(history))
		}

		secondCommitSHA := history[0].Hash()

		// Revert the second commit
		revertCmd := newRevertCmd()
		revertCmd.SetArgs([]string{secondCommitSHA.String()})

		if err := revertCmd.Execute(); err != nil {
			t.Fatalf("revert command failed: %v", err)
		}

		// Verify that we now have 3 commits (initial + second + revert)
		history, err = commitMgr.GetHistory(ctx, "", 10)
		if err != nil {
			t.Fatalf("failed to get history after revert: %v", err)
		}

		if len(history) != 3 {
			t.Errorf("expected 3 commits after revert, got %d", len(history))
		}

		// Verify the revert commit message
		revertCommit := history[0]
		if !containsString(revertCommit.Message, "Revert") {
			t.Errorf("expected revert commit message to contain 'Revert', got: %s", revertCommit.Message)
		}
	})

	t.Run("revert with no-commit flag", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		ctx := context.Background()
		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		// Create initial commit
		h.WriteFile("file1.txt", "initial content")
		if _, err := indexMgr.Add([]string{"file1.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("initial commit failed: %v", err)
		}

		// Create second commit
		h.WriteFile("file2.txt", "second file")
		if _, err := indexMgr.Add([]string{"file2.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file2: %v", err)
		}

		commitCmd2 := newCommitCmd()
		commitCmd2.SetArgs([]string{"-m", "Add file2"})
		if err := commitCmd2.Execute(); err != nil {
			t.Fatalf("second commit failed: %v", err)
		}

		// Get the commit SHA of the second commit
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		history, err := commitMgr.GetHistory(ctx, "", 2)
		if err != nil {
			t.Fatalf("failed to get history: %v", err)
		}

		secondCommitSHA := history[0].Hash()

		// Revert with --no-commit flag
		revertCmd := newRevertCmd()
		revertCmd.SetArgs([]string{"--no-commit", secondCommitSHA.String()})

		if err := revertCmd.Execute(); err != nil {
			t.Fatalf("revert --no-commit failed: %v", err)
		}

		// Verify that we still have only 2 commits (no new commit created)
		history, err = commitMgr.GetHistory(ctx, "", 10)
		if err != nil {
			t.Fatalf("failed to get history after revert: %v", err)
		}

		if len(history) != 2 {
			t.Errorf("expected 2 commits after revert --no-commit, got %d", len(history))
		}
	})

	t.Run("revert initial commit should fail", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		ctx := context.Background()
		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		// Create initial commit
		h.WriteFile("file1.txt", "initial content")
		if _, err := indexMgr.Add([]string{"file1.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("initial commit failed: %v", err)
		}

		// Get the initial commit SHA
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		history, err := commitMgr.GetHistory(ctx, "", 1)
		if err != nil {
			t.Fatalf("failed to get history: %v", err)
		}

		initialCommitSHA := history[0].Hash()

		// Try to revert the initial commit (should fail)
		revertCmd := newRevertCmd()
		revertCmd.SetArgs([]string{initialCommitSHA.String()})

		if err := revertCmd.Execute(); err == nil {
			t.Error("expected revert of initial commit to fail, but it succeeded")
		}
	})
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
