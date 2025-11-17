package stash

import (
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

// Entry represents a single stash entry
type Entry struct {
	// Index is the stash index (stash@{0}, stash@{1}, etc.)
	Index int

	// Message is the description of the stash
	Message string

	// Branch is the branch name where the stash was created
	Branch string

	// BaseCommit is the commit hash that the stash was based on
	BaseCommit objects.ObjectHash

	// WorkingTreeCommit is the commit representing the working tree state
	WorkingTreeCommit objects.ObjectHash

	// IndexCommit is the commit representing the index state
	IndexCommit objects.ObjectHash

	// Timestamp is when the stash was created
	Timestamp time.Time

	// Author is who created the stash
	Author string

	// AuthorEmail is the email of who created the stash
	AuthorEmail string
}

// StashRef represents a reference to a stash entry
type StashRef struct {
	// Name is the stash reference name (e.g., "stash@{0}")
	Name string

	// Index is the numeric index
	Index int
}

// ParseStashRef parses a stash reference string into a StashRef
func ParseStashRef(ref string) (*StashRef, error) {
	// Implementation will parse strings like "stash@{0}", "stash@{1}", or just "0", "1"
	return nil, nil
}
