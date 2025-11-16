package internal

import (
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

type FileMap = map[scpath.RelativePath]FileInfo

// ActionType represents the type of file operation to perform
type ActionType int

const (
	// ActionCreate creates a new file in the working directory
	ActionCreate ActionType = iota
	// ActionModify updates an existing file's content or permissions
	ActionModify
	// ActionDelete removes a file from the working directory
	ActionDelete
)

// String returns the string representation of the action type
func (a ActionType) String() string {
	switch a {
	case ActionCreate:
		return "create"
	case ActionModify:
		return "modify"
	case ActionDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// Operation represents a single file operation to be performed on the working directory.
type Operation struct {
	Path   scpath.RelativePath
	Action ActionType
	SHA    objects.ObjectHash
	Mode   objects.FileMode
}

// Backup represents a snapshot of a file before modification
type Backup struct {
	Path     scpath.RelativePath
	TempFile string
	Existed  bool
	Mode     objects.FileMode
}

// Status represents the state of the working directory relative to the index
type Status struct {
	Clean          bool
	ModifiedFiles  []scpath.RelativePath
	DeletedFiles   []scpath.RelativePath
	UntrackedFiles []scpath.RelativePath
	Details        []FileStatusDetail
}

// ChangeSummary provides statistics about detected changes
type ChangeSummary struct {
	Created  int
	Modified int
	Deleted  int
}

// IndexUpdateResult contains the outcome of an index synchronization operation
type IndexUpdateResult struct {
	Success        bool
	EntriesUpdated int
	EntriesRemoved int
	Errors         []error
}
