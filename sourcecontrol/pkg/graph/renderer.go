package graph

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Box drawing characters for graph visualization
const (
	// Vertical and horizontal lines
	LineVertical   = "│"
	LineHorizontal = "─"

	// Corners and junctions
	CornerTopLeft     = "┌"
	CornerTopRight    = "┐"
	CornerBottomLeft  = "└"
	CornerBottomRight = "┘"
	JunctionLeft      = "├"
	JunctionRight     = "┤"
	JunctionTop       = "┬"
	JunctionBottom    = "┴"
	JunctionCross     = "┼"

	// Commit markers
	CommitNormal = "●"
	CommitMerge  = "◎"
	CommitInitial = "◆"

	// Merge lines
	MergeRight = "┐"
	MergeLeft  = "┘"
	MergeFork  = "┬"
	MergeJoin  = "┴"
	MergeCross = "┼"
)

// Colors for different lanes
var laneColors = []lipgloss.Color{
	lipgloss.Color("#00D7FF"), // Cyan
	lipgloss.Color("#AF87FF"), // Purple
	lipgloss.Color("#00FF87"), // Green
	lipgloss.Color("#FFD700"), // Gold
	lipgloss.Color("#FF5F87"), // Pink
	lipgloss.Color("#5FD7FF"), // Light Blue
	lipgloss.Color("#FFD787"), // Light Orange
	lipgloss.Color("#87FFD7"), // Aqua
}

// GraphRenderer renders a commit graph as text with colors
type GraphRenderer struct {
	graph *CommitGraph
}

// NewRenderer creates a new graph renderer
func NewRenderer(graph *CommitGraph) *GraphRenderer {
	return &GraphRenderer{
		graph: graph,
	}
}

// Render renders the graph as a string
//
// If compact is true, uses single-line format per commit
// Otherwise, uses multi-line detailed format
func (r *GraphRenderer) Render(compact bool) string {
	if compact {
		return r.renderCompact()
	}
	return r.renderDetailed()
}

// renderCompact renders the graph in compact one-line-per-commit format
func (r *GraphRenderer) renderCompact() string {
	var output strings.Builder

	for i, node := range r.graph.Nodes {
		// Build graph prefix
		prefix := r.buildGraphLine(i, true)

		// Get commit info
		hash, _ := node.Commit.Hash()
		shortHash := hash.Short().String()

		// Get first line of message
		message := strings.Split(node.Commit.Message, "\n")[0]
		if len(message) > 60 {
			message = message[:57] + "..."
		}

		// Format line
		line := fmt.Sprintf("%s %s %s %s\n",
			prefix,
			r.colorize(shortHash, lipgloss.Color("#FFD700")),
			r.colorize(node.Commit.Author.Name, lipgloss.Color("#5FD7FF")),
			message,
		)

		output.WriteString(line)
	}

	return output.String()
}

// renderDetailed renders the graph in detailed multi-line format
func (r *GraphRenderer) renderDetailed() string {
	var output strings.Builder

	for i, node := range r.graph.Nodes {
		// Build graph visualization
		commitLine := r.buildGraphLine(i, true)
		continuationLine := r.buildGraphLine(i, false)

		// Get commit info
		hash, _ := node.Commit.Hash()

		// Commit hash line
		output.WriteString(fmt.Sprintf("%s %s %s\n",
			commitLine,
			r.colorize("●", r.getLaneColor(node.Lane)),
			r.colorize(hash.String(), lipgloss.Color("#FFD700")),
		))

		// Author line
		output.WriteString(fmt.Sprintf("%s Author: %s <%s>\n",
			continuationLine,
			r.colorize(node.Commit.Author.Name, lipgloss.Color("#5FD7FF")),
			node.Commit.Author.Email,
		))

		// Date line
		output.WriteString(fmt.Sprintf("%s Date:   %s\n",
			continuationLine,
			r.colorize(node.Commit.Author.When.Time().Format(time.RFC1123), lipgloss.Color("#AF87FF")),
		))

		output.WriteString(continuationLine + "\n")

		// Message lines
		messageLines := strings.Split(node.Commit.Message, "\n")
		for _, line := range messageLines {
			output.WriteString(fmt.Sprintf("%s     %s\n", continuationLine, line))
		}

		// Add spacing between commits
		if i < len(r.graph.Nodes)-1 {
			output.WriteString(continuationLine + "\n")
		}
	}

	return output.String()
}

// buildGraphLine builds the graph visualization for a commit line
func (r *GraphRenderer) buildGraphLine(nodeIndex int, isCommitLine bool) string {
	width := r.graph.Width()

	var line strings.Builder

	// Build the line character by character for each lane
	for lane := 0; lane < width; lane++ {
		char := r.getCharForLane(nodeIndex, lane, isCommitLine)
		color := r.getLaneColor(lane)
		line.WriteString(r.colorize(char, color))

		// Add spacing between lanes
		if lane < width-1 {
			line.WriteString(" ")
		}
	}

	return line.String()
}

// getCharForLane determines what character to show in a specific lane
func (r *GraphRenderer) getCharForLane(nodeIndex int, lane int, isCommitLine bool) string {
	node := r.graph.Nodes[nodeIndex]

	if isCommitLine && lane == node.Lane {
		// This is the commit's lane - show commit marker
		if node.IsInitial {
			return CommitInitial
		} else if node.IsMerge {
			return CommitMerge
		}
		return CommitNormal
	}

	// Check if there are merge lines in this lane
	if isCommitLine && node.IsMerge {
		for _, parentLane := range node.ParentLanes {
			if lane == parentLane && lane != node.Lane {
				// This lane has a merge line coming in
				if lane < node.Lane {
					return LineVertical
				} else if lane > node.Lane {
					return LineVertical
				}
			}
		}
	}

	// Check if previous commit uses this lane
	if nodeIndex > 0 {
		prevNode := r.graph.Nodes[nodeIndex-1]

		// Check if previous commit's parents use this lane
		for _, parentLane := range prevNode.ParentLanes {
			if lane == parentLane {
				return LineVertical
			}
		}
	}

	// Check if next commit uses this lane
	if nodeIndex < len(r.graph.Nodes)-1 {
		nextNode := r.graph.Nodes[nodeIndex+1]
		if lane == nextNode.Lane {
			return LineVertical
		}

		// Check if current commit's parents use this lane
		for _, parentLane := range node.ParentLanes {
			if lane == parentLane {
				return LineVertical
			}
		}
	}

	return " "
}

// getLaneColor returns the color for a specific lane
func (r *GraphRenderer) getLaneColor(lane int) lipgloss.Color {
	return laneColors[lane%len(laneColors)]
}

// colorize applies color to text
func (r *GraphRenderer) colorize(text string, color lipgloss.Color) string {
	style := lipgloss.NewStyle().Foreground(color)
	return style.Render(text)
}

// RenderOneLine renders a single commit with graph prefix (for integration with existing code)
func (r *GraphRenderer) RenderOneLine(nodeIndex int) (string, string) {
	if nodeIndex >= len(r.graph.Nodes) {
		return "", ""
	}

	node := r.graph.Nodes[nodeIndex]
	prefix := r.buildGraphLine(nodeIndex, true)

	// Get commit marker
	var marker string
	if node.IsInitial {
		marker = CommitInitial
	} else if node.IsMerge {
		marker = CommitMerge
	} else {
		marker = CommitNormal
	}

	marker = r.colorize(marker, r.getLaneColor(node.Lane))

	return prefix, marker
}
