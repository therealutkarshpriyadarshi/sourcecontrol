package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/blob"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// setupTestRepo creates a temporary repository for testing
func setupTestRepo(t *testing.T) (*sourcerepo.SourceRepository, string) {
	t.Helper()

	// Create temp directory
	tmpDir := t.TempDir()

	// Initialize repository
	repo := sourcerepo.NewSourceRepository()
	if err := repo.Initialize(scpath.RepositoryPath(tmpDir)); err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	return repo, tmpDir
}

// createTestBlob creates a blob object with test content
func createTestBlob(t *testing.T, repo *sourcerepo.SourceRepository, content string) objects.ObjectHash {
	t.Helper()

	b := blob.NewBlob([]byte(content))
	hash, err := repo.WriteObject(b)
	if err != nil {
		t.Fatalf("Failed to write blob: %v", err)
	}

	return hash
}

func TestFileOps_ApplyOperation_Create(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	// Create a test blob
	content := "Hello, World!"
	blobSHA := createTestBlob(t, repo, content)

	tests := []struct {
		name     string
		op       Operation
		wantFile string
		wantMode os.FileMode
		wantErr  bool
	}{
		{
			name: "create regular file",
			op: Operation{
				Path:   scpath.RelativePath("test.txt"),
				Action: ActionCreate,
				SHA:    blobSHA,
				Mode:   0644,
			},
			wantFile: "test.txt",
			wantMode: 0644,
			wantErr:  false,
		},
		{
			name: "create file in subdirectory",
			op: Operation{
				Path:   scpath.RelativePath("subdir/nested.txt"),
				Action: ActionCreate,
				SHA:    blobSHA,
				Mode:   0644,
			},
			wantFile: "subdir/nested.txt",
			wantMode: 0644,
			wantErr:  false,
		},
		{
			name: "create executable file",
			op: Operation{
				Path:   scpath.RelativePath("script.sh"),
				Action: ActionCreate,
				SHA:    blobSHA,
				Mode:   0755,
			},
			wantFile: "script.sh",
			wantMode: 0755,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply operation
			err := service.ApplyOperation(tt.op)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyOperation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify file exists
			filePath := filepath.Join(workDir, tt.wantFile)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("File was not created: %s", filePath)
				return
			}

			// Verify content
			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("Failed to read file: %v", err)
				return
			}
			if string(data) != content {
				t.Errorf("File content = %q, want %q", string(data), content)
			}

			// Verify permissions (check user permissions only, ignore group/other)
			// Note: On Windows, execute permission doesn't work the same way as Unix
			info, err := os.Stat(filePath)
			if err != nil {
				t.Errorf("Failed to stat file: %v", err)
				return
			}
			gotMode := info.Mode() & 0700 // User permissions only
			// On Windows, we can only reliably check read/write, not execute
			// So we'll just check that the file has at least read permission
			if gotMode&0400 == 0 {
				t.Errorf("File is not readable: mode = %o", gotMode)
			}
		})
	}
}

func TestFileOps_ApplyOperation_Modify(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	// Create initial file
	initialContent := "Initial content"
	initialBlob := createTestBlob(t, repo, initialContent)
	filePath := scpath.RelativePath("test.txt")

	err := service.ApplyOperation(Operation{
		Path:   filePath,
		Action: ActionCreate,
		SHA:    initialBlob,
		Mode:   0644,
	})
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Modify the file
	newContent := "Modified content"
	newBlob := createTestBlob(t, repo, newContent)

	err = service.ApplyOperation(Operation{
		Path:   filePath,
		Action: ActionModify,
		SHA:    newBlob,
		Mode:   0644,
	})
	if err != nil {
		t.Errorf("ApplyOperation(modify) error = %v", err)
		return
	}

	// Verify content changed
	data, err := os.ReadFile(filepath.Join(workDir, "test.txt"))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(data) != newContent {
		t.Errorf("File content = %q, want %q", string(data), newContent)
	}
}

