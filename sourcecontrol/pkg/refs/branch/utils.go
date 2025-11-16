package branch

import (
	"fmt"
	"os"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// resolveCommit resolves a commit SHA string (full or short) to an ObjectHash
func resolveCommit(commitStr string, repo sourcerepo.SourceRepository) (objects.ObjectHash, error) {
	sha, err := objects.NewObjectHashFromString(commitStr)
	if err == nil {
		_, err = repo.ObjectStore().HasObject(sha)
		if err != nil {
			return "", fmt.Errorf("commit %s does not exist: %w", sha.Short(), err)
		}
		return sha, nil
	}

	if !(commit.LooksLikeCommitSHA(commitStr)) {
		return "", fmt.Errorf("invalid target '%s': not a valid branch name or commit SHA", commitStr)
	}

	fullSHA, err := resolveShortSHA(commitStr, repo)
	if err != nil {
		return "", err
	}

	return fullSHA, nil
}

// resolveShortSHA finds the full SHA for a short SHA prefix
func resolveShortSHA(shortSHA string, repo sourcerepo.SourceRepository) (objects.ObjectHash, error) {
	objectsPath := repo.ObjectsPath()

	if len(shortSHA) < 4 {
		return "", fmt.Errorf("short SHA must be at least 4 characters")
	}

	dirName := shortSHA[:2]
	filePrefix := shortSHA[2:]

	dirPath := objectsPath.Join(dirName)
	entries, err := os.ReadDir(string(dirPath))
	if err != nil {
		return "", fmt.Errorf("commit %s does not exist", shortSHA)
	}

	var matches []objects.ObjectHash
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		if len(fileName) >= len(filePrefix) && fileName[:len(filePrefix)] == filePrefix {
			fullSHA := dirName + fileName
			hash, err := objects.NewObjectHashFromString(fullSHA)
			if err != nil {
				continue
			}

			_, err = repo.ObjectStore().HasObject(hash)
			if err == nil {
				matches = append(matches, hash)
			}
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("commit %s does not exist", shortSHA)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("short SHA %s is ambiguous (matches %d commits)", shortSHA, len(matches))
	}

	return matches[0], nil
}

func resolveBranch(refService *BranchRefManager, target string) (objects.ObjectHash, error) {
	exists, err := refService.Exists(target)
	if err != nil {
		return "", fmt.Errorf("check branch exists: %w", err)
	}

	if exists {
		sha, err := refService.Resolve(target)
		if err != nil {
			return "", fmt.Errorf("resolve branch: %w", err)
		}
		return sha, nil
	}

	return "", nil
}

// ResolveOptions configures the behavior of ResolveRefOrCommit
type ResolveOptions struct {
	AllowCreate bool

	// CreateFunc is called to create a new branch if AllowCreate is true
	CreateFunc func(string) (objects.ObjectHash, error)

	// DefaultValue is returned if target is empty
	DefaultValue objects.ObjectHash
}

// ResolveResult contains the resolution result
type ResolveResult struct {
	SHA               objects.ObjectHash
	IsBranch, Created bool
}

func newResolveResult(sha objects.ObjectHash, isBranch, created bool) *ResolveResult {
	return &ResolveResult{
		SHA:      sha,
		IsBranch: isBranch,
		Created:  created,
	}
}

// ResolveRefOrCommit resolves a target as either a branch name or commit SHA
// It checks if the target looks like a commit SHA first, then attempts branch resolution
func ResolveRefOrCommit(
	target string,
	refService *BranchRefManager,
	repo *sourcerepo.SourceRepository,
	o ResolveOptions,
) (*ResolveResult, error) {
	if target == "" {
		if o.DefaultValue != "" {
			return newResolveResult(o.DefaultValue, false, false), nil
		}
		return nil, fmt.Errorf("target cannot be empty")
	}

	if commit.LooksLikeCommitSHA(target) {
		sha, err := resolveCommit(target, *repo)
		if err == nil {
			return newResolveResult(sha, false, false), nil
		}

		if len(target) >= 4 && len(target) < 40 {
			return nil, err
		}
	}

	// Try to resolve as a branch name
	if err := refService.validateBranchName(target); err == nil {
		sha, err := resolveBranch(refService, target)
		if err != nil {
			return nil, err
		}

		if sha != "" {
			return newResolveResult(sha, true, false), nil
		}

		if o.AllowCreate && o.CreateFunc != nil {
			sha, err := o.CreateFunc(target)
			if err != nil {
				return nil, fmt.Errorf("create branch: %w", err)
			}

			return newResolveResult(sha, true, true), nil
		}

		return nil, NewNotFoundError(target)
	}

	return nil, fmt.Errorf("invalid target '%s': not a valid branch name or commit SHA", target)
}
