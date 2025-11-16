package index

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/common"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// Helper functions

func createTestEntry(path string, hash string) *Entry {
	objHash, _ := objects.ParseObjectHash(hash)
	entry := NewEntry(mustRelativePath(path))
	entry.BlobHash = objHash
	entry.SizeInBytes = 100
	entry.Mode = FileModeRegular
	entry.ModificationTime = common.NewTimestamp(1234567890, 0)
	entry.CreationTime = common.NewTimestamp(1234567890, 0)
	return entry
}

func mustRelativePath(path string) scpath.RelativePath {
	p, err := scpath.NewRelativePath(path)
	if err != nil {
		panic(err)
	}
	return p
}

func mustAbsolutePath(path string) scpath.AbsolutePath {
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return scpath.AbsolutePath(absPath)
}

func createTestHash(seed string) string {
	h := sha1.Sum([]byte(seed))
	return hex.EncodeToString(h[:])
}

// TestNewIndex tests the NewIndex constructor
func TestNewIndex(t *testing.T) {
	idx := NewIndex()

	if idx == nil {
		t.Fatal("NewIndex returned nil")
	}

	if idx.Version != IndexVersion {
		t.Errorf("expected version %d, got %d", IndexVersion, idx.Version)
	}

	if idx.Entries == nil {
		t.Fatal("Entries should not be nil")
	}

	if len(idx.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(idx.Entries))
	}

	if idx.Count() != 0 {
		t.Errorf("expected Count() to return 0, got %d", idx.Count())
	}
}

// TestIndexAdd tests adding entries to the index
func TestIndexAdd(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected int
	}{
		{
			name:     "add single entry",
			paths:    []string{"file1.txt"},
			expected: 1,
		},
		{
			name:     "add multiple entries",
			paths:    []string{"file1.txt", "file2.txt", "file3.txt"},
			expected: 3,
		},
		{
			name:     "add nested paths",
			paths:    []string{"src/main.go", "src/util/helper.go", "README.md"},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := NewIndex()

			for _, path := range tt.paths {
				entry := createTestEntry(path, createTestHash(path))
				idx.Add(entry)
			}

			if idx.Count() != tt.expected {
				t.Errorf("expected %d entries, got %d", tt.expected, idx.Count())
			}
		})
	}
}

// TestIndexAddDuplicate tests that adding duplicate paths updates the entry
func TestIndexAddDuplicate(t *testing.T) {
	idx := NewIndex()
	path := "file.txt"

	// Add first entry
	entry1 := createTestEntry(path, createTestHash("hash1"))
	entry1.SizeInBytes = 100
	idx.Add(entry1)

	if idx.Count() != 1 {
		t.Fatalf("expected 1 entry after first add, got %d", idx.Count())
	}

	// Add duplicate with different data
	entry2 := createTestEntry(path, createTestHash("hash2"))
	entry2.SizeInBytes = 200
	idx.Add(entry2)

	if idx.Count() != 1 {
		t.Errorf("expected 1 entry after duplicate add, got %d", idx.Count())
	}

	// Verify the entry was updated
	retrieved, ok := idx.Get(mustRelativePath(path))
	if !ok {
		t.Fatal("failed to retrieve entry")
	}

	if retrieved.SizeInBytes != 200 {
		t.Errorf("expected size 200, got %d", retrieved.SizeInBytes)
	}
}

// TestIndexRemove tests removing entries from the index
func TestIndexRemove(t *testing.T) {
	idx := NewIndex()

	// Add test entries
	paths := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, path := range paths {
		idx.Add(createTestEntry(path, createTestHash(path)))
	}

	tests := []struct {
		name         string
		pathToRemove string
		shouldRemove bool
		expectedLen  int
	}{
		{
			name:         "remove existing entry",
			pathToRemove: "file2.txt",
			shouldRemove: true,
			expectedLen:  2,
		},
		{
			name:         "remove non-existing entry",
			pathToRemove: "nonexistent.txt",
			shouldRemove: false,
			expectedLen:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			removed := idx.Remove(mustRelativePath(tt.pathToRemove))

			if removed != tt.shouldRemove {
				t.Errorf("expected Remove to return %v, got %v", tt.shouldRemove, removed)
			}

			if idx.Count() != tt.expectedLen {
				t.Errorf("expected %d entries, got %d", tt.expectedLen, idx.Count())
			}
		})
	}
}

