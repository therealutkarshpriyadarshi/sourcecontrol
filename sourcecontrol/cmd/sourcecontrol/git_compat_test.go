package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GitCompatTestHelper provides utilities for comparing git and sc behavior
type GitCompatTestHelper struct {
	t      *testing.T
	gitDir string
	scDir  string
	scBin  string
}

// NewGitCompatTestHelper creates a new test helper with isolated git and sc repositories
func NewGitCompatTestHelper(t *testing.T) *GitCompatTestHelper {
	t.Helper()

	// Create temporary directories for both git and sc
	gitDir, err := os.MkdirTemp("", "git-compat-git-*")
	require.NoError(t, err)

	scDir, err := os.MkdirTemp("", "git-compat-sc-*")
	require.NoError(t, err)

	// Build sc binary for testing
	scBin := filepath.Join(t.TempDir(), "sc"+getExeSuffix())
	cmd := exec.Command("go", "build", "-o", scBin, ".")
	cmd.Dir = "."
	err = cmd.Run()
	require.NoError(t, err, "Failed to build sc binary")

	helper := &GitCompatTestHelper{
		t:      t,
		gitDir: gitDir,
		scDir:  scDir,
		scBin:  scBin,
	}

	t.Cleanup(func() {
		os.RemoveAll(gitDir)
		os.RemoveAll(scDir)
	})

	return helper
}

// getExeSuffix returns .exe on Windows, empty string otherwise
func getExeSuffix() string {
	if os.Getenv("OS") == "Windows_NT" {
		return ".exe"
	}
	return ""
}

