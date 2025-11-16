package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/cmd/ui"
	"github.com/utkarsh5026/SourceControl/pkg/commitmanager"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
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

func newLogCmd() *cobra.Command {
	var limit int
	var useTable bool

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show commit logs",
		Long: `Show the commit logs.
Displays the commit history starting from the current HEAD.`,
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

			history, err := commitMgr.GetHistory(ctx, objects.ObjectHash(""), limit)
			if err != nil {
				return fmt.Errorf("failed to get history: %w", err)
			}

			if len(history) == 0 {
				fmt.Println(ui.Yellow("📝 No commits yet"))
				return nil
			}

			if useTable {
				displayCommitsAsTable(history)
			} else {
				displayCommitsDetailed(history)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Limit the number of commits to show")
	cmd.Flags().BoolVarP(&useTable, "table", "t", false, "Display commits in table format")

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
