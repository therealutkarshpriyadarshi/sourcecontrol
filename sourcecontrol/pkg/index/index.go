package index

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// Index represents the Git Index (Staging Area) - the bridge between working directory and commits.
//
// The index is Git's "staging area" - a snapshot of your working directory that
// you're building up for your next commit. It's stored as a binary file at .git/index.
//
// Index File Format:
//
//	┌────────────────────────────────────────┐
//	│ Header (12 bytes)                      │
//	│   Signature: "DIRC" (4 bytes)          │
//	│   Version: 2 (4 bytes)                 │
//	│   Entry Count: N (4 bytes)             │
//	├────────────────────────────────────────┤
//	│ Entries (variable length)              │
//	│   Entry 1                              │
//	│   Entry 2                              │
//	│   ...                                  │
//	│   Entry N                              │
//	├────────────────────────────────────────┤
//	│ Extensions (optional)                  │
//	├────────────────────────────────────────┤
//	│ SHA-1 Checksum (20 bytes)              │
//	└────────────────────────────────────────┘
type Index struct {
	// Version is the index file format version (typically 2)
	Version uint32

	// Entries contains all staged files, sorted by path
	Entries []*Entry

	// entryMap provides O(1) lookup by path
	entryMap map[scpath.RelativePath]*Entry
}

// NewIndex creates a new empty index with the default version.
func NewIndex() *Index {
	return &Index{
		Version:  IndexVersion,
		Entries:  make([]*Entry, 0),
		entryMap: make(map[scpath.RelativePath]*Entry),
	}
}

