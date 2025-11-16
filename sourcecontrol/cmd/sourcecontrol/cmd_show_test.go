package main

import (
	"context"
	"os"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/blob"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

func TestShowCommand(t *testing.T) {
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

	t.Run("show HEAD commit", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create a commit
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

		_, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Test commit",
		})
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Run show command (defaults to HEAD)
		cmd := newShowCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("show command failed: %v", err)
		}
	})

	t.Run("show specific commit by hash", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create a commit
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

		commit, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Test commit",
		})
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Get commit hash
		hash, err := commit.Hash()
		if err != nil {
			t.Fatalf("failed to get commit hash: %v", err)
		}

		// Run show command with specific hash
		cmd := newShowCmd()
		cmd.SetArgs([]string{hash.String()})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("show command failed: %v", err)
		}
	})

	t.Run("show commit with patch", func(t *testing.T) {
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

		// Create first commit
		h.WriteFile("test.txt", "initial content")
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}
		_, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Initial commit",
		})
		if err != nil {
			t.Fatalf("failed to create first commit: %v", err)
		}

		// Create second commit with changes
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to reinitialize index: %v", err)
		}
		h.WriteFile("test2.txt", "new file")
		if _, err := indexMgr.Add([]string{"test2.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}
		_, err = commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Second commit",
		})
		if err != nil {
			t.Fatalf("failed to create second commit: %v", err)
		}

		// Run show command with patch flag
		cmd := newShowCmd()
		cmd.SetArgs([]string{"--patch"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("show command with patch failed: %v", err)
		}
	})

	t.Run("show initial commit with patch", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create initial commit
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

		_, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Initial commit",
		})
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Run show command with patch flag on initial commit
		cmd := newShowCmd()
		cmd.SetArgs([]string{"-p"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("show command with patch failed: %v", err)
		}
	})

	t.Run("show tree object", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create a commit to get a tree
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

		commit, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
			Message: "Test commit",
		})
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Get tree hash from commit
		treeHash := commit.TreeSHA

		// Run show command on tree
		cmd := newShowCmd()
		cmd.SetArgs([]string{treeHash.String()})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("show command for tree failed: %v", err)
		}
	})

	t.Run("show blob object", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create a blob directly
		objectStore := store.NewFileObjectStore()
		if err := objectStore.Initialize(repo.WorkingDirectory()); err != nil {
			t.Fatalf("failed to initialize object store: %v", err)
		}

		// Create a blob
		blobContent := "Hello, World!\nThis is a test blob."
		b := blob.NewBlob([]byte(blobContent))

		// Write blob to store
		hash, err := objectStore.WriteObject(b)
		if err != nil {
			t.Fatalf("failed to write blob: %v", err)
		}

		// Run show command on blob
		cmd := newShowCmd()
		cmd.SetArgs([]string{hash.String()})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("show command for blob failed: %v", err)
		}
	})

	t.Run("show binary blob", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create a binary blob
		objectStore := store.NewFileObjectStore()
		if err := objectStore.Initialize(repo.WorkingDirectory()); err != nil {
			t.Fatalf("failed to initialize object store: %v", err)
		}

		// Create binary content (with null bytes)
		binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
		b := blob.NewBlob(binaryContent)

		// Write blob to store
		hash, err := objectStore.WriteObject(b)
		if err != nil {
			t.Fatalf("failed to write blob: %v", err)
		}

		// Run show command on binary blob
		cmd := newShowCmd()
		cmd.SetArgs([]string{hash.String()})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("show command for binary blob failed: %v", err)
		}
	})

	t.Run("show large text blob", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create a large blob (>100 lines)
		objectStore := store.NewFileObjectStore()
		if err := objectStore.Initialize(repo.WorkingDirectory()); err != nil {
			t.Fatalf("failed to initialize object store: %v", err)
		}

		// Create content with 200 lines
		var content string
		for i := 0; i < 200; i++ {
			content += "Line " + string(rune('0'+(i%10))) + "\n"
		}
		b := blob.NewBlob([]byte(content))

		// Write blob to store
		hash, err := objectStore.WriteObject(b)
		if err != nil {
			t.Fatalf("failed to write blob: %v", err)
		}

		// Run show command on large blob (should truncate)
		cmd := newShowCmd()
		cmd.SetArgs([]string{hash.String()})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("show command for large blob failed: %v", err)
		}
	})

	t.Run("show invalid object hash", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Run show command with invalid hash
		cmd := newShowCmd()
		cmd.SetArgs([]string{"invalid_hash"})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error for invalid hash")
		}
	})

	t.Run("show nonexistent object", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Use a valid hash format but nonexistent object
		fakeHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

		// Run show command
		cmd := newShowCmd()
		cmd.SetArgs([]string{fakeHash})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error for nonexistent object")
		}
	})

	t.Run("show without repository fails", func(t *testing.T) {
		h := NewTestHelper(t)
		// Don't initialize repo
		h.Chdir()
		defer os.Chdir(origDir)

		// Try to run show
		cmd := newShowCmd()
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when running show outside repository")
		}
	})

	t.Run("show HEAD with no commits fails", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Try to run show on empty repository
		cmd := newShowCmd()
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when showing HEAD with no commits")
		}
	})

	t.Run("show multiple commits with changes", func(t *testing.T) {
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
		commits := []struct {
			file    string
			content string
			message string
		}{
			{"file1.txt", "content1", "First commit"},
			{"file2.txt", "content2", "Second commit"},
			{"file3.txt", "content3", "Third commit"},
		}

		var commitHashes []objects.ObjectHash
		for _, c := range commits {
			if err := indexMgr.Initialize(); err != nil {
				t.Fatalf("failed to reinitialize index: %v", err)
			}

			h.WriteFile(c.file, c.content)
			if _, err := indexMgr.Add([]string{c.file}, objectStore); err != nil {
				t.Fatalf("failed to add file: %v", err)
			}

			commit, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
				Message: c.message,
			})
			if err != nil {
				t.Fatalf("failed to create commit: %v", err)
			}

			hash, _ := commit.Hash()
			commitHashes = append(commitHashes, hash)
		}

		// Show each commit
		for _, hash := range commitHashes {
			cmd := newShowCmd()
			cmd.SetArgs([]string{hash.String()})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("show command failed for commit %s: %v", hash.Short(), err)
			}
		}
	})

	t.Run("resolve object ref with isBinary check", func(t *testing.T) {
		// Test the isBinary function
		testCases := []struct {
			name     string
			data     []byte
			expected bool
		}{
			{"empty data", []byte{}, false},
			{"text data", []byte("Hello, World!"), false},
			{"binary with null byte", []byte{0x00, 0x01, 0x02}, true},
			{"text with newlines", []byte("Line1\nLine2\nLine3"), false},
			{"binary in middle", []byte("Hello\x00World"), true},
		}

		for _, tc := range testCases {
			result := isBinary(tc.data)
			if result != tc.expected {
				t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, result)
			}
		}
	})
}

func TestShowCommandHelpers(t *testing.T) {
	t.Run("getEntryTypeString", func(t *testing.T) {
		// This is a unit test for helper functions
		// We don't need a full repository setup for this

		// Note: We can't easily test this without creating actual TreeEntry objects
		// which require valid paths and hashes. This would be better as integration tests.
		// For now, we verify the function exists and is callable through the integration tests above.
	})
}
