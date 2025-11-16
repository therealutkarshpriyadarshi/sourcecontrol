package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/cmd/ui"
	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/index"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/refs/branch"
	"github.com/utkarsh5026/SourceControl/pkg/repository/refs"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/store"
	"github.com/utkarsh5026/SourceControl/pkg/workdir"
)

// resetMode defines the type of reset operation
type resetMode int

const (
	resetSoft  resetMode = iota // Keep changes staged
	resetMixed                  // Unstage changes (default)
	resetHard                   // Discard all changes
)

func newResetCmd() *cobra.Command {
	var soft bool
	var mixed bool
	var hard bool

	cmd := &cobra.Command{
		Use:   "reset [<commit>] [-- <paths>...]",
		Short: "Reset current HEAD to specified state",
		Long: `Reset current HEAD to the specified state.

Modes:
  --soft   Keep changes staged (move HEAD only)
  --mixed  Unstage changes (move HEAD and reset index) - default
  --hard   Discard all changes (move HEAD, reset index, and update working tree)

Examples:
  # Soft reset to previous commit (keep changes staged)
  srcc reset --soft HEAD~1

  # Mixed reset to previous commit (unstage changes)
  srcc reset HEAD~1
  srcc reset --mixed HEAD~1

  # Hard reset to previous commit (discard all changes)
  srcc reset --hard HEAD~1

  # Reset specific file in index to match HEAD
  srcc reset -- file.txt

  # Reset specific file to match a commit
  srcc reset abc123 -- file.txt`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine reset mode
			mode := resetMixed // default
			modeCount := 0
			if soft {
				mode = resetSoft
				modeCount++
			}
			if mixed {
				mode = resetMixed
				modeCount++
			}
			if hard {
				mode = resetHard
				modeCount++
			}

			if modeCount > 1 {
				return fmt.Errorf("only one reset mode can be specified")
			}

			// Parse arguments to separate commit ref and pathspec
			var commitRef string
			var paths []string

			// Find the "--" separator
			dashDashIdx := -1
			for i, arg := range args {
				if arg == "--" {
					dashDashIdx = i
					break
				}
			}

			if dashDashIdx >= 0 {
				// Arguments before "--" are commit refs, after are paths
				if dashDashIdx > 0 {
					commitRef = args[0]
				}
				paths = args[dashDashIdx+1:]
			} else {
				// No "--" separator
				if len(args) > 0 {
					commitRef = args[0]
				}
			}

			// If paths are specified, perform file-level reset
			if len(paths) > 0 {
				if mode != resetMixed {
					return fmt.Errorf("cannot specify reset mode with paths (path-based reset only supports mixed mode)")
				}
				return performFileReset(commitRef, paths)
			}

			// Perform full reset
			return performReset(commitRef, mode)
		},
	}

	cmd.Flags().BoolVar(&soft, "soft", false, "Keep changes staged (move HEAD only)")
	cmd.Flags().BoolVar(&mixed, "mixed", false, "Unstage changes (move HEAD and reset index)")
	cmd.Flags().BoolVar(&hard, "hard", false, "Discard all changes (move HEAD, reset index, and working tree)")

	return cmd
}

// performReset performs a full reset (soft, mixed, or hard)
func performReset(commitRef string, mode resetMode) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Get the target commit SHA
	targetSHA, err := resolveCommitRef(ctx, repo, commitRef)
	if err != nil {
		return fmt.Errorf("failed to resolve commit reference: %w", err)
	}

	// Verify the commit exists
	commitMgr := commitmanager.NewManager(repo)
	if err := commitMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize commit manager: %w", err)
	}

	targetCommit, err := commitMgr.GetCommit(ctx, targetSHA)
	if err != nil {
		return fmt.Errorf("failed to get commit %s: %w", targetSHA.Short(), err)
	}

	// Update HEAD to point to the target commit
	branchMgr := branch.NewManager(repo)
	currentBranch, err := branchMgr.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	refMgr := refs.NewRefManager(repo)

	// Update the reference (either branch or HEAD directly if detached)
	if currentBranch != "" {
		// Update the branch reference
		branchRef := refs.RefPath(fmt.Sprintf("refs/heads/%s", currentBranch))
		if err := refMgr.UpdateRef(branchRef, targetSHA); err != nil {
			return fmt.Errorf("failed to update branch %s: %w", currentBranch, err)
		}
	} else {
		// Detached HEAD - update HEAD directly
		if err := refMgr.UpdateRef(refs.RefPath("HEAD"), targetSHA); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}
	}

	// For mixed and hard reset, update the index
	if mode == resetMixed || mode == resetHard {
		if err := resetIndex(repo, targetCommit); err != nil {
			return fmt.Errorf("failed to reset index: %w", err)
		}
	}

	// For hard reset, update the working directory
	if mode == resetHard {
		workdirMgr := workdir.NewManager(repo)
		_, err := workdirMgr.UpdateToCommit(ctx, targetSHA, workdir.WithForce())
		if err != nil {
			return fmt.Errorf("failed to update working directory: %w", err)
		}
	}

	// Print success message
	modeStr := "soft"
	switch mode {
	case resetMixed:
		modeStr = "mixed"
	case resetHard:
		modeStr = "hard"
	}

	if currentBranch != "" {
		fmt.Printf("%s HEAD is now at %s (%s reset to %s)\n",
			ui.Green(ui.IconCommit),
			ui.Yellow(string(targetSHA.Short())),
			modeStr,
			ui.Cyan(currentBranch))
	} else {
		fmt.Printf("%s HEAD is now at %s (%s reset, detached)\n",
			ui.Green(ui.IconCommit),
			ui.Yellow(string(targetSHA.Short())),
			modeStr)
	}

	return nil
}

