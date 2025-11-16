package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/cmd/ui"
	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/blob"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/objects/tree"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

func newShowCmd() *cobra.Command {
	var format string
	var showPatch bool

	cmd := &cobra.Command{
		Use:   "show [object]",
		Short: "Show detailed information about an object",
		Long: `Display detailed information about Git objects (commits, trees, blobs).

For commits:
  - Shows commit metadata (hash, author, committer, date)
  - Displays the commit message
  - Optionally shows the diff/patch (with --patch flag)

For trees:
  - Lists all entries in the tree
  - Shows file modes, types, and hashes

For blobs:
  - Displays the blob content
  - Shows blob size and hash

If no object is specified, shows the current HEAD commit.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Determine which object to show
			var objectRef string
			if len(args) > 0 {
				objectRef = args[0]
			} else {
				objectRef = "HEAD"
			}

			// Resolve the object reference to a hash
			hash, err := resolveObjectRef(ctx, repo, objectRef)
			if err != nil {
				return fmt.Errorf("failed to resolve object reference '%s': %w", objectRef, err)
			}

			// Read the object from the object store
			objStore := store.NewFileObjectStore()
			if err := objStore.Initialize(repo.WorkingDirectory()); err != nil {
				return fmt.Errorf("failed to initialize object store: %w", err)
			}

			obj, err := objStore.ReadObject(hash)
			if err != nil {
				return fmt.Errorf("failed to read object: %w", err)
			}

			if obj == nil {
				return fmt.Errorf("object not found: %s", hash)
			}

			// Display the object based on its type
			switch obj.Type() {
			case objects.CommitType:
				return showCommit(ctx, repo, obj.(*commit.Commit), showPatch)
			case objects.TreeType:
				return showTree(obj.(*tree.Tree))
			case objects.BlobType:
				return showBlob(obj.(*blob.Blob))
			default:
				return fmt.Errorf("unsupported object type: %s", obj.Type())
			}
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "full", "Output format (full, short)")
	cmd.Flags().BoolVarP(&showPatch, "patch", "p", false, "Show diff/patch for commits")

	return cmd
}

// resolveObjectRef resolves an object reference (like "HEAD", commit hash, etc.) to an ObjectHash
func resolveObjectRef(ctx context.Context, repo *sourcerepo.SourceRepository, ref string) (objects.ObjectHash, error) {
	// If it's already a valid hash, return it
	if hash, err := objects.ParseObjectHash(ref); err == nil {
		return hash, nil
	}

	// If it's HEAD or a branch name, resolve it through the commit manager
	commitMgr := commitmanager.NewManager(repo)
	if err := commitMgr.Initialize(ctx); err != nil {
		return "", fmt.Errorf("failed to initialize commit manager: %w", err)
	}

	// Try to get HEAD commit
	if ref == "HEAD" || ref == "" {
		history, err := commitMgr.GetHistory(ctx, objects.ObjectHash(""), 1)
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD commit: %w", err)
		}
		if len(history) == 0 {
			return "", fmt.Errorf("no commits yet")
		}
		return history[0].Hash()
	}

	// Try to parse as a short hash
	if len(ref) >= 7 && len(ref) < 40 {
		// This is a short hash - we'll need to search for it
		// For now, pad it to 40 characters (this is a simplification)
		// A proper implementation would search the object database
		return "", fmt.Errorf("short hash resolution not yet implemented: %s", ref)
	}

	return "", fmt.Errorf("could not resolve reference: %s", ref)
}

// showCommit displays detailed information about a commit
func showCommit(ctx context.Context, repo *sourcerepo.SourceRepository, c *commit.Commit, showPatch bool) error {
	commitHash, _ := c.Hash()

	// Print header
	fmt.Println(ui.Header(" Commit Details "))
	fmt.Println()

	// Commit hash
	fmt.Printf("%s %s\n", ui.Yellow("commit"), ui.Yellow(commitHash.String()))

	// Check if it's a merge commit
	if c.IsMergeCommit() {
		fmt.Printf("%s ", ui.Cyan("Merge:"))
		for i, parent := range c.ParentSHAs {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Print(ui.Yellow(string(parent.Short())))
		}
		fmt.Println()
	}

	// Author information
	fmt.Printf("%s %s <%s>\n",
		ui.Cyan("Author:"),
		ui.Blue(c.Author.Name),
		ui.Blue(c.Author.Email))
	fmt.Printf("%s %s\n",
		ui.Cyan("Date:  "),
		ui.Magenta(c.Author.When.Time().Format(time.RFC1123)))

	// Tree hash
	fmt.Printf("%s %s\n", ui.Cyan("Tree:  "), ui.Yellow(string(c.TreeSHA.Short())))

	// Parent commits
	if len(c.ParentSHAs) > 0 && !c.IsMergeCommit() {
		for _, parent := range c.ParentSHAs {
			fmt.Printf("%s %s\n", ui.Cyan("Parent:"), ui.Yellow(string(parent.Short())))
		}
	}

	// Commit message
	fmt.Println()
	messageLines := strings.Split(strings.TrimSpace(c.Message), "\n")
	for _, line := range messageLines {
		fmt.Printf("    %s\n", line)
	}
	fmt.Println()

	// Show patch if requested
	if showPatch {
		fmt.Println(ui.Header(" Changes "))
		fmt.Println()
		if err := showCommitDiff(ctx, repo, c); err != nil {
			return fmt.Errorf("failed to show diff: %w", err)
		}
	}

	return nil
}

// showCommitDiff shows the diff for a commit
func showCommitDiff(ctx context.Context, repo *sourcerepo.SourceRepository, c *commit.Commit) error {
	// Get the object store
	objStore := store.NewFileObjectStore()
	if err := objStore.Initialize(repo.WorkingDirectory()); err != nil {
		return fmt.Errorf("failed to initialize object store: %w", err)
	}

	// Get the current tree
	currentTree, err := loadTree(objStore, c.TreeSHA)
	if err != nil {
		return fmt.Errorf("failed to load tree: %w", err)
	}

	// If there's a parent, compare with parent tree
	if len(c.ParentSHAs) > 0 {
		// Load parent commit
		parentObj, err := objStore.ReadObject(c.ParentSHAs[0])
		if err != nil {
			return fmt.Errorf("failed to read parent commit: %w", err)
		}
		parentCommit := parentObj.(*commit.Commit)

		// Load parent tree
		parentTree, err := loadTree(objStore, parentCommit.TreeSHA)
		if err != nil {
			return fmt.Errorf("failed to load parent tree: %w", err)
		}

		// Compare trees and show differences
		return compareTrees(objStore, parentTree, currentTree, "")
	} else {
		// Initial commit - show all files as new
		fmt.Println(ui.Green("Initial commit - all files are new:"))
		fmt.Println()
		return showTreeContents(objStore, currentTree, "", true)
	}
}

// loadTree loads a tree object from the object store
func loadTree(objStore *store.FileObjectStore, hash objects.ObjectHash) (*tree.Tree, error) {
	obj, err := objStore.ReadObject(hash)
	if err != nil {
		return nil, err
	}
	if obj.Type() != objects.TreeType {
		return nil, fmt.Errorf("expected tree object, got %s", obj.Type())
	}
	return obj.(*tree.Tree), nil
}

// compareTrees compares two trees and shows the differences
func compareTrees(objStore *store.FileObjectStore, oldTree, newTree *tree.Tree, prefix string) error {
	oldEntries := oldTree.Entries()
	newEntries := newTree.Entries()

	// Create maps for easier comparison
	oldMap := make(map[string]*tree.TreeEntry)
	newMap := make(map[string]*tree.TreeEntry)

	for _, entry := range oldEntries {
		oldMap[entry.Name().String()] = entry
	}

	for _, entry := range newEntries {
		newMap[entry.Name().String()] = entry
	}

	// Find added and modified files
	for name, newEntry := range newMap {
		oldEntry, existed := oldMap[name]
		path := prefix + name

		if !existed {
			// File was added
			fmt.Printf("%s %s\n", ui.Green("+ add"), ui.Green(path))
		} else if oldEntry.SHA() != newEntry.SHA() {
			// File was modified
			if newEntry.IsDirectory() {
				// Recursively compare subdirectories
				oldSubTree, _ := loadTree(objStore, oldEntry.SHA())
				newSubTree, _ := loadTree(objStore, newEntry.SHA())
				if oldSubTree != nil && newSubTree != nil {
					compareTrees(objStore, oldSubTree, newSubTree, path+"/")
				}
			} else {
				fmt.Printf("%s %s\n", ui.Yellow("~ mod"), ui.Yellow(path))
			}
		}
	}

	// Find deleted files
	for name := range oldMap {
		if _, exists := newMap[name]; !exists {
			path := prefix + name
			fmt.Printf("%s %s\n", ui.Red("- del"), ui.Red(path))
		}
	}

	return nil
}

// showTree displays detailed information about a tree
func showTree(t *tree.Tree) error {
	treeHash, _ := t.Hash()

	fmt.Println(ui.Header(" Tree Details "))
	fmt.Println()

	fmt.Printf("%s %s\n", ui.Yellow("tree"), ui.Yellow(treeHash.String()))
	size, _ := t.Size()
	fmt.Printf("%s %s\n", ui.Cyan("Size:"), ui.Blue(size.String()))
	fmt.Printf("%s %d\n", ui.Cyan("Entries:"), len(t.Entries()))
	fmt.Println()

	// Display entries
	entries := t.Entries()
	if len(entries) == 0 {
		fmt.Println(ui.Yellow("  (empty tree)"))
		return nil
	}

	fmt.Println(ui.Cyan("Contents:"))
	for _, entry := range entries {
		modeStr := entry.Mode().ToOctalString()
		typeStr := getEntryTypeString(entry)

		fmt.Printf("  %s %s %s  %s\n",
			ui.Magenta(modeStr),
			ui.Yellow(typeStr),
			ui.Yellow(string(entry.SHA().Short())),
			ui.Blue(entry.Name().String()))
	}
	fmt.Println()

	return nil
}

// showTreeContents recursively shows tree contents (for initial commits)
func showTreeContents(objStore *store.FileObjectStore, t *tree.Tree, prefix string, showFiles bool) error {
	entries := t.Entries()

	for _, entry := range entries {
		path := prefix + entry.Name().String()

		if entry.IsDirectory() {
			// Load and recursively show subdirectory
			subTree, err := loadTree(objStore, entry.SHA())
			if err == nil {
				showTreeContents(objStore, subTree, path+"/", showFiles)
			}
		} else if showFiles {
			fmt.Printf("%s %s\n", ui.Green("+ add"), ui.Green(path))
		}
	}

	return nil
}

// getEntryTypeString returns a string representation of the entry type
func getEntryTypeString(entry *tree.TreeEntry) string {
	if entry.IsDirectory() {
		return "tree"
	} else if entry.IsFile() {
		return "blob"
	} else if entry.IsSymbolicLink() {
		return "link"
	} else if entry.IsSubmodule() {
		return "commit"
	}
	return "unknown"
}

// showBlob displays detailed information about a blob
func showBlob(b *blob.Blob) error {
	blobHash, _ := b.Hash()

	fmt.Println(ui.Header(" Blob Details "))
	fmt.Println()

	fmt.Printf("%s %s\n", ui.Yellow("blob"), ui.Yellow(blobHash.String()))
	size, _ := b.Size()
	fmt.Printf("%s %s\n", ui.Cyan("Size:"), ui.Blue(size.String()))
	fmt.Println()

	// Get content
	content, err := b.Content()
	if err != nil {
		return fmt.Errorf("failed to get blob content: %w", err)
	}

	// Display content
	fmt.Println(ui.Cyan("Content:"))
	fmt.Println(ui.Header(""))

	contentStr := content.String()

	// Check if content is binary
	if isBinary([]byte(contentStr)) {
		fmt.Println(ui.Yellow("  (binary content, not displayed)"))
		fmt.Printf("  %s %d bytes\n", ui.Cyan("Size:"), len(contentStr))
	} else {
		// Display text content
		lines := strings.Split(contentStr, "\n")
		for i, line := range lines {
			// Limit output for very large files
			if i >= 100 {
				remaining := len(lines) - i
				fmt.Printf("\n%s (%d more lines...)\n", ui.Yellow("..."), remaining)
				break
			}
			fmt.Println(line)
		}
	}

	fmt.Println(ui.Header(""))
	fmt.Println()

	return nil
}

// isBinary checks if content appears to be binary
func isBinary(data []byte) bool {
	// Check first 512 bytes for null bytes
	checkLen := 512
	if len(data) < checkLen {
		checkLen = len(data)
	}

	for i := 0; i < checkLen; i++ {
		if data[i] == 0 {
			return true
		}
	}

	return false
}