// TestIndexGet tests retrieving entries from the index
func TestIndexGet(t *testing.T) {
	idx := NewIndex()

	// Add test entries
	testHash := createTestHash("test")
	entry := createTestEntry("test.txt", testHash)
	idx.Add(entry)

	tests := []struct {
		name        string
		path        string
		shouldExist bool
	}{
		{
			name:        "get existing entry",
			path:        "test.txt",
			shouldExist: true,
		},
		{
			name:        "get non-existing entry",
			path:        "nonexistent.txt",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, ok := idx.Get(mustRelativePath(tt.path))

			if ok != tt.shouldExist {
				t.Errorf("expected ok=%v, got ok=%v", tt.shouldExist, ok)
			}

			if tt.shouldExist && retrieved == nil {
				t.Error("expected non-nil entry for existing path")
			}

			if !tt.shouldExist && retrieved != nil {
				t.Error("expected nil entry for non-existing path")
			}
		})
	}
}

// TestIndexHas tests checking if entries exist in the index
func TestIndexHas(t *testing.T) {
	idx := NewIndex()
	idx.Add(createTestEntry("exists.txt", createTestHash("exists")))

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing path",
			path:     "exists.txt",
			expected: true,
		},
		{
			name:     "non-existing path",
			path:     "missing.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has := idx.Has(mustRelativePath(tt.path))
			if has != tt.expected {
				t.Errorf("expected Has(%s)=%v, got %v", tt.path, tt.expected, has)
			}
		})
	}
}

// TestIndexClear tests clearing all entries from the index
func TestIndexClear(t *testing.T) {
	idx := NewIndex()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		path := fmt.Sprintf("file/file%d.txt", i)
		idx.Add(createTestEntry(path, createTestHash(path)))
	}

	if idx.Count() != 5 {
		t.Fatalf("expected 5 entries before clear, got %d", idx.Count())
	}

	idx.Clear()

	if idx.Count() != 0 {
		t.Errorf("expected 0 entries after clear, got %d", idx.Count())
	}

	if idx.Entries == nil {
		t.Error("Entries slice should not be nil after clear")
	}
}

// TestIndexPaths tests retrieving all paths from the index
func TestIndexPaths(t *testing.T) {
	idx := NewIndex()

	paths := []string{"a.txt", "b.txt", "c.txt"}
	for _, path := range paths {
		idx.Add(createTestEntry(path, createTestHash(path)))
	}

	retrievedPaths := idx.Paths()

	if len(retrievedPaths) != len(paths) {
		t.Fatalf("expected %d paths, got %d", len(paths), len(retrievedPaths))
	}

	// Verify the paths match (order may differ due to sorting)
	pathMap := make(map[string]bool)
	for _, p := range retrievedPaths {
		pathMap[p.String()] = true
	}

	for _, expected := range paths {
		if !pathMap[expected] {
			t.Errorf("expected path %s not found in results", expected)
		}
	}
}

// TestIndexCount tests the Count method
func TestIndexCount(t *testing.T) {
	idx := NewIndex()

	if idx.Count() != 0 {
		t.Errorf("expected 0 for empty index, got %d", idx.Count())
	}

	for i := 1; i <= 10; i++ {
		path := fmt.Sprintf("file/file%d.txt", i)
		idx.Add(createTestEntry(path, createTestHash(path)))

		if idx.Count() != i {
			t.Errorf("expected %d entries, got %d", i, idx.Count())
		}
	}
}

// TestIndexSort tests that entries are sorted correctly
func TestIndexSort(t *testing.T) {
	idx := NewIndex()

	// Add entries in random order
	paths := []string{"z.txt", "a.txt", "m.txt", "b.txt"}
	for _, path := range paths {
		idx.Add(createTestEntry(path, createTestHash(path)))
	}

	// Verify they are sorted
	entries := idx.Entries
	for i := 1; i < len(entries); i++ {
		if entries[i-1].CompareTo(entries[i]) > 0 {
			t.Errorf("entries not sorted: %s should come after %s",
				entries[i-1].Path, entries[i].Path)
		}
	}
}

