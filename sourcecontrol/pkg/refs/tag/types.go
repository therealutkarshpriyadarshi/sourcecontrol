package tag

import (
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

// TagType represents the type of tag
type TagType int

const (
	// Lightweight tags are simple references to commits
	Lightweight TagType = iota
	// Annotated tags are full objects with metadata
	Annotated
	// Signed tags are annotated tags with GPG signature
	Signed
)

// Tag represents a Git tag
type Tag struct {
	Name       string              // Tag name (e.g., "v1.0.0")
	SHA        objects.ObjectHash  // SHA-1 hash of the tagged object
	Type       TagType             // Type of tag (lightweight, annotated, signed)
	Message    string              // Tag message (for annotated/signed tags)
	Tagger     *Person             // Person who created the tag (for annotated/signed tags)
	ObjectType string              // Type of tagged object (commit, tree, blob)
	Signature  string              // GPG signature (for signed tags)
}

// Person represents a person in Git (author, committer, tagger)
type Person struct {
	Name  string    // Person's name
	Email string    // Person's email
	When  time.Time // When the action occurred
}

// TagInfo represents minimal tag information for listing
type TagInfo struct {
	Name    string             // Tag name
	SHA     objects.ObjectHash // SHA of the tagged object
	Type    TagType            // Type of tag
	Message string             // First line of tag message
}

// CreateOption is a functional option for tag creation
type CreateOption func(*CreateOptions)

// CreateOptions holds options for creating a tag
type CreateOptions struct {
	Force      bool   // Force creation even if tag exists
	Message    string // Tag message for annotated tags
	Annotate   bool   // Create an annotated tag
	Sign       bool   // Create a signed tag
	LocalUser  string // GPG key to use for signing
	TaggerName string // Tagger name (defaults to user.name)
	TaggerEmail string // Tagger email (defaults to user.email)
}

// DeleteOption is a functional option for tag deletion
type DeleteOption func(*DeleteOptions)

// DeleteOptions holds options for deleting a tag
type DeleteOptions struct {
	// Currently no specific delete options
}

// ListOption is a functional option for tag listing
type ListOption func(*ListOptions)

// ListOptions holds options for listing tags
type ListOptions struct {
	Pattern string // Filter tags by pattern (e.g., "v1.*")
	Sort    string // Sort order: name, version, date
	Limit   int    // Maximum number of tags to return
}

// WithForceCreate forces tag creation even if it already exists
func WithForceCreate() CreateOption {
	return func(opts *CreateOptions) {
		opts.Force = true
	}
}

// WithMessage sets the tag message for annotated tags
func WithMessage(message string) CreateOption {
	return func(opts *CreateOptions) {
		opts.Message = message
		opts.Annotate = true // Automatically make it annotated
	}
}

// WithAnnotate creates an annotated tag
func WithAnnotate() CreateOption {
	return func(opts *CreateOptions) {
		opts.Annotate = true
	}
}

// WithSign creates a signed tag
func WithSign(keyID string) CreateOption {
	return func(opts *CreateOptions) {
		opts.Sign = true
		opts.Annotate = true // Signed tags are annotated
		opts.LocalUser = keyID
	}
}

// WithTagger sets the tagger information
func WithTagger(name, email string) CreateOption {
	return func(opts *CreateOptions) {
		opts.TaggerName = name
		opts.TaggerEmail = email
	}
}

// WithPattern filters tags by pattern
func WithPattern(pattern string) ListOption {
	return func(opts *ListOptions) {
		opts.Pattern = pattern
	}
}

// WithSort sets the sort order for tag listing
func WithSort(sort string) ListOption {
	return func(opts *ListOptions) {
		opts.Sort = sort
	}
}

// WithLimit limits the number of tags returned
func WithLimit(limit int) ListOption {
	return func(opts *ListOptions) {
		opts.Limit = limit
	}
}
