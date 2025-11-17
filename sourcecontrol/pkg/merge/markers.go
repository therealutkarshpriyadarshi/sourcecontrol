package merge

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

const (
	// ConflictMarkerStart marks the beginning of "ours" section
	ConflictMarkerStart = "<<<<<<<"

	// ConflictMarkerSeparator separates "ours" from "theirs"
	ConflictMarkerSeparator = "======="

	// ConflictMarkerEnd marks the end of "theirs" section
	ConflictMarkerEnd = ">>>>>>>"

	// ConflictMarkerBase marks the beginning of "base" section (for diff3 style)
	ConflictMarkerBase = "|||||||"
)

// ParsedConflictMarker represents a parsed conflict marker in a file
type ParsedConflictMarker struct {
	// StartLine is the line number where the conflict starts (0-indexed)
	StartLine int

	// EndLine is the line number where the conflict ends (0-indexed)
	EndLine int

	// OursLabel is the label for our version (e.g., "HEAD")
	OursLabel string

	// TheirsLabel is the label for their version (e.g., "feature-branch")
	TheirsLabel string

	// BaseLabel is the label for the base version (optional, for diff3 style)
	BaseLabel string

	// OursContent is the content from our version
	OursContent []string

	// TheirsContent is the content from their version
	TheirsContent []string

	// BaseContent is the content from the base version (optional, for diff3 style)
	BaseContent []string
}

// CreateConflictMarkers creates a file with conflict markers from three versions.
//
// Parameters:
//   - base: content from the common ancestor
//   - ours: content from our branch (HEAD)
//   - theirs: content from the branch being merged
//   - oursLabel: label for our version (e.g., "HEAD")
//   - theirsLabel: label for their version (e.g., "feature-branch")
//   - diff3Style: if true, include base content between ours and theirs
//
// Returns the merged content with conflict markers.
func CreateConflictMarkers(base, ours, theirs []byte, oursLabel, theirsLabel string, diff3Style bool) []byte {
	var result bytes.Buffer

	// Write our version
	result.WriteString(ConflictMarkerStart + " " + oursLabel + "\n")
	result.Write(ours)
	if !bytes.HasSuffix(ours, []byte("\n")) {
		result.WriteString("\n")
	}

	// Write base version if diff3 style
	if diff3Style && len(base) > 0 {
		result.WriteString(ConflictMarkerBase + " base\n")
		result.Write(base)
		if !bytes.HasSuffix(base, []byte("\n")) {
			result.WriteString("\n")
		}
	}

	// Write separator
	result.WriteString(ConflictMarkerSeparator + "\n")

	// Write their version
	result.Write(theirs)
	if !bytes.HasSuffix(theirs, []byte("\n")) {
		result.WriteString("\n")
	}
	result.WriteString(ConflictMarkerEnd + " " + theirsLabel + "\n")

	return result.Bytes()
}

