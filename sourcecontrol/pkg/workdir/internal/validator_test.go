package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/common"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/blob"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

func TestFileStatus_String(t *testing.T) {
	tests := []struct {
		status   FileStatus
		expected string
	}{
		{FileDeleted, "deleted"},
		{FileSizeChanged, "size-changed"},
		{FileContentChanged, "content-changed"},
		{FileTimeChanged, "time-changed"},
		{FileStatus(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("FileStatus.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewValidator(t *testing.T) {
	workDir := scpath.RepositoryPath("/tmp/test")
	v := NewValidator(workDir)

	if v == nil {
		t.Fatal("NewValidator() returned nil")
	}
	if v.workDir != workDir {
		t.Errorf("workDir = %v, want %v", v.workDir, workDir)
	}
}

func TestValidator_ValidateCleanState(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "validator-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath, err := scpath.NewRepositoryPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create repository path: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("hello world")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Get file info
	stats, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	// Create blob and hash
	b := blob.NewBlob(testContent)
	hash, err := b.Hash()
	if err != nil {
		t.Fatalf("Failed to hash blob: %v", err)
	}

	// Create index with matching entry
	idx := index.NewIndex()
	relPath, _ := scpath.NewRelativePath("test.txt")
	entry := &index.Entry{
		Path:             relPath,
		BlobHash:         hash,
		SizeInBytes:      uint32(stats.Size()),
		ModificationTime: common.NewTimestampFromTime(stats.ModTime()),
	}
	idx.Add(entry)

	validator := NewValidator(repoPath)

	t.Run("clean state - no changes", func(t *testing.T) {
		status, err := validator.ValidateCleanState(idx)
		if err != nil {
			t.Fatalf("ValidateCleanState() error = %v", err)
		}
		if !status.Clean {
			t.Error("Expected clean state")
		}
		if len(status.ModifiedFiles) != 0 {
			t.Errorf("Expected 0 modified files, got %d", len(status.ModifiedFiles))
		}
		if len(status.DeletedFiles) != 0 {
			t.Errorf("Expected 0 deleted files, got %d", len(status.DeletedFiles))
		}
	})

	t.Run("file modified", func(t *testing.T) {
		// Modify file
		newContent := []byte("modified content")
		if err := os.WriteFile(testFile, newContent, 0644); err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}

		status, err := validator.ValidateCleanState(idx)
		if err != nil {
			t.Fatalf("ValidateCleanState() error = %v", err)
		}
		if status.Clean {
			t.Error("Expected dirty state")
		}
		if len(status.ModifiedFiles) != 1 {
			t.Errorf("Expected 1 modified file, got %d", len(status.ModifiedFiles))
		}
		if len(status.Details) != 1 {
			t.Errorf("Expected 1 detail, got %d", len(status.Details))
		}
		// File can be detected as either size-changed or content-changed
		if len(status.Details) > 0 {
			s := status.Details[0].Status
			if s != FileContentChanged && s != FileSizeChanged {
				t.Errorf("Expected FileContentChanged or FileSizeChanged, got %v", s)
			}
		}

		// Restore original content for next test
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("Failed to restore test file: %v", err)
		}
	})

	t.Run("file deleted", func(t *testing.T) {
		// Delete file
		if err := os.Remove(testFile); err != nil {
			t.Fatalf("Failed to delete test file: %v", err)
		}

		status, err := validator.ValidateCleanState(idx)
		if err != nil {
			t.Fatalf("ValidateCleanState() error = %v", err)
		}
		if status.Clean {
			t.Error("Expected dirty state")
		}
		if len(status.DeletedFiles) != 1 {
			t.Errorf("Expected 1 deleted file, got %d", len(status.DeletedFiles))
		}
		if len(status.Details) != 1 {
			t.Errorf("Expected 1 detail, got %d", len(status.Details))
		}
		if len(status.Details) > 0 && status.Details[0].Status != FileDeleted {
			t.Errorf("Expected FileDeleted, got %v", status.Details[0].Status)
		}
	})
}

func TestValidator_CanSafelyOverwrite(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "validator-overwrite-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath, err := scpath.NewRepositoryPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create repository path: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("original content")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Get file info
	stats, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	// Create blob and hash
	b := blob.NewBlob(testContent)
	hash, err := b.Hash()
	if err != nil {
		t.Fatalf("Failed to hash blob: %v", err)
	}

	// Create index
	idx := index.NewIndex()
	relPath, _ := scpath.NewRelativePath("test.txt")
	entry := &index.Entry{
		Path:             relPath,
		BlobHash:         hash,
		SizeInBytes:      uint32(stats.Size()),
		ModificationTime: common.NewTimestampFromTime(stats.ModTime()),
	}
	idx.Add(entry)

	validator := NewValidator(repoPath)

	t.Run("safe to overwrite - no changes", func(t *testing.T) {
		paths := []scpath.RelativePath{relPath}
		err := validator.CanSafelyOverwrite(paths, idx)
		if err != nil {
			t.Errorf("CanSafelyOverwrite() error = %v, expected nil", err)
		}
	})

	t.Run("safe to overwrite - only time changed", func(t *testing.T) {
		// Touch file to change modification time but keep same content
		time.Sleep(time.Second) // Ensure time difference
		now := time.Now()
		if err := os.Chtimes(testFile, now, now); err != nil {
			t.Fatalf("Failed to change file time: %v", err)
		}

		paths := []scpath.RelativePath{relPath}
		err := validator.CanSafelyOverwrite(paths, idx)
		if err != nil {
			t.Errorf("CanSafelyOverwrite() error = %v, expected nil for time-only change", err)
		}
	})

	t.Run("not safe to overwrite - content changed", func(t *testing.T) {
		// Modify file content
		modifiedContent := []byte("modified content")
		if err := os.WriteFile(testFile, modifiedContent, 0644); err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}

		paths := []scpath.RelativePath{relPath}
		err := validator.CanSafelyOverwrite(paths, idx)
		if err == nil {
			t.Error("CanSafelyOverwrite() expected error for modified file, got nil")
		}
	})

	t.Run("path not in index", func(t *testing.T) {
		nonExistentPath, _ := scpath.NewRelativePath("nonexistent.txt")
		paths := []scpath.RelativePath{nonExistentPath}
		err := validator.CanSafelyOverwrite(paths, idx)
		if err != nil {
			t.Errorf("CanSafelyOverwrite() error = %v, expected nil for non-existent path", err)
		}
	})
}

func TestValidator_checkFileStatus(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "validator-check-status-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath, err := scpath.NewRepositoryPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create repository path: %v", err)
	}

	validator := NewValidator(repoPath)

	t.Run("file deleted", func(t *testing.T) {
		relPath, _ := scpath.NewRelativePath("deleted.txt")
		entry := &index.Entry{
			Path:             relPath,
			BlobHash:         objects.ObjectHash("dummy"),
			SizeInBytes:      100,
			ModificationTime: common.NewTimestampFromTime(time.Now()),
		}

		detail, err := validator.checkFileStatus(entry)
		if err != nil {
			t.Fatalf("checkFileStatus() error = %v", err)
		}
		if detail == nil {
			t.Fatal("Expected detail, got nil")
		}
		if detail.Status != FileDeleted {
			t.Errorf("Status = %v, want FileDeleted", detail.Status)
		}
	})

	t.Run("file size changed", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tmpDir, "size_test.txt")
		testContent := []byte("original")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		relPath, _ := scpath.NewRelativePath("size_test.txt")
		entry := &index.Entry{
			Path:             relPath,
			BlobHash:         objects.ObjectHash("dummy"),
			SizeInBytes:      999, // Different size
			ModificationTime: common.NewTimestampFromTime(time.Now()),
		}

		detail, err := validator.checkFileStatus(entry)
		if err != nil {
			t.Fatalf("checkFileStatus() error = %v", err)
		}
		if detail == nil {
			t.Fatal("Expected detail, got nil")
		}
		if detail.Status != FileSizeChanged {
			t.Errorf("Status = %v, want FileSizeChanged", detail.Status)
		}
	})
}

