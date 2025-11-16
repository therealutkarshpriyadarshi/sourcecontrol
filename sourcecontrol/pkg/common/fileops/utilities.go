package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// Exists checks if a file or directory exists at the given path.
// Returns true if the path exists, false if it doesn't exist.
// Returns an error only if there's a filesystem error other than non-existence.
func Exists(p scpath.AbsolutePath) (bool, error) {
	_, err := os.Stat(p.String())
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("check existence: %w", err)
}

// ExistsString checks if a file or directory exists at the given string path.
// This is a convenience function for when you're working with string paths.
func ExistsString(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("check existence: %w", err)
}

// EnsureDir ensures that a directory exists, creating it and any necessary
// parent directories if they don't exist.
// Returns nil if the directory already exists or was created successfully.
func EnsureDir(path scpath.AbsolutePath) error {
	if err := os.MkdirAll(path.String(), 0755); err != nil {
		return fmt.Errorf("ensure directory %s: %w", path.String(), err)
	}
	return nil
}

// EnsureDirString ensures that a directory exists using a string path.
func EnsureDirString(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("ensure directory %s: %w", path, err)
	}
	return nil
}

// EnsureParentDir ensures that the parent directory of a file exists.
// This is useful before creating or writing to a file.
func EnsureParentDir(p scpath.AbsolutePath) error {
	dir := filepath.Dir(p.String())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ensure parent directory: %w", err)
	}
	return nil
}

// ReadString reads a file and returns its content as a trimmed string.
// If the file doesn't exist, returns an empty string and nil error.
// This is useful for optional configuration files.
func ReadString(p scpath.AbsolutePath) (string, error) {
	data, err := os.ReadFile(p.String())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// ReadStringStrict reads a file and returns its content as a trimmed string.
// Returns an error if the file doesn't exist.
// Use this when the file is required to exist.
func ReadStringStrict(p scpath.AbsolutePath) (string, error) {
	data, err := os.ReadFile(p.String())
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// ReadBytes reads a file and returns its raw bytes.
// If the file doesn't exist, returns nil and nil error.
// This is useful for optional files where you need the raw content.
func ReadBytes(p scpath.AbsolutePath) ([]byte, error) {
	data, err := os.ReadFile(p.String())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read file: %w", err)
	}
	return data, nil
}

// ReadBytesStrict reads a file and returns its raw bytes.
// Returns an error if the file doesn't exist.
// Use this when the file is required to exist.
func ReadBytesStrict(p scpath.AbsolutePath) ([]byte, error) {
	data, err := os.ReadFile(p.String())
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return data, nil
}

// WriteConfig writes data to a configuration file with 0644 permissions.
// This permission allows the owner to read/write and others to read.
// Ensures the parent directory exists before writing.
func WriteConfig(p scpath.AbsolutePath, data []byte) error {
	if err := EnsureParentDir(p); err != nil {
		return err
	}
	if err := os.WriteFile(p.String(), data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

// WriteReadOnly writes data to a read-only file with 0444 permissions.
// This permission makes the file read-only for everyone.
// Useful for immutable objects like Git blobs, trees, and commits.
// Ensures the parent directory exists before writing.
func WriteReadOnly(p scpath.AbsolutePath, data []byte) error {
	if err := EnsureParentDir(p); err != nil {
		return err
	}
	if err := os.WriteFile(p.String(), data, 0444); err != nil {
		return fmt.Errorf("write read-only file: %w", err)
	}
	return nil
}

// WriteConfigString writes string content to a configuration file.
// This is a convenience wrapper around WriteConfig for string content.
func WriteConfigString(p scpath.AbsolutePath, content string) error {
	return WriteConfig(p, []byte(content))
}

// SafeRemove removes a file if it exists.
// Returns nil if the file doesn't exist (not considered an error).
// Returns an error only if the removal fails for reasons other than non-existence.
func SafeRemove(p scpath.AbsolutePath) error {
	if err := os.Remove(p.String()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

// SafeRemoveString removes a file using a string path.
func SafeRemoveString(p string) error {
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

// IsDirectory checks if the path exists and is a directory.
// Returns false if the path doesn't exist or is not a directory.
func IsDirectory(p scpath.AbsolutePath) (bool, error) {
	info, err := os.Stat(p.String())
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat path: %w", err)
	}
	return info.IsDir(), nil
}

// IsFile checks if the path exists and is a regular file (not a directory).
// Returns false if the path doesn't exist or is a directory.
func IsFile(p scpath.AbsolutePath) (bool, error) {
	info, err := os.Stat(p.String())
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat path: %w", err)
	}
	return !info.IsDir(), nil
}
