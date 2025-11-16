package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/refs/branch"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// TestComplexWorkflow_FeatureBranchDevelopment simulates a realistic feature branch workflow
func TestComplexWorkflow_FeatureBranchDevelopment(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Step 1: Create initial commit on main branch
	h.WriteFile("README.md", "# Project")
	h.WriteFile("main.go", "package main\n\nfunc main() {}")

	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"README.md", "main.go"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("initial add failed: %v", err)
	}

	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Initial project setup"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("initial commit failed: %v", err)
	}

	// Step 2: Create feature branch
	branchCmd := newBranchCmd()
	branchCmd.SetArgs([]string{"feature/user-auth"})
	if err := branchCmd.Execute(); err != nil {
		t.Fatalf("failed to create feature branch: %v", err)
	}

	// Step 3: Make multiple commits on feature branch (simulated by adding more commits to current branch)
	featureFiles := map[string]string{
		"auth/login.go":      "package auth\n\nfunc Login() {}",
		"auth/register.go":   "package auth\n\nfunc Register() {}",
		"auth/middleware.go": "package auth\n\nfunc Middleware() {}",
	}

	for file, content := range featureFiles {
		h.WriteFile(file, content)

		addCmd = newAddCmd()
		addCmd.SetArgs([]string{file})
		if err := addCmd.Execute(); err != nil {
			t.Fatalf("failed to add %s: %v", file, err)
		}

		commitCmd = newCommitCmd()
		commitCmd.SetArgs([]string{"-m", fmt.Sprintf("Add %s", file)})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("failed to commit %s: %v", file, err)
		}
	}

	// Step 4: Verify all commits exist
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	expectedCommits := 4 // 1 initial + 3 feature commits
	if len(history) != expectedCommits {
		t.Errorf("expected %d commits, got %d", expectedCommits, len(history))
	}

	// Step 5: Verify branches exist
	branchMgr := branch.NewManager(repo)
	branches, err := branchMgr.ListBranches(ctx)
	if err != nil {
		t.Fatalf("failed to list branches: %v", err)
	}

	branchNames := make(map[string]bool)
	for _, b := range branches {
		branchNames[b.Name] = true
	}

	if !branchNames["feature/user-auth"] {
		t.Error("feature branch not found")
	}

	t.Logf("Successfully completed feature branch workflow with %d commits", len(history))
}

// TestComplexWorkflow_LargeRepository simulates a large repository with many files
func TestComplexWorkflow_LargeRepository(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create a complex directory structure with many files
	structure := map[string]string{
		"src/main.go":                    "package main",
		"src/app/app.go":                 "package app",
		"src/app/config.go":              "package app",
		"src/handlers/user.go":           "package handlers",
		"src/handlers/admin.go":          "package handlers",
		"src/models/user.go":             "package models",
		"src/models/session.go":          "package models",
		"src/db/connection.go":           "package db",
		"src/db/migrations/001_init.sql": "CREATE TABLE users;",
		"src/db/migrations/002_add.sql":  "ALTER TABLE users;",
		"tests/unit/user_test.go":        "package unit",
		"tests/integration/api_test.go":  "package integration",
		"docs/API.md":                    "# API Documentation",
		"docs/SETUP.md":                  "# Setup Guide",
		"config/dev.json":                `{"env": "dev"}`,
		"config/prod.json":               `{"env": "prod"}`,
		"scripts/build.sh":               "#!/bin/bash\ngo build",
		"scripts/test.sh":                "#!/bin/bash\ngo test",
		".gitignore":                     "*.log\n*.tmp",
		"Makefile":                       "all:\n\tgo build",
		"go.mod":                         "module myapp",
		"go.sum":                         "# dependencies",
		"README.md":                      "# My Application",
		"LICENSE":                        "MIT License",
	}

	// Create all files
	var allFiles []string
	for file, content := range structure {
		h.WriteFile(file, content)
		allFiles = append(allFiles, file)
	}

	// Add all files in one go
	addCmd := newAddCmd()
	addCmd.SetArgs(allFiles)
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("failed to add files: %v", err)
	}

	// Verify all files are staged
	indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}

	if idx.Count() != len(structure) {
		t.Errorf("expected %d files in index, got %d", len(structure), idx.Count())
	}

	// Commit all files
	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Initial project structure with all files"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Verify commit was created
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 commit, got %d", len(history))
	}

	// Verify objects were created
	objectsDir := filepath.Join(h.RepoPath, scpath.SourceDir, "objects")
	entries, err := os.ReadDir(objectsDir)
	if err != nil {
		t.Fatalf("failed to read objects directory: %v", err)
	}

	// Should have created blobs for each file + tree objects + commit
	if len(entries) == 0 {
		t.Error("no objects were created")
	}

	t.Logf("Successfully managed repository with %d files in %d directories", len(structure), len(entries))
}

