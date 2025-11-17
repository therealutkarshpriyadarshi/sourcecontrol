package merge

import (
	"bytes"
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// ConflictMarker contains the conflict markers used in files
type ConflictMarker struct {
	Start  string // "<<<<<<< HEAD" or "<<<<<<< ours"
	Middle string // "======="
	End    string // ">>>>>>> branch-name" or ">>>>>>> theirs"
}

// DefaultConflictMarker returns the default Git-style conflict markers
func DefaultConflictMarker(ourLabel, theirLabel string) *ConflictMarker {
	return &ConflictMarker{
		Start:  fmt.Sprintf("<<<<<<< %s", ourLabel),
		Middle: "=======",
		End:    fmt.Sprintf(">>>>>>> %s", theirLabel),
	}
}

// ConflictResolver handles conflict resolution
type ConflictResolver struct {
	conflicts []Conflict
	strategy  ConflictResolution
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(strategy ConflictResolution) *ConflictResolver {
	return &ConflictResolver{
		conflicts: make([]Conflict, 0),
		strategy:  strategy,
	}
}

// AddConflict registers a new conflict
func (cr *ConflictResolver) AddConflict(conflict Conflict) {
	cr.conflicts = append(cr.conflicts, conflict)
}

// HasConflicts returns true if there are any conflicts
func (cr *ConflictResolver) HasConflicts() bool {
	return len(cr.conflicts) > 0
}

// GetConflicts returns all registered conflicts
func (cr *ConflictResolver) GetConflicts() []Conflict {
	return cr.conflicts
}

// Resolve resolves a conflict based on the configured strategy
func (cr *ConflictResolver) Resolve(conflict *Conflict) ([]byte, error) {
	switch cr.strategy {
	case ConflictOurs:
		return conflict.OurVersion, nil
	case ConflictTheirs:
		return conflict.TheirVersion, nil
	case ConflictManual:
		return cr.createConflictMarkers(conflict), nil
	case ConflictFail:
		return nil, fmt.Errorf("conflict in %s", conflict.Path)
	default:
		return nil, fmt.Errorf("unknown conflict resolution strategy")
	}
}

// createConflictMarkers creates a file with conflict markers
func (cr *ConflictResolver) createConflictMarkers(conflict *Conflict) []byte {
	var buf bytes.Buffer

	marker := DefaultConflictMarker("HEAD", conflict.TheirSHA.Short().String())

	buf.WriteString(marker.Start)
	buf.WriteString("\n")
	buf.Write(conflict.OurVersion)
	if len(conflict.OurVersion) > 0 && conflict.OurVersion[len(conflict.OurVersion)-1] != '\n' {
		buf.WriteString("\n")
	}
	buf.WriteString(marker.Middle)
	buf.WriteString("\n")
	buf.Write(conflict.TheirVersion)
	if len(conflict.TheirVersion) > 0 && conflict.TheirVersion[len(conflict.TheirVersion)-1] != '\n' {
		buf.WriteString("\n")
	}
	buf.WriteString(marker.End)
	buf.WriteString("\n")

	return buf.Bytes()
}

// ConflictPaths returns a list of paths with conflicts
func (cr *ConflictResolver) ConflictPaths() []scpath.RelativePath {
	paths := make([]scpath.RelativePath, len(cr.conflicts))
	for i, c := range cr.conflicts {
		paths[i] = c.Path
	}
	return paths
}

// MergeContent performs a three-way merge on file content
func MergeContent(base, ours, theirs []byte) ([]byte, bool) {
	// Simple implementation: if content is the same, no conflict
	if bytes.Equal(ours, theirs) {
		return ours, true
	}

	// If base equals ours, use theirs (they changed it)
	if bytes.Equal(base, ours) {
		return theirs, true
	}

	// If base equals theirs, use ours (we changed it)
	if bytes.Equal(base, theirs) {
		return ours, true
	}

	// Both changed differently - conflict
	return nil, false
}

// LineBasedMerge performs a line-based three-way merge
func LineBasedMerge(base, ours, theirs []byte) ([]byte, []ConflictRegion, error) {
	baseLines := splitLines(base)
	ourLines := splitLines(ours)
	theirLines := splitLines(theirs)

	// This is a simplified implementation
	// A full implementation would use a diff3 algorithm
	if bytes.Equal(ours, theirs) {
		return ours, nil, nil
	}

	// For now, if there's any difference, mark as conflict
	conflicts := []ConflictRegion{
		{
			BaseStart:  0,
			BaseEnd:    len(baseLines),
			OurStart:   0,
			OurEnd:     len(ourLines),
			TheirStart: 0,
			TheirEnd:   len(theirLines),
		},
	}

	return nil, conflicts, nil
}

// ConflictRegion represents a region of conflicting lines
type ConflictRegion struct {
	BaseStart  int
	BaseEnd    int
	OurStart   int
	OurEnd     int
	TheirStart int
	TheirEnd   int
}

// splitLines splits content into lines
func splitLines(content []byte) [][]byte {
	return bytes.Split(content, []byte("\n"))
}
