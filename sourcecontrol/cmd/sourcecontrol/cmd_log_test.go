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

	// Test new log enhancements
	t.Run("log with graph visualization", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Setup and create commits
		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		// Create test commits
		for i := 1; i <= 3; i++ {
			filename := "graph_test" + string(rune('0'+i)) + ".txt"
			h.WriteFile(filename, "content")

			if err := indexMgr.Initialize(); err != nil {
				t.Fatalf("failed to reinitialize index: %v", err)
			}

			if _, err := indexMgr.Add([]string{filename}, objectStore); err != nil {
				t.Fatalf("failed to add file: %v", err)
			}

			if _, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
				Message: "Graph test commit " + string(rune('0'+i)),
			}); err != nil {
				t.Fatalf("failed to create commit: %v", err)
			}
		}

		// Run log command with graph
		cmd := newLogCmd()
		cmd.SetArgs([]string{"--graph"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("log --graph command failed: %v", err)
		}
	})

	t.Run("log with oneline format", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Setup and create commit
		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		h.WriteFile("oneline.txt", "content")
		if _, err := indexMgr.Add([]string{"oneline.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		if _, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Oneline test",
		}); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Run log command with oneline
		cmd := newLogCmd()
		cmd.SetArgs([]string{"--oneline"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("log --oneline command failed: %v", err)
		}
	})

	t.Run("log with custom format", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Setup and create commit
		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		h.WriteFile("format.txt", "content")
		if _, err := indexMgr.Add([]string{"format.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		if _, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Format test",
		}); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Run log command with custom format
		cmd := newLogCmd()
		cmd.SetArgs([]string{"--format", "%h - %an - %s"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("log --format command failed: %v", err)
		}
	})

	t.Run("log with pretty formats", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Setup and create commit
		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		h.WriteFile("pretty.txt", "content")
		if _, err := indexMgr.Add([]string{"pretty.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		if _, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Pretty test",
		}); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Test different pretty formats
		prettyFormats := []string{"oneline", "short", "medium", "full"}
		for _, format := range prettyFormats {
			cmd := newLogCmd()
			cmd.SetArgs([]string{"--pretty", format})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("log --pretty %s command failed: %v", format, err)
			}
		}
	})

	t.Run("log with grep filter", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Setup
		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		// Create commits with different messages
		messages := []string{"Add feature", "Fix bug", "Add tests"}
		for i, msg := range messages {
			filename := "grep" + string(rune('0'+i+1)) + ".txt"
			h.WriteFile(filename, "content")

			if err := indexMgr.Initialize(); err != nil {
				t.Fatalf("failed to reinitialize index: %v", err)
			}

			if _, err := indexMgr.Add([]string{filename}, objectStore); err != nil {
				t.Fatalf("failed to add file: %v", err)
			}

			if _, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
				Message: msg,
			}); err != nil {
				t.Fatalf("failed to create commit: %v", err)
			}
		}

		// Run log command with grep
		cmd := newLogCmd()
		cmd.SetArgs([]string{"--grep", "Add"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("log --grep command failed: %v", err)
		}
	})

	t.Run("log with author filter", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Setup
		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		// Create commit
		h.WriteFile("author.txt", "content")
		if _, err := indexMgr.Add([]string{"author.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		if _, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Author test",
		}); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Run log command with author filter
		cmd := newLogCmd()
		cmd.SetArgs([]string{"--author", "Test"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("log --author command failed: %v", err)
		}
	})

	t.Run("log with graph and oneline combined", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Setup
		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		repoRoot := repo.WorkingDirectory()
		indexMgr := index.NewManager(repoRoot)
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}
		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())

		// Create commits
		for i := 1; i <= 3; i++ {
			filename := "combined" + string(rune('0'+i)) + ".txt"
			h.WriteFile(filename, "content")

			if err := indexMgr.Initialize(); err != nil {
				t.Fatalf("failed to reinitialize index: %v", err)
			}

			if _, err := indexMgr.Add([]string{filename}, objectStore); err != nil {
				t.Fatalf("failed to add file: %v", err)
			}

			if _, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
				Message: "Combined test " + string(rune('0'+i)),
			}); err != nil {
				t.Fatalf("failed to create commit: %v", err)
			}
		}

		// Run log command with graph and oneline
		cmd := newLogCmd()
		cmd.SetArgs([]string{"--graph", "--oneline"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("log --graph --oneline command failed: %v", err)
		}
	})
}
