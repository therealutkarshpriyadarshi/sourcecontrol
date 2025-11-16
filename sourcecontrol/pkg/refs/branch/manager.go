package branch

import (
	"context"
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/refs"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/workdir"
)

const (
	// DefaultBranch is the default branch name for new repositories
	DefaultBranch = "master"
)

// Manager handles comprehensive branch operations including creation,
// deletion, renaming, checkout, and branch information retrieval.
//
// It coordinates between multiple subsystems:
//   - RefManager for reference operations
//   - WorkingDirectoryManager for file system updates
//   - Internal services for specialized operations
//
// Thread Safety:
// Manager is not thread-safe. External synchronization is required when
// accessing a Manager instance from multiple goroutines.
type Manager struct {
	repo           *sourcerepo.SourceRepository
	refManager     *refs.RefManager
	branchRefSvc   *BranchRefManager
	branchInfoSvc  *InfoService
	workdirManager *workdir.Manager
}

// NewManager creates a new branch manager instance.
// All dependencies are initialized once and reused for efficiency.
//
// Example:
//
//	repo := sourcerepo.NewSourceRepository()
//	repo.Initialize(scpath.RepositoryPath("/path/to/repo"))
//	mgr := branch.NewManager(repo)
//	if err := mgr.Init(); err != nil {
//	    log.Fatal(err)
//	}
func NewManager(repo *sourcerepo.SourceRepository) *Manager {
	refMgr := refs.NewRefManager(repo)
	branchRefSvc := NewBranchRefManager(refMgr)
	branchInfoSvc := NewInfoService(repo, branchRefSvc)
	workdirMgr := workdir.NewManager(repo)

	return &Manager{
		repo:           repo,
		refManager:     refMgr,
		branchRefSvc:   branchRefSvc,
		branchInfoSvc:  branchInfoSvc,
		workdirManager: workdirMgr,
	}
}

// Init initializes the branch manager by creating necessary directories.
// This should be called once after creating a new Manager instance.
func (m *Manager) Init() error {
	return m.branchRefSvc.Init()
}

// CreateBranch creates a new branch with the given options.
// Context allows for cancellation of long-running operations.
func (m *Manager) CreateBranch(ctx context.Context, name string, opts ...CreateOption) (BranchInfo, error) {
	config := &CreateConfig{}
	for _, opt := range opts {
		opt(config)
	}

	creator := NewCreator(m.repo, m.branchRefSvc, m.branchInfoSvc)
	branchInfo, err := creator.Create(ctx, name, config)
	if err != nil {
		return BranchInfo{}, fmt.Errorf("create branch: %w", err)
	}

	if config.Checkout {
		checkoutConfig := &CheckoutConfig{
			Force:  false,
			Detach: false,
		}
		if err := m.checkout(ctx, name, checkoutConfig); err != nil {
			return *branchInfo, fmt.Errorf("checkout new branch: %w", err)
		}
	}

	return *branchInfo, nil
}

// Checkout switches to a different branch or commit.
// It updates both HEAD and the working directory.
//
// Example:
//
//	err := mgr.Checkout(ctx, "main")
//	err := mgr.Checkout(ctx, "abc123", branch.WithDetach())
func (m *Manager) Checkout(ctx context.Context, target string, opts ...CheckoutOption) error {
	config := &CheckoutConfig{}
	for _, opt := range opts {
		opt(config)
	}

	return m.checkout(ctx, target, config)
}

// checkout is the internal implementation of Checkout
func (m *Manager) checkout(ctx context.Context, target string, config *CheckoutConfig) error {
	creator := NewCreator(m.repo, m.branchRefSvc, m.branchInfoSvc)
	ch := NewCheckout(m.repo, m.branchRefSvc, creator, m.workdirManager)

	if err := ch.Checkout(ctx, target, config); err != nil {
		return fmt.Errorf("checkout %s: %w", target, err)
	}

	return nil
}

// DeleteBranch removes a branch reference.
// Use force=true to delete unmerged branches.
//
// Example:
//
//	err := mgr.DeleteBranch(ctx, "old-feature", false)
//	err := mgr.DeleteBranch(ctx, "experimental", true) // force delete
func (m *Manager) DeleteBranch(ctx context.Context, name string, opts ...DeleteOption) error {
	config := &DeleteConfig{}
	for _, opt := range opts {
		opt(config)
	}

	d := NewDelete(m.branchRefSvc)
	if err := d.Delete(ctx, name, config); err != nil {
		return fmt.Errorf("delete branch %s: %w", name, err)
	}
	return nil
}

// RenameBranch renames a branch from oldName to newName.
// Use force=true to overwrite existing branch with newName.
//
// Example:
//
//	err := mgr.RenameBranch(ctx, "old-name", "new-name", false)
//	err := mgr.RenameBranch(ctx, "temp", "feature", true) // force rename
func (m *Manager) RenameBranch(ctx context.Context, oldName, newName string, opts ...RenameOption) error {
	config := &RenameConfig{}
	for _, opt := range opts {
		opt(config)
	}

	r := NewRenameService(m.branchRefSvc, newName, oldName, config.Force)

	if err := r.Execute(ctx); err != nil {
		return fmt.Errorf("rename branch %s to %s: %w", oldName, newName, err)
	}

	return nil
}

// GetBranch retrieves detailed information about a specific branch.
//
// Example:
//
//	info, err := mgr.GetBranch(ctx, "main")
//	fmt.Printf("%s -> %s\n", info.Name, info.SHA.Short())
func (m *Manager) GetBranch(ctx context.Context, name string) (BranchInfo, error) {
	info, err := m.branchInfoSvc.GetInfo(ctx, name)
	if err != nil {
		return BranchInfo{}, fmt.Errorf("get branch %s: %w", name, err)
	}
	return *info, nil
}

// ListBranches returns information about all branches in the repository.
func (m *Manager) ListBranches(ctx context.Context) ([]BranchInfo, error) {
	branches, err := m.branchInfoSvc.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}
	return branches, nil
}

// CurrentBranch returns the name of the current branch, or empty string if detached.
func (m *Manager) CurrentBranch() (string, error) {
	name, err := m.branchRefSvc.Current()
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}
	return name, nil
}

// IsDetached checks if HEAD is in detached state.
func (m *Manager) IsDetached() (bool, error) {
	detached, err := m.branchRefSvc.IsDetached()
	if err != nil {
		return false, fmt.Errorf("check detached state: %w", err)
	}
	return detached, nil
}

// CurrentCommit returns the SHA of the current commit.
func (m *Manager) CurrentCommit() (objects.ObjectHash, error) {
	hash, err := m.branchRefSvc.GetHeadSHA()
	if err != nil {
		return "", fmt.Errorf("get current commit: %w", err)
	}
	return hash, nil
}

// BranchExists checks if a branch exists.
func (m *Manager) BranchExists(name string) (bool, error) {
	exists, err := m.branchRefSvc.Exists(name)
	if err != nil {
		return false, fmt.Errorf("check branch exists: %w", err)
	}
	return exists, nil
}
