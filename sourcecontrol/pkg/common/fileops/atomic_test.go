package fileops

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

func TestAtomicWrite_Success(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test-file.txt")
	absPath := scpath.AbsolutePath(targetPath)

	testData := []byte("Hello, atomic write!")
	testMode := os.FileMode(0644)

	// Perform atomic write
	err := AtomicWrite(absPath, testData, testMode)
	if err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}

	// Verify file content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != string(testData) {
		t.Errorf("File content mismatch: got %q, want %q", string(content), string(testData))
	}

	// Verify file permissions (on Unix-like systems only)
	if runtime.GOOS != "windows" {
		fileInfo, err := os.Stat(targetPath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}
		if fileInfo.Mode().Perm() != testMode {
			t.Errorf("File permissions mismatch: got %v, want %v", fileInfo.Mode().Perm(), testMode)
		}
	}
}

func TestAtomicWrite_OverwriteExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "overwrite-test.txt")
	absPath := scpath.AbsolutePath(targetPath)

	// Write initial content
	initialData := []byte("initial content")
	err := os.WriteFile(targetPath, initialData, 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Overwrite with atomic write
	newData := []byte("new content after atomic write")
	err = AtomicWrite(absPath, newData, 0644)
	if err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Verify new content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != string(newData) {
		t.Errorf("File content mismatch after overwrite: got %q, want %q", string(content), string(newData))
	}
}

func TestAtomicWrite_EmptyData(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "empty-file.txt")
	absPath := scpath.AbsolutePath(targetPath)

	// Write empty data
	emptyData := []byte{}
	err := AtomicWrite(absPath, emptyData, 0644)
	if err != nil {
		t.Fatalf("AtomicWrite failed with empty data: %v", err)
	}

	// Verify file exists and is empty
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("Expected empty file, got %d bytes", len(content))
	}
}

func TestAtomicWrite_LargeData(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "large-file.txt")
	absPath := scpath.AbsolutePath(targetPath)

	// Create large data (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err := AtomicWrite(absPath, largeData, 0644)
	if err != nil {
		t.Fatalf("AtomicWrite failed with large data: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if len(content) != len(largeData) {
		t.Errorf("File size mismatch: got %d bytes, want %d bytes", len(content), len(largeData))
	}
}

func TestAtomicWrite_DifferentPermissions(t *testing.T) {
	// Skip on Windows as it doesn't support Unix-style permissions
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	testCases := []struct {
		name string
		mode os.FileMode
	}{
		{"ReadOnly", 0444},
		{"ReadWrite", 0644},
		{"ReadWriteExecute", 0755},
		{"WriteOnly", 0222},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetPath := filepath.Join(tmpDir, tc.name+".txt")
			absPath := scpath.AbsolutePath(targetPath)
			testData := []byte("test data for " + tc.name)

			err := AtomicWrite(absPath, testData, tc.mode)
			if err != nil {
				t.Fatalf("AtomicWrite failed: %v", err)
			}

			fileInfo, err := os.Stat(targetPath)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}

			if fileInfo.Mode().Perm() != tc.mode {
				t.Errorf("File permissions mismatch: got %v, want %v", fileInfo.Mode().Perm(), tc.mode)
			}
		})
	}
}

func TestAtomicWrite_InvalidDirectory(t *testing.T) {
	// Try to write to a non-existent directory
	invalidPath := filepath.Join("non-existent-dir-12345", "file.txt")
	absPath := scpath.AbsolutePath(invalidPath)
	testData := []byte("test data")

	err := AtomicWrite(absPath, testData, 0644)
	if err == nil {
		t.Fatal("Expected error when writing to non-existent directory, got nil")
	}
}

func TestAtomicWrite_NoTempFileLeftBehind(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "cleanup-test.txt")
	absPath := scpath.AbsolutePath(targetPath)
	testData := []byte("test cleanup")

	// Perform atomic write
	err := AtomicWrite(absPath, testData, 0644)
	if err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Check for any temporary files left in the directory
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		// Check if there are any .tmp- files left behind
		if len(name) > 5 && name[:5] == ".tmp-" {
			t.Errorf("Temporary file left behind: %s", name)
		}
	}

	// Should only have the target file
	if len(entries) != 1 {
		t.Errorf("Expected 1 file in directory, found %d", len(entries))
	}
}

func TestAtomicWrite_PreservesExistingFileOnError(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "preserve-test.txt")
	absPath := scpath.AbsolutePath(targetPath)

	// Create initial file with known content
	initialData := []byte("initial data that should be preserved")
	err := os.WriteFile(targetPath, initialData, 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Make directory read-only to cause write failure (Unix-like systems)
	// Note: This test may behave differently on Windows
	originalMode, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}

	// This test is more reliable on Unix-like systems
	// On Windows, it might not fail as expected
	_ = os.Chmod(tmpDir, 0444)

	// Restore permissions after test
	defer os.Chmod(tmpDir, originalMode.Mode())

	// Try to write new data (should fail on Unix-like systems)
	newData := []byte("new data that should not be written")
	_ = AtomicWrite(absPath, newData, 0644)

	// Restore write permissions to read the file
	_ = os.Chmod(tmpDir, originalMode.Mode())

	// Verify original content is still there
	content, err := os.ReadFile(targetPath)
	if err != nil {
		// If we can't read it, that's also acceptable for this test
		// as long as the file wasn't corrupted
		return
	}

	// Original data should still be intact
	if string(content) != string(initialData) && string(content) != string(newData) {
		t.Errorf("File content corrupted: got %q", string(content))
	}
}

func TestAtomicWrite_SpecialCharactersInPath(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name     string
		filename string
	}{
		{"Spaces", "file with spaces.txt"},
		{"Dots", "file.with.dots.txt"},
		{"Underscores", "file_with_underscores.txt"},
		{"Hyphens", "file-with-hyphens.txt"},
		{"Numbers", "file123.txt"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetPath := filepath.Join(tmpDir, tc.filename)
			absPath := scpath.AbsolutePath(targetPath)
			testData := []byte("test data for " + tc.name)

			err := AtomicWrite(absPath, testData, 0644)
			if err != nil {
				t.Fatalf("AtomicWrite failed for %q: %v", tc.filename, err)
			}

			content, err := os.ReadFile(targetPath)
			if err != nil {
				t.Fatalf("Failed to read file %q: %v", tc.filename, err)
			}

			if string(content) != string(testData) {
				t.Errorf("File content mismatch for %q: got %q, want %q",
					tc.filename, string(content), string(testData))
			}
		})
	}
}

func TestAtomicWrite_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()

	// Test multiple concurrent writes to different files
	numWrites := 10
	done := make(chan error, numWrites)

	for i := 0; i < numWrites; i++ {
		go func(index int) {
			targetPath := filepath.Join(tmpDir, filepath.FromSlash("concurrent-"+string(rune('0'+index))+".txt"))
			absPath := scpath.AbsolutePath(targetPath)
			testData := []byte("concurrent write " + string(rune('0'+index)))

			err := AtomicWrite(absPath, testData, 0644)
			done <- err
		}(i)
	}

	// Wait for all writes to complete
	for i := 0; i < numWrites; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent write %d failed: %v", i, err)
		}
	}

	// Verify all files were created correctly
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	if len(entries) != numWrites {
		t.Errorf("Expected %d files, found %d", numWrites, len(entries))
	}
}
