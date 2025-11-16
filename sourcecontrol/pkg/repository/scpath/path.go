package scpath

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RepositoryPath represents an absolute path to a repository root directory
// Example: "/home/user/myproject" or "C:\Users\user\myproject"
type RepositoryPath string

// AbsolutePath represents any absolute path in the repository filesystem
// This can be used for paths anywhere in the repository structure
type AbsolutePath string

// SourcePath represents a path within the .git directory (Git metadata)
// Example: "/repo/.git" or "/repo/.git/refs/heads"
type SourcePath string

// RelativePath represents a normalized relative path (forward slashes, no ..)
// Example: "src/main.go" or "docs/README.md"
type RelativePath string

// String returns the path as a string
func (ap AbsolutePath) String() string {
	return string(ap)
}

// IsValid checks if this is a valid absolute path
func (ap AbsolutePath) IsValid() bool {
	return len(ap) > 0 && filepath.IsAbs(string(ap))
}

// Join joins path elements to the absolute path
func (ap AbsolutePath) Join(elem ...string) AbsolutePath {
	parts := append([]string{string(ap)}, elem...)
	return AbsolutePath(filepath.Join(parts...))
}

// RelativeTo returns a relative path from the base path
func (ap AbsolutePath) RelativeTo(base RepositoryPath) (RelativePath, error) {
	rel, err := filepath.Rel(string(base), string(ap))
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}
	return RelativePath(rel).Normalize(), nil
}

// Base returns the last element of the path
func (ap AbsolutePath) Base() string {
	return filepath.Base(string(ap))
}

// Dir returns all but the last element of the path
func (ap AbsolutePath) Dir() AbsolutePath {
	return AbsolutePath(filepath.Dir(string(ap)))
}

// NewAbsolutePath creates a new AbsolutePath from a string
// If the path is relative, it converts it to absolute using the current working directory
func NewAbsolutePath(path string) (AbsolutePath, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return AbsolutePath(absPath), nil
}

// SanitizePath sanitizes a path for use in Git
func SanitizePath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.Trim(path, "/")
	return path
}

// IsPathSafe checks if a path is safe (no directory traversal, etc.)
func IsPathSafe(path string) bool {
	if strings.Contains(path, "..") {
		return false
	}
	// Cannot be absolute (check both Unix and Windows style)
	if filepath.IsAbs(path) || strings.HasPrefix(path, "/") {
		return false
	}
	// Cannot contain backslashes (use forward slashes)
	if strings.Contains(path, "\\") {
		return false
	}
	return true
}

// NormalizePath normalizes a path for Git (forward slashes, no trailing slash)
func NormalizePath(path string) string {
	// Convert to forward slashes
	path = filepath.ToSlash(filepath.Clean(path))
	// Remove leading ./
	path = strings.TrimPrefix(path, "./")
	// Remove trailing slash (except for root)
	if len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

// JoinPaths joins multiple path segments using forward slashes
func JoinPaths(paths ...string) string {
	return NormalizePath(filepath.Join(paths...))
}
