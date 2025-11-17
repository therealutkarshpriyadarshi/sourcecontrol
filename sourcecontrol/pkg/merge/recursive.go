package merge

import (
	"context"
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// RecursiveMerger implements the recursive merge strategy
// This is Git's default merge strategy that handles complex merge scenarios
// by recursively merging common ancestors when there are multiple merge bases
type RecursiveMerger struct {
	repo           *sourcerepo.SourceRepository
	baseCalculator *MergeBaseCalculator
	threeWay       *ThreeWayMerger
}

// NewRecursiveMerger creates a new recursive merger
func NewRecursiveMerger(repo *sourcerepo.SourceRepository) *RecursiveMerger {
	return &RecursiveMerger{
		repo:           repo,
		baseCalculator: NewMergeBaseCalculator(repo),
		threeWay:       NewThreeWayMerger(repo),
	}
}

// Name returns the name of this strategy
func (rm *RecursiveMerger) Name() string {
	return "recursive"
}

// CanMerge checks if recursive merge can be performed
func (rm *RecursiveMerger) CanMerge(mergeCtx *MergeContext) bool {
	// Recursive merge works for single branch merges
	return len(mergeCtx.TheirCommits) == 1
}

// Merge performs a recursive merge
// The recursive strategy handles cases where there are multiple merge bases
// by creating a virtual merge commit of the merge bases
func (rm *RecursiveMerger) Merge(mergeCtx *MergeContext) (*MergeResult, error) {
	if !rm.CanMerge(mergeCtx) {
		return nil, fmt.Errorf("recursive merge requires exactly one commit to merge")
	}

	ourCommit := mergeCtx.OurCommit
	theirCommit := mergeCtx.TheirCommits[0]

	ourSHA, err := ourCommit.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get our commit hash: %w", err)
	}

	theirSHA, err := theirCommit.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get their commit hash: %w", err)
	}

	// Find merge base(s)
	mergeBases, err := rm.baseCalculator.FindMergeBases(mergeCtx.Ctx, ourSHA, theirSHA)
	if err != nil {
		if !mergeCtx.Config.AllowUnrelatedHistories {
			return nil, fmt.Errorf("no merge base found and unrelated histories not allowed: %w", err)
		}
		// No merge base - unrelated histories
		mergeBases = nil
	}

	var baseCommit = mergeCtx.BaseCommit
	if baseCommit == nil && len(mergeBases) > 0 {
		baseCommit = mergeBases[0]
	}

	// If there are multiple merge bases, we'd normally merge them recursively
	// For this implementation, we'll just use the first one
	if len(mergeBases) > 1 {
		// In a full implementation, we'd create a virtual merge commit
		// by recursively merging the merge bases
		if mergeCtx.Config.Verbose {
			fmt.Printf("Warning: Multiple merge bases found (%d), using first one\n", len(mergeBases))
		}
	}

	// Update merge context with the base commit
	updatedCtx := &MergeContext{
		Ctx:          mergeCtx.Ctx,
		OurCommit:    ourCommit,
		TheirCommits: mergeCtx.TheirCommits,
		BaseCommit:   baseCommit,
		Config:       mergeCtx.Config,
	}

	// Perform three-way merge
	return rm.threeWay.Merge(updatedCtx)
}

// mergeMultipleBases creates a virtual merge commit from multiple merge bases
// This is used when there are criss-cross merges in the history
func (rm *RecursiveMerger) mergeMultipleBases(ctx context.Context, mergeCtx *MergeContext) (*MergeResult, error) {
	// TODO: Implement recursive merging of multiple merge bases
	// This is a complex operation that involves:
	// 1. Taking the first two merge bases
	// 2. Merging them to create a virtual commit
	// 3. If more merge bases exist, merge the virtual commit with the next base
	// 4. Repeat until all merge bases are merged
	// 5. Use the final virtual commit as the merge base

	return nil, fmt.Errorf("merging multiple merge bases not yet implemented")
}
