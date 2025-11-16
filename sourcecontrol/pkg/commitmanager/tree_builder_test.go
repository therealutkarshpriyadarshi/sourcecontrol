package commitmanager

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/common"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/blob"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

func TestNewTreeBuilder(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	tb := NewTreeBuilder(repo)
	if tb == nil {
		t.Fatal("NewTreeBuilder returned nil")
	}
	if tb.repo != repo {
		t.Error("TreeBuilder repo not set correctly")
	}
}

func TestBuildFromIndex_EmptyIndex(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	tb := NewTreeBuilder(repo)
	idx := index.NewIndex()
	ctx := context.Background()

	treeSHA, err := tb.BuildFromIndex(ctx, idx)
	if err != nil {
		t.Fatalf("BuildFromIndex failed: %v", err)
	}

	if treeSHA == "" {
		t.Error("Expected tree SHA for empty index, got empty string")
	}

	// Verify the tree is actually empty
	treeObj, err := repo.ReadTreeObject(treeSHA)
	if err != nil {
		t.Fatalf("Failed to read tree object: %v", err)
	}

	if len(treeObj.Entries()) != 0 {
		t.Errorf("Expected empty tree, got %d entries", len(treeObj.Entries()))
	}
}

func TestBuildFromIndex_SingleFile(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	tb := NewTreeBuilder(repo)
	ctx := context.Background()

	// Create a blob
	content := []byte("Hello, World!")
	b := blob.NewBlob(content)
	blobSHA, err := repo.WriteObject(b)
	if err != nil {
		t.Fatalf("Failed to write blob: %v", err)
	}

	// Create index with single file
	idx := index.NewIndex()
	entry := index.NewEntry(scpath.RelativePath("test.txt"))
	entry.BlobHash = blobSHA
	entry.Mode = objects.FileModeRegular
	entry.SizeInBytes = uint32(len(content))
	entry.ModificationTime = common.NewTimestampFromTime(time.Now())
	idx.Add(entry)

	// Build tree
	treeSHA, err := tb.BuildFromIndex(ctx, idx)
	if err != nil {
		t.Fatalf("BuildFromIndex failed: %v", err)
	}

	// Verify tree
	treeObj, err := repo.ReadTreeObject(treeSHA)
	if err != nil {
		t.Fatalf("Failed to read tree object: %v", err)
	}

	if len(treeObj.Entries()) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(treeObj.Entries()))
	}

	entry0 := treeObj.Entries()[0]
	if entry0.Name().String() != "test.txt" {
		t.Errorf("Expected entry name 'test.txt', got '%s'", entry0.Name())
	}
	if entry0.SHA() != blobSHA {
		t.Error("Entry SHA doesn't match blob SHA")
	}
}

func TestBuildFromIndex_MultipleFilesInRoot(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	tb := NewTreeBuilder(repo)
	ctx := context.Background()

	// Create multiple blobs
	files := map[string]string{
		"README.md":  "# Project",
		"main.go":    "package main",
		"config.yml": "version: 1",
	}

	idx := index.NewIndex()
	for filename, content := range files {
		b := blob.NewBlob([]byte(content))
		blobSHA, err := repo.WriteObject(b)
		if err != nil {
			t.Fatalf("Failed to write blob for %s: %v", filename, err)
		}

		entry := index.NewEntry(scpath.RelativePath(filename))
		entry.BlobHash = blobSHA
		entry.Mode = objects.FileModeRegular
		entry.SizeInBytes = uint32(len(content))
		entry.ModificationTime = common.NewTimestampFromTime(time.Now())
		idx.Add(entry)
	}

	// Build tree
	treeSHA, err := tb.BuildFromIndex(ctx, idx)
	if err != nil {
		t.Fatalf("BuildFromIndex failed: %v", err)
	}

	// Verify tree
	treeObj, err := repo.ReadTreeObject(treeSHA)
	if err != nil {
		t.Fatalf("Failed to read tree object: %v", err)
	}

	if len(treeObj.Entries()) != len(files) {
		t.Errorf("Expected %d entries, got %d", len(files), len(treeObj.Entries()))
	}

	// Verify all files are present
	entryNames := make(map[string]bool)
	for _, e := range treeObj.Entries() {
		entryNames[e.Name().String()] = true
	}

	for filename := range files {
		if !entryNames[filename] {
			t.Errorf("File %s not found in tree", filename)
		}
	}
}

