package commitmanager

import (
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
)

// CommitOptions contains configuration for creating a commit
type CommitOptions struct {
	// Message is the commit message (required)
	Message string

	// Author is the commit author (optional, defaults to config user)
	Author *commit.CommitPerson

	// Committer is the person committing (optional, defaults to Author)
	Committer *commit.CommitPerson

	// Amend indicates whether to amend the previous commit
	Amend bool

	// AllowEmpty allows creating a commit with no changes
	AllowEmpty bool

	// NoVerify skips pre-commit and commit-msg hooks (currently not used)
	NoVerify bool
}

// Validate validates CommitOptions
func (opts *CommitOptions) Validate() error {
	if opts.Message == "" {
		return NewCommitError("validate options", ErrEmptyMessage, "")
	}
	return nil
}
