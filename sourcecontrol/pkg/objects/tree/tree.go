package tree

import (
	"bytes"
	"fmt"
	"io"
	"sort"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

// Tree represents a Git tree object implementation
//
// A tree object represents a directory snapshot in Git. It contains entries for
// files and subdirectories, each with their mode, name, and SHA-1 hash.
//
// Tree Object Structure:
// ┌─────────────────────────────────────────────────────────────────┐
// │ Header: "tree" SPACE size NULL                                  │
// │ Entry 1: mode SPACE name NULL [20-byte SHA-1]                   │
// │ Entry 2: mode SPACE name NULL [20-byte SHA-1]                   │
// │ ...                                                             │
// │ Entry N: mode SPACE name NULL [20-byte SHA-1]                   │
// └─────────────────────────────────────────────────────────────────┘
//
// Example tree object content (without header):
// "100644 README.md\0[20 bytes]040000 src\0[20 bytes]100755 build.sh\0[20 bytes]"
//
// Tree objects are essential for Git's content tracking because they:
// 1. Preserve directory structure and file organization
// 2. Track file permissions and types
// 3. Enable efficient diff calculations between directory states
// 4. Form the backbone of commit objects (each commit points to a root tree)
//
// Sorting Rules:
// Git sorts tree entries in a specific way to ensure deterministic hashes:
// - Entries are sorted lexicographically by name
// - Directories are treated as if they have a trailing "/"
// - This ensures that "file" comes before "file.txt" and "dir/" comes before "dir2"
type Tree struct {
	entries []*TreeEntry
	hash    *objects.ObjectHash
}

// NewTree creates a new Tree object with the given entries
func NewTree(entries []*TreeEntry) *Tree {
	tree := &Tree{
		entries: entries,
		hash:    nil,
	}
	tree.sortEntries()
	return tree
}

func NewEmptyTree() *Tree {
	return &Tree{
		entries: []*TreeEntry{},
		hash:    nil,
	}
}

// ParseTree parses a tree object from serialized data (with header)
func ParseTree(data []byte) (*Tree, error) {
	content, err := objects.ParseSerializedObject(data, objects.TreeType)
	if err != nil {
		return nil, err
	}

	entries, err := parseEntries(content.Bytes())
	if err != nil {
		return nil, err
	}

	tree := &Tree{
		entries: entries,
		hash:    nil,
	}
	tree.sortEntries()

	hash := objects.NewObjectHash(objects.SerializedObject(data))
	tree.hash = &hash

	return tree, nil
}

// Type returns the object type
func (t *Tree) Type() objects.ObjectType {
	return objects.TreeType
}

// Content returns the raw content of the tree (serialized entries without header)
func (t *Tree) Content() (objects.ObjectContent, error) {
	data, err := t.serializeContent()
	if err != nil {
		return nil, err
	}
	return objects.ObjectContent(data), nil
}

// Hash returns the SHA-1 hash of the tree
func (t *Tree) Hash() (objects.ObjectHash, error) {
	if t.hash != nil {
		return *t.hash, nil
	}

	content, err := t.Content()
	if err != nil {
		return "", fmt.Errorf("failed to get content: %w", err)
	}

	hash := objects.ComputeObjectHash(objects.TreeType, content)
	t.hash = &hash
	return hash, nil
}

// RawHash returns the SHA-1 hash as a 20-byte array
func (t *Tree) RawHash() (objects.RawHash, error) {
	hash, err := t.Hash()
	if err != nil {
		return objects.RawHash{}, err
	}
	return hash.Raw()
}

// Size returns the size of the content in bytes
func (t *Tree) Size() (objects.ObjectSize, error) {
	content, err := t.Content()
	if err != nil {
		return 0, err
	}
	return content.Size(), nil
}

// Serialize writes the tree in Git's storage format
func (t *Tree) Serialize(w io.Writer) error {
	content, err := t.Content()
	if err != nil {
		return fmt.Errorf("failed to get content: %w", err)
	}

	serialized := objects.NewSerializedObject(objects.TreeType, content)

	if _, err := w.Write(serialized.Bytes()); err != nil {
		return fmt.Errorf("failed to write tree: %w", err)
	}

	return nil
}

// String returns a human-readable representation
func (t *Tree) String() string {
	hash, err := t.Hash()
	if err != nil {
		return fmt.Sprintf("Tree{entries: %d, error: %v}", len(t.entries), err)
	}
	size, _ := t.Size()
	return fmt.Sprintf("Tree{entries: %d, size: %s, hash: %s}", len(t.entries), size, hash.Short())
}

// Entries returns a copy of the tree entries to prevent external modification
func (t *Tree) Entries() []*TreeEntry {
	entries := make([]*TreeEntry, len(t.entries))
	copy(entries, t.entries)
	return entries
}

// IsEmpty returns true if the tree has no entries
func (t *Tree) IsEmpty() bool {
	return len(t.entries) == 0
}

// sortEntries sorts the entries according to Git's sorting rules
func (t *Tree) sortEntries() {
	sort.Slice(t.entries, func(i, j int) bool {
		return t.entries[i].CompareTo(t.entries[j]) < 0
	})
}

// serializeContent serializes all entries into a byte array
func (t *Tree) serializeContent() ([]byte, error) {
	if len(t.entries) == 0 {
		return []byte{}, nil
	}

	var buf bytes.Buffer
	for _, entry := range t.entries {
		if err := entry.Serialize(&buf); err != nil {
			return nil, fmt.Errorf("failed to serialize tree entry: %w", err)
		}
	}
	return buf.Bytes(), nil
}

// parseEntries parses tree entries from content
func parseEntries(content []byte) ([]*TreeEntry, error) {
	var entries []*TreeEntry
	reader := bytes.NewReader(content)

	for reader.Len() > 0 {
		e := &TreeEntry{}
		err := e.Deserialize(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to parse tree entry: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, nil
}
