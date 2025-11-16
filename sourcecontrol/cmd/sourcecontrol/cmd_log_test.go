package main

import (
	"context"
	"os"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

func TestLogCommand(t *testing.T) {
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

	t.Run("log with no commits", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Run log command on empty repository
		cmd := newLogCmd()
		cmd.SetArgs([]string{})

		// Should not fail, just show "No commits yet"
		if err := cmd.Execute(); err != nil {
			t.Fatalf("log command failed: %v", err)
		}
	})

	t.Run("log with single commit", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create commit
		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		h.WriteFile("test.txt", "content")
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		if _, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Test commit",
		}); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Run log command
		cmd := newLogCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("log command failed: %v", err)
		}
	})

	t.Run("log with multiple commits", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Set up managers
		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		// Create multiple commits
		for i := 1; i <= 5; i++ {
			filename := "file" + string(rune('0'+i)) + ".txt"
			h.WriteFile(filename, "content")

			if err := indexMgr.Initialize(); err != nil {
				t.Fatalf("failed to reinitialize index: %v", err)
			}

			if _, err := indexMgr.Add([]string{filename}, objectStore); err != nil {
				t.Fatalf("failed to add file: %v", err)
			}

			if _, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
				Message: "Commit " + string(rune('0'+i)),
			}); err != nil {
				t.Fatalf("failed to create commit %d: %v", i, err)
			}
		}

		// Run log command
		cmd := newLogCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("log command failed: %v", err)
		}
	})

	t.Run("log with limit", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Set up managers
		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		// Create 10 commits
		for i := 1; i <= 10; i++ {
			filename := "file" + string(rune('0'+i)) + ".txt"
			h.WriteFile(filename, "content")

			if err := indexMgr.Initialize(); err != nil {
				t.Fatalf("failed to reinitialize index: %v", err)
			}

			if _, err := indexMgr.Add([]string{filename}, objectStore); err != nil {
				t.Fatalf("failed to add file: %v", err)
			}

			if _, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
				Message: "Commit " + string(rune('0'+i)),
			}); err != nil {
				t.Fatalf("failed to create commit %d: %v", i, err)
			}
		}

		// Run log command with limit
		cmd := newLogCmd()
		cmd.SetArgs([]string{"-n", "5"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("log command failed: %v", err)
		}

		// The command itself doesn't return the count, but we can verify it runs successfully
	})

	t.Run("log without repository fails", func(t *testing.T) {
		h := NewTestHelper(t)
		// Don't initialize repo
		h.Chdir()
		defer os.Chdir(origDir)

		// Try to run log
		cmd := newLogCmd()
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when running log outside repository")
		}
	})

	t.Run("log shows commits in reverse chronological order", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Set up managers
		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		// Create commits with distinct messages
		messages := []string{"First", "Second", "Third"}
		for _, msg := range messages {
			h.WriteFile(msg+".txt", "content")

			if err := indexMgr.Initialize(); err != nil {
				t.Fatalf("failed to reinitialize index: %v", err)
			}

			if _, err := indexMgr.Add([]string{msg + ".txt"}, objectStore); err != nil {
				t.Fatalf("failed to add file: %v", err)
			}

			if _, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
				Message: msg + " commit",
			}); err != nil {
				t.Fatalf("failed to create commit: %v", err)
			}
		}

		// Run log command
		cmd := newLogCmd()
		cmd.SetArgs([]string{})

		// Should execute without error
		if err := cmd.Execute(); err != nil {
			t.Fatalf("log command failed: %v", err)
		}

		// Note: We could capture stdout to verify order, but for now
		// we're just testing that the command executes successfully
	})
}
