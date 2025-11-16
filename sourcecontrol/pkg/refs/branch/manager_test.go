package branch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// TestBranchManager_CreateBranch tests basic branch creation
func TestBranchManager_CreateBranch(t *testing.T) {
	// Setup test repository
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create initial commit
	commitSHA := createTestCommit(t, repo, "Initial commit")

	// Create branch manager
	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Test: Create a new branch
	info, err := mgr.CreateBranch(ctx, "feature/test", WithStartPoint(commitSHA.String()))
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	if info.Name != "feature/test" {
		t.Errorf("Expected branch name 'feature/test', got '%s'", info.Name)
	}

	if info.SHA != commitSHA {
		t.Errorf("Expected SHA %s, got %s", commitSHA.Short(), info.SHA.Short())
	}

	// Test: Verify branch exists
	exists, err := mgr.BranchExists("feature/test")
	if err != nil {
		t.Fatalf("Failed to check branch exists: %v", err)
	}
	if !exists {
		t.Error("Branch should exist")
	}
}

// TestBranchManager_CreateBranchAlreadyExists tests creating a branch that already exists
func TestBranchManager_CreateBranchAlreadyExists(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commitSHA := createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Create first branch
	_, err := mgr.CreateBranch(ctx, "existing", WithStartPoint(commitSHA.String()))
	if err != nil {
		t.Fatalf("Failed to create first branch: %v", err)
	}

	// Test: Try to create same branch again
	_, err = mgr.CreateBranch(ctx, "existing", WithStartPoint(commitSHA.String()))
	if err == nil {
		t.Fatal("Expected error when creating existing branch")
	}

	// Check it's the right error type (could be wrapped)
	var alreadyExistsErr *AlreadyExistsError
	if !errors.As(err, &alreadyExistsErr) {
		t.Errorf("Expected AlreadyExistsError, got %T: %v", err, err)
	}
}

// TestBranchManager_ListBranches tests listing all branches
func TestBranchManager_ListBranches(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commitSHA := createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Create multiple branches
	branchNames := []string{"feature/a", "feature/b", "bugfix/c"}
	for _, name := range branchNames {
		_, err := mgr.CreateBranch(ctx, name, WithStartPoint(commitSHA.String()))
		if err != nil {
			t.Fatalf("Failed to create branch %s: %v", name, err)
		}
	}

	// Test: List all branches
	branches, err := mgr.ListBranches(ctx)
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	// We created 3 branches, but master might also exist from init
	if len(branches) < len(branchNames) {
		t.Errorf("Expected at least %d branches, got %d", len(branchNames), len(branches))
	}

	// Verify our created branches are in the list
	branchMap := make(map[string]bool)
	for _, b := range branches {
		branchMap[b.Name] = true
	}

	for _, name := range branchNames {
		if !branchMap[name] {
			t.Errorf("Branch %s not found in list", name)
		}
	}
}

// TestBranchManager_DeleteBranch tests branch deletion
func TestBranchManager_DeleteBranch(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commitSHA := createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Create a branch
	_, err := mgr.CreateBranch(ctx, "temp", WithStartPoint(commitSHA.String()))
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Test: Delete the branch
	err = mgr.DeleteBranch(ctx, "temp")
	if err != nil {
		t.Fatalf("Failed to delete branch: %v", err)
	}

	// Verify it's gone
	exists, err := mgr.BranchExists("temp")
	if err != nil {
		t.Fatalf("Failed to check branch exists: %v", err)
	}
	if exists {
		t.Error("Branch should not exist after deletion")
	}
}

// TestBranchManager_RenameBranch tests branch renaming
func TestBranchManager_RenameBranch(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commitSHA := createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Create a branch
	_, err := mgr.CreateBranch(ctx, "old-name", WithStartPoint(commitSHA.String()))
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Test: Rename the branch
	err = mgr.RenameBranch(ctx, "old-name", "new-name")
	if err != nil {
		t.Fatalf("Failed to rename branch: %v", err)
	}

	// Verify old name is gone
	exists, err := mgr.BranchExists("old-name")
	if err != nil {
		t.Fatalf("Failed to check old branch: %v", err)
	}
	if exists {
		t.Error("Old branch name should not exist")
	}

	// Verify new name exists
	exists, err = mgr.BranchExists("new-name")
	if err != nil {
		t.Fatalf("Failed to check new branch: %v", err)
	}
	if !exists {
		t.Error("New branch name should exist")
	}

	// Verify it points to the same commit
	info, err := mgr.GetBranch(ctx, "new-name")
	if err != nil {
		t.Fatalf("Failed to get renamed branch: %v", err)
	}
	if info.SHA != commitSHA {
		t.Errorf("Expected SHA %s, got %s", commitSHA.Short(), info.SHA.Short())
	}
}

