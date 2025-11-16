package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FileStatus represents the status of a file in the repository
type FileStatus int

const (
	StatusModified FileStatus = iota
	StatusDeleted
	StatusAdded
	StatusUntracked
)

// FormatFileStatus formats a file path with the appropriate status icon and color
func FormatFileStatus(status FileStatus, path string) string {
	switch status {
	case StatusModified:
		return fmt.Sprintf("  %s  %s", ModifiedStyle.Render(IconModified), ModifiedStyle.Render(path))
	case StatusDeleted:
		return fmt.Sprintf("  %s  %s", DeletedStyle.Render(IconDeleted), DeletedStyle.Render(path))
	case StatusAdded:
		return fmt.Sprintf("  %s  %s", AddedStyle.Render(IconAdded), AddedStyle.Render(path))
	case StatusUntracked:
		return fmt.Sprintf("  %s  %s", UntrackedStyle.Render(IconUntracked), UntrackedStyle.Render(path))
	default:
		return path
	}
}

// FormatModified formats a modified file path
func FormatModified(path string) string {
	return FormatFileStatus(StatusModified, path)
}

// FormatDeleted formats a deleted file path
func FormatDeleted(path string) string {
	return FormatFileStatus(StatusDeleted, path)
}

// FormatAdded formats an added file path
func FormatAdded(path string) string {
	return FormatFileStatus(StatusAdded, path)
}

// FormatUntracked formats an untracked file path
func FormatUntracked(path string) string {
	return FormatFileStatus(StatusUntracked, path)
}

// SuccessMessage creates a success message with a checkmark icon
func SuccessMessage(message string, details ...string) string {
	var parts []string
	parts = append(parts, Green(IconCheckmark), Green(message))

	for _, detail := range details {
		parts = append(parts, Blue(detail))
	}

	return strings.Join(parts, " ")
}

// BranchInfo formats branch information with an icon
func BranchInfo(branchName string) string {
	return fmt.Sprintf("%s Branch: %s", Cyan(IconBranch), Blue(branchName))
}

// CommitInfo represents information about a commit
type CommitInfo struct {
	Hash    string
	Author  string
	Date    string
	Message string
}

// FormatCommitDetailed formats a commit with full details in a box
func FormatCommitDetailed(commit CommitInfo) string {
	var content strings.Builder

	// Hash line
	content.WriteString(fmt.Sprintf("%s %s\n", Yellow(IconCommit), Yellow(commit.Hash)))

	// Author line
	content.WriteString(fmt.Sprintf("%s %s\n", Cyan(IconAuthor), Cyan(commit.Author)))

	// Date line
	content.WriteString(fmt.Sprintf("%s %s\n", Magenta(IconDate), Magenta(commit.Date)))

	// Message with top margin
	messageStyle := ColorCyanStyle.Copy().MarginTop(1)
	content.WriteString(messageStyle.Render(commit.Message))

	return CommitBox(content.String())
}

// FormatCommitSeparator creates a separator between commits
func FormatCommitSeparator() string {
	separatorStyle := ColorCyanStyle.Copy().Foreground(lipgloss.Color("#888888"))
	return separatorStyle.Render(IconSeparator)
}

// ErrorMessage formats an error message in red
func ErrorMessage(message string) string {
	return Red(message)
}

// WarningMessage formats a warning message in yellow
func WarningMessage(message string) string {
	return Yellow(message)
}

// InfoMessage formats an info message in blue
func InfoMessage(message string) string {
	return Blue(message)
}