// TestComplexWorkflow_IncrementalDevelopment simulates incremental development over time
func TestComplexWorkflow_IncrementalDevelopment(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Phase 1: Initial project
	h.WriteFile("main.go", "package main\n\nfunc main() {\n\tprintln(\"v1\")\n}")
	h.WriteFile("README.md", "# Version 1.0")

	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"main.go", "README.md"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("phase 1 add failed: %v", err)
	}

	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Release v1.0.0"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("phase 1 commit failed: %v", err)
	}

	// Phase 2: Add features
	h.WriteFile("config.go", "package main\n\ntype Config struct {}")
	h.WriteFile("utils.go", "package main\n\nfunc Helper() {}")

	addCmd = newAddCmd()
	addCmd.SetArgs([]string{"config.go", "utils.go"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("phase 2 add failed: %v", err)
	}

	commitCmd = newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Add configuration and utilities"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("phase 2 commit failed: %v", err)
	}

	// Phase 3: Update existing files
	h.WriteFile("main.go", "package main\n\nfunc main() {\n\tprintln(\"v2\")\n}")
	h.WriteFile("README.md", "# Version 2.0\n\nNew features added!")

	addCmd = newAddCmd()
	addCmd.SetArgs([]string{"main.go", "README.md"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("phase 3 add failed: %v", err)
	}

	commitCmd = newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Release v2.0.0 - Major update"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("phase 3 commit failed: %v", err)
	}

	// Phase 4: Add more features and fix bugs
	h.WriteFile("bugfix.go", "package main\n\n// Bug fix")
	addCmd = newAddCmd()
	addCmd.SetArgs([]string{"bugfix.go"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("phase 4 add failed: %v", err)
	}

	commitCmd = newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Fix critical bug in main logic"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("phase 4 commit failed: %v", err)
	}

	// Phase 5: Refactoring
	h.WriteFile("main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"v2.1\")\n}")
	h.WriteFile("utils.go", "package main\n\nfunc Helper() {\n\t// Improved implementation\n}")

	addCmd = newAddCmd()
	addCmd.SetArgs([]string{"main.go", "utils.go"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("phase 5 add failed: %v", err)
	}

	commitCmd = newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Refactor code for better maintainability"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("phase 5 commit failed: %v", err)
	}

	// Verify all 5 commits exist
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	expectedCommits := 5
	if len(history) != expectedCommits {
		t.Errorf("expected %d commits, got %d", expectedCommits, len(history))
	}

	// Verify commit messages are preserved in order
	expectedMessages := []string{
		"Refactor code for better maintainability",
		"Fix critical bug in main logic",
		"Release v2.0.0 - Major update",
		"Add configuration and utilities",
		"Release v1.0.0",
	}

	for i, expected := range expectedMessages {
		if i < len(history) {
			if history[i].Message != expected {
				t.Errorf("commit %d: expected '%s', got '%s'", i, expected, history[i].Message)
			}
		}
	}

	t.Logf("Successfully completed incremental development with %d phases", expectedCommits)
}

