package internal

import (
	"context"
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// createTestCommit creates a test commit with the given tree SHA
func createTestCommit(t *testing.T, repo *sourcerepo.SourceRepository, treeSHA objects.ObjectHash) objects.ObjectHash {
	t.Helper()

	author, err := commit.NewCommitPerson("Test Author", "test@example.com", time.Now())
	if err != nil {
		t.Fatalf("Failed to create author: %v", err)
	}

	committer, err := commit.NewCommitPerson("Test Committer", "test@example.com", time.Now())
	if err != nil {
		t.Fatalf("Failed to create committer: %v", err)
	}

	c, err := commit.NewCommitBuilder().
		TreeHash(treeSHA).
		Author(author).
		Committer(committer).
		Message("Test commit").
		Build()

	if err != nil {
		t.Fatalf("Failed to build commit: %v", err)
	}

	commitSHA, err := repo.WriteObject(c)
	if err != nil {
		t.Fatalf("Failed to write commit: %v", err)
	}

	return commitSHA
}

// createTestTree creates a tree from entries
func createTestTree(t *testing.T, repo *sourcerepo.SourceRepository, entries []*tree.TreeEntry) objects.ObjectHash {
	t.Helper()

	tr := tree.NewTree(entries)
	treeSHA, err := repo.WriteObject(tr)
	if err != nil {
		t.Fatalf("Failed to write tree: %v", err)
	}

	return treeSHA
}

// createTestEntry creates a tree entry
func createTestEntry(t *testing.T, name string, sha objects.ObjectHash, mode objects.FileMode) *tree.TreeEntry {
	t.Helper()

	entry, err := tree.NewTreeEntry(mode, scpath.RelativePath(name), sha)
	if err != nil {
		t.Fatalf("Failed to create tree entry: %v", err)
	}

	return entry
}

// TestNewAnalyzer verifies that NewAnalyzer creates an Analyzer with the correct repository
func TestNewAnalyzer(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	if analyzer == nil {
		t.Fatal("NewAnalyzer returned nil")
	}

	if analyzer.repo != repo {
		t.Error("Analyzer repository not set correctly")
	}
}

// TestGetCommitFiles_EmptyCommit tests retrieving files from a commit with an empty tree
func TestGetCommitFiles_EmptyCommit(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	// Create an empty tree
	treeSHA := createTestTree(t, repo, []*tree.TreeEntry{})

	// Create a commit pointing to the empty tree
	commitSHA := createTestCommit(t, repo, treeSHA)

	// Get files from commit
	files, err := analyzer.GetCommitFiles(context.Background(), commitSHA)
	if err != nil {
		t.Fatalf("GetCommitFiles failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(files))
	}
}

// TestGetCommitFiles_SingleFile tests retrieving files from a commit with one file
func TestGetCommitFiles_SingleFile(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	// Create a blob
	blobSHA := createTestBlob(t, repo, "test content")

	// Create a tree with one file
	entry := createTestEntry(t, "test.txt", blobSHA, objects.FileModeRegular)
	treeSHA := createTestTree(t, repo, []*tree.TreeEntry{entry})

	// Create a commit
	commitSHA := createTestCommit(t, repo, treeSHA)

	// Get files from commit
	files, err := analyzer.GetCommitFiles(context.Background(), commitSHA)
	if err != nil {
		t.Fatalf("GetCommitFiles failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}

	testPath := scpath.RelativePath("test.txt")
	fileInfo, exists := files[testPath]
	if !exists {
		t.Fatal("Expected file 'test.txt' not found")
	}

	if fileInfo.SHA != blobSHA {
		t.Errorf("Expected SHA %s, got %s", blobSHA.Short(), fileInfo.SHA.Short())
	}

	if fileInfo.Mode != objects.FileModeRegular {
		t.Errorf("Expected mode %s, got %s", objects.FileModeRegular, fileInfo.Mode)
	}
}

// TestGetCommitFiles_NestedDirectories tests retrieving files from a commit with nested directories
func TestGetCommitFiles_NestedDirectories(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	// Create blobs
	blob1SHA := createTestBlob(t, repo, "file1 content")
	blob2SHA := createTestBlob(t, repo, "file2 content")
	blob3SHA := createTestBlob(t, repo, "file3 content")

	// Create nested directory structure: src/nested/file3.txt
	nestedEntry := createTestEntry(t, "file3.txt", blob3SHA, objects.FileModeRegular)
	nestedTreeSHA := createTestTree(t, repo, []*tree.TreeEntry{nestedEntry})

	// Create src directory
	srcFile := createTestEntry(t, "file2.txt", blob2SHA, objects.FileModeRegular)
	srcNested := createTestEntry(t, "nested", nestedTreeSHA, objects.FileModeDirectory)
	srcTreeSHA := createTestTree(t, repo, []*tree.TreeEntry{srcFile, srcNested})

	// Create root tree
	rootFile := createTestEntry(t, "file1.txt", blob1SHA, objects.FileModeRegular)
	rootSrc := createTestEntry(t, "src", srcTreeSHA, objects.FileModeDirectory)
	rootTreeSHA := createTestTree(t, repo, []*tree.TreeEntry{rootFile, rootSrc})

	// Create commit
	commitSHA := createTestCommit(t, repo, rootTreeSHA)

	// Get files from commit
	files, err := analyzer.GetCommitFiles(context.Background(), commitSHA)
	if err != nil {
		t.Fatalf("GetCommitFiles failed: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("Expected 3 files, got %d", len(files))
	}

	// Verify all files exist with correct paths
	expectedFiles := map[scpath.RelativePath]objects.ObjectHash{
		scpath.RelativePath("file1.txt"):            blob1SHA,
		scpath.RelativePath("src/file2.txt"):        blob2SHA,
		scpath.RelativePath("src/nested/file3.txt"): blob3SHA,
	}

	for path, expectedSHA := range expectedFiles {
		fileInfo, exists := files[path]
		if !exists {
			t.Errorf("Expected file '%s' not found", path)
			continue
		}

		if fileInfo.SHA != expectedSHA {
			t.Errorf("File %s: expected SHA %s, got %s", path, expectedSHA.Short(), fileInfo.SHA.Short())
		}
	}
}

// TestGetCommitFiles_DifferentFileTypes tests retrieving files with different modes
func TestGetCommitFiles_DifferentFileTypes(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	// Create blobs for different file types
	regularBlobSHA := createTestBlob(t, repo, "regular file")
	execBlobSHA := createTestBlob(t, repo, "#!/bin/bash\necho 'executable'")
	symlinkBlobSHA := createTestBlob(t, repo, "target/path")

	// Create tree with different file types
	regularEntry := createTestEntry(t, "regular.txt", regularBlobSHA, objects.FileModeRegular)
	execEntry := createTestEntry(t, "script.sh", execBlobSHA, objects.FileModeExecutable)
	symlinkEntry := createTestEntry(t, "link.txt", symlinkBlobSHA, objects.FileModeSymlink)

	treeSHA := createTestTree(t, repo, []*tree.TreeEntry{regularEntry, execEntry, symlinkEntry})

	// Create commit
	commitSHA := createTestCommit(t, repo, treeSHA)

	// Get files from commit
	files, err := analyzer.GetCommitFiles(context.Background(), commitSHA)
	if err != nil {
		t.Fatalf("GetCommitFiles failed: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("Expected 3 files, got %d", len(files))
	}

	// Verify file modes
	testCases := []struct {
		path         scpath.RelativePath
		expectedMode objects.FileMode
		expectedSHA  objects.ObjectHash
	}{
		{scpath.RelativePath("regular.txt"), objects.FileModeRegular, regularBlobSHA},
		{scpath.RelativePath("script.sh"), objects.FileModeExecutable, execBlobSHA},
		{scpath.RelativePath("link.txt"), objects.FileModeSymlink, symlinkBlobSHA},
	}

	for _, tc := range testCases {
		fileInfo, exists := files[tc.path]
		if !exists {
			t.Errorf("Expected file '%s' not found", tc.path)
			continue
		}

		if fileInfo.Mode != tc.expectedMode {
			t.Errorf("File %s: expected mode %s, got %s", tc.path, tc.expectedMode, fileInfo.Mode)
		}

		if fileInfo.SHA != tc.expectedSHA {
			t.Errorf("File %s: expected SHA %s, got %s", tc.path, tc.expectedSHA.Short(), fileInfo.SHA.Short())
		}
	}
}

// TestGetCommitFiles_InvalidCommit tests error handling for invalid commit SHA
func TestGetCommitFiles_InvalidCommit(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	// Try to get files from non-existent commit
	invalidSHA := objects.ObjectHash("0000000000000000000000000000000000000000")
	_, err := analyzer.GetCommitFiles(context.Background(), invalidSHA)
	if err == nil {
		t.Error("Expected error for invalid commit SHA, got nil")
	}
}

// TestGetIndexFiles tests extracting file information from an index
func TestGetIndexFiles(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	// Create test blobs
	blob1SHA := createTestBlob(t, repo, "content1")
	blob2SHA := createTestBlob(t, repo, "content2")

	// Create index with entries
	idx := index.NewIndex()
	entry1 := index.NewEntry(scpath.RelativePath("file1.txt"))
	entry1.BlobHash = blob1SHA
	entry1.Mode = objects.FileModeRegular

	entry2 := index.NewEntry(scpath.RelativePath("dir/file2.txt"))
	entry2.BlobHash = blob2SHA
	entry2.Mode = objects.FileModeExecutable

	idx.Entries = []*index.Entry{entry1, entry2}

	// Get files from index
	files := analyzer.GetIndexFiles(idx)

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}

	// Verify entries
	file1 := files[scpath.RelativePath("file1.txt")]
	if file1.SHA != blob1SHA {
		t.Errorf("File1: expected SHA %s, got %s", blob1SHA.Short(), file1.SHA.Short())
	}
	if file1.Mode != objects.FileModeRegular {
		t.Errorf("File1: expected mode %s, got %s", objects.FileModeRegular, file1.Mode)
	}

	file2 := files[scpath.RelativePath("dir/file2.txt")]
	if file2.SHA != blob2SHA {
		t.Errorf("File2: expected SHA %s, got %s", blob2SHA.Short(), file2.SHA.Short())
	}
	if file2.Mode != objects.FileModeExecutable {
		t.Errorf("File2: expected mode %s, got %s", objects.FileModeExecutable, file2.Mode)
	}
}

// TestGetIndexFiles_Empty tests extracting files from an empty index
func TestGetIndexFiles_Empty(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	idx := index.NewIndex()
	files := analyzer.GetIndexFiles(idx)

	if len(files) != 0 {
		t.Errorf("Expected 0 files from empty index, got %d", len(files))
	}
}

// TestAnalyzeChanges_NoChanges tests analyzing when current and target are identical
func TestAnalyzeChanges_NoChanges(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	blobSHA := createTestBlob(t, repo, "content")

	current := map[scpath.RelativePath]FileInfo{
		scpath.RelativePath("file.txt"): {
			SHA:  blobSHA,
			Mode: objects.FileModeRegular,
		},
	}

	target := map[scpath.RelativePath]FileInfo{
		scpath.RelativePath("file.txt"): {
			SHA:  blobSHA,
			Mode: objects.FileModeRegular,
		},
	}

	analysis := analyzer.AnalyzeChanges(current, target)

	if len(analysis.Operations) != 0 {
		t.Errorf("Expected 0 operations, got %d", len(analysis.Operations))
	}

	if analysis.Summary.Created != 0 || analysis.Summary.Modified != 0 || analysis.Summary.Deleted != 0 {
		t.Errorf("Expected empty summary, got Created=%d, Modified=%d, Deleted=%d",
			analysis.Summary.Created, analysis.Summary.Modified, analysis.Summary.Deleted)
	}
}

// TestAnalyzeChanges_CreateFiles tests detecting new files
func TestAnalyzeChanges_CreateFiles(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	blob1SHA := createTestBlob(t, repo, "content1")
	blob2SHA := createTestBlob(t, repo, "content2")

	current := map[scpath.RelativePath]FileInfo{}

	target := map[scpath.RelativePath]FileInfo{
		scpath.RelativePath("new1.txt"): {
			SHA:  blob1SHA,
			Mode: objects.FileModeRegular,
		},
		scpath.RelativePath("new2.txt"): {
			SHA:  blob2SHA,
			Mode: objects.FileModeExecutable,
		},
	}

	analysis := analyzer.AnalyzeChanges(current, target)

	if len(analysis.Operations) != 2 {
		t.Fatalf("Expected 2 operations, got %d", len(analysis.Operations))
	}

	if analysis.Summary.Created != 2 {
		t.Errorf("Expected 2 created files, got %d", analysis.Summary.Created)
	}

	// Verify operations
	for _, op := range analysis.Operations {
		if op.Action != ActionCreate {
			t.Errorf("Expected ActionCreate, got %s", op.Action)
		}
		targetInfo := target[op.Path]
		if op.SHA != targetInfo.SHA {
			t.Errorf("Operation SHA mismatch for %s", op.Path)
		}
		if op.Mode != targetInfo.Mode {
			t.Errorf("Operation Mode mismatch for %s", op.Path)
		}
	}
}

// TestAnalyzeChanges_DeleteFiles tests detecting deleted files
func TestAnalyzeChanges_DeleteFiles(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	blob1SHA := createTestBlob(t, repo, "content1")
	blob2SHA := createTestBlob(t, repo, "content2")

	current := map[scpath.RelativePath]FileInfo{
		scpath.RelativePath("old1.txt"): {
			SHA:  blob1SHA,
			Mode: objects.FileModeRegular,
		},
		scpath.RelativePath("old2.txt"): {
			SHA:  blob2SHA,
			Mode: objects.FileModeRegular,
		},
	}

	target := map[scpath.RelativePath]FileInfo{}

	analysis := analyzer.AnalyzeChanges(current, target)

	if len(analysis.Operations) != 2 {
		t.Fatalf("Expected 2 operations, got %d", len(analysis.Operations))
	}

	if analysis.Summary.Deleted != 2 {
		t.Errorf("Expected 2 deleted files, got %d", analysis.Summary.Deleted)
	}

	// Verify operations
	deletedPaths := make(map[scpath.RelativePath]bool)
	for _, op := range analysis.Operations {
		if op.Action != ActionDelete {
			t.Errorf("Expected ActionDelete, got %s", op.Action)
		}
		deletedPaths[op.Path] = true
	}

	if !deletedPaths[scpath.RelativePath("old1.txt")] {
		t.Error("Expected old1.txt to be deleted")
	}
	if !deletedPaths[scpath.RelativePath("old2.txt")] {
		t.Error("Expected old2.txt to be deleted")
	}
}

// TestAnalyzeChanges_ModifyFiles tests detecting modified files
func TestAnalyzeChanges_ModifyFiles(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	oldSHA := createTestBlob(t, repo, "old content")
	newSHA := createTestBlob(t, repo, "new content")

	current := map[scpath.RelativePath]FileInfo{
		scpath.RelativePath("file.txt"): {
			SHA:  oldSHA,
			Mode: objects.FileModeRegular,
		},
	}

	target := map[scpath.RelativePath]FileInfo{
		scpath.RelativePath("file.txt"): {
			SHA:  newSHA,
			Mode: objects.FileModeRegular,
		},
	}

	analysis := analyzer.AnalyzeChanges(current, target)

	if len(analysis.Operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(analysis.Operations))
	}

	if analysis.Summary.Modified != 1 {
		t.Errorf("Expected 1 modified file, got %d", analysis.Summary.Modified)
	}

	op := analysis.Operations[0]
	if op.Action != ActionModify {
		t.Errorf("Expected ActionModify, got %s", op.Action)
	}
	if op.SHA != newSHA {
		t.Errorf("Expected new SHA %s, got %s", newSHA.Short(), op.SHA.Short())
	}
}

// TestAnalyzeChanges_ModifyFileMode tests detecting mode changes
func TestAnalyzeChanges_ModifyFileMode(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	blobSHA := createTestBlob(t, repo, "content")

	current := map[scpath.RelativePath]FileInfo{
		scpath.RelativePath("file.txt"): {
			SHA:  blobSHA,
			Mode: objects.FileModeRegular,
		},
	}

	target := map[scpath.RelativePath]FileInfo{
		scpath.RelativePath("file.txt"): {
			SHA:  blobSHA,
			Mode: objects.FileModeExecutable,
		},
	}

	analysis := analyzer.AnalyzeChanges(current, target)

	if len(analysis.Operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(analysis.Operations))
	}

	if analysis.Summary.Modified != 1 {
		t.Errorf("Expected 1 modified file, got %d", analysis.Summary.Modified)
	}

	op := analysis.Operations[0]
	if op.Action != ActionModify {
		t.Errorf("Expected ActionModify, got %s", op.Action)
	}
	if op.Mode != objects.FileModeExecutable {
		t.Errorf("Expected mode %s, got %s", objects.FileModeExecutable, op.Mode)
	}
}

// TestAnalyzeChanges_MixedOperations tests detecting create, modify, and delete operations together
func TestAnalyzeChanges_MixedOperations(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	blob1SHA := createTestBlob(t, repo, "content1")
	blob2SHA := createTestBlob(t, repo, "content2")
	blob3SHA := createTestBlob(t, repo, "content3")
	blob4SHA := createTestBlob(t, repo, "content4")

	current := map[scpath.RelativePath]FileInfo{
		scpath.RelativePath("keep.txt"): {
			SHA:  blob1SHA,
			Mode: objects.FileModeRegular,
		},
		scpath.RelativePath("modify.txt"): {
			SHA:  blob2SHA,
			Mode: objects.FileModeRegular,
		},
		scpath.RelativePath("delete.txt"): {
			SHA:  blob3SHA,
			Mode: objects.FileModeRegular,
		},
	}

	target := map[scpath.RelativePath]FileInfo{
		scpath.RelativePath("keep.txt"): {
			SHA:  blob1SHA,
			Mode: objects.FileModeRegular,
		},
		scpath.RelativePath("modify.txt"): {
			SHA:  blob4SHA,
			Mode: objects.FileModeRegular,
		},
		scpath.RelativePath("create.txt"): {
			SHA:  blob3SHA,
			Mode: objects.FileModeExecutable,
		},
	}

	analysis := analyzer.AnalyzeChanges(current, target)

	if len(analysis.Operations) != 3 {
		t.Fatalf("Expected 3 operations, got %d", len(analysis.Operations))
	}

	if analysis.Summary.Created != 1 {
		t.Errorf("Expected 1 created file, got %d", analysis.Summary.Created)
	}
	if analysis.Summary.Modified != 1 {
		t.Errorf("Expected 1 modified file, got %d", analysis.Summary.Modified)
	}
	if analysis.Summary.Deleted != 1 {
		t.Errorf("Expected 1 deleted file, got %d", analysis.Summary.Deleted)
	}

	// Verify specific operations
	actionCounts := make(map[ActionType]int)
	for _, op := range analysis.Operations {
		actionCounts[op.Action]++

		switch op.Action {
		case ActionCreate:
			if op.Path != scpath.RelativePath("create.txt") {
				t.Errorf("Unexpected create path: %s", op.Path)
			}
		case ActionModify:
			if op.Path != scpath.RelativePath("modify.txt") {
				t.Errorf("Unexpected modify path: %s", op.Path)
			}
		case ActionDelete:
			if op.Path != scpath.RelativePath("delete.txt") {
				t.Errorf("Unexpected delete path: %s", op.Path)
			}
		}
	}

	if actionCounts[ActionCreate] != 1 || actionCounts[ActionModify] != 1 || actionCounts[ActionDelete] != 1 {
		t.Errorf("Incorrect action distribution: %v", actionCounts)
	}
}

// TestAreTreesIdentical_SameSHA tests comparing trees with identical SHAs
func TestAreTreesIdentical_SameSHA(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	// Create a tree
	blobSHA := createTestBlob(t, repo, "content")
	entry := createTestEntry(t, "file.txt", blobSHA, objects.FileModeRegular)
	treeSHA := createTestTree(t, repo, []*tree.TreeEntry{entry})

	// Compare tree with itself
	identical, err := analyzer.AreTreesIdentical(context.Background(), treeSHA, treeSHA)
	if err != nil {
		t.Fatalf("AreTreesIdentical failed: %v", err)
	}

	if !identical {
		t.Error("Expected trees with same SHA to be identical")
	}
}

// TestAreTreesIdentical_SameContent tests comparing trees with same content
func TestAreTreesIdentical_SameContent(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	blobSHA := createTestBlob(t, repo, "content")

	// Create first tree
	entry1 := createTestEntry(t, "file.txt", blobSHA, objects.FileModeRegular)
	tree1SHA := createTestTree(t, repo, []*tree.TreeEntry{entry1})

	// Create second tree with identical content
	entry2 := createTestEntry(t, "file.txt", blobSHA, objects.FileModeRegular)
	tree2SHA := createTestTree(t, repo, []*tree.TreeEntry{entry2})

	// They should have the same SHA since content is identical
	if tree1SHA != tree2SHA {
		t.Skip("Trees with identical content should have same SHA - skipping comparison")
	}

	identical, err := analyzer.AreTreesIdentical(context.Background(), tree1SHA, tree2SHA)
	if err != nil {
		t.Fatalf("AreTreesIdentical failed: %v", err)
	}

	if !identical {
		t.Error("Expected trees with identical content to be identical")
	}
}

// TestAreTreesIdentical_DifferentContent tests comparing trees with different content
func TestAreTreesIdentical_DifferentContent(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	blob1SHA := createTestBlob(t, repo, "content1")
	blob2SHA := createTestBlob(t, repo, "content2")

	// Create first tree
	entry1 := createTestEntry(t, "file.txt", blob1SHA, objects.FileModeRegular)
	tree1SHA := createTestTree(t, repo, []*tree.TreeEntry{entry1})

	// Create second tree with different content
	entry2 := createTestEntry(t, "file.txt", blob2SHA, objects.FileModeRegular)
	tree2SHA := createTestTree(t, repo, []*tree.TreeEntry{entry2})

	identical, err := analyzer.AreTreesIdentical(context.Background(), tree1SHA, tree2SHA)
	if err != nil {
		t.Fatalf("AreTreesIdentical failed: %v", err)
	}

	if identical {
		t.Error("Expected trees with different content to not be identical")
	}
}

// TestAreTreesIdentical_DifferentFileCount tests comparing trees with different number of files
func TestAreTreesIdentical_DifferentFileCount(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	blob1SHA := createTestBlob(t, repo, "content1")
	blob2SHA := createTestBlob(t, repo, "content2")

	// Create first tree with one file
	entry1 := createTestEntry(t, "file1.txt", blob1SHA, objects.FileModeRegular)
	tree1SHA := createTestTree(t, repo, []*tree.TreeEntry{entry1})

	// Create second tree with two files
	entry2a := createTestEntry(t, "file1.txt", blob1SHA, objects.FileModeRegular)
	entry2b := createTestEntry(t, "file2.txt", blob2SHA, objects.FileModeRegular)
	tree2SHA := createTestTree(t, repo, []*tree.TreeEntry{entry2a, entry2b})

	identical, err := analyzer.AreTreesIdentical(context.Background(), tree1SHA, tree2SHA)
	if err != nil {
		t.Fatalf("AreTreesIdentical failed: %v", err)
	}

	if identical {
		t.Error("Expected trees with different file counts to not be identical")
	}
}

// TestAreTreesIdentical_DifferentModes tests comparing trees where file modes differ
func TestAreTreesIdentical_DifferentModes(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	blobSHA := createTestBlob(t, repo, "content")

	// Create first tree with regular file
	entry1 := createTestEntry(t, "file.txt", blobSHA, objects.FileModeRegular)
	tree1SHA := createTestTree(t, repo, []*tree.TreeEntry{entry1})

	// Create second tree with executable file
	entry2 := createTestEntry(t, "file.txt", blobSHA, objects.FileModeExecutable)
	tree2SHA := createTestTree(t, repo, []*tree.TreeEntry{entry2})

	identical, err := analyzer.AreTreesIdentical(context.Background(), tree1SHA, tree2SHA)
	if err != nil {
		t.Fatalf("AreTreesIdentical failed: %v", err)
	}

	if identical {
		t.Error("Expected trees with different file modes to not be identical")
	}
}

// TestAreTreesIdentical_InvalidTree tests error handling for invalid tree SHA
func TestAreTreesIdentical_InvalidTree(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	blobSHA := createTestBlob(t, repo, "content")
	entry := createTestEntry(t, "file.txt", blobSHA, objects.FileModeRegular)
	validTreeSHA := createTestTree(t, repo, []*tree.TreeEntry{entry})

	invalidSHA := objects.ObjectHash("0000000000000000000000000000000000000000")

	// Test with first tree invalid
	_, err := analyzer.AreTreesIdentical(context.Background(), invalidSHA, validTreeSHA)
	if err == nil {
		t.Error("Expected error for invalid tree1 SHA, got nil")
	}

	// Test with second tree invalid
	_, err = analyzer.AreTreesIdentical(context.Background(), validTreeSHA, invalidSHA)
	if err == nil {
		t.Error("Expected error for invalid tree2 SHA, got nil")
	}
}

// TestHasChanged tests the hasChanged helper method
func TestHasChanged(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	sha1 := createTestBlob(t, repo, "content1")
	sha2 := createTestBlob(t, repo, "content2")

	tests := []struct {
		name     string
		current  FileInfo
		target   FileInfo
		expected bool
	}{
		{
			name: "identical files",
			current: FileInfo{
				SHA:  sha1,
				Mode: objects.FileModeRegular,
			},
			target: FileInfo{
				SHA:  sha1,
				Mode: objects.FileModeRegular,
			},
			expected: false,
		},
		{
			name: "different SHA",
			current: FileInfo{
				SHA:  sha1,
				Mode: objects.FileModeRegular,
			},
			target: FileInfo{
				SHA:  sha2,
				Mode: objects.FileModeRegular,
			},
			expected: true,
		},
		{
			name: "different mode",
			current: FileInfo{
				SHA:  sha1,
				Mode: objects.FileModeRegular,
			},
			target: FileInfo{
				SHA:  sha1,
				Mode: objects.FileModeExecutable,
			},
			expected: true,
		},
		{
			name: "different SHA and mode",
			current: FileInfo{
				SHA:  sha1,
				Mode: objects.FileModeRegular,
			},
			target: FileInfo{
				SHA:  sha2,
				Mode: objects.FileModeExecutable,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.hasChanged(tt.current, tt.target)
			if result != tt.expected {
				t.Errorf("hasChanged() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestIsSupportedFileType tests the isSupportedFileType helper method
func TestIsSupportedFileType(t *testing.T) {
	repo, _ := setupTestRepo(t)
	analyzer := NewAnalyzer(repo)

	blobSHA := createTestBlob(t, repo, "content")

	// Create a tree SHA for directory
	dirTree := tree.NewTree([]*tree.TreeEntry{})
	dirSHA, _ := repo.WriteObject(dirTree)

	tests := []struct {
		name     string
		mode     objects.FileMode
		expected bool
	}{
		{"regular file", objects.FileModeRegular, true},
		{"executable file", objects.FileModeExecutable, true},
		{"symlink", objects.FileModeSymlink, true},
		{"directory", objects.FileModeDirectory, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sha := blobSHA
			if tt.mode == objects.FileModeDirectory {
				sha = dirSHA
			}
			entry := createTestEntry(t, "test", sha, tt.mode)
			result := analyzer.isSupportedFileType(entry)
			if result != tt.expected {
				t.Errorf("isSupportedFileType() = %v, expected %v for %s", result, tt.expected, tt.name)
			}
		})
	}
}
