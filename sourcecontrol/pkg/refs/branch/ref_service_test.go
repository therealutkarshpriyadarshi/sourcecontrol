package branch

import (
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/refs"
)

// TestRefService_CreateAndResolve tests creating and resolving branches
func TestRefService_CreateAndResolve(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	refMgr := refs.NewRefManager(repo)
	refSvc := NewBranchRefManager(refMgr)

	// Create a test commit SHA
	testSHA := objects.ObjectHash("0123456789abcdef0123456789abcdef01234567")

	// Test: Create a branch
	err := refSvc.Create("test-branch", testSHA)
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Test: Resolve the branch
	resolvedSHA, err := refSvc.Resolve("test-branch")
	if err != nil {
		t.Fatalf("Failed to resolve branch: %v", err)
	}

	if resolvedSHA != testSHA {
		t.Errorf("Expected SHA %s, got %s", testSHA, resolvedSHA)
	}
}

// TestRefService_CreateDuplicate tests creating a duplicate branch
func TestRefService_CreateDuplicate(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	refMgr := refs.NewRefManager(repo)
	refSvc := NewBranchRefManager(refMgr)

	testSHA := objects.ObjectHash("0123456789abcdef0123456789abcdef01234567")

	// Create first branch
	err := refSvc.Create("duplicate", testSHA)
	if err != nil {
		t.Fatalf("Failed to create first branch: %v", err)
	}

	// Test: Try to create duplicate
	err = refSvc.Create("duplicate", testSHA)
	if err == nil {
		t.Fatal("Expected error when creating duplicate branch")
	}

	if _, ok := err.(*AlreadyExistsError); !ok {
		t.Errorf("Expected AlreadyExistsError, got %T", err)
	}
}

// TestRefService_Update tests updating a branch
func TestRefService_Update(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	refMgr := refs.NewRefManager(repo)
	refSvc := NewBranchRefManager(refMgr)

	oldSHA := objects.ObjectHash("0123456789abcdef0123456789abcdef01234567")
	newSHA := objects.ObjectHash("fedcba9876543210fedcba9876543210fedcba98")

	// Create branch
	err := refSvc.Create("update-test", oldSHA)
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Test: Update the branch
	err = refSvc.Update("update-test", newSHA, false)
	if err != nil {
		t.Fatalf("Failed to update branch: %v", err)
	}

	// Verify it points to new SHA
	resolvedSHA, err := refSvc.Resolve("update-test")
	if err != nil {
		t.Fatalf("Failed to resolve branch: %v", err)
	}

	if resolvedSHA != newSHA {
		t.Errorf("Expected SHA %s, got %s", newSHA, resolvedSHA)
	}
}

// TestRefService_Delete tests deleting a branch
func TestRefService_Delete(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	refMgr := refs.NewRefManager(repo)
	refSvc := NewBranchRefManager(refMgr)

	testSHA := objects.ObjectHash("0123456789abcdef0123456789abcdef01234567")

	// Create branch
	err := refSvc.Create("delete-test", testSHA)
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Verify it exists
	exists, err := refSvc.Exists("delete-test")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Fatal("Branch should exist")
	}

	// Create and set a different branch as current (can't delete current)
	err = refSvc.Create("other", testSHA)
	if err != nil {
		t.Fatalf("Failed to create other branch: %v", err)
	}
	err = refSvc.SetHead("other")
	if err != nil {
		t.Fatalf("Failed to set HEAD: %v", err)
	}

	// Test: Delete the branch
	err = refSvc.Delete("delete-test")
	if err != nil {
		t.Fatalf("Failed to delete branch: %v", err)
	}

	// Verify it's gone
	exists, err = refSvc.Exists("delete-test")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Branch should not exist after deletion")
	}
}

// TestRefService_List tests listing all branches
func TestRefService_List(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	refMgr := refs.NewRefManager(repo)
	refSvc := NewBranchRefManager(refMgr)

	testSHA := objects.ObjectHash("0123456789abcdef0123456789abcdef01234567")

	// Create multiple branches
	branchNames := []string{"alpha", "beta", "gamma"}
	for _, name := range branchNames {
		err := refSvc.Create(name, testSHA)
		if err != nil {
			t.Fatalf("Failed to create branch %s: %v", name, err)
		}
	}

	// Test: List branches
	branches, err := refSvc.List()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	// Verify all created branches are in the list
	branchMap := make(map[string]bool)
	for _, name := range branches {
		branchMap[name] = true
	}

	for _, name := range branchNames {
		if !branchMap[name] {
			t.Errorf("Branch %s not found in list", name)
		}
	}
}

// TestRefService_CurrentAndSetHead tests getting and setting current branch
func TestRefService_CurrentAndSetHead(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	refMgr := refs.NewRefManager(repo)
	refSvc := NewBranchRefManager(refMgr)

	testSHA := objects.ObjectHash("0123456789abcdef0123456789abcdef01234567")

	// Create a branch
	err := refSvc.Create("test-branch", testSHA)
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Test: Set HEAD to this branch
	err = refSvc.SetHead("test-branch")
	if err != nil {
		t.Fatalf("Failed to set HEAD: %v", err)
	}

	// Test: Get current branch
	current, err := refSvc.Current()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	if current != "test-branch" {
		t.Errorf("Expected current branch 'test-branch', got '%s'", current)
	}

	// Test: Check not detached
	detached, err := refSvc.IsDetached()
	if err != nil {
		t.Fatalf("Failed to check detached: %v", err)
	}
	if detached {
		t.Error("HEAD should not be detached")
	}
}

// TestRefService_DetachedHead tests detached HEAD state
func TestRefService_DetachedHead(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	refMgr := refs.NewRefManager(repo)
	refSvc := NewBranchRefManager(refMgr)

	testSHA := objects.ObjectHash("0123456789abcdef0123456789abcdef01234567")

	// Test: Set HEAD to detached state
	err := refSvc.SetHeadDetached(testSHA)
	if err != nil {
		t.Fatalf("Failed to set detached HEAD: %v", err)
	}

	// Test: Check detached state
	detached, err := refSvc.IsDetached()
	if err != nil {
		t.Fatalf("Failed to check detached: %v", err)
	}
	if !detached {
		t.Error("HEAD should be detached")
	}

	// Test: Current should return empty string
	current, err := refSvc.Current()
	if err != nil {
		t.Fatalf("Failed to get current: %v", err)
	}
	if current != "" {
		t.Errorf("Expected empty current branch in detached state, got '%s'", current)
	}

	// Test: GetHeadSHA should return the SHA
	headSHA, err := refSvc.GetHeadSHA()
	if err != nil {
		t.Fatalf("Failed to get HEAD SHA: %v", err)
	}
	if headSHA != testSHA {
		t.Errorf("Expected HEAD SHA %s, got %s", testSHA, headSHA)
	}
}

// TestValidateBranchName tests the ValidateBranchName function
func TestValidateBranchName(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	refMgr := refs.NewRefManager(repo)
	refSvc := NewBranchRefManager(refMgr)
	testCases := []struct {
		name  string
		valid bool
	}{
		{"valid-name", true},
		{"feature/branch", true},
		{"test_123", true},
		{"", false},
		{".hidden", false},
		{"branch.lock", false},
		{"branch name", false},
		{"branch~1", false},
		{"/start-slash", false},
		{"end-slash/", false},
		{"double//slash", false},
		{"branch..name", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := refSvc.validateBranchName(tc.name)
			if tc.valid && err != nil {
				t.Errorf("Expected '%s' to be valid, got error: %v", tc.name, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("Expected '%s' to be invalid", tc.name)
			}
		})
	}
}