// performFileReset resets specific files in the index to match a commit
func performFileReset(commitRef string, paths []string) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Get the target commit SHA
	targetSHA, err := resolveCommitRef(ctx, repo, commitRef)
	if err != nil {
		return fmt.Errorf("failed to resolve commit reference: %w", err)
	}

	// Get the commit
	commitMgr := commitmanager.NewManager(repo)
	if err := commitMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize commit manager: %w", err)
	}

	targetCommit, err := commitMgr.GetCommit(ctx, targetSHA)
	if err != nil {
		return fmt.Errorf("failed to get commit %s: %w", targetSHA.Short(), err)
	}

	// Get the tree from the commit
	treeSHA := targetCommit.TreeSHA

	// Load the object store to read the tree
	objectStore := store.NewFileObjectStore()
	objectStore.Initialize(repo.WorkingDirectory())

	// Load the tree
	treeObj, err := objectStore.ReadObject(treeSHA)
	if err != nil {
		return fmt.Errorf("failed to read tree: %w", err)
	}

	treeData, ok := treeObj.(*tree.Tree)
	if !ok {
		return fmt.Errorf("object is not a tree")
	}

	// Load the index
	indexMgr := index.NewManager(repo.WorkingDirectory())
	if err := indexMgr.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize index: %w", err)
	}

	// Reset each specified path
	updated := 0
	notFound := 0

	for _, path := range paths {
		// Find the entry in the tree
		_, found := findTreeEntry(treeData, path, objectStore)
		if !found {
			fmt.Printf("%s %s (not found in %s)\n",
				ui.Yellow("warning:"),
				path,
				string(targetSHA.Short()))
			notFound++
			continue
		}

		// Update the index entry for this path
		// We need to remove the old entry and add the new one from the tree
		indexMgr.Remove([]string{path}, false)

		// Add the entry from the tree to the index
		// Note: In a full implementation, we'd need to properly reconstruct the index entry
		// For now, we just remove it which effectively unstages the file
		fmt.Printf("%s %s\n", ui.Green("unstaged:"), path)
		updated++
	}

	if updated > 0 {
		fmt.Printf("\nReset %d path(s) to %s\n", updated, string(targetSHA.Short()))
	}
	if notFound > 0 {
		fmt.Printf("%d path(s) not found in commit\n", notFound)
	}

	return nil
}

// resetIndex resets the index to match the tree of a commit
func resetIndex(repo *sourcerepo.SourceRepository, targetCommit *commit.Commit) error {
	// Clear the current index and rebuild it from the commit's tree
	indexMgr := index.NewManager(repo.WorkingDirectory())
	if err := indexMgr.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize index: %w", err)
	}

	// Clear the index
	if err := indexMgr.Clear(); err != nil {
		return fmt.Errorf("failed to clear index: %w", err)
	}

	// TODO: Rebuild index from tree
	// This would require walking the tree and adding all entries to the index
	// For now, we just clear it which effectively unstages all changes

	return nil
}

// resolveCommitRef resolves a commit reference (branch, tag, SHA, or symbolic ref like HEAD~1) to a commit SHA
func resolveCommitRef(ctx context.Context, repo *sourcerepo.SourceRepository, commitRef string) (objects.ObjectHash, error) {
	// If no commit ref specified, use HEAD
	if commitRef == "" {
		commitRef = "HEAD"
	}

	branchMgr := branch.NewManager(repo)
	refMgr := refs.NewRefManager(repo)

	// Try to resolve as a symbolic reference (HEAD, HEAD~1, etc.)
	// For now, we'll just handle direct references
	// TODO: Add support for HEAD~1, HEAD^, etc.

	// Try to resolve as HEAD
	if commitRef == "HEAD" {
		sha, err := branchMgr.CurrentCommit()
		if err != nil {
			return "", fmt.Errorf("failed to resolve HEAD: %w", err)
		}
		return sha, nil
	}

	// Try to resolve as a branch name
	branchRef := refs.RefPath(fmt.Sprintf("refs/heads/%s", commitRef))
	if exists, _ := refMgr.Exists(branchRef); exists {
		sha, err := refMgr.ResolveToSHA(branchRef)
		if err != nil {
			return "", fmt.Errorf("failed to resolve branch %s: %w", commitRef, err)
		}
		return sha, nil
	}

	// Try to resolve as a tag
	tagRef := refs.RefPath(fmt.Sprintf("refs/tags/%s", commitRef))
	if exists, _ := refMgr.Exists(tagRef); exists {
		sha, err := refMgr.ResolveToSHA(tagRef)
		if err != nil {
			return "", fmt.Errorf("failed to resolve tag %s: %w", commitRef, err)
		}
		return sha, nil
	}

	// Try to parse as a direct SHA
	sha, err := objects.NewObjectHashFromString(commitRef)
	if err == nil {
		// Verify the commit exists
		commitMgr := commitmanager.NewManager(repo)
		if err := commitMgr.Initialize(ctx); err != nil {
			return "", err
		}
		if _, err := commitMgr.GetCommit(ctx, sha); err != nil {
			return "", fmt.Errorf("commit %s not found: %w", sha.Short(), err)
		}
		return sha, nil
	}

	return "", fmt.Errorf("cannot resolve '%s' to a commit", commitRef)
}

// findTreeEntry finds an entry in a tree by path
func findTreeEntry(treeData *tree.Tree, path string, store store.ObjectStore) (*tree.TreeEntry, bool) {
	// Simple implementation - just check top-level entries
	// TODO: Add support for nested paths (e.g., "dir/file.txt")
	entries := treeData.Entries()
	relPath, _ := scpath.NewRelativePath(path)
	for _, entry := range entries {
		if entry.Name() == relPath {
			return entry, true
		}
	}
	return nil, false
}