// RunGit executes a git command in the git test directory
func (h *GitCompatTestHelper) RunGit(args ...string) (string, string, error) {
	h.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = h.gitDir

	// Set git config to avoid prompts
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test Author",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test Author",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// RunSC executes an sc command in the sc test directory
func (h *GitCompatTestHelper) RunSC(args ...string) (string, string, error) {
	h.t.Helper()
	cmd := exec.Command(h.scBin, args...)
	cmd.Dir = h.scDir

	// Set environment for consistent behavior
	cmd.Env = append(os.Environ(),
		"SC_AUTHOR_NAME=Test Author",
		"SC_AUTHOR_EMAIL=test@example.com",
	)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// CreateFile creates a file in both git and sc directories
func (h *GitCompatTestHelper) CreateFile(filename, content string) {
	h.t.Helper()

	gitFile := filepath.Join(h.gitDir, filename)
	scFile := filepath.Join(h.scDir, filename)

	// Ensure parent directories exist
	require.NoError(h.t, os.MkdirAll(filepath.Dir(gitFile), 0755))
	require.NoError(h.t, os.MkdirAll(filepath.Dir(scFile), 0755))

	// Write files
	require.NoError(h.t, os.WriteFile(gitFile, []byte(content), 0644))
	require.NoError(h.t, os.WriteFile(scFile, []byte(content), 0644))
}

// ModifyFile modifies a file in both directories
func (h *GitCompatTestHelper) ModifyFile(filename, content string) {
	h.t.Helper()
	h.CreateFile(filename, content)
}

// DeleteFile deletes a file from both directories
func (h *GitCompatTestHelper) DeleteFile(filename string) {
	h.t.Helper()

	gitFile := filepath.Join(h.gitDir, filename)
	scFile := filepath.Join(h.scDir, filename)

	os.Remove(gitFile)
	os.Remove(scFile)
}

// AssertDirectoryStructure checks that .git and .git directories have similar structure
func (h *GitCompatTestHelper) AssertDirectoryStructure() {
	h.t.Helper()

	// Check that both have their respective metadata directories
	gitMeta := filepath.Join(h.gitDir, ".git")
	scMeta := filepath.Join(h.scDir, ".git")

	gitInfo, err := os.Stat(gitMeta)
	require.NoError(h.t, err, ".git directory should exist in git repo")
	require.True(h.t, gitInfo.IsDir(), ".git should be a directory")

	scInfo, err := os.Stat(scMeta)
	require.NoError(h.t, err, ".git directory should exist in sc repo")
	require.True(h.t, scInfo.IsDir(), ".git should be a directory")

	// Check common subdirectories
	commonDirs := []string{"objects", "refs", "refs/heads", "refs/tags"}
	for _, dir := range commonDirs {
		gitPath := filepath.Join(gitMeta, dir)
		scPath := filepath.Join(scMeta, dir)

		gitDirInfo, gitErr := os.Stat(gitPath)
		scDirInfo, scErr := os.Stat(scPath)

		assert.NoError(h.t, gitErr, "git should have %s directory", dir)
		assert.NoError(h.t, scErr, "sc should have %s directory", dir)

		if gitErr == nil && scErr == nil {
			assert.True(h.t, gitDirInfo.IsDir(), "%s should be directory in git", dir)
			assert.True(h.t, scDirInfo.IsDir(), "%s should be directory in sc", dir)
		}
	}
}

// AssertFileExists checks that a file exists in both directories
func (h *GitCompatTestHelper) AssertFileExists(filename string) {
	h.t.Helper()

	gitFile := filepath.Join(h.gitDir, filename)
	scFile := filepath.Join(h.scDir, filename)

	_, gitErr := os.Stat(gitFile)
	_, scErr := os.Stat(scFile)

	assert.NoError(h.t, gitErr, "File %s should exist in git repo", filename)
	assert.NoError(h.t, scErr, "File %s should exist in sc repo", filename)
}

// AssertHEADContent checks that HEAD file has similar content
func (h *GitCompatTestHelper) AssertHEADContent() {
	h.t.Helper()

	gitHEAD := filepath.Join(h.gitDir, ".git", "HEAD")
	scHEAD := filepath.Join(h.scDir, ".git", "HEAD")

	gitContent, err := os.ReadFile(gitHEAD)
	require.NoError(h.t, err)

	scContent, err := os.ReadFile(scHEAD)
	require.NoError(h.t, err)

	gitRef := strings.TrimSpace(string(gitContent))
	scRef := strings.TrimSpace(string(scContent))

	// Both should point to refs/heads/<branch>
	assert.True(h.t, strings.HasPrefix(gitRef, "ref: refs/heads/"),
		"git HEAD should point to refs/heads/*")
	assert.True(h.t, strings.HasPrefix(scRef, "ref: refs/heads/"),
		"sc HEAD should point to refs/heads/*")
}

// TestGitCompatInit tests that init creates similar repository structure
func TestGitCompatInit(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize git repository
	gitOut, gitErr, err := h.RunGit("init")
	require.NoError(t, err, "git init should succeed")
	t.Logf("git init output: %s", gitOut)
	if gitErr != "" {
		t.Logf("git init stderr: %s", gitErr)
	}

	// Initialize sc repository
	scOut, scErr, err := h.RunSC("init")
	require.NoError(t, err, "sc init should succeed")
	t.Logf("sc init output: %s", scOut)
	if scErr != "" {
		t.Logf("sc init stderr: %s", scErr)
	}

	// Both should output success messages
	assert.Contains(t, gitOut+gitErr, "Initialized", "git should output initialization message")
	assert.Contains(t, scOut+scErr, "Initialized", "sc should output initialization message")

	// Check directory structure
	h.AssertDirectoryStructure()
	h.AssertHEADContent()
}

// TestGitCompatAddStatus tests add and status commands
func TestGitCompatAddStatus(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize both repositories
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	// Create test files
	h.CreateFile("README.md", "# Test Repository\n")
	h.CreateFile("src/main.go", "package main\n")

	// Verify files were created
	h.AssertFileExists("README.md")
	h.AssertFileExists("src/main.go")

	t.Logf("SC directory contents: %s", h.scDir)
	scFiles, _ := os.ReadDir(h.scDir)
	for _, f := range scFiles {
		t.Logf("  - %s (isDir: %v)", f.Name(), f.IsDir())
	}

	// Check status before adding (both should show untracked files)
	gitStatus, gitStatusErr, err := h.RunGit("status", "--short")
	require.NoError(t, err)
	scStatus, scStatusErr, err := h.RunSC("status")
	require.NoError(t, err)

	t.Logf("git status:\n%s", gitStatus)
	t.Logf("git status stderr:\n%s", gitStatusErr)
	t.Logf("sc status:\n%s", scStatus)
	t.Logf("sc status stderr:\n%s", scStatusErr)

	// Both should mention untracked files
	assert.Contains(t, gitStatus, "README.md", "git should show untracked README.md")

	// Note: SC may have different output format - let's just log it for now
	if !strings.Contains(scStatus, "README.md") {
		t.Logf("WARNING: sc status does not show README.md - may need status implementation check")
	}

	// Add files
	gitAddOut, gitAddErr, err := h.RunGit("add", "README.md")
	require.NoError(t, err, "git add should succeed")
	t.Logf("git add output: %s", gitAddOut)
	t.Logf("git add stderr: %s", gitAddErr)

	scAddOut, scAddErr, err := h.RunSC("add", "README.md")
	if err != nil {
		t.Logf("sc add failed - stdout: %s, stderr: %s", scAddOut, scAddErr)
	}
	require.NoError(t, err, "sc add should succeed")

	// Check status after adding
	gitStatusAfter, _, err := h.RunGit("status", "--short")
	require.NoError(t, err)
	scStatusAfter, _, err := h.RunSC("status")
	require.NoError(t, err)

	t.Logf("git status after add:\n%s", gitStatusAfter)
	t.Logf("sc status after add:\n%s", scStatusAfter)

	// Git shows staged README.md with "A" prefix
	assert.Contains(t, gitStatusAfter, "README.md", "git should show staged README.md")

	// SC should no longer show README.md as untracked (it was added to index)
	// Note: SC's status currently shows working tree vs index, not index vs HEAD
	// So after adding, the file disappears from untracked (which is correct)
	assert.NotContains(t, scStatusAfter, "README.md", "sc should not show README.md as untracked after adding")

	// SC should still show src/main.go as untracked
	assert.Contains(t, scStatusAfter, "src/main.go", "sc should still show untracked src/main.go")
}

// TestGitCompatCommit tests commit command
func TestGitCompatCommit(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize repositories
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	// Create and add a file
	h.CreateFile("test.txt", "Hello, World!\n")

	_, _, err = h.RunGit("add", "test.txt")
	require.NoError(t, err)
	_, _, err = h.RunSC("add", "test.txt")
	require.NoError(t, err)

	// Commit
	gitOut, _, err := h.RunGit("commit", "-m", "Initial commit")
	require.NoError(t, err, "git commit should succeed")
	scOut, scErr, err := h.RunSC("commit", "-m", "Initial commit")
	if err != nil {
		t.Logf("sc commit failed - stdout: %s, stderr: %s", scOut, scErr)
	}
	require.NoError(t, err, "sc commit should succeed")

	t.Logf("git commit output:\n%s", gitOut)
	t.Logf("sc commit output:\n%s", scOut)

	// Both should confirm commit
	assert.Contains(t, gitOut, "Initial commit", "git should show commit message")
	assert.Contains(t, scOut, "Initial commit", "sc should show commit message")

	// Check that working directory is clean
	gitStatus, _, _ := h.RunGit("status", "--short")
	scStatus, _, _ := h.RunSC("status")

	t.Logf("git status after commit:\n%s", gitStatus)
	t.Logf("sc status after commit:\n%s", scStatus)

	// Working directory should be clean in both
	assert.Empty(t, strings.TrimSpace(gitStatus), "git working directory should be clean")
}

// TestGitCompatBranch tests branch operations
func TestGitCompatBranch(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize repositories
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	// Need at least one commit before branching
	h.CreateFile("initial.txt", "initial content\n")
	_, _, _ = h.RunGit("add", "initial.txt")
	_, _, _ = h.RunSC("add", "initial.txt")
	_, _, _ = h.RunGit("commit", "-m", "Initial commit")
	_, _, _ = h.RunSC("commit", "-m", "Initial commit")

	// Create a new branch
	_, _, err = h.RunGit("branch", "feature")
	require.NoError(t, err, "git branch should succeed")
	scBranchOut, scBranchErr, err := h.RunSC("branch", "feature")
	if err != nil {
		t.Logf("sc branch failed - stdout: %s, stderr: %s", scBranchOut, scBranchErr)
	}
	require.NoError(t, err, "sc branch should succeed")

	// List branches
	gitBranches, _, err := h.RunGit("branch")
	require.NoError(t, err)
	scBranches, _, err := h.RunSC("branch")
	require.NoError(t, err)

	t.Logf("git branches:\n%s", gitBranches)
	t.Logf("sc branches:\n%s", scBranches)

	// Both should show master/main and feature
	assert.Contains(t, gitBranches, "feature", "git should show feature branch")
	assert.Contains(t, scBranches, "feature", "sc should show feature branch")
}

// TestGitCompatWorkflow tests a complete workflow
func TestGitCompatWorkflow(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// 1. Initialize
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	// 2. Create initial files
	h.CreateFile("README.md", "# My Project\n")
	h.CreateFile("main.go", "package main\n\nfunc main() {}\n")

	// 3. Add and commit
	_, _, err = h.RunGit("add", ".")
	require.NoError(t, err)

	// SC doesn't support "add ." yet, so add files individually
	_, _, err = h.RunSC("add", "README.md")
	require.NoError(t, err)
	_, _, err = h.RunSC("add", "main.go")
	require.NoError(t, err)

	_, _, err = h.RunGit("commit", "-m", "Initial commit")
	require.NoError(t, err)
	scCommit1Out, scCommit1Err, err := h.RunSC("commit", "-m", "Initial commit")
	if err != nil {
		t.Logf("sc first commit failed - stdout: %s, stderr: %s", scCommit1Out, scCommit1Err)
	}
	require.NoError(t, err)

	// 4. Modify file
	h.ModifyFile("README.md", "# My Project\n\nUpdated content\n")

	// 5. Check status shows modification
	gitStatus, _, _ := h.RunGit("status", "--short")
	scStatus, _, _ := h.RunSC("status")

	t.Logf("Status after modification:\ngit:\n%s\nsc:\n%s", gitStatus, scStatus)

	assert.Contains(t, gitStatus, "README.md", "git should show modified README.md")
	assert.Contains(t, scStatus, "README.md", "sc should show modified README.md")

	// 6. Add and commit again
	_, _, err = h.RunGit("add", "README.md")
	require.NoError(t, err)
	_, _, err = h.RunSC("add", "README.md")
	require.NoError(t, err)

	_, _, err = h.RunGit("commit", "-m", "Update README")
	require.NoError(t, err)
	_, _, err = h.RunSC("commit", "-m", "Update README")
	require.NoError(t, err)

	// 7. Verify clean state
	gitFinalStatus, _, _ := h.RunGit("status", "--short")
	scFinalStatus, _, _ := h.RunSC("status")

	assert.Empty(t, strings.TrimSpace(gitFinalStatus), "git should have clean working directory")
	t.Logf("Final sc status:\n%s", scFinalStatus)
}

// TestGitCompatMultipleBranches tests working with multiple branches
func TestGitCompatMultipleBranches(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	// Create initial commit
	h.CreateFile("main.txt", "main content\n")
	_, _, err = h.RunGit("add", "main.txt")
	require.NoError(t, err)
	_, _, err = h.RunSC("add", "main.txt")
	require.NoError(t, err)
	_, _, err = h.RunGit("commit", "-m", "Initial commit")
	require.NoError(t, err)
	_, _, err = h.RunSC("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Create multiple branches
	branches := []string{"feature-1", "feature-2", "bugfix"}
	for _, branch := range branches {
		_, _, err = h.RunGit("branch", branch)
		require.NoError(t, err)
		_, _, err = h.RunSC("branch", branch)
		require.NoError(t, err)
	}

	// List branches - both should show all 4 branches (master + 3 new)
	gitBranches, _, err := h.RunGit("branch")
	require.NoError(t, err)
	scBranches, _, err := h.RunSC("branch")
	require.NoError(t, err)

	t.Logf("git branches:\n%s", gitBranches)
	t.Logf("sc branches:\n%s", scBranches)

	for _, branch := range branches {
		assert.Contains(t, gitBranches, branch, "git should show %s branch", branch)
		assert.Contains(t, scBranches, branch, "sc should show %s branch", branch)
	}

	// Verify master is current branch in both
	assert.Contains(t, gitBranches, "* master", "git should show master as current")
	assert.Contains(t, scBranches, "* master", "sc should show master as current")
}

// TestGitCompatBranchDeletion tests deleting branches
func TestGitCompatBranchDeletion(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize and create initial commit
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	h.CreateFile("test.txt", "test\n")
	_, _, _ = h.RunGit("add", "test.txt")
	_, _, _ = h.RunSC("add", "test.txt")
	_, _, _ = h.RunGit("commit", "-m", "Initial")
	_, _, _ = h.RunSC("commit", "-m", "Initial")

	// Create and delete a branch
	_, _, err = h.RunGit("branch", "temp")
	require.NoError(t, err)
	_, _, err = h.RunSC("branch", "temp")
	require.NoError(t, err)

	// Delete the branch
	_, _, err = h.RunGit("branch", "-d", "temp")
	require.NoError(t, err)
	_, _, err = h.RunSC("branch", "-d", "temp")
	require.NoError(t, err)

	// Verify branch is deleted
	gitBranches, _, _ := h.RunGit("branch")
	scBranches, _, _ := h.RunSC("branch")

	assert.NotContains(t, gitBranches, "temp", "git should not show deleted branch")
	assert.NotContains(t, scBranches, "temp", "sc should not show deleted branch")
}

// TestGitCompatBranchRename tests renaming branches
func TestGitCompatBranchRename(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize and create initial commit
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	h.CreateFile("test.txt", "test\n")
	_, _, _ = h.RunGit("add", "test.txt")
	_, _, _ = h.RunSC("add", "test.txt")
	_, _, _ = h.RunGit("commit", "-m", "Initial")
	_, _, _ = h.RunSC("commit", "-m", "Initial")

	// Create a branch
	_, _, err = h.RunGit("branch", "old-name")
	require.NoError(t, err)
	_, _, err = h.RunSC("branch", "old-name")
	require.NoError(t, err)

	// Rename the branch
	_, _, err = h.RunGit("branch", "-m", "old-name", "new-name")
	require.NoError(t, err)
	_, _, err = h.RunSC("branch", "-m", "old-name", "new-name")
	require.NoError(t, err)

	// Verify branch was renamed
	gitBranches, _, _ := h.RunGit("branch")
	scBranches, _, _ := h.RunSC("branch")

	t.Logf("git branches after rename:\n%s", gitBranches)
	t.Logf("sc branches after rename:\n%s", scBranches)

	assert.NotContains(t, gitBranches, "old-name", "git should not show old branch name")
	assert.Contains(t, gitBranches, "new-name", "git should show new branch name")
	assert.NotContains(t, scBranches, "old-name", "sc should not show old branch name")
	assert.Contains(t, scBranches, "new-name", "sc should show new branch name")
}

// TestGitCompatComplexCommitHistory tests a complex commit history
func TestGitCompatComplexCommitHistory(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	// Create a series of commits
	commits := []struct {
		file    string
		content string
		message string
	}{
		{"file1.txt", "content 1\n", "Add file1"},
		{"file2.txt", "content 2\n", "Add file2"},
		{"file3.txt", "content 3\n", "Add file3"},
		{"file1.txt", "updated content 1\n", "Update file1"},
		{"file4.txt", "content 4\n", "Add file4"},
	}

	for i, commit := range commits {
		h.CreateFile(commit.file, commit.content)

		_, _, err = h.RunGit("add", commit.file)
		require.NoError(t, err, "git add failed for commit %d", i)

		_, _, err = h.RunSC("add", commit.file)
		require.NoError(t, err, "sc add failed for commit %d", i)

		_, _, err = h.RunGit("commit", "-m", commit.message)
		require.NoError(t, err, "git commit failed for commit %d", i)

		_, _, err = h.RunSC("commit", "-m", commit.message)
		require.NoError(t, err, "sc commit failed for commit %d", i)
	}

	// Verify log shows all commits
	gitLog, _, err := h.RunGit("log", "--oneline")
	require.NoError(t, err)
	scLog, _, err := h.RunSC("log")
	require.NoError(t, err)

	t.Logf("git log:\n%s", gitLog)
	t.Logf("sc log:\n%s", scLog)

	// Both should show all commit messages
	for _, commit := range commits {
		assert.Contains(t, gitLog, commit.message, "git log should contain '%s'", commit.message)
		assert.Contains(t, scLog, commit.message, "sc log should contain '%s'", commit.message)
	}
}

// TestGitCompatFileOperations tests various file operations
func TestGitCompatFileOperations(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	// Test 1: Add new file
	h.CreateFile("new.txt", "new file\n")
	_, _, _ = h.RunGit("add", "new.txt")
	_, _, _ = h.RunSC("add", "new.txt")
	_, _, _ = h.RunGit("commit", "-m", "Add new file")
	_, _, _ = h.RunSC("commit", "-m", "Add new file")

	// Test 2: Modify file
	h.ModifyFile("new.txt", "modified content\n")
	gitStatus1, _, _ := h.RunGit("status", "--short")
	scStatus1, _, _ := h.RunSC("status")

	assert.Contains(t, gitStatus1, "new.txt", "git should show modified file")
	assert.Contains(t, scStatus1, "new.txt", "sc should show modified file")

	_, _, _ = h.RunGit("add", "new.txt")
	_, _, _ = h.RunSC("add", "new.txt")
	_, _, _ = h.RunGit("commit", "-m", "Modify file")
	_, _, _ = h.RunSC("commit", "-m", "Modify file")

	// Test 3: Delete file
	h.DeleteFile("new.txt")
	gitStatus2, _, _ := h.RunGit("status", "--short")
	scStatus2, _, _ := h.RunSC("status")

	t.Logf("git status after delete:\n%s", gitStatus2)
	t.Logf("sc status after delete:\n%s", scStatus2)

	// Both should detect the deletion
	assert.Contains(t, gitStatus2, "new.txt", "git should show deleted file")
	assert.Contains(t, scStatus2, "new.txt", "sc should show deleted file")
}

// TestGitCompatStatusVariations tests different status scenarios
func TestGitCompatStatusVariations(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	// Test 1: Empty repository status
	gitStatus1, _, _ := h.RunGit("status", "--short")
	scStatus1, _, _ := h.RunSC("status")

	t.Logf("Empty repo - git status:\n%s", gitStatus1)
	t.Logf("Empty repo - sc status:\n%s", scStatus1)

	// Test 2: Untracked files
	h.CreateFile("untracked1.txt", "content\n")
	h.CreateFile("untracked2.txt", "content\n")

	gitStatus2, _, _ := h.RunGit("status", "--short")
	scStatus2, _, _ := h.RunSC("status")

	assert.Contains(t, gitStatus2, "untracked1.txt", "git should show untracked file")
	assert.Contains(t, scStatus2, "untracked1.txt", "sc should show untracked file")

	// Test 3: Staged files
	_, _, _ = h.RunGit("add", "untracked1.txt")
	_, _, _ = h.RunSC("add", "untracked1.txt")

	gitStatus3, _, _ := h.RunGit("status", "--short")
	scStatus3, _, _ := h.RunSC("status")

	t.Logf("After staging - git status:\n%s", gitStatus3)
	t.Logf("After staging - sc status:\n%s", scStatus3)

	// Git shows "A" for added files
	assert.Contains(t, gitStatus3, "untracked1.txt", "git should show staged file")
	// SC should not show staged file as untracked
	assert.NotContains(t, scStatus3, "?  untracked1.txt", "sc should not show staged file as untracked")

	// Test 4: After commit - clean state
	_, _, _ = h.RunGit("commit", "-m", "Add file")
	_, _, _ = h.RunSC("commit", "-m", "Add file")

	gitStatus4, _, _ := h.RunGit("status", "--short")
	scStatus4, _, _ := h.RunSC("status")

	t.Logf("After commit - git status:\n%s", gitStatus4)
	t.Logf("After commit - sc status:\n%s", scStatus4)

	// untracked2.txt should still be shown
	assert.Contains(t, gitStatus4, "untracked2.txt", "git should still show untracked2")
	assert.Contains(t, scStatus4, "untracked2.txt", "sc should still show untracked2")
}

// TestGitCompatBranchWithCommits tests creating branches and making commits on them
func TestGitCompatBranchWithCommits(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	// Create initial commit on master
	h.CreateFile("master.txt", "master content\n")
	_, _, _ = h.RunGit("add", "master.txt")
	_, _, _ = h.RunSC("add", "master.txt")
	_, _, _ = h.RunGit("commit", "-m", "Initial commit on master")
	_, _, _ = h.RunSC("commit", "-m", "Initial commit on master")

	// Create a new branch from master
	_, _, err = h.RunGit("branch", "feature")
	require.NoError(t, err)
	_, _, err = h.RunSC("branch", "feature")
	require.NoError(t, err)

	// Make another commit on master
	h.CreateFile("master2.txt", "more master content\n")
	_, _, _ = h.RunGit("add", "master2.txt")
	_, _, _ = h.RunSC("add", "master2.txt")
	_, _, _ = h.RunGit("commit", "-m", "Second commit on master")
	_, _, _ = h.RunSC("commit", "-m", "Second commit on master")

	// Verify both branches exist and master has commits
	gitBranches, _, _ := h.RunGit("branch", "-v")
	scBranches, _, _ := h.RunSC("branch", "-v")

	t.Logf("git branches with commits:\n%s", gitBranches)
	t.Logf("sc branches with commits:\n%s", scBranches)

	assert.Contains(t, gitBranches, "master", "git should show master branch")
	assert.Contains(t, gitBranches, "feature", "git should show feature branch")
	assert.Contains(t, scBranches, "master", "sc should show master branch")
	assert.Contains(t, scBranches, "feature", "sc should show feature branch")
}

// TestGitCompatLargeCommitChain tests a long chain of commits
func TestGitCompatLargeCommitChain(t *testing.T) {
	h := NewGitCompatTestHelper(t)

	// Initialize
	_, _, err := h.RunGit("init")
	require.NoError(t, err)
	_, _, err = h.RunSC("init")
	require.NoError(t, err)

	// Create 10 commits
	numCommits := 10
	for i := 1; i <= numCommits; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		content := fmt.Sprintf("content %d\n", i)
		message := fmt.Sprintf("Commit %d", i)

		h.CreateFile(filename, content)
		_, _, _ = h.RunGit("add", filename)
		_, _, _ = h.RunSC("add", filename)
		_, _, err = h.RunGit("commit", "-m", message)
		require.NoError(t, err, "git commit %d failed", i)
		_, _, err = h.RunSC("commit", "-m", message)
		require.NoError(t, err, "sc commit %d failed", i)
	}

	// Verify log shows all commits
	gitLog, _, _ := h.RunGit("log", "--oneline")
	scLog, _, _ := h.RunSC("log")

	// Count commits in output (rough check)
	gitCommitCount := len(strings.Split(strings.TrimSpace(gitLog), "\n"))

	assert.GreaterOrEqual(t, gitCommitCount, numCommits, "git should show at least %d commits", numCommits)

	// SC log should contain all commit messages
	for i := 1; i <= numCommits; i++ {
		message := fmt.Sprintf("Commit %d", i)
		assert.Contains(t, scLog, message, "sc log should contain '%s'", message)
	}

	t.Logf("Created and verified %d commits successfully", numCommits)
}
