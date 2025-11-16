package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// =====================================================
// Test Setup Helpers
// =====================================================

// createTestIndexUpdater creates an IndexUpdater with a temporary directory
func createTestIndexUpdater(t *testing.T) (*IndexUpdater, string, scpath.AbsolutePath) {
	t.Helper()

	// Create temporary working directory
	workDir := t.TempDir()

	// Create temporary index file path
	indexPath := scpath.AbsolutePath(filepath.Join(workDir, ".sc", "index"))

	// Ensure .sc directory exists
	if err := os.MkdirAll(filepath.Join(workDir, ".sc"), 0755); err != nil {
		t.Fatalf("failed to create .sc directory: %v", err)
	}

	updater := NewUpdater(workDir, indexPath)
	return updater, workDir, indexPath
}

// createTestFile creates a file in the working directory for testing
func createTestFile(t *testing.T, workDir, relPath, content string) scpath.RelativePath {
	t.Helper()

	fullPath := filepath.Join(workDir, relPath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file %s: %v", relPath, err)
	}

	return scpath.RelativePath(relPath)
}

// createFileInfo creates a FileInfo for testing
func createFileInfo(hash string) FileInfo {
	// Convert to valid hex characters (replace non-hex with valid hex)
	validHex := ""
	for _, c := range hash {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			validHex += string(c)
		} else {
			// Replace invalid characters with corresponding hex digit
			validHex += string('a' + (c % 6))
		}
	}

	// Pad to 40 characters (SHA-1 length) with zeros
	fullHash := validHex
	for len(fullHash) < 40 {
		fullHash += "0"
	}
	if len(fullHash) > 40 {
		fullHash = fullHash[:40]
	}
	return FileInfo{
		SHA:  objects.ObjectHash(fullHash),
		Mode: objects.FileModeRegular,
	}
}

// =====================================================
// TestNewUpdater
// =====================================================

func TestNewUpdater(t *testing.T) {
	workDir := "/test/work/dir"
	indexPath := scpath.AbsolutePath("/test/.sc/index")

	updater := NewUpdater(workDir, indexPath)

	if updater == nil {
		t.Fatal("NewUpdater returned nil")
	}

	if updater.workDir != workDir {
		t.Errorf("expected workDir %s, got %s", workDir, updater.workDir)
	}

	if updater.indexPath != indexPath {
		t.Errorf("expected indexPath %s, got %s", indexPath, updater.indexPath)
	}
}

// =====================================================
// TestUpdateToMatch
// =====================================================

func TestUpdateToMatch_EmptyIndex(t *testing.T) {
	updater, workDir, indexPath := createTestIndexUpdater(t)

	// Create test files
	file1Path := createTestFile(t, workDir, "file1.txt", "content1")
	file2Path := createTestFile(t, workDir, "dir/file2.txt", "content2")

	targetFiles := map[scpath.RelativePath]FileInfo{
		file1Path: createFileInfo("abc123"),
		file2Path: createFileInfo("def456"),
	}

	// Execute
	result, err := updater.UpdateToMatch(targetFiles)

	// Verify
	if err != nil {
		t.Fatalf("UpdateToMatch failed: %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.EntriesUpdated != 2 {
		t.Errorf("expected 2 entries updated, got %d", result.EntriesUpdated)
	}

	if result.EntriesRemoved != 0 {
		t.Errorf("expected 0 entries removed, got %d", result.EntriesRemoved)
	}

	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(result.Errors))
	}

	// Verify index was written
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}

	if !idx.Has(file1Path) {
		t.Error("expected index to contain file1.txt")
	}

	if !idx.Has(file2Path) {
		t.Error("expected index to contain dir/file2.txt")
	}
}

