package merge

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// Rerere (Reuse Recorded Resolution) records conflict resolutions
// and can automatically apply them when the same conflict occurs again.
//
// Directory structure:
//   .git/rr-cache/<hash>/
//     preimage    - conflict with markers
//     postimage   - resolved version
//     thisimage   - current state (for ongoing resolution)

type Rerere struct {
	repo    *sourcerepo.SourceRepository
	enabled bool
}

// NewRerere creates a new rerere instance
func NewRerere(repo *sourcerepo.SourceRepository) *Rerere {
	return &Rerere{
		repo:    repo,
		enabled: true, // TODO: Read from config
	}
}

// IsEnabled returns whether rerere is enabled
func (r *Rerere) IsEnabled() bool {
	return r.enabled
}

// Enable enables rerere
func (r *Rerere) Enable() {
	r.enabled = true
}

// Disable disables rerere
func (r *Rerere) Disable() {
	r.enabled = false
}

// getRerereDir returns the path to the rr-cache directory
func (r *Rerere) getRerereDir() string {
	return filepath.Join(r.repo.SourceDirectory().String(), "rr-cache")
}

// getConflictHash computes a hash for the conflict content
func (r *Rerere) getConflictHash(content []byte) string {
	// Normalize the conflict by removing labels (they may vary)
	normalized := r.normalizeConflict(content)
	hash := sha1.Sum(normalized)
	return hex.EncodeToString(hash[:])
}

// normalizeConflict removes variable parts (labels) from conflict markers
func (r *Rerere) normalizeConflict(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	var normalized []string

	for _, line := range lines {
		if strings.HasPrefix(line, ConflictMarkerStart) {
			normalized = append(normalized, ConflictMarkerStart)
		} else if strings.HasPrefix(line, ConflictMarkerEnd) {
			normalized = append(normalized, ConflictMarkerEnd)
		} else {
			normalized = append(normalized, line)
		}
	}

	return []byte(strings.Join(normalized, "\n"))
}

// RecordConflict records a conflict for later reuse
func (r *Rerere) RecordConflict(ctx context.Context, path scpath.RelativePath, content []byte) error {
	if !r.enabled {
		return nil
	}

	if !HasConflictMarkers(content) {
		return nil // Not a conflict
	}

	hash := r.getConflictHash(content)
	conflictDir := filepath.Join(r.getRerereDir(), hash)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(conflictDir, 0755); err != nil {
		return fmt.Errorf("failed to create rerere directory: %w", err)
	}

	// Write preimage (conflict with markers)
	preimagePath := filepath.Join(conflictDir, "preimage")
	if err := os.WriteFile(preimagePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write preimage: %w", err)
	}

	// Write path information
	pathInfoPath := filepath.Join(conflictDir, "path")
	if err := os.WriteFile(pathInfoPath, []byte(path.String()), 0644); err != nil {
		return fmt.Errorf("failed to write path info: %w", err)
	}

	return nil
}

// RecordResolution records the resolution of a conflict
func (r *Rerere) RecordResolution(ctx context.Context, path scpath.RelativePath, resolved []byte) error {
	if !r.enabled {
		return nil
	}

	// We need to find the conflict hash by matching the path
	// This is a simplified approach - Git uses more sophisticated matching
	rerereDir := r.getRerereDir()
	entries, err := os.ReadDir(rerereDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No rerere data
		}
		return fmt.Errorf("failed to read rerere directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		conflictDir := filepath.Join(rerereDir, entry.Name())
		pathInfoPath := filepath.Join(conflictDir, "path")

		pathData, err := os.ReadFile(pathInfoPath)
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(pathData)) == path.String() {
			// Found the conflict
			postimagePath := filepath.Join(conflictDir, "postimage")
			if err := os.WriteFile(postimagePath, resolved, 0644); err != nil {
				return fmt.Errorf("failed to write postimage: %w", err)
			}
			break
		}
	}

	return nil
}

// TryAutoResolve attempts to automatically resolve a conflict using recorded resolutions
func (r *Rerere) TryAutoResolve(ctx context.Context, path scpath.RelativePath, content []byte) ([]byte, bool, error) {
	if !r.enabled {
		return nil, false, nil
	}

	if !HasConflictMarkers(content) {
		return nil, false, nil
	}

	hash := r.getConflictHash(content)
	conflictDir := filepath.Join(r.getRerereDir(), hash)
	postimagePath := filepath.Join(conflictDir, "postimage")

	// Check if we have a recorded resolution
	resolved, err := os.ReadFile(postimagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil // No recorded resolution
		}
		return nil, false, fmt.Errorf("failed to read postimage: %w", err)
	}

	return resolved, true, nil
}

// Clear removes old rerere records
func (r *Rerere) Clear(ctx context.Context, olderThan time.Duration) error {
	rerereDir := r.getRerereDir()
	entries, err := os.ReadDir(rerereDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read rerere directory: %w", err)
	}

	now := time.Now()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		conflictDir := filepath.Join(rerereDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > olderThan {
			if err := os.RemoveAll(conflictDir); err != nil {
				return fmt.Errorf("failed to remove %s: %w", conflictDir, err)
			}
		}
	}

	return nil
}

// Status returns information about recorded resolutions
func (r *Rerere) Status(ctx context.Context) (*RerereStatus, error) {
	rerereDir := r.getRerereDir()
	entries, err := os.ReadDir(rerereDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &RerereStatus{
				Enabled:      r.enabled,
				RecordedCount: 0,
				Records:      []RerereRecord{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read rerere directory: %w", err)
	}

	var records []RerereRecord
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		conflictDir := filepath.Join(rerereDir, entry.Name())

		// Read path
		pathInfoPath := filepath.Join(conflictDir, "path")
		pathData, err := os.ReadFile(pathInfoPath)
		if err != nil {
			continue
		}

		// Check if resolution exists
		postimagePath := filepath.Join(conflictDir, "postimage")
		_, err = os.Stat(postimagePath)
		hasResolution := err == nil

		// Get timestamp
		info, err := entry.Info()
		var timestamp time.Time
		if err == nil {
			timestamp = info.ModTime()
		}

		records = append(records, RerereRecord{
			Hash:          entry.Name(),
			Path:          strings.TrimSpace(string(pathData)),
			HasResolution: hasResolution,
			Timestamp:     timestamp,
		})
	}

	return &RerereStatus{
		Enabled:       r.enabled,
		RecordedCount: len(records),
		Records:       records,
	}, nil
}

// Forget removes a recorded resolution for a path
func (r *Rerere) Forget(ctx context.Context, path scpath.RelativePath) error {
	rerereDir := r.getRerereDir()
	entries, err := os.ReadDir(rerereDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read rerere directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		conflictDir := filepath.Join(rerereDir, entry.Name())
		pathInfoPath := filepath.Join(conflictDir, "path")

		pathData, err := os.ReadFile(pathInfoPath)
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(pathData)) == path.String() {
			if err := os.RemoveAll(conflictDir); err != nil {
				return fmt.Errorf("failed to remove rerere record: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("no rerere record found for path: %s", path)
}

// RerereStatus represents the status of rerere
type RerereStatus struct {
	Enabled       bool
	RecordedCount int
	Records       []RerereRecord
}

// RerereRecord represents a single recorded resolution
type RerereRecord struct {
	Hash          string
	Path          string
	HasResolution bool
	Timestamp     time.Time
}