// TestIndexSerializeDeserialize tests serialization and deserialization
func TestIndexSerializeDeserialize(t *testing.T) {
	originalIdx := NewIndex()

	// Add test entries
	entries := []struct {
		path string
		hash string
	}{
		{"file1.txt", createTestHash("file1")},
		{"src/main.go", createTestHash("main")},
		{"docs/README.md", createTestHash("readme")},
	}

	for _, e := range entries {
		entry := createTestEntry(e.path, e.hash)
		originalIdx.Add(entry)
	}

	// Serialize
	buf := new(bytes.Buffer)
	err := originalIdx.Serialize(buf)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Deserialize
	deserializedIdx := NewIndex()
	err = deserializedIdx.Deserialize(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Verify
	if deserializedIdx.Version != originalIdx.Version {
		t.Errorf("version mismatch: expected %d, got %d",
			originalIdx.Version, deserializedIdx.Version)
	}

	if deserializedIdx.Count() != originalIdx.Count() {
		t.Errorf("entry count mismatch: expected %d, got %d",
			originalIdx.Count(), deserializedIdx.Count())
	}

	// Verify each entry
	for i, origEntry := range originalIdx.Entries {
		deserEntry := deserializedIdx.Entries[i]

		if deserEntry.Path.String() != origEntry.Path.String() {
			t.Errorf("entry %d path mismatch: expected %s, got %s",
				i, origEntry.Path, deserEntry.Path)
		}

		origHashStr := origEntry.BlobHash.String()
		deserHashStr := deserEntry.BlobHash.String()
		if deserHashStr != origHashStr {
			t.Errorf("entry %d hash mismatch: expected %s, got %s",
				i, origHashStr, deserHashStr)
		}
	}
}

// TestIndexSerializeEmpty tests serializing an empty index
func TestIndexSerializeEmpty(t *testing.T) {
	idx := NewIndex()

	buf := new(bytes.Buffer)
	err := idx.Serialize(buf)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Verify header + checksum
	expectedSize := IndexHeaderSize + IndexChecksumSize
	if buf.Len() != expectedSize {
		t.Errorf("expected %d bytes, got %d", expectedSize, buf.Len())
	}

	// Deserialize
	deserializedIdx := NewIndex()
	err = deserializedIdx.Deserialize(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if deserializedIdx.Count() != 0 {
		t.Errorf("expected 0 entries, got %d", deserializedIdx.Count())
	}
}

// TestIndexChecksumValidation tests checksum validation
func TestIndexChecksumValidation(t *testing.T) {
	idx := NewIndex()
	idx.Add(createTestEntry("test.txt", createTestHash("test")))

	buf := new(bytes.Buffer)
	err := idx.Serialize(buf)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Corrupt the checksum
	data := buf.Bytes()
	data[len(data)-1] ^= 0xFF // Flip last byte

	// Try to deserialize
	corruptedIdx := NewIndex()
	err = corruptedIdx.Deserialize(bytes.NewReader(data))
	if err == nil {
		t.Error("expected error for corrupted checksum, got nil")
	}
}

// TestIndexInvalidData tests deserialization with invalid data
func TestIndexInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "too small",
			data: make([]byte, 10),
		},
		{
			name: "invalid signature",
			data: append([]byte("XXXX"), make([]byte, 100)...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := NewIndex()
			err := idx.Deserialize(bytes.NewReader(tt.data))
			if err == nil {
				t.Error("expected error for invalid data, got nil")
			}
		})
	}
}

// TestIndexWriteRead tests writing to and reading from a file
func TestIndexWriteRead(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "index-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "index")

	// Create and populate index
	originalIdx := NewIndex()
	originalIdx.Add(createTestEntry("file1.txt", createTestHash("file1")))
	originalIdx.Add(createTestEntry("file2.txt", createTestHash("file2")))

	// Write to file
	err = originalIdx.Write(mustAbsolutePath(indexPath))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatal("index file was not created")
	}

	// Read from file
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index file: %v", err)
	}

	// Deserialize
	readIdx := NewIndex()
	err = readIdx.Deserialize(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Verify
	if readIdx.Count() != originalIdx.Count() {
		t.Errorf("count mismatch: expected %d, got %d",
			originalIdx.Count(), readIdx.Count())
	}
}

