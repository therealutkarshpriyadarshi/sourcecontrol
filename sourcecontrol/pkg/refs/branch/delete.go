package branch

import (
	"context"
	"fmt"
)

// Delete handles branch deletion operations
type Delete struct {
	refService *BranchRefManager
}

// NewDelete creates a new branch delete service
func NewDelete(refSvc *BranchRefManager) *Delete {
	return &Delete{
		refService: refSvc,
	}
}

// Delete deletes a branch with the given configuration
func (d *Delete) Delete(ctx context.Context, name string, config *DeleteConfig) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !config.Force {
		// TODO: Implement merge check
		// For now, we'll allow deletion
		// A proper implementation would check if all commits in this branch
		// are reachable from another branch (meaning it's been merged)
	}

	if err := d.refService.Delete(name); err != nil {
		return fmt.Errorf("delete branch: %w", err)
	}

	return nil
}

// DeleteMultiple deletes multiple branches
func (d *Delete) DeleteMultiple(ctx context.Context, names []string, config *DeleteConfig) error {
	var firstError error

	for _, name := range names {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := d.Delete(ctx, name, config); err != nil {
			if firstError == nil {
				firstError = err
			}
		}
	}

	return firstError
}

// IsMerged checks if a branch has been fully merged into another branch
// This is a placeholder for future implementation
func (d *Delete) IsMerged(ctx context.Context, branchName, targetBranch string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	// TODO: Implement proper merge check
	// This would involve:
	// 1. Getting all commits in branchName
	// 2. Getting all commits in targetBranch
	// 3. Checking if all commits from branchName are reachable from targetBranch
	// For now, return false (not merged) to be safe

	return false, nil
}
