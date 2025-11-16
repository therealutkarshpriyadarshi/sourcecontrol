package sourcerepo

import (
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/common/fileops"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/blob"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

// SourceRepository is a Git repository implementation that manages the complete Git repository
// structure and provides access to Git objects, references, and configuration.
//
// The repository manages both the working directory (user files) and the Source
// directory (metadata and object storage).
//
// Thread Safety:
// This struct is not thread-safe. External synchronization is required when
// accessing a SourceRepository instance from multiple goroutines.
type SourceRepository struct {
	workingDir  scpath.RepositoryPath
	sourceDir   scpath.SourcePath
	objectStore store.ObjectStore
	initialized bool
}

// NewSourceRepository creates a new SourceRepository instance in an uninitialized state.
//
// The returned repository must be initialized by calling Initialize() before use.
// The repository is configured with a FileObjectStore for persistent object storage.
//
// Returns:
//   - *SourceRepository: A new uninitialized repository instance
func NewSourceRepository() *SourceRepository {
	return &SourceRepository{
		objectStore: store.NewFileObjectStore(),
		initialized: false,
	}
}

// Initialize creates a new repository at the given path.
//
// This method sets up the complete repository structure including all necessary
// directories and initial configuration files. It will fail if a repository
// already exists at the specified path.
//
// Directory structure created:
//   - .source/              (main Git metadata directory)
//   - .source/objects/      (object database for blobs, trees, commits, tags)
//   - .source/refs/         (references directory)
//   - .source/refs/heads/   (branch references)
//   - .source/refs/tags/    (tag references)
//
// Files created:
//   - .source/HEAD          (points to refs/heads/master by default)
//   - .source/config        (repository configuration with core settings)
//   - .source/description   (human-readable repository description)
//
// Parameters:
//   - path: The directory path where the repository should be initialized
func (sr *SourceRepository) Initialize(path scpath.RepositoryPath) error {
	exists, err := RepositoryExists(path)
	if err != nil {
		return fmt.Errorf("failed to check if repository exists: %w", err)
	}
	if exists {
		return fmt.Errorf("already a source repository: %s", path)
	}

	sr.workingDir = path
	sr.sourceDir = path.SourcePath()

	if err := sr.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	if err := sr.objectStore.Initialize(sr.workingDir); err != nil {
		return fmt.Errorf("failed to initialize object store: %w", err)
	}

	if err := sr.createInitialFiles(); err != nil {
		return fmt.Errorf("failed to create initial files: %w", err)
	}

	sr.initialized = true
	return nil
}

// WorkingDirectory returns the path to the repository's working directory.
//
// The working directory is the root directory of the repository where user files
// and directories are stored. It is the parent directory of the .git folder.
//
// Returns:
//   - scpath.RepositoryPath: The absolute path to the working directory
//
// Panics:
//   - If the repository has not been initialized via Initialize()
func (sr *SourceRepository) WorkingDirectory() scpath.RepositoryPath {
	if !sr.initialized {
		panic("repository not initialized")
	}
	return sr.workingDir
}

// SourceDirectory returns the path to the .git metadata directory.
//
// Returns:
//   - scpath.SourcePath: The absolute path to the .git directory
//
// Panics:
//   - If the repository has not been initialized via Initialize()
func (sr *SourceRepository) SourceDirectory() scpath.SourcePath {
	if !sr.initialized {
		panic("repository not initialized")
	}
	return sr.sourceDir
}

// ObjectStore returns the object store for this repository.
//
// The object store provides the interface for reading and writing Git objects
// (blobs, trees, commits, and tags) to persistent storage.
//
// Returns:
//   - store.ObjectStore: The object store instance used by this repository
func (sr *SourceRepository) ObjectStore() store.ObjectStore {
	return sr.objectStore
}

// ReadObject reads a Git object by its SHA-1 hash from the object store.
//
// This method retrieves and deserializes a Git object (blob, tree, commit, or tag)
// from the repository's object database. The object is identified by its SHA-1
// hash, which is computed from the object's content.
//
// Parameters:
//   - hash: The SHA-1 hash of the object to read (40 character hex string)
func (sr *SourceRepository) ReadObject(hash objects.ObjectHash) (objects.BaseObject, error) {
	if !sr.initialized {
		return nil, fmt.Errorf("repository not initialized")
	}

	obj, err := sr.objectStore.ReadObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}
	return obj, nil
}

// WriteObject writes a Git object to the repository and returns its SHA-1 hash.
//
// This method serializes a Git object (blob, tree, commit, or tag) and stores it
// in the repository's object database. The object is content-addressable, meaning
// its SHA-1 hash is computed from its content and serves as its unique identifier.
//
// The object is stored in the following location:
//
//	.source/objects/<first-2-chars-of-hash>/<remaining-38-chars-of-hash>
//
// Parameters:
//   - obj: The Git object to write (must implement objects.BaseObject interface)
func (sr *SourceRepository) WriteObject(obj objects.BaseObject) (objects.ObjectHash, error) {
	if !sr.initialized {
		return "", fmt.Errorf("repository not initialized")
	}

	hash, err := sr.objectStore.WriteObject(obj)
	if err != nil {
		return "", fmt.Errorf("failed to write object: %w", err)
	}
	return hash, nil
}