// ParseConflictMarkers parses a file and returns all conflict markers found.
func ParseConflictMarkers(content []byte) ([]*ParsedConflictMarker, error) {
	var conflicts []*ParsedConflictMarker
	var currentConflict *ParsedConflictMarker
	var currentSection string // "ours", "base", "theirs", or ""

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Check for conflict markers
		if strings.HasPrefix(line, ConflictMarkerStart) {
			if currentConflict != nil {
				return nil, fmt.Errorf("nested conflict markers at line %d", lineNum)
			}
			currentConflict = &ParsedConflictMarker{
				StartLine:    lineNum - 1, // 0-indexed
				OursContent:  make([]string, 0),
				TheirsContent: make([]string, 0),
				BaseContent:  make([]string, 0),
			}
			// Extract label
			parts := strings.SplitN(line, " ", 2)
			if len(parts) > 1 {
				currentConflict.OursLabel = strings.TrimSpace(parts[1])
			}
			currentSection = "ours"
			continue
		}

		if strings.HasPrefix(line, ConflictMarkerBase) {
			if currentConflict == nil {
				return nil, fmt.Errorf("conflict base marker without start at line %d", lineNum)
			}
			// Extract label
			parts := strings.SplitN(line, " ", 2)
			if len(parts) > 1 {
				currentConflict.BaseLabel = strings.TrimSpace(parts[1])
			}
			currentSection = "base"
			continue
		}

		if strings.HasPrefix(line, ConflictMarkerSeparator) {
			if currentConflict == nil {
				return nil, fmt.Errorf("conflict separator without start at line %d", lineNum)
			}
			currentSection = "theirs"
			continue
		}

		if strings.HasPrefix(line, ConflictMarkerEnd) {
			if currentConflict == nil {
				return nil, fmt.Errorf("conflict end marker without start at line %d", lineNum)
			}
			// Extract label
			parts := strings.SplitN(line, " ", 2)
			if len(parts) > 1 {
				currentConflict.TheirsLabel = strings.TrimSpace(parts[1])
			}
			currentConflict.EndLine = lineNum - 1 // 0-indexed
			conflicts = append(conflicts, currentConflict)
			currentConflict = nil
			currentSection = ""
			continue
		}

		// Add line to current section
		if currentConflict != nil {
			switch currentSection {
			case "ours":
				currentConflict.OursContent = append(currentConflict.OursContent, line)
			case "base":
				currentConflict.BaseContent = append(currentConflict.BaseContent, line)
			case "theirs":
				currentConflict.TheirsContent = append(currentConflict.TheirsContent, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file: %w", err)
	}

	if currentConflict != nil {
		return nil, fmt.Errorf("unclosed conflict marker starting at line %d", currentConflict.StartLine+1)
	}

	return conflicts, nil
}

// HasConflictMarkers checks if the content contains any conflict markers.
func HasConflictMarkers(content []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, ConflictMarkerStart) ||
			strings.HasPrefix(line, ConflictMarkerSeparator) ||
			strings.HasPrefix(line, ConflictMarkerEnd) {
			return true
		}
	}
	return false
}

// RemoveConflictMarkers removes all conflict markers from content,
// keeping only the specified version ("ours", "theirs", or "base").
//
// If keepVersion is empty or invalid, it returns an error.
func RemoveConflictMarkers(content []byte, keepVersion string) ([]byte, error) {
	conflicts, err := ParseConflictMarkers(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse conflict markers: %w", err)
	}

	if len(conflicts) == 0 {
		return content, nil // No conflicts to remove
	}

	var result bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	conflictIdx := 0
	inConflict := false

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Check if we're entering a conflict
		if conflictIdx < len(conflicts) && lineNum-1 == conflicts[conflictIdx].StartLine {
			inConflict = true

			// Write the content we want to keep
			var linesToKeep []string
			switch keepVersion {
			case "ours":
				linesToKeep = conflicts[conflictIdx].OursContent
			case "theirs":
				linesToKeep = conflicts[conflictIdx].TheirsContent
			case "base":
				linesToKeep = conflicts[conflictIdx].BaseContent
			default:
				return nil, fmt.Errorf("invalid keep version: %s (must be 'ours', 'theirs', or 'base')", keepVersion)
			}

			for _, l := range linesToKeep {
				result.WriteString(l + "\n")
			}
		}

		// Check if we're exiting a conflict
		if inConflict && conflictIdx < len(conflicts) && lineNum-1 == conflicts[conflictIdx].EndLine {
			inConflict = false
			conflictIdx++
			continue // Skip the end marker line
		}

		// If not in a conflict, write the line
		if !inConflict {
			result.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file: %w", err)
	}

	return result.Bytes(), nil
}

// WriteConflictedFile writes a file with conflict markers by performing a three-way merge.
//
// Parameters:
//   - w: writer to output the conflicted content
//   - base: content from common ancestor
//   - ours: content from our branch
//   - theirs: content from their branch
//   - oursLabel: label for our version
//   - theirsLabel: label for their version
//   - diff3Style: include base content in conflicts
func WriteConflictedFile(w io.Writer, base, ours, theirs []byte, oursLabel, theirsLabel string, diff3Style bool) error {
	// For now, we'll use a simple approach: if the files differ, create a conflict marker
	// A more sophisticated implementation would use diff3 algorithm to merge non-conflicting changes

	if bytes.Equal(ours, theirs) {
		// No conflict - files are identical
		_, err := w.Write(ours)
		return err
	}

	// Files differ - create conflict markers
	conflicted := CreateConflictMarkers(base, ours, theirs, oursLabel, theirsLabel, diff3Style)
	_, err := w.Write(conflicted)
	return err
}
