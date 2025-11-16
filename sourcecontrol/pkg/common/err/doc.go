// Package errors provides a standardized error handling system for the entire project.
//
// # Design Principles
//
// 1. Consistency: All packages use the same base error structure
// 2. Context: Errors carry package, operation, and code information
// 3. Wrapping: Full support for Go 1.13+ error wrapping with errors.Is/As
// 4. Categorization: Machine-readable error codes enable programmatic handling
// 5. Performance: Minimal allocations with lazy context initialization
//
// # Usage Patterns
//
// ## Creating Package-Specific Errors
//
// Each package should define its own error types that embed errors.Error:
//
//	type MyPackageError struct {
//	    *errors.Error
//	    CustomField string
//	}
//
//	func NewMyPackageError(op, customField string, err error) *MyPackageError {
//	    return &MyPackageError{
//	        Error: errors.New("mypackage", errors.CodeInternal, op, "", err),
//	        CustomField: customField,
//	    }
//	}
//
// ## Defining Package Constants
//
// Define package name and error codes as constants:
//
//	const (
//	    pkgName = "mypackage"
//	    CodeSpecificError = "SPECIFIC_ERROR"
//	)
//
// ## Error Checking
//
// Use standard Go error checking patterns:
//
//	if errors.IsCode(err, errors.CodeNotFound) {
//	    // handle not found
//	}
//
//	var myErr *MyPackageError
//	if errors.As(err, &myErr) {
//	    // access custom fields
//	}
//
// ## Adding Context
//
// Add structured context to errors:
//
//	err := errors.New("pkg", "CODE", "operation", "message", nil)
//	err.WithContext("user_id", userID).WithContext("retry_count", 3)
//
// # Error Codes
//
// Standard error codes are provided as constants (CodeNotFound, CodeValidation, etc.).
// Packages can define additional codes as needed, following the UPPER_SNAKE_CASE convention.
package err
