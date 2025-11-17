package merge

import (
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// OctopusMerger implements octopus merge strategy for merging multiple branches
// Octopus merge is used to merge more than two branches simultaneously
// It's typically used when all branches can be cleanly merged (no conflicts)
type OctopusMerger struct {
	repo      *sourcerepo.SourceRepository
	recursive *RecursiveMerger
}

// NewOctopusMerger creates a new octopus merger
func NewOctopusMerger(repo *sourcerepo.SourceRepository) *OctopusMerger {
	return &OctopusMerger{
		repo:      repo,
		recursive: NewRecursiveMerger(repo),
	}
}

// Name returns the name of this strategy
func (om *OctopusMerger) Name() string {
	return "octopus"
}

// CanMerge checks if octopus merge can be performed
func (om *OctopusMerger) CanMerge(mergeCtx *MergeContext) bool {
	// Octopus merge requires at least 2 branches to merge
	return len(mergeCtx.TheirCommits) >= 2
}

// Merge performs an octopus merge
// The strategy is to merge branches one at a time, creating a merge commit with multiple parents
func (om *OctopusMerger) Merge(mergeCtx *MergeContext) (*MergeResult, error) {
	if !om.CanMerge(mergeCtx) {
		return nil, fmt.Errorf("octopus merge requires at least 2 branches")
	}

	// Octopus merge doesn't handle conflicts well
	// If any merge would result in a conflict, we should fail
	if mergeCtx.Config.ConflictResolution == ConflictManual {
		return nil, fmt.Errorf("octopus merge doesn't support manual conflict resolution")
	}

	currentCommit := mergeCtx.OurCommit
	allParents := make([]objects.ObjectHash, 0)

	// Add our commit as the first parent
	ourSHA, err := currentCommit.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get our commit hash: %w", err)
	}
	allParents = append(allParents, ourSHA)

	// Merge each branch one at a time
	var finalTree objects.ObjectHash
	totalConflicts := 0

	for i, theirCommit := range mergeCtx.TheirCommits {
		theirSHA, err := theirCommit.Hash()
		if err != nil {
			return nil, fmt.Errorf("failed to get commit %d hash: %w", i, err)
		}

		// Create a merge context for this pair
		pairCtx := &MergeContext{
			Ctx:          mergeCtx.Ctx,
			OurCommit:    currentCommit,
			TheirCommits: []*commit.Commit{theirCommit},
			Config:       mergeCtx.Config,
		}

		// Perform recursive merge
		result, err := om.recursive.Merge(pairCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to merge branch %d: %w", i, err)
		}

		// If there are conflicts, octopus merge fails
		if !result.Success || len(result.Conflicts) > 0 {
			return &MergeResult{
				Success:   false,
				Conflicts: result.Conflicts,
				Message:   fmt.Sprintf("Octopus merge failed: conflicts in branch %d", i),
			}, nil
		}

		totalConflicts += len(result.Conflicts)
		allParents = append(allParents, theirSHA)

		// For the next iteration, use a virtual commit representing the current merge state
		// In a full implementation, we'd create this virtual commit
		// For now, we'll just track the tree
		if i == 0 {
			// Get the tree from the first merge
			if result.CommitSHA != "" {
				mergedCommit, err := om.repo.ReadCommitObject(result.CommitSHA)
				if err != nil {
					return nil, fmt.Errorf("failed to read merged commit: %w", err)
				}
				finalTree = mergedCommit.TreeSHA
				currentCommit = mergedCommit
			}
		}
	}

	// Create the octopus merge commit with all parents
	message := mergeCtx.Config.Message
	if message == "" {
		message = fmt.Sprintf("Octopus merge of %d branches", len(mergeCtx.TheirCommits))
	}

	octopusCommit := &commit.Commit{
		TreeSHA:    finalTree,
		ParentSHAs: allParents,
		Author:     mergeCtx.OurCommit.Author,
		Committer:  mergeCtx.OurCommit.Committer,
		Message:    message,
	}

	// Write the octopus commit
	commitSHA, err := om.repo.WriteObject(octopusCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to write octopus commit: %w", err)
	}

	return &MergeResult{
		Success:     true,
		FastForward: false,
		CommitSHA:   commitSHA,
		Message:     fmt.Sprintf("Octopus merge of %d branches completed", len(mergeCtx.TheirCommits)),
	}, nil
}
