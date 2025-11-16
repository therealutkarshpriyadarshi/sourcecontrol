package sourcerepo

import (
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

// Repository defines the interface for Git repository operations.
// It provides access to the repository's working directory, git directory,
// and object storage.
type Repository interface {
	// Initialize creates a new repository at the given path
	Initialize(path scpath.RepositoryPath) error

	// WorkingDirectory returns the path to the repository's working directory
	WorkingDirectory() scpath.RepositoryPath

	// SourceDirectory returns the path to the .source directory (equivalent to .git)
	SourceDirectory() scpath.SourcePath

	// ObjectStore returns the object store for this repository
	ObjectStore() store.ObjectStore

	// ReadObject reads a Git object by its SHA-1 hash
	ReadObject(hash objects.ObjectHash) (objects.BaseObject, error)

	// WriteObject writes a Git object to the repository
	WriteObject(obj objects.BaseObject) (objects.ObjectHash, error)

	// Exists checks if a repository exists at the working directory
	Exists() (bool, error)
}
