package tag

import (
	"fmt"
)

// Error codes for tag operations
const (
	ErrCodeTagNotFound      = "TAG_NOT_FOUND"
	ErrCodeTagExists        = "TAG_EXISTS"
	ErrCodeInvalidTagName   = "INVALID_TAG_NAME"
	ErrCodeInvalidObject    = "INVALID_OBJECT"
	ErrCodeSigningFailed    = "SIGNING_FAILED"
	ErrCodeInvalidSignature = "INVALID_SIGNATURE"
)

// ErrTagNotFound is returned when a tag is not found
type ErrTagNotFound struct {
	TagName string
}

func (e *ErrTagNotFound) Error() string {
	return fmt.Sprintf("tag '%s' not found", e.TagName)
}

// NewErrTagNotFound creates a new ErrTagNotFound
func NewErrTagNotFound(tagName string) *ErrTagNotFound {
	return &ErrTagNotFound{
		TagName: tagName,
	}
}

// ErrTagExists is returned when trying to create a tag that already exists
type ErrTagExists struct {
	TagName string
}

func (e *ErrTagExists) Error() string {
	return fmt.Sprintf("tag '%s' already exists", e.TagName)
}

// NewErrTagExists creates a new ErrTagExists
func NewErrTagExists(tagName string) *ErrTagExists {
	return &ErrTagExists{
		TagName: tagName,
	}
}

// ErrInvalidTagName is returned when a tag name is invalid
type ErrInvalidTagName struct {
	TagName string
	Reason  string
}

func (e *ErrInvalidTagName) Error() string {
	return fmt.Sprintf("invalid tag name '%s': %s", e.TagName, e.Reason)
}

// NewErrInvalidTagName creates a new ErrInvalidTagName
func NewErrInvalidTagName(tagName, reason string) *ErrInvalidTagName {
	return &ErrInvalidTagName{
		TagName: tagName,
		Reason:  reason,
	}
}

// ErrInvalidObject is returned when trying to tag an invalid object
type ErrInvalidObject struct {
	ObjectRef string
}

func (e *ErrInvalidObject) Error() string {
	return fmt.Sprintf("invalid object '%s'", e.ObjectRef)
}

// NewErrInvalidObject creates a new ErrInvalidObject
func NewErrInvalidObject(objectRef string) *ErrInvalidObject {
	return &ErrInvalidObject{
		ObjectRef: objectRef,
	}
}

// ErrSigningFailed is returned when GPG signing fails
type ErrSigningFailed struct {
	Reason string
}

func (e *ErrSigningFailed) Error() string {
	return fmt.Sprintf("GPG signing failed: %s", e.Reason)
}

// NewErrSigningFailed creates a new ErrSigningFailed
func NewErrSigningFailed(reason string) *ErrSigningFailed {
	return &ErrSigningFailed{
		Reason: reason,
	}
}
