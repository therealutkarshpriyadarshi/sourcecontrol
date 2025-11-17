package merge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// MergeState manages the state of an in-progress merge operation
type MergeState struct {
	repo *sourcerepo.SourceRepository
}

// NewMergeState creates a new merge state manager
func NewMergeState(repo *sourcerepo.SourceRepository) *MergeState {
	return &MergeState{repo: repo}
}

// InProgress checks if a merge is currently in progress
func (ms *MergeState) InProgress() bool {
	mergeHeadPath := filepath.Join(ms.repo.SourceDirectory().String(), "MERGE_HEAD")
	_, err := os.Stat(mergeHeadPath)
	return err == nil
}

// SaveState saves the merge state to disk
func (ms *MergeState) SaveState(ctx context.Context, mergeHead objects.ObjectHash, mergeMsg string) error {
	gitDir := ms.repo.SourceDirectory().String()

	// Save MERGE_HEAD
	mergeHeadPath := filepath.Join(gitDir, "MERGE_HEAD")
	if err := os.WriteFile(mergeHeadPath, []byte(mergeHead.String()+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write MERGE_HEAD: %w", err)
	}

	// Save MERGE_MSG
	mergeMsgPath := filepath.Join(gitDir, "MERGE_MSG")
	if err := os.WriteFile(mergeMsgPath, []byte(mergeMsg), 0644); err != nil {
		return fmt.Errorf("failed to write MERGE_MSG: %w", err)
	}

	// Save ORIG_HEAD (current HEAD)
	// TODO: Get current HEAD from repository
	// head, err := ms.repo.Head(ctx)
	// if err != nil {
	// 	return fmt.Errorf("failed to get HEAD: %w", err)
	// }
	// origHeadPath := filepath.Join(gitDir, "ORIG_HEAD")
	// if err := os.WriteFile(origHeadPath, []byte(head.String()+"\n"), 0644); err != nil {
	// 	return fmt.Errorf("failed to write ORIG_HEAD: %w", err)
	// }

	return nil
}

// GetMergeHead returns the hash of the commit being merged
func (ms *MergeState) GetMergeHead() (objects.ObjectHash, error) {
	mergeHeadPath := filepath.Join(ms.repo.SourceDirectory().String(), "MERGE_HEAD")
	data, err := os.ReadFile(mergeHeadPath)
	if err != nil {
		return objects.ZeroHash(), fmt.Errorf("failed to read MERGE_HEAD: %w", err)
	}

	hashStr := strings.TrimSpace(string(data))
	hash, err := objects.ParseObjectHash(hashStr)
	if err != nil {
		return objects.ZeroHash(), fmt.Errorf("failed to parse MERGE_HEAD: %w", err)
	}

	return hash, nil
}

// GetMergeMessage returns the prepared merge commit message
func (ms *MergeState) GetMergeMessage() (string, error) {
	mergeMsgPath := filepath.Join(ms.repo.SourceDirectory().String(), "MERGE_MSG")
	data, err := os.ReadFile(mergeMsgPath)
	if err != nil {
		return "", fmt.Errorf("failed to read MERGE_MSG: %w", err)
	}

	return string(data), nil
}

// GetOrigHead returns the original HEAD before merge started
func (ms *MergeState) GetOrigHead() (objects.ObjectHash, error) {
	origHeadPath := filepath.Join(ms.repo.SourceDirectory().String(), "ORIG_HEAD")
	data, err := os.ReadFile(origHeadPath)
	if err != nil {
		return objects.ZeroHash(), fmt.Errorf("failed to read ORIG_HEAD: %w", err)
	}

	hashStr := strings.TrimSpace(string(data))
	hash, err := objects.ParseObjectHash(hashStr)
	if err != nil {
		return objects.ZeroHash(), fmt.Errorf("failed to parse ORIG_HEAD: %w", err)
	}

	return hash, nil
}

// ClearState removes all merge state files
func (ms *MergeState) ClearState() error {
	gitDir := ms.repo.SourceDirectory().String()

	files := []string{
		filepath.Join(gitDir, "MERGE_HEAD"),
		filepath.Join(gitDir, "MERGE_MSG"),
		filepath.Join(gitDir, "ORIG_HEAD"),
		filepath.Join(gitDir, "MERGE_MODE"),
	}

	var errs []error
	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("failed to remove %s: %w", filepath.Base(file), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors clearing merge state: %v", errs)
	}

	return nil
}

// Abort aborts an in-progress merge
func (ms *MergeState) Abort(ctx context.Context) error {
	if !ms.InProgress() {
		return fmt.Errorf("no merge in progress")
	}

	// Get the original HEAD
	origHead, err := ms.GetOrigHead()
	if err != nil {
		return fmt.Errorf("failed to get original HEAD: %w", err)
	}

	// Reset the index and working directory to ORIG_HEAD
	// TODO: Implement reset functionality
	// This requires SetHead method on SourceRepository
	_ = origHead

	// Clear the merge state
	if err := ms.ClearState(); err != nil {
		return fmt.Errorf("failed to clear merge state: %w", err)
	}

	return nil
}

// Continue continues a merge after conflicts have been resolved
func (ms *MergeState) Continue(ctx context.Context) error {
	if !ms.InProgress() {
		return fmt.Errorf("no merge in progress")
	}

	// Get the merge message
	mergeMsg, err := ms.GetMergeMessage()
	if err != nil {
		return fmt.Errorf("failed to get merge message: %w", err)
	}

	// The actual commit creation will be handled by the caller
	// This just validates that we can continue
	_ = mergeMsg
	return nil
}

// SaveMergeMode saves additional merge mode flags
func (ms *MergeState) SaveMergeMode(mode string) error {
	gitDir := ms.repo.SourceDirectory().String()
	mergeModePath := filepath.Join(gitDir, "MERGE_MODE")
	return os.WriteFile(mergeModePath, []byte(mode), 0644)
}

// GetMergeMode returns the merge mode if set
func (ms *MergeState) GetMergeMode() (string, error) {
	mergeModePath := filepath.Join(ms.repo.SourceDirectory().String(), "MERGE_MODE")
	data, err := os.ReadFile(mergeModePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read MERGE_MODE: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}
