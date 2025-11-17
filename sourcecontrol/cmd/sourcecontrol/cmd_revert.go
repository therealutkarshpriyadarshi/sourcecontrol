package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/cmd/ui"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/revert"
)

func newRevertCmd() *cobra.Command {
	var noCommit bool
	var message string

	cmd := &cobra.Command{
		Use:   "revert <commit> [<commit>...]",
		Short: "Revert commits by creating new commits that undo changes",
		Long: `Revert one or more commits by creating new commits that undo their changes.

The revert command creates new commits that undo the changes made by the specified
commits. This is different from reset, which moves the HEAD pointer backwards.

Reverting is a safe way to undo changes that have already been shared with others,
as it doesn't rewrite history.

Examples:
  # Revert a single commit
  srcc revert abc123

  # Revert a range of commits (exclusive start, inclusive end)
  srcc revert abc123..def456

  # Revert without committing (stage changes only)
  srcc revert --no-commit abc123

  # Revert with a custom message
  srcc revert -m "Reverting changes" abc123

Conflict Handling:
  If reverting a commit would cause conflicts with the current working directory,
  the operation will fail. Make sure your working directory is clean before
  reverting commits.

Limitations:
  - Cannot revert merge commits (commits with multiple parents)
  - Cannot revert the initial commit (no parent to revert to)
  - Cannot revert if working directory has uncommitted changes`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if this is a range revert (contains "..")
			if len(args) == 1 && strings.Contains(args[0], "..") {
				parts := strings.Split(args[0], "..")
				if len(parts) != 2 {
					return fmt.Errorf("invalid range format: %s (expected format: <commit>..<commit>)", args[0])
				}
				return performRangeRevert(parts[0], parts[1], noCommit, message)
			}

			// Single or multiple commit revert
			if len(args) > 1 {
				// Multiple commits - revert them one by one
				return performMultipleRevert(args, noCommit, message)
			}

			// Single commit revert
			return performSingleRevert(args[0], noCommit, message)
		},
	}

	cmd.Flags().BoolVarP(&noCommit, "no-commit", "n", false, "Stage changes but don't create commit")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Custom commit message")

	return cmd
}

// performSingleRevert reverts a single commit
func performSingleRevert(commitRef string, noCommit bool, customMessage string) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve the commit reference
	targetSHA, err := resolveCommitRef(ctx, repo, commitRef)
	if err != nil {
		return fmt.Errorf("failed to resolve commit reference: %w", err)
	}

	// Initialize revert manager
	revertMgr := revert.NewManager(repo)
	if err := revertMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize revert manager: %w", err)
	}

	// Perform the revert
	result, err := revertMgr.Revert(ctx, targetSHA, revert.RevertOptions{
		NoCommit: noCommit,
		Message:  customMessage,
	})
	if err != nil {
		return fmt.Errorf("failed to revert commit: %w", err)
	}

	// Display results
	if noCommit {
		fmt.Printf("%s Changes from commit %s have been staged\n",
			ui.Green(ui.IconCheck),
			ui.Yellow(string(targetSHA.Short())))
		fmt.Printf("  Use 'srcc commit' to create the revert commit\n")
	} else {
		if result.NewCommit != nil {
			fmt.Printf("%s Reverted commit %s\n",
				ui.Green(ui.IconCommit),
				ui.Yellow(string(targetSHA.Short())))

			newCommitHash, _ := result.NewCommit.Hash()
			fmt.Printf("  New commit: %s\n",
				ui.Yellow(newCommitHash.Short().String()))

			// Show the first line of the commit message
			firstLine := result.NewCommit.Message
			if idx := strings.Index(firstLine, "\n"); idx > 0 {
				firstLine = firstLine[:idx]
			}
			fmt.Printf("  Message: %s\n", ui.Cyan(firstLine))
		}
	}

	return nil
}

// performRangeRevert reverts a range of commits
func performRangeRevert(startRef, endRef string, noCommit bool, customMessage string) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve the start and end commit references
	startSHA, err := resolveCommitRef(ctx, repo, startRef)
	if err != nil {
		return fmt.Errorf("failed to resolve start commit reference: %w", err)
	}

	endSHA, err := resolveCommitRef(ctx, repo, endRef)
	if err != nil {
		return fmt.Errorf("failed to resolve end commit reference: %w", err)
	}

	// Initialize revert manager
	revertMgr := revert.NewManager(repo)
	if err := revertMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize revert manager: %w", err)
	}

	// Perform the range revert
	fmt.Printf("%s Reverting commits from %s to %s...\n",
		ui.Blue(ui.IconCommit),
		ui.Yellow(string(startSHA.Short())),
		ui.Yellow(string(endSHA.Short())))

	result, err := revertMgr.RevertRange(ctx, startSHA, endSHA, revert.RevertOptions{
		NoCommit: noCommit,
		Message:  customMessage,
	})
	if err != nil {
		return fmt.Errorf("failed to revert commit range: %w", err)
	}

	// Display results
	fmt.Printf("%s Reverted %d commit(s)\n",
		ui.Green(ui.IconCheck),
		len(result.RevertedCommits))

	if noCommit {
		fmt.Printf("  Changes have been staged\n")
		fmt.Printf("  Use 'srcc commit' to create the revert commit\n")
	} else {
		if result.NewCommit != nil {
			newCommitHash, _ := result.NewCommit.Hash()
			fmt.Printf("  New commit: %s\n",
				ui.Yellow(newCommitHash.Short().String()))
		}
	}

	return nil
}

// performMultipleRevert reverts multiple commits one by one
func performMultipleRevert(commitRefs []string, noCommit bool, customMessage string) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Initialize revert manager
	revertMgr := revert.NewManager(repo)
	if err := revertMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize revert manager: %w", err)
	}

	// Resolve all commit references first
	commitSHAs := make([]objects.ObjectHash, 0, len(commitRefs))
	for _, ref := range commitRefs {
		sha, err := resolveCommitRef(ctx, repo, ref)
		if err != nil {
			return fmt.Errorf("failed to resolve commit reference %s: %w", ref, err)
		}
		commitSHAs = append(commitSHAs, sha)
	}

	fmt.Printf("%s Reverting %d commit(s)...\n", ui.Blue(ui.IconCommit), len(commitSHAs))

	// Revert commits in order
	revertedCount := 0
	for i, sha := range commitSHAs {
		isLast := i == len(commitSHAs)-1
		commitNoCommit := noCommit || !isLast

		result, err := revertMgr.Revert(ctx, sha, revert.RevertOptions{
			NoCommit: commitNoCommit,
			Message:  customMessage,
		})
		if err != nil {
			return fmt.Errorf("failed to revert commit %s: %w", sha.Short(), err)
		}

		revertedCount++
		fmt.Printf("  %s Reverted %s\n",
			ui.Green(ui.IconCheck),
			ui.Yellow(string(sha.Short())))

		if result.NewCommit != nil && isLast {
			newCommitHash, _ := result.NewCommit.Hash()
			fmt.Printf("  New commit: %s\n",
				ui.Yellow(newCommitHash.Short().String()))
		}
	}

	fmt.Printf("%s Successfully reverted %d commit(s)\n",
		ui.Green(ui.IconCommit),
		revertedCount)

	if noCommit {
		fmt.Printf("  Changes have been staged\n")
		fmt.Printf("  Use 'srcc commit' to create the revert commit\n")
	}

	return nil
}
