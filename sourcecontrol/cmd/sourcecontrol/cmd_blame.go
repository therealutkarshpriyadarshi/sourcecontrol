package main

import (
	"context"
	"fmt"
	"path/filepath"
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

// BlameLineInfo contains the blame information for a single line
type BlameLineInfo struct {
	CommitHash  objects.ObjectHash
	Author      string
	AuthorEmail string
	Date        time.Time
	LineNumber  int
	Content     string
	ShortHash   string
}

func newBlameCmd() *cobra.Command {
	var ignoreWhitespace bool
	var followRenames bool

	cmd := &cobra.Command{
		Use:   "blame <file>",
		Short: "Show what revision and author last modified each line of a file",
		Long: `Show what revision and author last modified each line of a file.

For each line in the specified file, blame shows:
  - The commit hash that last modified the line
  - The author who made the change
  - The date of the commit
  - The line number
  - The line content

This is useful for understanding the history and evolution of a file.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return fmt.Errorf("not in a repository: %w", err)
			}

			filePath := args[0]
			ctx := context.Background()

			// Run blame
			blameInfo, err := runBlame(ctx, repo, filePath, ignoreWhitespace, followRenames)
			if err != nil {
				return fmt.Errorf("blame failed: %w", err)
			}

			// Display results
			displayBlame(blameInfo, filePath)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&ignoreWhitespace, "ignore-whitespace", "w", false, "Ignore whitespace changes")
	cmd.Flags().BoolVarP(&followRenames, "follow", "f", false, "Follow file renames (not yet implemented)")

	return cmd
}

func newAnnotateCmd() *cobra.Command {
	// annotate is an alias for blame
	blameCmd := newBlameCmd()
	blameCmd.Use = "annotate <file>"
	blameCmd.Short = "Annotate file lines with commit information (alias for blame)"
	return blameCmd
}

// runBlame performs the blame operation on a file
func runBlame(ctx context.Context, repo *sourcerepo.SourceRepository, filePath string, ignoreWhitespace bool, followRenames bool) ([]BlameLineInfo, error) {
	// Initialize managers
	commitMgr := commitmanager.NewManager(repo)
	if err := commitMgr.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize commit manager: %w", err)
	}

	objStore := store.NewFileObjectStore()
	if err := objStore.Initialize(repo.WorkingDirectory()); err != nil {
		return nil, fmt.Errorf("failed to initialize object store: %w", err)
	}

	// Get commit history
	history, err := commitMgr.GetHistory(ctx, objects.ObjectHash(""), 100000)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("no commits found")
	}

	// Normalize file path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %w", err)
	}

	repoPath := string(repo.WorkingDirectory())
	relPath, err := filepath.Rel(repoPath, absPath)
	if err != nil {
		return nil, fmt.Errorf("file is not in repository: %w", err)
	}

	// Get current file content from HEAD
	currentContent, err := getFileContentFromCommit(objStore, history[0], relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get current file content: %w", err)
	}

	if currentContent == nil {
		return nil, fmt.Errorf("file not found in repository: %s", filePath)
	}

	// Parse file into lines
	lines := splitLines(currentContent)
	if len(lines) == 0 {
		return []BlameLineInfo{}, nil
	}

	// Initialize blame info for each line
	blameInfo := make([]BlameLineInfo, len(lines))
	for i := range blameInfo {
		blameInfo[i] = BlameLineInfo{
			LineNumber: i + 1,
			Content:    lines[i],
		}
	}

	// Walk through history from oldest to newest to build blame information
	// We reverse the history so we process commits chronologically
	for i := len(history) - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		currentCommit := history[i]
		commitHash, _ := currentCommit.Hash()

		// Get file content in this commit
		currentFileContent, err := getFileContentFromCommit(objStore, currentCommit, relPath)
		if err != nil || currentFileContent == nil {
			// File doesn't exist in this commit, skip
			continue
		}

		// Get parent commit (if any)
		var parentFileContent []byte
		if len(currentCommit.ParentSHAs) > 0 {
			parentCommit, err := repo.ReadCommitObject(currentCommit.ParentSHAs[0])
			if err == nil {
				parentFileContent, _ = getFileContentFromCommit(objStore, parentCommit, relPath)
			}
		}

		// Compare current and parent to find changed lines
		currentLines := splitLines(currentFileContent)
		parentLines := splitLines(parentFileContent)

		// Use a simple diff algorithm to find changed lines
		changedLines := findChangedLines(parentLines, currentLines, ignoreWhitespace)

		// Update blame info for changed lines
		for _, lineNum := range changedLines {
			if lineNum > 0 && lineNum <= len(lines) {
				// Check if this line matches the current version
				if lineNum <= len(currentLines) &&
				   normalizeForComparison(lines[lineNum-1], ignoreWhitespace) ==
				   normalizeForComparison(currentLines[lineNum-1], ignoreWhitespace) {
					blameInfo[lineNum-1].CommitHash = commitHash
					blameInfo[lineNum-1].Author = currentCommit.Author.Name
					blameInfo[lineNum-1].AuthorEmail = currentCommit.Author.Email
					blameInfo[lineNum-1].Date = currentCommit.Author.When.Time()
					blameInfo[lineNum-1].ShortHash = commitHash.Short().String()
				}
			}
		}
	}

	// Handle any lines that don't have blame info (shouldn't happen normally)
	// These get attributed to the first commit where the file appeared
	for i := range blameInfo {
		if blameInfo[i].CommitHash == "" {
			// Find the first commit where this file exists
			for j := len(history) - 1; j >= 0; j-- {
				c := history[j]
				content, err := getFileContentFromCommit(objStore, c, relPath)
				if err == nil && content != nil {
					hash, _ := c.Hash()
					blameInfo[i].CommitHash = hash
					blameInfo[i].Author = c.Author.Name
					blameInfo[i].AuthorEmail = c.Author.Email
					blameInfo[i].Date = c.Author.When.Time()
					blameInfo[i].ShortHash = hash.Short().String()
					break
				}
			}
		}
	}

	return blameInfo, nil
}

// getFileContentFromCommit retrieves the content of a file from a commit
func getFileContentFromCommit(objStore *store.FileObjectStore, c *commit.Commit, filePath string) ([]byte, error) {
	// Load the tree
	treeObj, err := objStore.ReadObject(c.TreeSHA)
	if err != nil {
		return nil, err
	}

	t, ok := treeObj.(*tree.Tree)
	if !ok {
		return nil, fmt.Errorf("expected tree object")
	}

	// Navigate to the file
	pathParts := strings.Split(filepath.ToSlash(filePath), "/")
	return findFileInTree(objStore, t, pathParts)
}

// findFileInTree recursively searches for a file in a tree
func findFileInTree(objStore *store.FileObjectStore, t *tree.Tree, pathParts []string) ([]byte, error) {
	if len(pathParts) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	entries := t.Entries()
	currentPart := pathParts[0]

	for _, entry := range entries {
		if entry.Name().String() == currentPart {
			if len(pathParts) == 1 {
				// Found the file
				if entry.IsFile() {
					blobObj, err := objStore.ReadObject(entry.SHA())
					if err != nil {
						return nil, err
					}
					b, ok := blobObj.(*blob.Blob)
					if !ok {
						return nil, fmt.Errorf("expected blob object")
					}
					content, err := b.Content()
					if err != nil {
						return nil, err
					}
					return []byte(content.String()), nil
				}
				return nil, fmt.Errorf("path is a directory, not a file")
			} else {
				// Navigate into subdirectory
				if entry.IsDirectory() {
					subTreeObj, err := objStore.ReadObject(entry.SHA())
					if err != nil {
						return nil, err
					}
					subTree, ok := subTreeObj.(*tree.Tree)
					if !ok {
						return nil, fmt.Errorf("expected tree object")
					}
					return findFileInTree(objStore, subTree, pathParts[1:])
				}
				return nil, fmt.Errorf("path component is a file, not a directory")
			}
		}
	}

	return nil, fmt.Errorf("file not found in tree")
}

// splitLines splits content into lines, preserving empty lines
func splitLines(content []byte) []string {
	if len(content) == 0 {
		return []string{}
	}

	// Split by newline, handling both \n and \r\n
	contentStr := string(content)
	contentStr = strings.ReplaceAll(contentStr, "\r\n", "\n")
	lines := strings.Split(contentStr, "\n")

	// Remove the last empty line if content ends with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// normalizeForComparison normalizes a line for comparison
func normalizeForComparison(line string, ignoreWhitespace bool) string {
	if ignoreWhitespace {
		// Remove all whitespace for comparison
		return strings.Join(strings.Fields(line), "")
	}
	return line
}

// findChangedLines finds which lines were changed between parent and current
// Returns line numbers (1-indexed) in the current version
func findChangedLines(parentLines, currentLines []string, ignoreWhitespace bool) []int {
	changedLines := make([]int, 0)

	// Simple line-by-line comparison
	// A more sophisticated algorithm would use LCS (Longest Common Subsequence)
	// but this simple approach works well for most cases

	// For lines that exist in both versions
	minLen := len(parentLines)
	if len(currentLines) < minLen {
		minLen = len(currentLines)
	}

	for i := 0; i < minLen; i++ {
		parentLine := normalizeForComparison(parentLines[i], ignoreWhitespace)
		currentLine := normalizeForComparison(currentLines[i], ignoreWhitespace)
		if parentLine != currentLine {
			changedLines = append(changedLines, i+1)
		}
	}

	// Any additional lines in current are new
	for i := len(parentLines); i < len(currentLines); i++ {
		changedLines = append(changedLines, i+1)
	}

	return changedLines
}

// displayBlame displays the blame information
func displayBlame(blameInfo []BlameLineInfo, filePath string) {
	fmt.Println(ui.Header(fmt.Sprintf(" Blame: %s ", filePath)))
	fmt.Println()

	if len(blameInfo) == 0 {
		fmt.Println(ui.Yellow("File is empty"))
		return
	}

	// Find the maximum author name length for alignment
	maxAuthorLen := 0
	for _, info := range blameInfo {
		if len(info.Author) > maxAuthorLen {
			maxAuthorLen = len(info.Author)
		}
	}
	if maxAuthorLen > 20 {
		maxAuthorLen = 20 // Cap at 20 characters
	}

	// Display each line with blame info
	for _, info := range blameInfo {
		// Format the commit hash (first 8 characters)
		hashStr := info.ShortHash
		if len(hashStr) > 8 {
			hashStr = hashStr[:8]
		}

		// Format author name (truncate if too long)
		authorStr := info.Author
		if len(authorStr) > maxAuthorLen {
			authorStr = authorStr[:maxAuthorLen-2] + ".."
		}

		// Format date (YYYY-MM-DD)
		dateStr := "????-??-??"
		if !info.Date.IsZero() {
			dateStr = info.Date.Format("2006-01-02")
		}

		// Format line number
		lineNumStr := fmt.Sprintf("%4d", info.LineNumber)

		// Display the line
		fmt.Printf("%s %s %-*s %s %s %s\n",
			ui.Yellow(hashStr),
			ui.Cyan("("),
			maxAuthorLen, ui.Blue(authorStr),
			ui.Magenta(dateStr),
			ui.Cyan(lineNumStr+")"),
			info.Content)
	}

	fmt.Println()
}
