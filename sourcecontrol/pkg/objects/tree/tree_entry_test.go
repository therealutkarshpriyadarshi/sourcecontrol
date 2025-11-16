package tree

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

func TestNewTreeEntry(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		ename   string
		sha     string
		wantErr bool
	}{
		{
			name:    "valid regular file entry",
			mode:    "100644",
			ename:   "README.md",
			sha:     "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
			wantErr: false,
		},
		{
			name:    "valid directory entry",
			mode:    "040000",
			ename:   "src",
			sha:     "1234567890abcdef1234567890abcdef12345678",
			wantErr: false,
		},
		{
			name:    "valid executable file",
			mode:    "100755",
			ename:   "build.sh",
			sha:     "abcdef1234567890abcdef1234567890abcdef12",
			wantErr: false,
		},
		{
			name:    "empty name",
			mode:    "100644",
			ename:   "",
			sha:     "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
			wantErr: true,
		},
		{
			name:    "invalid name with slash",
			mode:    "100644",
			ename:   "path/to/file",
			sha:     "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
			wantErr: true,
		},
		{
			name:    "invalid SHA length",
			mode:    "100644",
			ename:   "file.txt",
			sha:     "short",
			wantErr: true,
		},
		{
			name:    "invalid SHA characters",
			mode:    "100644",
			ename:   "file.txt",
			sha:     "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := NewTreeEntryFromStrings(tt.mode, tt.ename, tt.sha)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTreeEntryFromStrings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				expectedMode, _ := objects.FromOctalString(tt.mode)
				if entry.Mode() != expectedMode {
					t.Errorf("Mode() = %v, want %v", entry.Mode(), expectedMode)
				}
				if entry.Name().String() != tt.ename {
					t.Errorf("Name() = %v, want %v", entry.Name(), tt.ename)
				}
				expectedSha, _ := objects.ParseObjectHash(tt.sha)
				if entry.SHA() != expectedSha {
					t.Errorf("SHA() = %v, want %v", entry.SHA(), expectedSha)
				}
			}
		})
	}
}

func TestTreeEntryTypes(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		isDir        bool
		isFile       bool
		isExecutable bool
		isSymlink    bool
		isSubmodule  bool
	}{
		{
			name:         "directory",
			mode:         "040000",
			isDir:        true,
			isFile:       false,
			isExecutable: false,
			isSymlink:    false,
			isSubmodule:  false,
		},
		{
			name:         "regular file",
			mode:         "100644",
			isDir:        false,
			isFile:       true,
			isExecutable: false,
			isSymlink:    false,
			isSubmodule:  false,
		},
		{
			name:         "executable file",
			mode:         "100755",
			isDir:        false,
			isFile:       true,
			isExecutable: true,
			isSymlink:    false,
			isSubmodule:  false,
		},
		{
			name:         "symbolic link",
			mode:         "120000",
			isDir:        false,
			isFile:       false,
			isExecutable: false,
			isSymlink:    true,
			isSubmodule:  false,
		},
		{
			name:         "submodule",
			mode:         "160000",
			isDir:        false,
			isFile:       false,
			isExecutable: false,
			isSymlink:    false,
			isSubmodule:  true,
		},
	}

	sha := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := NewTreeEntryFromStrings(tt.mode, "test", sha)
			if err != nil {
				t.Fatalf("NewTreeEntryFromStrings() error = %v", err)
			}

			if entry.IsDirectory() != tt.isDir {
				t.Errorf("IsDirectory() = %v, want %v", entry.IsDirectory(), tt.isDir)
			}
			if entry.IsFile() != tt.isFile {
				t.Errorf("IsFile() = %v, want %v", entry.IsFile(), tt.isFile)
			}
			if entry.IsExecutable() != tt.isExecutable {
				t.Errorf("IsExecutable() = %v, want %v", entry.IsExecutable(), tt.isExecutable)
			}
			if entry.IsSymbolicLink() != tt.isSymlink {
				t.Errorf("IsSymbolicLink() = %v, want %v", entry.IsSymbolicLink(), tt.isSymlink)
			}
			if entry.IsSubmodule() != tt.isSubmodule {
				t.Errorf("IsSubmodule() = %v, want %v", entry.IsSubmodule(), tt.isSubmodule)
			}
		})
	}
}