func TestUpdateToMatch_ReplacesExistingIndex(t *testing.T) {
	updater, workDir, indexPath := createTestIndexUpdater(t)

	// Create initial files
	file1Path := createTestFile(t, workDir, "file1.txt", "content1")
	file2Path := createTestFile(t, workDir, "file2.txt", "content2")

	// Create initial index
	initialFiles := map[scpath.RelativePath]FileInfo{
		file1Path: createFileInfo("abc123"),
		file2Path: createFileInfo("def456"),
	}

	_, err := updater.UpdateToMatch(initialFiles)
	if err != nil {
		t.Fatalf("initial UpdateToMatch failed: %v", err)
	}

	// Now replace with different files
	file3Path := createTestFile(t, workDir, "file3.txt", "content3")

	newFiles := map[scpath.RelativePath]FileInfo{
		file3Path: createFileInfo("ghi789"),
	}

	result, err := updater.UpdateToMatch(newFiles)

	// Verify
	if err != nil {
		t.Fatalf("UpdateToMatch failed: %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.EntriesUpdated != 1 {
		t.Errorf("expected 1 entry updated, got %d", result.EntriesUpdated)
	}

	// Verify index only contains file3
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}

	if idx.Has(file1Path) {
		t.Error("expected index to not contain file1.txt")
	}

	if idx.Has(file2Path) {
		t.Error("expected index to not contain file2.txt")
	}

	if !idx.Has(file3Path) {
		t.Error("expected index to contain file3.txt")
	}
}

func TestUpdateToMatch_NonExistentFile(t *testing.T) {
	updater, _, _ := createTestIndexUpdater(t)

	nonExistentPath := scpath.RelativePath("nonexistent.txt")
	targetFiles := map[scpath.RelativePath]FileInfo{
		nonExistentPath: createFileInfo("abc123"),
	}

	result, err := updater.UpdateToMatch(targetFiles)

	// Should not return error but should record failure
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if result.Success {
		t.Error("expected Success to be false")
	}

	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}

	if result.EntriesUpdated != 0 {
		t.Errorf("expected 0 entries updated, got %d", result.EntriesUpdated)
	}
}

func TestUpdateToMatch_PartialSuccess(t *testing.T) {
	updater, workDir, indexPath := createTestIndexUpdater(t)

	// Create one valid file
	validPath := createTestFile(t, workDir, "valid.txt", "content")
	invalidPath := scpath.RelativePath("invalid.txt") // doesn't exist

	targetFiles := map[scpath.RelativePath]FileInfo{
		validPath:   createFileInfo("abc123"),
		invalidPath: createFileInfo("def456"),
	}

	result, err := updater.UpdateToMatch(targetFiles)

	// Should not return error but should record failure in result
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if result.Success {
		t.Error("expected Success to be false")
	}

	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}

	// Valid file should still be processed
	if result.EntriesUpdated != 1 {
		t.Errorf("expected 1 entry updated, got %d", result.EntriesUpdated)
	}

	// Index should not be written on failure
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}

	if idx.Has(validPath) {
		t.Error("expected index to not be updated on partial failure")
	}
}

// =====================================================
// TestUpdateIncremental
// =====================================================

func TestUpdateIncremental_AddNewFiles(t *testing.T) {
	updater, workDir, indexPath := createTestIndexUpdater(t)

	// Create initial index
	file1Path := createTestFile(t, workDir, "file1.txt", "content1")
	initialFiles := map[scpath.RelativePath]FileInfo{
		file1Path: createFileInfo("abc123"),
	}

	_, err := updater.UpdateToMatch(initialFiles)
	if err != nil {
		t.Fatalf("initial UpdateToMatch failed: %v", err)
	}

	// Add new files incrementally
	file2Path := createTestFile(t, workDir, "file2.txt", "content2")
	file3Path := createTestFile(t, workDir, "file3.txt", "content3")

	toAdd := map[scpath.RelativePath]FileInfo{
		file2Path: createFileInfo("def456"),
		file3Path: createFileInfo("ghi789"),
	}

	result, err := updater.UpdateIncremental(toAdd, nil)

	// Verify
	if err != nil {
		t.Fatalf("UpdateIncremental failed: %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.EntriesUpdated != 2 {
		t.Errorf("expected 2 entries updated, got %d", result.EntriesUpdated)
	}

	if result.EntriesRemoved != 0 {
		t.Errorf("expected 0 entries removed, got %d", result.EntriesRemoved)
	}

	// Verify index contains all files
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}

	if !idx.Has(file1Path) {
		t.Error("expected index to contain file1.txt")
	}

	if !idx.Has(file2Path) {
		t.Error("expected index to contain file2.txt")
	}

	if !idx.Has(file3Path) {
		t.Error("expected index to contain file3.txt")
	}
}

