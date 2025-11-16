package index

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/common"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// TestNewEntry tests the NewEntry constructor
func TestNewEntry(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected struct {
			mode        FileMode
			assumeValid bool
			stage       uint8
		}
	}{
		{
			name: "simple file path",
			path: "file.txt",
			expected: struct {
				mode        FileMode
				assumeValid bool
				stage       uint8
			}{
				mode:        FileModeRegular,
				assumeValid: false,
				stage:       0,
			},
		},
		{
			name: "nested file path",
			path: "src/main/file.go",
			expected: struct {
				mode        FileMode
				assumeValid bool
				stage       uint8
			}{
				mode:        FileModeRegular,
				assumeValid: false,
				stage:       0,
			},
		},
		{
			name: "path with spaces",
			path: "my file.txt",
			expected: struct {
				mode        FileMode
				assumeValid bool
				stage       uint8
			}{
				mode:        FileModeRegular,
				assumeValid: false,
				stage:       0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := scpath.NewRelativePath(tt.path)
			if err != nil {
				t.Fatalf("failed to create path: %v", err)
			}

			entry := NewEntry(path)

			if entry.Mode != tt.expected.mode {
				t.Errorf("Mode = %v, want %v", entry.Mode, tt.expected.mode)
			}
			if entry.AssumeValid != tt.expected.assumeValid {
				t.Errorf("AssumeValid = %v, want %v", entry.AssumeValid, tt.expected.assumeValid)
			}
			if entry.Stage != tt.expected.stage {
				t.Errorf("Stage = %v, want %v", entry.Stage, tt.expected.stage)
			}
			if entry.Path.String() != path.Normalize().String() {
				t.Errorf("Path = %v, want %v", entry.Path, path.Normalize())
			}
		})
	}
}

// TestNewEntryFromFileInfo tests creating an entry from file info
func TestNewEntryFromFileInfo(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test-entry-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write some content
	content := []byte("test content")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	// Get file info
	info, err := tmpFile.Stat()
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	// Create a test hash
	hash, err := objects.ParseObjectHash("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")
	if err != nil {
		t.Fatalf("failed to create hash: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "valid path",
			path:      "test.txt",
			wantError: false,
		},
		{
			name:      "nested path",
			path:      "src/test/file.txt",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := scpath.NewRelativePath(tt.path)
			if err != nil {
				t.Fatalf("failed to create path: %v", err)
			}

			entry, err := NewEntryFromFileInfo(path, info, hash)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if entry.SizeInBytes != uint32(info.Size()) {
				t.Errorf("SizeInBytes = %v, want %v", entry.SizeInBytes, info.Size())
			}

			if entry.BlobHash.String() != hash.String() {
				t.Errorf("BlobHash = %v, want %v", entry.BlobHash, hash)
			}

			if entry.ModificationTime.Seconds != uint32(info.ModTime().Unix()) {
				t.Errorf("ModificationTime.Seconds = %v, want %v", entry.ModificationTime.Seconds, info.ModTime().Unix())
			}

			if entry.CreationTime.Seconds != uint32(info.ModTime().Unix()) {
				t.Errorf("CreationTime.Seconds = %v, want %v", entry.CreationTime.Seconds, info.ModTime().Unix())
			}
		})
	}
}

