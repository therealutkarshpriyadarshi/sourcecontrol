package commitmanager

import (
	"container/list"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/common/logger"
	"github.com/utkarsh5026/SourceControl/pkg/config"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/refs/branch"
	"github.com/utkarsh5026/SourceControl/pkg/repository/refs"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// Manager handles the creation and management of Git commits.
//
// The commit creation process follows these steps:
//  1. Read the index to get staged changes
//  2. Build a tree object from the index
//  3. Get the current HEAD commit (parent)
//  4. Create a new commit object
//  5. Update the current branch reference
//
// This ensures that commits are properly linked in the Git DAG (Directed Acyclic Graph)
// and that references are updated atomically.
//
// Thread Safety:
// Manager is not thread-safe. External synchronization is required when
// accessing a Manager instance from multiple goroutines.
type Manager struct {
	repo          *sourcerepo.SourceRepository
	treeBuilder   *TreeBuilder
	refManager    *refs.RefManager
	branchManager *branch.BranchRefManager
	configManager *config.Manager
	typedConfig   *config.TypedConfig
	logger        *slog.Logger
}

// NewManager creates a new CommitManager instance
//
// Example:
//
//	repo := sourcerepo.NewSourceRepository()
//	repo.Initialize(scpath.RepositoryPath("/path/to/repo"))
//	mgr := commitmanager.NewManager(repo)
func NewManager(repo *sourcerepo.SourceRepository) *Manager {
	refMgr := refs.NewRefManager(repo)
	branchMgr := branch.NewBranchRefManager(refMgr)
	configMgr := config.NewManager(repo.WorkingDirectory())
	typedConfig := config.NewTypedConfig(configMgr)

	return &Manager{
		repo:          repo,
		treeBuilder:   NewTreeBuilder(repo),
		refManager:    refMgr,
		branchManager: branchMgr,
		configManager: configMgr,
		typedConfig:   typedConfig,
		logger:        logger.With("component", "commitmanager"),
	}
}

// Initialize initializes the commit manager by loading configuration and
// initializing dependent managers.
//
// This should be called once after creating a new Manager instance.
func (m *Manager) Initialize(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.logger.Info("initializing commit manager")

	// Load configuration
	if err := m.configManager.Load(ctx); err != nil {
		m.logger.Error("failed to load config", "error", err)
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize ref manager
	if err := m.refManager.Init(); err != nil {
		m.logger.Error("failed to initialize ref manager", "error", err)
		return fmt.Errorf("init ref manager: %w", err)
	}

	// Initialize branch manager
	if err := m.branchManager.Init(); err != nil {
		m.logger.Error("failed to initialize branch manager", "error", err)
		return fmt.Errorf("init branch manager: %w", err)
	}

	m.logger.Info("commit manager initialized successfully")
	return nil
}

// CreateCommit creates a new commit from the current index
//
// This method performs the complete commit creation workflow:
//  1. Validates the commit options
//  2. Reads the index to get staged changes
//  3. Builds a tree from the index
//  4. Determines parent commits
//  5. Creates the commit object
//  6. Updates the current branch reference
func (m *Manager) CreateCommit(ctx context.Context, options CommitOptions) (*commit.Commit, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if err := options.Validate(); err != nil {
		m.logger.Error("invalid commit options", "error", err)
		return nil, err
	}

	idx, err := m.readIndex(options.AllowEmpty)
	if err != nil {
		return nil, err
	}

	treeSHA, err := m.treeBuilder.BuildFromIndex(ctx, idx)
	if err != nil {
		return nil, NewCommitError("build tree", err, "")
	}

	parentSHAs, err := m.getParentCommits(ctx, options.Amend)
	if err != nil {
		return nil, NewCommitError("get parents", err, "")
	}

	// Check if tree is different from parent (avoid empty commits)
	if !options.AllowEmpty && len(parentSHAs) > 0 {
		parentCommit, err := m.repo.ReadCommitObject(parentSHAs[0])
		if err == nil && parentCommit.TreeSHA == treeSHA {
			return nil, NewCommitError("validate", ErrNoTreeChanges, "")
		}
	}

	commitObj, err := m.createCommit(options, treeSHA, parentSHAs)
	if err != nil {
		return nil, NewCommitError("build commit", err, "")
	}

	commitSHA, err := m.repo.WriteObject(commitObj)
	if err != nil {
		return nil, NewCommitError("write commit", err, "")
	}

	if err := m.updateCurrentRef(ctx, commitSHA); err != nil {
		return nil, NewCommitError("update ref", err, "")
	}

	return commitObj, nil
}

func (m *Manager) readIndex(allowEmpty bool) (*index.Index, error) {
	indexPath := m.repo.SourceDirectory().IndexPath()
	idx, err := index.Read(indexPath.ToAbsolutePath())
	if err != nil {
		m.logger.Error("failed to read index", "error", err, "path", indexPath)
		return nil, NewCommitError("read index", err, "")
	}

	if idx.Count() == 0 && !allowEmpty {
		return nil, NewCommitError("validate", ErrNoChanges, "")
	}

	return idx, nil
}

func (m *Manager) createCommit(options CommitOptions, treeSHA objects.ObjectHash, parentSHAs []objects.ObjectHash) (*commit.Commit, error) {
	var err error

	author := options.Author
	if author == nil {
		author, err = m.getCurrentUser()
		if err != nil {
			return nil, NewCommitError("get user", err, "")
		}
	}

	committer := options.Committer
	if committer == nil {
		committer = author
	}

	commitObj, err := commit.NewCommitBuilder().
		TreeHash(treeSHA).
		ParentHashes(parentSHAs...).
		Author(author).
		Committer(committer).
		Message(options.Message).
		Build()
	if err != nil {
		return nil, NewCommitError("build commit", err, "")
	}

	return commitObj, nil
}

// GetCommit retrieves information about a specific commit
func (m *Manager) GetCommit(ctx context.Context, sha objects.ObjectHash) (*commit.Commit, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	commitObj, err := m.repo.ReadCommitObject(sha)
	if err != nil {
		return nil, NewCommitError("read commit", err, sha.Short().String())
	}

	return commitObj, nil
}

// GetHistory retrieves the commit history starting from a given commit
//
// The history is returned in reverse chronological order (newest first).
// This uses a breadth-first traversal of the commit graph.
//
// Parameters:
//   - ctx: Context for cancellation
//   - startSHA: Starting commit SHA (empty string for HEAD)
//   - limit: Maximum number of commits to retrieve
func (m *Manager) GetHistory(ctx context.Context, startSHA objects.ObjectHash, limit int) ([]*commit.Commit, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	history := make([]*commit.Commit, 0, limit)

	var currentSHA objects.ObjectHash
	if startSHA == "" {
		sha, err := m.branchManager.GetHeadSHA()
		if err != nil {
			return history, nil
		}
		currentSHA = sha
	} else {
		currentSHA = startSHA
	}

	return m.bfsCommitHistory(ctx, currentSHA, limit)
}

func (m *Manager) bfsCommitHistory(ctx context.Context, currentSHA objects.ObjectHash, limit int) ([]*commit.Commit, error) {
	history := make([]*commit.Commit, 0, limit)
	visited := make(map[string]bool)

	queue := list.New()
	queue.PushBack(currentSHA)

	for queue.Len() > 0 && len(history) < limit {
		select {
		case <-ctx.Done():
			return history, ctx.Err()
		default:
		}

		elem := queue.Front()
		sha := queue.Remove(elem).(objects.ObjectHash)

		if visited[sha.String()] {
			continue
		}
		visited[sha.String()] = true

		result, err := m.GetCommit(ctx, sha)
		if err != nil {
			continue
		}

		history = append(history, result)

		for _, parentSHA := range result.ParentSHAs {
			if !visited[parentSHA.String()] {
				queue.PushBack(parentSHA)
			}
		}
	}

	return history, nil
}

// getParentCommits determines the parent commits for a new commit
func (m *Manager) getParentCommits(ctx context.Context, amend bool) ([]objects.ObjectHash, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	headSHA, err := m.branchManager.GetHeadSHA()
	if err != nil {
		return []objects.ObjectHash{}, nil
	}

	if amend {
		headCommit, err := m.repo.ReadCommitObject(headSHA)
		if err == nil {
			return headCommit.ParentSHAs, nil
		}
	}

	return []objects.ObjectHash{headSHA}, nil
}

// getCurrentUser gets the current user information from config or environment
func (m *Manager) getCurrentUser() (*commit.CommitPerson, error) {
	name := m.typedConfig.UserName()
	if name == "" {
		name = os.Getenv("GIT_AUTHOR_NAME")
	}
	if name == "" {
		name = "Unknown User"
	}

	email := m.typedConfig.UserEmail()
	if email == "" {
		email = os.Getenv("GIT_AUTHOR_EMAIL")
	}
	if email == "" {
		email = "unknown@example.com"
	}

	// Get current time
	now := time.Now()

	person, err := commit.NewCommitPerson(name, email, now)
	if err != nil {
		return nil, fmt.Errorf("create commit person: %w", err)
	}

	return person, nil
}

// updateCurrentRef updates the current branch reference or HEAD
func (m *Manager) updateCurrentRef(ctx context.Context, commitSHA objects.ObjectHash) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	currentBranch, err := m.branchManager.Current()
	if err == nil && currentBranch != "" {
		exists, _ := m.branchManager.Exists(currentBranch)
		force := !exists

		if err := m.branchManager.Update(currentBranch, commitSHA, force); err != nil {
			return fmt.Errorf("update branch manager for %s: %w", currentBranch, err)
		}
		return nil
	}

	// No current branch - this might be an initial commit
	// Create the default branch
	defaultBranch := m.typedConfig.DefaultBranch()
	if defaultBranch == "" {
		defaultBranch = branch.DefaultBranch
	}

	if err := m.branchManager.Update(defaultBranch, commitSHA, true); err != nil {
		return fmt.Errorf("update branch manager for %s: %w", defaultBranch, err)
	}

	return m.branchManager.SetHead(defaultBranch)
}
