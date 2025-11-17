package graph

import (
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
)

func TestGraphBuilder_LinearHistory(t *testing.T) {
	// Create a linear history: C <- B <- A (A is newest)
	commits := createLinearHistory(3)

	builder := NewGraphBuilder()
	graph, err := builder.Build(commits)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}

	// All commits should be in lane 0 for linear history
	for i, node := range graph.Nodes {
		if node.Lane != 0 {
			t.Errorf("commit %d: expected lane 0, got %d", i, node.Lane)
		}
	}

	// Check graph dimensions
	if graph.Width() != 1 {
		t.Errorf("expected width 1, got %d", graph.Width())
	}

	if graph.Height() != 3 {
		t.Errorf("expected height 3, got %d", graph.Height())
	}
}

func TestGraphBuilder_MergeCommit(t *testing.T) {
	// Create a simple merge:
	//   A (merge of B and C)
	//   |\
	//   B C
	//   |/
	//   D
	commits := createMergeHistory()

	builder := NewGraphBuilder()
	graph, err := builder.Build(commits)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}

	// Merge commit should have IsMerge = true
	mergeNode := graph.Nodes[0] // First commit is the merge
	if !mergeNode.IsMerge {
		t.Error("expected merge commit to have IsMerge = true")
	}

	// Merge commit should have 2 parent lanes
	if len(mergeNode.ParentLanes) != 2 {
		t.Errorf("expected 2 parent lanes, got %d", len(mergeNode.ParentLanes))
	}

	// Graph should use at least 2 lanes
	if graph.Width() < 2 {
		t.Errorf("expected width >= 2 for merge, got %d", graph.Width())
	}
}

func TestGraphBuilder_InitialCommit(t *testing.T) {
	// Create a single initial commit
	commits := createLinearHistory(1)

	builder := NewGraphBuilder()
	graph, err := builder.Build(commits)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}

	if len(graph.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(graph.Nodes))
	}

	node := graph.Nodes[0]
	if !node.IsInitial {
		t.Error("expected IsInitial = true for initial commit")
	}

	if len(node.ParentLanes) != 0 {
		t.Errorf("expected 0 parent lanes, got %d", len(node.ParentLanes))
	}
}

func TestGraphBuilder_EmptyHistory(t *testing.T) {
	commits := []*commit.Commit{}

	builder := NewGraphBuilder()
	graph, err := builder.Build(commits)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}

	if graph.Height() != 0 {
		t.Errorf("expected height 0, got %d", graph.Height())
	}

	if len(graph.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(graph.Nodes))
	}
}

func TestGraphBuilder_GetNodeByHash(t *testing.T) {
	commits := createLinearHistory(3)

	builder := NewGraphBuilder()
	_, err := builder.Build(commits)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}

	// Try to get first commit by hash
	firstCommit := commits[0]
	hash, err := firstCommit.Hash()
	if err != nil {
		t.Fatalf("failed to get hash: %v", err)
	}

	node := builder.GetNodeByHash(hash)
	if node == nil {
		t.Error("expected to find node by hash")
	}

	if node.Commit != firstCommit {
		t.Error("got wrong commit from hash lookup")
	}
}

// Helper functions to create test commits

func createLinearHistory(count int) []*commit.Commit {
	commits := make([]*commit.Commit, count)
	var prevHash objects.ObjectHash

	for i := count - 1; i >= 0; i-- {
		parents := make([]objects.ObjectHash, 0)
		if prevHash != "" {
			parents = append(parents, prevHash)
		}

		c := createTestCommit(parents, "Commit "+string(rune('A'+count-1-i)))
		commits[i] = c

		hash, _ := c.Hash()
		prevHash = hash
	}

	return commits
}

func createMergeHistory() []*commit.Commit {
	// Create: D -> B -> A (merge)
	//              \-> C /

	// Initial commit D
	commitD := createTestCommit([]objects.ObjectHash{}, "Commit D")
	hashD, _ := commitD.Hash()

	// Commit B (child of D)
	commitB := createTestCommit([]objects.ObjectHash{hashD}, "Commit B")
	hashB, _ := commitB.Hash()

	// Commit C (child of D, parallel to B)
	commitC := createTestCommit([]objects.ObjectHash{hashD}, "Commit C")
	hashC, _ := commitC.Hash()

	// Commit A (merge of B and C)
	commitA := createTestCommit([]objects.ObjectHash{hashB, hashC}, "Commit A (merge)")

	return []*commit.Commit{commitA, commitB, commitC, commitD}
}

func createTestCommit(parents []objects.ObjectHash, message string) *commit.Commit {
	author, _ := commit.NewCommitPerson("Test User", "test@example.com", time.Now())

	c, _ := commit.NewCommitBuilder().
		TreeHash(objects.ObjectHash("0000000000000000000000000000000000000000")).
		ParentHashes(parents...).
		Author(author).
		Committer(author).
		Message(message).
		Build()

	return c
}

func TestCommitGraph_AddNode(t *testing.T) {
	graph := NewCommitGraph()

	node := &GraphNode{
		Commit: createTestCommit([]objects.ObjectHash{}, "Test"),
		Lane:   2,
	}

	graph.AddNode(node)

	if len(graph.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(graph.Nodes))
	}

	if graph.MaxLane != 2 {
		t.Errorf("expected MaxLane 2, got %d", graph.MaxLane)
	}

	if graph.Width() != 3 {
		t.Errorf("expected width 3, got %d", graph.Width())
	}
}

func TestCommitGraph_GetNode(t *testing.T) {
	graph := NewCommitGraph()

	c := createTestCommit([]objects.ObjectHash{}, "Test")
	hash, _ := c.Hash()

	node := &GraphNode{
		Commit: c,
		Lane:   0,
	}

	graph.AddNode(node)

	retrieved := graph.GetNode(hash.String())
	if retrieved == nil {
		t.Error("expected to retrieve node")
	}

	if retrieved != node {
		t.Error("retrieved wrong node")
	}
}