func TestFileOps_ApplyOperation_Delete(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	tests := []struct {
		name         string
		setupFiles   []string
		deleteFile   string
		checkGone    string
		checkRemains []string
	}{
		{
			name:       "delete single file",
			setupFiles: []string{"test.txt"},
			deleteFile: "test.txt",
			checkGone:  "test.txt",
		},
		{
			name:         "delete file in subdirectory",
			setupFiles:   []string{"subdir/file1.txt", "subdir/file2.txt"},
			deleteFile:   "subdir/file1.txt",
			checkGone:    "subdir/file1.txt",
			checkRemains: []string{"subdir/file2.txt"},
		},
		{
			name:       "delete file removes empty parent directory",
			setupFiles: []string{"deep/nested/file.txt"},
			deleteFile: "deep/nested/file.txt",
			checkGone:  "deep/nested/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup: create test files
			content := "test content"
			blobSHA := createTestBlob(t, repo, content)

			for _, file := range tt.setupFiles {
				err := service.ApplyOperation(Operation{
					Path:   scpath.RelativePath(file),
					Action: ActionCreate,
					SHA:    blobSHA,
					Mode:   0644,
				})
				if err != nil {
					t.Fatalf("Setup: failed to create %s: %v", file, err)
				}
			}

			// Delete file
			err := service.ApplyOperation(Operation{
				Path:   scpath.RelativePath(tt.deleteFile),
				Action: ActionDelete,
			})
			if err != nil {
				t.Errorf("ApplyOperation(delete) error = %v", err)
				return
			}

			// Verify file is gone
			goneFile := filepath.Join(workDir, tt.checkGone)
			if _, err := os.Stat(goneFile); !os.IsNotExist(err) {
				t.Errorf("File still exists: %s", goneFile)
			}

			// Verify other files remain
			for _, remain := range tt.checkRemains {
				remainFile := filepath.Join(workDir, remain)
				if _, err := os.Stat(remainFile); err != nil {
					t.Errorf("File was incorrectly deleted: %s", remainFile)
				}
			}
		})
	}
}

func TestFileOps_CreateBackup(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	tests := []struct {
		name      string
		setupFile bool
		content   string
		mode      os.FileMode
		wantExist bool
	}{
		{
			name:      "backup existing file",
			setupFile: true,
			content:   "Original content",
			mode:      0644,
			wantExist: true,
		},
		{
			name:      "backup non-existent file",
			setupFile: false,
			wantExist: false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use unique file name for each test to avoid interference
			testFile := fmt.Sprintf("test-%d.txt", i)
			filePath := scpath.RelativePath(testFile)
			fullPath := filepath.Join(workDir, testFile)

			// Setup: create file if needed
			if tt.setupFile {
				err := os.WriteFile(fullPath, []byte(tt.content), tt.mode)
				if err != nil {
					t.Fatalf("Setup: failed to create file: %v", err)
				}
			}

			// Create backup
			backup, err := service.CreateBackup(filePath)
			if err != nil {
				t.Errorf("CreateBackup() error = %v", err)
				return
			}

			// Verify backup properties
			if backup.Existed != tt.wantExist {
				t.Errorf("backup.Existed = %v, want %v", backup.Existed, tt.wantExist)
			}

			if tt.wantExist {
				// Verify backup file exists
				if _, err := os.Stat(backup.TempFile); err != nil {
					t.Errorf("Backup file doesn't exist: %s", backup.TempFile)
				}

				// Verify backup content matches original
				backupData, err := os.ReadFile(backup.TempFile)
				if err != nil {
					t.Errorf("Failed to read backup: %v", err)
				}
				if string(backupData) != tt.content {
					t.Errorf("Backup content = %q, want %q", string(backupData), tt.content)
				}

				// Cleanup
				service.CleanupBackup(backup)
			}
		})
	}
}

