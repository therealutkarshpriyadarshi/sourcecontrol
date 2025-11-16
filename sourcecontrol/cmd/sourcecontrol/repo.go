package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/cmd/ui"
	"github.com/utkarsh5026/SourceControl/pkg/refs/branch"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/workdir"
)

func newInitCmd() *cobra.Command {
	var bare bool

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a new SourceControl repository",
		Long: `Initialize a new SourceControl repository in the current directory or specified path.
This creates a .git directory with all necessary subdirectories and files.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("failed to resolve path: %w", err)
			}

			repoPath, err := scpath.NewRepositoryPath(absPath)
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}

			repo := sourcerepo.NewSourceRepository()
			if err := repo.Initialize(repoPath); err != nil {
				return fmt.Errorf("failed to initialize repository: %w", err)
			}

			message := "Initialized empty SourceControl repository in"
			if bare {
				message = "Initialized empty bare SourceControl repository in"
			}

			displayPath := fmt.Sprintf("%s/%s", absPath, scpath.SourceDir)
			fmt.Println(ui.SuccessMessage(message, displayPath))

			return nil
		},
	}

	cmd.Flags().BoolVar(&bare, "bare", false, "Create a bare repository")

	return cmd
}

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the working directory status",
		Long: `Show the status of the working directory and staging area.
Displays which files are modified, staged, untracked, etc.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			manager := workdir.NewManager(repo)
			status, err := manager.IsClean()
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			branchName, err := getCurrentBranchName(repo)
			if err != nil {
				branchName = branch.DefaultBranch
			}

			fmt.Println(ui.Header(" Repository Status "))
			fmt.Println(ui.BranchInfo(branchName))
			fmt.Println()

			if status.Clean {
				fmt.Println(ui.Green(fmt.Sprintf("  %s  Working tree clean - nothing to commit", ui.IconCheck)))
			} else {
				hasChanges := false

				if len(status.ModifiedFiles) > 0 {
					hasChanges = true
					fmt.Println(ui.Section("Changes not staged for commit:"))
					for _, path := range status.ModifiedFiles {
						fmt.Println(ui.FormatModified(string(path)))
					}
					fmt.Println()
				}

				if len(status.DeletedFiles) > 0 {
					hasChanges = true
					fmt.Println(ui.Section("Deleted files:"))
					for _, path := range status.DeletedFiles {
						fmt.Println(ui.FormatDeleted(string(path)))
					}
					fmt.Println()
				}

				if len(status.UntrackedFiles) > 0 {
					hasChanges = true
					fmt.Println(ui.Section("Untracked files:"))
					for _, path := range status.UntrackedFiles {
						fmt.Println(ui.FormatUntracked(string(path)))
					}
					fmt.Println()
				}

				if hasChanges {
					fmt.Println(ui.Yellow("  💡 Use 'sc add <file>' to stage changes for commit"))
				}
			}

			return nil
		},
	}

	return cmd
}
