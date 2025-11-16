package fileops

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// AtomicWrite writes data to a file atomically by using a temporary file and rename.
// This ensures that the file is never in a partial state.
func AtomicWrite(targetPath scpath.AbsolutePath, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(targetPath.String())
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	if err := writeTempFile(data, tmpFile); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	return renameTempFile(tmpFile.Name(), targetPath.String(), mode)
}

// writeTempFile writes the provided data to the supplied temporary file,
// synchronizes it to underlying storage using fsync, and then closes the file.
// It returns any encountered error wrapped with context.
func writeTempFile(data []byte, tmpFile *os.File) error {
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return nil
}

// renameTempFile atomically replaces the target file with the temp file.
// It applies the correct file mode to the temporary file before renaming,
// ensuring file permissions are maintained as expected. Returns an error
// if either the chmod or the rename fails.
func renameTempFile(tmpPath string, targetPath string, mode os.FileMode) error {
	if err := os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}
