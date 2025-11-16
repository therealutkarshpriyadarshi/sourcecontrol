package internal

import (
	"context"
	"errors"
	"fmt"

	pool "github.com/utkarsh5026/SourceControl/pkg/common/concurrency"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

var (
	// ErrInvalidOperation is returned when an operation is malformed
	ErrInvalidOperation = errors.New("invalid operation")
	// ErrLockAcquisitionFailed is returned when unable to acquire repository lock
	ErrLockAcquisitionFailed = errors.New("failed to acquire lock")
)

// DryRunAnalysis categorizes operations for dry run reporting
type DryRunAnalysis struct {
	WillCreate []scpath.RelativePath
	WillModify []scpath.RelativePath
	WillDelete []scpath.RelativePath
	Conflicts  []string
}

// DryRunResult contains the analysis of operations without executing them
type DryRunResult struct {
	Valid     bool
	Analysis  DryRunAnalysis
	Conflicts []string
	Errors    []string
}

// TransactionResult contains the outcome of an atomic transaction
type TransactionResult struct {
	Success           bool
	OperationsApplied int
	TotalOperations   int
	Err               error
}

func newTr(success bool, opsApplied, totalOps int, err error) TransactionResult {
	return TransactionResult{
		Success:           success,
		OperationsApplied: opsApplied,
		TotalOperations:   totalOps,
		Err:               err,
	}
}

func success(opsApplied, totalOps int) TransactionResult {
	return newTr(true, opsApplied, totalOps, nil)
}

func failure(opsApplied, totalOps int, err error) TransactionResult {
	return newTr(false, opsApplied, totalOps, err)
}

// Manager implements atomic transaction execution with rollback support.
// It ensures all-or-nothing semantics for file operations.
type Manager struct {
	fileOps   *FileOps
	sourceDir scpath.SourcePath
}

// NewManager creates a new transaction manager
func NewManager(fileOps *FileOps, sourceDir scpath.SourcePath) *Manager {
	return &Manager{
		fileOps:   fileOps,
		sourceDir: sourceDir,
	}
}

// ExecuteAtomically executes all operations as a single transaction.
// If any operation fails, all changes are rolled back.
func (m *Manager) ExecuteAtomically(ctx context.Context, ops []Operation) TransactionResult {
	if len(ops) == 0 {
		return success(0, 0)
	}

	lock, err := AcquireLock(m.sourceDir)
	if err != nil {
		return failure(0, len(ops), err)
	}
	defer lock.Release()

	if err := m.validateOperations(ops); err != nil {
		return failure(0, len(ops), err)
	}

	backups, err := m.createBackups(ops)
	if err != nil {
		return failure(0, len(ops), fmt.Errorf("create backups: %w", err))
	}

	applied := 0

	for i, op := range ops {
		select {
		case <-ctx.Done():
			m.rollback(backups)
			return failure(applied, len(ops), ctx.Err())
		default:
		}

		if err := m.fileOps.ApplyOperation(op); err != nil {
			failedOp := &ops[i]
			rollbackOK := m.rollback(backups)

			errMsg := fmt.Sprintf("operation failed (failed at: %s %s)", failedOp.Action, failedOp.Path)
			if applied > 0 {
				errMsg += fmt.Sprintf(" (%d operations completed before failure)", applied)
			}
			if !rollbackOK {
				errMsg += " (WARNING: rollback failed, working directory may be in inconsistent state)"
			}
			errMsg += fmt.Sprintf(": %v", err)

			return failure(applied, len(ops), fmt.Errorf("%s", errMsg))
		}
		applied++
	}

	for _, backup := range backups {
		m.fileOps.CleanupBackup(backup)
	}

	return success(applied, len(ops))
}

// DryRun analyzes operations without executing them.
// It validates operations and checks for conflicts.
func (m *Manager) DryRun(ops []Operation) DryRunResult {
	result := DryRunResult{
		Valid:     true,
		Analysis:  DryRunAnalysis{},
		Conflicts: []string{},
		Errors:    []string{},
	}

	if err := m.validateOperations(ops); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	for _, op := range ops {
		switch op.Action {
		case ActionCreate:
			result.Analysis.WillCreate = append(result.Analysis.WillCreate, op.Path)
		case ActionModify:
			result.Analysis.WillModify = append(result.Analysis.WillModify, op.Path)
		case ActionDelete:
			result.Analysis.WillDelete = append(result.Analysis.WillDelete, op.Path)
		}
	}

	return result
}

// validateOperations checks if operations are valid
func (m *Manager) validateOperations(ops []Operation) error {
	pathsSeen := make(map[scpath.RelativePath]bool)

	for i, op := range ops {
		if !op.Path.IsValid() {
			return fmt.Errorf("%w: operation %d has empty path", ErrInvalidOperation, i)
		}

		if op.Action != ActionCreate &&
			op.Action != ActionModify &&
			op.Action != ActionDelete {
			return fmt.Errorf("%w: operation %d has invalid action %d", ErrInvalidOperation, i, op.Action)
		}

		if (op.Action == ActionCreate || op.Action == ActionModify) && op.SHA == "" {
			return fmt.Errorf("%w: operation %d (%s) missing SHA", ErrInvalidOperation, i, op.Action)
		}

		if pathsSeen[op.Path] {
			return fmt.Errorf("%w: duplicate operation on path %s", ErrInvalidOperation, op.Path)
		}
		pathsSeen[op.Path] = true
	}

	return nil
}

// createBackups creates backups for all modify/delete operations.
// Uses concurrent processing for better performance when backing up multiple files.
func (m *Manager) createBackups(ops []Operation) ([]*Backup, error) {
	var needsBackup []Operation
	for _, op := range ops {
		if op.Action == ActionModify || op.Action == ActionDelete {
			needsBackup = append(needsBackup, op)
		}
	}

	if len(needsBackup) == 0 {
		return nil, nil
	}

	workerPool := pool.NewWorkerPool[Operation, *Backup]()
	backups, err := workerPool.Process(
		context.Background(),
		needsBackup,
		func(ctx context.Context, op Operation) (*Backup, error) {
			backup, err := m.fileOps.CreateBackup(op.Path)
			if err != nil {
				return nil, fmt.Errorf("backup %s: %w", op.Path, err)
			}
			return backup, nil
		},
	)

	if err != nil {
		// Cleanup any successfully created backups
		for _, b := range backups {
			if b != nil {
				m.fileOps.CleanupBackup(b)
			}
		}
		return nil, err
	}

	return backups, nil
}

// rollback restores all files from backups (in reverse order)
func (m *Manager) rollback(backups []*Backup) bool {
	success := true

	for i := len(backups) - 1; i >= 0; i-- {
		if err := m.fileOps.RestoreBackup(backups[i]); err != nil {
			success = false
		}
	}

	for _, backup := range backups {
		m.fileOps.CleanupBackup(backup)
	}

	return success
}
