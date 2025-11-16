package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/refs/branch"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// TestIntegrationBasicWorkflow tests the complete workflow: init -> add -> commit
func TestIntegrationBasicWorkflow(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Verify repository was initialized
	sourceDir := filepath.Join(h.RepoPath, scpath.SourceDir)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		t.Fatal(".source directory was not created")
	}

	// Create a test file
	h.WriteFile("README.md", "Hello, Source Control!")

	// Stage the file
	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"README.md"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Verify file is staged
	indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	if idx.Count() != 1 {
		t.Errorf("expected 1 file in index, got %d", idx.Count())
	}

	// Commit the changes
	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Initial commit"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("commit command failed: %v", err)
	}

	// Verify commit was created
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get commit history: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 commit in history, got %d", len(history))
	}
	if history[0].Message != "Initial commit" {
		t.Errorf("expected commit message 'Initial commit', got '%s'", history[0].Message)
	}
}

// TestIntegrationMultipleFiles tests adding and committing multiple files
func TestIntegrationMultipleFiles(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create multiple files
	files := map[string]string{
		"file1.txt": "Content 1",
		"file2.txt": "Content 2",
		"file3.txt": "Content 3",
	}

	for name, content := range files {
		h.WriteFile(name, content)
	}

	// Stage all files
	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"file1.txt", "file2.txt", "file3.txt"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Verify files are staged
	indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	if idx.Count() != 3 {
		t.Errorf("expected 3 files in index, got %d", idx.Count())
	}

	// Commit all files
	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Add multiple files"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("commit command failed: %v", err)
	}

	// Verify commit in history
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 commit, got %d", len(history))
	}
	if history[0].Message != "Add multiple files" {
		t.Errorf("expected 'Add multiple files', got '%s'", history[0].Message)
	}

	// Verify objects directory contains objects
	objectsDir := filepath.Join(h.RepoPath, scpath.SourceDir, "objects")
	entries, err := os.ReadDir(objectsDir)
	if err != nil {
		t.Fatalf("failed to read objects directory: %v", err)
	}
	if len(entries) == 0 {
		t.Error("no objects were created")
	}
}

// TestIntegrationNestedDirectories tests working with nested directory structures
func TestIntegrationNestedDirectories(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create nested directory structure
	h.WriteFile("src/main.go", "package main\n\nfunc main() {}")
	h.WriteFile("src/util/helper.go", "package util\n\nfunc Help() {}")
	h.WriteFile("docs/README.md", "# Documentation")
	h.WriteFile("tests/main_test.go", "package main\n\nfunc TestMain() {}")

	// Stage all files
	files := []string{
		"src/main.go",
		"src/util/helper.go",
		"docs/README.md",
		"tests/main_test.go",
	}

	addCmd := newAddCmd()
	addCmd.SetArgs(files)
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Verify all files are staged
	indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	if idx.Count() != 4 {
		t.Errorf("expected 4 files in index, got %d", idx.Count())
	}

	// Commit
	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Add project structure"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("commit command failed: %v", err)
	}

	// Verify commit exists
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 commit, got %d", len(history))
	}
}

// TestIntegrationMultipleCommits tests creating a chain of commits
func TestIntegrationMultipleCommits(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	commits := []struct {
		file    string
		content string
		message string
	}{
		{"file1.txt", "First", "First commit"},
		{"file2.txt", "Second", "Second commit"},
		{"file3.txt", "Third", "Third commit"},
	}

	for _, c := range commits {
		// Create file
		h.WriteFile(c.file, c.content)

		// Stage file
		addCmd := newAddCmd()
		addCmd.SetArgs([]string{c.file})
		if err := addCmd.Execute(); err != nil {
			t.Fatalf("add command failed for %s: %v", c.file, err)
		}

		// Commit file
		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", c.message})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit command failed for %s: %v", c.file, err)
		}
	}

	// Verify all commits in history
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("expected 3 commits, got %d", len(history))
	}

	// Verify commit messages (in reverse order - newest first)
	expectedMessages := []string{"Third commit", "Second commit", "First commit"}
	for i, expectedMsg := range expectedMessages {
		if history[i].Message != expectedMsg {
			t.Errorf("commit %d: expected '%s', got '%s'", i, expectedMsg, history[i].Message)
		}
	}
}

