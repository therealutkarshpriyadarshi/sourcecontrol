package workdir

import (
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/common/err"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/workdir/internal"
)

const (
	pkgName = "workdir"

	// Package-specific error codes
	CodeDirtyWorkDir   = "DIRTY_WORKDIR"
	CodeInvalidOp      = "INVALID_OP"
	CodeLockFailed     = err.CodeLockFailed
	CodeValidationErr  = err.CodeValidation
	CodeTransactionErr = err.CodeTransaction
	CodeIndexErr       = "INDEX_ERROR"
)

// Common error variables for type checking with errors.Is()
var (
	// ErrDirtyWorkingDirectory is returned when uncommitted changes would be overwritten
	ErrDirtyWorkingDirectory = err.New(pkgName, CodeDirtyWorkDir, "", "working directory has uncommitted changes", nil)
	// ErrInvalidOperation is returned when an operation is malformed
	ErrInvalidOperation = internal.ErrInvalidOperation
	// ErrLockAcquisitionFailed is returned when unable to acquire repository lock
	ErrLockAcquisitionFailed = internal.ErrLockAcquisitionFailed
)

// WorkdirError represents an error that occurred during working directory operations.
// It embeds the base Error type and adds path-specific context.
type WorkdirError struct {
	base *err.Error
	Path scpath.RelativePath
}

// Error implements the error interface
func (e *WorkdirError) Error() string {
	if e.Path.String() != "" {
		return fmt.Sprintf("%s [path=%s]", e.base.Error(), e.Path)
	}
	return e.base.Error()
}

// Unwrap returns the underlying error
func (e *WorkdirError) Unwrap() error {
	return e.base
}

// ValidationError represents an error during working directory validation
type ValidationError struct {
	base *err.Error
	// ModifiedFiles lists files with uncommitted changes
	ModifiedFiles []scpath.RelativePath
	// DeletedFiles lists files that are missing from the working directory
	DeletedFiles []scpath.RelativePath
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	msg := e.base.Error()
	if len(e.ModifiedFiles) > 0 {
		msg += fmt.Sprintf("\n  Modified files (%d):", len(e.ModifiedFiles))
		for i, path := range e.ModifiedFiles {
			if i < 10 {
				msg += fmt.Sprintf("\n    %s", path)
			} else if i == 10 {
				msg += fmt.Sprintf("\n    ... and %d more files", len(e.ModifiedFiles)-10)
				break
			}
		}
	}
	if len(e.DeletedFiles) > 0 {
		msg += fmt.Sprintf("\n  Deleted files (%d):", len(e.DeletedFiles))
		for i, path := range e.DeletedFiles {
			if i < 10 {
				msg += fmt.Sprintf("\n    %s", path)
			} else if i == 10 {
				msg += fmt.Sprintf("\n    ... and %d more files", len(e.DeletedFiles)-10)
				break
			}
		}
	}
	return msg
}

// Unwrap returns the underlying error
func (e *ValidationError) Unwrap() error {
	return e.base
}

// TransactionError represents an error during atomic transaction execution
type TransactionError struct {
	base *err.Error
	// FailedOperation is the operation that caused the failure
	FailedOperation *Operation
	// OperationsCompleted is the number of operations that succeeded before failure
	OperationsCompleted int
	// RollbackSucceeded indicates whether the rollback was successful
	RollbackSucceeded bool
}

// Error implements the error interface
func (e *TransactionError) Error() string {
	msg := e.base.Error()
	if e.FailedOperation != nil {
		msg += fmt.Sprintf(" (failed at: %s %s)", e.FailedOperation.Action, e.FailedOperation.Path)
	}
	if e.OperationsCompleted > 0 {
		msg += fmt.Sprintf(" (%d operations completed before failure)", e.OperationsCompleted)
	}
	if !e.RollbackSucceeded {
		msg += " (WARNING: rollback failed, working directory may be in inconsistent state)"
	}
	return msg
}

// Unwrap returns the underlying error
func (e *TransactionError) Unwrap() error {
	return e.base
}

// LockError represents an error acquiring or managing a repository lock
type LockError struct {
	base *err.Error
	// LockPath is the path to the lock file
	LockPath string
}

// Error implements the error interface
func (e *LockError) Error() string {
	return fmt.Sprintf("%s [lock_path=%s]", e.base.Error(), e.LockPath)
}

// Unwrap returns the underlying error
func (e *LockError) Unwrap() error {
	return e.base
}

// IndexError represents an error during index operations
type IndexError struct {
	base *err.Error
	// Path is the index file path
	Path string
}

// Error implements the error interface
func (e *IndexError) Error() string {
	return fmt.Sprintf("%s [index_path=%s]", e.base.Error(), e.Path)
}

// Unwrap returns the underlying error
func (e *IndexError) Unwrap() error {
	return e.base
}

// NewWorkdirError creates a new WorkdirError
func NewWorkdirError(op string, path scpath.RelativePath, e error) *WorkdirError {
	return &WorkdirError{
		base: err.New(pkgName, "", op, "", e),
		Path: path,
	}
}

// NewWorkdirErrorWithCode creates a new WorkdirError with an error code
func NewWorkdirErrorWithCode(code, op string, path scpath.RelativePath, e error) *WorkdirError {
	return &WorkdirError{
		base: err.New(pkgName, code, op, "", e),
		Path: path,
	}
}

// NewValidationError creates a new ValidationError
func NewValidationError(message string, modified, deleted []scpath.RelativePath) *ValidationError {
	return &ValidationError{
		base:          err.New(pkgName, CodeValidationErr, "validate", message, nil),
		ModifiedFiles: modified,
		DeletedFiles:  deleted,
	}
}

// NewTransactionError creates a new TransactionError
func NewTransactionError(message string, failedOp *Operation, completed int, rollbackOK bool, e error) *TransactionError {
	return &TransactionError{
		base:                err.New(pkgName, CodeTransactionErr, "execute_transaction", message, e),
		FailedOperation:     failedOp,
		OperationsCompleted: completed,
		RollbackSucceeded:   rollbackOK,
	}
}

// NewLockError creates a new LockError
func NewLockError(lockPath, message string, e error) *LockError {
	return &LockError{
		base:     err.New(pkgName, CodeLockFailed, "acquire_lock", message, e),
		LockPath: lockPath,
	}
}

// NewIndexError creates a new IndexError
func NewIndexError(operation, path string, e error) *IndexError {
	return &IndexError{
		base: err.New(pkgName, CodeIndexErr, operation, "", e),
		Path: path,
	}
}
