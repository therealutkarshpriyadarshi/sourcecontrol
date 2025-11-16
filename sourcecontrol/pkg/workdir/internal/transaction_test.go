package internal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// =====================================================
// Test helpers
// =====================================================

// =====================================================
// Tests for helper functions
// =====================================================

func TestNewTr(t *testing.T) {
	tests := []struct {
		name         string
		success      bool
		opsApplied   int
		totalOps     int
		err          error
		wantSuccess  bool
		wantApplied  int
		wantTotal    int
		wantHasError bool
	}{
		{
			name:         "successful transaction",
			success:      true,
			opsApplied:   5,
			totalOps:     5,
			err:          nil,
			wantSuccess:  true,
			wantApplied:  5,
			wantTotal:    5,
			wantHasError: false,
		},
		{
			name:         "failed transaction with error",
			success:      false,
			opsApplied:   3,
			totalOps:     5,
			err:          errors.New("operation failed"),
			wantSuccess:  false,
			wantApplied:  3,
			wantTotal:    5,
			wantHasError: true,
		},
		{
			name:         "zero operations",
			success:      true,
			opsApplied:   0,
			totalOps:     0,
			err:          nil,
			wantSuccess:  true,
			wantApplied:  0,
			wantTotal:    0,
			wantHasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newTr(tt.success, tt.opsApplied, tt.totalOps, tt.err)

			if result.Success != tt.wantSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.wantSuccess)
			}
			if result.OperationsApplied != tt.wantApplied {
				t.Errorf("OperationsApplied = %d, want %d", result.OperationsApplied, tt.wantApplied)
			}
			if result.TotalOperations != tt.wantTotal {
				t.Errorf("TotalOperations = %d, want %d", result.TotalOperations, tt.wantTotal)
			}
			if (result.Err != nil) != tt.wantHasError {
				t.Errorf("Err = %v, wantHasError %v", result.Err, tt.wantHasError)
			}
		})
	}
}

func TestSuccess(t *testing.T) {
	result := success(10, 10)

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.OperationsApplied != 10 {
		t.Errorf("OperationsApplied = %d, want 10", result.OperationsApplied)
	}
	if result.TotalOperations != 10 {
		t.Errorf("TotalOperations = %d, want 10", result.TotalOperations)
	}
	if result.Err != nil {
		t.Errorf("Err = %v, want nil", result.Err)
	}
}

func TestFailure(t *testing.T) {
	testErr := errors.New("test error")
	result := failure(5, 10, testErr)

	if result.Success {
		t.Error("Success should be false")
	}
	if result.OperationsApplied != 5 {
		t.Errorf("OperationsApplied = %d, want 5", result.OperationsApplied)
	}
	if result.TotalOperations != 10 {
		t.Errorf("TotalOperations = %d, want 10", result.TotalOperations)
	}
	if result.Err != testErr {
		t.Errorf("Err = %v, want %v", result.Err, testErr)
	}
}

// =====================================================
// Tests for NewManager
// =====================================================

func TestNewManager(t *testing.T) {
	repo, _ := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	sourceDir := repo.SourceDirectory()

	manager := NewManager(fileOps, sourceDir)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
	if manager.fileOps != fileOps {
		t.Error("fileOps not set correctly")
	}
	if manager.sourceDir != sourceDir {
		t.Error("sourceDir not set correctly")
	}
}

// =====================================================
// Tests for Manager.validateOperations
// =====================================================

