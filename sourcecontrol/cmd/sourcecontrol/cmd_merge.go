package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/cmd/ui"
	"github.com/utkarsh5026/SourceControl/pkg/merge"
)

func newMergeCmd() *cobra.Command {
	var (
		// Merge strategies
		strategyRecursive bool
		strategyOctopus   bool

		// Merge modes
		noCommit       bool
		squash         bool
		ffOnly         bool
		noFF           bool
		commitMsg      string
		allowUnrelated bool

		// Conflict resolution
		conflictOurs   bool
		conflictTheirs bool

		// Other options
		verboseOutput bool
		abort         bool
		continuemerge bool
	)

	cmd := &cobra.Command{
		Use:   "merge [<branch>...] [flags]",
		Short: "Join two or more development histories together",
		Long: `Incorporates changes from the named commits (since the time their histories
diverged from the current branch) into the current branch.

Merge Strategies:
  --strategy=recursive  Use recursive merge strategy (default)
  --strategy=octopus    Use octopus merge for multiple branches

Merge Modes:
  --ff-only      Only allow fast-forward merges
  --no-ff        Always create a merge commit even if fast-forward is possible
  --squash       Squash all commits into a single commit
  --no-commit    Perform merge but don't create a commit

Examples:
  # Fast-forward merge if possible
  srcc merge feature-branch

  # Always create a merge commit
  srcc merge --no-ff feature-branch

  # Squash merge
  srcc merge --squash feature-branch

  # Merge multiple branches (octopus)
  srcc merge branch1 branch2 branch3

  # Merge with custom message
  srcc merge -m "Merge feature X" feature-branch

  # Only allow fast-forward
  srcc merge --ff-only feature-branch

  # Abort merge in progress
  srcc merge --abort

  # Continue merge after resolving conflicts
  srcc merge --continue`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle special operations
			if abort {
				return performMergeAbort()
			}

			if continuemerge {
				return performMergeContinue()
			}

			// Validate arguments
			if len(args) == 0 {
				return fmt.Errorf("no branch specified for merge")
			}

			// Build merge options
			opts := make([]merge.MergeOption, 0)

			// Set strategy
			if strategyOctopus {
				opts = append(opts, merge.WithStrategy(merge.StrategyOctopus))
			} else if strategyRecursive {
				opts = append(opts, merge.WithStrategy(merge.StrategyRecursive))
			}

			// Set mode
			modeCount := 0
			if noCommit {
				opts = append(opts, merge.WithNoCommit())
				modeCount++
			}
			if squash {
				opts = append(opts, merge.WithSquash())
				modeCount++
			}
			if ffOnly {
				opts = append(opts, merge.WithFastForwardOnly())
				modeCount++
			}
			if noFF {
				opts = append(opts, merge.WithNoFastForward())
				modeCount++
			}

			if modeCount > 1 {
				return fmt.Errorf("only one merge mode can be specified")
			}

			// Set commit message
			if commitMsg != "" {
				opts = append(opts, merge.WithMessage(commitMsg))
			}

			// Set conflict resolution
			if conflictOurs {
				opts = append(opts, merge.WithConflictResolution(merge.ConflictOurs))
			} else if conflictTheirs {
				opts = append(opts, merge.WithConflictResolution(merge.ConflictTheirs))
			}

			// Set other options
			if allowUnrelated {
				opts = append(opts, merge.WithAllowUnrelatedHistories())
			}

			if verboseOutput {
				opts = append(opts, merge.WithVerbose())
			}

			// Perform the merge
			return performMerge(args, opts)
		},
	}

	// Strategy flags
	cmd.Flags().BoolVar(&strategyRecursive, "strategy-recursive", false, "Use recursive merge strategy (default)")
	cmd.Flags().BoolVar(&strategyOctopus, "strategy-octopus", false, "Use octopus merge strategy for multiple branches")

	// Mode flags
	cmd.Flags().BoolVar(&noCommit, "no-commit", false, "Perform merge but don't create a commit")
	cmd.Flags().BoolVar(&squash, "squash", false, "Squash all commits into a single commit")
	cmd.Flags().BoolVar(&ffOnly, "ff-only", false, "Only allow fast-forward merges")
	cmd.Flags().BoolVar(&noFF, "no-ff", false, "Always create a merge commit")
	cmd.Flags().StringVarP(&commitMsg, "message", "m", "", "Commit message for merge commit")
	cmd.Flags().BoolVar(&allowUnrelated, "allow-unrelated-histories", false, "Allow merging unrelated histories")

	// Conflict resolution flags
	cmd.Flags().BoolVar(&conflictOurs, "ours", false, "Use our version in case of conflicts")
	cmd.Flags().BoolVar(&conflictTheirs, "theirs", false, "Use their version in case of conflicts")

	// Other flags
	cmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false, "Verbose output")
	cmd.Flags().BoolVar(&abort, "abort", false, "Abort the current merge")
	cmd.Flags().BoolVar(&continuemerge, "continue", false, "Continue merge after resolving conflicts")

	return cmd
}

