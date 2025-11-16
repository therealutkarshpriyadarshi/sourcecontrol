package workdir

import (
	"context"

	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/workdir/internal"
)

// Re-export types from internal package for public API
type (
	// ActionType represents the type of file operation to perform
	ActionType = internal.ActionType

	// Operation represents a single file operation to be performed on the working directory.
	Operation = internal.Operation

	// FileInfo represents metadata about a file in a tree or index
	FileInfo = internal.FileInfo

	// Backup represents a snapshot of a file before modification
	Backup = internal.Backup

	// FileStatusDetail contains detailed status information for a single file
	FileStatusDetail = internal.FileStatusDetail

	// Status represents the state of the working directory relative to the index
	Status = internal.Status

	// ChangeSummary provides statistics about detected changes
	ChangeSummary = internal.ChangeSummary

	// ChangeAnalysis contains the result of comparing two file states
	ChangeAnalysis = internal.ChangeAnalysis

	// IndexUpdateResult contains the outcome of an index synchronization operation
	IndexUpdateResult = internal.IndexUpdateResult
)

// Re-export action type constants
const (
	ActionCreate = internal.ActionCreate
	ActionModify = internal.ActionModify
	ActionDelete = internal.ActionDelete
)

// UpdateOptions configures how the working directory update should be performed
type UpdateOptions struct {
	// Force bypasses safety checks for uncommitted changes
	Force bool
	// DryRun analyzes what would change without making actual modifications
	DryRun bool
	// OnProgress is called during the update to report progress (optional)
	OnProgress func(completed, total int, currentFile string)
}

// UpdateResult contains the outcome of a working directory update operation
type UpdateResult struct {
	// Success indicates whether the update completed successfully
	Success bool
	// FilesChanged is the number of files that were created, modified, or deleted
	FilesChanged int
	// Operations lists all operations that were performed or attempted
	Operations []Operation
	// IndexUpdate contains the result of the index synchronization (may be nil)
	IndexUpdate *IndexUpdateResult
	// Err contains any error that occurred during the update
	Err error
}

// TransactionResult contains the outcome of an atomic transaction
type TransactionResult struct {
	// Success indicates whether all operations completed successfully
	Success bool
	// OperationsApplied is the number of operations successfully completed
	OperationsApplied int
	// TotalOperations is the total number of operations attempted
	TotalOperations int
	// Err contains the error that caused the transaction to fail (if any)
	Err error
}

// DryRunResult contains the analysis of operations without executing them
type DryRunResult struct {
	// Valid is true if all operations can be safely executed
	Valid bool
	// Analysis breaks down operations by type
	Analysis DryRunAnalysis
	// Conflicts lists any problems that would prevent execution
	Conflicts []string
	// Errors contains detailed error messages
	Errors []string
}

// DryRunAnalysis categorizes operations for dry run reporting
type DryRunAnalysis struct {
	// WillCreate lists files that would be created
	WillCreate []scpath.RelativePath
	// WillModify lists files that would be modified
	WillModify []scpath.RelativePath
	// WillDelete lists files that would be deleted
	WillDelete []scpath.RelativePath
	// Conflicts lists files with conflicting operations
	Conflicts []string
}

// FileOperator defines the interface for low-level file system operations
type FileOperator interface {
	// ApplyOperation executes a single file operation (create, modify, or delete)
	ApplyOperation(op Operation) error
	// CreateBackup creates a backup of a file before modification
	CreateBackup(path scpath.RelativePath) (*Backup, error)
	// RestoreBackup restores a file from a backup
	RestoreBackup(backup *Backup) error
	// CleanupBackup removes a backup file after successful operation
	CleanupBackup(backup *Backup) error
}

// TreeAnalyzer defines the interface for analyzing Git tree objects and detecting changes
type TreeAnalyzer interface {
	// GetCommitFiles retrieves all files from a commit's tree
	GetCommitFiles(commitSHA objects.ObjectHash) (map[scpath.RelativePath]FileInfo, error)
	// GetIndexFiles extracts file information from the index
	GetIndexFiles(idx *index.Index) map[scpath.RelativePath]FileInfo
	// AnalyzeChanges compares current and target states to generate operations
	AnalyzeChanges(current, target map[scpath.RelativePath]FileInfo) ChangeAnalysis
}

// Validator defines the interface for validating working directory state
type Validator interface {
	// ValidateCleanState checks if the working directory has uncommitted changes
	ValidateCleanState(idx *index.Index) (Status, error)
	// CanSafelyOverwrite checks if files can be safely replaced
	CanSafelyOverwrite(paths []scpath.RelativePath, idx *index.Index) error
}

// TransactionManager defines the interface for atomic operation execution
type TransactionManager interface {
	// ExecuteAtomically executes all operations as a single atomic transaction
	ExecuteAtomically(ctx context.Context, ops []Operation) TransactionResult
	// DryRun analyzes operations without executing them
	DryRun(ops []Operation) DryRunResult
}

// IndexUpdater defines the interface for synchronizing the Git index
type IndexUpdater interface {
	// UpdateToMatch replaces the entire index to match the target files
	UpdateToMatch(targetFiles map[scpath.RelativePath]FileInfo) (IndexUpdateResult, error)
	// UpdateIncremental applies specific additions and removals to the index
	UpdateIncremental(toAdd map[scpath.RelativePath]FileInfo, toRemove []scpath.RelativePath) (IndexUpdateResult, error)
}
