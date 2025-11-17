package merge

import (
	"context"
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// MergeBaseCalculator finds the common ancestor(s) between commits
type MergeBaseCalculator struct {
	repo       *sourcerepo.SourceRepository
	commitMgr  *commitmanager.Manager
	visitedMap map[string]bool
}

// NewMergeBaseCalculator creates a new merge base calculator
func NewMergeBaseCalculator(repo *sourcerepo.SourceRepository) *MergeBaseCalculator {
	return &MergeBaseCalculator{
		repo:       repo,
		commitMgr:  commitmanager.NewManager(repo),
		visitedMap: make(map[string]bool),
	}
}

// FindMergeBase finds the best common ancestor between two commits
// This implements a simplified version of Git's merge base algorithm
func (mbc *MergeBaseCalculator) FindMergeBase(ctx context.Context, commit1SHA, commit2SHA objects.ObjectHash) (*commit.Commit, error) {
	if err := mbc.commitMgr.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize commit manager: %w", err)
	}

	// Get both commits
	c1, err := mbc.commitMgr.GetCommit(ctx, commit1SHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit %s: %w", commit1SHA.Short(), err)
	}

	c2, err := mbc.commitMgr.GetCommit(ctx, commit2SHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit %s: %w", commit2SHA.Short(), err)
	}

	// Find common ancestors
	ancestors1 := mbc.getAncestors(ctx, c1)
	ancestors2 := mbc.getAncestors(ctx, c2)

	// Find common ancestors
	commonAncestors := make(map[string]bool)
	for sha := range ancestors1 {
		if ancestors2[sha] {
			commonAncestors[sha] = true
		}
	}

	if len(commonAncestors) == 0 {
		return nil, fmt.Errorf("no common ancestor found")
	}

	// Return the first common ancestor (in a real implementation, we'd find the "best" one)
	for sha := range commonAncestors {
		hash, _ := objects.NewObjectHashFromString(sha)
		baseCommit, err := mbc.commitMgr.GetCommit(ctx, hash)
		if err != nil {
			continue
		}
		return baseCommit, nil
	}

	return nil, fmt.Errorf("failed to retrieve common ancestor")
}

// FindMergeBases finds all merge bases between two commits
func (mbc *MergeBaseCalculator) FindMergeBases(ctx context.Context, commit1SHA, commit2SHA objects.ObjectHash) ([]*commit.Commit, error) {
	// For now, just return the single best merge base
	base, err := mbc.FindMergeBase(ctx, commit1SHA, commit2SHA)
	if err != nil {
		return nil, err
	}
	return []*commit.Commit{base}, nil
}

// getAncestors returns all ancestors of a commit
func (mbc *MergeBaseCalculator) getAncestors(ctx context.Context, c *commit.Commit) map[string]bool {
	ancestors := make(map[string]bool)
	mbc.collectAncestors(ctx, c, ancestors)
	return ancestors
}

// collectAncestors recursively collects all ancestors
func (mbc *MergeBaseCalculator) collectAncestors(ctx context.Context, c *commit.Commit, ancestors map[string]bool) {
	commitSHA, err := c.Hash()
	if err != nil {
		return
	}

	shaStr := commitSHA.String()
	if ancestors[shaStr] {
		return // Already visited
	}

	ancestors[shaStr] = true

	// Recursively collect parent ancestors
	for _, parentSHA := range c.ParentSHAs {
		parent, err := mbc.commitMgr.GetCommit(ctx, parentSHA)
		if err != nil {
			continue
		}
		mbc.collectAncestors(ctx, parent, ancestors)
	}
}

// IsAncestor checks if possibleAncestor is an ancestor of commit
func (mbc *MergeBaseCalculator) IsAncestor(ctx context.Context, possibleAncestorSHA, commitSHA objects.ObjectHash) (bool, error) {
	if err := mbc.commitMgr.Initialize(ctx); err != nil {
		return false, fmt.Errorf("failed to initialize commit manager: %w", err)
	}

	c, err := mbc.commitMgr.GetCommit(ctx, commitSHA)
	if err != nil {
		return false, fmt.Errorf("failed to get commit %s: %w", commitSHA.Short(), err)
	}

	ancestors := mbc.getAncestors(ctx, c)
	return ancestors[possibleAncestorSHA.String()], nil
}

// CanFastForward checks if a fast-forward merge is possible from 'from' to 'to'
// Fast-forward is possible if 'from' is an ancestor of 'to'
func (mbc *MergeBaseCalculator) CanFastForward(ctx context.Context, fromSHA, toSHA objects.ObjectHash) (bool, error) {
	// If they're the same commit, we're already up to date
	if fromSHA.Equal(toSHA) {
		return true, nil
	}

	// Check if 'from' is an ancestor of 'to'
	isAncestor, err := mbc.IsAncestor(ctx, fromSHA, toSHA)
	if err != nil {
		return false, err
	}

	return isAncestor, nil
}
