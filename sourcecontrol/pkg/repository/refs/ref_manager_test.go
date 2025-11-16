package refs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

// mockRepository is a simple mock implementation of Repository for testing
type mockRepository struct {
	workingDir scpath.RepositoryPath
	sourceDir  scpath.SourcePath
}

func (m *mockRepository) Initialize(path scpath.RepositoryPath) error {
	return nil
}

func (m *mockRepository) WorkingDirectory() scpath.RepositoryPath {
	return m.workingDir
}

func (m *mockRepository) SourceDirectory() scpath.SourcePath {
	return m.sourceDir
}

func (m *mockRepository) ObjectStore() store.ObjectStore {
	return nil
}

func (m *mockRepository) ReadObject(hash objects.ObjectHash) (objects.BaseObject, error) {
	return nil, nil
}

func (m *mockRepository) WriteObject(obj objects.BaseObject) (objects.ObjectHash, error) {
	return objects.ZeroHash(), nil
}

func (m *mockRepository) Exists() (bool, error) {
	return true, nil
}

func setupTestRepo(t *testing.T) (*RefManager, string, func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "ref-manager-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create .source directory
	sourceDir := filepath.Join(tempDir, scpath.SourceDir)
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	// Create mock repository
	repo := &mockRepository{
		workingDir: scpath.RepositoryPath(tempDir),
		sourceDir:  scpath.SourcePath(sourceDir),
	}

	// Create RefManager
	rm := NewRefManager(repo)

	// Cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return rm, tempDir, cleanup
}

func TestNewRefManager(t *testing.T) {
	rm, _, cleanup := setupTestRepo(t)
	defer cleanup()

	if rm == nil {
		t.Fatal("Expected RefManager to be created")
	}

	if rm.refsPath == "" {
		t.Error("Expected refsPath to be set")
	}

	if rm.headPath == "" {
		t.Error("Expected headPath to be set")
	}
}

func TestRefManager_Init(t *testing.T) {
	rm, tempDir, cleanup := setupTestRepo(t)
	defer cleanup()

	err := rm.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check refs directory exists
	refsDir := filepath.Join(tempDir, scpath.SourceDir, scpath.RefsDir)
	if _, err := os.Stat(refsDir); os.IsNotExist(err) {
		t.Error("refs directory was not created")
	}

	// Check HEAD file exists and has correct content
	headFile := filepath.Join(tempDir, scpath.SourceDir, scpath.HeadFile)
	content, err := os.ReadFile(headFile)
	if err != nil {
		t.Fatalf("Failed to read HEAD file: %v", err)
	}

	expected := "ref: refs/heads/master\n"
	if string(content) != expected {
		t.Errorf("HEAD content = %q, want %q", string(content), expected)
	}
}

func TestRefManager_UpdateRef(t *testing.T) {
	rm, _, cleanup := setupTestRepo(t)
	defer cleanup()

	if err := rm.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	tests := []struct {
		name string
		ref  RefPath
		sha  string
	}{
		{
			name: "update branch ref",
			ref:  "refs/heads/main",
			sha:  "1234567890abcdef1234567890abcdef12345678",
		},
		{
			name: "update tag ref",
			ref:  "refs/tags/v1.0.0",
			sha:  "abcdef1234567890abcdef1234567890abcdef12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := objects.NewObjectHashFromString(tt.sha)
			if err != nil {
				t.Fatalf("Failed to create hash: %v", err)
			}

			err = rm.UpdateRef(tt.ref, hash)
			if err != nil {
				t.Fatalf("UpdateRef failed: %v", err)
			}

			// Read back and verify
			content, err := rm.ReadRef(tt.ref)
			if err != nil {
				t.Fatalf("ReadRef failed: %v", err)
			}

			if content != tt.sha {
				t.Errorf("ReadRef = %q, want %q", content, tt.sha)
			}
		})
	}
}

func TestRefManager_ReadRef(t *testing.T) {
	rm, _, cleanup := setupTestRepo(t)
	defer cleanup()

	if err := rm.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	sha := "1234567890abcdef1234567890abcdef12345678"
	ref := RefPath("refs/heads/test")

	// Write ref first
	hash, err := objects.NewObjectHashFromString(sha)
	if err != nil {
		t.Fatalf("Failed to create hash: %v", err)
	}

	if err := rm.UpdateRef(ref, hash); err != nil {
		t.Fatalf("UpdateRef failed: %v", err)
	}

	// Read it back
	content, err := rm.ReadRef(ref)
	if err != nil {
		t.Fatalf("ReadRef failed: %v", err)
	}

	if content != sha {
		t.Errorf("ReadRef = %q, want %q", content, sha)
	}
}

func TestRefManager_ReadRef_NotFound(t *testing.T) {
	rm, _, cleanup := setupTestRepo(t)
	defer cleanup()

	if err := rm.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	_, err := rm.ReadRef("refs/heads/nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent ref")
	}
}

