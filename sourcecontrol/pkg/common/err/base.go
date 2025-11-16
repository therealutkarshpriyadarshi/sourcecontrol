package err

import (
	"errors"
	"strings"
)

// Error is the base error type for the entire project.
// It provides a consistent structure for error handling across all packages.
//
// Key features:
//   - Package namespacing for error origin tracking
//   - Machine-readable error codes for programmatic handling
//   - Operation context for debugging
//   - Error wrapping with full errors.Is/As support
//   - Optional structured context data
//
// Design philosophy:
//   - Package-specific errors embed this type and add domain fields
//   - Error codes enable retry logic and error categorization
//   - Minimal allocation overhead with lazy context initialization
type Error struct {
	// Package identifies the originating package (e.g., "workdir", "config", "index")
	Package string

	// Code is a machine-readable error code for categorization and handling.
	// Use package constants (e.g., CodeNotFound, CodeValidation).
	Code string

	// Op is the operation being performed when the error occurred.
	// Use descriptive names like "read", "write", "validate", "acquire_lock".
	Op string

	// Message provides human-readable context. Keep it brief and actionable.
	// Detailed information should go in Context or be part of the wrapped error.
	Message string

	// Err is the underlying/wrapped error. Can be nil for leaf errors.
	Err error

	// Context holds optional structured metadata about the error.
	// Initialized lazily to avoid allocations when not needed.
	// Use WithContext() to add fields.
	Context map[string]interface{}
}

// Error implements the error interface.
// Format: [package][code] operation: message: wrapped_error
func (e *Error) Error() string {
	var parts []string

	// Build prefix with package and code
	var prefix strings.Builder
	if e.Package != "" {
		prefix.WriteString("[")
		prefix.WriteString(e.Package)
		prefix.WriteString("]")
	}
	if e.Code != "" {
		prefix.WriteString("[")
		prefix.WriteString(e.Code)
		prefix.WriteString("]")
	}
	if prefix.Len() > 0 {
		parts = append(parts, prefix.String())
	}

	// Add operation
	if e.Op != "" {
		parts = append(parts, e.Op)
	}

	// Add message
	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	// Build the main error string
	result := strings.Join(parts, ": ")

	// Append wrapped error
	if e.Err != nil {
		if result != "" {
			result += ": " + e.Err.Error()
		} else {
			result = e.Err.Error()
		}
	}

	return result
}

// Unwrap returns the underlying error for errors.Is() and errors.As() support.
func (e *Error) Unwrap() error {
	return e.Err
}

// Is enables error matching by code for errors.Is() checks.
// Two errors match if they have the same non-empty code.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	// Only match on code if both have it set
	return e.Code != "" && e.Code == t.Code
}

// WithContext adds a key-value pair to the error's context.
// Returns the error for method chaining.
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// GetContext retrieves a value from the error's context.
// Returns nil if the key doesn't exist.
func (e *Error) GetContext(key string) interface{} {
	if e.Context == nil {
		return nil
	}
	return e.Context[key]
}

// New creates a new base error with the specified fields.
func New(pkg, code, op, message string, err error) *Error {
	return &Error{
		Package: pkg,
		Code:    code,
		Op:      op,
		Message: message,
		Err:     err,
	}
}

// Wrap wraps an error with package and operation context.
// Returns nil if err is nil.
func Wrap(err error, pkg, op string) error {
	if err == nil {
		return nil
	}
	return &Error{
		Package: pkg,
		Op:      op,
		Err:     err,
	}
}

// WrapWithCode wraps an error with package, operation, and code.
// Returns nil if err is nil.
func WrapWithCode(err error, pkg, code, op string) error {
	if err == nil {
		return nil
	}
	return &Error{
		Package: pkg,
		Code:    code,
		Op:      op,
		Err:     err,
	}
}

// Standard error codes used across packages.
// Packages can define their own specific codes, but should use these when applicable.
const (
	// CodeInvalidInput indicates invalid or malformed input parameters
	CodeInvalidInput = "INVALID_INPUT"

	// CodeNotFound indicates a requested resource was not found
	CodeNotFound = "NOT_FOUND"

	// CodeAlreadyExists indicates a resource already exists when it shouldn't
	CodeAlreadyExists = "ALREADY_EXISTS"

	// CodePermissionDenied indicates insufficient permissions for the operation
	CodePermissionDenied = "PERMISSION_DENIED"

	// CodeTimeout indicates an operation exceeded its time limit
	CodeTimeout = "TIMEOUT"

	// CodeInternal indicates an unexpected internal error
	CodeInternal = "INTERNAL"

	// CodeLockFailed indicates failure to acquire a required lock
	CodeLockFailed = "LOCK_FAILED"

	// CodeValidation indicates data validation failed
	CodeValidation = "VALIDATION"

	// CodeTransaction indicates a transaction operation failed
	CodeTransaction = "TRANSACTION"

	// CodeConflict indicates a conflict with current state (e.g., dirty working directory)
	CodeConflict = "CONFLICT"

	// CodeInvalidFormat indicates data is in an invalid format
	CodeInvalidFormat = "INVALID_FORMAT"

	// CodeReadOnly indicates an attempt to modify read-only data
	CodeReadOnly = "READ_ONLY"
)

// IsCode checks if an error has a specific error code.
// Works with wrapped errors.
func IsCode(err error, code string) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Code == code
	}
	return false
}

// GetCode extracts the error code from an error.
// Returns empty string if the error is not a base Error.
func GetCode(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// GetPackage extracts the package name from an error.
// Returns empty string if the error is not a base Error.
func GetPackage(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Package
	}
	return ""
}

// GetOp extracts the operation from an error.
// Returns empty string if the error is not a base Error.
func GetOp(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Op
	}
	return ""
}
