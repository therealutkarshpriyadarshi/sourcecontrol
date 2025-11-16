package main

import (
	"context"
	"os"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/refs/branch"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

func TestBranchCommand(t *testing.T) {
	// Save and restore current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Set up git config for commits
	os.Setenv("GIT_AUTHOR_NAME", "Test User")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	defer os.Unsetenv("GIT_AUTHOR_NAME")
	defer os.Unsetenv("GIT_AUTHOR_EMAIL")

	t.Run("list branches with no commits", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Run branch command
		cmd := newBranchCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("branch command failed: %v", err)
		}

		// Just verify command succeeds - output goes to stdout
	})

	t.Run("create new branch", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create initial commit
		h.WriteFile("test.txt", "content")
		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// Create new branch
		cmd := newBranchCmd()
		cmd.SetArgs([]string{"feature"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("create branch failed: %v", err)
		}

		// Verify branch was created
		branchMgr := branch.NewManager(repo)
		if err := branchMgr.Init(); err != nil {
			t.Fatalf("failed to init branch manager: %v", err)
		}

		ctx := context.Background()
		branches, err := branchMgr.ListBranches(ctx)
		if err != nil {
			t.Fatalf("failed to list branches: %v", err)
		}

		foundFeature := false
		for _, br := range branches {
			if br.Name == "feature" {
				foundFeature = true
				break
			}
		}

		if !foundFeature {
			t.Error("expected to find 'feature' branch")
		}
	})

	t.Run("list branches shows current branch", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create initial commit
		h.WriteFile("test.txt", "content")
		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// List branches
		cmd := newBranchCmd()
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("branch command failed: %v", err)
		}

		// Just verify command succeeds - output goes to stdout
	})

	t.Run("delete branch", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create initial commit
		h.WriteFile("test.txt", "content")
		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// Create branch
		createCmd := newBranchCmd()
		createCmd.SetArgs([]string{"feature"})
		if err := createCmd.Execute(); err != nil {
			t.Fatalf("create branch failed: %v", err)
		}

		// Delete branch
		deleteCmd := newBranchCmd()
		deleteCmd.SetArgs([]string{"-d", "feature"})

		if err := deleteCmd.Execute(); err != nil {
			t.Fatalf("delete branch failed: %v", err)
		}

		// Verify branch was deleted
		branchMgr := branch.NewManager(repo)
		if err := branchMgr.Init(); err != nil {
			t.Fatalf("failed to init branch manager: %v", err)
		}

		ctx := context.Background()
		branches, err := branchMgr.ListBranches(ctx)
		if err != nil {
			t.Fatalf("failed to list branches: %v", err)
		}

		for _, br := range branches {
			if br.Name == "feature" {
				t.Error("expected 'feature' branch to be deleted")
			}
		}
	})

	t.Run("create multiple branches", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create initial commit
		h.WriteFile("test.txt", "content")
		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// Create multiple branches
		branchNames := []string{"feature1", "feature2", "bugfix"}
		for _, name := range branchNames {
			cmd := newBranchCmd()
			cmd.SetArgs([]string{name})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("create branch %s failed: %v", name, err)
			}
		}

		// Verify all branches exist
		branchMgr := branch.NewManager(repo)
		if err := branchMgr.Init(); err != nil {
			t.Fatalf("failed to init branch manager: %v", err)
		}

		ctx := context.Background()
		branches, err := branchMgr.ListBranches(ctx)
		if err != nil {
			t.Fatalf("failed to list branches: %v", err)
		}

		// Should have master + 3 new branches = 4 total
		if len(branches) < 3 {
			t.Errorf("expected at least 3 branches, got %d", len(branches))
		}

		// Check each branch exists
		branchMap := make(map[string]bool)
		for _, br := range branches {
			branchMap[br.Name] = true
		}

		for _, name := range branchNames {
			if !branchMap[name] {
				t.Errorf("expected to find branch %s", name)
			}
		}
	})

	t.Run("branch without repository fails", func(t *testing.T) {
		h := NewTestHelper(t)
		// Don't initialize repo
		h.Chdir()
		defer os.Chdir(origDir)

		// Try to create branch
		cmd := newBranchCmd()
		cmd.SetArgs([]string{"feature"})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when creating branch outside repository")
		}
	})

	t.Run("delete non-existent branch fails", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create initial commit
		h.WriteFile("test.txt", "content")
		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// Try to delete non-existent branch
		cmd := newBranchCmd()
		cmd.SetArgs([]string{"-d", "nonexistent"})

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when deleting non-existent branch")
		}
	})

	t.Run("create branch with same name twice fails", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create initial commit
		h.WriteFile("test.txt", "content")
		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// Create branch
		cmd1 := newBranchCmd()
		cmd1.SetArgs([]string{"feature"})
		if err := cmd1.Execute(); err != nil {
			t.Fatalf("first create branch failed: %v", err)
		}

		// Try to create same branch again
		cmd2 := newBranchCmd()
		cmd2.SetArgs([]string{"feature"})

		err := cmd2.Execute()
		if err == nil {
			t.Error("expected error when creating branch with duplicate name")
		}
	})

	t.Run("list branches with -l flag", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create initial commit
		h.WriteFile("test.txt", "content")
		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// Create a branch
		createCmd := newBranchCmd()
		createCmd.SetArgs([]string{"feature"})
		if err := createCmd.Execute(); err != nil {
			t.Fatalf("create branch failed: %v", err)
		}

		// List branches with -l flag
		cmd := newBranchCmd()
		cmd.SetArgs([]string{"-l"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("branch list command failed: %v", err)
		}

		// Just verify command succeeds - output goes to stdout
	})

	t.Run("branch names with special characters", func(t *testing.T) {
		h := NewTestHelper(t)
		repo := h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Create initial commit
		h.WriteFile("test.txt", "content")
		indexMgr := index.NewManager(repo.WorkingDirectory())
		if err := indexMgr.Initialize(); err != nil {
			t.Fatalf("failed to initialize index: %v", err)
		}

		objectStore := store.NewFileObjectStore()
		objectStore.Initialize(repo.WorkingDirectory())
		if _, err := indexMgr.Add([]string{"test.txt"}, objectStore); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Initial commit"})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed: %v", err)
		}

		// Create branch with dashes and underscores
		branchNames := []string{"feature-123", "bugfix_456", "release-v1.0"}
		for _, name := range branchNames {
			cmd := newBranchCmd()
			cmd.SetArgs([]string{name})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("create branch %s failed: %v", name, err)
			}
		}

		// Verify branches were created
		branchMgr := branch.NewManager(repo)
		if err := branchMgr.Init(); err != nil {
			t.Fatalf("failed to init branch manager: %v", err)
		}

		ctx := context.Background()
		branches, err := branchMgr.ListBranches(ctx)
		if err != nil {
			t.Fatalf("failed to list branches: %v", err)
		}

		branchMap := make(map[string]bool)
		for _, br := range branches {
			branchMap[br.Name] = true
		}

		for _, name := range branchNames {
			if !branchMap[name] {
				t.Errorf("expected to find branch %s", name)
			}
		}
	})
}
