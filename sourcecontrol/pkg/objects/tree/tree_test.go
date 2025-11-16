package tree

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

func TestNewTree(t *testing.T) {
	sha := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"

	entry1, _ := NewTreeEntryFromStrings("100644", "README.md", sha)
	entry2, _ := NewTreeEntryFromStrings("040000", "src", sha)
	entry3, _ := NewTreeEntryFromStrings("100755", "build.sh", sha)

	entries := []*TreeEntry{entry3, entry1, entry2} // Unsorted order

	tree := NewTree(entries)

	if tree == nil {
		t.Fatal("NewTree() returned nil")
	}

	if len(tree.Entries()) != 3 {
		t.Errorf("Entries() length = %v, want 3", len(tree.Entries()))
	}

	// Verify entries are sorted
	if tree.Entries()[0].Name() != "README.md" {
		t.Errorf("First entry name = %v, want README.md", tree.Entries()[0].Name())
	}
	if tree.Entries()[1].Name() != "build.sh" {
		t.Errorf("Second entry name = %v, want build.sh", tree.Entries()[1].Name())
	}
	if tree.Entries()[2].Name() != "src" {
		t.Errorf("Third entry name = %v, want src", tree.Entries()[2].Name())
	}
}

func TestTreeType(t *testing.T) {
	tree := NewTree([]*TreeEntry{})
	if tree.Type() != objects.TreeType {
		t.Errorf("Type() = %v, want %v", tree.Type(), objects.TreeType)
	}
}

func TestTreeIsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		entries []*TreeEntry
		isEmpty bool
	}{
		{
			name:    "empty tree",
			entries: []*TreeEntry{},
			isEmpty: true,
		},
		{
			name: "non-empty tree",
			entries: []*TreeEntry{
				mustCreateEntry("100644", "file.txt", "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"),
			},
			isEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewTree(tt.entries)
			if tree.IsEmpty() != tt.isEmpty {
				t.Errorf("IsEmpty() = %v, want %v", tree.IsEmpty(), tt.isEmpty)
			}
		})
	}
}

func TestTreeContent(t *testing.T) {
	sha := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"
	entry, _ := NewTreeEntryFromStrings("100644", "test.txt", sha)

	tree := NewTree([]*TreeEntry{entry})
	content, err := tree.Content()
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}

	// Verify content is the serialized entry
	var expectedBuf bytes.Buffer
	_ = entry.Serialize(&expectedBuf)
	expectedContent := expectedBuf.Bytes()
	if !bytes.Equal(content.Bytes(), expectedContent) {
		t.Errorf("Content() = %x, want %x", content.Bytes(), expectedContent)
	}
}

func TestTreeSize(t *testing.T) {
	sha := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"
	entry1, _ := NewTreeEntryFromStrings("100644", "a.txt", sha)
	entry2, _ := NewTreeEntryFromStrings("100644", "b.txt", sha)

	tree := NewTree([]*TreeEntry{entry1, entry2})

	content, err := tree.Content()
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	expectedSize := int64(len(content.Bytes()))
	size, err := tree.Size()
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size.Int64() != expectedSize {
		t.Errorf("Size() = %v, want %v", size, expectedSize)
	}
}

func TestTreeHash(t *testing.T) {
	sha := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"
	entry, _ := NewTreeEntryFromStrings("100644", "test.txt", sha)

	tree := NewTree([]*TreeEntry{entry})
	hash1, err := tree.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	// Hash should be consistent
	hash2, err := tree.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash1 != hash2 {
		t.Errorf("Hash() not consistent: %s != %s", hash1, hash2)
	}

	// Hash should be valid (40 hex characters)
	if !hash1.IsValid() {
		t.Errorf("Hash() is not valid: %s", hash1)
	}
}

func TestTreeSerialize(t *testing.T) {
	sha := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"
	entry, _ := NewTreeEntryFromStrings("100644", "test.txt", sha)

	tree := NewTree([]*TreeEntry{entry})

	var buf bytes.Buffer
	err := tree.Serialize(&buf)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Verify header
	data := buf.Bytes()
	nullIndex := bytes.IndexByte(data, objects.NullByte)
	if nullIndex == -1 {
		t.Fatal("Serialize() missing null byte in header")
	}

	header := string(data[:nullIndex])
	expectedHeaderPrefix := "tree "
	if !bytes.HasPrefix([]byte(header), []byte(expectedHeaderPrefix)) {
		t.Errorf("Serialize() header = %q, want prefix %q", header, expectedHeaderPrefix)
	}

	// Verify content after header
	contentBytes := data[nullIndex+1:]
	var expectedBuf bytes.Buffer
	_ = entry.Serialize(&expectedBuf)
	expectedContent := expectedBuf.Bytes()
	if !bytes.Equal(contentBytes, expectedContent) {
		t.Errorf("Serialize() content = %x, want %x", contentBytes, expectedContent)
	}
}

