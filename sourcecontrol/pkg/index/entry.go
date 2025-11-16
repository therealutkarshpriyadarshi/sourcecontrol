package index

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/common"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// Entry represents a single file entry in the Git index (staging area).
//
// Binary Layout:
//   - Fixed header: 62 bytes (timestamps, metadata, hash, flags)
//   - Variable path: null-terminated file path relative to repository root
//   - Padding: Aligned to 8-byte boundary for efficient disk I/O
//
// The entry format enables Git to quickly determine if a file has changed
// without reading its entire contents by comparing metadata and timestamps.
type Entry struct {
	// CreationTime stores when the file was first created (ctime).
	// In Git's index, this is used with ModificationTime to detect changes.
	CreationTime common.Timestamp

	// ModificationTime stores when the file content was last modified (mtime).
	// This is the primary timestamp used for change detection.
	ModificationTime common.Timestamp

	// DeviceID identifies the device containing the file.
	// Used to detect if a file has moved across file systems.
	DeviceID uint32

	// Inode is the file system inode number.
	// Helps detect file moves and hard link changes.
	Inode uint32

	// Mode stores the file type and Unix permissions.
	// Encodes whether this is a regular file, symlink, or directory,
	// along with the permission bits.
	Mode FileMode

	// UserID is the numeric user ID of the file owner.
	// Tracked for completeness but rarely affects Git operations.
	UserID uint32

	// GroupID is the numeric group ID of the file owner.
	// Tracked for completeness but rarely affects Git operations.
	GroupID uint32

	// SizeInBytes stores the file size in bytes.
	// Used as a quick check to detect content changes.
	SizeInBytes uint32

	// BlobHash is the SHA-1 hash of the file's content.
	// This hash references the blob object stored in Git's object database.
	BlobHash objects.ObjectHash

	// AssumeValid is an optimization flag that tells Git to assume
	// the file hasn't changed, skipping expensive stat checks.
	// Set with: git update-index --assume-unchanged <file>
	AssumeValid bool

	// Stage indicates the merge conflict state:
	//   - 0: Normal entry (no conflict)
	//   - 1: Base/ancestor version (merge base)
	//   - 2: "Ours" version (current branch)
	//   - 3: "Theirs" version (branch being merged)
	Stage uint8

	// Path is the file path relative to the repository root.
	// Normalized to use forward slashes and relative format.
	Path scpath.RelativePath
}

// NewEntry creates a new Entry with default values for the given path.
//
// The path is automatically normalized to ensure consistency across
// different platforms (e.g., converting backslashes to forward slashes).
func NewEntry(path scpath.RelativePath) *Entry {
	return &Entry{
		Path:        path.Normalize(),
		Mode:        FileModeRegular,
		AssumeValid: false,
		Stage:       0,
	}
}

