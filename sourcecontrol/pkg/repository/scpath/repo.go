package scpath

import (
	"fmt"
	"path/filepath"
	"strings"
)

// String returns the path as a string
func (rp RepositoryPath) String() string {
	return string(rp)
}

// IsValid checks if this is a valid absolute path
func (rp RepositoryPath) IsValid() bool {
	return filepath.IsAbs(string(rp))
}

// Join joins path elements to the repository path
func (rp RepositoryPath) Join(elem ...string) AbsolutePath {
	parts := append([]string{string(rp)}, elem...)
	return AbsolutePath(filepath.Join(parts...))
}

// JoinRelative safely joins a relative path to the repository path with validation
// This method ensures the resulting path stays within the repository directory
// The RelativePath type guarantees the input is already normalized and validated
func (rp RepositoryPath) JoinRelative(relPath RelativePath) (AbsolutePath, error) {
	if !relPath.IsValid() {
		return "", fmt.Errorf("invalid relative path: %s", relPath)
	}

	normalized := relPath.Normalize()
	if normalized == "" || normalized == "." {
		return AbsolutePath(rp), nil
	}

	result := filepath.Join(string(rp), string(normalized))
	absResult := AbsolutePath(result)

	relCheck, err := filepath.Rel(string(rp), string(absResult))
	if err != nil {
		return "", fmt.Errorf("failed to validate path: %w", err)
	}

	if filepath.IsAbs(relCheck) || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repository: %s", relPath)
	}

	return absResult, nil
}

// SourcePath returns the path to the .source directory
func (rp RepositoryPath) SourcePath() SourcePath {
	return SourcePath(filepath.Join(string(rp), SourceDir))
}

// NewRepositoryPath creates a new RepositoryPath from a string
func NewRepositoryPath(path string) (RepositoryPath, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	return RepositoryPath(absPath), nil
}
