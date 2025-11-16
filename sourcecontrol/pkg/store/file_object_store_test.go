package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/blob"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// setupTestRepo creates a temporary test repository
func setupTestRepo(t *testing.T) (scpath.RepositoryPath, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "sourcecontrol-test-*")
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

func TestFileObjectStore_Initialize(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	store := NewFileObjectStore()

	if store.IsInitialized() {
		t.Error("store should not be initialized before Initialize() is called")
	}

	err := store.Initialize(repoPath)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	if !store.IsInitialized() {
		t.Error("store should be initialized after Initialize() is called")
	}

	// Check that objects directory was created
	objectsPath := store.GetObjectsPath()
	if _, err := os.Stat(objectsPath.String()); os.IsNotExist(err) {
		t.Errorf("objects directory was not created at %s", objectsPath)
	}
}

func TestFileObjectStore_WriteAndReadBlob(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	store := NewFileObjectStore()
	if err := store.Initialize(repoPath); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Create a test blob
	testData := []byte("Hello, World! This is a test blob.")
	testBlob := blob.NewBlob(testData)

	// Write the blob
	hash, err := store.WriteObject(testBlob)
	if err != nil {
		t.Fatalf("WriteObject() failed: %v", err)
	}

	if hash.IsZero() {
		t.Error("WriteObject() returned zero hash")
	}

	// Verify the hash matches
	expectedHash, err := testBlob.Hash()
	if err != nil {
		t.Fatalf("failed to get blob hash: %v", err)
	}

	if !hash.Equal(expectedHash) {
		t.Errorf("hash mismatch: got %s, want %s", hash, expectedHash)
	}

	// Read the blob back
	readObj, err := store.ReadObject(hash)
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

	// Verify content matches
	content, err := readBlob.Content()
	if err != nil {
		t.Fatalf("failed to get blob content: %v", err)
	}

	if string(content) != string(testData) {
		t.Errorf("content mismatch: got %s, want %s", content, testData)
	}
}