func TestTreeEntrySerializeDeserialize(t *testing.T) {
	tests := []struct {
		name  string
		mode  string
		ename string
		sha   string
	}{
		{
			name:  "regular file",
			mode:  "100644",
			ename: "README.md",
			sha:   "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
		},
		{
			name:  "directory",
			mode:  "040000",
			ename: "src",
			sha:   "1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:  "executable",
			mode:  "100755",
			ename: "build.sh",
			sha:   "abcdef1234567890abcdef1234567890abcdef12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create entry
			entry, err := NewTreeEntryFromStrings(tt.mode, tt.ename, tt.sha)
			if err != nil {
				t.Fatalf("NewTreeEntryFromStrings() error = %v", err)
			}

			// Serialize
			var buf bytes.Buffer
			err = entry.Serialize(&buf)
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}
			serialized := buf.Bytes()

			// Deserialize
			deserialized := &TreeEntry{}
			err = deserialized.Deserialize(bytes.NewReader(serialized))
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}

			// Verify
			if deserialized.Mode() != entry.Mode() {
				t.Errorf("Mode() = %v, want %v", deserialized.Mode(), entry.Mode())
			}
			if deserialized.Name() != entry.Name() {
				t.Errorf("Name() = %v, want %v", deserialized.Name(), entry.Name())
			}
			if deserialized.SHA() != entry.SHA() {
				t.Errorf("SHA() = %v, want %v", deserialized.SHA(), entry.SHA())
			}
		})
	}
}

func TestTreeEntryCompareTo(t *testing.T) {
	sha := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"

	fileA, _ := NewTreeEntryFromStrings("100644", "a.txt", sha)
	fileB, _ := NewTreeEntryFromStrings("100644", "b.txt", sha)
	dirA, _ := NewTreeEntryFromStrings("040000", "a", sha)
	dirB, _ := NewTreeEntryFromStrings("040000", "b", sha)

	tests := []struct {
		name     string
		entry1   *TreeEntry
		entry2   *TreeEntry
		expected int
	}{
		{
			name:     "file a < file b",
			entry1:   fileA,
			entry2:   fileB,
			expected: -1,
		},
		{
			name:     "file b > file a",
			entry1:   fileB,
			entry2:   fileA,
			expected: 1,
		},
		{
			name:     "dir a < dir b",
			entry1:   dirA,
			entry2:   dirB,
			expected: -1,
		},
		{
			name:     "same name, dir before file",
			entry1:   dirA,
			entry2:   fileA,
			expected: -1,
		},
		{
			name:     "same name, file after dir",
			entry1:   fileA,
			entry2:   dirA,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry1.CompareTo(tt.entry2)
			if (result < 0 && tt.expected >= 0) ||
				(result > 0 && tt.expected <= 0) ||
				(result == 0 && tt.expected != 0) {
				t.Errorf("CompareTo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTreeEntrySerializeFormat(t *testing.T) {
	entry, err := NewTreeEntryFromStrings("100644", "test.txt", "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0")
	if err != nil {
		t.Fatalf("NewTreeEntryFromStrings() error = %v", err)
	}

	var buf bytes.Buffer
	err = entry.Serialize(&buf)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	serialized := buf.Bytes()

	// Expected format: "100644 test.txt\0[20 bytes]"
	expectedPrefix := "100644 test.txt\x00"
	if string(serialized[:len(expectedPrefix)]) != expectedPrefix {
		t.Errorf("Serialized prefix = %q, want %q", string(serialized[:len(expectedPrefix)]), expectedPrefix)
	}

	// Verify SHA bytes
	shaBytes := serialized[len(expectedPrefix):]
	if len(shaBytes) != objects.RawHashLength {
		t.Errorf("SHA bytes length = %v, want %v", len(shaBytes), objects.RawHashLength)
	}

	expectedShaBytes, _ := hex.DecodeString("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0")
	if hex.EncodeToString(shaBytes) != hex.EncodeToString(expectedShaBytes) {
		t.Errorf("SHA bytes = %x, want %x", shaBytes, expectedShaBytes)
	}
}