// TestComplexWorkflow_BinaryFiles tests handling of binary content
func TestComplexWorkflow_BinaryFiles(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create files with various binary-like content
	binaryFiles := map[string][]byte{
		"image.dat":   {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG header
		"data.bin":    {0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0xFC},
		"random.blob": make([]byte, 256), // Random binary data
	}

	// Fill random.blob with sequential bytes
	for i := range binaryFiles["random.blob"] {
		binaryFiles["random.blob"][i] = byte(i)
	}

	// Write binary files
	for filename, content := range binaryFiles {
		path := filepath.Join(h.RepoPath, filename)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	// Add binary files
	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"image.dat", "data.bin", "random.blob"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("failed to add binary files: %v", err)
	}

	// Commit binary files
	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Add binary assets"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("failed to commit binary files: %v", err)
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

	t.Log("Successfully handled binary files")
}

// TestComplexWorkflow_UnicodeAndSpecialContent tests handling of unicode and special characters
func TestComplexWorkflow_UnicodeAndSpecialContent(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create files with unicode and special content
	unicodeFiles := map[string]string{
		"unicode.txt":  "Hello 世界 🌍 مرحبا мир",
		"emoji.txt":    "🚀 🎉 💻 ✨ 🔥",
		"russian.txt":  "Привет мир",
		"chinese.txt":  "你好世界",
		"arabic.txt":   "مرحبا بالعالم",
		"japanese.txt": "こんにちは世界",
		"mixed.txt":    "English + 中文 + Русский + العربية + 日本語",
	}

	for filename, content := range unicodeFiles {
		h.WriteFile(filename, content)
	}

	// Add all unicode files
	var files []string
	for filename := range unicodeFiles {
		files = append(files, filename)
	}

	addCmd := newAddCmd()
	addCmd.SetArgs(files)
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("failed to add unicode files: %v", err)
	}

	// Commit
	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Add international content"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("failed to commit unicode files: %v", err)
	}

	// Verify
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 commit, got %d", len(history))
	}

	t.Log("Successfully handled unicode and international content")
}

// TestComplexWorkflow_FileLifecycle tests the complete lifecycle of files
func TestComplexWorkflow_FileLifecycle(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Stage 1: Create initial files
	h.WriteFile("keep.txt", "This file will stay")
	h.WriteFile("modify.txt", "Original content")
	h.WriteFile("delete.txt", "This will be deleted")

	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"keep.txt", "modify.txt", "delete.txt"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("stage 1 add failed: %v", err)
	}

	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Initial files"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("stage 1 commit failed: %v", err)
	}

	// Stage 2: Modify one file
	h.WriteFile("modify.txt", "Modified content - version 2")

	addCmd = newAddCmd()
	addCmd.SetArgs([]string{"modify.txt"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("stage 2 add failed: %v", err)
	}

	commitCmd = newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Update modify.txt"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("stage 2 commit failed: %v", err)
	}

	// Stage 3: Delete a file (simulate by removing and tracking deletion)
	if err := os.Remove(filepath.Join(h.RepoPath, "delete.txt")); err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	// Note: The system may or may not auto-track deletions, so we just verify the file is gone
	if _, err := os.Stat(filepath.Join(h.RepoPath, "delete.txt")); !os.IsNotExist(err) {
		t.Error("file should be deleted from working directory")
	}

	// Stage 4: Modify again
	h.WriteFile("modify.txt", "Modified content - version 3")

	addCmd = newAddCmd()
	addCmd.SetArgs([]string{"modify.txt"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("stage 4 add failed: %v", err)
	}

	commitCmd = newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Update modify.txt again"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("stage 4 commit failed: %v", err)
	}

	// Verify history
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	expectedCommits := 3
	if len(history) != expectedCommits {
		t.Errorf("expected %d commits, got %d", expectedCommits, len(history))
	}

	t.Log("Successfully tracked complete file lifecycle")
}

// TestComplexWorkflow_MultipleBranches tests working with multiple branches
func TestComplexWorkflow_MultipleBranches(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create initial commit
	h.WriteFile("base.txt", "Base content")

	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"base.txt"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("initial add failed: %v", err)
	}

	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Base commit"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("initial commit failed: %v", err)
	}

	// Create multiple branches with different purposes
	branches := []struct {
		name    string
		purpose string
	}{
		{"develop", "Development branch"},
		{"feature/login", "Login feature"},
		{"feature/signup", "Signup feature"},
		{"bugfix/auth-error", "Auth bug fix"},
		{"release/v1.0", "Release branch"},
		{"hotfix/security", "Security hotfix"},
	}

	for _, branch := range branches {
		branchCmd := newBranchCmd()
		branchCmd.SetArgs([]string{branch.name})
		if err := branchCmd.Execute(); err != nil {
			t.Fatalf("failed to create branch %s: %v", branch.name, err)
		}
	}

	// Verify all branches were created
	branchMgr := branch.NewManager(repo)
	ctx := context.Background()
	branchList, err := branchMgr.ListBranches(ctx)
	if err != nil {
		t.Fatalf("failed to list branches: %v", err)
	}

	branchNames := make(map[string]bool)
	for _, b := range branchList {
		branchNames[b.Name] = true
	}

	for _, branch := range branches {
		if !branchNames[branch.name] {
			t.Errorf("branch %s not found", branch.name)
		}
	}

	// Verify default branch exists
	if !branchNames["master"] && !branchNames["main"] {
		t.Error("default branch not found")
	}

	totalExpected := len(branches) + 1 // +1 for default branch
	if len(branchList) < totalExpected {
		t.Errorf("expected at least %d branches, got %d", totalExpected, len(branchList))
	}

	t.Logf("Successfully created and managed %d branches", len(branches))
}