// Exists checks if a valid repository exists at the working directory.
func (sr *SourceRepository) Exists() (bool, error) {
	if !sr.initialized {
		return false, fmt.Errorf("repository not initialized")
	}
	return RepositoryExists(sr.workingDir)
}

// IsInitialized returns whether the repository has been properly initialized.
func (sr *SourceRepository) IsInitialized() bool {
	return sr.initialized
}

// createDirectories creates all necessary directories for the repository structure.
//
// This internal method creates the complete directory hierarchy required for a
// Git repository, including the main .source directory and all subdirectories
// for objects and references.
//
// Directories created:
//   - .source/              (main metadata directory)
//   - .source/objects/      (object database)
//   - .source/refs/         (references root)
//   - .source/refs/heads/   (branch references)
//   - .source/refs/tags/    (tag references)
//
// All directories are created with permissions 0755 (rwxr-xr-x).
//
// Returns:
//   - error: nil on success, or an error if any directory creation fails
func (sr *SourceRepository) createDirectories() error {
	source := sr.sourceDir
	directories := []scpath.SourcePath{
		source,
		source.ObjectsPath(),
		source.RefsPath(),
		source.RefsPath().Join(scpath.HeadsDir),
		source.RefsPath().Join(scpath.TagsDir),
	}

	for _, dir := range directories {
		if err := fileops.EnsureDir(dir.ToAbsolutePath()); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// createInitialFiles creates the initial configuration and metadata files for a new repository.
//
// This internal method creates three essential files that every Git repository needs:
//
//  1. HEAD: Points to the current branch (initially refs/heads/master)
//  2. description: Contains a human-readable description of the repository
//  3. config: Contains the repository configuration in Git INI format
//
// The config file includes:
//   - repositoryformatversion = 0 (Git object format version)
//   - filemode = false (disable executable bit tracking on Windows)
//   - bare = false (this is a repository with a working directory)
//
// All files are created with permissions 0644 (rw-r--r--).
//
// Returns:
//   - error: nil on success, or an error if any file creation fails
func (sr *SourceRepository) createInitialFiles() error {
	files := []struct {
		path    scpath.SourcePath
		content string
		name    string
	}{
		{
			path:    sr.sourceDir.HeadPath(),
			content: "ref: refs/heads/master\n",
			name:    "HEAD",
		},
		{
			path:    sr.sourceDir.Join("description"),
			content: "Unnamed repository; edit this file 'description' to name the repository.\n",
			name:    "description",
		},
		{
			path: sr.sourceDir.ConfigPath(),
			content: `[core]
    repositoryformatversion = 0
    filemode = false
    bare = false
`,
			name: "config",
		},
	}

	for _, file := range files {
		if err := fileops.WriteConfig(file.path.ToAbsolutePath(), []byte(file.content)); err != nil {
			return fmt.Errorf("failed to create %s file: %w", file.name, err)
		}
	}

	return nil
}

// ReadBlobObject reads and validates a blob object from the repository.
//
// Parameters:
//   - blobSHA: The SHA-1 hash of the blob object to read
func (sr *SourceRepository) ReadBlobObject(blobSHA objects.ObjectHash) (*blob.Blob, error) {
	obj, err := sr.ReadObject(blobSHA)
	if err != nil {
		return nil, fmt.Errorf("read blob %s: %w", blobSHA.Short(), err)
	}

	blobObj, ok := obj.(*blob.Blob)
	if !ok {
		return nil, fmt.Errorf("object %s is not a blob", blobSHA.Short())
	}

	return blobObj, nil
}

// ReadTreeObject reads and validates a tree object from the repository.
//
// Parameters:
//   - treeSHA: The SHA-1 hash of the tree object to read
func (sr *SourceRepository) ReadTreeObject(treeSHA objects.ObjectHash) (*tree.Tree, error) {
	obj, err := sr.ReadObject(treeSHA)
	if err != nil {
		return nil, fmt.Errorf("read tree %s: %w", treeSHA.Short(), err)
	}

	treeObj, ok := obj.(*tree.Tree)
	if !ok {
		return nil, fmt.Errorf("object %s is not a tree", treeSHA.Short())
	}

	return treeObj, nil
}

// ReadCommitObject reads and validates a commit object from the repository.
//
// Parameters:
//   - commitSHA: The SHA-1 hash of the commit object to read
func (sr *SourceRepository) ReadCommitObject(commitSHA objects.ObjectHash) (*commit.Commit, error) {
	obj, err := sr.ReadObject(commitSHA)
	if err != nil {
		return nil, fmt.Errorf("read commit %s: %w", commitSHA.Short(), err)
	}

	commitObj, ok := obj.(*commit.Commit)
	if !ok {
		return nil, fmt.Errorf("object %s is not a commit", commitSHA.Short())
	}

	return commitObj, nil
}

// SourcePath returns the path to the .source directory.
// This is a convenience method for accessing the source directory path.
func (sr *SourceRepository) SourcePath() scpath.SourcePath {
	return sr.SourceDirectory()
}

// ObjectsPath returns the path to the objects directory.
// This is a convenience method for accessing the objects storage path.
func (sr *SourceRepository) ObjectsPath() scpath.SourcePath {
	return sr.sourceDir.ObjectsPath()
}
