package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// LockFile represents a file-based lock for repository operations.
// It prevents concurrent modifications to the working directory.
type LockFile struct {
	path string
	file *os.File
}

// AcquireLock attempts to acquire an exclusive lock on the index.
// Returns an error if another process already holds the lock.
func AcquireLock(sourceDir scpath.SourcePath) (*LockFile, error) {
	lockPath := filepath.Join(sourceDir.String(), "index.lock")

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("lock error (%s): another process holds the lock: %w", lockPath, ErrLockAcquisitionFailed)
		}
		return nil, fmt.Errorf("lock error (%s): failed to create lock file: %w", lockPath, err)
	}

	return &LockFile{
		path: lockPath,
		file: file,
	}, nil
}

// Release releases the lock by closing and deleting the lock file
func (l *LockFile) Release() error {
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("close lock file: %w", err)
	}

	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove lock file: %w", err)
	}

	return nil
}

// Path returns the lock file path
func (l *LockFile) Path() string {
	return l.path
}