// NewEntryFromFileInfo creates an Entry from file system information.
//
// This constructor populates the entry with metadata from the actual file,
// making it suitable for adding files to the index during staging operations.
//
// Parameters:
//   - path: Relative path to the file from repository root
//   - info: File system information from os.Stat() or os.Lstat()
//   - hash: SHA-1 hash of the file's content (blob object hash)
func NewEntryFromFileInfo(path scpath.RelativePath, info os.FileInfo, hash objects.ObjectHash) (*Entry, error) {
	if !path.IsValid() {
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	e := NewEntry(path)
	e.SizeInBytes = uint32(info.Size())
	e.Mode = FileMode(info.Mode())
	e.BlobHash = hash

	modTime := info.ModTime()
	e.ModificationTime = common.NewTimestampFromTime(modTime)
	e.CreationTime = common.NewTimestampFromTime(modTime)

	// Extract platform-specific metadata (device, inode, uid, gid)
	e.DeviceID, e.Inode, e.UserID, e.GroupID = extractSystemMetadata(info)

	return e, nil
}

// IsModified checks if the entry has been modified compared to file stats.
//
// Parameters:
//   - info: Current file system information from os.Stat()
//
// Returns:
//   - true if the file appears to have been modified
//   - false if the file appears unchanged or AssumeValid is set
func (e *Entry) IsModified(info os.FileInfo) bool {
	if e.AssumeValid {
		return false
	}

	if e.SizeInBytes != uint32(info.Size()) {
		return true
	}

	mtimeSeconds := info.ModTime().Unix()
	return int64(e.ModificationTime.Seconds) != mtimeSeconds
}

// CompareTo compares this entry with another for sorting.
//
// Git maintains the index in sorted order for efficient binary search
// and directory-tree optimizations. The sorting rules are:
//  1. Entries are sorted lexicographically by path
//  2. Directories are treated as having a trailing '/' character
//  3. This ensures proper tree structure: parent directories sort before files
func (e *Entry) CompareTo(other *Entry) int {
	thisKey := e.Path.String()
	otherKey := other.Path.String()

	if e.Mode.IsDirectory() {
		thisKey += "/"
	}
	if other.Mode.IsDirectory() {
		otherKey += "/"
	}

	return strings.Compare(thisKey, otherKey)
}

// Serialize writes the entry in Git's index binary format.
//
// Binary Layout Example:
//
//	[62 bytes: header][variable: path\0][0-7 bytes: padding to 8-byte boundary]
//
// Parameters:
//   - w: Writer to output the serialized data
//
// Returns an error if any write operation fails.
func (e *Entry) Serialize(w io.Writer) error {
	buf := new(bytes.Buffer)

	if err := e.writeFixedFields(buf); err != nil {
		return fmt.Errorf("failed to write fixed fields: %w", err)
	}

	if _, err := buf.WriteString(e.Path.String()); err != nil {
		return fmt.Errorf("failed to write path: %w", err)
	}

	if err := buf.WriteByte(objects.NullByte); err != nil {
		return fmt.Errorf("failed to write null terminator: %w", err)
	}

	pathLen := len(e.Path.String())
	entrySize := FixedHeaderSize + pathLen + 1
	paddedSize := (entrySize + AlignmentBoundary - 1) / AlignmentBoundary * AlignmentBoundary
	padding := paddedSize - entrySize

	for range padding {
		if err := buf.WriteByte(0); err != nil {
			return fmt.Errorf("failed to write padding: %w", err)
		}
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	return nil
}

// writeFixedFields writes the 62-byte fixed header portion of the entry.
//
// All multi-byte integers are written in big-endian (network) byte order
// for cross-platform compatibility.
func (e *Entry) writeFixedFields(w io.Writer) error {
	buf := new(bytes.Buffer)

	if err := e.writeTimestampsAndMetadata(buf); err != nil {
		return fmt.Errorf("failed to write timestamps and metadata %w", err)
	}

	hashBytes, err := e.BlobHash.Raw()
	if err != nil {
		return fmt.Errorf("failed to get hash bytes: %w", err)
	}
	if _, err := buf.Write(hashBytes[:]); err != nil {
		return fmt.Errorf("failed to write hash: %w", err)
	}

	flags := NewEntryFlags(e.AssumeValid, e.Stage, len(e.Path.String()))
	if err := binary.Write(buf, binary.BigEndian, flags); err != nil {
		return fmt.Errorf("failed to write flags: %w", err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write fixed fields: %w", err)
	}

	return nil
}

// writeTimestampsAndMetadata writes the first 40 bytes of the entry header.
//
// This includes all metadata fields except the hash and flags:
//   - Creation time (seconds and nanoseconds)
//   - Modification time (seconds and nanoseconds)
//   - Device ID, inode, mode, user ID, group ID, file size
//
// All values are written as 32-bit unsigned integers in big-endian format.
func (e *Entry) writeTimestampsAndMetadata(w io.Writer) error {
	fields := []uint32{
		e.CreationTime.Seconds,
		e.CreationTime.Nanoseconds,
		e.ModificationTime.Seconds,
		e.ModificationTime.Nanoseconds,
		e.DeviceID,
		e.Inode,
		uint32(e.Mode),
		e.UserID,
		e.GroupID,
		e.SizeInBytes,
	}

	for _, field := range fields {
		if err := binary.Write(w, binary.BigEndian, field); err != nil {
			return fmt.Errorf("failed to write field: %w", err)
		}
	}

	return nil
}

// Deserialize reads an entry from binary data in Git's index format.
//
// This method reconstructs an Entry from its serialized form, performing
// the reverse operation of Serialize(). It reads:
//  1. 62-byte fixed header
//  2. Null-terminated path string
//  3. Padding bytes to 8-byte boundary
//
// Parameters:
//   - r: Reader containing binary index entry data
//
// Returns:
//   - Total bytes read (including padding)
//   - Error if data is invalid or incomplete
func (e *Entry) Deserialize(r io.Reader) (int, error) {
	fixedData := make([]byte, FixedHeaderSize)
	if _, err := io.ReadFull(r, fixedData); err != nil {
		return 0, fmt.Errorf("failed to read fixed header: %w", err)
	}

	if err := e.readFixedFields(fixedData); err != nil {
		return 0, fmt.Errorf("failed to parse fixed fields: %w", err)
	}

	if err := e.readFilePath(r); err != nil {
		return 0, err
	}

	return e.calculatePadding(r)
}

// readFixedFields parses the 62-byte fixed header from raw bytes.
//
// Validates that extended flags are not set, as this implementation
// only supports Git index version 2.
func (e *Entry) readFixedFields(data []byte) error {
	if len(data) < FixedHeaderSize {
		return fmt.Errorf("insufficient data for fixed header: got %d bytes, need %d", len(data), FixedHeaderSize)
	}

	buf := bytes.NewReader(data)

	if err := e.readTimestamp(buf); err != nil {
		return err
	}

	if err := e.readMetadata(buf); err != nil {
		return err
	}

	if err := e.readHash(buf); err != nil {
		return err
	}

	var flags EntryFlags
	if err := binary.Read(buf, binary.BigEndian, &flags); err != nil {
		return err
	}

	if flags.Extended() {
		return fmt.Errorf("extended flags not supported in index version 2")
	}

	e.AssumeValid = flags.AssumeValid()
	e.Stage = flags.Stage()
	return nil
}

// readTimestamp reads and parses creation and modification timestamps.
//
// Each timestamp is stored as two 32-bit unsigned integers:
//   - Seconds since Unix epoch
//   - Nanoseconds (0-999999999)
//
// This provides nanosecond precision for change detection.
func (e *Entry) readTimestamp(r io.Reader) error {
	var createMs, createNanos, modMs, modNanos uint32
	if err := binary.Read(r, binary.BigEndian, &createMs); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &createNanos); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &modMs); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &modNanos); err != nil {
		return err
	}

	e.CreationTime = common.NewTimestamp(createMs, createNanos)
	e.ModificationTime = common.NewTimestamp(modMs, modNanos)
	return nil
}

// readMetadata reads file system metadata fields from the header.
//
// All fields are 32-bit unsigned integers in big-endian format.
func (e *Entry) readMetadata(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &e.DeviceID); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &e.Inode); err != nil {
		return err
	}

	var mode uint32
	if err := binary.Read(r, binary.BigEndian, &mode); err != nil {
		return err
	}
	e.Mode = FileMode(mode)

	if err := binary.Read(r, binary.BigEndian, &e.UserID); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &e.GroupID); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &e.SizeInBytes); err != nil {
		return err
	}

	return nil
}

// readHash reads and parses the 20-byte SHA-1 hash from the header.
//
// The hash is stored as raw bytes and converted to a hex string
// for internal representation. It represents the object ID of
// the blob object containing the file's contents.
func (e *Entry) readHash(r io.Reader) error {
	hashBytes := make([]byte, SHALength)
	if _, err := io.ReadFull(r, hashBytes); err != nil {
		return fmt.Errorf("failed to read hash: %w", err)
	}
	hashStr := hex.EncodeToString(hashBytes)
	hash, err := objects.ParseObjectHash(hashStr)
	if err != nil {
		return fmt.Errorf("invalid hash: %w", err)
	}
	e.BlobHash = hash
	return nil
}

// readFilePath reads the null-terminated path string from the entry.
//
// The path is read byte-by-byte until a null terminator (0x00) is found.
// This allows for variable-length paths without a fixed buffer size.
//
// After reading, the path is:
//  1. Validated to ensure it's a proper relative path
//  2. Normalized to use forward slashes and relative format
//
// Returns an error if the path is invalid or cannot be read.
func (e *Entry) readFilePath(r io.Reader) error {
	pathBytes := make([]byte, 0, 256) // Start with reasonable capacity
	for {
		b := make([]byte, 1)
		if _, err := r.Read(b); err != nil {
			return fmt.Errorf("failed to read path: %w", err)
		}
		if b[0] == 0 {
			break
		}
		pathBytes = append(pathBytes, b[0])
	}

	pathStr := string(pathBytes)
	relativePath, err := scpath.NewRelativePath(pathStr)
	if err != nil {
		return fmt.Errorf("invalid path in index: %w", err)
	}

	e.Path = relativePath
	return nil
}

// calculatePadding reads padding bytes to reach 8-byte alignment.
//
// After the null-terminated path, Git pads the entry with zeros
// to ensure the next entry starts at an 8-byte boundary. This
// alignment is important for:
//   - Memory-mapped file performance
//   - Consistent entry offsets for binary search
//   - Cache line efficiency
//
// Returns the total size of the entry including padding.
func (e *Entry) calculatePadding(r io.Reader) (int, error) {
	pathLen := len(e.Path.String())
	bytesRead := FixedHeaderSize + pathLen + 1 // +1 for null terminator

	paddedSize := (bytesRead + AlignmentBoundary - 1) / AlignmentBoundary * AlignmentBoundary
	padding := paddedSize - bytesRead

	if padding > 0 {
		paddingBuf := make([]byte, padding)
		if _, err := io.ReadFull(r, paddingBuf); err != nil {
			return 0, fmt.Errorf("failed to read padding: %w", err)
		}
	}

	return paddedSize, nil
}

// String returns a human-readable representation of the entry.
func (e *Entry) String() string {
	return fmt.Sprintf("Entry{path: %s, mode: %s, hash: %s, size: %d}",
		e.Path, e.Mode, e.BlobHash.Short(), e.SizeInBytes)
}