// TestEntrySerializeDeserialize tests serialization and deserialization
func TestEntrySerializeDeserialize(t *testing.T) {
	tests := []struct {
		name  string
		entry *Entry
	}{
		{
			name: "simple entry",
			entry: &Entry{
				CreationTime:     common.NewTimestamp(1234567890, 123456789),
				ModificationTime: common.NewTimestamp(1234567900, 987654321),
				DeviceID:         1,
				Inode:            2,
				Mode:             FileModeRegular,
				UserID:           1000,
				GroupID:          1000,
				SizeInBytes:      100,
				BlobHash:         mustParseHash(t, "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"),
				AssumeValid:      false,
				Stage:            0,
				Path:             mustCreatePath(t, "test.txt"),
			},
		},
		{
			name: "entry with long path",
			entry: &Entry{
				CreationTime:     common.NewTimestamp(1234567890, 123456789),
				ModificationTime: common.NewTimestamp(1234567900, 987654321),
				DeviceID:         1,
				Inode:            2,
				Mode:             FileModeRegular,
				UserID:           1000,
				GroupID:          1000,
				SizeInBytes:      200,
				BlobHash:         mustParseHash(t, "b94a8fe5ccb19ba61c4c0873d391e987982fbbd4"),
				AssumeValid:      false,
				Stage:            0,
				Path:             mustCreatePath(t, "src/main/java/com/example/MyClass.java"),
			},
		},
		{
			name: "entry with assume valid flag",
			entry: &Entry{
				CreationTime:     common.NewTimestamp(1234567890, 123456789),
				ModificationTime: common.NewTimestamp(1234567900, 987654321),
				DeviceID:         1,
				Inode:            2,
				Mode:             FileModeRegular,
				UserID:           1000,
				GroupID:          1000,
				SizeInBytes:      300,
				BlobHash:         mustParseHash(t, "c94a8fe5ccb19ba61c4c0873d391e987982fbbd5"),
				AssumeValid:      true,
				Stage:            0,
				Path:             mustCreatePath(t, "README.md"),
			},
		},
		{
			name: "entry with stage number",
			entry: &Entry{
				CreationTime:     common.NewTimestamp(1234567890, 123456789),
				ModificationTime: common.NewTimestamp(1234567900, 987654321),
				DeviceID:         1,
				Inode:            2,
				Mode:             FileModeRegular,
				UserID:           1000,
				GroupID:          1000,
				SizeInBytes:      400,
				BlobHash:         mustParseHash(t, "d94a8fe5ccb19ba61c4c0873d391e987982fbbd6"),
				AssumeValid:      false,
				Stage:            2,
				Path:             mustCreatePath(t, "conflict.txt"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			buf := new(bytes.Buffer)
			if err := tt.entry.Serialize(buf); err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			// Deserialize
			deserializedEntry := &Entry{}
			reader := bytes.NewReader(buf.Bytes())
			bytesRead, err := deserializedEntry.Deserialize(reader)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}

			if bytesRead == 0 {
				t.Error("Deserialize() bytesRead = 0")
			}

			// Compare fields
			if deserializedEntry.CreationTime.Seconds != tt.entry.CreationTime.Seconds {
				t.Errorf("CreationTime.Seconds = %v, want %v", deserializedEntry.CreationTime.Seconds, tt.entry.CreationTime.Seconds)
			}
			if deserializedEntry.CreationTime.Nanoseconds != tt.entry.CreationTime.Nanoseconds {
				t.Errorf("CreationTime.Nanoseconds = %v, want %v", deserializedEntry.CreationTime.Nanoseconds, tt.entry.CreationTime.Nanoseconds)
			}
			if deserializedEntry.ModificationTime.Seconds != tt.entry.ModificationTime.Seconds {
				t.Errorf("ModificationTime.Seconds = %v, want %v", deserializedEntry.ModificationTime.Seconds, tt.entry.ModificationTime.Seconds)
			}
			if deserializedEntry.ModificationTime.Nanoseconds != tt.entry.ModificationTime.Nanoseconds {
				t.Errorf("ModificationTime.Nanoseconds = %v, want %v", deserializedEntry.ModificationTime.Nanoseconds, tt.entry.ModificationTime.Nanoseconds)
			}
			if deserializedEntry.DeviceID != tt.entry.DeviceID {
				t.Errorf("DeviceID = %v, want %v", deserializedEntry.DeviceID, tt.entry.DeviceID)
			}
			if deserializedEntry.Inode != tt.entry.Inode {
				t.Errorf("Inode = %v, want %v", deserializedEntry.Inode, tt.entry.Inode)
			}
			if deserializedEntry.Mode != tt.entry.Mode {
				t.Errorf("Mode = %v, want %v", deserializedEntry.Mode, tt.entry.Mode)
			}
			if deserializedEntry.UserID != tt.entry.UserID {
				t.Errorf("UserID = %v, want %v", deserializedEntry.UserID, tt.entry.UserID)
			}
			if deserializedEntry.GroupID != tt.entry.GroupID {
				t.Errorf("GroupID = %v, want %v", deserializedEntry.GroupID, tt.entry.GroupID)
			}
			if deserializedEntry.SizeInBytes != tt.entry.SizeInBytes {
				t.Errorf("SizeInBytes = %v, want %v", deserializedEntry.SizeInBytes, tt.entry.SizeInBytes)
			}
			if deserializedEntry.BlobHash.String() != tt.entry.BlobHash.String() {
				t.Errorf("BlobHash = %v, want %v", deserializedEntry.BlobHash, tt.entry.BlobHash)
			}
			if deserializedEntry.AssumeValid != tt.entry.AssumeValid {
				t.Errorf("AssumeValid = %v, want %v", deserializedEntry.AssumeValid, tt.entry.AssumeValid)
			}
			if deserializedEntry.Stage != tt.entry.Stage {
				t.Errorf("Stage = %v, want %v", deserializedEntry.Stage, tt.entry.Stage)
			}
			if deserializedEntry.Path.String() != tt.entry.Path.String() {
				t.Errorf("Path = %v, want %v", deserializedEntry.Path, tt.entry.Path)
			}
		})
	}
}

