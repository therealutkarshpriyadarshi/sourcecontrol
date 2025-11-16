package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/refs/tag"
	"github.com/utkarsh5026/SourceControl/pkg/repository/refs"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

func newDescribeCmd() *cobra.Command {
	var allFlag bool
	var tagsFlag bool
	var longFlag bool
	var abbrevFlag int

	cmd := &cobra.Command{
		Use:   "describe [commit-ish]",
		Short: "Give a human-readable name to a commit",
		Long: `Show the most recent tag that is reachable from a commit.

This command finds the most recent tag reachable from a commit and creates
a human-readable name for the commit based on that tag.

The output format is:
  <tag>-<count>-g<sha>

Where:
  - <tag> is the most recent tag name
  - <count> is the number of commits since that tag
  - <sha> is the abbreviated commit SHA (default 7 characters)

If the commit is tagged directly, only the tag name is shown.

Examples:
  # Describe current HEAD
  srcc describe

  # Describe a specific commit
  srcc describe abc123

  # Show longer SHA (10 characters instead of 7)
  srcc describe --abbrev=10

  # Always show long format even if commit is tagged
  srcc describe --long

  # Consider all refs, not just annotated tags
  srcc describe --all

  # Consider only tags (default)
  srcc describe --tags`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			commitRef := "HEAD"
			if len(args) > 0 {
				commitRef = args[0]
			}

			ctx := context.Background()
			result, err := describeCommit(ctx, repo, commitRef, allFlag, tagsFlag, longFlag, abbrevFlag)
			if err != nil {
				return err
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&allFlag, "all", false, "Consider all refs, not just annotated tags")
	cmd.Flags().BoolVar(&tagsFlag, "tags", true, "Consider only tags (default)")
	cmd.Flags().BoolVar(&longFlag, "long", false, "Always show long format (tag-count-sha)")
	cmd.Flags().IntVar(&abbrevFlag, "abbrev", 7, "Length of abbreviated SHA (default 7)")

	return cmd
}

func describeCommit(ctx context.Context, repo interface{}, commitRef string, all, onlyTags, long bool, abbrev int) (string, error) {
	// Convert to proper repository type
	sourceRepo, ok := repo.(*sourcerepo.SourceRepository)
	if !ok {
		return "", fmt.Errorf("invalid repository type")
	}

	// Get the commit SHA
	refManager := refs.NewRefManager(sourceRepo)
	commitSHA, err := refManager.ResolveToSHA(refs.RefPath(commitRef))
	if err != nil {
		return "", fmt.Errorf("cannot resolve '%s': %w", commitRef, err)
	}

	// Get all tags
	tagManager := tag.NewManager(sourceRepo)
	tags, err := tagManager.ListTags(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list tags: %w", err)
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found in repository")
	}

	// Find the nearest tag
	objStore := store.NewFileObjectStore()
	if err := objStore.Initialize(sourceRepo.WorkingDirectory()); err != nil {
		return "", fmt.Errorf("failed to initialize object store: %w", err)
	}

	nearestTag, distance, err := findNearestTag(objStore, tags, commitSHA)
	if err != nil {
		return "", err
	}

	// Format the output
	if distance == 0 && !long {
		// Commit is tagged directly
		return nearestTag.Name, nil
	}

	// Format: <tag>-<count>-g<sha>
	shortSHA := commitSHA.Short()
	if abbrev > 0 && abbrev < len(string(shortSHA)) {
		shortSHA = objects.ShortHash(commitSHA.String()[:abbrev])
	}

	return fmt.Sprintf("%s-%d-g%s", nearestTag.Name, distance, shortSHA), nil
}

// findNearestTag finds the nearest tag to a commit
func findNearestTag(store *store.FileObjectStore, tags []tag.TagInfo, commitSHA objects.ObjectHash) (*tag.TagInfo, int, error) {
	// Create a map of tag SHA to tag info
	tagMap := make(map[string]*tag.TagInfo)
	for i := range tags {
		tagMap[tags[i].SHA.String()] = &tags[i]
	}

	// Check if the commit is tagged directly
	if tagInfo, ok := tagMap[commitSHA.String()]; ok {
		return tagInfo, 0, nil
	}

	// Walk the commit history to find the nearest tag
	visited := make(map[string]bool)
	queue := []commitDistance{{sha: commitSHA, distance: 0}}

	// Sort tags by name for consistent results
	sortedTags := make([]tag.TagInfo, len(tags))
	copy(sortedTags, tags)
	sort.Slice(sortedTags, func(i, j int) bool {
		return sortedTags[i].Name > sortedTags[j].Name // Prefer newer tags (reverse sort)
	})

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.sha.String()] {
			continue
		}
		visited[current.sha.String()] = true

		// Check if this commit is tagged
		if tagInfo, ok := tagMap[current.sha.String()]; ok {
			return tagInfo, current.distance, nil
		}

		// Get parent commits
		commit, err := store.ReadObject(current.sha)
		if err != nil {
			continue
		}

		if commit.Type() != objects.CommitType {
			continue
		}

		// Parse commit to get parents
		content, err := commit.Content()
		if err != nil {
			continue
		}
		parents := parseCommitParents(content)
		for _, parent := range parents {
			queue = append(queue, commitDistance{
				sha:      parent,
				distance: current.distance + 1,
			})
		}
	}

	// If no tag found in history, return the most recent tag with unknown distance
	if len(sortedTags) > 0 {
		return &sortedTags[0], -1, nil
	}

	return nil, 0, fmt.Errorf("no reachable tags found")
}

type commitDistance struct {
	sha      objects.ObjectHash
	distance int
}

// parseCommitParents extracts parent commit SHAs from commit content
func parseCommitParents(content []byte) []objects.ObjectHash {
	var parents []objects.ObjectHash
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		if line == "" {
			break // End of header
		}
		if strings.HasPrefix(line, "parent ") {
			parentSHA := strings.TrimPrefix(line, "parent ")
			if hash, err := objects.NewObjectHashFromString(strings.TrimSpace(parentSHA)); err == nil {
				parents = append(parents, hash)
			}
		}
	}

	return parents
}