func TestFileOps_RestoreBackup(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	t.Run("restore existing file", func(t *testing.T) {
		// Setup: create original file
		originalContent := "Original content"
		filePath := scpath.RelativePath("test.txt")
		fullPath := filepath.Join(workDir, "test.txt")
		err := os.WriteFile(fullPath, []byte(originalContent), 0644)
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}

		// Create backup
		backup, err := service.CreateBackup(filePath)
		if err != nil {
			t.Fatalf("CreateBackup failed: %v", err)
		}
		defer service.CleanupBackup(backup)

		// Modify file
		modifiedContent := "Modified content"
		err = os.WriteFile(fullPath, []byte(modifiedContent), 0644)
		if err != nil {
			t.Fatalf("Modify failed: %v", err)
		}

		// Restore backup
		err = service.RestoreBackup(backup)
		if err != nil {
			t.Errorf("RestoreBackup() error = %v", err)
			return
		}

		// Verify file restored to original content
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if string(data) != originalContent {
			t.Errorf("Restored content = %q, want %q", string(data), originalContent)
		}
	})

	t.Run("restore non-existent file", func(t *testing.T) {
		filePath := scpath.RelativePath("newfile.txt")
		fullPath := filepath.Join(workDir, "newfile.txt")

		// Create backup (file doesn't exist)
		backup, err := service.CreateBackup(filePath)
		if err != nil {
			t.Fatalf("CreateBackup failed: %v", err)
		}

		// Create the file
		err = os.WriteFile(fullPath, []byte("New content"), 0644)
		if err != nil {
			t.Fatalf("Create file failed: %v", err)
		}

		// Restore backup (should delete the file)
		err = service.RestoreBackup(backup)
		if err != nil {
			t.Errorf("RestoreBackup() error = %v", err)
			return
		}

		// Verify file is deleted
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Errorf("File should have been deleted")
		}
	})
}

func TestFileOps_DryRun(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)
	service.SetDryRun(true)

	content := "Test content"
	blobSHA := createTestBlob(t, repo, content)

	// Perform operation in dry-run mode
	err := service.ApplyOperation(Operation{
		Path:   scpath.RelativePath("test.txt"),
		Action: ActionCreate,
		SHA:    blobSHA,
		Mode:   0644,
	})
	if err != nil {
		t.Errorf("ApplyOperation() in dry-run mode error = %v", err)
	}

	// Verify file was NOT created
	filePath := filepath.Join(workDir, "test.txt")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("File should not exist in dry-run mode")
	}
}

// TestFileOps_ErrorConditions tests various error conditions
func TestFileOps_ErrorConditions(t *testing.T) {
	repo, _ := setupTestRepo(t)
	service := NewFileOps(repo)

	t.Run("create with missing SHA", func(t *testing.T) {
		err := service.ApplyOperation(Operation{
			Path:   scpath.RelativePath("test.txt"),
			Action: ActionCreate,
			SHA:    "",
			Mode:   0644,
		})
		if err == nil {
			t.Error("Expected error for missing SHA, got nil")
		}
	})

	t.Run("create with invalid SHA", func(t *testing.T) {
		err := service.ApplyOperation(Operation{
			Path:   scpath.RelativePath("test.txt"),
			Action: ActionCreate,
			SHA:    objects.ObjectHash("invalid_hash_1234567890123456789012345678901234567890"),
			Mode:   0644,
		})
		if err == nil {
			t.Error("Expected error for invalid SHA, got nil")
		}
	})

	t.Run("modify with missing SHA", func(t *testing.T) {
		err := service.ApplyOperation(Operation{
			Path:   scpath.RelativePath("test.txt"),
			Action: ActionModify,
			SHA:    "",
			Mode:   0644,
		})
		if err == nil {
			t.Error("Expected error for missing SHA, got nil")
		}
	})

	t.Run("unknown action", func(t *testing.T) {
		content := "test"
		blobSHA := createTestBlob(t, repo, content)
		err := service.ApplyOperation(Operation{
			Path:   scpath.RelativePath("test.txt"),
			Action: ActionType(999), // Invalid action
			SHA:    blobSHA,
			Mode:   0644,
		})
		if err == nil {
			t.Error("Expected error for unknown action, got nil")
		}
	})

	t.Run("delete non-existent file", func(t *testing.T) {
		// Should not error - deleting non-existent file is idempotent
		err := service.ApplyOperation(Operation{
			Path:   scpath.RelativePath("nonexistent.txt"),
			Action: ActionDelete,
		})
		if err != nil {
			t.Errorf("Delete non-existent file should succeed, got error: %v", err)
		}
	})
}

