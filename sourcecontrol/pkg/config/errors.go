package config

import (
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/common/err"
)

const (
	pkgName = "config"
)

// ============================================================================
// Error Codes
// ============================================================================

const (
	// Shared error codes from common package
	CodeNotFoundErr      = err.CodeNotFound
	CodeInvalidFormatErr = err.CodeInvalidFormat
	CodeInvalidValueErr  = err.CodeInvalidFormat
	CodeReadOnlyErr      = err.CodeReadOnly

	// Package-specific error codes
	CodeConversionErr   = "CONVERSION_FAILED"
	CodeInvalidLevelErr = "INVALID_LEVEL"
)

// ============================================================================
// Error Types
// ============================================================================

// ConfigError represents a configuration-related error with detailed context
type ConfigError struct {
	base  *err.Error
	Path  string // file path if applicable
	Key   string // config key if applicable
	Level string // config level if applicable
}

// NewConfigError creates a new ConfigError
func NewConfigError(op, code, key, path, level string, underlying error) *ConfigError {
	return &ConfigError{
		base:  err.New(pkgName, code, op, "", underlying),
		Path:  path,
		Key:   key,
		Level: level,
	}
}

// Error implements the error interface
func (e *ConfigError) Error() string {
	msg := e.base.Error()
	if e.Key != "" {
		msg += fmt.Sprintf(" [key=%s]", e.Key)
	}
	if e.Path != "" {
		msg += fmt.Sprintf(" [path=%s]", e.Path)
	}
	if e.Level != "" {
		msg += fmt.Sprintf(" [level=%s]", e.Level)
	}
	return msg
}

// Unwrap returns the underlying error
func (e *ConfigError) Unwrap() error {
	return e.base
}

// ============================================================================
// Sentinel Errors
// ============================================================================

var (
	// ErrNotFound indicates a configuration key was not found
	ErrNotFound = err.New(pkgName, CodeNotFoundErr, "", "configuration key not found", nil)

	// ErrInvalidFormat indicates the config file has invalid format
	ErrInvalidFormat = err.New(pkgName, CodeInvalidFormatErr, "", "invalid configuration format", nil)

	// ErrInvalidValue indicates a configuration value is invalid
	ErrInvalidValue = err.New(pkgName, CodeInvalidValueErr, "", "invalid configuration value", nil)

	// ErrInvalidLevel indicates an invalid configuration level
	ErrInvalidLevel = err.New(pkgName, CodeInvalidLevelErr, "", "invalid configuration level", nil)

	// ErrReadOnly indicates an attempt to modify a read-only configuration
	ErrReadOnly = err.New(pkgName, CodeReadOnlyErr, "", "configuration is read-only", nil)

	// ErrConversion indicates a type conversion error
	ErrConversion = err.New(pkgName, CodeConversionErr, "", "configuration value conversion failed", nil)
)

// ============================================================================
// Specialized Error Constructors
// ============================================================================

// NewNotFoundError creates a ConfigError for configuration key not found errors
func NewNotFoundError(key, level string) *ConfigError {
	return &ConfigError{
		base:  err.New(pkgName, CodeNotFoundErr, "get", "configuration key not found", ErrNotFound),
		Key:   key,
		Level: level,
	}
}

// NewInvalidFormatError creates a ConfigError for configuration format errors
func NewInvalidFormatError(op, path string, underlying error) *ConfigError {
	return &ConfigError{
		base: err.New(pkgName, CodeInvalidFormatErr, op, "invalid configuration format", underlying),
		Path: path,
	}
}

// NewInvalidValueError creates a ConfigError for configuration value validation errors
func NewInvalidValueError(key string, underlying error) *ConfigError {
	return &ConfigError{
		base: err.New(pkgName, CodeInvalidValueErr, "validate", "invalid configuration value", underlying),
		Key:  key,
	}
}

// NewConversionError creates a ConfigError for type conversion failures
func NewConversionError(key string, underlying error) *ConfigError {
	return &ConfigError{
		base: err.New(pkgName, CodeConversionErr, "convert", "configuration value conversion failed", underlying),
		Key:  key,
	}
}

// ============================================================================
// Error Checking Helpers
// ============================================================================

// IsNotFound returns true if the error is ErrNotFound
func IsNotFound(e error) bool {
	return err.IsCode(e, CodeNotFoundErr)
}

// IsInvalidFormat returns true if the error is ErrInvalidFormat
func IsInvalidFormat(e error) bool {
	return err.IsCode(e, CodeInvalidFormatErr)
}

// IsReadOnly returns true if the error is ErrReadOnly
func IsReadOnly(e error) bool {
	return err.IsCode(e, CodeReadOnlyErr)
}

// IsConversion returns true if the error is ErrConversion
func IsConversion(e error) bool {
	return err.IsCode(e, CodeConversionErr)
}
