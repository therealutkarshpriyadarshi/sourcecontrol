package branch

import (
	"context"
	"fmt"
)

// Rename provides functionality for renaming Git branches.
// It handles the complete rename operation including validation,
// updating references, and managing HEAD if the current branch is being renamed.
type Rename struct {
	rs               *BranchRefManager // Manager for branch reference operations
	oldName, newName string            // Source and destination branch names
	force            bool              // If true, overwrite existing branch with newName
}

// NewRenameService creates a new Rename service instance.
//
// Parameters:
//   - rs: The BranchRefManager to use for branch operations
//   - newName: The desired new name for the branch
//   - oldName: The current name of the branch to rename
//   - force: If true, allows overwriting an existing branch with newName
//
// Returns a configured Rename service ready to execute the rename operation.
func NewRenameService(rs *BranchRefManager, newName, oldName string, force bool) *Rename {
	return &Rename{rs: rs, oldName: oldName, newName: newName, force: force}
}

// Execute performs the complete branch rename operation.
// The operation is atomic in the sense that if any step fails, the error is returned
// and subsequent steps are not executed. However, partial changes may have been made
// (e.g., the new branch ref may exist even if deletion of the old branch fails).
func (r *Rename) Execute(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := r.checkBranchExistence(); err != nil {
		return err
	}

	if err := r.updateNew(); err != nil {
		return err
	}

	if err := r.updateOld(); err != nil {
		return err
	}

	return nil
}

// checkBranchExistence validates that the rename operation is possible.
// It verifies that:
//   - The old branch exists
//   - The new branch name is available (or force is true)
func (r *Rename) checkBranchExistence() error {
	exists, err := r.rs.Exists(r.oldName)
	if err != nil {
		return fmt.Errorf("check old branch exists: %w", err)
	}
	if !exists {
		return NewNotFoundError(r.oldName)
	}

	newExists, err := r.rs.Exists(r.newName)
	if err != nil {
		return fmt.Errorf("check new branch exists: %w", err)
	}
	if newExists && !r.force {
		return NewAlreadyExistsError(r.newName)
	}

	return nil
}

// updateNew creates the new branch reference pointing to the same commit as the old branch.
// This step creates the destination branch before any cleanup of the old branch.
func (r *Rename) updateNew() error {
	sha, err := r.rs.Resolve(r.oldName)
	if err != nil {
		return fmt.Errorf("resolve old branch: %w", err)
	}

	if err := r.rs.Update(r.newName, sha, true); err != nil {
		return fmt.Errorf("update new branch ref: %w", err)
	}
	return nil
}

// updateOld handles cleanup of the old branch reference and HEAD update if needed.
// If the current HEAD points to the branch being renamed, HEAD is updated to point
// to the new branch name before the old branch is deleted.
func (r *Rename) updateOld() error {
	current, err := r.rs.Current()
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}
	if current == r.oldName {
		if err := r.rs.SetHead(r.newName); err != nil {
			return fmt.Errorf("update HEAD: %w", err)
		}
	}

	if err := r.rs.Delete(r.oldName); err != nil {
		return fmt.Errorf("delete old branch: %w", err)
	}

	return nil
}
