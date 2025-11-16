package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/common/fileops"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// FileOps implements the FileOperator interface for low-level file system operations.
// It handles creating, modifying, and deleting files in the working directory.
type FileOps struct {
	repo    *sourcerepo.SourceRepository
	workDir scpath.RepositoryPath
	tempDir scpath.AbsolutePath // Directory for temporary files
	dryRun  bool                // If true, operations are simulated
}

// NewFileOps creates a new FileOps service
func NewFileOps(repo *sourcerepo.SourceRepository) *FileOps {
	workDir := repo.WorkingDirectory()
	tempDir := workDir.Join(scpath.SourceDir, "tmp")
	return &FileOps{
		repo:    repo,
		workDir: workDir,
		tempDir: tempDir,
		dryRun:  false,
	}
}

// SetDryRun enables or disables dry-run mode
func (f *FileOps) SetDryRun(enabled bool) {
	f.dryRun = enabled
}

// ApplyOperation executes a single file operation (create, modify, or delete).
// Returns a WorkdirError if the operation fails.
func (f *FileOps) ApplyOperation(op Operation) error {
	if f.dryRun {
		return nil // In dry-run mode, don't actually perform operations
	}

	switch op.Action {
	case ActionCreate, ActionModify:
		return f.writeFile(op)
	case ActionDelete:
		return f.deleteFile(op.Path)
	default:
		return fmt.Errorf("apply %s: %w: unknown action %v", op.Path, ErrInvalidOperation, op.Action)
	}
}

// writeFile creates or modifies a file by reading content from a blob object.
// Uses atomic write pattern: write to temp file, then rename.
func (f *FileOps) writeFile(op Operation) error {
	content, err := f.readBlobContent(op)
	if err != nil {
		return fmt.Errorf("%s %s: get blob content: %w", op.Action.String(), op.Path, err)
	}

	fullPath := f.workDir.Join(op.Path.String())
	if err := fileops.EnsureParentDir(fullPath); err != nil {
		return fmt.Errorf("%s %s: create parent directory: %w", op.Action.String(), op.Path, err)
	}

	if err := f.atomicWrite(fullPath, content, op.Mode.ToOSFileMode()); err != nil {
		return fmt.Errorf("%s %s: write file: %w", op.Action.String(), op.Path, err)
	}

	return nil
}

func (f *FileOps) readBlobContent(op Operation) ([]byte, error) {
	if op.SHA == "" {
		return nil, fmt.Errorf("%s %s: %w: missing SHA", op.Action.String(), op.Path, ErrInvalidOperation)
	}

	blob, err := f.repo.ReadBlobObject(op.SHA)
	if err != nil {
		return nil, fmt.Errorf("%s %s: object %s is not a blob", op.Action.String(), op.Path, op.SHA.Short())
	}

	content, err := blob.Content()
	if err != nil {
		return nil, fmt.Errorf("%s %s: get blob content: %w", op.Action.String(), op.Path, err)
	}

	return content.Bytes(), nil
}

// atomicWrite writes data to a file atomically by using a temporary file and rename.
// This ensures that the file is never in a partial state.
func (f *FileOps) atomicWrite(targetPath scpath.AbsolutePath, data []byte, mode os.FileMode) error {
	if err := f.ensureTempDir(); err != nil {
		return err
	}

	return fileops.AtomicWrite(targetPath, data, mode)
}

// deleteFile removes a file from the working directory and cleans up empty parent directories
func (f *FileOps) deleteFile(path scpath.RelativePath) error {
	fullPath := f.workDir.Join(path.String())

	exists, err := fileops.Exists(fullPath)
	if err != nil {
		return fmt.Errorf("delete %s: check existence: %w", path, err)
	}
	if !exists {
		return nil
	}

	if err := fileops.SafeRemove(fullPath); err != nil {
		return fmt.Errorf("delete %s: remove file: %w", path, err)
	}

	parentDir := fullPath.Dir()
	if err := f.cleanEmptyParents(parentDir); err != nil {
		_ = err
	}

	return nil
}