func TestUpdateIncremental_RemoveFiles(t *testing.T) {
	updater, workDir, indexPath := createTestIndexUpdater(t)

	// Create initial index with multiple files
	file1Path := createTestFile(t, workDir, "file1.txt", "content1")
	file2Path := createTestFile(t, workDir, "file2.txt", "content2")
	file3Path := createTestFile(t, workDir, "file3.txt", "content3")

	initialFiles := map[scpath.RelativePath]FileInfo{
		file1Path: createFileInfo("abc123"),
		file2Path: createFileInfo("def456"),
		file3Path: createFileInfo("ghi789"),
	}

	_, err := updater.UpdateToMatch(initialFiles)
	if err != nil {
		t.Fatalf("initial UpdateToMatch failed: %v", err)
	}

	// Remove files incrementally
	toRemove := []scpath.RelativePath{file2Path, file3Path}

	result, err := updater.UpdateIncremental(nil, toRemove)

	// Verify
	if err != nil {
		t.Fatalf("UpdateIncremental failed: %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.EntriesRemoved != 2 {
		t.Errorf("expected 2 entries removed, got %d", result.EntriesRemoved)
	}

	if result.EntriesUpdated != 0 {
		t.Errorf("expected 0 entries updated, got %d", result.EntriesUpdated)
	}

	// Verify index only contains file1
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}

	if !idx.Has(file1Path) {
		t.Error("expected index to contain file1.txt")
	}

	if idx.Has(file2Path) {
		t.Error("expected index to not contain file2.txt")
	}

	if idx.Has(file3Path) {
		t.Error("expected index to not contain file3.txt")
	}
}

func TestUpdateIncremental_AddAndRemove(t *testing.T) {
	updater, workDir, indexPath := createTestIndexUpdater(t)

	// Create initial index
	file1Path := createTestFile(t, workDir, "file1.txt", "content1")
	file2Path := createTestFile(t, workDir, "file2.txt", "content2")

	initialFiles := map[scpath.RelativePath]FileInfo{
		file1Path: createFileInfo("abc123"),
		file2Path: createFileInfo("def456"),
	}

	_, err := updater.UpdateToMatch(initialFiles)
	if err != nil {
		t.Fatalf("initial UpdateToMatch failed: %v", err)
	}

	// Add and remove simultaneously
	file3Path := createTestFile(t, workDir, "file3.txt", "content3")

	toAdd := map[scpath.RelativePath]FileInfo{
		file3Path: createFileInfo("ghi789"),
	}
	toRemove := []scpath.RelativePath{file2Path}

	result, err := updater.UpdateIncremental(toAdd, toRemove)

	// Verify
	if err != nil {
		t.Fatalf("UpdateIncremental failed: %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.EntriesUpdated != 1 {
		t.Errorf("expected 1 entry updated, got %d", result.EntriesUpdated)
	}

	if result.EntriesRemoved != 1 {
		t.Errorf("expected 1 entry removed, got %d", result.EntriesRemoved)
	}

	// Verify final index state
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}

	if !idx.Has(file1Path) {
		t.Error("expected index to contain file1.txt")
	}

	if idx.Has(file2Path) {
		t.Error("expected index to not contain file2.txt")
	}

	if !idx.Has(file3Path) {
		t.Error("expected index to contain file3.txt")
	}
}

