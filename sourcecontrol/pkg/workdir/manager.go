package workdir

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/workdir/internal"
)

// Manager handles updating the working directory when switching between branches or commits.
// It orchestrates file operations, validation, transactions, and index updates.
type Manager struct {
	repo         *sourcerepo.SourceRepository
	fileOps      *internal.FileOps
	treeAnalyzer *internal.Analyzer
	validator    *internal.Validator
	transaction  *internal.Manager
	indexer      *internal.IndexUpdater
	indexPath    scpath.AbsolutePath
	workDir      string
}

// NewManager creates a new working directory manager
func NewManager(repo *sourcerepo.SourceRepository) *Manager {
	workDir := repo.WorkingDirectory().String()
	sourceDir := repo.SourceDirectory()
	indexPath := sourceDir.IndexPath().ToAbsolutePath()

	fileService := internal.NewFileOps(repo)
	treeAnalyzer := internal.NewAnalyzer(repo)
	workDirValidator := internal.NewValidator(repo.WorkingDirectory())
	txnManager := internal.NewManager(fileService, sourceDir)
	indexUpdater := internal.NewUpdater(workDir, indexPath)

	return &Manager{
		repo:         repo,
		fileOps:      fileService,
		treeAnalyzer: treeAnalyzer,
		validator:    workDirValidator,
		transaction:  txnManager,
		indexer:      indexUpdater,
		indexPath:    indexPath,
		workDir:      workDir,
	}
}

// UpdateToCommit updates the working directory to match a specific commit.
// It performs safety checks, analyzes changes, executes operations atomically,
// and updates the index.
func (m *Manager) UpdateToCommit(ctx context.Context, commitSHA objects.ObjectHash, opts ...Option) (UpdateResult, error) {
	config := &updateConfig{}
	for _, opt := range opts {
		opt(config)
	}

	if !config.force {
		if err := m.performSafetyChecks(); err != nil {
			return UpdateResult{
				Success: false,
				Err:     err,
			}, err
		}
	}

	analysis, err := m.analyzeChanges(ctx, commitSHA)
	if err != nil {
		return UpdateResult{
			Success: false,
			Err:     fmt.Errorf("analyze changes: %w", err),
		}, err
	}

	if len(analysis.Operations) == 0 {
		return UpdateResult{
			Success:      true,
			FilesChanged: 0,
			Operations:   []Operation{},
		}, nil
	}

	if config.dryRun {
		return m.performDryRun(analysis.Operations), nil
	}

	txnResult := m.transaction.ExecuteAtomically(ctx, analysis.Operations)
	if !txnResult.Success {
		return UpdateResult{
			Success:      false,
			FilesChanged: txnResult.OperationsApplied,
			Operations:   analysis.Operations,
			Err:          txnResult.Err,
		}, txnResult.Err
	}

	internalResult, err := m.indexer.UpdateToMatch(analysis.TargetFiles)
	if err != nil || !internalResult.Success {
		indexResult := internalResult
		return UpdateResult{
			Success:      true,
			FilesChanged: txnResult.OperationsApplied,
			Operations:   analysis.Operations,
			IndexUpdate:  &indexResult,
			Err:          nil, // Success despite index issue
		}, nil
	}

	indexResult := internalResult
	return UpdateResult{
		Success:      true,
		FilesChanged: txnResult.OperationsApplied,
		Operations:   analysis.Operations,
		IndexUpdate:  &indexResult,
	}, nil
}

// IsClean checks if the working directory has uncommitted changes
func (m *Manager) IsClean() (Status, error) {
	idx, err := index.Read(m.indexPath)
	if err != nil {
		return Status{}, NewIndexError("read", m.indexPath.String(), err)
	}

	internalStatus, err := m.validator.ValidateCleanState(idx)
	if err != nil {
		return Status{}, err
	}
	return internalStatus, nil
}

// performSafetyChecks verifies the working directory is clean before making changes
func (m *Manager) performSafetyChecks() error {
	status, err := m.IsClean()
	if err != nil {
		return fmt.Errorf("check working directory: %w", err)
	}

	if !status.Clean {
		return NewValidationError(
			"error: Your local changes to the following files would be overwritten by checkout",
			status.ModifiedFiles,
			status.DeletedFiles,
		)
	}

	return nil
}

// analyzeChanges determines what operations are needed to reach the target commit.
// It fetches commit files and reads the index concurrently for better performance.
func (m *Manager) analyzeChanges(ctx context.Context, commitSHA objects.ObjectHash) (ChangeAnalysis, error) {
	var change ChangeAnalysis
	var targetFiles map[scpath.RelativePath]internal.FileInfo
	var idx *index.Index

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		files, err := m.treeAnalyzer.GetCommitFiles(ctx, commitSHA)
		if err != nil {
			return fmt.Errorf("get commit files: %w", err)
		}
		targetFiles = files
		return nil
	})

	g.Go(func() error {
		indexData, err := index.Read(m.indexPath)
		if err != nil {
			return fmt.Errorf("read index: %w", err)
		}
		idx = indexData
		return nil
	})

	if err := g.Wait(); err != nil {
		return change, err
	}

	currentFiles := m.treeAnalyzer.GetIndexFiles(idx)
	return m.treeAnalyzer.AnalyzeChanges(currentFiles, targetFiles), nil
}

// performDryRun analyzes what would change without making actual modifications
func (m *Manager) performDryRun(ops []internal.Operation) UpdateResult {
	dryRunResult := m.transaction.DryRun(ops)

	return UpdateResult{
		Success:      dryRunResult.Valid,
		FilesChanged: 0,
		Operations:   ops,
		Err:          nil,
	}
}

// updateConfig holds configuration for update operations
type updateConfig struct {
	force      bool
	dryRun     bool
	onProgress func(completed, total int, currentFile string)
}

type Option func(*updateConfig)

// WithForce bypasses safety checks for uncommitted changes
func WithForce() Option {
	return func(c *updateConfig) {
		c.force = true
	}
}

// WithDryRun analyzes what would change without making modifications
func WithDryRun() Option {
	return func(c *updateConfig) {
		c.dryRun = true
	}
}

// WithProgress sets a progress callback
func WithProgress(fn func(completed, total int, currentFile string)) Option {
	return func(c *updateConfig) {
		c.onProgress = fn
	}
}
