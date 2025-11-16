package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// TestHelper provides utilities for CLI command testing
type TestHelper struct {
	t        *testing.T
	tempDir  string
	repo     *sourcerepo.SourceRepository
	RepoPath string
}

// NewTestHelper creates a new test helper with automatic cleanup
func NewTestHelper(t *testing.T) *TestHelper {
	t.Helper()

	// Create temp directory that will be auto-cleaned
	tempDir := t.TempDir()

	return &TestHelper{
		t:        t,
		tempDir:  tempDir,
		RepoPath: tempDir,
	}
}

// InitRepo initializes a test repository
func (th *TestHelper) InitRepo() *sourcerepo.SourceRepository {
	th.t.Helper()

	repoPath, err := scpath.NewRepositoryPath(th.tempDir)
	if err != nil {
		th.t.Fatalf("failed to create repo path: %v", err)
	}

	repo := sourcerepo.NewSourceRepository()
	if err := repo.Initialize(repoPath); err != nil {
		th.t.Fatalf("failed to initialize repo: %v", err)
	}

	th.repo = repo
	return repo
}

// TempDir returns the temporary directory path
func (th *TestHelper) TempDir() string {
	return th.tempDir
}

// WriteFile creates a test file with content
func (th *TestHelper) WriteFile(name, content string) string {
	th.t.Helper()

	filePath := filepath.Join(th.tempDir, name)

	// Create parent directories if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		th.t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		th.t.Fatalf("failed to write file %s: %v", filePath, err)
	}

	return filePath
}

// Chdir changes to the test directory
func (th *TestHelper) Chdir() {
	th.t.Helper()

	if err := os.Chdir(th.tempDir); err != nil {
		th.t.Fatalf("failed to chdir to %s: %v", th.tempDir, err)
	}
}

// Repo returns the initialized repository (must call InitRepo first)
func (th *TestHelper) Repo() *sourcerepo.SourceRepository {
	if th.repo == nil {
		th.t.Fatal("repository not initialized, call InitRepo() first")
	}
	return th.repo
}