func TestBuildFromIndex_FilesInSubdirectory(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	tb := NewTreeBuilder(repo)
	ctx := context.Background()

	// Create files in subdirectory
	files := map[string]string{
		"README.md":    "# Project",
		"src/main.go":  "package main",
		"src/utils.go": "package main",
	}

	idx := index.NewIndex()
	for filename, content := range files {
		b := blob.NewBlob([]byte(content))
		blobSHA, err := repo.WriteObject(b)
		if err != nil {
			t.Fatalf("Failed to write blob for %s: %v", filename, err)
		}

		entry := index.NewEntry(scpath.RelativePath(filename))
		entry.BlobHash = blobSHA
		entry.Mode = objects.FileModeRegular
		entry.SizeInBytes = uint32(len(content))
		entry.ModificationTime = common.NewTimestampFromTime(time.Now())
		idx.Add(entry)
	}

	// Build tree
	treeSHA, err := tb.BuildFromIndex(ctx, idx)
	if err != nil {
		t.Fatalf("BuildFromIndex failed: %v", err)
	}

	// Verify root tree
	treeObj, err := repo.ReadTreeObject(treeSHA)
	if err != nil {
		t.Fatalf("Failed to read tree object: %v", err)
	}

	if len(treeObj.Entries()) != 2 {
		t.Errorf("Expected 2 entries in root (README.md + src/), got %d", len(treeObj.Entries()))
	}

	// Find src directory entry
	var srcEntry *tree.TreeEntry
	for _, e := range treeObj.Entries() {
		if e.Name().String() == "src" && e.IsDirectory() {
			srcEntry = e
			break
		}
	}

	if srcEntry == nil {
		t.Fatal("src directory not found in root tree")
	}

	// Verify src subdirectory
	srcTree, err := repo.ReadTreeObject(srcEntry.SHA())
	if err != nil {
		t.Fatalf("Failed to read src tree: %v", err)
	}

	if len(srcTree.Entries()) != 2 {
		t.Errorf("Expected 2 files in src directory, got %d", len(srcTree.Entries()))
	}

	// Verify files in src
	srcFiles := make(map[string]bool)
	for _, e := range srcTree.Entries() {
		srcFiles[e.Name().String()] = true
	}

	if !srcFiles["main.go"] {
		t.Error("main.go not found in src directory")
	}
	if !srcFiles["utils.go"] {
		t.Error("utils.go not found in src directory")
	}
}

func TestBuildFromIndex_NestedSubdirectories(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	tb := NewTreeBuilder(repo)
	ctx := context.Background()

	// Create files in nested directories
	files := map[string]string{
		"README.md":                 "# Project",
		"src/main.go":               "package main",
		"src/utils/helper.go":       "package utils",
		"src/utils/types/models.go": "package types",
		"docs/guide.md":             "# Guide",
	}

	idx := index.NewIndex()
	for filename, content := range files {
		b := blob.NewBlob([]byte(content))
		blobSHA, err := repo.WriteObject(b)
		if err != nil {
			t.Fatalf("Failed to write blob for %s: %v", filename, err)
		}

		entry := index.NewEntry(scpath.RelativePath(filename))
		entry.BlobHash = blobSHA
		entry.Mode = objects.FileModeRegular
		entry.SizeInBytes = uint32(len(content))
		entry.ModificationTime = common.NewTimestampFromTime(time.Now())
		idx.Add(entry)
	}

	// Build tree
	treeSHA, err := tb.BuildFromIndex(ctx, idx)
	if err != nil {
		t.Fatalf("BuildFromIndex failed: %v", err)
	}

	// Verify root tree
	rootTree, err := repo.ReadTreeObject(treeSHA)
	if err != nil {
		t.Fatalf("Failed to read root tree: %v", err)
	}

	if len(rootTree.Entries()) != 3 {
		t.Errorf("Expected 3 entries in root (README.md, src/, docs/), got %d", len(rootTree.Entries()))
	}

	// Find and verify src directory
	var srcEntry *tree.TreeEntry
	for _, e := range rootTree.Entries() {
		if e.Name().String() == "src" {
			srcEntry = e
			break
		}
	}

	if srcEntry == nil {
		t.Fatal("src directory not found")
	}

	srcTree, err := repo.ReadTreeObject(srcEntry.SHA())
	if err != nil {
		t.Fatalf("Failed to read src tree: %v", err)
	}

	// Verify src has main.go and utils/
	if len(srcTree.Entries()) != 2 {
		t.Errorf("Expected 2 entries in src (main.go, utils/), got %d", len(srcTree.Entries()))
	}

	// Find utils directory
	var utilsEntry *tree.TreeEntry
	for _, e := range srcTree.Entries() {
		if e.Name().String() == "utils" {
			utilsEntry = e
			break
		}
	}

	if utilsEntry == nil {
		t.Fatal("utils directory not found")
	}

	utilsTree, err := repo.ReadTreeObject(utilsEntry.SHA())
	if err != nil {
		t.Fatalf("Failed to read utils tree: %v", err)
	}

	// Verify utils has helper.go and types/
	if len(utilsTree.Entries()) != 2 {
		t.Errorf("Expected 2 entries in utils (helper.go, types/), got %d", len(utilsTree.Entries()))
	}
}

