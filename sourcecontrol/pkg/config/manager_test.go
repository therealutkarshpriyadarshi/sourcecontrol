package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

func TestManager_Hierarchy(t *testing.T) {
	// Create temporary directory for test configs
	tmpDir := t.TempDir()

	// Create a manager with repository path
	manager := NewManager(scpath.RepositoryPath(tmpDir))

	// Set values at different levels
	manager.SetCommandLine("test.key", "command-line-value")

	// The command line should have highest precedence
	entry := manager.Get("test.key")
	if entry == nil {
		t.Fatal("Get() returned nil")
	}
	if entry.Value != "command-line-value" {
		t.Errorf("Get() = %q, want %q", entry.Value, "command-line-value")
	}
	if entry.Level != CommandLineLevel {
		t.Errorf("Get() level = %v, want %v", entry.Level, CommandLineLevel)
	}
}

func TestManager_BuiltinDefaults(t *testing.T) {
	manager := NewManager(scpath.RepositoryPath(""))

	tests := []struct {
		key   string
		value string
	}{
		{"core.repositoryformatversion", "0"},
		{"core.filemode", "true"},
		{"core.bare", "false"},
		{"init.defaultbranch", "main"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			entry := manager.Get(tt.key)
			if entry == nil {
				t.Fatalf("Get(%q) returned nil", tt.key)
			}
			if entry.Value != tt.value {
				t.Errorf("Get(%q) = %q, want %q", tt.key, entry.Value, tt.value)
			}
			if entry.Level != BuiltinLevel {
				t.Errorf("Get(%q) level = %v, want %v", tt.key, entry.Level, BuiltinLevel)
			}
		})
	}
}

func TestManager_GetAll(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(scpath.RepositoryPath(tmpDir))

	// Add multiple values for a key
	if err := manager.Add("remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*", UserLevel); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := manager.Add("remote.origin.fetch", "+refs/tags/*:refs/tags/*", UserLevel); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Get all values
	entries := manager.GetAll("remote.origin.fetch")
	if len(entries) != 2 {
		t.Errorf("GetAll() returned %d entries, want 2", len(entries))
	}

	expectedValues := []string{
		"+refs/heads/*:refs/remotes/origin/*",
		"+refs/tags/*:refs/tags/*",
	}

	for i, entry := range entries {
		if entry.Value != expectedValues[i] {
			t.Errorf("GetAll()[%d] = %q, want %q", i, entry.Value, expectedValues[i])
		}
	}
}

func TestManager_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(scpath.RepositoryPath(tmpDir))

	// Set a value
	if err := manager.Set("user.name", "John Doe", UserLevel); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get the value
	entry := manager.Get("user.name")
	if entry == nil {
		t.Fatal("Get() returned nil")
	}
	if entry.Value != "John Doe" {
		t.Errorf("Get() = %q, want %q", entry.Value, "John Doe")
	}
	if entry.Level != UserLevel {
		t.Errorf("Get() level = %v, want %v", entry.Level, UserLevel)
	}
}

func TestManager_Unset(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(scpath.RepositoryPath(tmpDir))

	// Set a value
	if err := manager.Set("test.key", "test-value", UserLevel); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify it exists
	if entry := manager.Get("test.key"); entry == nil {
		t.Fatal("Get() returned nil after Set()")
	}

	// Unset the value
	if err := manager.Unset("test.key", UserLevel); err != nil {
		t.Fatalf("Unset() error = %v", err)
	}

	// Verify it's gone
	if entry := manager.Get("test.key"); entry != nil {
		t.Errorf("Get() = %v after Unset(), want nil", entry)
	}
}

func TestManager_List(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(scpath.RepositoryPath(tmpDir))

	// Set some values
	if err := manager.Set("user.name", "John Doe", UserLevel); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := manager.Set("user.email", "john@example.com", UserLevel); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// List all entries
	entries := manager.List()

	// Should include user-set values plus builtin defaults
	if len(entries) < 2 {
		t.Errorf("List() returned %d entries, want at least 2", len(entries))
	}

	// Check that our values are included
	found := make(map[string]bool)
	for _, entry := range entries {
		if entry.Key == "user.name" && entry.Value == "John Doe" {
			found["user.name"] = true
		}
		if entry.Key == "user.email" && entry.Value == "john@example.com" {
			found["user.email"] = true
		}
	}

	if !found["user.name"] {
		t.Error("List() missing user.name")
	}
	if !found["user.email"] {
		t.Error("List() missing user.email")
	}
}

func TestManager_Load(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a config file
	configPath := filepath.Join(tmpDir, "config.json")
	configContent := `{
		"user": {
			"name": "Test User",
			"email": "test@example.com"
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create manager pointing to the directory
	manager := NewManager(scpath.RepositoryPath(tmpDir))

	// Load configs
	if err := manager.Load(context.Background()); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify values were loaded
	if entry := manager.Get("user.name"); entry == nil || entry.Value != "Test User" {
		t.Errorf("Get(user.name) = %v, want Test User", entry)
	}
	if entry := manager.Get("user.email"); entry == nil || entry.Value != "test@example.com" {
		t.Errorf("Get(user.email) = %v, want test@example.com", entry)
	}
}

func TestManager_ExportJSON(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(scpath.RepositoryPath(tmpDir))

	// Set some values
	if err := manager.Set("user.name", "John Doe", UserLevel); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := manager.Set("user.email", "john@example.com", UserLevel); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Export as JSON
	json, err := manager.ExportJSON(nil)
	if err != nil {
		t.Fatalf("ExportJSON() error = %v", err)
	}

	// Verify it's valid JSON
	parser := &Parser{}
	validation := parser.Validate(json)
	if !validation.Valid {
		t.Errorf("ExportJSON() produced invalid JSON: %v", validation.Errors)
	}

	// Parse and verify content
	entries, err := parser.Parse(json, ConfigSource("test"), UserLevel)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if userEntries, exists := entries["user.name"]; !exists || len(userEntries) == 0 || userEntries[0].Value != "John Doe" {
		t.Error("ExportJSON() missing or incorrect user.name")
	}
	if emailEntries, exists := entries["user.email"]; !exists || len(emailEntries) == 0 || emailEntries[0].Value != "john@example.com" {
		t.Error("ExportJSON() missing or incorrect user.email")
	}
}

func TestManager_ReadOnlyLevels(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(scpath.RepositoryPath(tmpDir))

	// Try to set at command-line level (read-only)
	err := manager.Set("test.key", "value", CommandLineLevel)
	if err == nil {
		t.Error("Set() at CommandLineLevel should fail, but succeeded")
	}
	if !IsReadOnly(err) {
		t.Errorf("Set() at CommandLineLevel error = %v, want ErrReadOnly", err)
	}

	// Try to set at builtin level (read-only)
	err = manager.Set("test.key", "value", BuiltinLevel)
	if err == nil {
		t.Error("Set() at BuiltinLevel should fail, but succeeded")
	}
}

func TestManager_ThreadSafety(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(scpath.RepositoryPath(tmpDir))

	// Run concurrent operations
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = manager.Set("test.key", "value", UserLevel)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = manager.Get("test.key")
		}
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done

	// If we get here without a race condition, the test passes
}
