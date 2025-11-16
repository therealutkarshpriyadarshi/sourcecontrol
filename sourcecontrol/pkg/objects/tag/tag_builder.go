package tag

import (
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
)

// TagBuilder provides a fluent interface for building tag objects
type TagBuilder struct {
	tag  *Tag
	errs []error
}

// NewTagBuilder creates a new TagBuilder
func NewTagBuilder() *TagBuilder {
	return &TagBuilder{
		tag:  &Tag{},
		errs: make([]error, 0),
	}
}

// Object sets the object SHA for the tag
func (b *TagBuilder) Object(objectSHA string) *TagBuilder {
	hash, err := objects.NewObjectHashFromString(objectSHA)
	if err != nil {
		b.errs = append(b.errs, fmt.Errorf("invalid object SHA: %w", err))
	} else {
		b.tag.ObjectSHA = hash
	}
	return b
}

// ObjectHash sets the object SHA using an ObjectHash
func (b *TagBuilder) ObjectHash(objectSHA objects.ObjectHash) *TagBuilder {
	b.tag.ObjectSHA = objectSHA
	return b
}

// ObjectType sets the type of the tagged object
func (b *TagBuilder) ObjectType(objType objects.ObjectType) *TagBuilder {
	if _, err := objects.ParseObjectType(string(objType)); err != nil {
		b.errs = append(b.errs, fmt.Errorf("invalid object type: %w", err))
	} else {
		b.tag.ObjectType = objType
	}
	return b
}

// Name sets the tag name
func (b *TagBuilder) Name(name string) *TagBuilder {
	b.tag.Name = name
	return b
}

// Tagger sets the tagger information
func (b *TagBuilder) Tagger(tagger *commit.CommitPerson) *TagBuilder {
	if tagger == nil {
		b.errs = append(b.errs, fmt.Errorf("tagger cannot be nil"))
	} else {
		b.tag.Tagger = tagger
	}
	return b
}

// Message sets the tag message
func (b *TagBuilder) Message(message string) *TagBuilder {
	b.tag.Message = message
	return b
}

// Signature sets the GPG signature
func (b *TagBuilder) Signature(signature string) *TagBuilder {
	b.tag.Signature = signature
	return b
}

// Build creates the Tag, returning an error if validation fails
func (b *TagBuilder) Build() (*Tag, error) {
	if len(b.errs) > 0 {
		return nil, fmt.Errorf("tag builder errors: %v", b.errs)
	}

	if err := b.tag.Validate(); err != nil {
		return nil, err
	}

	return b.tag, nil
}