func TestParseTree(t *testing.T) {
	// Create a tree and serialize it
	sha := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"
	entry1, _ := NewTreeEntryFromStrings("100644", "README.md", sha)
	entry2, _ := NewTreeEntryFromStrings("040000", "src", sha)

	originalTree := NewTree([]*TreeEntry{entry1, entry2})

	var buf bytes.Buffer
	err := originalTree.Serialize(&buf)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Parse it back
	parsedTree, err := ParseTree(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseTree() error = %v", err)
	}

	// Verify entries
	if len(parsedTree.Entries()) != len(originalTree.Entries()) {
		t.Errorf("ParseTree() entries length = %v, want %v", len(parsedTree.Entries()), len(originalTree.Entries()))
	}

	for i, entry := range parsedTree.Entries() {
		originalEntry := originalTree.Entries()[i]
		if entry.Mode() != originalEntry.Mode() {
			t.Errorf("Entry %d mode = %v, want %v", i, entry.Mode(), originalEntry.Mode())
		}
		if entry.Name() != originalEntry.Name() {
			t.Errorf("Entry %d name = %v, want %v", i, entry.Name(), originalEntry.Name())
		}
		if entry.SHA() != originalEntry.SHA() {
			t.Errorf("Entry %d SHA = %v, want %v", i, entry.SHA(), originalEntry.SHA())
		}
	}

	// Verify hash
	parsedHash, err := parsedTree.Hash()
	if err != nil {
		t.Fatalf("parsedTree.Hash() error = %v", err)
	}
	originalHash, err := originalTree.Hash()
	if err != nil {
		t.Fatalf("originalTree.Hash() error = %v", err)
	}
	if parsedHash != originalHash {
		t.Errorf("ParseTree() hash = %s, want %s", parsedHash, originalHash)
	}
}

func TestParseTreeInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "invalid header type",
			data: []byte("blob 10\x00test data"),
		},
		{
			name: "missing null byte",
			data: []byte("tree 10 test data"),
		},
		{
			name: "size mismatch",
			data: []byte("tree 100\x00short"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTree(tt.data)
			if err == nil {
				t.Error("ParseTree() expected error, got nil")
			}
		})
	}
}

func TestTreeEntrySorting(t *testing.T) {
	sha := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"

	// Create entries in random order
	entries := []*TreeEntry{
		mustCreateEntry("100644", "z.txt", sha),
		mustCreateEntry("040000", "a", sha),
		mustCreateEntry("100644", "b.txt", sha),
		mustCreateEntry("040000", "c", sha),
		mustCreateEntry("100755", "a.sh", sha),
	}

	tree := NewTree(entries)

	// Verify sorted order: a, a.sh, b.txt, c, z.txt
	expectedOrder := []string{"a", "a.sh", "b.txt", "c", "z.txt"}
	for i, expectedName := range expectedOrder {
		if tree.Entries()[i].Name().String() != expectedName {
			t.Errorf("Entry %d name = %v, want %v", i, tree.Entries()[i].Name(), expectedName)
		}
	}
}

func TestTreeBaseObjectInterface(t *testing.T) {
	// Ensure Tree implements BaseObject interface
	var _ objects.BaseObject = (*Tree)(nil)

	sha := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"
	entry, _ := NewTreeEntryFromStrings("100644", "test.txt", sha)
	tree := NewTree([]*TreeEntry{entry})

	// Test interface methods
	if tree.Type() != objects.TreeType {
		t.Errorf("Type() = %v, want %v", tree.Type(), objects.TreeType)
	}

	content, err := tree.Content()
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	if content.IsEmpty() && len(tree.Entries()) > 0 {
		t.Error("Content() should not be empty for non-empty tree")
	}

	hash, err := tree.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if !hash.IsValid() {
		t.Errorf("Hash() is not valid: %s", hash)
	}

	size, err := tree.Size()
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size.Int64() != int64(len(content.Bytes())) {
		t.Errorf("Size() = %v, want %v", size, len(content.Bytes()))
	}

	var buf bytes.Buffer
	err = tree.Serialize(&buf)
	if err != nil {
		t.Errorf("Serialize() error = %v", err)
	}

	str := tree.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
}

