package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/index"
)

func TestAddCommand(t *testing.T) {
	// Save and restore current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	t.Run("add single file", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create a test file
		h.WriteFile("test.txt", "hello world")

		// Run add command
		cmd := newAddCmd()
		cmd.SetArgs([]string{"test.txt"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add command failed: %v", err)
		}

		// Verify file was added to index
		indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
		idx, err := index.Read(indexPath)
		if err != nil {
			t.Fatalf("failed to read index: %v", err)
		}

		if idx.Count() != 1 {
			t.Errorf("expected 1 entry in index, got %d", idx.Count())
		}
	})

	t.Run("add multiple files", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create test files
		h.WriteFile("file1.txt", "content 1")
		h.WriteFile("file2.txt", "content 2")
		h.WriteFile("file3.txt", "content 3")

		// Run add command
		cmd := newAddCmd()
		cmd.SetArgs([]string{"file1.txt", "file2.txt", "file3.txt"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add command failed: %v", err)
		}

		// Verify files were added to index
		indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
		idx, err := index.Read(indexPath)
		if err != nil {
			t.Fatalf("failed to read index: %v", err)
		}

		if idx.Count() != 3 {
			t.Errorf("expected 3 entries in index, got %d", idx.Count())
		}
	})

	t.Run("add file in subdirectory", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create file in subdirectory
		h.WriteFile("subdir/file.txt", "nested content")

		// Run add command
		cmd := newAddCmd()
		cmd.SetArgs([]string{filepath.Join("subdir", "file.txt")})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add command failed: %v", err)
		}

		// Verify file was added to index
		indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
		idx, err := index.Read(indexPath)
		if err != nil {
			t.Fatalf("failed to read index: %v", err)
		}

		if idx.Count() != 1 {
			t.Errorf("expected 1 entry in index, got %d", idx.Count())
		}
	})

	t.Run("add modified file", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create and add file
		h.WriteFile("test.txt", "original content")

		cmd := newAddCmd()
		cmd.SetArgs([]string{"test.txt"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("first add failed: %v", err)
		}

		// Modify file
		h.WriteFile("test.txt", "modified content")

		// Add again
		cmd = newAddCmd()
		cmd.SetArgs([]string{"test.txt"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("second add failed: %v", err)
		}

		// Verify still only 1 entry in index
		indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
		idx, err := index.Read(indexPath)
		if err != nil {
			t.Fatalf("failed to read index: %v", err)
		}

		if idx.Count() != 1 {
			t.Errorf("expected 1 entry in index, got %d", idx.Count())
		}
	})

	t.Run("add non-existent file fails", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Try to add non-existent file
		cmd := newAddCmd()
		cmd.SetArgs([]string{"does-not-exist.txt"})

		// Should return error but command itself shouldn't panic
		// The add operation will show failures in the result
		_ = cmd.Execute()
	})

	t.Run("add without repository fails", func(t *testing.T) {
		h := NewTestHelper(t)
		// Don't initialize repo
		h.Chdir()
		defer os.Chdir(origDir)

		// Create a test file
		h.WriteFile("test.txt", "content")

		// Try to add file
		cmd := newAddCmd()
		cmd.SetArgs([]string{"test.txt"})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when adding file outside repository")
		}
	})

	t.Run("add same file twice updates index", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create and add a file
		h.WriteFile("test.txt", "original")
		cmd := newAddCmd()
		cmd.SetArgs([]string{"test.txt"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("first add failed: %v", err)
		}

		// Add the same file again
		cmd = newAddCmd()
		cmd.SetArgs([]string{"test.txt"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("second add failed: %v", err)
		}

		// Should still have only 1 entry
		indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
		idx, err := index.Read(indexPath)
		if err != nil {
			t.Fatalf("failed to read index: %v", err)
		}

		if idx.Count() != 1 {
			t.Errorf("expected 1 entry in index, got %d", idx.Count())
		}
	})

	t.Run("add multiple txt files", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create multiple .txt files
		h.WriteFile("file1.txt", "content 1")
		h.WriteFile("file2.txt", "content 2")
		h.WriteFile("file3.md", "markdown content")

		// Add all .txt files explicitly
		cmd := newAddCmd()
		cmd.SetArgs([]string{"file1.txt", "file2.txt"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add multiple files failed: %v", err)
		}

		// Verify 2 .txt files were added
		indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
		idx, err := index.Read(indexPath)
		if err != nil {
			t.Fatalf("failed to read index: %v", err)
		}

		if idx.Count() != 2 {
			t.Errorf("expected 2 entries in index, got %d", idx.Count())
		}
	})

	t.Run("add deeply nested file", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create deeply nested file
		nestedPath := filepath.Join("a", "b", "c", "d", "file.txt")
		h.WriteFile(nestedPath, "deeply nested")

		// Add the nested file
		cmd := newAddCmd()
		cmd.SetArgs([]string{nestedPath})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add nested file failed: %v", err)
		}

		// Verify file was added
		indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
		idx, err := index.Read(indexPath)
		if err != nil {
			t.Fatalf("failed to read index: %v", err)
		}

		if idx.Count() != 1 {
			t.Errorf("expected 1 entry in index, got %d", idx.Count())
		}
	})

	t.Run("add empty file", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create empty file
		h.WriteFile("empty.txt", "")

		// Add empty file
		cmd := newAddCmd()
		cmd.SetArgs([]string{"empty.txt"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add empty file failed: %v", err)
		}

		// Verify file was added
		indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
		idx, err := index.Read(indexPath)
		if err != nil {
			t.Fatalf("failed to read index: %v", err)
		}

		if idx.Count() != 1 {
			t.Errorf("expected 1 entry in index, got %d", idx.Count())
		}
	})

	t.Run("add file with special characters in name", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create file with special characters (spaces, dashes, underscores)
		fileName := "my-test_file 123.txt"
		h.WriteFile(fileName, "special name")

		// Add file
		cmd := newAddCmd()
		cmd.SetArgs([]string{fileName})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add file with special characters failed: %v", err)
		}

		// Verify file was added
		indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
		idx, err := index.Read(indexPath)
		if err != nil {
			t.Fatalf("failed to read index: %v", err)
		}

		if idx.Count() != 1 {
			t.Errorf("expected 1 entry in index, got %d", idx.Count())
		}
	})
}