func TestUpdateIncremental_UpdateExistingFile(t *testing.T) {
	updater, workDir, indexPath := createTestIndexUpdater(t)

	// Create initial index
	filePath := createTestFile(t, workDir, "file.txt", "original content")

	initialFiles := map[scpath.RelativePath]FileInfo{
		filePath: createFileInfo("abc123"),
	}

	_, err := updater.UpdateToMatch(initialFiles)
	if err != nil {
		t.Fatalf("initial UpdateToMatch failed: %v", err)
	}

	// Read initial entry
	idx1, _ := index.Read(indexPath)
	entry1, _ := idx1.Get(filePath)
	if entry1 == nil {
		t.Fatal("expected to find initial entry")
	}

	// Update the file content
	if err := os.WriteFile(filepath.Join(workDir, filePath.String()), []byte("new content"), 0644); err != nil {
		t.Fatalf("failed to update file: %v", err)
	}

	// Update incrementally with new hash
	newFileInfo := createFileInfo("newHash123")
	toAdd := map[scpath.RelativePath]FileInfo{
		filePath: newFileInfo,
	}

	result, err := updater.UpdateIncremental(toAdd, nil)

	// Verify
	if err != nil {
		t.Fatalf("UpdateIncremental failed: %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.EntriesUpdated != 1 {
		t.Errorf("expected 1 entry updated, got %d", result.EntriesUpdated)
	}

	// Verify entry was updated
	idx2, _ := index.Read(indexPath)
	entry2, _ := idx2.Get(filePath)
	if entry2 == nil {
		t.Fatal("expected to find updated entry")
	}

	if entry2.BlobHash == entry1.BlobHash {
		t.Error("expected BlobHash to be updated")
	}

	if entry2.BlobHash != newFileInfo.SHA {
		t.Errorf("expected BlobHash '%s', got '%s'", newFileInfo.SHA, entry2.BlobHash)
	}
}

func TestUpdateIncremental_RemoveNonExistentFile(t *testing.T) {
	updater, workDir, indexPath := createTestIndexUpdater(t)

	// Create initial index
	filePath := createTestFile(t, workDir, "file.txt", "content")
	initialFiles := map[scpath.RelativePath]FileInfo{
		filePath: createFileInfo("abc123"),
	}

	_, err := updater.UpdateToMatch(initialFiles)
	if err != nil {
		t.Fatalf("initial UpdateToMatch failed: %v", err)
	}

	// Try to remove a file that doesn't exist in index
	nonExistent := scpath.RelativePath("nonexistent.txt")
	toRemove := []scpath.RelativePath{nonExistent}

	result, err := updater.UpdateIncremental(nil, toRemove)

	// Verify
	if err != nil {
		t.Fatalf("UpdateIncremental failed: %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.EntriesRemoved != 0 {
		t.Errorf("expected 0 entries removed, got %d", result.EntriesRemoved)
	}

	// Verify original file still exists
	idx, _ := index.Read(indexPath)
	if !idx.Has(filePath) {
		t.Error("expected original file to remain in index")
	}
}

func TestUpdateIncremental_EmptyIndex(t *testing.T) {
	updater, workDir, _ := createTestIndexUpdater(t)

	// Add files to empty index
	file1Path := createTestFile(t, workDir, "file1.txt", "content1")
	file2Path := createTestFile(t, workDir, "file2.txt", "content2")

	toAdd := map[scpath.RelativePath]FileInfo{
		file1Path: createFileInfo("abc123"),
		file2Path: createFileInfo("def456"),
	}

	result, err := updater.UpdateIncremental(toAdd, nil)

	// Verify
	if err != nil {
		t.Fatalf("UpdateIncremental failed: %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.EntriesUpdated != 2 {
		t.Errorf("expected 2 entries updated, got %d", result.EntriesUpdated)
	}
}

func TestUpdateIncremental_PartialFailure(t *testing.T) {
	updater, workDir, indexPath := createTestIndexUpdater(t)

	// Create initial index
	file1Path := createTestFile(t, workDir, "file1.txt", "content1")
	initialFiles := map[scpath.RelativePath]FileInfo{
		file1Path: createFileInfo("abc123"),
	}

	_, err := updater.UpdateToMatch(initialFiles)
	if err != nil {
		t.Fatalf("initial UpdateToMatch failed: %v", err)
	}

	// Try to add valid and invalid files
	file2Path := createTestFile(t, workDir, "file2.txt", "content2")
	invalidPath := scpath.RelativePath("invalid.txt") // doesn't exist

	toAdd := map[scpath.RelativePath]FileInfo{
		file2Path:   createFileInfo("def456"),
		invalidPath: createFileInfo("ghi789"),
	}

	result, err := updater.UpdateIncremental(toAdd, nil)

	// Should not return error but should record failure in result
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if result.Success {
		t.Error("expected Success to be false")
	}

	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}

	// Valid file should be processed, but index should not be written
	if result.EntriesUpdated != 1 {
		t.Errorf("expected 1 entry updated, got %d", result.EntriesUpdated)
	}

	// Index should not be written on failure
	idx, _ := index.Read(indexPath)
	if idx.Has(file2Path) {
		t.Error("expected index to not be updated on partial failure")
	}
}

// =====================================================
// TestCreateIndexEntry
// =====================================================

func TestCreateIndexEntry_Success(t *testing.T) {
	updater, workDir, _ := createTestIndexUpdater(t)

	// Create test file
	filePath := createTestFile(t, workDir, "test.txt", "test content")
	fileInfo := createFileInfo("abc123")

	entry, err := updater.createIndexEntry(filePath, fileInfo)

	// Verify
	if err != nil {
		t.Fatalf("createIndexEntry failed: %v", err)
	}

	if entry == nil {
		t.Fatal("expected non-nil entry")
	}

	if entry.Path != filePath {
		t.Errorf("expected path %s, got %s", filePath, entry.Path)
	}

	if entry.BlobHash != fileInfo.SHA {
		t.Errorf("expected BlobHash %s, got %s", fileInfo.SHA, entry.BlobHash)
	}
}

func TestCreateIndexEntry_FileNotFound(t *testing.T) {
	updater, _, _ := createTestIndexUpdater(t)

	filePath := scpath.RelativePath("nonexistent.txt")
	fileInfo := createFileInfo("abc123")

	entry, err := updater.createIndexEntry(filePath, fileInfo)

	// Verify
	if err == nil {
		t.Error("expected error for non-existent file")
	}

	if entry != nil {
		t.Error("expected nil entry on error")
	}
}

// =====================================================
// Benchmark Tests
// =====================================================

func BenchmarkUpdateToMatch_100Files(b *testing.B) {
	updater, workDir, _ := createTestIndexUpdater(&testing.T{})

	// Create 100 test files
	targetFiles := make(map[scpath.RelativePath]FileInfo)
	for i := 0; i < 100; i++ {
		path := createTestFile(&testing.T{}, workDir, filepath.Join("dir", "file"+string(rune(i))+".txt"), "content")
		targetFiles[path] = createFileInfo("hash" + string(rune(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = updater.UpdateToMatch(targetFiles)
	}
}

func BenchmarkUpdateIncremental_AddFiles(b *testing.B) {
	updater, workDir, _ := createTestIndexUpdater(&testing.T{})

	// Create initial index
	file1Path := createTestFile(&testing.T{}, workDir, "file1.txt", "content1")
	initialFiles := map[scpath.RelativePath]FileInfo{
		file1Path: createFileInfo("abc123"),
	}
	_, _ = updater.UpdateToMatch(initialFiles)

	// Prepare files to add
	file2Path := createTestFile(&testing.T{}, workDir, "file2.txt", "content2")
	toAdd := map[scpath.RelativePath]FileInfo{
		file2Path: createFileInfo("def456"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = updater.UpdateIncremental(toAdd, nil)
	}
}
