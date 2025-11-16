package scpath

import (
	"fmt"
	"path/filepath"
	"strings"
)

// String returns the path as a string
func (rp RelativePath) String() string {
	return string(rp)
}

// IsValid checks if this is a valid relative path
func (rp RelativePath) IsValid() bool {
	s := string(rp)
	if len(s) == 0 {
		return false
	}

	if filepath.IsAbs(s) || strings.HasPrefix(s, "/") {
		return false
	}

	if strings.Contains(s, "..") {
		return false
	}
	return true
}

// Normalize normalizes the path (converts to forward slashes, cleans)
func (rp RelativePath) Normalize() RelativePath {
	// Convert to forward slashes (Git convention)
	normalized := filepath.ToSlash(filepath.Clean(string(rp)))
	// Remove leading ./
	normalized = strings.TrimPrefix(normalized, "./")
	return RelativePath(normalized)
}

// Components returns the path components
func (rp RelativePath) Components() []string {
	normalized := rp.Normalize()
	if normalized == "" || normalized == "." {
		return []string{}
	}
	return strings.Split(string(normalized), "/")
}

// Join joins path elements to create a new relative path
func (rp RelativePath) Join(elem ...string) RelativePath {
	parts := append([]string{string(rp)}, elem...)
	joined := filepath.Join(parts...)
	return RelativePath(joined).Normalize()
}

// Base returns the last element of the path
func (rp RelativePath) Base() string {
	normalized := rp.Normalize()
	components := normalized.Components()
	if len(components) == 0 {
		return ""
	}
	return components[len(components)-1]
}

// Dir returns all but the last element of the path
func (rp RelativePath) Dir() RelativePath {
	normalized := rp.Normalize()
	components := normalized.Components()
	if len(components) <= 1 {
		return ""
	}
	return RelativePath(strings.Join(components[:len(components)-1], "/"))
}

// IsInSubdir checks if the path is within the given subdirectory
func (rp RelativePath) IsInSubdir(subdir string) bool {
	normalized := rp.Normalize()
	return strings.HasPrefix(string(normalized), subdir+"/") || string(normalized) == subdir
}

// Depth returns the directory depth (number of path components)
func (rp RelativePath) Depth() int {
	return len(rp.Components())
}

// NewRelativePath creates and validates a new RelativePath
func NewRelativePath(path string) (RelativePath, error) {
	rp := RelativePath(path).Normalize()
	if !rp.IsValid() {
		return "", fmt.Errorf("invalid relative path: %s", path)
	}
	return rp, nil
}
