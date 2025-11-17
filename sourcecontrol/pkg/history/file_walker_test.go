package history

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

func TestFileHistoryWalker_FilterByFile(t *testing.T) {
	// Set up git config for commits
	os.Setenv("GIT_AUTHOR_NAME", "Test User")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	defer os.Unsetenv("GIT_AUTHOR_NAME")
	defer os.Unsetenv("GIT_AUTHOR_EMAIL")

	t.Run("file modified in multiple commits", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		repo := sourcerepo.NewSourceRepository()
		repoPath := scpath.RepositoryPath(tmpDir)
		if err := repo.Initialize(repoPath); err != nil {
			t.Fatalf("failed to initialize repo: %v", err)
		}

		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		indexMgr := index.NewManager(scpath.RepositoryPath(tmpDir))
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(scpath.RepositoryPath(tmpDir))

		// Create file and make first commit
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content1"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		_, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "First commit",
		})
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Modify file and make second commit
		if err := os.WriteFile(testFile, []byte("content2"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to reinitialize index: %v", err)
		}

		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		_, err = commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Second commit",
		})
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Get history
		history, err := commitMgr.GetHistory(ctx, "", 10)
		if err != nil {
			t.Fatalf("failed to get history: %v", err)
		}

		// Filter by file
		walker := NewFileHistoryWalker(repo)
		filtered, err := walker.FilterByFile(ctx, history, "test.txt")
		if err != nil {
			t.Fatalf("failed to filter by file: %v", err)
		}

		// Should have 2 commits (both modified test.txt)
		if len(filtered) != 2 {
			t.Errorf("expected 2 commits, got %d", len(filtered))
		}
	})

	t.Run("file in subdirectory", func(t *testing.T) {
		tmpDir := t.TempDir()

		repo := sourcerepo.NewSourceRepository()
		repoPath := scpath.RepositoryPath(tmpDir)
		if err := repo.Initialize(repoPath); err != nil {
			t.Fatalf("failed to initialize repo: %v", err)
		}

		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		indexMgr := index.NewManager(scpath.RepositoryPath(tmpDir))
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(scpath.RepositoryPath(tmpDir))

		// Create directory and file
		srcDir := filepath.Join(tmpDir, "src")
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		srcFile := filepath.Join(srcDir, "main.go")
		if err := os.WriteFile(srcFile, []byte("package main"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		if _, err := indexMgr.Add([]string{"src/main.go"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		_, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Add main.go",
		})
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Get history and filter
		history, err := commitMgr.GetHistory(ctx, "", 10)
		if err != nil {
			t.Fatalf("failed to get history: %v", err)
		}

		walker := NewFileHistoryWalker(repo)
		filtered, err := walker.FilterByFile(ctx, history, "src/main.go")
		if err != nil {
			t.Fatalf("failed to filter by file: %v", err)
		}

		if len(filtered) != 1 {
			t.Errorf("expected 1 commit, got %d", len(filtered))
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()

		repo := sourcerepo.NewSourceRepository()
		repoPath := scpath.RepositoryPath(tmpDir)
		if err := repo.Initialize(repoPath); err != nil {
			t.Fatalf("failed to initialize repo: %v", err)
		}

		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		indexMgr := index.NewManager(scpath.RepositoryPath(tmpDir))
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(scpath.RepositoryPath(tmpDir))

		// Create a different file
		testFile := filepath.Join(tmpDir, "other.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		if _, err := indexMgr.Add([]string{"other.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		_, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Add other.txt",
		})
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Filter by non-existent file
		history, err := commitMgr.GetHistory(ctx, "", 10)
		if err != nil {
			t.Fatalf("failed to get history: %v", err)
		}

		walker := NewFileHistoryWalker(repo)
		filtered, err := walker.FilterByFile(ctx, history, "does_not_exist.txt")
		if err != nil {
			t.Fatalf("failed to filter by file: %v", err)
		}

		if len(filtered) != 0 {
			t.Errorf("expected 0 commits, got %d", len(filtered))
		}
	})

	t.Run("empty file path returns all commits", func(t *testing.T) {
		tmpDir := t.TempDir()

		repo := sourcerepo.NewSourceRepository()
		repoPath := scpath.RepositoryPath(tmpDir)
		if err := repo.Initialize(repoPath); err != nil {
			t.Fatalf("failed to initialize repo: %v", err)
		}

		ctx := context.Background()
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize commit manager: %v", err)
		}

		indexMgr := index.NewManager(scpath.RepositoryPath(tmpDir))
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(scpath.RepositoryPath(tmpDir))

		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		_, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Test commit",
		})
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		history, err := commitMgr.GetHistory(ctx, "", 10)
		if err != nil {
			t.Fatalf("failed to get history: %v", err)
		}

		walker := NewFileHistoryWalker(repo)
		filtered, err := walker.FilterByFile(ctx, history, "")
		if err != nil {
			t.Fatalf("failed to filter by file: %v", err)
		}

		// Empty path should return all commits
		if len(filtered) != len(history) {
			t.Errorf("expected %d commits, got %d", len(history), len(filtered))
		}
	})
}
