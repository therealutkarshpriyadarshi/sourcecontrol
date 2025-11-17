package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/stash"
)

func TestStashCommands(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "stash-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a repository
	repo, err := sourcerepo.InitRepository(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	stashMgr := stash.NewManager(repo)
	if err := stashMgr.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize stash manager: %v", err)
	}

	t.Run("List empty stash", func(t *testing.T) {
		entries, err := stashMgr.List()
		if err != nil {
			t.Errorf("Failed to list stashes: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("Expected 0 stashes, got %d", len(entries))
		}
	})

	t.Run("ParseStashName", func(t *testing.T) {
		index, err := stash.ParseStashName("stash@{0}")
		if err != nil {
			t.Errorf("Failed to parse stash name: %v", err)
		}
		if index != 0 {
			t.Errorf("Expected index 0, got %d", index)
		}
	})

	t.Run("GetStashName", func(t *testing.T) {
		name := stash.GetStashName(0)
		if name != "stash@{0}" {
			t.Errorf("Expected 'stash@{0}', got '%s'", name)
		}
	})
}