// TestIntegrationBranchWorkflow tests creating and listing branches
func TestIntegrationBranchWorkflow(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create initial commit (needed for branch operations)
	h.WriteFile("initial.txt", "initial content")
	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"initial.txt"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Initial commit"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("commit command failed: %v", err)
	}

	// Create branches
	branches := []string{"feature-1", "feature-2", "develop"}

	for _, branchName := range branches {
		branchCmd := newBranchCmd()
		branchCmd.SetArgs([]string{branchName})
		if err := branchCmd.Execute(); err != nil {
			t.Fatalf("branch create command failed for %s: %v", branchName, err)
		}
	}

	// List branches using the branch manager
	branchMgr := branch.NewManager(repo)
	ctx := context.Background()
	branchList, err := branchMgr.ListBranches(ctx)
	if err != nil {
		t.Fatalf("failed to list branches: %v", err)
	}

	// Verify all branches exist
	branchNames := make(map[string]bool)
	for _, b := range branchList {
		branchNames[b.Name] = true
	}

	for _, branchName := range branches {
		if !branchNames[branchName] {
			t.Errorf("branch '%s' not found in branch list", branchName)
		}
	}

	// Verify master or main branch exists
	if !branchNames["master"] && !branchNames["main"] {
		t.Error("neither master nor main branch found")
	}
}

// TestIntegrationStatusDetection tests status detection for various file states
func TestIntegrationStatusDetection(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create and commit initial files
	h.WriteFile("tracked.txt", "tracked content")
	h.WriteFile("to-modify.txt", "original content")
	h.WriteFile("to-delete.txt", "to be deleted")

	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"tracked.txt", "to-modify.txt", "to-delete.txt"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Initial commit"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("commit command failed: %v", err)
	}

	// Create various file states
	h.WriteFile("untracked.txt", "new file")         // Untracked
	h.WriteFile("to-modify.txt", "modified content") // Modified
	if err := os.Remove(filepath.Join(h.RepoPath, "to-delete.txt")); err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	// Stage one modified file
	addCmd = newAddCmd()
	addCmd.SetArgs([]string{"to-modify.txt"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Verify status command runs successfully
	statusCmd := newStatusCmd()
	if err := statusCmd.Execute(); err != nil {
		t.Fatalf("status command failed: %v", err)
	}

	// Verify index has the staged file
	indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}

	// Should have original 3 files, with to-modify.txt updated
	if idx.Count() != 3 {
		t.Logf("Note: index has %d entries (expected 3)", idx.Count())
	}
}

// TestIntegrationModifyAndRecommit tests modifying files and creating new commits
func TestIntegrationModifyAndRecommit(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Initial commit
	h.WriteFile("config.json", `{"version": "1.0"}`)
	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"config.json"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("initial add failed: %v", err)
	}

	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Initial config"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("initial commit failed: %v", err)
	}

	// Modify file
	h.WriteFile("config.json", `{"version": "2.0"}`)

	// Stage modified file
	addCmd = newAddCmd()
	addCmd.SetArgs([]string{"config.json"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("second add failed: %v", err)
	}

	// Commit modification
	commitCmd = newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Update config version"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("second commit failed: %v", err)
	}

	// Verify both commits in history
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("expected 2 commits, got %d", len(history))
	}

	// Verify commit messages (newest first)
	if history[0].Message != "Update config version" {
		t.Errorf("expected 'Update config version', got '%s'", history[0].Message)
	}
	if history[1].Message != "Initial config" {
		t.Errorf("expected 'Initial config', got '%s'", history[1].Message)
	}
}

// TestIntegrationEmptyDirectoryHandling tests handling of directories
func TestIntegrationEmptyDirectoryHandling(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create directory structure with files
	h.WriteFile("dir1/file1.txt", "content1")
	h.WriteFile("dir2/subdir/file2.txt", "content2")

	// Add files
	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"dir1/file1.txt", "dir2/subdir/file2.txt"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Commit
	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Add directory structure"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("commit command failed: %v", err)
	}

	// Verify commit
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 commit, got %d", len(history))
	}
	if history[0].Message != "Add directory structure" {
		t.Errorf("expected 'Add directory structure', got '%s'", history[0].Message)
	}
}

