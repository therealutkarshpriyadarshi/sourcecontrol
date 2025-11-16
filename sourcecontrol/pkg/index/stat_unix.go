//go:build unix || linux || darwin

package index

import (
	"os"
	"syscall"
)

// extractSystemMetadata extracts platform-specific file system metadata.
// On Unix systems, this includes device ID, inode, user ID, and group ID.
func extractSystemMetadata(info os.FileInfo) (dev, ino, uid, gid uint32) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint32(stat.Dev),
			uint32(stat.Ino),
			uint32(stat.Uid),
			uint32(stat.Gid)
	}
	return 0, 0, 0, 0
}