func TestRefManager_ResolveToSHA(t *testing.T) {
	rm, _, cleanup := setupTestRepo(t)
	defer cleanup()

	if err := rm.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	sha := "1234567890abcdef1234567890abcdef12345678"

	tests := []struct {
		name        string
		setupRefs   map[RefPath]string
		resolveRef  RefPath
		expectedSHA string
		expectError bool
	}{
		{
			name: "direct SHA reference",
			setupRefs: map[RefPath]string{
				"refs/heads/main": sha,
			},
			resolveRef:  "refs/heads/main",
			expectedSHA: sha,
			expectError: false,
		},
		{
			name: "symbolic reference",
			setupRefs: map[RefPath]string{
				"HEAD":            "ref: refs/heads/main",
				"refs/heads/main": sha,
			},
			resolveRef:  "HEAD",
			expectedSHA: sha,
			expectError: false,
		},
		{
			name: "chained symbolic references",
			setupRefs: map[RefPath]string{
				"refs/heads/alias": "ref: refs/heads/main",
				"refs/heads/main":  sha,
			},
			resolveRef:  "refs/heads/alias",
			expectedSHA: sha,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup refs
			for ref, content := range tt.setupRefs {
				fullPath := rm.resolveReferencePath(ref)
				if err := os.MkdirAll(filepath.Dir(fullPath.String()), 0755); err != nil {
					t.Fatalf("Failed to create ref dir: %v", err)
				}
				if err := os.WriteFile(fullPath.String(), []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write ref: %v", err)
				}
			}

			// Resolve
			result, err := rm.ResolveToSHA(tt.resolveRef)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveToSHA failed: %v", err)
			}

			if result.String() != tt.expectedSHA {
				t.Errorf("ResolveToSHA = %q, want %q", result, tt.expectedSHA)
			}
		})
	}
}

func TestRefManager_DeleteRef(t *testing.T) {
	rm, _, cleanup := setupTestRepo(t)
	defer cleanup()

	if err := rm.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	ref := RefPath("refs/heads/test")
	sha := "1234567890abcdef1234567890abcdef12345678"

	// Create ref
	hash, err := objects.NewObjectHashFromString(sha)
	if err != nil {
		t.Fatalf("Failed to create hash: %v", err)
	}

	if err := rm.UpdateRef(ref, hash); err != nil {
		t.Fatalf("UpdateRef failed: %v", err)
	}

	// Delete it
	deleted, err := rm.DeleteRef(ref)
	if err != nil {
		t.Fatalf("DeleteRef failed: %v", err)
	}

	if !deleted {
		t.Error("Expected ref to be deleted")
	}

	// Verify it's gone
	exists, err := rm.Exists(ref)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if exists {
		t.Error("Ref should not exist after deletion")
	}
}

func TestRefManager_DeleteRef_NonExistent(t *testing.T) {
	rm, _, cleanup := setupTestRepo(t)
	defer cleanup()

	if err := rm.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	deleted, err := rm.DeleteRef("refs/heads/nonexistent")
	if err != nil {
		t.Fatalf("DeleteRef failed: %v", err)
	}

	if deleted {
		t.Error("Expected deleted to be false for non-existent ref")
	}
}

func TestRefManager_Exists(t *testing.T) {
	rm, _, cleanup := setupTestRepo(t)
	defer cleanup()

	if err := rm.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	ref := RefPath("refs/heads/test")
	sha := "1234567890abcdef1234567890abcdef12345678"

	// Check non-existent ref
	exists, err := rm.Exists(ref)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Ref should not exist")
	}

	// Create ref
	hash, err := objects.NewObjectHashFromString(sha)
	if err != nil {
		t.Fatalf("Failed to create hash: %v", err)
	}

	if err := rm.UpdateRef(ref, hash); err != nil {
		t.Fatalf("UpdateRef failed: %v", err)
	}

	// Check existing ref
	exists, err = rm.Exists(ref)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Ref should exist")
	}
}

func TestRefManager_GetPaths(t *testing.T) {
	rm, tempDir, cleanup := setupTestRepo(t)
	defer cleanup()

	expectedRefsPath := filepath.Join(tempDir, scpath.SourceDir, scpath.RefsDir)
	expectedHeadPath := filepath.Join(tempDir, scpath.SourceDir, scpath.HeadFile)

	refsPath := rm.GetRefsPath()
	if refsPath.String() != expectedRefsPath {
		t.Errorf("GetRefsPath = %q, want %q", refsPath, expectedRefsPath)
	}

	headPath := rm.GetHeadPath()
	if headPath.String() != expectedHeadPath {
		t.Errorf("GetHeadPath = %q, want %q", headPath, expectedHeadPath)
	}
}

func TestRefManager_ResolveReferencePath(t *testing.T) {
	rm, tempDir, cleanup := setupTestRepo(t)
	defer cleanup()

	tests := []struct {
		name     string
		ref      RefPath
		expected string
	}{
		{
			name:     "HEAD reference",
			ref:      "HEAD",
			expected: filepath.Join(tempDir, scpath.SourceDir, scpath.HeadFile),
		},
		{
			name:     "branch reference with refs prefix",
			ref:      "refs/heads/main",
			expected: filepath.Join(tempDir, scpath.SourceDir, scpath.RefsDir, "heads", "main"),
		},
		{
			name:     "tag reference with refs prefix",
			ref:      "refs/tags/v1.0.0",
			expected: filepath.Join(tempDir, scpath.SourceDir, scpath.RefsDir, "tags", "v1.0.0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rm.resolveReferencePath(tt.ref)
			if result.String() != tt.expected {
				t.Errorf("resolveReferencePath(%q) = %q, want %q", tt.ref, result, tt.expected)
			}
		})
	}
}
