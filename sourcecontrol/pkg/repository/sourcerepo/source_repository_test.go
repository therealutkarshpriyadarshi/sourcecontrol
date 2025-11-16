package sourcerepo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/objects/blob"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// setupTestDirectory creates a temporary test directory
func setupTestDirectory(t *testing.T) (scpath.RepositoryPath, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "sourcecontrol-repo-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repoPath, err := scpath.NewRepositoryPath(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create repository path: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return repoPath, cleanup
}

func TestNewSourceRepository(t *testing.T) {
	repo := NewSourceRepository()

	if repo == nil {
		t.Fatal("NewSourceRepository() returned nil")
	}

	if repo.IsInitialized() {
		t.Error("new repository should not be initialized")
	}

	if repo.ObjectStore() == nil {
		t.Error("object store should not be nil")
	}
}

func TestSourceRepository_Initialize(t *testing.T) {
	repoPath, cleanup := setupTestDirectory(t)
	defer cleanup()

	repo := NewSourceRepository()

	// Initialize the repository
	err := repo.Initialize(repoPath)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Check that repository is marked as initialized
	if !repo.IsInitialized() {
		t.Error("repository should be initialized")
	}

	// Verify directory structure
	sourcePath := repoPath.SourcePath()
	if _, err := os.Stat(sourcePath.String()); os.IsNotExist(err) {
		t.Errorf(".source directory not created at %s", sourcePath)
	}

	objectsPath := sourcePath.ObjectsPath()
	if _, err := os.Stat(objectsPath.String()); os.IsNotExist(err) {
		t.Errorf("objects directory not created at %s", objectsPath)
	}

	refsPath := sourcePath.RefsPath()
	if _, err := os.Stat(refsPath.String()); os.IsNotExist(err) {
		t.Errorf("refs directory not created at %s", refsPath)
	}

	headsPath := refsPath.Join("heads")
	if _, err := os.Stat(headsPath.String()); os.IsNotExist(err) {
		t.Errorf("refs/heads directory not created at %s", headsPath)
	}

	tagsPath := refsPath.Join("tags")
	if _, err := os.Stat(tagsPath.String()); os.IsNotExist(err) {
		t.Errorf("refs/tags directory not created at %s", tagsPath)
	}
}

func TestSourceRepository_InitialFiles(t *testing.T) {
	repoPath, cleanup := setupTestDirectory(t)
	defer cleanup()

	repo := NewSourceRepository()
	err := repo.Initialize(repoPath)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Check HEAD file
	sourcePath := repoPath.SourcePath()
	headPath := sourcePath.HeadPath()
	headContent, err := os.ReadFile(headPath.String())
	if err != nil {
		t.Errorf("HEAD file not created: %v", err)
	}
	expectedHead := "ref: refs/heads/master\n"
	if string(headContent) != expectedHead {
		t.Errorf("HEAD content = %q, want %q", headContent, expectedHead)
	}

	// Check description file
	descPath := sourcePath.Join("description")
	if _, err := os.Stat(descPath.String()); os.IsNotExist(err) {
		t.Error("description file not created")
	}

	// Check config file
	configPath := sourcePath.ConfigPath()
	configContent, err := os.ReadFile(configPath.String())
	if err != nil {
		t.Errorf("config file not created: %v", err)
	}
	if len(configContent) == 0 {
		t.Error("config file is empty")
	}
	// Check for expected config content
	configStr := string(configContent)
	if !contains(configStr, "[core]") {
		t.Error("config file missing [core] section")
	}
	if !contains(configStr, "repositoryformatversion") {
		t.Error("config file missing repositoryformatversion")
	}
}

func TestSourceRepository_InitializeExisting(t *testing.T) {
	repoPath, cleanup := setupTestDirectory(t)
	defer cleanup()

	// Initialize first repository
	repo1 := NewSourceRepository()
	err := repo1.Initialize(repoPath)
	if err != nil {
		t.Fatalf("first Initialize() failed: %v", err)
	}

	// Try to initialize again (should fail)
	repo2 := NewSourceRepository()
	err = repo2.Initialize(repoPath)
	if err == nil {
		t.Error("Initialize() should fail for existing repository")
	}
}

func TestSourceRepository_WorkingDirectory(t *testing.T) {
	repoPath, cleanup := setupTestDirectory(t)
	defer cleanup()

	repo := NewSourceRepository()
	err := repo.Initialize(repoPath)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	workingDir := repo.WorkingDirectory()
	if workingDir.String() != repoPath.String() {
		t.Errorf("WorkingDirectory() = %s, want %s", workingDir, repoPath)
	}
}

func TestSourceRepository_SourceDirectory(t *testing.T) {
	repoPath, cleanup := setupTestDirectory(t)
	defer cleanup()

	repo := NewSourceRepository()
	err := repo.Initialize(repoPath)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	sourceDir := repo.SourceDirectory()
	expectedSourceDir := repoPath.SourcePath()
	if sourceDir.String() != expectedSourceDir.String() {
		t.Errorf("SourceDirectory() = %s, want %s", sourceDir, expectedSourceDir)
	}
}

