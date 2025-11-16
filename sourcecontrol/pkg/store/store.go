package store

import (
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// ObjectStore defines the interface for Git object storage operations
// It provides methods to read, write, and check the existence of Git objects
type ObjectStore interface {
	// Initialize sets up the object store with the given repository path
	// Creates necessary directory structures if they don't exist
	Initialize(repoPath scpath.RepositoryPath) error

	// WriteObject stores a Git object and returns its SHA-1 hash
	// If the object already exists, it returns the hash without rewriting
	WriteObject(obj objects.BaseObject) (objects.ObjectHash, error)

	// ReadObject retrieves a Git object by its SHA-1 hash
	// Returns nil if the object doesn't exist
	ReadObject(hash objects.ObjectHash) (objects.BaseObject, error)

	// HasObject checks if an object exists in the store
	// Returns true if the object exists, false otherwise
	HasObject(hash objects.ObjectHash) (bool, error)
}
