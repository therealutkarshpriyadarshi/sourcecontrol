package tag

import (
	"fmt"
	"io"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
)

// Tag represents a Git tag object implementation.
//
// A tag object is used to mark a specific object (usually a commit) with a name.
// It contains:
// - A reference to the tagged object
// - The type of object being tagged
// - The tag name
// - Tagger information (who created the tag)
// - A tag message
// - Optional GPG signature for signed tags
//
// Tag Object Structure:
// ┌─────────────────────────────────────────────────────────────────┐
// │ Header: "tag" SPACE size NULL                                   │
// │ "object" SPACE object-sha LF                                    │
// │ "type" SPACE object-type LF                                     │
// │ "tag" SPACE tag-name LF                                         │
// │ "tagger" SPACE name SPACE email SPACE timestamp SPACE tz LF     │
// │ LF                                                              │
// │ tag-message                                                     │
// │ (optional GPG signature)                                        │
// └─────────────────────────────────────────────────────────────────┘
//
// Example tag object content:
// object 4b825dc642cb6eb9a060e54bf8d69288fbee4904
// type commit
// tag v1.0.0
// tagger John Doe <john@example.com> 1609459200 +0000
//
// Release version 1.0.0
type Tag struct {
	ObjectSHA  objects.ObjectHash    // SHA-1 hash of the tagged object
	ObjectType objects.ObjectType    // Type of the tagged object (commit, tree, blob)
	Name       string                // Tag name (e.g., "v1.0.0")
	Tagger     *commit.CommitPerson  // Person who created the tag
	Message    string                // Tag message
	Signature  string                // GPG signature (for signed tags)
	hash       *objects.ObjectHash   // cached hash
}

// Validate checks that all required fields are present
func (t *Tag) Validate() error {
	if t.ObjectSHA == "" {
		return fmt.Errorf("object SHA is required")
	}
	if err := t.ObjectSHA.Validate(); err != nil {
		return fmt.Errorf("invalid object SHA: %w", err)
	}
	if t.ObjectType == "" {
		return fmt.Errorf("object type is required")
	}
	if _, err := objects.ParseObjectType(string(t.ObjectType)); err != nil {
		return fmt.Errorf("invalid object type: %w", err)
	}
	if t.Name == "" {
		return fmt.Errorf("tag name is required")
	}
	if t.Tagger == nil {
		return fmt.Errorf("tagger is required")
	}
	return nil
}

// Type returns the object type
func (t *Tag) Type() objects.ObjectType {
	return objects.TagType
}

// Content returns the raw content of the tag (without header)
func (t *Tag) Content() (objects.ObjectContent, error) {
	var buf strings.Builder

	// Object line
	buf.WriteString("object ")
	buf.WriteString(t.ObjectSHA.String())
	buf.WriteString("\n")

	// Type line
	buf.WriteString("type ")
	buf.WriteString(string(t.ObjectType))
	buf.WriteString("\n")

	// Tag line
	buf.WriteString("tag ")
	buf.WriteString(t.Name)
	buf.WriteString("\n")

	// Tagger line
	buf.WriteString("tagger ")
	buf.WriteString(t.Tagger.FormatForGit())
	buf.WriteString("\n")

	// Blank line before message
	buf.WriteString("\n")

	// Message
	buf.WriteString(t.Message)

	// Signature (if present)
	if t.Signature != "" {
		if !strings.HasSuffix(t.Message, "\n") {
			buf.WriteString("\n")
		}
		buf.WriteString(t.Signature)
	}

	return objects.ObjectContent(buf.String()), nil
}

// Hash returns the SHA-1 hash of the tag
func (t *Tag) Hash() (objects.ObjectHash, error) {
	if t.hash != nil {
		return *t.hash, nil
	}

	// Calculate hash if not cached
	content, err := t.Content()
	if err != nil {
		return "", fmt.Errorf("failed to get content: %w", err)
	}

	hash := objects.ComputeObjectHash(objects.TagType, content)
	t.hash = &hash
	return hash, nil
}

// RawHash returns the SHA-1 hash as a 20-byte array
func (t *Tag) RawHash() (objects.RawHash, error) {
	hash, err := t.Hash()
	if err != nil {
		return objects.RawHash{}, err
	}
	return hash.Raw()
}

// Size returns the size of the content in bytes
func (t *Tag) Size() (objects.ObjectSize, error) {
	content, err := t.Content()
	if err != nil {
		return 0, err
	}
	return content.Size(), nil
}

// Serialize writes the tag in Git's storage format
func (t *Tag) Serialize(w io.Writer) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("invalid tag: %w", err)
	}

	content, err := t.Content()
	if err != nil {
		return fmt.Errorf("failed to get content: %w", err)
	}

	serialized := objects.NewSerializedObject(objects.TagType, content)

	if _, err := w.Write(serialized.Bytes()); err != nil {
		return fmt.Errorf("failed to write tag: %w", err)
	}

	return nil
}

