package objects

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
)

// ObjectContent represents raw object data (before compression, without header)
// This is the actual content of a Git object - for a blob it's the file data,
// for a tree it's the serialized entries, for a commit it's the commit metadata.
type ObjectContent []byte

// CompressedData represents DEFLATE-compressed data
// Git stores all objects in compressed form using DEFLATE algorithm
type CompressedData []byte

// SerializedObject represents an object in Git's serialized format (with header)
// Format: "<type> <size>\0<content>"
// Example: "blob 12\0Hello World!"
type SerializedObject []byte

// ObjectSize represents the size of object content in bytes
type ObjectSize int64

// Bytes returns the underlying byte slice
func (oc ObjectContent) Bytes() []byte {
	return []byte(oc)
}

// String returns the content as a string (useful for text content)
func (oc ObjectContent) String() string {
	return string(oc)
}

// Size returns the size of the content in bytes
func (oc ObjectContent) Size() ObjectSize {
	return ObjectSize(len(oc))
}

// IsEmpty returns true if the content is empty
func (oc ObjectContent) IsEmpty() bool {
	return len(oc) == 0
}

// Compress compresses the content using DEFLATE algorithm
// Returns the compressed data or an error if compression fails
func (oc ObjectContent) Compress() (CompressedData, error) {
	if oc.IsEmpty() {
		return CompressedData{}, nil
	}

	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed to create compressor: %w", err)
	}

	if _, err := w.Write(oc); err != nil {
		w.Close()
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize compression: %w", err)
	}

	return CompressedData(buf.Bytes()), nil
}

// Bytes returns the underlying byte slice
func (cd CompressedData) Bytes() []byte {
	return []byte(cd)
}

// Size returns the size of the compressed data in bytes
func (cd CompressedData) Size() ObjectSize {
	return ObjectSize(len(cd))
}

// IsEmpty returns true if the compressed data is empty
func (cd CompressedData) IsEmpty() bool {
	return len(cd) == 0
}

// Decompress decompresses the DEFLATE-compressed data
// Returns the original content or an error if decompression fails
func (cd CompressedData) Decompress() (ObjectContent, error) {
	if cd.IsEmpty() {
		return ObjectContent{}, nil
	}

	r := flate.NewReader(bytes.NewReader(cd))
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	return ObjectContent(data), nil
}

// CompressionRatio returns the compression ratio (original size / compressed size)
// A ratio > 1 means compression was effective
func (cd CompressedData) CompressionRatio(originalSize ObjectSize) float64 {
	if cd.IsEmpty() || originalSize == 0 {
		return 1.0
	}
	return float64(originalSize) / float64(cd.Size())
}

// Bytes returns the underlying byte slice
func (so SerializedObject) Bytes() []byte {
	return []byte(so)
}

// Size returns the size of the serialized object in bytes
func (so SerializedObject) Size() ObjectSize {
	return ObjectSize(len(so))
}

// IsEmpty returns true if the serialized object is empty
func (so SerializedObject) IsEmpty() bool {
	return len(so) == 0
}

// ParseHeader parses the header of a serialized object
// Returns the object type, content size, and the offset where content starts
// Format: "<type> <size>\0<content>"
func (so SerializedObject) ParseHeader() (ObjectType, ObjectSize, int, error) {
	data := []byte(so)
	nullIndex := bytes.IndexByte(data, NullByte)

	if nullIndex == -1 {
		return "", 0, 0, fmt.Errorf("invalid object header: missing null byte")
	}

	spaceIndex := bytes.IndexByte(data[:nullIndex], SpaceByte)
	if spaceIndex == -1 {
		return "", 0, 0, fmt.Errorf("invalid object header: missing space")
	}

	typeBytes := data[:spaceIndex]
	sizeBytes := data[spaceIndex+1 : nullIndex]

	objType, err := ParseObjectType(string(typeBytes))
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid object type: %w", err)
	}

	var size int64
	_, err = fmt.Sscanf(string(sizeBytes), "%d", &size)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid size in header: %w", err)
	}

	return objType, ObjectSize(size), nullIndex + 1, nil
}

// Content extracts the content portion from a serialized object (without header)
func (so SerializedObject) Content() (ObjectContent, error) {
	_, expectedSize, contentStart, err := so.ParseHeader()
	if err != nil {
		return nil, err
	}

	content := []byte(so)[contentStart:]
	if ObjectSize(len(content)) != expectedSize {
		return nil, fmt.Errorf("content size mismatch: expected %d, got %d", expectedSize, len(content))
	}

	return ObjectContent(content), nil
}

// Type returns the object type from the header
func (so SerializedObject) Type() (ObjectType, error) {
	objType, _, _, err := so.ParseHeader()
	return objType, err
}

// Compress compresses the entire serialized object
func (so SerializedObject) Compress() (CompressedData, error) {
	return ObjectContent(so).Compress()
}

// NewSerializedObject creates a new serialized object from type and content
func NewSerializedObject(objType ObjectType, content ObjectContent) SerializedObject {
	header := CreateHeader(objType, int64(content.Size()))
	fullData := append(header, content.Bytes()...)
	return SerializedObject(fullData)
}

// IsValid returns true if the size is non-negative
func (os ObjectSize) IsValid() bool {
	return os >= 0
}

// String returns a human-readable size string
func (os ObjectSize) String() string {
	return formatBytes(int64(os))
}

// Int64 returns the size as an int64
func (os ObjectSize) Int64() int64 {
	return int64(os)
}

// formatBytes formats bytes into human-readable format (B, KiB, MiB, etc.)
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
