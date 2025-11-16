package branch

import (
	"context"
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/workdir"
)

type branchResolve struct {
	sha      objects.ObjectHash
	isBranch bool
}

// Checkout handles branch checkout operations, including updating
// the working directory and HEAD reference
type Checkout struct {
	repo           *sourcerepo.SourceRepository
	refService     *BranchRefManager
	creator        *Creator
	workdirManager *workdir.Manager
}

// NewCheckout creates a new checkout service
func NewCheckout(
	repo *sourcerepo.SourceRepository,
	refSvc *BranchRefManager,
	creator *Creator,
	workdirMgr *workdir.Manager,
) *Checkout {
	return &Checkout{
		repo:           repo,
		refService:     refSvc,
		creator:        creator,
		workdirManager: workdirMgr,
	}
}

// Checkout switches to a different branch or commit
func (co *Checkout) Checkout(ctx context.Context, target string, config *CheckoutConfig) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if config.Orphan {
		return co.checkoutOrphan(ctx, target)
	}

	resolved, err := co.resolveTarget(target, config)
	if err != nil {
		return err
	}

	if err := co.checkAlreadyCheckedOut(target, resolved.isBranch); err != nil {
		return err
	}

	updateOpts := []workdir.Option{}
	if config.Force {
		updateOpts = append(updateOpts, workdir.WithForce())
	}

	result, err := co.workdirManager.UpdateToCommit(ctx, resolved.sha, updateOpts...)
	if err != nil {
		return fmt.Errorf("update working directory: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("failed to update working directory: %v", result.Err)
	}

	if config.Detach || !resolved.isBranch {
		if err := co.refService.SetHeadDetached(resolved.sha); err != nil {
			return fmt.Errorf("set detached HEAD: %w", err)
		}
	} else {
		if err := co.refService.SetHead(target); err != nil {
			return fmt.Errorf("set HEAD to branch: %w", err)
		}
	}

	return nil
}

// resolveTarget resolves a target (branch name or commit SHA) to a commit hash
// Returns: (commitSHA, isBranch, error)
func (co *Checkout) resolveTarget(target string, config *CheckoutConfig) (*branchResolve, error) {
	var options ResolveOptions
	if config.Create {
		options = ResolveOptions{
			AllowCreate: true,
			CreateFunc:  co.createBranch,
		}
	}

	result, err := ResolveRefOrCommit(target, co.refService, co.repo, options)
	if err != nil {
		return nil, err
	}

	return &branchResolve{
		sha:      result.SHA,
		isBranch: result.IsBranch,
	}, nil
}

func (co *Checkout) createBranch(name string) (objects.ObjectHash, error) {
	createConfig := &CreateConfig{
		StartPoint: "",
		Force:      false,
	}
	info, err := co.creator.Create(context.Background(), name, createConfig)
	if err != nil {
		return "", fmt.Errorf("create branch: %w", err)
	}
	return info.SHA, nil
}

// checkAlreadyCheckedOut checks if we're already on the target
func (co *Checkout) checkAlreadyCheckedOut(target string, isBranch bool) error {
	if !isBranch {
		return nil
	}

	current, err := co.refService.Current()
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}

	if current == target {
		return nil
	}

	return nil
}

// checkoutOrphan creates and checks out an orphan branch
func (co *Checkout) checkoutOrphan(ctx context.Context, branchName string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := co.creator.CreateOrphan(ctx, branchName); err != nil {
		return fmt.Errorf("create orphan branch: %w", err)
	}

	return nil
}

// CheckoutCommit is a convenience method to checkout a specific commit (detached HEAD)
func (co *Checkout) CheckoutCommit(ctx context.Context, sha objects.ObjectHash, force bool) error {
	config := &CheckoutConfig{
		Force:  force,
		Detach: true,
	}

	return co.Checkout(ctx, sha.String(), config)
}

// CheckoutBranch is a convenience method to checkout a branch by name
func (co *Checkout) CheckoutBranch(ctx context.Context, branchName string, force bool) error {
	config := &CheckoutConfig{
		Force:  force,
		Detach: false,
	}

	return co.Checkout(ctx, branchName, config)
}
