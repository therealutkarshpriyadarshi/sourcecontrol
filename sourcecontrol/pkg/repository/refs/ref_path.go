package refs

import (
	"fmt"
	"strings"
)

// RefPath represents a Git reference path
// Examples: "refs/heads/main", "refs/tags/v1.0.0", "HEAD"
type RefPath string

// String returns the reference path as a string
func (rp RefPath) String() string {
	return string(rp)
}

// IsValid checks if this is a valid reference path
func (rp RefPath) IsValid() bool {
	s := string(rp)
	if len(s) == 0 {
		return false
	}

	invalidChars := []string{" ", "~", "^", ":", "?", "*", "[", "\\", "..", "@{", "//"}
	for _, invalid := range invalidChars {
		if strings.Contains(s, invalid) {
			return false
		}
	}

	if strings.HasSuffix(s, ".lock") || strings.HasSuffix(s, ".") {
		return false
	}

	if strings.HasPrefix(s, ".") {
		return false
	}
	return true
}

// IsBranch checks if this is a branch reference
func (rp RefPath) IsBranch() bool {
	return strings.HasPrefix(string(rp), "refs/heads/")
}

// IsTag checks if this is a tag reference
func (rp RefPath) IsTag() bool {
	return strings.HasPrefix(string(rp), "refs/tags/")
}

// IsRemote checks if this is a remote reference
func (rp RefPath) IsRemote() bool {
	return strings.HasPrefix(string(rp), "refs/remotes/")
}

// IsHEAD checks if this is the HEAD reference
func (rp RefPath) IsHEAD() bool {
	return rp == RefHEAD
}

// ShortName returns the short name of the reference
// "refs/heads/main" -> "main"
// "refs/tags/v1.0.0" -> "v1.0.0"
// "refs/remotes/origin/main" -> "origin/main"
// "HEAD" -> "HEAD"
func (rp RefPath) ShortName() string {
	s := string(rp)
	if rp.IsBranch() {
		return strings.TrimPrefix(s, "refs/heads/")
	}
	if rp.IsTag() {
		return strings.TrimPrefix(s, "refs/tags/")
	}
	if rp.IsRemote() {
		return strings.TrimPrefix(s, "refs/remotes/")
	}
	return s
}

// NewBranchRef creates a branch reference path
func NewBranchRef(name string) (RefPath, error) {
	if len(name) == 0 {
		return "", fmt.Errorf("branch name cannot be empty")
	}
	refPath := RefPath("refs/heads/" + name)
	if !refPath.IsValid() {
		return "", fmt.Errorf("invalid branch name: %s", name)
	}
	return refPath, nil
}

// NewTagRef creates a tag reference path
func NewTagRef(name string) (RefPath, error) {
	if len(name) == 0 {
		return "", fmt.Errorf("tag name cannot be empty")
	}
	refPath := RefPath("refs/tags/" + name)
	if !refPath.IsValid() {
		return "", fmt.Errorf("invalid tag name: %s", name)
	}
	return refPath, nil
}

// NewRemoteRef creates a remote reference path
func NewRemoteRef(remote, branch string) (RefPath, error) {
	if len(remote) == 0 {
		return "", fmt.Errorf("remote name cannot be empty")
	}
	if len(branch) == 0 {
		return "", fmt.Errorf("branch name cannot be empty")
	}
	refPath := RefPath(fmt.Sprintf("refs/remotes/%s/%s", remote, branch))
	if !refPath.IsValid() {
		return "", fmt.Errorf("invalid remote ref: %s/%s", remote, branch)
	}
	return refPath, nil
}
