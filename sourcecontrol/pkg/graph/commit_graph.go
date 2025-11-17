package graph

import (
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
)

// GraphNode represents a single commit in the visual graph with layout information
type GraphNode struct {
	// Commit is the actual commit object
	Commit *commit.Commit

	// Lane is the vertical column this commit occupies in the graph
	Lane int

	// ParentLanes maps each parent commit to its lane number
	// For merge commits, this tracks where merge lines come from
	ParentLanes []int

	// IsMerge indicates if this is a merge commit (2+ parents)
	IsMerge bool

	// IsInitial indicates if this is an initial commit (no parents)
	IsInitial bool

	// Children are commits that have this commit as a parent
	Children []*GraphNode

	// Index is the position in the commit list (0 = most recent)
	Index int
}

// CommitGraph represents the visual structure of commit history
type CommitGraph struct {
	// Nodes are the commits in display order (reverse chronological)
	Nodes []*GraphNode

	// MaxLane is the maximum lane number used (determines graph width)
	MaxLane int

	// CommitMap maps commit hash to graph node for quick lookup
	CommitMap map[string]*GraphNode
}

// NewCommitGraph creates a new empty commit graph
func NewCommitGraph() *CommitGraph {
	return &CommitGraph{
		Nodes:     make([]*GraphNode, 0),
		MaxLane:   0,
		CommitMap: make(map[string]*GraphNode),
	}
}

// AddNode adds a node to the graph
func (g *CommitGraph) AddNode(node *GraphNode) {
	g.Nodes = append(g.Nodes, node)

	hash, err := node.Commit.Hash()
	if err == nil {
		g.CommitMap[hash.String()] = node
	}

	if node.Lane > g.MaxLane {
		g.MaxLane = node.Lane
	}
}

// GetNode retrieves a node by commit hash
func (g *CommitGraph) GetNode(commitHash string) *GraphNode {
	return g.CommitMap[commitHash]
}

// Width returns the width of the graph in lanes
func (g *CommitGraph) Width() int {
	return g.MaxLane + 1
}

// Height returns the height of the graph in commits
func (g *CommitGraph) Height() int {
	return len(g.Nodes)
}