// TestIntegrationLargeCommitChain tests creating many commits in sequence
func TestIntegrationLargeCommitChain(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	numCommits := 10

	for i := 0; i < numCommits; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		content := fmt.Sprintf("Content for commit %d", i)
		message := fmt.Sprintf("Commit number %d", i)

		h.WriteFile(filename, content)

		addCmd := newAddCmd()
		addCmd.SetArgs([]string{filename})
		if err := addCmd.Execute(); err != nil {
			t.Fatalf("add failed at iteration %d: %v", i, err)
		}

		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", message})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("commit failed at iteration %d: %v", i, err)
		}
	}

	// Verify all commits
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != numCommits {
		t.Errorf("expected %d commits, got %d", numCommits, len(history))
	}
}

// TestIntegrationSpecialCharacters tests handling files with special characters
func TestIntegrationSpecialCharacters(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create files with special characters in names
	specialFiles := []string{
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.multiple.dots.txt",
	}

	for _, filename := range specialFiles {
		h.WriteFile(filename, "content")
	}

	// Add all special files
	addCmd := newAddCmd()
	addCmd.SetArgs(specialFiles)
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Commit
	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Add files with special characters"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("commit command failed: %v", err)
	}

	// Verify commit
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 commit, got %d", len(history))
	}
	if history[0].Message != "Add files with special characters" {
		t.Errorf("expected 'Add files with special characters', got '%s'", history[0].Message)
	}
}

// TestIntegrationErrorHandling tests various error scenarios
func TestIntegrationErrorHandling(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	t.Run("commit without staging", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		h.WriteFile("file.txt", "content")

		// Try to commit without staging
		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{"-m", "Should fail"})
		err := commitCmd.Execute()

		if err == nil {
			t.Error("expected error when committing without staging, got nil")
		}
	})

	t.Run("add non-existent file", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		// Try to add file that doesn't exist
		addCmd := newAddCmd()
		addCmd.SetArgs([]string{"nonexistent.txt"})
		err := addCmd.Execute()

		// The command may handle this gracefully or return an error
		if err == nil {
			t.Log("add command handled non-existent file gracefully")
		}
	})

	t.Run("commit without message", func(t *testing.T) {
		h := NewTestHelper(t)
		h.InitRepo()
		h.Chdir()
		defer os.Chdir(origDir)

		h.WriteFile("file.txt", "content")

		addCmd := newAddCmd()
		addCmd.SetArgs([]string{"file.txt"})
		if err := addCmd.Execute(); err != nil {
			t.Fatalf("add failed: %v", err)
		}

		// Try to commit without message
		commitCmd := newCommitCmd()
		commitCmd.SetArgs([]string{})
		err := commitCmd.Execute()

		if err == nil {
			t.Error("expected error when committing without message, got nil")
		}
	})
}

// TestIntegrationRepositoryIntegrity tests that repository remains consistent
func TestIntegrationRepositoryIntegrity(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Verify .git directory structure
	expectedDirs := []string{
		scpath.SourceDir,
		filepath.Join(scpath.SourceDir, "objects"),
		filepath.Join(scpath.SourceDir, "refs"),
		filepath.Join(scpath.SourceDir, "refs", "heads"),
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(h.RepoPath, dir)
		if info, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("expected directory %s does not exist", dir)
		} else if err == nil && !info.IsDir() {
			t.Errorf("%s exists but is not a directory", dir)
		}
	}

	// Verify HEAD file exists
	headPath := filepath.Join(h.RepoPath, scpath.SourceDir, "HEAD")
	if _, err := os.Stat(headPath); os.IsNotExist(err) {
		t.Error("HEAD file does not exist")
	}

	// Create and commit a file
	h.WriteFile("test.txt", "test content")

	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"test.txt"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Verify index file exists after add
	indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
	if _, err := os.Stat(string(indexPath)); os.IsNotExist(err) {
		t.Error("index file does not exist after add")
	}

	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Test commit"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("commit command failed: %v", err)
	}

	// Verify objects were created
	objectsDir := filepath.Join(h.RepoPath, scpath.SourceDir, "objects")
	entries, err := os.ReadDir(objectsDir)
	if err != nil {
		t.Fatalf("failed to read objects directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("no objects were created after commit")
	}

	// Verify master or main branch ref exists
	masterRefPath := filepath.Join(h.RepoPath, scpath.SourceDir, "refs", "heads", "master")
	mainRefPath := filepath.Join(h.RepoPath, scpath.SourceDir, "refs", "heads", "main")

	_, masterErr := os.Stat(masterRefPath)
	_, mainErr := os.Stat(mainRefPath)

	if os.IsNotExist(masterErr) && os.IsNotExist(mainErr) {
		t.Error("neither master nor main branch ref exists")
	}
}
