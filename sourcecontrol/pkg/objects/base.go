package objects

import (
	"crypto/sha1"
	"fmt"
	"io"
)

// ObjectType represents the type of Git object
type ObjectType string

const (
	BlobType   ObjectType = "blob"
	TreeType   ObjectType = "tree"
	CommitType ObjectType = "commit"
	TagType    ObjectType = "tag"
)

const (
	NullByte  = byte(0)
	SpaceByte = byte(' ')
)

// String implements the Stringer interface
func (o ObjectType) String() string {
	return string(o)
}

// BaseObject represents a Git object interface
type BaseObject interface {
	// Type returns the object type
	Type() ObjectType

	// Content returns the raw content of the object
	Content() (ObjectContent, error)

	// Hash returns the SHA-1 hash of the object
	Hash() (ObjectHash, error)

	// RawHash returns the SHA-1 hash as a 20-byte array
	RawHash() (RawHash, error)

	// Size returns the size of the content in bytes
	Size() (ObjectSize, error)

	// Serialize writes the object in Git's storage format
	Serialize(w io.Writer) error

	// String returns a human-readable representation
	String() string
}

// ParseObjectType converts a string to ObjectType
func ParseObjectType(s string) (ObjectType, error) {
	switch ObjectType(s) {
	case BlobType, TreeType, CommitType, TagType:
		return ObjectType(s), nil
	default:
		return "", fmt.Errorf("unknown object type: %s", s)
	}
}

// CreateSha creates a SHA-1 hash from data
// Deprecated: Use ComputeHash instead
func CreateSha(data []byte) [20]byte {
	return sha1.Sum(data)
}

// ParseSerializedObject parses a serialized object and validates the type
func ParseSerializedObject(data []byte, expectedType ObjectType) (ObjectContent, error) {
	serialized := SerializedObject(data)

	objType, err := serialized.Type()
	if err != nil {
		return nil, err
	}

	if objType != expectedType {
		return nil, fmt.Errorf("object type mismatch: expected %s, got %s", expectedType, objType)
	}

	return serialized.Content()
}

func CreateHeader(ot ObjectType, contentSize int64) []byte {
	header := fmt.Sprintf("%s %d%c", ot.String(), contentSize, NullByte)
	return []byte(header)
}