func TestManager_ValidateOperations(t *testing.T) {
	repo, _ := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	tests := []struct {
		name    string
		ops     []Operation
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid operations",
			ops: []Operation{
				{Path: scpath.RelativePath("file1.txt"), Action: ActionCreate, SHA: "abc123"},
				{Path: scpath.RelativePath("file2.txt"), Action: ActionModify, SHA: "def456"},
				{Path: scpath.RelativePath("file3.txt"), Action: ActionDelete},
			},
			wantErr: false,
		},
		{
			name: "empty path",
			ops: []Operation{
				{Path: scpath.RelativePath(""), Action: ActionCreate, SHA: "abc123"},
			},
			wantErr: true,
			errMsg:  "empty path",
		},
		{
			name: "invalid action",
			ops: []Operation{
				{Path: scpath.RelativePath("file.txt"), Action: ActionType(999), SHA: "abc123"},
			},
			wantErr: true,
			errMsg:  "invalid action",
		},
		{
			name: "create without SHA",
			ops: []Operation{
				{Path: scpath.RelativePath("file.txt"), Action: ActionCreate},
			},
			wantErr: true,
			errMsg:  "missing SHA",
		},
		{
			name: "modify without SHA",
			ops: []Operation{
				{Path: scpath.RelativePath("file.txt"), Action: ActionModify},
			},
			wantErr: true,
			errMsg:  "missing SHA",
		},
		{
			name: "delete without SHA (valid)",
			ops: []Operation{
				{Path: scpath.RelativePath("file.txt"), Action: ActionDelete},
			},
			wantErr: false,
		},
		{
			name: "duplicate paths",
			ops: []Operation{
				{Path: scpath.RelativePath("file.txt"), Action: ActionCreate, SHA: "abc123"},
				{Path: scpath.RelativePath("file.txt"), Action: ActionModify, SHA: "def456"},
			},
			wantErr: true,
			errMsg:  "duplicate operation",
		},
		{
			name: "multiple operations on different paths",
			ops: []Operation{
				{Path: scpath.RelativePath("file1.txt"), Action: ActionCreate, SHA: "abc123"},
				{Path: scpath.RelativePath("file2.txt"), Action: ActionCreate, SHA: "def456"},
				{Path: scpath.RelativePath("file3.txt"), Action: ActionCreate, SHA: "ghi789"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateOperations(tt.ops)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateOperations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// =====================================================
// Tests for Manager.ExecuteAtomically
// =====================================================

func TestManager_ExecuteAtomically_EmptyOperations(t *testing.T) {
	repo, _ := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	result := manager.ExecuteAtomically(context.Background(), []Operation{})

	if !result.Success {
		t.Error("Empty operations should succeed")
	}
	if result.OperationsApplied != 0 {
		t.Errorf("OperationsApplied = %d, want 0", result.OperationsApplied)
	}
	if result.TotalOperations != 0 {
		t.Errorf("TotalOperations = %d, want 0", result.TotalOperations)
	}
	if result.Err != nil {
		t.Errorf("Err = %v, want nil", result.Err)
	}
}

func TestManager_ExecuteAtomically_Success(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Create test files and blobs
	content1 := "File 1 content"
	content2 := "File 2 initial"
	content3 := "File 3 to delete"

	blob1 := createTestBlob(t, repo, content1)
	blob2 := createTestBlob(t, repo, content2)

	// Setup: Create file2 and file3 for modify and delete operations
	file2Path := filepath.Join(workDir, "file2.txt")
	file3Path := filepath.Join(workDir, "file3.txt")
	os.WriteFile(file2Path, []byte("old content"), 0644)
	os.WriteFile(file3Path, []byte(content3), 0644)

	ops := []Operation{
		{Path: scpath.RelativePath("file1.txt"), Action: ActionCreate, SHA: blob1, Mode: 0644},
		{Path: scpath.RelativePath("file2.txt"), Action: ActionModify, SHA: blob2, Mode: 0644},
		{Path: scpath.RelativePath("file3.txt"), Action: ActionDelete},
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if !result.Success {
		t.Errorf("ExecuteAtomically() failed: %v", result.Err)
	}
	if result.OperationsApplied != 3 {
		t.Errorf("OperationsApplied = %d, want 3", result.OperationsApplied)
	}
	if result.TotalOperations != 3 {
		t.Errorf("TotalOperations = %d, want 3", result.TotalOperations)
	}

	// Verify file1 was created
	file1Path := filepath.Join(workDir, "file1.txt")
	data1, err := os.ReadFile(file1Path)
	if err != nil {
		t.Errorf("Failed to read file1.txt: %v", err)
	}
	if string(data1) != content1 {
		t.Errorf("file1.txt content = %q, want %q", string(data1), content1)
	}

	// Verify file2 was modified
	data2, err := os.ReadFile(file2Path)
	if err != nil {
		t.Errorf("Failed to read file2.txt: %v", err)
	}
	if string(data2) != content2 {
		t.Errorf("file2.txt content = %q, want %q", string(data2), content2)
	}

	// Verify file3 was deleted
	if _, err := os.Stat(file3Path); !os.IsNotExist(err) {
		t.Error("file3.txt should be deleted")
	}
}

func TestManager_ExecuteAtomically_ValidationError(t *testing.T) {
	repo, _ := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	ops := []Operation{
		{Path: scpath.RelativePath(""), Action: ActionCreate, SHA: "abc123"}, // Invalid: empty path
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if result.Success {
		t.Error("ExecuteAtomically() should fail for invalid operations")
	}
	if result.OperationsApplied != 0 {
		t.Errorf("OperationsApplied = %d, want 0", result.OperationsApplied)
	}
}

func TestManager_ExecuteAtomically_OperationFailureWithRollback(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Setup: Create file1 for modification
	initialContent := "Initial content"
	file1Path := filepath.Join(workDir, "file1.txt")
	os.WriteFile(file1Path, []byte(initialContent), 0644)

	// Create a valid blob for file1
	newContent := "New content"
	validBlob := createTestBlob(t, repo, newContent)

	// Create operations: modify file1 (will succeed), then create with invalid SHA (will fail)
	ops := []Operation{
		{Path: scpath.RelativePath("file1.txt"), Action: ActionModify, SHA: validBlob, Mode: 0644},
		{Path: scpath.RelativePath("file2.txt"), Action: ActionCreate, SHA: objects.ObjectHash("invalid_sha"), Mode: 0644},
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if result.Success {
		t.Error("ExecuteAtomically() should fail when operation fails")
	}
	if result.TotalOperations != 2 {
		t.Errorf("TotalOperations = %d, want 2", result.TotalOperations)
	}
	if result.Err == nil {
		t.Error("Err should not be nil")
	}

	// Verify file1 was rolled back to original content
	data, err := os.ReadFile(file1Path)
	if err != nil {
		t.Fatalf("Failed to read file after rollback: %v", err)
	}
	if string(data) != initialContent {
		t.Errorf("File content after rollback = %q, want %q", string(data), initialContent)
	}

	// Verify file2 was not created
	file2Path := filepath.Join(workDir, "file2.txt")
	if _, err := os.Stat(file2Path); !os.IsNotExist(err) {
		t.Error("file2.txt should not exist after rollback")
	}
}

func TestManager_ExecuteAtomically_ContextCancellation(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Setup: Create file1 for modification
	initialContent := "Initial content"
	file1Path := filepath.Join(workDir, "file1.txt")
	os.WriteFile(file1Path, []byte(initialContent), 0644)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	content := "New content"
	blob := createTestBlob(t, repo, content)

	ops := []Operation{
		{Path: scpath.RelativePath("file1.txt"), Action: ActionModify, SHA: blob, Mode: 0644},
		{Path: scpath.RelativePath("file2.txt"), Action: ActionCreate, SHA: blob, Mode: 0644},
	}

	result := manager.ExecuteAtomically(ctx, ops)

	if result.Success {
		t.Error("ExecuteAtomically() should fail when context is cancelled")
	}
	if result.Err == nil {
		t.Error("Err should not be nil")
	}

	// Verify file1 was rolled back if it was modified
	data, err := os.ReadFile(file1Path)
	if err != nil {
		t.Fatalf("Failed to read file after context cancellation: %v", err)
	}
	if string(data) != initialContent {
		t.Errorf("File should be rolled back, got content = %q, want %q", string(data), initialContent)
	}
}

func TestManager_ExecuteAtomically_FirstOperationFailure(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Use invalid SHA to force first operation to fail
	ops := []Operation{
		{Path: scpath.RelativePath("file1.txt"), Action: ActionCreate, SHA: objects.ObjectHash("invalid"), Mode: 0644},
		{Path: scpath.RelativePath("file2.txt"), Action: ActionCreate, SHA: objects.ObjectHash("invalid2"), Mode: 0644},
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if result.Success {
		t.Error("ExecuteAtomically() should fail")
	}
	if result.OperationsApplied != 0 {
		t.Errorf("OperationsApplied = %d, want 0", result.OperationsApplied)
	}

	// Verify no files were created
	file1Path := filepath.Join(workDir, "file1.txt")
	file2Path := filepath.Join(workDir, "file2.txt")
	if _, err := os.Stat(file1Path); !os.IsNotExist(err) {
		t.Error("file1.txt should not exist after failure")
	}
	if _, err := os.Stat(file2Path); !os.IsNotExist(err) {
		t.Error("file2.txt should not exist after failure")
	}
}

// =====================================================
// Tests for Manager.DryRun
// =====================================================

func TestManager_DryRun_ValidOperations(t *testing.T) {
	repo, _ := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	ops := []Operation{
		{Path: scpath.RelativePath("file1.txt"), Action: ActionCreate, SHA: "abc123"},
		{Path: scpath.RelativePath("file2.txt"), Action: ActionModify, SHA: "def456"},
		{Path: scpath.RelativePath("file3.txt"), Action: ActionDelete},
		{Path: scpath.RelativePath("file4.txt"), Action: ActionCreate, SHA: "ghi789"},
	}

	result := manager.DryRun(ops)

	if !result.Valid {
		t.Errorf("DryRun should be valid, got errors: %v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", result.Errors)
	}

	// Verify categorization
	if len(result.Analysis.WillCreate) != 2 {
		t.Errorf("WillCreate = %d items, want 2", len(result.Analysis.WillCreate))
	}
	if len(result.Analysis.WillModify) != 1 {
		t.Errorf("WillModify = %d items, want 1", len(result.Analysis.WillModify))
	}
	if len(result.Analysis.WillDelete) != 1 {
		t.Errorf("WillDelete = %d items, want 1", len(result.Analysis.WillDelete))
	}

}

func TestManager_DryRun_InvalidOperations(t *testing.T) {
	repo, _ := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	ops := []Operation{
		{Path: scpath.RelativePath(""), Action: ActionCreate, SHA: "abc123"}, // Invalid: empty path
	}

	result := manager.DryRun(ops)

	if result.Valid {
		t.Error("DryRun should be invalid for empty path")
	}
	if len(result.Errors) == 0 {
		t.Error("Errors should not be empty")
	}

	// Verify error message
	if !strings.Contains(result.Errors[0], "empty path") {
		t.Errorf("Error message should mention empty path: %s", result.Errors[0])
	}
}

func TestManager_DryRun_EmptyOperations(t *testing.T) {
	repo, _ := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	result := manager.DryRun([]Operation{})

	if !result.Valid {
		t.Error("Empty operations should be valid")
	}
	if len(result.Analysis.WillCreate) != 0 {
		t.Error("WillCreate should be empty")
	}
	if len(result.Analysis.WillModify) != 0 {
		t.Error("WillModify should be empty")
	}
	if len(result.Analysis.WillDelete) != 0 {
		t.Error("WillDelete should be empty")
	}
}

func TestManager_DryRun_MultipleValidationErrors(t *testing.T) {
	repo, _ := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	ops := []Operation{
		{Path: scpath.RelativePath("file.txt"), Action: ActionCreate, SHA: "abc123"},
		{Path: scpath.RelativePath("file.txt"), Action: ActionModify, SHA: "def456"}, // Duplicate
	}

	result := manager.DryRun(ops)

	if result.Valid {
		t.Error("DryRun should be invalid for duplicate paths")
	}
}

// =====================================================
// Integration tests with real FileOps
// =====================================================

func TestManager_ExecuteAtomically_Integration_Success(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Create test blobs
	content1 := "File 1 content"
	content2 := "File 2 content"
	blob1 := createTestBlob(t, repo, content1)
	blob2 := createTestBlob(t, repo, content2)

	ops := []Operation{
		{Path: scpath.RelativePath("test1.txt"), Action: ActionCreate, SHA: blob1, Mode: 0644},
		{Path: scpath.RelativePath("test2.txt"), Action: ActionCreate, SHA: blob2, Mode: 0644},
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if !result.Success {
		t.Fatalf("ExecuteAtomically() failed: %v", result.Err)
	}

	// Verify files were created
	file1Path := filepath.Join(workDir, "test1.txt")
	data1, err := os.ReadFile(file1Path)
	if err != nil {
		t.Errorf("Failed to read test1.txt: %v", err)
	}
	if string(data1) != content1 {
		t.Errorf("test1.txt content = %q, want %q", string(data1), content1)
	}

	file2Path := filepath.Join(workDir, "test2.txt")
	data2, err := os.ReadFile(file2Path)
	if err != nil {
		t.Errorf("Failed to read test2.txt: %v", err)
	}
	if string(data2) != content2 {
		t.Errorf("test2.txt content = %q, want %q", string(data2), content2)
	}
}

func TestManager_ExecuteAtomically_Integration_RollbackOnFailure(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Create initial file
	initialContent := "Initial content"
	filePath := filepath.Join(workDir, "existing.txt")
	err := os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Create operations: modify existing file, then try to create with invalid SHA
	newContent := "New content"
	newBlob := createTestBlob(t, repo, newContent)

	ops := []Operation{
		{Path: scpath.RelativePath("existing.txt"), Action: ActionModify, SHA: newBlob, Mode: 0644},
		{Path: scpath.RelativePath("invalid.txt"), Action: ActionCreate, SHA: objects.ObjectHash("invalid_sha_123"), Mode: 0644},
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if result.Success {
		t.Error("ExecuteAtomically() should fail with invalid SHA")
	}

	// Verify the first file was rolled back to original content
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file after rollback: %v", err)
	}
	if string(data) != initialContent {
		t.Errorf("File content after rollback = %q, want %q", string(data), initialContent)
	}

	// Verify the second file was not created
	invalidPath := filepath.Join(workDir, "invalid.txt")
	if _, err := os.Stat(invalidPath); !os.IsNotExist(err) {
		t.Error("invalid.txt should not exist after rollback")
	}
}

func TestManager_ExecuteAtomically_Integration_DeleteAndRollback(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Create initial file
	content := "File to delete"
	filePath := filepath.Join(workDir, "delete_me.txt")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Create operations: delete file, then fail with invalid operation
	ops := []Operation{
		{Path: scpath.RelativePath("delete_me.txt"), Action: ActionDelete},
		{Path: scpath.RelativePath("invalid.txt"), Action: ActionCreate, SHA: objects.ObjectHash("invalid_sha"), Mode: 0644},
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if result.Success {
		t.Error("ExecuteAtomically() should fail")
	}

	// Verify the deleted file was restored
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file after rollback: %v", err)
	}
	if string(data) != content {
		t.Errorf("Restored content = %q, want %q", string(data), content)
	}
}

func TestManager_ExecuteAtomically_Integration_ContextTimeout(t *testing.T) {
	repo, _ := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Create a context with immediate timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give the context time to expire
	time.Sleep(10 * time.Millisecond)

	content := "Test content"
	blob := createTestBlob(t, repo, content)

	ops := []Operation{
		{Path: scpath.RelativePath("test1.txt"), Action: ActionCreate, SHA: blob, Mode: 0644},
		{Path: scpath.RelativePath("test2.txt"), Action: ActionCreate, SHA: blob, Mode: 0644},
	}

	result := manager.ExecuteAtomically(ctx, ops)

	if result.Success {
		t.Error("ExecuteAtomically() should fail with expired context")
	}
	if result.Err != context.DeadlineExceeded && result.Err != context.Canceled {
		t.Errorf("Expected context error, got: %v", result.Err)
	}
}

func TestManager_ExecuteAtomically_Integration_MixedOperations(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Create initial files
	file1Content := "File 1"
	file2Content := "File 2"
	file3Content := "File 3"

	file1Path := filepath.Join(workDir, "file1.txt")
	file2Path := filepath.Join(workDir, "file2.txt")
	file3Path := filepath.Join(workDir, "file3.txt")

	os.WriteFile(file1Path, []byte(file1Content), 0644)
	os.WriteFile(file2Path, []byte(file2Content), 0644)

	// Create blobs for operations
	newContent := "Modified content"
	newBlob := createTestBlob(t, repo, newContent)
	createBlob := createTestBlob(t, repo, file3Content)

	ops := []Operation{
		{Path: scpath.RelativePath("file1.txt"), Action: ActionModify, SHA: newBlob, Mode: 0644},
		{Path: scpath.RelativePath("file2.txt"), Action: ActionDelete},
		{Path: scpath.RelativePath("file3.txt"), Action: ActionCreate, SHA: createBlob, Mode: 0644},
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if !result.Success {
		t.Fatalf("ExecuteAtomically() failed: %v", result.Err)
	}

	// Verify file1 was modified
	data1, _ := os.ReadFile(file1Path)
	if string(data1) != newContent {
		t.Errorf("file1.txt content = %q, want %q", string(data1), newContent)
	}

	// Verify file2 was deleted
	if _, err := os.Stat(file2Path); !os.IsNotExist(err) {
		t.Error("file2.txt should be deleted")
	}

	// Verify file3 was created
	data3, err := os.ReadFile(file3Path)
	if err != nil {
		t.Fatalf("file3.txt should exist: %v", err)
	}
	if string(data3) != file3Content {
		t.Errorf("file3.txt content = %q, want %q", string(data3), file3Content)
	}
}

// =====================================================
// Tests for Lock Acquisition
// =====================================================

func TestManager_ExecuteAtomically_LockAcquisitionFailure(t *testing.T) {
	repo, _ := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Acquire lock manually to simulate another process holding it
	lock, err := AcquireLock(repo.SourceDirectory())
	if err != nil {
		t.Fatalf("Failed to acquire lock for test setup: %v", err)
	}
	defer lock.Release()

	content := "Test"
	blob := createTestBlob(t, repo, content)

	ops := []Operation{
		{Path: scpath.RelativePath("test.txt"), Action: ActionCreate, SHA: blob, Mode: 0644},
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if result.Success {
		t.Error("ExecuteAtomically() should fail when lock cannot be acquired")
	}
	if result.Err == nil {
		t.Error("Err should not be nil")
	}
	if !errors.Is(result.Err, ErrLockAcquisitionFailed) {
		t.Errorf("Expected ErrLockAcquisitionFailed, got: %v", result.Err)
	}
}

// =====================================================
// Edge case tests
// =====================================================

func TestManager_ExecuteAtomically_OnlyCreateOperations(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	content1 := "Content 1"
	content2 := "Content 2"
	blob1 := createTestBlob(t, repo, content1)
	blob2 := createTestBlob(t, repo, content2)

	ops := []Operation{
		{Path: scpath.RelativePath("file1.txt"), Action: ActionCreate, SHA: blob1, Mode: 0644},
		{Path: scpath.RelativePath("file2.txt"), Action: ActionCreate, SHA: blob2, Mode: 0644},
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if !result.Success {
		t.Errorf("ExecuteAtomically() failed: %v", result.Err)
	}

	// Verify files were created
	file1Path := filepath.Join(workDir, "file1.txt")
	file2Path := filepath.Join(workDir, "file2.txt")

	data1, err := os.ReadFile(file1Path)
	if err != nil {
		t.Errorf("Failed to read file1.txt: %v", err)
	}
	if string(data1) != content1 {
		t.Errorf("file1.txt content = %q, want %q", string(data1), content1)
	}

	data2, err := os.ReadFile(file2Path)
	if err != nil {
		t.Errorf("Failed to read file2.txt: %v", err)
	}
	if string(data2) != content2 {
		t.Errorf("file2.txt content = %q, want %q", string(data2), content2)
	}
}

func TestManager_ExecuteAtomically_OnlyDeleteOperations(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Setup: Create files to delete
	file1Path := filepath.Join(workDir, "file1.txt")
	file2Path := filepath.Join(workDir, "file2.txt")
	os.WriteFile(file1Path, []byte("File 1"), 0644)
	os.WriteFile(file2Path, []byte("File 2"), 0644)

	ops := []Operation{
		{Path: scpath.RelativePath("file1.txt"), Action: ActionDelete},
		{Path: scpath.RelativePath("file2.txt"), Action: ActionDelete},
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if !result.Success {
		t.Errorf("ExecuteAtomically() failed: %v", result.Err)
	}

	// Verify files were deleted
	if _, err := os.Stat(file1Path); !os.IsNotExist(err) {
		t.Error("file1.txt should be deleted")
	}
	if _, err := os.Stat(file2Path); !os.IsNotExist(err) {
		t.Error("file2.txt should be deleted")
	}
}

func TestManager_ExecuteAtomically_LargeNumberOfOperations(t *testing.T) {
	repo, workDir := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	// Create 100 operations
	ops := make([]Operation, 100)
	for i := 0; i < 100; i++ {
		content := fmt.Sprintf("File %d content", i)
		blob := createTestBlob(t, repo, content)
		ops[i] = Operation{
			Path:   scpath.RelativePath(fmt.Sprintf("file%d.txt", i)),
			Action: ActionCreate,
			SHA:    blob,
			Mode:   0644,
		}
	}

	result := manager.ExecuteAtomically(context.Background(), ops)

	if !result.Success {
		t.Errorf("ExecuteAtomically() failed: %v", result.Err)
	}
	if result.OperationsApplied != 100 {
		t.Errorf("OperationsApplied = %d, want 100", result.OperationsApplied)
	}

	// Verify a few random files were created
	for _, idx := range []int{0, 50, 99} {
		filePath := filepath.Join(workDir, fmt.Sprintf("file%d.txt", idx))
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("file%d.txt should exist", idx)
		}
	}
}

func TestManager_DryRun_AllActionTypes(t *testing.T) {
	repo, _ := setupTestRepo(t)
	fileOps := NewFileOps(repo)
	manager := NewManager(fileOps, repo.SourceDirectory())

	ops := []Operation{
		{Path: scpath.RelativePath("create1.txt"), Action: ActionCreate, SHA: "abc"},
		{Path: scpath.RelativePath("create2.txt"), Action: ActionCreate, SHA: "def"},
		{Path: scpath.RelativePath("modify1.txt"), Action: ActionModify, SHA: "ghi"},
		{Path: scpath.RelativePath("delete1.txt"), Action: ActionDelete},
	}

	result := manager.DryRun(ops)

	if !result.Valid {
		t.Errorf("DryRun failed: %v", result.Errors)
	}

	// Verify all action types are categorized correctly
	if len(result.Analysis.WillCreate) != 2 {
		t.Errorf("WillCreate count = %d, want 2", len(result.Analysis.WillCreate))
	}
	if len(result.Analysis.WillModify) != 1 {
		t.Errorf("WillModify count = %d, want 1", len(result.Analysis.WillModify))
	}
	if len(result.Analysis.WillDelete) != 1 {
		t.Errorf("WillDelete count = %d, want 1", len(result.Analysis.WillDelete))
	}

	// Verify specific paths
	if result.Analysis.WillCreate[0] != scpath.RelativePath("create1.txt") {
		t.Error("First create operation not captured correctly")
	}
	if result.Analysis.WillModify[0] != scpath.RelativePath("modify1.txt") {
		t.Error("Modify operation not captured correctly")
	}
	if result.Analysis.WillDelete[0] != scpath.RelativePath("delete1.txt") {
		t.Error("Delete operation not captured correctly")
	}
}
