package merge

import (
	"context"
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// Strategy defines the type of merge to perform
type Strategy int

const (
	// StrategyRecursive is the default merge strategy (3-way merge with recursive ancestor merging)
	StrategyRecursive Strategy = iota
	// StrategyOctopus merges multiple branches at once
	StrategyOctopus
	// StrategyOurs always uses our version in case of conflicts
	StrategyOurs
	// StrategySubtree is for merging subtrees
	StrategySubtree
)

// String returns the string representation of the strategy
func (s Strategy) String() string {
	switch s {
	case StrategyRecursive:
		return "recursive"
	case StrategyOctopus:
		return "octopus"
	case StrategyOurs:
		return "ours"
	case StrategySubtree:
		return "subtree"
	default:
		return "unknown"
	}
}

// MergeMode defines how the merge should be executed
type MergeMode int

const (
	// ModeDefault performs a normal merge with commit creation
	ModeDefault MergeMode = iota
	// ModeNoCommit stages changes but doesn't create a commit
	ModeNoCommit
	// ModeSquash squashes all commits into a single commit
	ModeSquash
	// ModeFastForwardOnly only allows fast-forward merges
	ModeFastForwardOnly
	// ModeNoFastForward always creates a merge commit even if fast-forward is possible
	ModeNoFastForward
)

// String returns the string representation of the merge mode
func (m MergeMode) String() string {
	switch m {
	case ModeDefault:
		return "default"
	case ModeNoCommit:
		return "no-commit"
	case ModeSquash:
		return "squash"
	case ModeFastForwardOnly:
		return "ff-only"
	case ModeNoFastForward:
		return "no-ff"
	default:
		return "unknown"
	}
}

// ConflictResolution defines how to handle conflicts
type ConflictResolution int

const (
	// ConflictFail aborts merge on conflict
	ConflictFail ConflictResolution = iota
	// ConflictOurs uses our version
	ConflictOurs
	// ConflictTheirs uses their version
	ConflictTheirs
	// ConflictManual requires manual conflict resolution
	ConflictManual
)

// Config holds the configuration for a merge operation
type Config struct {
	// Strategy to use for merging
	Strategy Strategy
	// Mode defines merge behavior (no-commit, squash, etc.)
	Mode MergeMode
	// ConflictResolution strategy
	ConflictResolution ConflictResolution
	// Message for the merge commit
	Message string
	// AllowUnrelatedHistories permits merging branches with no common ancestor
	AllowUnrelatedHistories bool
	// Verbose enables detailed output
	Verbose bool
}

// DefaultConfig returns a config with default settings
func DefaultConfig() *Config {
	return &Config{
		Strategy:                StrategyRecursive,
		Mode:                    ModeDefault,
		ConflictResolution:      ConflictFail,
		AllowUnrelatedHistories: false,
		Verbose:                 false,
	}
}

// MergeResult represents the outcome of a merge operation
type MergeResult struct {
	// Success indicates if the merge completed without conflicts
	Success bool
	// FastForward indicates if this was a fast-forward merge
	FastForward bool
	// CommitSHA is the SHA of the merge commit (if created)
	CommitSHA objects.ObjectHash
	// Conflicts lists files with conflicts
	Conflicts []string
	// FilesChanged is the number of files modified
	FilesChanged int
	// Insertions is the number of lines added
	Insertions int
	// Deletions is the number of lines removed
	Deletions int
	// Message provides additional information about the merge
	Message string
}

// Conflict represents a merge conflict for a specific file
type Conflict struct {
	// Path to the conflicting file
	Path scpath.RelativePath
	// OurVersion is the content from the current branch
	OurVersion []byte
	// TheirVersion is the content from the branch being merged
	TheirVersion []byte
	// BaseVersion is the content from the common ancestor (if available)
	BaseVersion []byte
	// OurSHA is the object hash of our version
	OurSHA objects.ObjectHash
	// TheirSHA is the object hash of their version
	TheirSHA objects.ObjectHash
	// BaseSHA is the object hash of the base version
	BaseSHA objects.ObjectHash
}

// String returns a human-readable representation of the conflict
func (c *Conflict) String() string {
	return fmt.Sprintf("Conflict in %s (ours: %s, theirs: %s, base: %s)",
		c.Path, c.OurSHA.Short(), c.TheirSHA.Short(), c.BaseSHA.Short())
}

// MergeContext holds the context for a merge operation
type MergeContext struct {
	// Ctx is the Go context for cancellation
	Ctx context.Context
	// OurCommit is the current HEAD commit
	OurCommit *commit.Commit
	// TheirCommits are the commits being merged in
	TheirCommits []*commit.Commit
	// BaseCommit is the common ancestor (merge base)
	BaseCommit *commit.Commit
	// Config holds the merge configuration
	Config *Config
}

// Merger defines the interface for merge strategy implementations
type Merger interface {
	// CanMerge checks if this strategy can handle the given merge context
	CanMerge(ctx *MergeContext) bool
	// Merge performs the actual merge operation
	Merge(ctx *MergeContext) (*MergeResult, error)
	// Name returns the name of this merge strategy
	Name() string
}

// MergeOption is a functional option for configuring merges
type MergeOption func(*Config)

// WithStrategy sets the merge strategy
func WithStrategy(s Strategy) MergeOption {
	return func(c *Config) {
		c.Strategy = s
	}
}

// WithMode sets the merge mode
func WithMode(m MergeMode) MergeOption {
	return func(c *Config) {
		c.Mode = m
	}
}

// WithMessage sets the merge commit message
func WithMessage(msg string) MergeOption {
	return func(c *Config) {
		c.Message = msg
	}
}

// WithNoCommit configures merge to not create a commit
func WithNoCommit() MergeOption {
	return func(c *Config) {
		c.Mode = ModeNoCommit
	}
}

// WithSquash configures merge to squash commits
func WithSquash() MergeOption {
	return func(c *Config) {
		c.Mode = ModeSquash
	}
}

// WithFastForwardOnly only allows fast-forward merges
func WithFastForwardOnly() MergeOption {
	return func(c *Config) {
		c.Mode = ModeFastForwardOnly
	}
}

// WithNoFastForward always creates a merge commit
func WithNoFastForward() MergeOption {
	return func(c *Config) {
		c.Mode = ModeNoFastForward
	}
}

// WithAllowUnrelatedHistories allows merging unrelated histories
func WithAllowUnrelatedHistories() MergeOption {
	return func(c *Config) {
		c.AllowUnrelatedHistories = true
	}
}

// WithVerbose enables verbose output
func WithVerbose() MergeOption {
	return func(c *Config) {
		c.Verbose = true
	}
}

// WithConflictResolution sets the conflict resolution strategy
func WithConflictResolution(cr ConflictResolution) MergeOption {
	return func(c *Config) {
		c.ConflictResolution = cr
	}
}