// setupTestRepo creates a temporary test repository
func setupTestRepo(t *testing.T) (*sourcerepo.SourceRepository, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "branch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create and initialize repository
	repo := sourcerepo.NewSourceRepository()
	repoPath := scpath.RepositoryPath(tmpDir)

	if err := repo.Initialize(repoPath); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

// createTestCommit creates a test commit and returns its SHA
func createTestCommit(t *testing.T, repo *sourcerepo.SourceRepository, message string) objects.ObjectHash {
	t.Helper()

	// Create an empty tree
	emptyTree := tree.NewTree([]*tree.TreeEntry{})
	treeSHA, err := repo.WriteObject(emptyTree)
	if err != nil {
		t.Fatalf("Failed to write tree: %v", err)
	}

	// Create commit
	author, err := commit.NewCommitPerson("Test User", "test@example.com", time.Now())
	if err != nil {
		t.Fatalf("Failed to create author: %v", err)
	}
	c, err := commit.NewCommitBuilder().
		TreeHash(treeSHA).
		Author(author).
		Committer(author).
		Message(message).
		Build()
	if err != nil {
		t.Fatalf("Failed to build commit: %v", err)
	}

	commitSHA, err := repo.WriteObject(c)
	if err != nil {
		t.Fatalf("Failed to write commit: %v", err)
	}

	// Update master branch to point to this commit
	refPath := filepath.Join(repo.SourceDirectory().String(), "refs", "heads", "master")
	if err := os.MkdirAll(filepath.Dir(refPath), 0755); err != nil {
		t.Fatalf("Failed to create refs dir: %v", err)
	}

	if err := os.WriteFile(refPath, []byte(commitSHA.String()+"\n"), 0644); err != nil {
		t.Fatalf("Failed to write branch ref: %v", err)
	}

	return commitSHA
}

// TestBranchManager_CreateBranchWithForce tests force creation of an existing branch
func TestBranchManager_CreateBranchWithForce(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commit1 := createTestCommit(t, repo, "First commit")
	commit2 := createTestCommitWithParent(t, repo, "Second commit", commit1)

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Create initial branch
	_, err := mgr.CreateBranch(ctx, "test", WithStartPoint(commit1.String()))
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Verify it points to commit1
	info, err := mgr.GetBranch(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get branch: %v", err)
	}
	if info.SHA != commit1 {
		t.Errorf("Expected SHA %s, got %s", commit1.Short(), info.SHA.Short())
	}

	// Force create with commit2
	_, err = mgr.CreateBranch(ctx, "test", WithStartPoint(commit2.String()), WithForceCreate())
	if err != nil {
		t.Fatalf("Failed to force create branch: %v", err)
	}

	// Verify it now points to commit2
	info, err = mgr.GetBranch(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get branch: %v", err)
	}
	if info.SHA != commit2 {
		t.Errorf("Expected SHA %s, got %s", commit2.Short(), info.SHA.Short())
	}
}

// TestBranchManager_CreateBranchInvalidStartPoint tests creating a branch with invalid start point
func TestBranchManager_CreateBranchInvalidStartPoint(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Test: Create branch from non-existent commit
	_, err := mgr.CreateBranch(ctx, "test", WithStartPoint("0000000000000000000000000000000000000000"))
	if err == nil {
		t.Fatal("Expected error when creating branch from non-existent commit")
	}
}

// TestBranchManager_CreateBranchInvalidName tests creating branches with invalid names
func TestBranchManager_CreateBranchInvalidName(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commitSHA := createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	invalidNames := []string{
		"",
		".hidden",
		"branch.lock",
		"my branch",
		"branch~1",
		"/branch",
		"branch/",
		"feature//test",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			_, err := mgr.CreateBranch(ctx, name, WithStartPoint(commitSHA.String()))
			if err == nil {
				t.Errorf("Expected error for invalid branch name '%s'", name)
			}

			var invalidNameErr *InvalidNameError
			if !errors.As(err, &invalidNameErr) {
				t.Errorf("Expected InvalidNameError, got %T", err)
			}
		})
	}
}

// TestBranchManager_DeleteNonExistentBranch tests deleting a branch that doesn't exist
func TestBranchManager_DeleteNonExistentBranch(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Test: Delete non-existent branch
	err := mgr.DeleteBranch(ctx, "non-existent")
	if err == nil {
		t.Fatal("Expected error when deleting non-existent branch")
	}

	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Expected NotFoundError, got %T", err)
	}
}

// TestBranchManager_RenameNonExistentBranch tests renaming a branch that doesn't exist
func TestBranchManager_RenameNonExistentBranch(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Test: Rename non-existent branch
	err := mgr.RenameBranch(ctx, "non-existent", "new-name")
	if err == nil {
		t.Fatal("Expected error when renaming non-existent branch")
	}

	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Expected NotFoundError, got %T", err)
	}
}

