package branch

import (
	"context"
	"fmt"
	"runtime"

	pool "github.com/utkarsh5026/SourceControl/pkg/common/concurrency"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"golang.org/x/sync/errgroup"
)

// InfoService provides branch information and metadata
type InfoService struct {
	repo *sourcerepo.SourceRepository
	rs   *BranchRefManager
}

// NewInfoService creates a new branch info service
func NewInfoService(repo *sourcerepo.SourceRepository, refSvc *BranchRefManager) *InfoService {
	return &InfoService{
		repo: repo,
		rs:   refSvc,
	}
}

// GetInfo retrieves detailed information about a specific branch
func (is *InfoService) GetInfo(ctx context.Context, name string) (*BranchInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if err := is.rs.ValidateExists(name); err != nil {
		return nil, err
	}

	branchSha, err := is.rs.Resolve(name)
	if err != nil {
		return nil, fmt.Errorf("resolve branch: %w", err)
	}

	currentBranch, err := is.rs.Current()
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}

	info := &BranchInfo{
		Name:            name,
		SHA:             branchSha,
		IsCurrentBranch: name == currentBranch,
	}

	if err := is.enrichWithCommitInfo(ctx, info); err != nil {
		return nil, fmt.Errorf("enrich branch info: %w", err)
	}

	return info, nil
}

// ListAll returns information about all branches in the repository.
// It uses concurrent processing via WorkerPool to improve performance when
// there are many branches, as branch resolution involves I/O operations.
func (is *InfoService) ListAll(ctx context.Context) ([]BranchInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	branchNames, err := is.rs.List()
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	currentBranch, err := is.rs.Current()
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}

	// Create worker pool with more workers for I/O-bound operations
	// Using 2x GOMAXPROCS as branch resolution is I/O-bound (filesystem access)
	workerPool := pool.NewWorkerPool[string, BranchInfo](
		pool.WithWorkerCount(runtime.GOMAXPROCS(0) * 2),
	)

	branches, err := workerPool.Process(ctx, branchNames, func(ctx context.Context, name string) (BranchInfo, error) {
		return is.getBranchInfoQuick(name, currentBranch)
	})

	if err != nil {
		return nil, fmt.Errorf("process branches: %w", err)
	}

	return branches, nil
}

// getBranchInfoQuick gets basic branch info without expensive operations
func (is *InfoService) getBranchInfoQuick(name, currentBranch string) (BranchInfo, error) {
	sha, err := is.rs.Resolve(name)
	if err != nil {
		return BranchInfo{}, err
	}

	return BranchInfo{
		Name:            name,
		SHA:             sha,
		IsCurrentBranch: name == currentBranch,
	}, nil
}

// enrichWithCommitInfo adds commit-related information to branch info
func (is *InfoService) enrichWithCommitInfo(ctx context.Context, info *BranchInfo) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	commit, err := is.repo.ReadCommitObject(info.SHA)
	if err != nil {
		return fmt.Errorf("read commit: %w", err)
	}

	if commit == nil {
		return nil
	}

	info.LastCommitMessage = commit.Message
	if commit.Author != nil {
		info.LastCommitDate = commit.Author.When.Time()
	}

	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		count, err := is.countCommits(ctx, info.SHA)
		if err == nil {
			info.CommitCount = count
		}
		return err
	})

	return g.Wait()
}

// countCommits counts the number of commits reachable from the given SHA
func (is *InfoService) countCommits(ctx context.Context, startSHA objects.ObjectHash) (int, error) {
	count := 0
	currentSHA := startSHA

	for currentSHA != "" {
		select {
		case <-ctx.Done():
			return count, ctx.Err()
		default:
		}

		commit, err := is.repo.ReadCommitObject(currentSHA)
		if err != nil {
			break
		}

		count++

		parents := commit.ParentSHAs
		if len(parents) == 0 {
			break
		}

		currentSHA = parents[0]
	}

	return count, nil
}

// CompareWithBase compares a branch with a base branch to determine ahead/behind status
func (is *InfoService) CompareWithBase(ctx context.Context, branchName, baseName string) (ahead, behind int, err error) {
	select {
	case <-ctx.Done():
		return 0, 0, ctx.Err()
	default:
	}

	branchSHA, err := is.rs.Resolve(branchName)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve branch: %w", err)
	}

	baseSHA, err := is.rs.Resolve(baseName)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve base: %w", err)
	}

	// If they point to the same commit, they're even
	if branchSHA == baseSHA {
		return 0, 0, nil
	}

	// TODO: Implement proper commit graph traversal to calculate ahead/behind
	// For now, return 0, 0 as this requires merge-base calculation
	// This would involve finding the common ancestor and counting commits
	// from there to each branch tip

	return 0, 0, nil
}
