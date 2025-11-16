package branch

import (
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

// BranchInfo contains detailed information about a branch
type BranchInfo struct {
	// Name is the branch name (e.g., "main", "feature/new-feature")
	Name string

	// SHA is the commit hash the branch points to
	SHA objects.ObjectHash

	// IsCurrentBranch indicates if this is the currently checked out branch
	IsCurrentBranch bool

	// CommitCount is the total number of commits in this branch
	CommitCount int

	// LastCommitDate is when the last commit was made
	LastCommitDate time.Time

	// LastCommitMessage is the message of the last commit
	LastCommitMessage string

	// Ahead is the number of commits ahead of the upstream/base branch
	Ahead int

	// Behind is the number of commits behind the upstream/base branch
	Behind int
}

// ValidationResult contains the result of branch name validation
type ValidationResult struct {
	// IsValid indicates if the branch name is valid
	IsValid bool

	// Errors contains validation error messages
	Errors []string
}

// CreateConfig holds configuration for branch creation
type CreateConfig struct {
	// StartPoint is the commit SHA or branch name to start from
	// If empty, uses HEAD
	StartPoint string

	// Checkout switches to the new branch after creation
	Checkout bool

	// Force overwrites the branch if it already exists
	Force bool

	// Track sets up tracking for a remote branch
	Track string
}

// CreateOption is a functional option for configuring branch creation
type CreateOption func(*CreateConfig)

// WithStartPoint sets the starting point for the new branch
func WithStartPoint(ref string) CreateOption {
	return func(c *CreateConfig) {
		c.StartPoint = ref
	}
}

// WithCheckout makes the operation checkout the new branch after creation
func WithCheckout() CreateOption {
	return func(c *CreateConfig) {
		c.Checkout = true
	}
}

// WithForceCreate forces creation even if the branch exists
func WithForceCreate() CreateOption {
	return func(c *CreateConfig) {
		c.Force = true
	}
}

// WithTrack sets up tracking for a remote branch
func WithTrack(remote string) CreateOption {
	return func(c *CreateConfig) {
		c.Track = remote
	}
}

// CheckoutConfig holds configuration for branch checkout
type CheckoutConfig struct {
	// Force discards local changes during checkout
	Force bool

	// Create creates the branch if it doesn't exist
	Create bool

	// Orphan creates an orphan branch (no parent commits)
	Orphan bool

	// Detach checks out in detached HEAD state
	Detach bool
}

// CheckoutOption is a functional option for configuring checkout
type CheckoutOption func(*CheckoutConfig)

// WithForceCheckout forces checkout even with uncommitted changes
func WithForceCheckout() CheckoutOption {
	return func(c *CheckoutConfig) {
		c.Force = true
	}
}

// WithCreateBranch creates the branch if it doesn't exist
func WithCreateBranch() CheckoutOption {
	return func(c *CheckoutConfig) {
		c.Create = true
	}
}

// WithOrphan creates an orphan branch with no parent commits
func WithOrphan() CheckoutOption {
	return func(c *CheckoutConfig) {
		c.Orphan = true
	}
}

// WithDetach checks out in detached HEAD state
func WithDetach() CheckoutOption {
	return func(c *CheckoutConfig) {
		c.Detach = true
	}
}

// DeleteConfig holds configuration for branch deletion
type DeleteConfig struct {
	// Force deletes even if the branch is not fully merged
	Force bool

	// Remote indicates this is a remote branch deletion
	Remote bool
}

// DeleteOption is a functional option for configuring deletion
type DeleteOption func(*DeleteConfig)

// WithForceDelete forces deletion even if not merged
func WithForceDelete() DeleteOption {
	return func(c *DeleteConfig) {
		c.Force = true
	}
}

// WithRemoteDelete marks this as a remote branch deletion
func WithRemoteDelete() DeleteOption {
	return func(c *DeleteConfig) {
		c.Remote = true
	}
}

// RenameConfig holds configuration for branch renaming
type RenameConfig struct {
	// Force overwrites the target branch if it exists
	Force bool
}

// RenameOption is a functional option for configuring rename
type RenameOption func(*RenameConfig)

// WithForceRename forces rename even if target exists
func WithForceRename() RenameOption {
	return func(c *RenameConfig) {
		c.Force = true
	}
}
