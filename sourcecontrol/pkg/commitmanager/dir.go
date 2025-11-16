package commitmanager

import (
	"path/filepath"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

// directoryNode represents a directory in the in-memory tree structure.
//
// It holds references to:
//   - files: Regular files in this directory (name -> blob SHA)
//   - modes: File permissions/modes for each file
//   - subdirs: Child directories (name -> directoryNode)
//
// This structure is built up from flat paths and then recursively converted
// into Git tree objects.
type directoryNode struct {
	name    string                        // Directory name (empty for root)
	files   map[string]objects.ObjectHash // filename -> blob SHA
	modes   map[string]objects.FileMode   // filename -> file mode
	subdirs map[string]*directoryNode     // dirname -> subdirectory node
}

// newDirectoryNode creates a new directory node with initialized maps
func newDirectoryNode(name string) *directoryNode {
	return &directoryNode{
		name:    name,
		files:   make(map[string]objects.ObjectHash),
		modes:   make(map[string]objects.FileMode),
		subdirs: make(map[string]*directoryNode),
	}
}

// addEntry adds a file entry to the directory tree recursively.
//
// For a path like "src/utils/helper.go":
//  1. Split into parts: ["src", "utils", "helper.go"]
//  2. If only one part, add as file to current directory
//  3. Otherwise, create/get subdirectory "src" and recurse with "utils/helper.go"
func (dn *directoryNode) addEntry(path string, sha objects.ObjectHash, mode objects.FileMode) {
	parts := strings.Split(filepath.ToSlash(path), "/")

	if len(parts) == 1 {
		dn.addFile(parts[0], sha, mode)
		return
	}

	firstDir := parts[0]
	restOfPath := strings.Join(parts[1:], "/")

	subdir := dn.getOrCreateSubdir(firstDir)
	subdir.addEntry(restOfPath, sha, mode)
}

// addFile adds a file to this directory node
func (dn *directoryNode) addFile(name string, sha objects.ObjectHash, mode objects.FileMode) {
	dn.files[name] = sha
	dn.modes[name] = mode
}

// getOrCreateSubdir gets an existing subdirectory or creates a new one
func (dn *directoryNode) getOrCreateSubdir(name string) *directoryNode {
	if subdir, exists := dn.subdirs[name]; exists {
		return subdir
	}

	subdir := newDirectoryNode(name)
	dn.subdirs[name] = subdir
	return subdir
}
