package merge

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

// ResolutionStrategy defines how to resolve a conflict
type ResolutionStrategy string

const (
	// ResolutionOurs uses our version (HEAD) to resolve conflicts
	ResolutionOurs ResolutionStrategy = "ours"

	// ResolutionTheirs uses their version (merging branch) to resolve conflicts
	ResolutionTheirs ResolutionStrategy = "theirs"

	// ResolutionUnion combines both versions (not recommended for code)
	ResolutionUnion ResolutionStrategy = "union"

	// ResolutionManual requires manual conflict resolution
	ResolutionManual ResolutionStrategy = "manual"
)

// AdvancedConflictResolver handles conflict resolution operations
type AdvancedConflictResolver struct {
	repo  *sourcerepo.SourceRepository
	store store.ObjectStore
	idx   *index.Index
}

// NewAdvancedConflictResolver creates a new conflict resolver
func NewAdvancedConflictResolver(repo *sourcerepo.SourceRepository, store store.ObjectStore, idx *index.Index) *AdvancedConflictResolver {
	return &AdvancedConflictResolver{
		repo:  repo,
		store: store,
		idx:   idx,
	}
}

// ResolveConflict resolves a single conflict using the specified strategy
func (cr *AdvancedConflictResolver) ResolveConflict(ctx context.Context, path scpath.RelativePath, strategy ResolutionStrategy) error {
	conflict, exists := cr.idx.GetConflict(path)
	if !exists {
		return fmt.Errorf("no conflict found for path: %s", path)
	}

	var resolvedHash objects.ObjectHash
	var resolvedContent []byte
	var err error

	switch strategy {
	case ResolutionOurs:
		resolvedHash = conflict.Ours
		resolvedContent, err = cr.getObjectContent(ctx, conflict.Ours)
	case ResolutionTheirs:
		resolvedHash = conflict.Theirs
		resolvedContent, err = cr.getObjectContent(ctx, conflict.Theirs)
	case ResolutionUnion:
		resolvedContent, err = cr.resolveUnion(ctx, conflict)
		if err == nil {
			resolvedHash, err = cr.storeContent(ctx, resolvedContent)
		}
	default:
		return fmt.Errorf("unsupported resolution strategy: %s", strategy)
	}

	if err != nil {
		return fmt.Errorf("failed to resolve conflict: %w", err)
	}

	// Write resolved content to working directory
	workdirPath := filepath.Join(cr.repo.WorkingDirectory().String(), path.String())
	if err := os.WriteFile(workdirPath, resolvedContent, 0644); err != nil {
		return fmt.Errorf("failed to write resolved file: %w", err)
	}

	// Mark conflict as resolved in index
	if err := cr.idx.ResolveConflict(path, resolvedHash); err != nil {
		return fmt.Errorf("failed to mark conflict as resolved: %w", err)
	}

	return nil
}

// ResolveAllConflicts resolves all conflicts using the specified strategy
func (cr *AdvancedConflictResolver) ResolveAllConflicts(ctx context.Context, strategy ResolutionStrategy) error {
	conflicts := cr.idx.GetConflicts()
	if len(conflicts) == 0 {
		return fmt.Errorf("no conflicts to resolve")
	}

	for path := range conflicts {
		if err := cr.ResolveConflict(ctx, path, strategy); err != nil {
			return fmt.Errorf("failed to resolve %s: %w", path, err)
		}
	}

	return nil
}

// resolveUnion creates a union of both versions
func (cr *AdvancedConflictResolver) resolveUnion(ctx context.Context, conflict *index.ConflictEntry) ([]byte, error) {
	oursContent, err := cr.getObjectContent(ctx, conflict.Ours)
	if err != nil {
		return nil, fmt.Errorf("failed to get ours content: %w", err)
	}

	theirsContent, err := cr.getObjectContent(ctx, conflict.Theirs)
	if err != nil {
		return nil, fmt.Errorf("failed to get theirs content: %w", err)
	}

	// Simple union: concatenate both versions
	var result bytes.Buffer
	result.Write(oursContent)
	if !bytes.HasSuffix(oursContent, []byte("\n")) {
		result.WriteString("\n")
	}
	result.Write(theirsContent)

	return result.Bytes(), nil
}

