package stash

import (
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

// Entry represents a single stash entry
// A stash entry contains:
// - Index state (staged changes)
// - Working directory state
// - Metadata (message, timestamp, branch)
type Entry struct {
	// SHA of the stash commit object
	SHA objects.ObjectHash

	// Message describing the stash
	Message string

	// Branch where the stash was created
	Branch string

	// Timestamp when the stash was created
	CreatedAt time.Time

	// Index of the stash (0 is most recent)
	Index int
}

// StashCommit represents the structure of a stash as commits
// A stash consists of:
// - Base commit (where stash was created from)
// - Index commit (staged changes)
// - Working tree commit (unstaged changes) - this is the main stash commit
type StashCommit struct {
	// WorkingTreeCommit is the main stash commit containing working directory changes
	WorkingTreeCommit objects.ObjectHash

	// IndexCommit contains the staged changes
	IndexCommit objects.ObjectHash

	// BaseCommit is the commit that was HEAD when stash was created
	BaseCommit objects.ObjectHash

	// UntrackedCommit contains untracked files (optional)
	UntrackedCommit objects.ObjectHash
}

// StashOptions configures how a stash is created
type StashOptions struct {
	// Message is the stash description
	Message string

	// KeepIndex preserves staged changes after stashing
	KeepIndex bool

	// IncludeUntracked includes untracked files in stash
	IncludeUntracked bool

	// Paths specifies specific files to stash (partial stash)
	Paths []string
}

// ApplyOptions configures how a stash is applied
type ApplyOptions struct {
	// Index restores staged changes as staged
	Index bool

	// Quiet suppresses output
	Quiet bool

	// Pop removes the stash after applying
	Pop bool
}