func TestSourceRepository_WriteAndReadObject(t *testing.T) {
	repoPath, cleanup := setupTestDirectory(t)
	defer cleanup()

	repo := NewSourceRepository()
	err := repo.Initialize(repoPath)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Create a test blob
	testData := []byte("Hello, World! This is a test blob in the repository.")
	testBlob := blob.NewBlob(testData)

	// Write the blob
	hash, err := repo.WriteObject(testBlob)
	if err != nil {
		t.Fatalf("WriteObject() failed: %v", err)
	}

	if hash.IsZero() {
		t.Error("WriteObject() returned zero hash")
	}

	// Read the blob back
	readObj, err := repo.ReadObject(hash)
	if err != nil {
		t.Fatalf("ReadObject() failed: %v", err)
	}

	if readObj == nil {
		t.Fatal("ReadObject() returned nil")
	}

	// Verify it's a blob
	readBlob, ok := readObj.(*blob.Blob)
	if !ok {
		t.Fatalf("expected *blob.Blob, got %T", readObj)
	}

	// Verify content
	content, err := readBlob.Content()
	if err != nil {
		t.Fatalf("failed to get blob content: %v", err)
	}

	if string(content) != string(testData) {
		t.Errorf("content mismatch: got %s, want %s", content, testData)
	}
}

func TestRepositoryExists(t *testing.T) {
	repoPath, cleanup := setupTestDirectory(t)
	defer cleanup()

	// Should not exist initially
	exists, err := RepositoryExists(repoPath)
	if err != nil {
		t.Fatalf("RepositoryExists() failed: %v", err)
	}
	if exists {
		t.Error("repository should not exist before initialization")
	}

	// Initialize repository
	repo := NewSourceRepository()
	err = repo.Initialize(repoPath)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Should exist now
	exists, err = RepositoryExists(repoPath)
	if err != nil {
		t.Fatalf("RepositoryExists() failed: %v", err)
	}
	if !exists {
		t.Error("repository should exist after initialization")
	}
}

func TestFindRepository(t *testing.T) {
	repoPath, cleanup := setupTestDirectory(t)
	defer cleanup()

	// Initialize repository
	repo := NewSourceRepository()
	err := repo.Initialize(repoPath)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(repoPath.String(), "src", "main")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	subDirPath, err := scpath.NewRepositoryPath(subDir)
	if err != nil {
		t.Fatalf("failed to create subdirectory path: %v", err)
	}

	// Find repository from subdirectory
	foundRepo, err := FindRepository(subDirPath)
	if err != nil {
		t.Fatalf("FindRepository() failed: %v", err)
	}

	if foundRepo == nil {
		t.Fatal("FindRepository() returned nil")
	}

	// Verify it's the same repository
	if foundRepo.WorkingDirectory().String() != repoPath.String() {
		t.Errorf("found repository at %s, want %s", foundRepo.WorkingDirectory(), repoPath)
	}
}

func TestFindRepository_NotFound(t *testing.T) {
	tempDir, cleanup := setupTestDirectory(t)
	defer cleanup()

	// Don't initialize repository, just search
	foundRepo, err := FindRepository(tempDir)
	if err != nil {
		t.Fatalf("FindRepository() failed: %v", err)
	}

	if foundRepo != nil {
		t.Error("FindRepository() should return nil when no repository found")
	}
}

func TestOpen(t *testing.T) {
	repoPath, cleanup := setupTestDirectory(t)
	defer cleanup()

	// Initialize repository
	repo := NewSourceRepository()
	err := repo.Initialize(repoPath)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Open the repository
	openedRepo, err := Open(repoPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	if openedRepo == nil {
		t.Fatal("Open() returned nil")
	}

	if !openedRepo.IsInitialized() {
		t.Error("opened repository should be initialized")
	}

	if openedRepo.WorkingDirectory().String() != repoPath.String() {
		t.Errorf("opened repository working dir = %s, want %s", openedRepo.WorkingDirectory(), repoPath)
	}
}

func TestOpen_NonExistent(t *testing.T) {
	tempDir, cleanup := setupTestDirectory(t)
	defer cleanup()

	// Try to open non-existent repository
	_, err := Open(tempDir)
	if err == nil {
		t.Error("Open() should fail for non-existent repository")
	}
}

func TestSourceRepository_Exists(t *testing.T) {
	repoPath, cleanup := setupTestDirectory(t)
	defer cleanup()

	repo := NewSourceRepository()
	err := repo.Initialize(repoPath)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	exists, err := repo.Exists()
	if err != nil {
		t.Fatalf("Exists() failed: %v", err)
	}

	if !exists {
		t.Error("Exists() should return true for initialized repository")
	}
}

func TestSourceRepository_UninitializedPanics(t *testing.T) {
	repo := NewSourceRepository()

	// Test that accessing methods on uninitialized repo panics or errors appropriately
	defer func() {
		if r := recover(); r == nil {
			t.Error("WorkingDirectory() should panic on uninitialized repository")
		}
	}()

	repo.WorkingDirectory()
}

func TestSourceRepository_ObjectStore(t *testing.T) {
	repo := NewSourceRepository()

	store := repo.ObjectStore()
	if store == nil {
		t.Error("ObjectStore() should return non-nil even before initialization")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
