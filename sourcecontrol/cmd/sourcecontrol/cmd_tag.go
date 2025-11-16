package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/pkg/refs/tag"
)

func newTagCmd() *cobra.Command {
	var deleteFlag bool
	var listFlag bool
	var messageFlag string
	var annotateFlag bool
	var signFlag bool
	var forceFlag bool
	var localUserFlag string

	cmd := &cobra.Command{
		Use:   "tag [tag-name] [object]",
		Short: "Create, list, delete, or verify tags",
		Long: `Create, list, delete, or verify a tag object signed with GPG.

Tags are used to mark specific points in history as important, typically for releases.

Tag Types:
  - Lightweight: Simple pointer to a commit (like a branch that doesn't move)
  - Annotated: Full object with tagger name, email, date, and message
  - Signed: Annotated tag with a GPG signature

Examples:
  # List all tags
  srcc tag
  srcc tag -l

  # List tags matching a pattern
  srcc tag -l "v1.*"

  # Create a lightweight tag on current commit
  srcc tag v1.0.0

  # Create a lightweight tag on a specific commit
  srcc tag v1.0.0 abc123

  # Create an annotated tag
  srcc tag -a v1.0.0 -m "Release version 1.0.0"

  # Create a signed tag
  srcc tag -s v1.0.0 -m "Signed release"

  # Force create/update a tag
  srcc tag -f v1.0.0

  # Delete a tag
  srcc tag -d v1.0.0`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			manager := tag.NewManager(repo)
			ctx := context.Background()

			// Handle delete operation
			if deleteFlag {
				if len(args) == 0 {
					return fmt.Errorf("tag name required for deletion")
				}
				return deleteTag(ctx, manager, args[0])
			}

			// Handle list operation (default when no tag name is provided)
			if len(args) == 0 || listFlag {
				pattern := ""
				if len(args) > 0 {
					pattern = args[0]
				}
				return listTags(ctx, manager, pattern)
			}

			// Handle create operation
			tagName := args[0]
			objectRef := ""
			if len(args) > 1 {
				objectRef = args[1]
			}

			return createTag(ctx, manager, tagName, objectRef, messageFlag, annotateFlag, signFlag, forceFlag, localUserFlag)
		},
	}

	cmd.Flags().BoolVarP(&deleteFlag, "delete", "d", false, "Delete a tag")
	cmd.Flags().BoolVarP(&listFlag, "list", "l", false, "List tags")
	cmd.Flags().StringVarP(&messageFlag, "message", "m", "", "Tag message for annotated tags")
	cmd.Flags().BoolVarP(&annotateFlag, "annotate", "a", false, "Create an annotated tag")
	cmd.Flags().BoolVarP(&signFlag, "sign", "s", false, "Create a GPG-signed tag")
	cmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force create tag even if it exists")
	cmd.Flags().StringVarP(&localUserFlag, "local-user", "u", "", "GPG key to use for signing")

	return cmd
}

func createTag(ctx context.Context, manager *tag.Manager, name, objectRef, message string, annotate, sign, force bool, localUser string) error {
	var opts []tag.CreateOption

	if force {
		opts = append(opts, tag.WithForceCreate())
	}

	if message != "" {
		opts = append(opts, tag.WithMessage(message))
	} else if annotate {
		opts = append(opts, tag.WithAnnotate())
	}

	if sign {
		opts = append(opts, tag.WithSign(localUser))
	}

	if err := manager.CreateTag(ctx, name, objectRef, opts...); err != nil {
		return err
	}

	switch {
	case sign:
		fmt.Printf("Created signed tag '%s'\n", name)
	case annotate || message != "":
		fmt.Printf("Created annotated tag '%s'\n", name)
	default:
		fmt.Printf("Created tag '%s'\n", name)
	}

	return nil
}

func deleteTag(ctx context.Context, manager *tag.Manager, name string) error {
	if err := manager.DeleteTag(ctx, name); err != nil {
		return err
	}

	fmt.Printf("Deleted tag '%s'\n", name)
	return nil
}

func listTags(ctx context.Context, manager *tag.Manager, pattern string) error {
	var opts []tag.ListOption
	if pattern != "" {
		opts = append(opts, tag.WithPattern(pattern))
	}

	tags, err := manager.ListTags(ctx, opts...)
	if err != nil {
		return err
	}

	if len(tags) == 0 {
		if pattern != "" {
			fmt.Printf("No tags found matching pattern '%s'\n", pattern)
		} else {
			fmt.Println("No tags found")
		}
		return nil
	}

	for _, t := range tags {
		// Display tag name and type indicator
		typeIndicator := ""
		switch t.Type {
		case tag.Annotated:
			typeIndicator = " (annotated)"
		case tag.Signed:
			typeIndicator = " (signed)"
		}

		if t.Message != "" {
			fmt.Printf("%s%s - %s\n", t.Name, typeIndicator, strings.TrimSpace(t.Message))
		} else {
			fmt.Printf("%s%s\n", t.Name, typeIndicator)
		}
	}

	return nil
}
