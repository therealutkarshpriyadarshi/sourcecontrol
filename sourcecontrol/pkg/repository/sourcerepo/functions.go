package sourcerepo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// FindRepository searches for a source control repository by traversing up the directory tree
// from the given start path. It implements a bottom-up search strategy to locate the nearest
// repository in the parent directory hierarchy.
//
// Parameters:
//   - startPath: The initial directory path from which to begin the search
//
// Returns:
//   - *SourceRepository: A pointer to the found repository, fully initialized and ready to use
//   - error: An error if the search fails due to filesystem issues or initialization problems
func FindRepository(startPath scpath.RepositoryPath) (*SourceRepository, error) {
	currentPath := startPath.String()

	for {
		repoPath, err := scpath.NewRepositoryPath(currentPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create repository path: %w", err)
		}

		exists, err := RepositoryExists(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to check repository existence: %w", err)
		}

		if exists {
			repo := NewSourceRepository()
			repo.workingDir = repoPath
			repo.sourceDir = repoPath.SourcePath()

			if err := repo.objectStore.Initialize(repoPath); err != nil {
				return nil, fmt.Errorf("failed to initialize object store: %w", err)
			}

			repo.initialized = true
			return repo, nil
		}

		parentPath := filepath.Dir(currentPath)

		if parentPath == currentPath {
			return nil, nil
		}

		currentPath = parentPath
	}
}

// RepositoryExists checks whether a valid source control repository exists at the specified path.
// A repository is considered to exist if there is a .git directory at the given location.
//
// Parameters:
//   - path: The repository path to check for existence
//
// Returns:
//   - bool: true if a valid repository exists at the path, false otherwise
//   - error: An error if the filesystem check fails (excluding non-existence errors)
func RepositoryExists(path scpath.RepositoryPath) (bool, error) {
	sourcePath := path.SourcePath()
	info, err := os.Stat(sourcePath.String())

	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to check .git directory: %w", err)
	}

	return info.IsDir(), nil
}

// Open opens and initializes an existing source control repository at the specified path.
// This function is used to work with repositories that have already been initialized.
//
// Parameters:
//   - path: The root directory path of the repository to open
//
// Returns:
//   - *SourceRepository: A fully initialized repository instance ready for operations
//   - error: An error if the repository doesn't exist or initialization fails
func Open(path scpath.RepositoryPath) (*SourceRepository, error) {
	exists, err := RepositoryExists(path)
	if err != nil {
		return nil, fmt.Errorf("failed to check repository existence: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("not a source repository: %s", path)
	}

	repo := NewSourceRepository()
	repo.workingDir = path
	repo.sourceDir = path.SourcePath()

	if err := repo.objectStore.Initialize(path); err != nil {
		return nil, fmt.Errorf("failed to initialize object store: %w", err)
	}

	repo.initialized = true
	return repo, nil
}

// InitializeRepository is a convenience function to initialize a new repository.
// It creates a new SourceRepository instance and initializes it at the given path.
//
// Parameters:
//   - path: The source path where the repository should be initialized
//   - bare: Whether to create a bare repository (no working directory)
//
// Returns:
//   - error: nil on success, or an error if initialization fails
func InitializeRepository(path scpath.SourcePath, bare bool) error {
	repoPath := scpath.RepositoryPath(path.String())

	repo := NewSourceRepository()
	if err := repo.Initialize(repoPath); err != nil {
		return err
	}

	return nil
}