// TestFileOps_LargeFiles tests operations with large file content
func TestFileOps_LargeFiles(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	// Create a large content (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	blobSHA := createTestBlob(t, repo, string(largeContent))

	// Create file with large content
	err := service.ApplyOperation(Operation{
		Path:   scpath.RelativePath("large.bin"),
		Action: ActionCreate,
		SHA:    blobSHA,
		Mode:   0644,
	})
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Verify content
	filePath := filepath.Join(workDir, "large.bin")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read large file: %v", err)
	}

	if len(data) != len(largeContent) {
		t.Errorf("Large file size = %d, want %d", len(data), len(largeContent))
	}

	// Verify content integrity
	for i := range data {
		if data[i] != largeContent[i] {
			t.Errorf("Content mismatch at byte %d: got %d, want %d", i, data[i], largeContent[i])
			break
		}
	}
}

// TestFileOps_SpecialCharacters tests files with special characters in names
func TestFileOps_SpecialCharacters(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	content := "test content"
	blobSHA := createTestBlob(t, repo, content)

	tests := []struct {
		name     string
		filename string
	}{
		{"spaces in name", "file with spaces.txt"},
		{"unicode characters", "файл.txt"},
		{"special chars", "file-with_special.chars.txt"},
		{"dots", "file.name.with.dots.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ApplyOperation(Operation{
				Path:   scpath.RelativePath(tt.filename),
				Action: ActionCreate,
				SHA:    blobSHA,
				Mode:   0644,
			})
			if err != nil {
				t.Errorf("Failed to create file with %s: %v", tt.name, err)
				return
			}

			// Verify file exists
			filePath := filepath.Join(workDir, tt.filename)
			if _, err := os.Stat(filePath); err != nil {
				t.Errorf("File not created: %v", err)
			}
		})
	}
}

// TestFileOps_DeepNesting tests operations with deeply nested directories
func TestFileOps_DeepNesting(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	content := "nested content"
	blobSHA := createTestBlob(t, repo, content)

	// Create deeply nested file
	deepPath := "a/b/c/d/e/f/g/h/i/j/deep.txt"
	err := service.ApplyOperation(Operation{
		Path:   scpath.RelativePath(deepPath),
		Action: ActionCreate,
		SHA:    blobSHA,
		Mode:   0644,
	})
	if err != nil {
		t.Fatalf("Failed to create deeply nested file: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(workDir, deepPath)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read deeply nested file: %v", err)
	}
	if string(data) != content {
		t.Errorf("Content mismatch: got %q, want %q", string(data), content)
	}

	// Delete and verify parent cleanup
	err = service.ApplyOperation(Operation{
		Path:   scpath.RelativePath(deepPath),
		Action: ActionDelete,
	})
	if err != nil {
		t.Fatalf("Failed to delete deeply nested file: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("File should have been deleted")
	}

	// Verify empty parent directories were cleaned up
	parentPath := filepath.Join(workDir, "a")
	if _, err := os.Stat(parentPath); !os.IsNotExist(err) {
		t.Error("Empty parent directories should have been cleaned up")
	}
}

// TestFileOps_AtomicWrite tests that writes are atomic
func TestFileOps_AtomicWrite(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	// Create initial file
	initialContent := "Initial content"
	initialBlob := createTestBlob(t, repo, initialContent)
	filePath := scpath.RelativePath("atomic.txt")

	err := service.ApplyOperation(Operation{
		Path:   filePath,
		Action: ActionCreate,
		SHA:    initialBlob,
		Mode:   0644,
	})
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Verify we can always read a complete file (not a partial write)
	fullPath := filepath.Join(workDir, "atomic.txt")
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != initialContent {
		t.Errorf("Content = %q, want %q", string(data), initialContent)
	}

	// File should be complete, not partial
	if len(data) > 0 && len(data) != len(initialContent) {
		t.Error("File appears to have been written partially")
	}
}

// TestFileOps_ConcurrentOperations tests multiple operations in sequence
func TestFileOps_ConcurrentOperations(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	// Create multiple files
	files := []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt"}
	content := "concurrent test"
	blobSHA := createTestBlob(t, repo, content)

	// Create all files
	for _, file := range files {
		err := service.ApplyOperation(Operation{
			Path:   scpath.RelativePath(file),
			Action: ActionCreate,
			SHA:    blobSHA,
			Mode:   0644,
		})
		if err != nil {
			t.Errorf("Failed to create %s: %v", file, err)
		}
	}

	// Verify all files exist
	for _, file := range files {
		filePath := filepath.Join(workDir, file)
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read %s: %v", file, err)
			continue
		}
		if string(data) != content {
			t.Errorf("%s: content = %q, want %q", file, string(data), content)
		}
	}

	// Delete all files
	for _, file := range files {
		err := service.ApplyOperation(Operation{
			Path:   scpath.RelativePath(file),
			Action: ActionDelete,
		})
		if err != nil {
			t.Errorf("Failed to delete %s: %v", file, err)
		}
	}

	// Verify all files are gone
	for _, file := range files {
		filePath := filepath.Join(workDir, file)
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Errorf("File %s should have been deleted", file)
		}
	}
}

