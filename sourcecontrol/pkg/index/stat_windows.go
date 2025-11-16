//go:build windows

package index

import (
	"os"
	"syscall"
)

// extractSystemMetadata extracts platform-specific file system metadata.
// On Windows, device and inode information is limited.
// We extract what's available from the Win32FileAttributeData structure.
func extractSystemMetadata(info os.FileInfo) (dev, ino, uid, gid uint32) {
	// Windows doesn't have traditional Unix inode/device concepts
	// We can get file index and volume serial number for uniqueness
	if stat, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		// Windows doesn't expose device/inode in the same way
		// We'll use what's available or return zeros
		// Git on Windows also typically uses zeros for these fields
		_ = stat // Use the stat to avoid unused variable warning
		return 0, 0, 0, 0
	}
	return 0, 0, 0, 0
}
