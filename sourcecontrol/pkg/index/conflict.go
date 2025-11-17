package index

import (
	"fmt"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// ConflictEntry represents a three-way merge conflict for a single file
type ConflictEntry struct {
	Path   scpath.RelativePath
	Base   objects.ObjectHash // Stage 1: common ancestor
	Ours   objects.ObjectHash // Stage 2: our version (HEAD)
	Theirs objects.ObjectHash // Stage 3: their version (merging branch)
}

// AddConflict adds a conflict to the index with three stages:
// - Stage 1 (base): common ancestor version
// - Stage 2 (ours): current branch version (HEAD)
// - Stage 3 (theirs): version from the branch being merged
//
// This removes any existing normal entry (stage 0) for the path.
func (idx *Index) AddConflict(path scpath.RelativePath, base, ours, theirs objects.ObjectHash) error {
	normalizedPath := path.Normalize()

	// Remove any existing stage 0 entry
	idx.Remove(normalizedPath)

	// Remove any existing conflict entries for this path
	idx.RemoveConflict(normalizedPath)

	// Create base entry (stage 1) if it exists
	if !base.IsZero() {
		baseEntry := &Entry{
			Stage: 1,
			Path:  path,
			BlobHash:  base,
		}
		idx.Entries = append(idx.Entries, baseEntry)
		// Don't add to entryMap - conflicts are handled specially
	}

	// Create ours entry (stage 2) if it exists
	if !ours.IsZero() {
		oursEntry := &Entry{
			Stage: 2,
			Path:  path,
			BlobHash:  ours,
		}
		idx.Entries = append(idx.Entries, oursEntry)
	}

	// Create theirs entry (stage 3) if it exists
	if !theirs.IsZero() {
		theirsEntry := &Entry{
			Stage: 3,
			Path:  path,
			BlobHash:  theirs,
		}
		idx.Entries = append(idx.Entries, theirsEntry)
	}

	// Re-sort entries to maintain index order
	idx.sort()

	return nil
}

// RemoveConflict removes all conflict entries (stages 1-3) for a path.
// This is typically used after manually resolving a conflict.
func (idx *Index) RemoveConflict(path scpath.RelativePath) {
	normalizedPath := path.Normalize()

	// Remove all entries with stages 1, 2, or 3 for this path
	filtered := make([]*Entry, 0, len(idx.Entries))
	for _, entry := range idx.Entries {
		if entry.Path.Normalize() == normalizedPath && entry.Stage >= 1 && entry.Stage <= 3 {
			// Skip conflict entries
			continue
		}
		filtered = append(filtered, entry)
	}
	idx.Entries = filtered
}

// GetConflicts returns all conflicts in the index.
// Returns a map where the key is the normalized path and the value is the conflict entry.
func (idx *Index) GetConflicts() map[scpath.RelativePath]*ConflictEntry {
	conflicts := make(map[scpath.RelativePath]*ConflictEntry)

	for _, entry := range idx.Entries {
		if entry.Stage == 0 {
			continue
		}

		normalizedPath := entry.Path.Normalize()
		conflict, exists := conflicts[normalizedPath]
		if !exists {
			conflict = &ConflictEntry{
				Path: entry.Path,
			}
			conflicts[normalizedPath] = conflict
		}

		switch entry.Stage {
		case 1:
			conflict.Base = entry.BlobHash
		case 2:
			conflict.Ours = entry.BlobHash
		case 3:
			conflict.Theirs = entry.BlobHash
		}
	}

	return conflicts
}

// GetConflict returns the conflict entry for a specific path, if it exists.
func (idx *Index) GetConflict(path scpath.RelativePath) (*ConflictEntry, bool) {
	normalizedPath := path.Normalize()
	conflicts := idx.GetConflicts()
	conflict, exists := conflicts[normalizedPath]
	return conflict, exists
}

// HasConflicts returns true if there are any unresolved conflicts in the index.
func (idx *Index) HasConflicts() bool {
	for _, entry := range idx.Entries {
		if entry.Stage > 0 {
			return true
		}
	}
	return false
}

// IsConflicted returns true if the specified path has a conflict.
func (idx *Index) IsConflicted(path scpath.RelativePath) bool {
	normalizedPath := path.Normalize()
	for _, entry := range idx.Entries {
		if entry.Path.Normalize() == normalizedPath && entry.Stage > 0 {
			return true
		}
	}
	return false
}

// GetConflictedPaths returns a list of all paths with conflicts.
func (idx *Index) GetConflictedPaths() []scpath.RelativePath {
	conflicts := idx.GetConflicts()
	paths := make([]scpath.RelativePath, 0, len(conflicts))
	for path := range conflicts {
		paths = append(paths, path)
	}
	return paths
}

// ResolveConflict marks a conflict as resolved by:
// 1. Removing all conflict entries (stages 1-3)
// 2. Adding a new stage 0 entry with the resolved content
func (idx *Index) ResolveConflict(path scpath.RelativePath, resolvedHash objects.ObjectHash) error {
	if !idx.IsConflicted(path) {
		return fmt.Errorf("no conflict for path: %s", path)
	}

	// Remove conflict entries
	idx.RemoveConflict(path)

	// Add resolved entry as stage 0
	resolvedEntry := &Entry{
		Stage: 0,
		Path:  path,
		BlobHash:  resolvedHash,
	}
	idx.Add(resolvedEntry)

	return nil
}

// GetConflictCount returns the number of conflicted files.
func (idx *Index) GetConflictCount() int {
	return len(idx.GetConflicts())
}
