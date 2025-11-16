package commitmanager

import (
	"errors"
	"fmt"
)

var (
	// ErrEmptyMessage indicates an empty commit message was provided
	ErrEmptyMessage = errors.New("commit message cannot be empty")

	// ErrNoChanges indicates no changes are staged for commit
	ErrNoChanges = errors.New("no changes staged for commit")

	// ErrNoTreeChanges indicates the tree is identical to the parent
	ErrNoTreeChanges = errors.New("no changes to commit (tree is identical to parent)")

	// ErrInvalidCommit indicates the object is not a valid commit
	ErrInvalidCommit = errors.New("object is not a valid commit")

	// ErrNoParent indicates no parent commit exists
	ErrNoParent = errors.New("no parent commit found")
)

// CommitError represents an error that occurred during commit operations
type CommitError struct {
	Op      string // Operation that failed
	Err     error  // Underlying error
	Details string // Additional details
}

// Error implements the error interface
func (e *CommitError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("commit %s: %v (%s)", e.Op, e.Err, e.Details)
	}
	return fmt.Sprintf("commit %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
func (e *CommitError) Unwrap() error {
	return e.Err
}

// NewCommitError creates a new CommitError
func NewCommitError(op string, err error, details string) error {
	return &CommitError{
		Op:      op,
		Err:     err,
		Details: details,
	}
}