// TestEntryIsModified tests the IsModified method
func TestEntryIsModified(t *testing.T) {
	now := time.Now()
	later := now.Add(1 * time.Hour)

	tests := []struct {
		name         string
		entry        *Entry
		fileSize     int64
		fileModTime  time.Time
		wantModified bool
	}{
		{
			name: "not modified - same size and time",
			entry: &Entry{
				SizeInBytes:      100,
				ModificationTime: common.NewTimestampFromTime(now),
				AssumeValid:      false,
			},
			fileSize:     100,
			fileModTime:  now,
			wantModified: false,
		},
		{
			name: "modified - different size",
			entry: &Entry{
				SizeInBytes:      100,
				ModificationTime: common.NewTimestampFromTime(now),
				AssumeValid:      false,
			},
			fileSize:     200,
			fileModTime:  now,
			wantModified: true,
		},
		{
			name: "modified - different time",
			entry: &Entry{
				SizeInBytes:      100,
				ModificationTime: common.NewTimestampFromTime(now),
				AssumeValid:      false,
			},
			fileSize:     100,
			fileModTime:  later,
			wantModified: true,
		},
		{
			name: "assume valid - not checked",
			entry: &Entry{
				SizeInBytes:      100,
				ModificationTime: common.NewTimestampFromTime(now),
				AssumeValid:      true,
			},
			fileSize:     200,
			fileModTime:  later,
			wantModified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock file info
			info := &mockFileInfo{
				size:    tt.fileSize,
				modTime: tt.fileModTime,
			}

			got := tt.entry.IsModified(info)
			if got != tt.wantModified {
				t.Errorf("IsModified() = %v, want %v", got, tt.wantModified)
			}
		})
	}
}