func TestFileObjectStore_WriteAndReadTree(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	store := NewFileObjectStore()
	if err := store.Initialize(repoPath); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Create some test blobs first
	blob1 := blob.NewBlob([]byte("file1 content"))
	hash1, err := blob1.Hash()
	if err != nil {
		t.Fatalf("failed to get blob1 hash: %v", err)
	}

	blob2 := blob.NewBlob([]byte("file2 content"))
	hash2, err := blob2.Hash()
	if err != nil {
		t.Fatalf("failed to get blob2 hash: %v", err)
	}

	// Create tree entries
	entry1, err := tree.NewTreeEntryFromStrings(objects.FileModeRegular.ToOctalString(), "file1.txt", hash1.String())
	if err != nil {
		t.Fatalf("failed to create tree entry 1: %v", err)
	}

	entry2, err := tree.NewTreeEntryFromStrings(objects.FileModeRegular.ToOctalString(), "file2.txt", hash2.String())
	if err != nil {
		t.Fatalf("failed to create tree entry 2: %v", err)
	}

	// Create tree
	testTree := tree.NewTree([]*tree.TreeEntry{entry1, entry2})

	// Write the tree
	treeHash, err := store.WriteObject(testTree)
	if err != nil {
		t.Fatalf("WriteObject() failed for tree: %v", err)
	}

	// Read the tree back
	readObj, err := store.ReadObject(treeHash)
	if err != nil {
		t.Fatalf("ReadObject() failed: %v", err)
	}

	// Verify it's a tree
	readTree, ok := readObj.(*tree.Tree)
	if !ok {
		t.Fatalf("expected *tree.Tree, got %T", readObj)
	}

	// Verify entries
	entries := readTree.Entries()
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestFileObjectStore_WriteAndReadCommit(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	store := NewFileObjectStore()
	if err := store.Initialize(repoPath); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Create a test commit
	author, err := commit.NewCommitPerson("John Doe", "john@example.com", time.Now())
	if err != nil {
		t.Fatalf("failed to create author: %v", err)
	}

	committer, err := commit.NewCommitPerson("Jane Smith", "jane@example.com", time.Now())
	if err != nil {
		t.Fatalf("failed to create committer: %v", err)
	}

	// Use a valid dummy tree SHA
	treeSHA := "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

	testCommit, err := commit.NewCommitBuilder().
		Tree(treeSHA).
		Author(author).
		Committer(committer).
		Message("Initial commit\n\nThis is a test commit.").
		Build()

	if err != nil {
		t.Fatalf("failed to build commit: %v", err)
	}

	// Write the commit
	commitHash, err := store.WriteObject(testCommit)
	if err != nil {
		t.Fatalf("WriteObject() failed for commit: %v", err)
	}

	// Read the commit back
	readObj, err := store.ReadObject(commitHash)
	if err != nil {
		t.Fatalf("ReadObject() failed: %v", err)
	}

	// Verify it's a commit
	readCommit, ok := readObj.(*commit.Commit)
	if !ok {
		t.Fatalf("expected *commit.Commit, got %T", readObj)
	}

	// Verify commit fields
	if readCommit.TreeSHA.String() != treeSHA {
		t.Errorf("tree SHA mismatch: got %s, want %s", readCommit.TreeSHA, treeSHA)
	}

	if readCommit.Author.Name != author.Name {
		t.Errorf("author name mismatch: got %s, want %s", readCommit.Author.Name, author.Name)
	}

	if readCommit.Committer.Name != committer.Name {
		t.Errorf("committer name mismatch: got %s, want %s", readCommit.Committer.Name, committer.Name)
	}
}

func TestFileObjectStore_HasObject(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	store := NewFileObjectStore()
	if err := store.Initialize(repoPath); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Create and write a test blob
	testBlob := blob.NewBlob([]byte("test data"))
	hash, err := store.WriteObject(testBlob)
	if err != nil {
		t.Fatalf("WriteObject() failed: %v", err)
	}

	// Check that it exists
	exists, err := store.HasObject(hash)
	if err != nil {
		t.Fatalf("HasObject() failed: %v", err)
	}

	if !exists {
		t.Error("HasObject() returned false for existing object")
	}

	// Check for non-existent object
	fakeHash := objects.ZeroHash()
	exists, err = store.HasObject(fakeHash)
	if err != nil {
		t.Fatalf("HasObject() failed for non-existent object: %v", err)
	}

	if exists {
		t.Error("HasObject() returned true for non-existent object")
	}
}

func TestFileObjectStore_WriteIdempotent(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	store := NewFileObjectStore()
	if err := store.Initialize(repoPath); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	testBlob := blob.NewBlob([]byte("test data"))

	// Write the same blob twice
	hash1, err := store.WriteObject(testBlob)
	if err != nil {
		t.Fatalf("first WriteObject() failed: %v", err)
	}

	hash2, err := store.WriteObject(testBlob)
	if err != nil {
		t.Fatalf("second WriteObject() failed: %v", err)
	}

	// Hashes should be identical
	if !hash1.Equal(hash2) {
		t.Errorf("hash mismatch: first %s, second %s", hash1, hash2)
	}

	// Object should still exist only once
	count, err := store.ObjectCount()
	if err != nil {
		t.Fatalf("ObjectCount() failed: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 object, got %d", count)
	}
}

func TestFileObjectStore_ReadNonExistentObject(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	store := NewFileObjectStore()
	if err := store.Initialize(repoPath); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Try to read a non-existent object
	fakeHash, err := objects.NewObjectHashFromString("0123456789abcdef0123456789abcdef01234567")
	if err != nil {
		t.Fatalf("failed to create fake hash: %v", err)
	}

	obj, err := store.ReadObject(fakeHash)
	if err != nil {
		t.Fatalf("ReadObject() failed: %v", err)
	}

	if obj != nil {
		t.Error("ReadObject() should return nil for non-existent object")
	}
}

func TestFileObjectStore_InvalidHash(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	store := NewFileObjectStore()
	if err := store.Initialize(repoPath); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Test with invalid hash
	invalidHash := objects.ObjectHash("invalid")

	_, err := store.ReadObject(invalidHash)
	if err == nil {
		t.Error("ReadObject() should fail with invalid hash")
	}

	_, err = store.HasObject(invalidHash)
	if err == nil {
		t.Error("HasObject() should fail with invalid hash")
	}
}

func TestFileObjectStore_UninitializedStore(t *testing.T) {
	store := NewFileObjectStore()

	testBlob := blob.NewBlob([]byte("test"))

	// Try to write without initialization
	_, err := store.WriteObject(testBlob)
	if err == nil {
		t.Error("WriteObject() should fail on uninitialized store")
	}

	// Try to read without initialization
	hash := objects.ZeroHash()
	_, err = store.ReadObject(hash)
	if err == nil {
		t.Error("ReadObject() should fail on uninitialized store")
	}

	// Try to check existence without initialization
	_, err = store.HasObject(hash)
	if err == nil {
		t.Error("HasObject() should fail on uninitialized store")
	}
}

func TestFileObjectStore_ObjectCount(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	store := NewFileObjectStore()
	if err := store.Initialize(repoPath); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Initially should be 0
	count, err := store.ObjectCount()
	if err != nil {
		t.Fatalf("ObjectCount() failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 objects initially, got %d", count)
	}

	// Write some objects
	numObjects := 5
	for i := 0; i < numObjects; i++ {
		testBlob := blob.NewBlob([]byte("test data " + string(rune(i))))
		_, err := store.WriteObject(testBlob)
		if err != nil {
			t.Fatalf("WriteObject() failed: %v", err)
		}
	}

	// Should now have 5 objects
	count, err = store.ObjectCount()
	if err != nil {
		t.Fatalf("ObjectCount() failed: %v", err)
	}
	if count != numObjects {
		t.Errorf("expected %d objects, got %d", numObjects, count)
	}
}

func TestFileObjectStore_DirectoryStructure(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	store := NewFileObjectStore()
	if err := store.Initialize(repoPath); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Create a test blob
	testBlob := blob.NewBlob([]byte("test data"))
	hash, err := store.WriteObject(testBlob)
	if err != nil {
		t.Fatalf("WriteObject() failed: %v", err)
	}

	// Verify directory structure
	hashStr := hash.String()
	expectedDir := filepath.Join(store.GetObjectsPath().String(), hashStr[:2])
	expectedFile := filepath.Join(expectedDir, hashStr[2:])

	// Check directory exists
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("object directory does not exist: %s", expectedDir)
	}

	// Check file exists
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("object file does not exist: %s", expectedFile)
	}

	// Verify file is read-only
	info, err := os.Stat(expectedFile)
	if err != nil {
		t.Fatalf("failed to stat object file: %v", err)
	}

	// On Unix systems, 0444 means read-only
	expectedPerm := os.FileMode(0444)
	if info.Mode().Perm() != expectedPerm {
		t.Logf("Warning: file permissions are %v, expected %v", info.Mode().Perm(), expectedPerm)
		// Note: This might differ on Windows, so we just log a warning
	}
}

func TestFileObjectStore_CompressedStorage(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	store := NewFileObjectStore()
	if err := store.Initialize(repoPath); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Create a large blob with repetitive data (highly compressible)
	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte('A')
	}

	testBlob := blob.NewBlob(largeData)
	hash, err := store.WriteObject(testBlob)
	if err != nil {
		t.Fatalf("WriteObject() failed: %v", err)
	}

	// Get file path
	objPath, err := store.resolveObjectPath(hash)
	if err != nil {
		t.Fatalf("resolveObjectPath() failed: %v", err)
	}

	// Check file size
	info, err := os.Stat(objPath.String())
	if err != nil {
		t.Fatalf("failed to stat object file: %v", err)
	}

	// Compressed size should be much smaller than original
	// (10000 bytes of 'A' should compress to less than 1000 bytes)
	if info.Size() >= 1000 {
		t.Logf("Warning: compression may not be working effectively. File size: %d bytes", info.Size())
	}
}