func TestBuildFromIndex_ContextCancellation(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	tb := NewTreeBuilder(repo)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Create a simple index
	idx := index.NewIndex()
	content := []byte("test")
	b := blob.NewBlob(content)
	blobSHA, err := repo.WriteObject(b)
	if err != nil {
		t.Fatalf("Failed to write blob: %v", err)
	}

	entry := index.NewEntry(scpath.RelativePath("test.txt"))
	entry.BlobHash = blobSHA
	entry.Mode = objects.FileModeRegular
	entry.SizeInBytes = uint32(len(content))
	entry.ModificationTime = common.NewTimestampFromTime(time.Now())
	idx.Add(entry)

	// Build should fail with context error
	_, err = tb.BuildFromIndex(ctx, idx)
	if err == nil {
		t.Fatal("Expected error from cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestDirectoryNode_AddEntry_SingleFile(t *testing.T) {
	node := newDirectoryNode("root")

	sha := objects.ObjectHash("abc123")
	node.addEntry("test.txt", sha, objects.FileModeRegular)

	if len(node.files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(node.files))
	}

	if node.files["test.txt"] != sha {
		t.Error("File SHA not set correctly")
	}

	if len(node.subdirs) != 0 {
		t.Errorf("Expected 0 subdirectories, got %d", len(node.subdirs))
	}
}

func TestDirectoryNode_AddEntry_FileInSubdir(t *testing.T) {
	node := newDirectoryNode("root")

	sha := objects.ObjectHash("abc123")
	node.addEntry("src/main.go", sha, objects.FileModeRegular)

	if len(node.files) != 0 {
		t.Errorf("Expected 0 files in root, got %d", len(node.files))
	}

	if len(node.subdirs) != 1 {
		t.Errorf("Expected 1 subdirectory, got %d", len(node.subdirs))
	}

	srcDir, exists := node.subdirs["src"]
	if !exists {
		t.Fatal("src subdirectory not created")
	}

	if len(srcDir.files) != 1 {
		t.Errorf("Expected 1 file in src, got %d", len(srcDir.files))
	}

	if srcDir.files["main.go"] != sha {
		t.Error("File not added to correct subdirectory")
	}
}

func TestDirectoryNode_AddEntry_NestedPath(t *testing.T) {
	node := newDirectoryNode("root")

	sha := objects.ObjectHash("abc123")
	node.addEntry("a/b/c/file.txt", sha, objects.FileModeRegular)

	// Verify directory structure
	if len(node.subdirs) != 1 {
		t.Errorf("Expected 1 subdirectory at root, got %d", len(node.subdirs))
	}

	aDir := node.subdirs["a"]
	if aDir == nil {
		t.Fatal("Directory 'a' not created")
	}

	bDir := aDir.subdirs["b"]
	if bDir == nil {
		t.Fatal("Directory 'b' not created")
	}

	cDir := bDir.subdirs["c"]
	if cDir == nil {
		t.Fatal("Directory 'c' not created")
	}

	if len(cDir.files) != 1 {
		t.Errorf("Expected 1 file in 'c', got %d", len(cDir.files))
	}

	if cDir.files["file.txt"] != sha {
		t.Error("File not added to correct nested directory")
	}
}

func TestDirectoryNode_AddEntry_MultipleFiles(t *testing.T) {
	node := newDirectoryNode("root")

	files := map[string]objects.ObjectHash{
		"README.md":         "sha1",
		"src/main.go":       "sha2",
		"src/utils.go":      "sha3",
		"docs/guide.md":     "sha4",
		"docs/api/index.md": "sha5",
	}

	for path, sha := range files {
		node.addEntry(path, sha, objects.FileModeRegular)
	}

	// Verify root level
	if len(node.files) != 1 {
		t.Errorf("Expected 1 file at root, got %d", len(node.files))
	}
	if node.files["README.md"] != "sha1" {
		t.Error("README.md not at root")
	}

	// Verify src directory
	srcDir := node.subdirs["src"]
	if srcDir == nil {
		t.Fatal("src directory not created")
	}
	if len(srcDir.files) != 2 {
		t.Errorf("Expected 2 files in src, got %d", len(srcDir.files))
	}

	// Verify docs directory
	docsDir := node.subdirs["docs"]
	if docsDir == nil {
		t.Fatal("docs directory not created")
	}
	if len(docsDir.files) != 1 {
		t.Errorf("Expected 1 file in docs, got %d", len(docsDir.files))
	}

	// Verify docs/api directory
	apiDir := docsDir.subdirs["api"]
	if apiDir == nil {
		t.Fatal("docs/api directory not created")
	}
	if len(apiDir.files) != 1 {
		t.Errorf("Expected 1 file in docs/api, got %d", len(apiDir.files))
	}
}

func TestBuildTree_Integration(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	tb := NewTreeBuilder(repo)
	ctx := context.Background()

	// Create a complex directory structure
	files := map[string]string{
		"README.md":                 "# Project",
		".gitignore":                "*.log",
		"main.go":                   "package main",
		"go.mod":                    "module test",
		"cmd/server/main.go":        "package main",
		"cmd/cli/main.go":           "package main",
		"internal/api/handler.go":   "package api",
		"internal/db/connection.go": "package db",
		"pkg/utils/helper.go":       "package utils",
		"pkg/models/user.go":        "package models",
		"docs/README.md":            "# Documentation",
		"test/integration_test.go":  "package test",
	}

	idx := index.NewIndex()
	for filename, content := range files {
		b := blob.NewBlob([]byte(content))
		blobSHA, err := repo.WriteObject(b)
		if err != nil {
			t.Fatalf("Failed to write blob for %s: %v", filename, err)
		}

		entry := index.NewEntry(scpath.RelativePath(filename))
		entry.BlobHash = blobSHA
		entry.Mode = objects.FileModeRegular
		entry.SizeInBytes = uint32(len(content))
		entry.ModificationTime = common.NewTimestampFromTime(time.Now())
		idx.Add(entry)
	}

	// Build tree
	treeSHA, err := tb.BuildFromIndex(ctx, idx)
	if err != nil {
		t.Fatalf("BuildFromIndex failed: %v", err)
	}

	// Verify we can read the tree
	rootTree, err := repo.ReadTreeObject(treeSHA)
	if err != nil {
		t.Fatalf("Failed to read root tree: %v", err)
	}

	// Root should have: .gitignore, README.md, main.go, go.mod, cmd/, internal/, pkg/, docs/, test/
	expectedRootEntries := 9
	if len(rootTree.Entries()) != expectedRootEntries {
		t.Errorf("Expected %d entries in root, got %d", expectedRootEntries, len(rootTree.Entries()))
	}

	// Count total entries (for verification)
	totalCount := 0
	for _, e := range rootTree.Entries() {
		if !e.IsDirectory() {
			totalCount++
		}
	}

	// We should have 4 files at root level
	if totalCount != 4 {
		t.Errorf("Expected 4 files at root level, got %d", totalCount)
	}
}