// TestComplexWorkflow_DeepDirectoryStructure tests very deep directory nesting
func TestComplexWorkflow_DeepDirectoryStructure(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create deep directory structure
	deepPath := "level1/level2/level3/level4/level5/level6/level7/level8/level9/level10/deep.txt"
	h.WriteFile(deepPath, "Content in deeply nested file")

	// Also create files at various levels
	h.WriteFile("level1/file1.txt", "Level 1")
	h.WriteFile("level1/level2/file2.txt", "Level 2")
	h.WriteFile("level1/level2/level3/file3.txt", "Level 3")
	h.WriteFile("level1/level2/level3/level4/file4.txt", "Level 4")
	h.WriteFile("level1/level2/level3/level4/level5/file5.txt", "Level 5")

	// Add all files
	files := []string{
		deepPath,
		"level1/file1.txt",
		"level1/level2/file2.txt",
		"level1/level2/level3/file3.txt",
		"level1/level2/level3/level4/file4.txt",
		"level1/level2/level3/level4/level5/file5.txt",
	}

	addCmd := newAddCmd()
	addCmd.SetArgs(files)
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("failed to add deep files: %v", err)
	}

	// Commit
	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Add deep directory structure"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Verify
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 commit, got %d", len(history))
	}

	// Verify index has all files
	indexPath := repo.SourceDirectory().IndexPath().ToAbsolutePath()
	idx, err := index.Read(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}

	if idx.Count() != len(files) {
		t.Errorf("expected %d files in index, got %d", len(files), idx.Count())
	}

	t.Log("Successfully handled deep directory structure with 10+ levels")
}

// TestComplexWorkflow_MixedOperations tests a realistic mix of operations
func TestComplexWorkflow_MixedOperations(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	ctx := context.Background()
	mgr := commitmanager.NewManager(repo)

	// Operation 1: Initial setup
	h.WriteFile("main.go", "package main")
	h.WriteFile("README.md", "# Project")

	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"main.go", "README.md"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("op1: add failed: %v", err)
	}

	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Initial commit"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("op1: commit failed: %v", err)
	}

	// Operation 2: Create branch and add features
	branchCmd := newBranchCmd()
	branchCmd.SetArgs([]string{"develop"})
	if err := branchCmd.Execute(); err != nil {
		t.Fatalf("op2: branch creation failed: %v", err)
	}

	h.WriteFile("feature.go", "package main\n\nfunc Feature() {}")

	addCmd = newAddCmd()
	addCmd.SetArgs([]string{"feature.go"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("op2: add failed: %v", err)
	}

	commitCmd = newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Add feature"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("op2: commit failed: %v", err)
	}

	// Operation 3: Update multiple files
	h.WriteFile("main.go", "package main\n\nfunc main() {}")
	h.WriteFile("README.md", "# Project\n\nUpdated docs")

	addCmd = newAddCmd()
	addCmd.SetArgs([]string{"main.go", "README.md"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("op3: add failed: %v", err)
	}

	commitCmd = newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Update documentation and main"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("op3: commit failed: %v", err)
	}

	// Operation 4: Add nested structure
	h.WriteFile("pkg/utils/helper.go", "package utils")
	h.WriteFile("pkg/models/user.go", "package models")

	addCmd = newAddCmd()
	addCmd.SetArgs([]string{"pkg/utils/helper.go", "pkg/models/user.go"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("op4: add failed: %v", err)
	}

	commitCmd = newCommitCmd()
	commitCmd.SetArgs([]string{"-m", "Add package structure"})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("op4: commit failed: %v", err)
	}

	// Operation 5: Check status
	statusCmd := newStatusCmd()
	if err := statusCmd.Execute(); err != nil {
		t.Fatalf("op5: status failed: %v", err)
	}

	// Operation 6: Create more branches
	for _, name := range []string{"feature/auth", "feature/api"} {
		branchCmd = newBranchCmd()
		branchCmd.SetArgs([]string{name})
		if err := branchCmd.Execute(); err != nil {
			t.Fatalf("op6: branch %s creation failed: %v", name, err)
		}
	}

	// Verify final state
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	expectedCommits := 4
	if len(history) != expectedCommits {
		t.Errorf("expected %d commits, got %d", expectedCommits, len(history))
	}

	// Verify branches
	branchMgr := branch.NewManager(repo)
	branches, err := branchMgr.ListBranches(ctx)
	if err != nil {
		t.Fatalf("failed to list branches: %v", err)
	}

	if len(branches) < 3 { // develop, feature/auth, feature/api (+ default)
		t.Errorf("expected at least 3 branches, got %d", len(branches))
	}

	t.Logf("Successfully completed mixed operations: %d commits, %d branches", len(history), len(branches))
}

