package commitmanager

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects/blob"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// setupTestRepo creates a test repository
func setupTestRepo(t *testing.T) (*sourcerepo.SourceRepository, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "commitmanager-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	repo := sourcerepo.NewSourceRepository()
	repoPath := scpath.RepositoryPath(tempDir)
	if err := repo.Initialize(repoPath); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	return repo, tempDir
}

// setupTestConfig sets up test user configuration via environment variables
func setupTestConfig(t *testing.T, repo *sourcerepo.SourceRepository) {
	t.Helper()

	// Create a temporary HOME directory to isolate config files
	// This prevents tests from reading/writing to the real user config
	tempHome, err := os.MkdirTemp("", "test-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tempHome)
	})

	// Use t.Setenv for test-scoped environment variables
	// This is safe for parallel test execution and handles cleanup automatically
	t.Setenv("HOME", tempHome)        // Unix/Linux
	t.Setenv("USERPROFILE", tempHome) // Windows
	t.Setenv("GIT_AUTHOR_NAME", "Test User")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
}

// addFileToIndex adds a file to the index
func addFileToIndex(t *testing.T, repo *sourcerepo.SourceRepository, filename, content string) {
	t.Helper()

	// Write file to working directory
	filePath := filepath.Join(repo.WorkingDirectory().String(), filename)
	// Create parent directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatalf("Failed to create directory for %s: %v", filename, err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", filename, err)
	}

	// Create blob
	b := blob.NewBlob([]byte(content))
	blobSHA, err := repo.WriteObject(b)
	if err != nil {
		t.Fatalf("Failed to write blob: %v", err)
	}

	// Add to index
	indexPath := repo.SourceDirectory().IndexPath()
	idx, err := index.Read(indexPath.ToAbsolutePath())
	if err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	entry, err := index.NewEntryFromFileInfo(scpath.RelativePath(filename), info, blobSHA)
	if err != nil {
		t.Fatalf("Failed to create entry: %v", err)
	}

	idx.Add(entry)

	if err := idx.Write(indexPath.ToAbsolutePath()); err != nil {
		t.Fatalf("Failed to write index: %v", err)
	}
}

func TestNewManager(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	mgr := NewManager(repo)
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if mgr.repo != repo {
		t.Error("Manager repo not set correctly")
	}
	if mgr.treeBuilder == nil {
		t.Error("TreeBuilder not initialized")
	}
	if mgr.refManager == nil {
		t.Error("RefManager not initialized")
	}
}

func TestInitialize(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)
	setupTestConfig(t, repo)

	mgr := NewManager(repo)
	ctx := context.Background()

	err := mgr.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
}

func TestCreateCommit_EmptyMessage(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)
	setupTestConfig(t, repo)

	mgr := NewManager(repo)
	ctx := context.Background()

	if err := mgr.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	_, err := mgr.CreateCommit(ctx, CommitOptions{
		Message: "",
	})

	if err == nil {
		t.Fatal("Expected error for empty message, got nil")
	}
	if err != ErrEmptyMessage && !isCommitError(err, ErrEmptyMessage) {
		t.Errorf("Expected ErrEmptyMessage, got %v", err)
	}
}

func TestCreateCommit_NoChanges(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)
	setupTestConfig(t, repo)

	mgr := NewManager(repo)
	ctx := context.Background()

	if err := mgr.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	_, err := mgr.CreateCommit(ctx, CommitOptions{
		Message: "Test commit",
	})

	if err == nil {
		t.Fatal("Expected error for no changes, got nil")
	}
	if !isCommitError(err, ErrNoChanges) {
		t.Errorf("Expected ErrNoChanges, got %v", err)
	}
}

func TestCreateCommit_InitialCommit(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)
	setupTestConfig(t, repo)

	mgr := NewManager(repo)
	ctx := context.Background()

	if err := mgr.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Add a file to the index
	addFileToIndex(t, repo, "README.md", "# Test Project\n")

	// Create commit
	result, err := mgr.CreateCommit(ctx, CommitOptions{
		Message: "Initial commit",
	})

	if err != nil {
		t.Fatalf("CreateCommit failed: %v", err)
	}

	resultHash, _ := result.Hash()
	if resultHash == "" {
		t.Error("Expected commit SHA, got empty string")
	}
	if result.TreeSHA == "" {
		t.Error("Expected tree SHA, got empty string")
	}
	if len(result.ParentSHAs) != 0 {
		t.Errorf("Expected 0 parents for initial commit, got %d", len(result.ParentSHAs))
	}
	if result.Message != "Initial commit" {
		t.Errorf("Expected message 'Initial commit', got '%s'", result.Message)
	}
	if result.Author.Name != "Test User" {
		t.Errorf("Expected author 'Test User', got '%s'", result.Author.Name)
	}
	if result.Author.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", result.Author.Email)
	}
}

