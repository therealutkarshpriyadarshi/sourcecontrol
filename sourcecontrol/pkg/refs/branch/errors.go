package branch

import (
	"fmt"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/common/err"
)

const (
	// Package name for error reporting
	pkgName = "branch"
)

// Error codes for branch operations
const (
	CodeNotFound      = "BRANCH_NOT_FOUND"
	CodeAlreadyExists = "BRANCH_ALREADY_EXISTS"
	CodeInvalidName   = "BRANCH_INVALID_NAME"
	CodeNotMerged     = "BRANCH_NOT_MERGED"
	CodeIsCurrent     = "BRANCH_IS_CURRENT"
	CodeDetached      = "BRANCH_DETACHED_HEAD"
)

// NotFoundError indicates a branch doesn't exist
type NotFoundError struct {
	baseError  *err.Error
	BranchName string
}

// NewNotFoundError creates a new branch not found error
func NewNotFoundError(name string) error {
	return &NotFoundError{
		baseError: err.New(
			pkgName,
			CodeNotFound,
			"lookup",
			fmt.Sprintf("branch '%s' not found", name),
			nil,
		),
		BranchName: name,
	}
}

// Error implements the error interface
func (e *NotFoundError) Error() string {
	return e.baseError.Error()
}

// Unwrap returns the underlying error
func (e *NotFoundError) Unwrap() error {
	return e.baseError
}

// AlreadyExistsError indicates a branch already exists
type AlreadyExistsError struct {
	baseError  *err.Error
	BranchName string
}

// NewAlreadyExistsError creates a new branch already exists error
func NewAlreadyExistsError(name string) error {
	return &AlreadyExistsError{
		baseError: err.New(
			pkgName,
			CodeAlreadyExists,
			"create",
			fmt.Sprintf("branch '%s' already exists", name),
			nil,
		),
		BranchName: name,
	}
}

// Error implements the error interface
func (e *AlreadyExistsError) Error() string {
	return e.baseError.Error()
}

// Unwrap returns the underlying error
func (e *AlreadyExistsError) Unwrap() error {
	return e.baseError
}

// InvalidNameError indicates an invalid branch name
type InvalidNameError struct {
	baseError  *err.Error
	BranchName string
	Reasons    []string
}

// NewInvalidNameError creates a new invalid branch name error
func NewInvalidNameError(name string, reasons ...string) error {
	msg := fmt.Sprintf("invalid branch name '%s'", name)
	if len(reasons) > 0 {
		msg += ": " + strings.Join(reasons, "; ")
	}

	return &InvalidNameError{
		baseError: err.New(
			pkgName,
			CodeInvalidName,
			"validate",
			msg,
			nil,
		),
		BranchName: name,
		Reasons:    reasons,
	}
}

// Error implements the error interface
func (e *InvalidNameError) Error() string {
	return e.baseError.Error()
}

// Unwrap returns the underlying error
func (e *InvalidNameError) Unwrap() error {
	return e.baseError
}

// NotMergedError indicates a branch is not fully merged
type NotMergedError struct {
	baseError  *err.Error
	BranchName string
}

// NewNotMergedError creates a new branch not merged error
func NewNotMergedError(name string) error {
	return &NotMergedError{
		baseError: err.New(
			pkgName,
			CodeNotMerged,
			"delete",
			fmt.Sprintf("branch '%s' is not fully merged", name),
			nil,
		),
		BranchName: name,
	}
}

// Error implements the error interface
func (e *NotMergedError) Error() string {
	return e.baseError.Error()
}

// Unwrap returns the underlying error
func (e *NotMergedError) Unwrap() error {
	return e.baseError
}

// IsCurrentError indicates attempting to delete the current branch
type IsCurrentError struct {
	baseError  *err.Error
	BranchName string
}

// NewIsCurrentError creates a new is current branch error
func NewIsCurrentError(name string) error {
	return &IsCurrentError{
		baseError: err.New(
			pkgName,
			CodeIsCurrent,
			"delete",
			fmt.Sprintf("cannot delete current branch '%s'", name),
			nil,
		),
		BranchName: name,
	}
}

// Error implements the error interface
func (e *IsCurrentError) Error() string {
	return e.baseError.Error()
}

// Unwrap returns the underlying error
func (e *IsCurrentError) Unwrap() error {
	return e.baseError
}

// DetachedHeadError indicates HEAD is in detached state
type DetachedHeadError struct {
	baseError *err.Error
	CommitSHA string
}

// NewDetachedHeadError creates a new detached HEAD error
func NewDetachedHeadError(sha string) error {
	msg := "HEAD is detached"
	if sha != "" {
		msg = fmt.Sprintf("HEAD is detached at %s", sha)
	}

	return &DetachedHeadError{
		baseError: err.New(
			pkgName,
			CodeDetached,
			"check",
			msg,
			nil,
		),
		CommitSHA: sha,
	}
}

// Error implements the error interface
func (e *DetachedHeadError) Error() string {
	return e.baseError.Error()
}

// Unwrap returns the underlying error
func (e *DetachedHeadError) Unwrap() error {
	return e.baseError
}