// Write persists the index to disk at the specified path.
// The index is serialized in Git's binary format and includes a SHA-1 checksum.
//
// Parameters:
//   - path: Absolute path where the index file should be written (typically .git/index)
//
// Returns an error if:
//   - Serialization fails
//   - File cannot be written (permissions, disk full, etc.)
func (idx *Index) Write(path scpath.AbsolutePath) error {
	buf := new(bytes.Buffer)

	if err := idx.Serialize(buf); err != nil {
		return fmt.Errorf("failed to serialize index: %w", err)
	}

	if err := os.WriteFile(path.String(), buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// Add stages a new file or updates an existing entry in the index.
// If an entry with the same path already exists, it is replaced.
// After adding, entries are automatically sorted according to Git's rules.
//
// Parameters:
//   - entry: The entry to add or update
func (idx *Index) Add(entry *Entry) {
	pathKey := entry.Path.Normalize()

	if existingEntry, exists := idx.entryMap[pathKey]; exists {
		*existingEntry = *entry
		idx.entryMap[pathKey] = existingEntry
	} else {
		idx.Entries = append(idx.Entries, entry)
		idx.entryMap[pathKey] = entry
		idx.sort()
	}
}

// Remove unstages a file by removing its entry from the index.
//
// Parameters:
//   - path: Relative path of the file to remove
//
// Returns:
//   - true if the entry was found and removed
//   - false if no entry with that path exists
func (idx *Index) Remove(path scpath.RelativePath) bool {
	normalizedPath := path.Normalize()

	entry, exists := idx.entryMap[normalizedPath]
	if !exists {
		return false
	}

	delete(idx.entryMap, normalizedPath)
	for i, e := range idx.Entries {
		if e == entry { // pointer comparison
			idx.Entries = append(idx.Entries[:i], idx.Entries[i+1:]...)
			return true
		}
	}
	return false
}

// Get retrieves an entry by its path.
//
// Parameters:
//   - path: Relative path of the file to find
//
// Returns:
//   - The entry if found, along with true
//   - nil and false if not found
func (idx *Index) Get(path scpath.RelativePath) (*Entry, bool) {
	normalizedPath := path.Normalize()
	entry, ok := idx.entryMap[normalizedPath]
	return entry, ok
}

// Has checks if a file is currently staged in the index.
//
// Parameters:
//   - path: Relative path of the file to check
//
// Returns true if the file is staged, false otherwise.
func (idx *Index) Has(path scpath.RelativePath) bool {
	_, ok := idx.Get(path)
	return ok
}

// Clear removes all entries from the index, effectively unstaging all files.
func (idx *Index) Clear() {
	idx.Entries = make([]*Entry, 0)
	idx.entryMap = make(map[scpath.RelativePath]*Entry)
}

// Paths returns a slice of all staged file paths.
func (idx *Index) Paths() []scpath.RelativePath {
	paths := make([]scpath.RelativePath, len(idx.Entries))
	for i, e := range idx.Entries {
		paths[i] = e.Path
	}
	return paths
}

// Count returns the total number of staged files in the index.
func (idx *Index) Count() int {
	return len(idx.Entries)
}

// Serialize writes the index in Git's binary format to the provided writer.
// The output includes a SHA-1 checksum of all content for integrity verification.
//
// Format:
//  1. Header (12 bytes)
//  2. All entries (variable length)
//  3. SHA-1 checksum (20 bytes)
//
// Parameters:
//   - w: Writer to output the serialized index
//
// Returns an error if writing fails at any stage.
func (idx *Index) Serialize(w io.Writer) error {
	buf := new(bytes.Buffer)

	if err := idx.writeHeader(buf); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	for _, entry := range idx.Entries {
		if err := entry.Serialize(buf); err != nil {
			return fmt.Errorf("failed to serialize entry %s: %w", entry.Path, err)
		}
	}

	content := buf.Bytes()
	checksum := sha1.Sum(content)

	if _, err := w.Write(content); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}
	if _, err := w.Write(checksum[:]); err != nil {
		return fmt.Errorf("failed to write checksum: %w", err)
	}

	return nil
}

// writeHeader writes the 12-byte index header in Git's format.
//
// Header structure:
//   - Signature: "DIRC" (4 bytes)
//   - Version: uint32 big-endian (4 bytes)
//   - Entry count: uint32 big-endian (4 bytes)
func (idx *Index) writeHeader(w io.Writer) error {
	if _, err := w.Write([]byte(IndexSignature)); err != nil {
		return fmt.Errorf("failed to write signature: %w", err)
	}

	if err := binary.Write(w, binary.BigEndian, idx.Version); err != nil {
		return fmt.Errorf("failed to write version: %w", err)
	}

	entryCount := uint32(len(idx.Entries))
	if err := binary.Write(w, binary.BigEndian, entryCount); err != nil {
		return fmt.Errorf("failed to write entry count: %w", err)
	}

	return nil
}

// Deserialize reads an index from binary data in Git's format.
// The data must include a valid header, entries, and matching SHA-1 checksum.
//
// Parameters:
//   - r: Reader containing the serialized index data
//
// Returns an error if:
//   - Data is too small or corrupted
//   - Checksum doesn't match (data integrity failure)
//   - Header is invalid (wrong signature or unsupported version)
//   - Any entry cannot be deserialized
func (idx *Index) Deserialize(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	if len(data) < IndexHeaderSize+IndexChecksumSize {
		return fmt.Errorf("invalid index file: too small")
	}

	if err := validateChecksum(data); err != nil {
		return err
	}

	content := data[:len(data)-IndexChecksumSize]
	buf := bytes.NewReader(content)
	if err := idx.readHeader(buf); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	idx.entryMap = make(map[scpath.RelativePath]*Entry, len(idx.Entries))

	for i := range idx.Entries {
		entry := &Entry{}
		if _, err := entry.Deserialize(buf); err != nil {
			return fmt.Errorf("failed to deserialize entry %d: %w", i, err)
		}
		idx.Entries[i] = entry
		idx.entryMap[entry.Path.Normalize()] = entry
	}

	return nil
}

// validateChecksum verifies the SHA-1 checksum of the index data.
// This ensures the index file hasn't been corrupted or tampered with.
func validateChecksum(data []byte) error {
	if len(data) < IndexHeaderSize+IndexChecksumSize {
		return fmt.Errorf("invalid index file: too small")
	}

	contentSize := len(data) - IndexChecksumSize
	content := data[:contentSize]
	expectedChecksum := data[contentSize:]
	actualChecksum := sha1.Sum(content)

	if !bytes.Equal(expectedChecksum, actualChecksum[:]) {
		return fmt.Errorf("index checksum mismatch")
	}
	return nil
}

// readHeader reads and validates the 12-byte index header.
// Initializes the Entries slice based on the entry count from the header.
func (idx *Index) readHeader(r io.Reader) error {
	sig := make([]byte, 4)
	if _, err := io.ReadFull(r, sig); err != nil {
		return fmt.Errorf("failed to read signature: %w", err)
	}
	if string(sig) != IndexSignature {
		return fmt.Errorf("invalid index signature: %s", string(sig))
	}

	if err := binary.Read(r, binary.BigEndian, &idx.Version); err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}
	if idx.Version != IndexVersion {
		return fmt.Errorf("unsupported index version: %d", idx.Version)
	}

	var entryCount uint32
	if err := binary.Read(r, binary.BigEndian, &entryCount); err != nil {
		return fmt.Errorf("failed to read entry count: %w", err)
	}

	idx.Entries = make([]*Entry, entryCount)
	return nil
}

// sort sorts entries according to Git's path ordering rules.
// Git sorts entries lexicographically by path, treating directories
// as if they have a trailing '/' character.
//
// This is called automatically after Add operations to maintain
// the correct Git index ordering.
func (idx *Index) sort() {
	sort.Slice(idx.Entries, func(i, j int) bool {
		return idx.Entries[i].CompareTo(idx.Entries[j]) < 0
	})
}
