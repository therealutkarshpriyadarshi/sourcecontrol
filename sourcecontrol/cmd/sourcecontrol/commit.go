package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/cmd/ui"
	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/graph"
	filehistory "github.com/utkarsh5026/SourceControl/pkg/history"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

func newCommitCmd() *cobra.Command {
	var message string

	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Record changes to the repository",
		Long: `Create a new commit with the staged changes.
Commits are snapshots of your project at a specific point in time.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			if message == "" {
				return fmt.Errorf("commit message required (use -m flag)")
			}

			ctx := context.Background()
			commitMgr := commitmanager.NewManager(repo)
			if err := commitMgr.Initialize(ctx); err != nil {
				return fmt.Errorf("failed to initialize commit manager: %w", err)
			}

			result, err := commitMgr.CreateCommit(ctx, commitmanager.CommitOptions{
				Message: message,
			})
			if err != nil {
				return fmt.Errorf("failed to create commit: %w", err)
			}

			commitHash, _ := result.Hash()

			// Format commit output with colors
			fmt.Printf("%s [%s] %s\n",
				ui.Green(ui.IconCommit),
				ui.Yellow(string(commitHash.Short())),
				ui.Cyan(result.Message))
			fmt.Printf("%s %s <%s>\n",
				ui.Cyan(ui.IconAuthor),
				ui.Blue(result.Author.Name),
				ui.Blue(result.Author.Email))

			return nil
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "Commit message")

	return cmd
}

// logOptions holds all the options for the log command
type logOptions struct {
	limit      int
	useTable   bool
	useGraph   bool
	oneline    bool
	format     string
	follow     string
	author     string
	since      string
	until      string
	grep       string
	pretty     string
}

func newLogCmd() *cobra.Command {
	opts := &logOptions{}

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show commit logs",
		Long: `Show the commit logs with various formatting and filtering options.

Displays the commit history starting from the current HEAD with support for:
- Graph visualization (--graph)
- Custom formatting (--format, --oneline, --pretty)
- File history tracking (--follow)
- Author and date filtering (--author, --since, --until)
- Commit message search (--grep)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			ctx := context.Background()
			commitMgr := commitmanager.NewManager(repo)
			if err := commitMgr.Initialize(ctx); err != nil {
				return fmt.Errorf("failed to initialize commit manager: %w", err)
			}

			// Get all history (we'll filter later)
			maxLimit := opts.limit
			if maxLimit == 0 {
				maxLimit = 1000 // reasonable default for filtering
			}

			history, err := commitMgr.GetHistory(ctx, objects.ObjectHash(""), maxLimit)
			if err != nil {
				return fmt.Errorf("failed to get history: %w", err)
			}

			if len(history) == 0 {
				fmt.Println(ui.Yellow("📝 No commits yet"))
				return nil
			}

			// Apply filters
			history, err = applyFilters(history, opts, repo)
			if err != nil {
				return fmt.Errorf("failed to apply filters: %w", err)
			}

			if len(history) == 0 {
				fmt.Println(ui.Yellow("📝 No commits match the filter criteria"))
				return nil
			}

			// Apply limit after filtering
			if opts.limit > 0 && len(history) > opts.limit {
				history = history[:opts.limit]
			}

			// Display commits based on options
			if err := displayCommits(history, opts); err != nil {
				return fmt.Errorf("failed to display commits: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.limit, "limit", "n", 20, "Limit the number of commits to show")
	cmd.Flags().BoolVarP(&opts.useTable, "table", "t", false, "Display commits in table format")
	cmd.Flags().BoolVar(&opts.useGraph, "graph", false, "Show commit graph visualization")
	cmd.Flags().BoolVar(&opts.oneline, "oneline", false, "Show each commit as a single line")
	cmd.Flags().StringVar(&opts.format, "format", "", "Custom format string (e.g., '%h %an %s')")
	cmd.Flags().StringVar(&opts.follow, "follow", "", "Follow history of a specific file")
	cmd.Flags().StringVar(&opts.author, "author", "", "Filter commits by author name or email")
	cmd.Flags().StringVar(&opts.since, "since", "", "Show commits since date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.until, "until", "", "Show commits until date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.grep, "grep", "", "Filter commits by message content (regex)")
	cmd.Flags().StringVar(&opts.pretty, "pretty", "", "Pretty format: oneline, short, medium, full")

	return cmd
}

// displayCommitsDetailed shows commits in a detailed, beautiful format
func displayCommitsDetailed(history []*commit.Commit) {
	fmt.Println(ui.Header(" Commit History "))
	fmt.Println()

	for i, c := range history {
		commitHash, _ := c.Hash()

		commitInfo := ui.CommitInfo{
			Hash:    commitHash.String(),
			Author:  fmt.Sprintf("%s <%s>", c.Author.Name, c.Author.Email),
			Date:    c.Author.When.Time().Format(time.RFC1123),
			Message: c.Message,
		}

		fmt.Println(ui.FormatCommitDetailed(commitInfo))
		if i < len(history)-1 {
			fmt.Println(ui.FormatCommitSeparator())
		}
	}
}

// displayCommitsAsTable shows commits in a compact table format
func displayCommitsAsTable(history []*commit.Commit) {
	fmt.Println(ui.Header(" Commit History "))
	fmt.Println()

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Commit", "Author", "Date", "Message")

	for _, c := range history {
		commitHash, _ := c.Hash()
		shortHash := commitHash.String()
		if len(shortHash) > 8 {
			shortHash = shortHash[:8]
		}

		message := c.Message
		if len(message) > 50 {
			message = message[:47] + "..."
		}

		table.Append(
			ui.Yellow(shortHash),
			ui.Cyan(c.Author.Name),
			ui.Magenta(c.Author.When.Time().Format("2006-01-02 15:04")),
			message,
		)
	}

	table.Render()
}

// applyFilters applies all the requested filters to the commit history
func applyFilters(history []*commit.Commit, opts *logOptions, repo interface{}) ([]*commit.Commit, error) {
	filtered := history

	// File history filter (apply FIRST before other filters)
	if opts.follow != "" {
		sourceRepo, ok := repo.(*sourcerepo.SourceRepository)
		if !ok {
			return nil, fmt.Errorf("invalid repository type for file history")
		}

		walker := filehistory.NewFileHistoryWalker(sourceRepo)
		fileFiltered, err := walker.FilterByFile(context.Background(), filtered, opts.follow)
		if err != nil {
			return nil, fmt.Errorf("failed to filter file history: %w", err)
		}
		filtered = fileFiltered
	}

	// Parse date filters
	var sinceTime, untilTime time.Time
	var err error

	if opts.since != "" {
		sinceTime, err = time.Parse("2006-01-02", opts.since)
		if err != nil {
			return nil, fmt.Errorf("invalid --since date format (use YYYY-MM-DD): %w", err)
		}
	}

	if opts.until != "" {
		untilTime, err = time.Parse("2006-01-02", opts.until)
		if err != nil {
			return nil, fmt.Errorf("invalid --until date format (use YYYY-MM-DD): %w", err)
		}
		// Set to end of day
		untilTime = untilTime.Add(24*time.Hour - time.Second)
	}

	// Compile grep pattern if provided
	var grepPattern *regexp.Regexp
	if opts.grep != "" {
		grepPattern, err = regexp.Compile(opts.grep)
		if err != nil {
			return nil, fmt.Errorf("invalid --grep pattern: %w", err)
		}
	}

	// Apply other filters to the file-filtered history
	finalFiltered := make([]*commit.Commit, 0, len(filtered))
	for _, c := range filtered {
		// Author filter
		if opts.author != "" {
			authorMatch := strings.Contains(strings.ToLower(c.Author.Name), strings.ToLower(opts.author)) ||
				strings.Contains(strings.ToLower(c.Author.Email), strings.ToLower(opts.author))
			if !authorMatch {
				continue
			}
		}

		// Date filters
		commitTime := c.Author.When.Time()
		if opts.since != "" && commitTime.Before(sinceTime) {
			continue
		}
		if opts.until != "" && commitTime.After(untilTime) {
			continue
		}

		// Message grep filter
		if grepPattern != nil && !grepPattern.MatchString(c.Message) {
			continue
		}

		finalFiltered = append(finalFiltered, c)
	}

	return finalFiltered, nil
}

// displayCommits displays commits based on the selected options
func displayCommits(history []*commit.Commit, opts *logOptions) error {
	// Handle custom format
	if opts.format != "" {
		return displayCommitsCustomFormat(history, opts.format, opts.useGraph)
	}

	// Handle pretty formats
	if opts.pretty != "" {
		return displayCommitsPretty(history, opts.pretty, opts.useGraph)
	}

	// Handle oneline format
	if opts.oneline {
		return displayCommitsOneline(history, opts.useGraph)
	}

	// Handle graph with default format
	if opts.useGraph {
		return displayCommitsGraph(history, false)
	}

	// Handle table format
	if opts.useTable {
		displayCommitsAsTable(history)
		return nil
	}

	// Default detailed format
	displayCommitsDetailed(history)
	return nil
}

// displayCommitsOneline shows commits in a compact one-line format
func displayCommitsOneline(history []*commit.Commit, withGraph bool) error {
	if withGraph {
		// Use the new graph renderer in compact mode
		return displayCommitsGraph(history, true)
	}

	// No graph - simple oneline format
	for _, c := range history {
		commitHash, _ := c.Hash()
		shortHash := commitHash.Short().String()

		// Get first line of message
		message := strings.Split(c.Message, "\n")[0]
		if len(message) > 70 {
			message = message[:67] + "..."
		}

		fmt.Printf("%s %s\n",
			ui.Yellow(shortHash),
			message)
	}
	return nil
}

// displayCommitsGraph shows commits with graph visualization
func displayCommitsGraph(history []*commit.Commit, compact bool) error {
	fmt.Println(ui.Header(" Commit Graph "))
	fmt.Println()

	// Build the commit graph
	builder := graph.NewGraphBuilder()
	commitGraph, err := builder.Build(history)
	if err != nil {
		return fmt.Errorf("failed to build commit graph: %w", err)
	}

	// Render the graph
	renderer := graph.NewRenderer(commitGraph)
	output := renderer.Render(compact)

	fmt.Print(output)
	return nil
}

// buildGraphPrefix builds the graph visualization prefix for a commit
func buildGraphPrefix(history []*commit.Commit, index int) string {
	c := history[index]

	// Simple linear graph for now
	if len(c.ParentSHAs) == 0 {
		// Initial commit
		return ui.Yellow("◉")
	} else if len(c.ParentSHAs) == 1 {
		// Normal commit
		return ui.Yellow("◉")
	} else {
		// Merge commit
		return ui.Magenta("◎")
	}
}

// buildGraphContinuation builds the continuation line for graph
func buildGraphContinuation(history []*commit.Commit, index int) string {
	if index < len(history)-1 {
		return ui.Cyan("│")
	}
	return " "
}

// displayCommitsPretty displays commits using predefined pretty formats
func displayCommitsPretty(history []*commit.Commit, format string, withGraph bool) error {
	switch format {
	case "oneline":
		return displayCommitsOneline(history, withGraph)
	case "short":
		return displayCommitsShort(history, withGraph)
	case "medium":
		return displayCommitsMedium(history, withGraph)
	case "full":
		return displayCommitsFull(history, withGraph)
	default:
		return fmt.Errorf("unknown pretty format: %s (use: oneline, short, medium, full)", format)
	}
}

// displayCommitsShort displays commits in short format
func displayCommitsShort(history []*commit.Commit, withGraph bool) error {
	for i, c := range history {
		commitHash, _ := c.Hash()
		graphPrefix := ""
		if withGraph {
			graphPrefix = buildGraphPrefix(history, i) + " "
		}

		fmt.Printf("%s%s %s\n", graphPrefix, ui.Yellow("commit"), ui.Yellow(commitHash.Short().String()))
		fmt.Printf("Author: %s <%s>\n", c.Author.Name, c.Author.Email)
		fmt.Println()

		lines := strings.Split(c.Message, "\n")
		for _, line := range lines {
			fmt.Printf("    %s\n", line)
		}

		if i < len(history)-1 {
			fmt.Println()
		}
	}
	return nil
}

// displayCommitsMedium displays commits in medium format (default git log format)
func displayCommitsMedium(history []*commit.Commit, withGraph bool) error {
	for i, c := range history {
		commitHash, _ := c.Hash()
		graphPrefix := ""
		if withGraph {
			graphPrefix = buildGraphPrefix(history, i) + " "
		}

		fmt.Printf("%s%s %s\n", graphPrefix, ui.Yellow("commit"), ui.Yellow(commitHash.String()))
		fmt.Printf("Author: %s <%s>\n", c.Author.Name, c.Author.Email)
		fmt.Printf("Date:   %s\n", c.Author.When.Time().Format(time.RFC1123))
		fmt.Println()

		lines := strings.Split(c.Message, "\n")
		for _, line := range lines {
			fmt.Printf("    %s\n", line)
		}

		if i < len(history)-1 {
			fmt.Println()
		}
	}
	return nil
}

// displayCommitsFull displays commits in full format with all details
func displayCommitsFull(history []*commit.Commit, withGraph bool) error {
	for i, c := range history {
		commitHash, _ := c.Hash()
		graphPrefix := ""
		if withGraph {
			graphPrefix = buildGraphPrefix(history, i) + " "
		}

		fmt.Printf("%s%s %s\n", graphPrefix, ui.Yellow("commit"), ui.Yellow(commitHash.String()))
		fmt.Printf("Author: %s <%s>\n", c.Author.Name, c.Author.Email)
		fmt.Printf("Date:   %s\n", c.Author.When.Time().Format(time.RFC1123))

		if c.Committer != nil && (c.Committer.Name != c.Author.Name || c.Committer.Email != c.Author.Email) {
			fmt.Printf("Commit: %s <%s>\n", c.Committer.Name, c.Committer.Email)
			fmt.Printf("CommitDate: %s\n", c.Committer.When.Time().Format(time.RFC1123))
		}

		if len(c.ParentSHAs) > 1 {
			fmt.Printf("Merge:")
			for _, parent := range c.ParentSHAs {
				fmt.Printf(" %s", parent.Short().String())
			}
			fmt.Println()
		}

		fmt.Println()
		lines := strings.Split(c.Message, "\n")
		for _, line := range lines {
			fmt.Printf("    %s\n", line)
		}

		if i < len(history)-1 {
			fmt.Println()
		}
	}
	return nil
}

// displayCommitsCustomFormat displays commits using a custom format string
func displayCommitsCustomFormat(history []*commit.Commit, format string, withGraph bool) error {
	for i, c := range history {
		commitHash, _ := c.Hash()
		graphPrefix := ""
		if withGraph {
			graphPrefix = buildGraphPrefix(history, i) + " "
		}

		output := format

		// Replace format placeholders
		output = strings.ReplaceAll(output, "%H", commitHash.String())
		output = strings.ReplaceAll(output, "%h", commitHash.Short().String())
		output = strings.ReplaceAll(output, "%an", c.Author.Name)
		output = strings.ReplaceAll(output, "%ae", c.Author.Email)
		output = strings.ReplaceAll(output, "%ad", c.Author.When.Time().Format(time.RFC1123))
		output = strings.ReplaceAll(output, "%ar", formatRelativeTime(c.Author.When.Time()))
		output = strings.ReplaceAll(output, "%s", strings.Split(c.Message, "\n")[0])
		output = strings.ReplaceAll(output, "%b", strings.Join(strings.Split(c.Message, "\n")[1:], "\n"))
		output = strings.ReplaceAll(output, "%B", c.Message)

		if c.Committer != nil {
			output = strings.ReplaceAll(output, "%cn", c.Committer.Name)
			output = strings.ReplaceAll(output, "%ce", c.Committer.Email)
			output = strings.ReplaceAll(output, "%cd", c.Committer.When.Time().Format(time.RFC1123))
			output = strings.ReplaceAll(output, "%cr", formatRelativeTime(c.Committer.When.Time()))
		}

		// Handle parent info
		if len(c.ParentSHAs) > 0 {
			output = strings.ReplaceAll(output, "%P", strings.Join(toStringArray(c.ParentSHAs), " "))
			output = strings.ReplaceAll(output, "%p", strings.Join(toShortStringArray(c.ParentSHAs), " "))
		} else {
			output = strings.ReplaceAll(output, "%P", "")
			output = strings.ReplaceAll(output, "%p", "")
		}

		fmt.Printf("%s%s\n", graphPrefix, output)
	}
	return nil
}

// formatRelativeTime formats a time as a relative string (e.g., "2 hours ago")
func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 30*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if duration < 365*24*time.Hour {
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	} else {
		years := int(duration.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// toStringArray converts ObjectHash array to string array
func toStringArray(hashes []objects.ObjectHash) []string {
	result := make([]string, len(hashes))
	for i, h := range hashes {
		result[i] = h.String()
	}
	return result
}

// toShortStringArray converts ObjectHash array to short string array
func toShortStringArray(hashes []objects.ObjectHash) []string {
	result := make([]string, len(hashes))
	for i, h := range hashes {
		result[i] = h.Short().String()
	}
	return result
}