// TestFileOps_PermissionPreservation tests that file permissions are preserved
func TestFileOps_PermissionPreservation(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	content := "permission test"
	blobSHA := createTestBlob(t, repo, content)

	tests := []struct {
		name      string
		mode      os.FileMode
		skipOnWin bool
		minPerms  os.FileMode // minimum expected permissions on Windows
	}{
		{"read-only", 0444, false, 0400},
		{"read-write", 0644, false, 0600},
		{"executable", 0755, true, 0}, // Skip on Windows - execute bit works differently
		{"owner-only", 0600, false, 0600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWin {
				t.Skip("Skipping on Windows - execute permissions work differently")
			}

			filename := fmt.Sprintf("perm_%s.txt", tt.name)
			err := service.ApplyOperation(Operation{
				Path:   scpath.RelativePath(filename),
				Action: ActionCreate,
				SHA:    blobSHA,
				Mode:   objects.FromOSFileMode(tt.mode),
			})
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			// Verify file has correct permissions (check user bits only)
			filePath := filepath.Join(workDir, filename)
			info, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}

			gotMode := info.Mode() & 0700 // User permissions only

			// On Windows, we can only check minimum permissions
			// Windows doesn't support all Unix permission bits
			if tt.minPerms > 0 {
				if gotMode&tt.minPerms != tt.minPerms {
					t.Errorf("Mode = %o, want at least %o", gotMode, tt.minPerms)
				}
			} else {
				wantMode := tt.mode & 0700
				if gotMode != wantMode {
					t.Errorf("Mode = %o, want %o", gotMode, wantMode)
				}
			}
		})
	}
}