// TestBranchManager_RenameToExistingBranch tests renaming to a branch that already exists
func TestBranchManager_RenameToExistingBranch(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commitSHA := createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Create two branches
	_, err := mgr.CreateBranch(ctx, "branch1", WithStartPoint(commitSHA.String()))
	if err != nil {
		t.Fatalf("Failed to create branch1: %v", err)
	}

	_, err = mgr.CreateBranch(ctx, "branch2", WithStartPoint(commitSHA.String()))
	if err != nil {
		t.Fatalf("Failed to create branch2: %v", err)
	}

	// Test: Rename branch1 to branch2 (should fail)
	err = mgr.RenameBranch(ctx, "branch1", "branch2")
	if err == nil {
		t.Fatal("Expected error when renaming to existing branch")
	}

	var alreadyExistsErr *AlreadyExistsError
	if !errors.As(err, &alreadyExistsErr) {
		t.Errorf("Expected AlreadyExistsError, got %T", err)
	}
}

// TestBranchManager_RenameWithForce tests force renaming to an existing branch
func TestBranchManager_RenameWithForce(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commit1 := createTestCommit(t, repo, "First commit")
	commit2 := createTestCommitWithParent(t, repo, "Second commit", commit1)

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Create two branches pointing to different commits
	_, err := mgr.CreateBranch(ctx, "branch1", WithStartPoint(commit1.String()))
	if err != nil {
		t.Fatalf("Failed to create branch1: %v", err)
	}

	_, err = mgr.CreateBranch(ctx, "branch2", WithStartPoint(commit2.String()))
	if err != nil {
		t.Fatalf("Failed to create branch2: %v", err)
	}

	// Force rename branch1 to branch2
	err = mgr.RenameBranch(ctx, "branch1", "branch2", WithForceRename())
	if err != nil {
		t.Fatalf("Failed to force rename branch: %v", err)
	}

	// Verify branch1 no longer exists
	exists, err := mgr.BranchExists("branch1")
	if err != nil {
		t.Fatalf("Failed to check branch1: %v", err)
	}
	if exists {
		t.Error("branch1 should not exist after rename")
	}

	// Verify branch2 now points to commit1
	info, err := mgr.GetBranch(ctx, "branch2")
	if err != nil {
		t.Fatalf("Failed to get branch2: %v", err)
	}
	if info.SHA != commit1 {
		t.Errorf("Expected SHA %s, got %s", commit1.Short(), info.SHA.Short())
	}
}

// TestBranchManager_DeeplyNestedBranches tests creating and listing deeply nested branches
func TestBranchManager_DeeplyNestedBranches(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commitSHA := createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Create branches with various nesting levels
	nestedBranches := []string{
		"feature/auth/login",
		"feature/auth/logout",
		"feature/ui/components/button",
		"bugfix/critical/issue-123",
		"release/v1.0.0/rc1",
	}

	for _, name := range nestedBranches {
		_, err := mgr.CreateBranch(ctx, name, WithStartPoint(commitSHA.String()))
		if err != nil {
			t.Fatalf("Failed to create branch %s: %v", name, err)
		}
	}

	// List all branches
	branches, err := mgr.ListBranches(ctx)
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	// Verify all nested branches are present
	branchMap := make(map[string]bool)
	for _, b := range branches {
		branchMap[b.Name] = true
	}

	for _, name := range nestedBranches {
		if !branchMap[name] {
			t.Errorf("Branch %s not found in list", name)
		}
	}
}

// TestBranchManager_GetNonExistentBranch tests getting info for a non-existent branch
func TestBranchManager_GetNonExistentBranch(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Test: Get non-existent branch
	_, err := mgr.GetBranch(ctx, "non-existent")
	if err == nil {
		t.Fatal("Expected error when getting non-existent branch")
	}

	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Expected NotFoundError, got %T", err)
	}
}

// TestBranchManager_MultipleCommits tests branches with multiple commits
func TestBranchManager_MultipleCommits(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// Create a chain of commits
	commit1 := createTestCommit(t, repo, "First commit")
	commit2 := createTestCommitWithParent(t, repo, "Second commit", commit1)
	commit3 := createTestCommitWithParent(t, repo, "Third commit", commit2)

	// Create branch at different points
	_, err := mgr.CreateBranch(ctx, "at-commit1", WithStartPoint(commit1.String()))
	if err != nil {
		t.Fatalf("Failed to create branch at commit1: %v", err)
	}

	_, err = mgr.CreateBranch(ctx, "at-commit3", WithStartPoint(commit3.String()))
	if err != nil {
		t.Fatalf("Failed to create branch at commit3: %v", err)
	}

	// Get info for branch at commit3
	info, err := mgr.GetBranch(ctx, "at-commit3")
	if err != nil {
		t.Fatalf("Failed to get branch: %v", err)
	}

	// Verify commit count
	if info.CommitCount != 3 {
		t.Errorf("Expected commit count 3, got %d", info.CommitCount)
	}

	// Verify last commit message
	if info.LastCommitMessage != "Third commit" {
		t.Errorf("Expected message 'Third commit', got '%s'", info.LastCommitMessage)
	}
}

