// Package blob implements Git blob objects, which represent file content in the Git object database.
// Blobs are the fundamental storage unit for file data in Git's content-addressable storage.
package blob

import (
	"fmt"
	"io"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

// Blob represents a Git blob object, which stores the content of a file.
// It contains the raw content and a lazily-computed SHA-1 hash.
// Blobs are immutable once created and are identified by their content hash.
type Blob struct {
	content objects.ObjectContent
	hash    *objects.ObjectHash
}

// NewBlob creates a new Blob object from raw data.
// The hash is computed lazily on first access for performance.
//
// Parameters:
//   - data: The raw file content to store in the blob
//
// Returns:
//   - A pointer to a new Blob instance
func NewBlob(data []byte) *Blob {
	return &Blob{
		content: objects.ObjectContent(data),
		hash:    nil, // Lazy computation
	}
}

// ParseBlob parses a blob from serialized data (with header).
// The serialized data should be in Git's object storage format:
// "<type> <size>\0<content>"
//
// Parameters:
//   - data: The serialized blob data including the Git object header
//
// Returns:
//   - A pointer to the parsed Blob instance
//   - An error if the data is invalid or doesn't represent a blob
func ParseBlob(data []byte) (*Blob, error) {
	content, err := objects.ParseSerializedObject(data, objects.BlobType)
	if err != nil {
		return nil, err
	}

	hash := objects.NewObjectHash(objects.SerializedObject(data))
	return &Blob{
		content: content,
		hash:    &hash,
	}, nil
}

// Type returns the object type identifier for this blob.
// Always returns objects.BlobType.
//
// Returns:
//   - The ObjectType constant identifying this as a blob
func (b *Blob) Type() objects.ObjectType {
	return objects.BlobType
}

// Content returns the raw content of the blob.
// This is the actual file data stored in the blob.
//
// Returns:
//   - The blob's content as ObjectContent
//   - An error (currently always nil, but included for interface compatibility)
func (b *Blob) Content() (objects.ObjectContent, error) {
	return b.content, nil
}

// Hash returns the SHA-1 hash of the blob.
// The hash is computed lazily on first access and then cached.
// The hash is computed over the serialized object format: "<type> <size>\0<content>"
//
// Returns:
//   - The 40-character hexadecimal SHA-1 hash
//   - An error if hash computation fails
func (b *Blob) Hash() (objects.ObjectHash, error) {
	if b.hash != nil {
		return *b.hash, nil
	}

	hash := objects.ComputeObjectHash(objects.BlobType, b.content)
	b.hash = &hash
	return hash, nil
}

// RawHash returns the SHA-1 hash as a 20-byte array.
// This is useful for compact storage or binary operations.
//
// Returns:
//   - A 20-byte array containing the raw SHA-1 hash
//   - An error if hash computation fails
func (b *Blob) RawHash() (objects.RawHash, error) {
	hash, err := b.Hash()
	if err != nil {
		return objects.RawHash{}, err
	}
	return hash.Raw()
}

// Size returns the size of the content in bytes.
// This is the size of the actual content, not including the Git object header.
//
// Returns:
//   - The size of the blob content in bytes
//   - An error (currently always nil, but included for interface compatibility)
func (b *Blob) Size() (objects.ObjectSize, error) {
	return b.content.Size(), nil
}

// Serialize writes the blob in Git's storage format to the provided writer.
// The format is: "blob <size>\0<content>"
//
// Parameters:
//   - w: The io.Writer to write the serialized blob to
//
// Returns:
//   - An error if writing fails, nil otherwise
func (b *Blob) Serialize(w io.Writer) error {
	serialized := objects.NewSerializedObject(objects.BlobType, b.content)

	if _, err := w.Write(serialized.Bytes()); err != nil {
		return fmt.Errorf("failed to write blob: %w", err)
	}

	return nil
}

// String returns a human-readable representation of the blob.
// Includes the content size and a shortened version of the hash.
//
// Returns:
//   - A string in the format "Blob{size: <size>, hash: <short_hash>}"
//   - If hash computation fails, returns "Blob{size: <size>, error: <error>}"
func (b *Blob) String() string {
	hash, err := b.Hash()
	if err != nil {
		return fmt.Sprintf("Blob{size: %s, error: %v}", b.content.Size(), err)
	}
	return fmt.Sprintf("Blob{size: %s, hash: %s}", b.content.Size(), hash.Short())
}
