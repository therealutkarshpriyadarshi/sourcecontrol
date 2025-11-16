package tree

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// TreeEntry represents a single entry in a Git tree object.
//
// Each entry contains:
// - mode: File permissions and type (FileMode)
// - name: Filename or directory name (RelativePath)
// - sha: SHA-1 hash of the referenced object (ObjectHash)
//
// Entry types by mode:
// - 040000: Directory (tree object)
// - 100644: Regular file (blob object)
// - 100755: Executable file (blob object)
// - 120000: Symbolic link (blob object)
// - 160000: Git submodule (commit object)
//
// Serialized format in tree object:
// [mode] [space] [filename] [null byte] [20-byte SHA-1 binary]
//
// Example serialized entry for "hello.txt" file:
// "100644 hello.txt\0[20 bytes of SHA-1]"
type TreeEntry struct {
	mode objects.FileMode
	name scpath.RelativePath
	sha  objects.ObjectHash
}

// NewTreeEntry creates a new TreeEntry with validation
func NewTreeEntry(mode objects.FileMode, name scpath.RelativePath, sha objects.ObjectHash) (*TreeEntry, error) {
	if !name.IsValid() {
		return nil, fmt.Errorf("invalid path: %s", name)
	}

	// Tree entries must be single path components (no slashes)
	nameStr := name.String()
	if len(nameStr) == 0 || nameStr == "." {
		return nil, fmt.Errorf("empty name not allowed")
	}
	if bytes.Contains([]byte(nameStr), []byte{'/'}) || bytes.Contains([]byte(nameStr), []byte{'\\'}) {
		return nil, fmt.Errorf("tree entry name cannot contain path separators: %s", nameStr)
	}

	if err := sha.Validate(); err != nil {
		return nil, fmt.Errorf("invalid SHA: %w", err)
	}

	entry := &TreeEntry{
		mode: mode,
		name: name.Normalize(),
		sha:  sha,
	}

	return entry, nil
}

// NewTreeEntryFromStrings creates a new TreeEntry from string values (for backward compatibility)
func NewTreeEntryFromStrings(modeStr, name, shaStr string) (*TreeEntry, error) {
	mode, err := objects.FromOctalString(modeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid mode: %w", err)
	}

	path, err := scpath.NewRelativePath(name)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	sha, err := objects.ParseObjectHash(shaStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SHA: %w", err)
	}

	return NewTreeEntry(mode, path, sha)
}

// Mode returns the entry mode
func (e *TreeEntry) Mode() objects.FileMode {
	return e.mode
}

// Name returns the entry name
func (e *TreeEntry) Name() scpath.RelativePath {
	return e.name
}

// Path returns the entry path
func (e *TreeEntry) Path() scpath.RelativePath {
	return e.name
}

// SHA returns the entry SHA-1 hash
func (e *TreeEntry) SHA() objects.ObjectHash {
	return e.sha
}

// IsDirectory returns true if this entry is a directory
func (e *TreeEntry) IsDirectory() bool {
	return e.mode == objects.FileModeDirectory
}

// IsFile returns true if this entry is a regular or executable file
func (e *TreeEntry) IsFile() bool {
	return e.mode == objects.FileModeRegular || e.mode == objects.FileModeExecutable
}

// IsExecutable returns true if this entry is an executable file
func (e *TreeEntry) IsExecutable() bool {
	return e.mode == objects.FileModeExecutable
}

// IsSymbolicLink returns true if this entry is a symbolic link
func (e *TreeEntry) IsSymbolicLink() bool {
	return e.mode == objects.FileModeSymlink
}

// IsSubmodule returns true if this entry is a submodule
func (e *TreeEntry) IsSubmodule() bool {
	return e.mode == objects.FileModeGitlink
}

// Serialize writes the serialized entry to the provided writer
// Format: [mode] [space] [filename] [null byte] [20-byte SHA-1 binary]
func (e *TreeEntry) Serialize(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s %s%c", e.mode.ToOctalString(), e.name.String(), objects.NullByte); err != nil {
		return fmt.Errorf("failed to write entry header: %w", err)
	}

	shaBytes, err := e.sha.Bytes()
	if err != nil {
		return fmt.Errorf("failed to get SHA bytes: %w", err)
	}

	if _, err := w.Write(shaBytes); err != nil {
		return fmt.Errorf("failed to write SHA bytes: %w", err)
	}

	return nil
}

// CompareTo compares this entry with another entry.
// The comparison is based on the entry's name, with directories sorted before files.
// Returns a negative value if this entry is less than the other, zero if they are equal,
// or a positive value if this entry is greater than the other.
func (e *TreeEntry) CompareTo(other *TreeEntry) int {
	if e.name == other.name {
		if e.IsDirectory() && !other.IsDirectory() {
			return -1
		}
		if !e.IsDirectory() && other.IsDirectory() {
			return 1
		}
		return 0
	}
	if e.name < other.name {
		return -1
	}
	return 1
}

// Deserialize reads and creates a TreeEntry from an io.Reader
// Returns the created entry or an error if parsing fails
func (e *TreeEntry) Deserialize(r io.Reader) error {
	mode, err := readUntil(r, objects.SpaceByte)
	if err != nil {
		return fmt.Errorf("invalid tree entry: failed to read mode: %w", err)
	}

	name, err := readUntil(r, objects.NullByte)
	if err != nil {
		return fmt.Errorf("invalid tree entry: failed to read name: %w", err)
	}

	shaBytes := make([]byte, objects.RawHashLength)
	if _, err := io.ReadFull(r, shaBytes); err != nil {
		return fmt.Errorf("invalid tree entry: incomplete SHA: %w", err)
	}

	entry, err := NewTreeEntryFromStrings(string(mode), string(name), hex.EncodeToString(shaBytes))
	if err != nil {
		return err
	}

	*e = *entry
	return nil
}

// readUntil reads from reader until delimiter is found
func readUntil(r io.Reader, delim byte) ([]byte, error) {
	var buf bytes.Buffer
	b := make([]byte, 1)

	for {
		if _, err := r.Read(b); err != nil {
			return nil, err
		}
		if b[0] == delim {
			break
		}
		buf.WriteByte(b[0])
	}

	return buf.Bytes(), nil
}