// TestComplexWorkflow_StressTest performs a stress test with many operations
func TestComplexWorkflow_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	h := NewTestHelper(t)
	repo := h.InitRepo()
	h.Chdir()
	defer os.Chdir(origDir)

	// Create 50 files
	numFiles := 50
	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("file_%03d.txt", i)
		content := fmt.Sprintf("Content for file %d\n%s", i, strings.Repeat("data ", 100))
		h.WriteFile(filename, content)
	}

	// Add all files at once
	var allFiles []string
	for i := 0; i < numFiles; i++ {
		allFiles = append(allFiles, fmt.Sprintf("file_%03d.txt", i))
	}

	addCmd := newAddCmd()
	addCmd.SetArgs(allFiles)
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("failed to add files: %v", err)
	}

	// Commit
	commitCmd := newCommitCmd()
	commitCmd.SetArgs([]string{"-m", fmt.Sprintf("Add %d files", numFiles)})
	if err := commitCmd.Execute(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Create 20 more commits by modifying files
	for i := 0; i < 20; i++ {
		// Modify a subset of files
		for j := 0; j < 5; j++ {
			fileIdx := (i*5 + j) % numFiles
			filename := fmt.Sprintf("file_%03d.txt", fileIdx)
			content := fmt.Sprintf("Updated content for file %d, iteration %d", fileIdx, i)
			h.WriteFile(filename, content)
		}

		// Add modified files
		var modifiedFiles []string
		for j := 0; j < 5; j++ {
			fileIdx := (i*5 + j) % numFiles
			modifiedFiles = append(modifiedFiles, fmt.Sprintf("file_%03d.txt", fileIdx))
		}

		addCmd = newAddCmd()
		addCmd.SetArgs(modifiedFiles)
		if err := addCmd.Execute(); err != nil {
			t.Fatalf("iteration %d: add failed: %v", i, err)
		}

		commitCmd = newCommitCmd()
		commitCmd.SetArgs([]string{"-m", fmt.Sprintf("Update iteration %d", i)})
		if err := commitCmd.Execute(); err != nil {
			t.Fatalf("iteration %d: commit failed: %v", i, err)
		}
	}

	// Verify all commits
	mgr := commitmanager.NewManager(repo)
	ctx := context.Background()
	history, err := mgr.GetHistory(ctx, objects.ObjectHash(""), 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	expectedCommits := 21 // 1 initial + 20 updates
	if len(history) != expectedCommits {
		t.Errorf("expected %d commits, got %d", expectedCommits, len(history))
	}

	// Verify repository integrity
	objectsDir := filepath.Join(h.RepoPath, scpath.SourceDir, "objects")
	entries, err := os.ReadDir(objectsDir)
	if err != nil {
		t.Fatalf("failed to read objects directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("no objects were created")
	}

	t.Logf("Stress test completed: %d files, %d commits, %d object directories", numFiles, len(history), len(entries))
}