func TestValidator_isContentModified(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "validator-content-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath, err := scpath.NewRepositoryPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create repository path: %v", err)
	}

	validator := NewValidator(repoPath)

	t.Run("content not modified", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tmpDir, "content_test.txt")
		testContent := []byte("test content")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Create blob and hash
		b := blob.NewBlob(testContent)
		hash, err := b.Hash()
		if err != nil {
			t.Fatalf("Failed to hash blob: %v", err)
		}

		relPath, _ := scpath.NewRelativePath("content_test.txt")
		entry := &index.Entry{
			Path:     relPath,
			BlobHash: hash,
		}

		modified, currentHash, err := validator.isContentModified(entry)
		if err != nil {
			t.Fatalf("isContentModified() error = %v", err)
		}
		if modified {
			t.Error("Expected content not modified")
		}
		if currentHash != hash {
			t.Errorf("currentHash = %v, want %v", currentHash, hash)
		}
	})

	t.Run("content modified", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tmpDir, "modified_test.txt")
		originalContent := []byte("original")
		if err := os.WriteFile(testFile, originalContent, 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Create blob with original content
		b := blob.NewBlob(originalContent)
		originalHash, err := b.Hash()
		if err != nil {
			t.Fatalf("Failed to hash blob: %v", err)
		}

		// Modify file
		modifiedContent := []byte("modified")
		if err := os.WriteFile(testFile, modifiedContent, 0644); err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}

		relPath, _ := scpath.NewRelativePath("modified_test.txt")
		entry := &index.Entry{
			Path:     relPath,
			BlobHash: originalHash,
		}

		modified, currentHash, err := validator.isContentModified(entry)
		if err != nil {
			t.Fatalf("isContentModified() error = %v", err)
		}
		if !modified {
			t.Error("Expected content modified")
		}
		if currentHash == originalHash {
			t.Error("currentHash should differ from original")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		relPath, _ := scpath.NewRelativePath("nonexistent.txt")
		entry := &index.Entry{
			Path:     relPath,
			BlobHash: objects.ObjectHash("dummy"),
		}

		modified, _, err := validator.isContentModified(entry)
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
		if !modified {
			t.Error("Expected modified=true on error")
		}
	})
}