// TestBranchManager_ListBranchesEmptyRepo tests listing branches in an empty repository
func TestBranchManager_ListBranchesEmptyRepo(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	// List branches (should only have master if it exists)
	branches, err := mgr.ListBranches(ctx)
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	// Should have at least master from initialization
	if len(branches) == 0 {
		t.Log("No branches found (expected if no initial commit)")
	}
}

// TestBranchManager_CurrentBranch tests getting the current branch
func TestBranchManager_CurrentBranch(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commitSHA := createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Current branch should be master
	current, err := mgr.CurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	if current != "master" {
		t.Errorf("Expected current branch 'master', got '%s'", current)
	}

	ctx := context.Background()

	// Create and checkout a new branch
	_, err = mgr.CreateBranch(ctx, "new-branch", WithStartPoint(commitSHA.String()), WithCheckout())
	if err != nil {
		t.Fatalf("Failed to create and checkout branch: %v", err)
	}

	// Current branch should now be new-branch
	current, err = mgr.CurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	if current != "new-branch" {
		t.Errorf("Expected current branch 'new-branch', got '%s'", current)
	}
}

// TestBranchManager_BranchWithSpecialCharacters tests branches with allowed special characters
func TestBranchManager_BranchWithSpecialCharacters(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commitSHA := createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()

	validNames := []string{
		"feature-123",
		"bug_fix",
		"release-v1.0.0",
		"test/branch-name_123",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			_, err := mgr.CreateBranch(ctx, name, WithStartPoint(commitSHA.String()))
			if err != nil {
				t.Errorf("Failed to create branch with valid name '%s': %v", name, err)
			}

			// Verify it exists
			exists, err := mgr.BranchExists(name)
			if err != nil {
				t.Fatalf("Failed to check branch exists: %v", err)
			}
			if !exists {
				t.Errorf("Branch '%s' should exist", name)
			}
		})
	}
}

// TestBranchManager_ContextCancellation tests context cancellation
func TestBranchManager_ContextCancellation(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commitSHA := createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Try to create branch with cancelled context
	_, err := mgr.CreateBranch(ctx, "test", WithStartPoint(commitSHA.String()))
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

// TestBranchManager_CurrentCommit tests getting the current commit SHA
func TestBranchManager_CurrentCommit(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	commitSHA := createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Get current commit
	current, err := mgr.CurrentCommit()
	if err != nil {
		t.Fatalf("Failed to get current commit: %v", err)
	}

	if current != commitSHA {
		t.Errorf("Expected current commit %s, got %s", commitSHA.Short(), current.Short())
	}
}

// TestBranchManager_IsDetached tests checking detached HEAD state
func TestBranchManager_IsDetached(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	createTestCommit(t, repo, "Initial commit")

	mgr := NewManager(repo)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Should not be detached initially
	detached, err := mgr.IsDetached()
	if err != nil {
		t.Fatalf("Failed to check detached state: %v", err)
	}
	if detached {
		t.Error("HEAD should not be detached initially")
	}
}

// createTestCommitWithParent creates a test commit with a parent commit
func createTestCommitWithParent(t *testing.T, repo *sourcerepo.SourceRepository, message string, parent objects.ObjectHash) objects.ObjectHash {
	t.Helper()

	// Create an empty tree
	emptyTree := tree.NewTree([]*tree.TreeEntry{})
	treeSHA, err := repo.WriteObject(emptyTree)
	if err != nil {
		t.Fatalf("Failed to write tree: %v", err)
	}

	// Create commit with parent
	author, err := commit.NewCommitPerson("Test User", "test@example.com", time.Now())
	if err != nil {
		t.Fatalf("Failed to create author: %v", err)
	}
	c, err := commit.NewCommitBuilder().
		TreeHash(treeSHA).
		Author(author).
		Committer(author).
		ParentHashes(parent).
		Message(message).
		Build()
	if err != nil {
		t.Fatalf("Failed to build commit: %v", err)
	}

	commitSHA, err := repo.WriteObject(c)
	if err != nil {
		t.Fatalf("Failed to write commit: %v", err)
	}

	// Update master branch to point to this commit
	refPath := filepath.Join(repo.SourceDirectory().String(), "refs", "heads", "master")
	if err := os.WriteFile(refPath, []byte(commitSHA.String()+"\n"), 0644); err != nil {
		t.Fatalf("Failed to write branch ref: %v", err)
	}

	return commitSHA
}