func TestTreeRoundTrip(t *testing.T) {
	// Create a complex tree
	sha1 := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"
	sha2 := "b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1"
	sha3 := "c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2"

	entries := []*TreeEntry{
		mustCreateEntry("100644", "README.md", sha1),
		mustCreateEntry("040000", "src", sha2),
		mustCreateEntry("100755", "build.sh", sha3),
		mustCreateEntry("120000", "link", sha1),
		mustCreateEntry("160000", "submodule", sha2),
	}

	originalTree := NewTree(entries)

	// Serialize
	var buf bytes.Buffer
	err := originalTree.Serialize(&buf)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Parse back
	parsedTree, err := ParseTree(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseTree() error = %v", err)
	}

	// Compare
	if len(parsedTree.Entries()) != len(originalTree.Entries()) {
		t.Fatalf("Entry count mismatch: got %d, want %d", len(parsedTree.Entries()), len(originalTree.Entries()))
	}

	for i := range originalTree.Entries() {
		orig := originalTree.Entries()[i]
		parsed := parsedTree.Entries()[i]

		if parsed.Mode() != orig.Mode() {
			t.Errorf("Entry %d mode: got %s, want %s", i, parsed.Mode(), orig.Mode())
		}
		if parsed.Name() != orig.Name() {
			t.Errorf("Entry %d name: got %s, want %s", i, parsed.Name(), orig.Name())
		}
		if parsed.SHA() != orig.SHA() {
			t.Errorf("Entry %d SHA: got %s, want %s", i, parsed.SHA(), orig.SHA())
		}
	}

	// Hashes should match
	parsedHash, err := parsedTree.Hash()
	if err != nil {
		t.Fatalf("parsedTree.Hash() error = %v", err)
	}
	originalHash, err := originalTree.Hash()
	if err != nil {
		t.Fatalf("originalTree.Hash() error = %v", err)
	}
	if parsedHash != originalHash {
		t.Errorf("Hash mismatch: got %s, want %s", parsedHash, originalHash)
	}
}

func TestTreeEmptySerialization(t *testing.T) {
	tree := NewTree([]*TreeEntry{})

	var buf bytes.Buffer
	err := tree.Serialize(&buf)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Should have header with size 0
	data := buf.Bytes()
	expected := "tree 0\x00"
	if string(data) != expected {
		t.Errorf("Serialize() = %q, want %q", string(data), expected)
	}

	// Should be parseable
	parsed, err := ParseTree(data)
	if err != nil {
		t.Fatalf("ParseTree() error = %v", err)
	}

	if !parsed.IsEmpty() {
		t.Error("ParseTree() expected empty tree")
	}
}

// Helper function to create entries without error handling in tests
func mustCreateEntry(mode, name, sha string) *TreeEntry {
	entry, err := NewTreeEntryFromStrings(mode, name, sha)
	if err != nil {
		panic(err)
	}
	return entry
}

func TestTreeContentWithMultipleEntries(t *testing.T) {
	sha1 := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"
	sha2 := "b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1"

	entry1, _ := NewTreeEntryFromStrings("100644", "a.txt", sha1)
	entry2, _ := NewTreeEntryFromStrings("100644", "b.txt", sha2)

	tree := NewTree([]*TreeEntry{entry1, entry2})
	content, err := tree.Content()
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}

	// Content should be concatenation of serialized entries
	var buf1, buf2 bytes.Buffer
	_ = entry1.Serialize(&buf1)
	_ = entry2.Serialize(&buf2)
	serialized1 := buf1.Bytes()
	serialized2 := buf2.Bytes()

	expectedContent := append(serialized1, serialized2...)
	if !bytes.Equal(content.Bytes(), expectedContent) {
		t.Errorf("Content() = %x, want %x", content.Bytes(), expectedContent)
	}
}

func TestParseEntriesWithInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "truncated entry",
			data: []byte("100644 test.txt"),
		},
		{
			name: "missing SHA bytes",
			data: []byte("100644 test.txt\x00abc"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseEntries(tt.data)
			if err == nil {
				t.Error("parseEntries() expected error, got nil")
			}
		})
	}
}

func TestTreeEntryInvalidMode(t *testing.T) {
	// Invalid modes should be rejected during parsing
	sha, _ := hex.DecodeString("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0")
	data := append([]byte("999999 test.txt\x00"), sha...)

	entry := &TreeEntry{}
	err := entry.Deserialize(bytes.NewReader(data))
	if err == nil {
		t.Error("Deserialize() expected error for invalid mode, got nil")
	}
}
