package scpath

import "path/filepath"

// String returns the path as a string
func (sp SourcePath) String() string {
	return string(sp)
}

// IsValid checks if this is a valid source path
func (sp SourcePath) IsValid() bool {
	return len(sp) > 0
}

// Join joins path elements to the source path
func (sp SourcePath) Join(elem ...string) SourcePath {
	parts := append([]string{string(sp)}, elem...)
	return SourcePath(filepath.Join(parts...))
}

// ToAbsolutePath converts to an absolute path
func (sp SourcePath) ToAbsolutePath() AbsolutePath {
	return AbsolutePath(sp)
}

// Base returns the last element of the path
func (sp SourcePath) Base() string {
	return filepath.Base(string(sp))
}

// Dir returns all but the last element of the path
func (sp SourcePath) Dir() SourcePath {
	return SourcePath(filepath.Dir(string(sp)))
}

// ObjectsPath returns the path to the objects directory
func (sp SourcePath) ObjectsPath() SourcePath {
	return sp.Join(ObjectsDir)
}

// RefsPath returns the path to the refs directory
func (sp SourcePath) RefsPath() SourcePath {
	return sp.Join(RefsDir)
}

// HeadPath returns the path to the HEAD file
func (sp SourcePath) HeadPath() SourcePath {
	return sp.Join(HeadFile)
}

// IndexPath returns the path to the index file
func (sp SourcePath) IndexPath() SourcePath {
	return sp.Join(IndexFile)
}

// ConfigPath returns the path to the config file
func (sp SourcePath) ConfigPath() SourcePath {
	return sp.Join(ConfigFile)
}

// ObjectFilePath returns the path to an object file given its hash
// Example: hash "abcdef..." returns ".source/objects/ab/cdef..."
func (sp SourcePath) ObjectFilePath(hash string) SourcePath {
	if len(hash) != 40 {
		return ""
	}
	prefix := hash[:2]
	suffix := hash[2:]
	return sp.Join(prefix, suffix)
}
