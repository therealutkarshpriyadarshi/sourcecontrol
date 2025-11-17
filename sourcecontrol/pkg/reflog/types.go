package reflog

import (
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

// Entry represents a single reflog entry
type Entry struct {
	// OldHash is the previous commit hash
	OldHash objects.ObjectHash

	// NewHash is the new commit hash
	NewHash objects.ObjectHash

	// Committer is who made the change
	Committer string

	// CommitterEmail is the email of who made the change
	CommitterEmail string

	// Timestamp is when the change was made
	Timestamp time.Time

	// Message describes the change
	Message string
}

// RefName represents a reference name (HEAD, branch name, etc.)
type RefName string

const (
	// RefHead is the HEAD reference
	RefHead RefName = "HEAD"
)

// BranchRef returns a branch reference name
func BranchRef(branchName string) RefName {
	return RefName("refs/heads/" + branchName)
}