// TestIndexWriteInvalidPath tests writing to an invalid path
func TestIndexWriteInvalidPath(t *testing.T) {
	idx := NewIndex()
	idx.Add(createTestEntry("test.txt", createTestHash("test")))

	// Try to write to an invalid path
	invalidPath := filepath.Join("/nonexistent/directory/that/does/not/exist", "index")
	err := idx.Write(mustAbsolutePath(invalidPath))
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

// TestIndexHeaderFormat tests the header format
func TestIndexHeaderFormat(t *testing.T) {
	idx := NewIndex()
	idx.Add(createTestEntry("test.txt", createTestHash("test")))

	buf := new(bytes.Buffer)
	err := idx.writeHeader(buf)
	if err != nil {
		t.Fatalf("writeHeader failed: %v", err)
	}

	if buf.Len() != IndexHeaderSize {
		t.Errorf("expected header size %d, got %d", IndexHeaderSize, buf.Len())
	}

	// Verify signature
	data := buf.Bytes()
	signature := string(data[0:4])
	if signature != IndexSignature {
		t.Errorf("expected signature %s, got %s", IndexSignature, signature)
	}
}

// TestValidateChecksum tests the checksum validation function
func TestValidateChecksum(t *testing.T) {
	// Create valid data with checksum
	content := []byte("test content for checksum validation")
	checksum := sha1.Sum(content)
	validData := append(content, checksum[:]...)

	// Test valid checksum
	err := validateChecksum(validData)
	if err != nil {
		t.Errorf("expected no error for valid checksum, got: %v", err)
	}

	// Test invalid checksum
	invalidData := append(content, []byte("invalid checksum data")...)
	err = validateChecksum(invalidData)
	if err == nil {
		t.Error("expected error for invalid checksum, got nil")
	}

	// Test data too small
	smallData := []byte("small")
	err = validateChecksum(smallData)
	if err == nil {
		t.Error("expected error for data too small, got nil")
	}
}

// TestIndexLargeNumberOfEntries tests index with many entries
func TestIndexLargeNumberOfEntries(t *testing.T) {
	idx := NewIndex()

	// Add 1000 entries (need unique paths, not just 10)
	numEntries := 1000
	for i := 0; i < numEntries; i++ {
		path := fmt.Sprintf("dir/file%04d.txt", i)
		idx.Add(createTestEntry(path, createTestHash(path)))
	}

	if idx.Count() != numEntries {
		t.Errorf("expected %d entries, got %d", numEntries, idx.Count())
	}

	// Serialize and deserialize
	buf := new(bytes.Buffer)
	err := idx.Serialize(buf)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	deserializedIdx := NewIndex()
	err = deserializedIdx.Deserialize(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if deserializedIdx.Count() != numEntries {
		t.Errorf("after deserialize: expected %d entries, got %d",
			numEntries, deserializedIdx.Count())
	}
}

// TestIndexPathNormalization tests that paths are normalized
func TestIndexPathNormalization(t *testing.T) {
	idx := NewIndex()

	// Add entry with unnormalized path (if applicable)
	path := "dir/subdir/file.txt"
	idx.Add(createTestEntry(path, createTestHash(path)))

	// Retrieve and verify
	retrieved, ok := idx.Get(mustRelativePath(path))
	if !ok {
		t.Fatal("failed to retrieve entry")
	}

	if retrieved.Path.String() != path {
		t.Errorf("path mismatch: expected %s, got %s", path, retrieved.Path)
	}
}

// BenchmarkIndexAdd benchmarks adding entries to the index
func BenchmarkIndexAdd(b *testing.B) {
	idx := NewIndex()
	hash := createTestHash("benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("file/file%d.txt", i%1000)
		entry := createTestEntry(path, hash)
		idx.Add(entry)
	}
}

// BenchmarkIndexSerialize benchmarks index serialization
func BenchmarkIndexSerialize(b *testing.B) {
	idx := NewIndex()

	// Add 100 entries
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("file/file%d.txt", i)
		idx.Add(createTestEntry(path, createTestHash(path)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		_ = idx.Serialize(buf)
	}
}

// BenchmarkIndexDeserialize benchmarks index deserialization
func BenchmarkIndexDeserialize(b *testing.B) {
	idx := NewIndex()

	// Add 100 entries
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("file/file%d.txt", i)
		idx.Add(createTestEntry(path, createTestHash(path)))
	}

	buf := new(bytes.Buffer)
	_ = idx.Serialize(buf)
	data := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deserializedIdx := NewIndex()
		_ = deserializedIdx.Deserialize(bytes.NewReader(data))
	}
}