// TestEntryCompareTo tests the CompareTo method
func TestEntryCompareTo(t *testing.T) {
	tests := []struct {
		name     string
		entry1   *Entry
		entry2   *Entry
		expected int // -1 for less than, 0 for equal, 1 for greater than
	}{
		{
			name: "same path",
			entry1: &Entry{
				Path: mustCreatePath(t, "file.txt"),
				Mode: FileModeRegular,
			},
			entry2: &Entry{
				Path: mustCreatePath(t, "file.txt"),
				Mode: FileModeRegular,
			},
			expected: 0,
		},
		{
			name: "first comes before second",
			entry1: &Entry{
				Path: mustCreatePath(t, "a.txt"),
				Mode: FileModeRegular,
			},
			entry2: &Entry{
				Path: mustCreatePath(t, "b.txt"),
				Mode: FileModeRegular,
			},
			expected: -1,
		},
		{
			name: "first comes after second",
			entry1: &Entry{
				Path: mustCreatePath(t, "z.txt"),
				Mode: FileModeRegular,
			},
			entry2: &Entry{
				Path: mustCreatePath(t, "a.txt"),
				Mode: FileModeRegular,
			},
			expected: 1,
		},
		{
			name: "file comes before another alphabetically",
			entry1: &Entry{
				Path: mustCreatePath(t, "dir/file.txt"),
				Mode: FileModeRegular,
			},
			entry2: &Entry{
				Path: mustCreatePath(t, "dir/other.txt"),
				Mode: FileModeRegular,
			},
			expected: -1,
		},
		{
			name: "nested paths",
			entry1: &Entry{
				Path: mustCreatePath(t, "src/a.txt"),
				Mode: FileModeRegular,
			},
			entry2: &Entry{
				Path: mustCreatePath(t, "src/b.txt"),
				Mode: FileModeRegular,
			},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry1.CompareTo(tt.entry2)

			// Normalize result to -1, 0, or 1
			var normalizedResult int
			if result < 0 {
				normalizedResult = -1
			} else if result > 0 {
				normalizedResult = 1
			} else {
				normalizedResult = 0
			}

			if normalizedResult != tt.expected {
				t.Errorf("CompareTo() = %v, want %v", normalizedResult, tt.expected)
			}
		})
	}
}

// TestEntryString tests the String method
func TestEntryString(t *testing.T) {
	entry := &Entry{
		Path:        mustCreatePath(t, "test.txt"),
		Mode:        FileModeRegular,
		BlobHash:    mustParseHash(t, "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"),
		SizeInBytes: 100,
	}

	str := entry.String()
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Check that it contains key information
	if !containsString(str, "test.txt") {
		t.Error("String() should contain path")
	}
	if !containsString(str, "100") {
		t.Error("String() should contain size")
	}
}

// TestEntryDeserializeErrors tests error cases in deserialization
func TestEntryDeserializeErrors(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantError bool
	}{
		{
			name:      "insufficient data",
			data:      []byte{1, 2, 3},
			wantError: true,
		},
		{
			name:      "empty data",
			data:      []byte{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &Entry{}
			reader := bytes.NewReader(tt.data)
			_, err := entry.Deserialize(reader)

			if tt.wantError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestEntrySerializationPadding tests that serialization includes proper padding
func TestEntrySerializationPadding(t *testing.T) {
	tests := []struct {
		name    string
		pathLen int
		path    string
	}{
		{
			name:    "short path",
			pathLen: 8,
			path:    "test.txt",
		},
		{
			name:    "medium path",
			pathLen: 20,
			path:    "src/main/test/file.txt",
		},
		{
			name:    "long path",
			pathLen: 50,
			path:    "src/main/java/com/example/package/MyClass.java",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &Entry{
				CreationTime:     common.NewTimestamp(1234567890, 123456789),
				ModificationTime: common.NewTimestamp(1234567900, 987654321),
				DeviceID:         1,
				Inode:            2,
				Mode:             FileModeRegular,
				UserID:           1000,
				GroupID:          1000,
				SizeInBytes:      100,
				BlobHash:         mustParseHash(t, "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"),
				AssumeValid:      false,
				Stage:            0,
				Path:             mustCreatePath(t, tt.path),
			}

			buf := new(bytes.Buffer)
			if err := entry.Serialize(buf); err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			// Check that the size is aligned to 8-byte boundary
			size := buf.Len()
			if size%AlignmentBoundary != 0 {
				t.Errorf("Serialized size %d is not aligned to %d-byte boundary", size, AlignmentBoundary)
			}
		})
	}
}

// Helper functions

func mustCreatePath(t *testing.T, path string) scpath.RelativePath {
	t.Helper()
	p, err := scpath.NewRelativePath(path)
	if err != nil {
		t.Fatalf("failed to create path %q: %v", path, err)
	}
	return p
}

func mustParseHash(t *testing.T, hash string) objects.ObjectHash {
	t.Helper()
	h, err := objects.ParseObjectHash(hash)
	if err != nil {
		t.Fatalf("failed to parse hash %q: %v", hash, err)
	}
	return h
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }
