package objects

import (
	"fmt"
	"os"
)

// FileMode represents Git file mode (type + permissions).
// Git stores file type in the upper 4 bits and permissions in the lower bits.
// This type provides a type-safe way to work with file modes across trees, index, and working directory.
type FileMode uint32

// File mode constants that Git uses
const (
	FileModeTypeMask FileMode = 0xF000 // Upper 4 bits (bits 12-15)
	FileModePermMask FileMode = 0x01FF // Lower 9 bits (permissions)
	FileModeExecMask FileMode = 0x0049 // Execute bits (owner/group/other)

	// File type values (after shifting right by 12 bits)
	FileModeTypeRegular FileMode = 0x8000 // 0b1000 << 12 - Regular file
	FileModeTypeSymlink FileMode = 0xA000 // 0b1010 << 12 - Symbolic link
	FileModeTypeGitlink FileMode = 0xE000 // 0b1110 << 12 - Gitlink (submodule)
	FileModeTypeDir     FileMode = 0x0000 // 0b0000 << 12 - Directory (rare in index)

	// Common mode values
	FileModeRegular    FileMode = 0o100644 // Regular file, rw-r--r--
	FileModeExecutable FileMode = 0o100755 // Executable file, rwxr-xr-x
	FileModeSymlink    FileMode = 0o120000 // Symbolic link
	FileModeGitlink    FileMode = 0o160000 // Gitlink (submodule)
	FileModeDirectory  FileMode = 0o040000 // Directory (used in trees)
)

// Type returns the file type portion of the mode.
func (m FileMode) Type() FileMode {
	return m & FileModeTypeMask
}

// Permissions returns the permission bits.
func (m FileMode) Permissions() FileMode {
	return m & FileModePermMask
}

// IsRegular returns true if this is a regular file.
func (m FileMode) IsRegular() bool {
	return m.Type() == FileModeTypeRegular
}

// IsSymlink returns true if this is a symbolic link.
func (m FileMode) IsSymlink() bool {
	return m.Type() == FileModeTypeSymlink
}

// IsGitlink returns true if this is a gitlink (submodule).
func (m FileMode) IsGitlink() bool {
	return m.Type() == FileModeTypeGitlink
}

// IsDirectory returns true if this is a directory.
func (m FileMode) IsDirectory() bool {
	return m.Type() == FileModeTypeDir
}

// IsExecutable returns true if the file has execute permissions.
func (m FileMode) IsExecutable() bool {
	return (m & FileModeExecMask) != 0
}

// IsFile returns true if this is a regular or executable file.
func (m FileMode) IsFile() bool {
	return m.IsRegular()
}

// String returns a human-readable representation of the file mode.
func (m FileMode) String() string {
	switch m.Type() {
	case FileModeTypeRegular:
		return fmt.Sprintf("regular(%o)", m.Permissions())
	case FileModeTypeSymlink:
		return "symlink"
	case FileModeTypeGitlink:
		return "gitlink"
	case FileModeTypeDir:
		return "directory"
	default:
		return fmt.Sprintf("unknown(%o)", m)
	}
}

// ToOctalString returns the mode as an octal string (e.g., "100644", "040000").
// This is the format used in Git tree objects.
func (m FileMode) ToOctalString() string {
	return fmt.Sprintf("%06o", m)
}

// FromOctalString parses a mode from an octal string (e.g., "100644", "040000").
// This is used when reading Git tree objects.
func FromOctalString(s string) (FileMode, error) {
	var mode uint32
	_, err := fmt.Sscanf(s, "%o", &mode)
	if err != nil {
		return 0, fmt.Errorf("invalid mode string %q: %w", s, err)
	}
	return FileMode(mode), nil
}

// FromOSFileMode converts os.FileMode to Git FileMode.
// This is used when staging files from the working directory.
func FromOSFileMode(mode os.FileMode) FileMode {
	if mode&os.ModeSymlink != 0 {
		return FileModeSymlink
	}
	if mode&0o111 != 0 { // Any execute bit set
		return FileModeExecutable
	}
	return FileModeRegular
}

// ToOSFileMode converts Git FileMode to os.FileMode.
// This is used when checking out files to the working directory.
func (m FileMode) ToOSFileMode() os.FileMode {
	switch m.Type() {
	case FileModeTypeSymlink:
		return os.ModeSymlink | 0o644
	case FileModeTypeRegular:
		if m.IsExecutable() {
			return 0o755
		}
		return 0o644
	default:
		return os.FileMode(m)
	}
}