func TestCreateCommit_SecondCommit(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)
	setupTestConfig(t, repo)

	mgr := NewManager(repo)
	ctx := context.Background()

	if err := mgr.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create initial commit
	addFileToIndex(t, repo, "README.md", "# Test Project\n")
	firstCommit, err := mgr.CreateCommit(ctx, CommitOptions{
		Message: "Initial commit",
	})
	if err != nil {
		t.Fatalf("First commit failed: %v", err)
	}

	// Create second commit
	addFileToIndex(t, repo, "main.go", "package main\n")
	secondCommit, err := mgr.CreateCommit(ctx, CommitOptions{
		Message: "Add main.go",
	})
	if err != nil {
		t.Fatalf("Second commit failed: %v", err)
	}

	if len(secondCommit.ParentSHAs) != 1 {
		t.Errorf("Expected 1 parent, got %d", len(secondCommit.ParentSHAs))
	}
	firstCommitHash, _ := firstCommit.Hash()
	if secondCommit.ParentSHAs[0] != firstCommitHash {
		t.Error("Second commit parent should be first commit")
	}
}

func TestCreateCommit_WithCustomAuthor(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)
	setupTestConfig(t, repo)

	mgr := NewManager(repo)
	ctx := context.Background()

	if err := mgr.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	addFileToIndex(t, repo, "test.txt", "test content\n")

	customAuthor, err := commit.NewCommitPerson(
		"Custom Author",
		"custom@example.com",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to create commit person: %v", err)
	}

	result, err := mgr.CreateCommit(ctx, CommitOptions{
		Message: "Custom author commit",
		Author:  customAuthor,
	})

	if err != nil {
		t.Fatalf("CreateCommit failed: %v", err)
	}

	if result.Author.Name != "Custom Author" {
		t.Errorf("Expected author 'Custom Author', got '%s'", result.Author.Name)
	}
	if result.Author.Email != "custom@example.com" {
		t.Errorf("Expected email 'custom@example.com', got '%s'", result.Author.Email)
	}
}

func TestCreateCommit_AllowEmpty(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)
	setupTestConfig(t, repo)

	mgr := NewManager(repo)
	ctx := context.Background()

	if err := mgr.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	result, err := mgr.CreateCommit(ctx, CommitOptions{
		Message:    "Empty commit",
		AllowEmpty: true,
	})

	if err != nil {
		t.Fatalf("CreateCommit failed: %v", err)
	}

	resultHash, _ := result.Hash()
	if resultHash == "" {
		t.Error("Expected commit SHA, got empty string")
	}
}

func TestGetCommit(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)
	setupTestConfig(t, repo)

	mgr := NewManager(repo)
	ctx := context.Background()

	if err := mgr.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create a commit
	addFileToIndex(t, repo, "test.txt", "test\n")
	created, err := mgr.CreateCommit(ctx, CommitOptions{
		Message: "Test commit",
	})
	if err != nil {
		t.Fatalf("CreateCommit failed: %v", err)
	}

	// Get the commit
	createdHash, _ := created.Hash()
	retrieved, err := mgr.GetCommit(ctx, createdHash)
	if err != nil {
		t.Fatalf("GetCommit failed: %v", err)
	}

	retrievedHash, _ := retrieved.Hash()
	if retrievedHash != createdHash {
		t.Error("Retrieved commit SHA doesn't match created")
	}
	if retrieved.Message != created.Message {
		t.Error("Retrieved commit message doesn't match created")
	}
}

func TestGetHistory(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)
	setupTestConfig(t, repo)

	mgr := NewManager(repo)
	ctx := context.Background()

	if err := mgr.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create multiple commits
	addFileToIndex(t, repo, "file1.txt", "content1\n")
	commit1, _ := mgr.CreateCommit(ctx, CommitOptions{Message: "Commit 1"})

	addFileToIndex(t, repo, "file2.txt", "content2\n")
	_, _ = mgr.CreateCommit(ctx, CommitOptions{Message: "Commit 2"})

	addFileToIndex(t, repo, "file3.txt", "content3\n")
	commit3, _ := mgr.CreateCommit(ctx, CommitOptions{Message: "Commit 3"})

	// Get history
	history, err := mgr.GetHistory(ctx, "", 10)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 commits in history, got %d", len(history))
	}

	// History should be in reverse chronological order
	commit3Hash, _ := commit3.Hash()
	commit1Hash, _ := commit1.Hash()
	history0Hash, _ := history[0].Hash()
	history2Hash, _ := history[2].Hash()
	if history0Hash != commit3Hash {
		t.Error("First commit in history should be most recent")
	}
	if history2Hash != commit1Hash {
		t.Error("Last commit in history should be oldest")
	}
}

func TestGetHistory_WithLimit(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)
	setupTestConfig(t, repo)

	mgr := NewManager(repo)
	ctx := context.Background()

	if err := mgr.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create multiple commits
	for i := 1; i <= 5; i++ {
		addFileToIndex(t, repo, filepath.Join("file", string(rune('0'+i))+".txt"), "content\n")
		_, _ = mgr.CreateCommit(ctx, CommitOptions{Message: "Commit"})
	}

	// Get history with limit
	history, err := mgr.GetHistory(ctx, "", 3)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 commits with limit, got %d", len(history))
	}
}

// Helper function to check if error is a CommitError with specific underlying error
func isCommitError(err error, target error) bool {
	if ce, ok := err.(*CommitError); ok {
		return ce.Err == target
	}
	return false
}
