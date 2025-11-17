package graph

import (
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
)

// GraphBuilder builds a visual commit graph with proper lane allocation
//
// The builder uses a lane allocation algorithm that:
// 1. Assigns each commit to a vertical lane (column)
// 2. Keeps related commits in the same lane when possible
// 3. Handles merge commits by tracking multiple parent lanes
// 4. Minimizes lane crossings for better visualization
type GraphBuilder struct {
	// graph is the graph being built
	graph *CommitGraph

	// lanes tracks which commit is currently occupying each lane
	// lanes[i] = commit hash in lane i (empty string if free)
	lanes []string

	// reservations tracks future lane reservations for parent commits
	// reservations[commitHash] = preferred lane number
	reservations map[string]int
}

// NewGraphBuilder creates a new graph builder
func NewGraphBuilder() *GraphBuilder {
	return &GraphBuilder{
		graph:        NewCommitGraph(),
		lanes:        make([]string, 0, 10),
		reservations: make(map[string]int),
	}
}

// Build creates a commit graph from a list of commits
//
// Commits should be in reverse chronological order (newest first)
func (b *GraphBuilder) Build(commits []*commit.Commit) (*CommitGraph, error) {
	for i, c := range commits {
		if err := b.processCommit(c, i); err != nil {
			return nil, err
		}
	}

	return b.graph, nil
}

// processCommit processes a single commit and adds it to the graph
func (b *GraphBuilder) processCommit(c *commit.Commit, index int) error {
	hash, err := c.Hash()
	if err != nil {
		return err
	}
	hashStr := hash.String()

	// Determine lane for this commit
	lane := b.allocateLane(hashStr)

	// Create graph node
	node := &GraphNode{
		Commit:      c,
		Lane:        lane,
		ParentLanes: make([]int, 0, len(c.ParentSHAs)),
		IsMerge:     len(c.ParentSHAs) > 1,
		IsInitial:   len(c.ParentSHAs) == 0,
		Children:    make([]*GraphNode, 0),
		Index:       index,
	}

	// Process parents and allocate lanes for them
	for i, parentSHA := range c.ParentSHAs {
		parentHashStr := parentSHA.String()

		var parentLane int
		if i == 0 {
			// First parent stays in same lane (main line)
			parentLane = lane
			b.reserveLane(parentHashStr, parentLane)
		} else {
			// Additional parents (merge sources) get their own lanes
			parentLane = b.allocateLaneForParent(parentHashStr)
			b.reserveLane(parentHashStr, parentLane)
		}

		node.ParentLanes = append(node.ParentLanes, parentLane)
	}

	// Free current lane if this commit has no parents or first parent is done
	if len(c.ParentSHAs) == 0 {
		b.freeLane(lane)
	} else {
		// Keep lane occupied for first parent
		firstParent := c.ParentSHAs[0].String()
		b.lanes[lane] = firstParent
	}

	// Add node to graph
	b.graph.AddNode(node)

	return nil
}

// allocateLane finds or creates a lane for a commit
func (b *GraphBuilder) allocateLane(commitHash string) int {
	// Check if this commit has a reserved lane
	if reserved, exists := b.reservations[commitHash]; exists {
		delete(b.reservations, commitHash)

		// Make sure lane is available
		if reserved < len(b.lanes) {
			b.lanes[reserved] = commitHash
			return reserved
		}
	}

	// Find first available lane
	for i, occupant := range b.lanes {
		if occupant == "" || occupant == commitHash {
			b.lanes[i] = commitHash
			return i
		}
	}

	// No available lane - create new one
	b.lanes = append(b.lanes, commitHash)
	return len(b.lanes) - 1
}

// allocateLaneForParent allocates a lane for a parent commit (used in merges)
func (b *GraphBuilder) allocateLaneForParent(commitHash string) int {
	// Check if already reserved
	if reserved, exists := b.reservations[commitHash]; exists {
		return reserved
	}

	// Find an available lane (preferably not lane 0)
	for i := 1; i < len(b.lanes); i++ {
		if b.lanes[i] == "" {
			return i
		}
	}

	// Need a new lane
	newLane := len(b.lanes)
	b.lanes = append(b.lanes, "")
	return newLane
}

// reserveLane reserves a lane for a future commit
func (b *GraphBuilder) reserveLane(commitHash string, lane int) {
	b.reservations[commitHash] = lane

	// Ensure lanes slice is large enough
	for len(b.lanes) <= lane {
		b.lanes = append(b.lanes, "")
	}
}

// freeLane marks a lane as available
func (b *GraphBuilder) freeLane(lane int) {
	if lane >= 0 && lane < len(b.lanes) {
		b.lanes[lane] = ""
	}
}

// GetNodeByHash retrieves a graph node by commit hash
func (b *GraphBuilder) GetNodeByHash(hash objects.ObjectHash) *GraphNode {
	return b.graph.GetNode(hash.String())
}