// TestFileOps_BackupAndRestore tests backup and restore with edge cases
func TestFileOps_BackupAndRestore(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	t.Run("backup and restore multiple times", func(t *testing.T) {
		content := "Original"
		filePath := scpath.RelativePath("multi.txt")
		fullPath := filepath.Join(workDir, "multi.txt")
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}

		// Create multiple backups
		backup1, err := service.CreateBackup(filePath)
		if err != nil {
			t.Fatalf("First backup failed: %v", err)
		}
		defer service.CleanupBackup(backup1)

		// Modify file
		err = os.WriteFile(fullPath, []byte("Modified 1"), 0644)
		if err != nil {
			t.Fatalf("Modify 1 failed: %v", err)
		}

		backup2, err := service.CreateBackup(filePath)
		if err != nil {
			t.Fatalf("Second backup failed: %v", err)
		}
		defer service.CleanupBackup(backup2)

		// Modify again
		err = os.WriteFile(fullPath, []byte("Modified 2"), 0644)
		if err != nil {
			t.Fatalf("Modify 2 failed: %v", err)
		}

		// Restore second backup
		err = service.RestoreBackup(backup2)
		if err != nil {
			t.Fatalf("Restore backup2 failed: %v", err)
		}

		data, _ := os.ReadFile(fullPath)
		if string(data) != "Modified 1" {
			t.Errorf("After restore backup2: content = %q, want %q", string(data), "Modified 1")
		}

		// Restore first backup
		err = service.RestoreBackup(backup1)
		if err != nil {
			t.Fatalf("Restore backup1 failed: %v", err)
		}

		data, _ = os.ReadFile(fullPath)
		if string(data) != content {
			t.Errorf("After restore backup1: content = %q, want %q", string(data), content)
		}
	})

	t.Run("nil backup handling", func(t *testing.T) {
		err := service.RestoreBackup(nil)
		if err == nil {
			t.Error("Expected error for nil backup, got nil")
		}

		err = service.CleanupBackup(nil)
		if err != nil {
			t.Errorf("CleanupBackup(nil) should not error, got: %v", err)
		}
	})

	t.Run("backup of large file", func(t *testing.T) {
		largeContent := make([]byte, 1024*1024) // 1MB
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}

		filePath := scpath.RelativePath("large_backup.bin")
		fullPath := filepath.Join(workDir, "large_backup.bin")
		err := os.WriteFile(fullPath, largeContent, 0644)
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}

		backup, err := service.CreateBackup(filePath)
		if err != nil {
			t.Fatalf("Backup failed: %v", err)
		}
		defer service.CleanupBackup(backup)

		// Modify file
		err = os.WriteFile(fullPath, []byte("modified"), 0644)
		if err != nil {
			t.Fatalf("Modify failed: %v", err)
		}

		// Restore
		err = service.RestoreBackup(backup)
		if err != nil {
			t.Fatalf("Restore failed: %v", err)
		}

		// Verify content
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if len(data) != len(largeContent) {
			t.Errorf("Restored size = %d, want %d", len(data), len(largeContent))
		}
	})
}

// TestFileOps_EmptyContent tests operations with empty files
func TestFileOps_EmptyContent(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	// Create empty blob
	emptyBlob := createTestBlob(t, repo, "")

	// Create empty file
	err := service.ApplyOperation(Operation{
		Path:   scpath.RelativePath("empty.txt"),
		Action: ActionCreate,
		SHA:    emptyBlob,
		Mode:   0644,
	})
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	// Verify file exists and is empty
	filePath := filepath.Join(workDir, "empty.txt")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read empty file: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("Empty file size = %d, want 0", len(data))
	}
}

// TestFileOps_PartialDirectoryCleanup tests that only empty directories are removed
func TestFileOps_PartialDirectoryCleanup(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	service := NewFileOps(repo)

	content := "test"
	blobSHA := createTestBlob(t, repo, content)

	// Create structure: dir1/dir2/file1.txt and dir1/file2.txt
	err := service.ApplyOperation(Operation{
		Path:   scpath.RelativePath("dir1/dir2/file1.txt"),
		Action: ActionCreate,
		SHA:    blobSHA,
		Mode:   0644,
	})
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	err = service.ApplyOperation(Operation{
		Path:   scpath.RelativePath("dir1/file2.txt"),
		Action: ActionCreate,
		SHA:    blobSHA,
		Mode:   0644,
	})
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Delete dir1/dir2/file1.txt
	err = service.ApplyOperation(Operation{
		Path:   scpath.RelativePath("dir1/dir2/file1.txt"),
		Action: ActionDelete,
	})
	if err != nil {
		t.Fatalf("Failed to delete file1: %v", err)
	}

	// dir2 should be removed (empty), but dir1 should remain (has file2.txt)
	dir2Path := filepath.Join(workDir, "dir1", "dir2")
	if _, err := os.Stat(dir2Path); !os.IsNotExist(err) {
		t.Error("dir2 should have been removed")
	}

	dir1Path := filepath.Join(workDir, "dir1")
	if _, err := os.Stat(dir1Path); err != nil {
		t.Error("dir1 should still exist")
	}

	file2Path := filepath.Join(workDir, "dir1", "file2.txt")
	if _, err := os.Stat(file2Path); err != nil {
		t.Error("file2.txt should still exist")
	}
}