// getObjectContent retrieves the content of an object
func (cr *AdvancedConflictResolver) getObjectContent(ctx context.Context, hash objects.ObjectHash) ([]byte, error) {
	if hash.IsZero() {
		return []byte{}, nil
	}

	// TODO: Implement object content retrieval
	// This requires accessing the object store
	return nil, fmt.Errorf("object content retrieval not implemented")
}

// storeContent stores content as a blob and returns its hash
func (cr *AdvancedConflictResolver) storeContent(ctx context.Context, content []byte) (objects.ObjectHash, error) {
	// TODO: Implement content storage
	// This requires accessing the object store
	return objects.ZeroHash(), fmt.Errorf("content storage not implemented")
}

// CheckoutConflictVersion checks out a specific version for a conflicted file
func (cr *AdvancedConflictResolver) CheckoutConflictVersion(ctx context.Context, path scpath.RelativePath, version string) error {
	conflict, exists := cr.idx.GetConflict(path)
	if !exists {
		return fmt.Errorf("no conflict found for path: %s", path)
	}

	var hash objects.ObjectHash
	switch version {
	case "ours", "2":
		hash = conflict.Ours
	case "theirs", "3":
		hash = conflict.Theirs
	case "base", "1":
		hash = conflict.Base
	default:
		return fmt.Errorf("invalid version: %s (must be 'ours', 'theirs', or 'base')", version)
	}

	if hash.IsZero() {
		return fmt.Errorf("version %s does not exist for this conflict", version)
	}

	content, err := cr.getObjectContent(ctx, hash)
	if err != nil {
		return fmt.Errorf("failed to get content: %w", err)
	}

	// Write to working directory
	workdirPath := filepath.Join(cr.repo.WorkingDirectory().String(), path.String())
	if err := os.WriteFile(workdirPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Diff3Merge performs a three-way merge using the diff3 algorithm
func (cr *AdvancedConflictResolver) Diff3Merge(ctx context.Context, path scpath.RelativePath, diff3Style bool) error {
	conflict, exists := cr.idx.GetConflict(path)
	if !exists {
		return fmt.Errorf("no conflict found for path: %s", path)
	}

	baseContent, err := cr.getObjectContent(ctx, conflict.Base)
	if err != nil {
		return fmt.Errorf("failed to get base content: %w", err)
	}

	oursContent, err := cr.getObjectContent(ctx, conflict.Ours)
	if err != nil {
		return fmt.Errorf("failed to get ours content: %w", err)
	}

	theirsContent, err := cr.getObjectContent(ctx, conflict.Theirs)
	if err != nil {
		return fmt.Errorf("failed to get theirs content: %w", err)
	}

	// Create conflict markers
	conflicted := CreateConflictMarkers(baseContent, oursContent, theirsContent, "HEAD", "MERGE_HEAD", diff3Style)

	// Write to working directory
	workdirPath := filepath.Join(cr.repo.WorkingDirectory().String(), path.String())
	if err := os.WriteFile(workdirPath, conflicted, 0644); err != nil {
		return fmt.Errorf("failed to write conflicted file: %w", err)
	}

	return nil
}

// GetConflictStatus returns information about the current conflict state
func (cr *AdvancedConflictResolver) GetConflictStatus() *ConflictStatus {
	conflicts := cr.idx.GetConflicts()
	paths := make([]scpath.RelativePath, 0, len(conflicts))
	for path := range conflicts {
		paths = append(paths, path)
	}

	return &ConflictStatus{
		HasConflicts:  len(conflicts) > 0,
		ConflictCount: len(conflicts),
		ConflictPaths: paths,
	}
}

// ConflictStatus represents the current conflict state
type ConflictStatus struct {
	HasConflicts  bool
	ConflictCount int
	ConflictPaths []scpath.RelativePath
}
