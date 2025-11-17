package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/cmd/ui"
	"github.com/utkarsh5026/SourceControl/pkg/stash"
)

func newStashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stash",
		Short: "Stash changes in a dirty working directory",
		Long: `Stash the changes in a dirty working directory away.

Use "stash save" to save your local modifications to a new stash entry and
roll them back to HEAD. The modifications stashed away can be listed with
"stash list", inspected with "stash show", and restored with "stash apply"
or "stash pop".`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior is to save a stash
			return stashSave(cmd, args, stash.StashOptions{
				Message: "WIP on stash",
			})
		},
	}

	// Add subcommands
	cmd.AddCommand(newStashSaveCmd())
	cmd.AddCommand(newStashListCmd())
	cmd.AddCommand(newStashShowCmd())
	cmd.AddCommand(newStashApplyCmd())
	cmd.AddCommand(newStashPopCmd())
	cmd.AddCommand(newStashDropCmd())
	cmd.AddCommand(newStashClearCmd())

	return cmd
}

func newStashSaveCmd() *cobra.Command {
	var message string
	var keepIndex bool
	var includeUntracked bool

	cmd := &cobra.Command{
		Use:   "save [<message>]",
		Short: "Save your local modifications to a new stash entry",
		Long: `Save your local modifications to a new stash entry and roll back to HEAD.

The modifications stashed away can be restored with "stash apply" or "stash pop".

Examples:
  # Stash with a message
  srcc stash save "work in progress"

  # Stash but keep staged changes
  srcc stash save --keep-index "temporary work"

  # Stash including untracked files
  srcc stash save --include-untracked "all changes"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := stash.StashOptions{
				Message:          message,
				KeepIndex:        keepIndex,
				IncludeUntracked: includeUntracked,
			}

			if len(args) > 0 {
				opts.Message = args[0]
			}

			if opts.Message == "" {
				opts.Message = "WIP on stash"
			}

			return stashSave(cmd, args, opts)
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "Stash message")
	cmd.Flags().BoolVarP(&keepIndex, "keep-index", "k", false, "Keep changes staged after stashing")
	cmd.Flags().BoolVarP(&includeUntracked, "include-untracked", "u", false, "Include untracked files in stash")

	return cmd
}

func newStashListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all stash entries",
		Long: `List the stash entries that you currently have.

Each stash entry is listed with its name and the description. The most
recent entry is stash@{0}, older entries have higher indices.

Examples:
  # List all stashes
  srcc stash list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return stashList(cmd, args)
		},
	}

	return cmd
}

func newStashShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [<stash>]",
		Short: "Show the changes recorded in the stash",
		Long: `Show the changes recorded in the stash as a diff.

If no stash is specified, shows the latest stash (stash@{0}).

Examples:
  # Show latest stash
  srcc stash show

  # Show specific stash
  srcc stash show stash@{1}
  srcc stash show 1`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return stashShow(cmd, args)
		},
	}

	return cmd
}

func newStashApplyCmd() *cobra.Command {
	var restoreIndex bool

	cmd := &cobra.Command{
		Use:   "apply [<stash>]",
		Short: "Apply a stash to the working directory",
		Long: `Apply the changes recorded in the stash to the working directory.

The stash entry is kept in the stash list for possible later reuse.
Use "stash pop" if you want to apply and remove in one operation.

Examples:
  # Apply latest stash
  srcc stash apply

  # Apply specific stash
  srcc stash apply stash@{1}
  srcc stash apply 1

  # Apply and restore staged changes
  srcc stash apply --index`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := stash.ApplyOptions{
				Index: restoreIndex,
			}
			return stashApply(cmd, args, opts)
		},
	}

	cmd.Flags().BoolVar(&restoreIndex, "index", false, "Try to restore staged changes as staged")

	return cmd
}

func newStashPopCmd() *cobra.Command {
	var restoreIndex bool

	cmd := &cobra.Command{
		Use:   "pop [<stash>]",
		Short: "Apply and remove a stash from the stash list",
		Long: `Apply the changes recorded in the stash to the working directory and remove
the stash entry from the stash list.

This is equivalent to "stash apply" followed by "stash drop".

Examples:
  # Pop latest stash
  srcc stash pop

  # Pop specific stash
  srcc stash pop stash@{1}
  srcc stash pop 1`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := stash.ApplyOptions{
				Index: restoreIndex,
				Pop:   true,
			}
			return stashApply(cmd, args, opts)
		},
	}

	cmd.Flags().BoolVar(&restoreIndex, "index", false, "Try to restore staged changes as staged")

	return cmd
}

func newStashDropCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drop [<stash>]",
		Short: "Remove a single stash entry",
		Long: `Remove a single stash entry from the list of stash entries.

If no stash is specified, removes the latest stash (stash@{0}).

Examples:
  # Drop latest stash
  srcc stash drop

  # Drop specific stash
  srcc stash drop stash@{1}
  srcc stash drop 1`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return stashDrop(cmd, args)
		},
	}

	return cmd
}

func newStashClearCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove all stash entries",
		Long: `Remove all stash entries.

This operation cannot be undone.

Examples:
  # Clear all stashes
  srcc stash clear`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return stashClear(cmd, args)
		},
	}

	return cmd
}

// Command implementations

func stashSave(cmd *cobra.Command, args []string, opts stash.StashOptions) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	ctx := context.Background()
	stashMgr := stash.NewManager(repo)
	if err := stashMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize stash manager: %w", err)
	}

	entry, err := stashMgr.Save(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to save stash: %w", err)
	}

	fmt.Printf("%s Saved working directory and index state %s: %s\n",
		ui.Green("✓"),
		ui.Yellow(stash.GetStashName(entry.Index)),
		ui.Cyan(entry.Message))
	fmt.Printf("%s on branch %s\n",
		ui.Cyan(ui.IconBranch),
		ui.Blue(entry.Branch))

	return nil
}

func stashList(cmd *cobra.Command, args []string) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	stashMgr := stash.NewManager(repo)

	entries, err := stashMgr.List()
	if err != nil {
		return fmt.Errorf("failed to list stashes: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println(ui.Yellow("📦 No stashes found"))
		return nil
	}

	fmt.Println(ui.Header(" Stash List "))
	fmt.Println()

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Stash", "Branch", "Message", "Created")

	for _, entry := range entries {
		stashName := stash.GetStashName(entry.Index)
		relativeTime := formatRelativeTime(entry.CreatedAt)

		table.Append(
			ui.Yellow(stashName),
			ui.Cyan(entry.Branch),
			entry.Message,
			ui.Magenta(relativeTime),
		)
	}

	table.Render()

	return nil
}

func stashShow(cmd *cobra.Command, args []string) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	ctx := context.Background()
	stashMgr := stash.NewManager(repo)
	if err := stashMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize stash manager: %w", err)
	}

	// Parse stash index
	stashIndex := 0
	if len(args) > 0 {
		stashIndex, err = stash.ParseStashName(args[0])
		if err != nil {
			return err
		}
	}

	stashCommit, err := stashMgr.Show(ctx, stashIndex)
	if err != nil {
		return fmt.Errorf("failed to show stash: %w", err)
	}

	// Display stash information
	fmt.Println(ui.Header(fmt.Sprintf(" Stash %s ", stash.GetStashName(stashIndex))))
	fmt.Println()

	commitHash, _ := stashCommit.Hash()
	fmt.Printf("%s %s\n", ui.Yellow("Commit:"), ui.Yellow(commitHash.String()))
	fmt.Printf("%s %s <%s>\n",
		ui.Cyan("Author:"),
		ui.Blue(stashCommit.Author.Name),
		ui.Blue(stashCommit.Author.Email))
	fmt.Printf("%s %s\n",
		ui.Magenta("Date:"),
		ui.Magenta(stashCommit.Author.When.Time().Format(time.RFC1123)))
	fmt.Println()
	fmt.Printf("    %s\n", stashCommit.Message)

	return nil
}

func stashApply(cmd *cobra.Command, args []string, opts stash.ApplyOptions) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	ctx := context.Background()
	stashMgr := stash.NewManager(repo)
	if err := stashMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize stash manager: %w", err)
	}

	// Parse stash index
	stashIndex := 0
	if len(args) > 0 {
		stashIndex, err = stash.ParseStashName(args[0])
		if err != nil {
			return err
		}
	}

	if err := stashMgr.Apply(ctx, stashIndex, opts); err != nil {
		return fmt.Errorf("failed to apply stash: %w", err)
	}

	action := "Applied"
	if opts.Pop {
		action = "Popped"
	}

	fmt.Printf("%s %s %s\n",
		ui.Green("✓"),
		action,
		ui.Yellow(stash.GetStashName(stashIndex)))

	return nil
}

func stashDrop(cmd *cobra.Command, args []string) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	stashMgr := stash.NewManager(repo)

	// Parse stash index
	stashIndex := 0
	if len(args) > 0 {
		stashIndex, err = stash.ParseStashName(args[0])
		if err != nil {
			return err
		}
	}

	if err := stashMgr.Drop(stashIndex); err != nil {
		return fmt.Errorf("failed to drop stash: %w", err)
	}

	fmt.Printf("%s Dropped %s\n",
		ui.Green("✓"),
		ui.Yellow(stash.GetStashName(stashIndex)))

	return nil
}

func stashClear(cmd *cobra.Command, args []string) error {
	repo, err := findRepository()
	if err != nil {
		return err
	}

	stashMgr := stash.NewManager(repo)

	entries, err := stashMgr.List()
	if err != nil {
		return fmt.Errorf("failed to list stashes: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println(ui.Yellow("📦 No stashes to clear"))
		return nil
	}

	if err := stashMgr.Clear(); err != nil {
		return fmt.Errorf("failed to clear stashes: %w", err)
	}

	fmt.Printf("%s Cleared %d stash %s\n",
		ui.Green("✓"),
		len(entries),
		pluralize("entry", "entries", len(entries)))

	return nil
}

// Helper functions

func pluralize(singular, plural string, count int) string {
	if count == 1 {
		return singular
	}
	return plural
}
