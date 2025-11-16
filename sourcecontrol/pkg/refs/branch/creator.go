package branch

import (
	"context"
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// Creator handles branch creation operations
type Creator struct {
	repo        *sourcerepo.SourceRepository
	refService  *BranchRefManager
	infoService *InfoService
}

// NewCreator creates a new branch creator service
func NewCreator(repo *sourcerepo.SourceRepository, refSvc *BranchRefManager, infoSvc *InfoService) *Creator {
	return &Creator{
		repo:        repo,
		refService:  refSvc,
		infoService: infoSvc,
	}
}

// Create creates a new branch with the given configuration
func (c *Creator) Create(ctx context.Context, name string, config *CreateConfig) (*BranchInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if !config.Force {
		if err := c.validateNotExists(name); err != nil {
			return nil, err
		}
	}

	startSha, err := c.resolveStartPoint(config.StartPoint)
	if err != nil {
		return nil, fmt.Errorf("resolve start point: %w", err)
	}

	if err := c.verifyCommitExists(startSha); err != nil {
		return nil, fmt.Errorf("verify commit: %w", err)
	}

	if err := c.createOrUpdate(name, startSha, config.Force); err != nil {
		return nil, err
	}

	return c.infoService.GetInfo(ctx, name)

}

func (c *Creator) createOrUpdate(name string, startSha objects.ObjectHash, force bool) error {
	if force {
		if err := c.refService.Update(name, startSha, true); err != nil {
			return fmt.Errorf("update branch: %w", err)
		}
		return nil
	}

	if err := c.refService.Create(name, startSha); err != nil {
		return fmt.Errorf("create branch: %w", err)
	}
	return nil
}

// resolveStartPoint resolves the start point to a commit SHA
func (c *Creator) resolveStartPoint(startPoint string) (objects.ObjectHash, error) {
	headSHA, err := c.refService.GetHeadSHA()
	if err != nil {
		return "", fmt.Errorf("get HEAD SHA: %w", err)
	}

	options := ResolveOptions{
		DefaultValue: headSHA,
	}

	result, err := ResolveRefOrCommit(startPoint, c.refService, c.repo, options)
	if err != nil {
		return "", err
	}

	return result.SHA, nil
}

// verifyCommitExists checks if a commit object exists in the repository
func (c *Creator) verifyCommitExists(sha objects.ObjectHash) error {
	exists, err := c.repo.ObjectStore().HasObject(sha)
	if err != nil || !exists {
		return fmt.Errorf("commit %s does not exist: %w", sha.Short(), err)
	}
	return nil
}

// CreateOrphan creates an orphan branch (a branch with no parent commits)
func (c *Creator) CreateOrphan(ctx context.Context, name string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := c.validateNotExists(name); err != nil {
		return err
	}

	// For orphan branches, we just update HEAD to point to the new branch
	// The branch ref won't be created until the first commit is made
	if err := c.refService.SetHead(name); err != nil {
		// If setting HEAD fails, it's because the branch doesn't exist yet
		// For orphan branches, we need to create an empty ref first
		// We'll use a special empty commit marker
		return fmt.Errorf("create orphan branch: %w", err)
	}

	return nil
}

func (c *Creator) validateNotExists(name string) error {
	exists, err := c.refService.Exists(name)
	if err != nil {
		return fmt.Errorf("check branch exists: %w", err)
	}
	if exists {
		return NewAlreadyExistsError(name)
	}

	return nil
}
