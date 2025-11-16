package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

func TestInitCommand(t *testing.T) {
	// Save and restore current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Create test helper - automatically cleans up after test
	th := NewTestHelper(t)
	th.Chdir()

	// Run init command
	cmd := newInitCmd()
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify .source directory was created
	sourceDir := filepath.Join(th.TempDir(), scpath.SourceDir)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		t.Error(".source directory was not created")
	}

	// Verify HEAD file exists
	headFile := filepath.Join(sourceDir, "HEAD")
	if _, err := os.Stat(headFile); os.IsNotExist(err) {
		t.Error("HEAD file was not created")
	}

	// Verify config file exists
	configFile := filepath.Join(sourceDir, "config")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Temp directory will be automatically cleaned up by t.TempDir()
}

func TestInitCommandWithExistingRepo(t *testing.T) {
	// Save and restore current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	th := NewTestHelper(t)
	th.Chdir()

	// Initialize first time
	cmd1 := newInitCmd()
	cmd1.SetArgs([]string{})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Try to initialize again - should fail
	cmd2 := newInitCmd()
	cmd2.SetArgs([]string{})
	err = cmd2.Execute()

	if err == nil {
		t.Error("expected error when reinitializing repository, got nil")
	}
}