// String returns a human-readable representation
func (t *Tag) String() string {
	hash, err := t.Hash()
	if err != nil {
		return fmt.Sprintf("Tag{name: %s, object: %s, error: %v}",
			t.Name, t.ObjectSHA.Short(), err)
	}
	return fmt.Sprintf("Tag{hash: %s, name: %s, object: %s, message: %.50s...}",
		hash.Short(), t.Name, t.ObjectSHA.Short(), t.Message)
}

// ParseTag parses a tag object from serialized data (with header)
func ParseTag(data []byte) (*Tag, error) {
	content, err := objects.ParseSerializedObject(data, objects.TagType)
	if err != nil {
		return nil, err
	}

	tag, err := parseTagContent(content.String())
	if err != nil {
		return nil, err
	}

	hash := objects.NewObjectHash(objects.SerializedObject(data))
	tag.hash = &hash
	return tag, nil
}

// parseTagContent parses the tag content (without header)
func parseTagContent(content string) (*Tag, error) {
	lines := strings.Split(content, "\n")
	tag := &Tag{}

	messageStartIndex := -1

	for i, line := range lines {
		// Empty line indicates start of message
		if strings.TrimSpace(line) == "" {
			messageStartIndex = i + 1
			break
		}

		if err := parseTagLine(tag, line); err != nil {
			return nil, err
		}
	}

	if err := tag.Validate(); err != nil {
		return nil, fmt.Errorf("invalid tag: %w", err)
	}

	// Extract message and signature
	if messageStartIndex != -1 && messageStartIndex < len(lines) {
		remainingContent := strings.Join(lines[messageStartIndex:], "\n")

		// Check for GPG signature
		sigStart := strings.Index(remainingContent, "-----BEGIN PGP SIGNATURE-----")
		if sigStart != -1 {
			tag.Message = strings.TrimSpace(remainingContent[:sigStart])
			tag.Signature = strings.TrimSpace(remainingContent[sigStart:])
		} else {
			tag.Message = remainingContent
		}
	}

	return tag, nil
}

// parseTagLine parses a single header line
func parseTagLine(tag *Tag, line string) error {
	switch {
	case strings.HasPrefix(line, "object "):
		if tag.ObjectSHA != "" {
			return fmt.Errorf("multiple object entries found")
		}
		objectSHAStr := strings.TrimPrefix(line, "object ")
		objectSHA, err := objects.NewObjectHashFromString(objectSHAStr)
		if err != nil {
			return fmt.Errorf("invalid object SHA: %w", err)
		}
		tag.ObjectSHA = objectSHA

	case strings.HasPrefix(line, "type "):
		if tag.ObjectType != "" {
			return fmt.Errorf("multiple type entries found")
		}
		typeStr := strings.TrimPrefix(line, "type ")
		objType, err := objects.ParseObjectType(typeStr)
		if err != nil {
			return fmt.Errorf("invalid object type: %w", err)
		}
		tag.ObjectType = objType

	case strings.HasPrefix(line, "tag "):
		if tag.Name != "" {
			return fmt.Errorf("multiple tag name entries found")
		}
		tag.Name = strings.TrimPrefix(line, "tag ")

	case strings.HasPrefix(line, "tagger "):
		if tag.Tagger != nil {
			return fmt.Errorf("multiple tagger entries found")
		}
		taggerData := strings.TrimPrefix(line, "tagger ")
		tagger, err := commit.ParseCommitPerson(taggerData)
		if err != nil {
			return fmt.Errorf("invalid tagger: %w", err)
		}
		tag.Tagger = tagger

	default:
		return fmt.Errorf("unknown header line: %s", line)
	}

	return nil
}

// IsSigned returns true if the tag has a GPG signature
func (t *Tag) IsSigned() bool {
	return t.Signature != ""
}

// ShortSHA returns the first 7 characters of the tag SHA
func (t *Tag) ShortSHA() (objects.ShortHash, error) {
	hash, err := t.Hash()
	if err != nil {
		return "", err
	}
	return hash.Short(), nil
}

// Equal compares two tags for equality
func (t *Tag) Equal(other *Tag) bool {
	if other == nil {
		return false
	}

	if t.ObjectSHA != other.ObjectSHA {
		return false
	}

	if t.ObjectType != other.ObjectType {
		return false
	}

	if t.Name != other.Name {
		return false
	}

	if !t.Tagger.Equal(other.Tagger) {
		return false
	}

	if t.Message != other.Message {
		return false
	}

	return t.Signature == other.Signature
}

// Clone creates a deep copy of the tag
func (t *Tag) Clone() *Tag {
	clone := &Tag{
		ObjectSHA:  t.ObjectSHA,
		ObjectType: t.ObjectType,
		Name:       t.Name,
		Tagger:     &commit.CommitPerson{Name: t.Tagger.Name, Email: t.Tagger.Email, When: t.Tagger.When},
		Message:    t.Message,
		Signature:  t.Signature,
	}
	return clone
}