// performMerge executes the merge operation
func performMerge(branches []string, opts []merge.MergeOption) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Create merge manager
	mergeMgr := merge.NewManager(repo)
	if err := mergeMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize merge manager: %w", err)
	}

	// Print merge info
	if len(branches) == 1 {
		fmt.Printf("%s Merging %s into current branch...\n",
			ui.Blue(ui.IconBranch),
			ui.Cyan(branches[0]))
	} else {
		fmt.Printf("%s Merging %d branches: %s...\n",
			ui.Blue(ui.IconBranch),
			len(branches),
			ui.Cyan(strings.Join(branches, ", ")))
	}

	// Perform the merge
	result, err := mergeMgr.Merge(ctx, branches, opts...)
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	// Display results
	return displayMergeResult(result, branches)
}

// displayMergeResult displays the merge result to the user
func displayMergeResult(result *merge.MergeResult, branches []string) error {
	if !result.Success {
		// Merge failed due to conflicts
		fmt.Printf("\n%s %s\n\n",
			ui.Red(ui.IconDeleted),
			ui.Red("Merge failed due to conflicts"))

		if len(result.Conflicts) > 0 {
			fmt.Println(ui.Yellow("Conflicts in the following files:"))
			for _, conflict := range result.Conflicts {
				fmt.Printf("  %s %s\n", ui.Red("✗"), conflict)
			}
			fmt.Println()
			fmt.Println("Fix conflicts and then run:")
			fmt.Printf("  %s\n", ui.Cyan("srcc add <file>..."))
			fmt.Printf("  %s\n", ui.Cyan("srcc merge --continue"))
			fmt.Println()
			fmt.Println("Or abort the merge:")
			fmt.Printf("  %s\n", ui.Cyan("srcc merge --abort"))
		}

		return fmt.Errorf("merge conflicts detected")
	}

	// Merge succeeded
	fmt.Println()

	if result.FastForward {
		// Fast-forward merge
		fmt.Printf("%s %s\n",
			ui.Green(ui.IconCheck),
			ui.Green("Fast-forward merge completed"))
		fmt.Printf("  %s %s → %s\n",
			ui.Blue(ui.IconCommit),
			ui.Yellow("HEAD"),
			ui.Yellow(string(result.CommitSHA.Short())))
	} else if result.CommitSHA != "" {
		// Merge commit created
		fmt.Printf("%s %s\n",
			ui.Green(ui.IconCheck),
			ui.Green("Merge completed"))
		fmt.Printf("  %s Merge commit: %s\n",
			ui.Blue(ui.IconCommit),
			ui.Yellow(string(result.CommitSHA.Short())))
	} else {
		// No commit created (--no-commit or --squash)
		fmt.Printf("%s %s\n",
			ui.Green(ui.IconCheck),
			ui.Green("Merge completed (no commit created)"))
		fmt.Println("  Changes staged. Create commit with:")
		fmt.Printf("    %s\n", ui.Cyan("srcc commit"))
	}

	if result.Message != "" {
		fmt.Printf("  %s\n", result.Message)
	}

	// Display statistics if available
	if result.FilesChanged > 0 {
		fmt.Println()
		fmt.Printf("%s Statistics:\n", ui.Blue(ui.IconCommit))
		fmt.Printf("  Files changed: %s\n",
			ui.Cyan(fmt.Sprintf("%d", result.FilesChanged)))

		if result.Insertions > 0 || result.Deletions > 0 {
			fmt.Printf("  Insertions:    %s\n",
				ui.Green(fmt.Sprintf("+%d", result.Insertions)))
			fmt.Printf("  Deletions:     %s\n",
				ui.Red(fmt.Sprintf("-%d", result.Deletions)))
		}
	}

	fmt.Println()
	return nil
}

// performMergeAbort aborts an in-progress merge
func performMergeAbort() error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	ctx := context.Background()

	mergeMgr := merge.NewManager(repo)
	if err := mergeMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize merge manager: %w", err)
	}

	if err := mergeMgr.AbortMerge(ctx); err != nil {
		return fmt.Errorf("failed to abort merge: %w", err)
	}

	fmt.Printf("%s Merge aborted\n", ui.Green(ui.IconCheck))
	return nil
}

// performMergeContinue continues a merge after resolving conflicts
func performMergeContinue() error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	ctx := context.Background()

	mergeMgr := merge.NewManager(repo)
	if err := mergeMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize merge manager: %w", err)
	}

	if err := mergeMgr.ContinueMerge(ctx); err != nil {
		return fmt.Errorf("failed to continue merge: %w", err)
	}

	fmt.Printf("%s Merge completed\n", ui.Green(ui.IconCheck))
	return nil
}