// cleanEmptyParents recursively removes empty directories up to the working directory root
func (f *FileOps) cleanEmptyParents(dir scpath.AbsolutePath) error {
	rel, err := filepath.Rel(f.workDir.String(), dir.String())
	if err != nil {
		return nil
	}
	// If relative path starts with "..", dir is outside workDir
	// Also stop if dir equals workDir
	if strings.HasPrefix(rel, "..") || rel == "." || rel == "" {
		return nil
	}

	entries, err := os.ReadDir(dir.String())
	if err != nil {
		return err
	}

	if len(entries) > 0 {
		return nil
	}

	if err := os.Remove(dir.String()); err != nil {
		return err
	}

	parentDir := dir.Dir()
	return f.cleanEmptyParents(parentDir)
}

// CreateBackup creates a backup of a file before modification.
// Returns a Backup struct that can be used to restore the file later.
func (f *FileOps) CreateBackup(path scpath.RelativePath) (*Backup, error) {
	fullPath := f.workDir.Join(path.String())

	info, err := os.Stat(fullPath.String())
	if os.IsNotExist(err) {
		return &Backup{
			Path:     path,
			TempFile: "",
			Existed:  false,
			Mode:     0,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("backup %s: stat file: %w", path, err)
	}

	tmpFile, err := f.createTempBackupFile("backup-*")
	if err != nil {
		return nil, err
	}

	success := false
	defer func() {
		tmpFile.Close()
		if !success {
			os.Remove(tmpFile.Name())
		}
	}()

	if err := writeToTemp(tmpFile, path, fullPath); err != nil {
		return nil, err
	}

	success = true
	return &Backup{
		Path:     path,
		TempFile: tmpFile.Name(),
		Existed:  true,
		Mode:     objects.FromOSFileMode(info.Mode()),
	}, nil
}

func (f *FileOps) createTempBackupFile(pattern string) (*os.File, error) {
	if err := f.ensureTempDir(); err != nil {
		return nil, err
	}

	tmpFile, err := os.CreateTemp(f.tempDir.String(), pattern)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	return tmpFile, nil
}

func (f *FileOps) ensureTempDir() error {
	return fileops.EnsureDir(f.tempDir)
}

func writeToTemp(tmpFile *os.File, path scpath.RelativePath, fullPath scpath.AbsolutePath) error {
	srcFile, err := os.Open(fullPath.String())
	if err != nil {
		return fmt.Errorf("backup %s: open source: %w", path, err)
	}
	defer srcFile.Close()

	if _, err := io.Copy(tmpFile, srcFile); err != nil {
		return fmt.Errorf("backup %s: copy content: %w", path, err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("backup %s: sync backup: %w", path, err)
	}

	return nil
}

// RestoreBackup restores a file from a backup
func (f *FileOps) RestoreBackup(backup *Backup) error {
	if backup == nil {
		return fmt.Errorf("nil backup")
	}

	fullPath := f.workDir.Join(backup.Path.String())

	if !backup.Existed {
		if err := fileops.SafeRemove(fullPath); err != nil {
			return fmt.Errorf("restore %s: remove file: %w", backup.Path, err)
		}
		return nil
	}

	if backup.TempFile == "" {
		return fmt.Errorf("restore %s: backup has no temp file", backup.Path)
	}

	if err := fileops.EnsureParentDir(fullPath); err != nil {
		return fmt.Errorf("restore %s: create parent directory: %w", backup.Path, err)
	}

	return f.writeFromBackup(backup)
}

func (f *FileOps) writeFromBackup(backup *Backup) error {
	backupPath := f.workDir.Join(backup.Path.String())
	srcFile, err := os.Open(backup.TempFile)
	if err != nil {
		return fmt.Errorf("restore %s: open backup: %w", backup.Path, err)
	}
	defer srcFile.Close()

	data, err := io.ReadAll(srcFile)
	if err != nil {
		return fmt.Errorf("restore %s: read backup: %w", backup.Path, err)
	}

	if err := f.atomicWrite(backupPath, data, backup.Mode.ToOSFileMode()); err != nil {
		return fmt.Errorf("restore %s: write file: %w", backup.Path, err)
	}

	return nil
}

// CleanupBackup removes a backup file after successful operation
func (f *FileOps) CleanupBackup(backup *Backup) error {
	if backup == nil || backup.TempFile == "" {
		return nil
	}

	if err := fileops.SafeRemoveString(backup.TempFile); err != nil {
		return fmt.Errorf("remove backup file: %w", err)
	}

	return nil
}
