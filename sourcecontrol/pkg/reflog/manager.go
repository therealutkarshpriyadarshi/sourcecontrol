package reflog

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/config"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
)

// Manager handles reflog operations
type Manager struct {
	repo *sourcerepo.SourceRepository
}

// NewManager creates a new reflog manager
func NewManager(repo *sourcerepo.SourceRepository) *Manager {
	return &Manager{
		repo: repo,
	}
}

// Append adds a new entry to the reflog for the specified reference
func (m *Manager) Append(ref RefName, oldHash, newHash objects.ObjectHash, message string) error {
	// Get committer info from config
	cfgMgr := config.NewManager(m.repo.WorkingDirectory())
	committerName := "Unknown"
	if entry := cfgMgr.Get("user.name"); entry != nil {
		committerName = entry.Value
	}
	committerEmail := "unknown@example.com"
	if entry := cfgMgr.Get("user.email"); entry != nil {
		committerEmail = entry.Value
	}

	entry := &Entry{
		OldHash:        oldHash,
		NewHash:        newHash,
		Committer:      committerName,
		CommitterEmail: committerEmail,
		Timestamp:      time.Now(),
		Message:        message,
	}

	return m.appendEntry(ref, entry)
}

// Read returns all reflog entries for the specified reference
func (m *Manager) Read(ref RefName) ([]*Entry, error) {
	reflogFile := m.getReflogPath(ref)
	if _, err := os.Stat(reflogFile); os.IsNotExist(err) {
		return []*Entry{}, nil
	}

	file, err := os.Open(reflogFile)
	if err != nil {
		return nil, fmt.Errorf("open reflog file: %w", err)
	}
	defer file.Close()

	var entries []*Entry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		entry, err := m.parseLine(line)
		if err != nil {
			// Skip malformed lines
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read reflog file: %w", err)
	}

	// Reverse to show newest first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

// ReadHead returns reflog entries for HEAD
func (m *Manager) ReadHead() ([]*Entry, error) {
	return m.Read(RefHead)
}

// ReadBranch returns reflog entries for a branch
func (m *Manager) ReadBranch(branchName string) ([]*Entry, error) {
	return m.Read(BranchRef(branchName))
}

// Clear removes all reflog entries for the specified reference
func (m *Manager) Clear(ref RefName) error {
	reflogFile := m.getReflogPath(ref)
	if err := os.Remove(reflogFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove reflog file: %w", err)
	}
	return nil
}

// Expire removes reflog entries older than the specified duration
func (m *Manager) Expire(ref RefName, maxAge time.Duration) error {
	entries, err := m.Read(ref)
	if err != nil {
		return fmt.Errorf("read reflog: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	var kept []*Entry

	for _, entry := range entries {
		if entry.Timestamp.After(cutoff) {
			kept = append(kept, entry)
		}
	}

	// Rewrite the reflog with only kept entries
	return m.writeEntries(ref, kept)
}

// Helper methods

func (m *Manager) getReflogPath(ref RefName) string {
	refPath := string(ref)
	if refPath == "HEAD" {
		return filepath.Join(m.repo.WorkingDirectory().String(), scpath.SourceDir, "logs", "HEAD")
	}
	return filepath.Join(m.repo.WorkingDirectory().String(), scpath.SourceDir, "logs", refPath)
}

func (m *Manager) appendEntry(ref RefName, entry *Entry) error {
	reflogFile := m.getReflogPath(ref)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(reflogFile), 0755); err != nil {
		return fmt.Errorf("create reflog directory: %w", err)
	}

	file, err := os.OpenFile(reflogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open reflog file: %w", err)
	}
	defer file.Close()

	line := m.formatLine(entry)
	if _, err := file.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("write reflog entry: %w", err)
	}

	return nil
}

func (m *Manager) writeEntries(ref RefName, entries []*Entry) error {
	reflogFile := m.getReflogPath(ref)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(reflogFile), 0755); err != nil {
		return fmt.Errorf("create reflog directory: %w", err)
	}

	file, err := os.Create(reflogFile)
	if err != nil {
		return fmt.Errorf("create reflog file: %w", err)
	}
	defer file.Close()

	// Reverse entries to write oldest first
	reversed := make([]*Entry, len(entries))
	for i, j := 0, len(entries)-1; i < len(entries); i, j = i+1, j-1 {
		reversed[i] = entries[j]
	}

	for _, entry := range reversed {
		line := m.formatLine(entry)
		if _, err := file.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("write reflog entry: %w", err)
		}
	}

	return nil
}

func (m *Manager) formatLine(entry *Entry) string {
	// Format: <old-hash> <new-hash> <committer> <email> <timestamp> <timezone> <message>
	return fmt.Sprintf("%s %s %s <%s> %d +0000\t%s",
		entry.OldHash,
		entry.NewHash,
		entry.Committer,
		entry.CommitterEmail,
		entry.Timestamp.Unix(),
		entry.Message,
	)
}

func (m *Manager) parseLine(line string) (*Entry, error) {
	// Split on tab first to separate metadata from message
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid reflog line format")
	}

	metadata := parts[0]
	message := parts[1]

	// Parse metadata: <old-hash> <new-hash> <committer> <email> <timestamp> <timezone>
	fields := strings.Fields(metadata)
	if len(fields) < 5 {
		return nil, fmt.Errorf("invalid metadata format")
	}

	oldHash := objects.ObjectHash(fields[0])
	newHash := objects.ObjectHash(fields[1])
	committer := fields[2]

	// Email is in angle brackets
	email := ""
	for i := 3; i < len(fields); i++ {
		if strings.HasPrefix(fields[i], "<") {
			email = strings.TrimPrefix(fields[i], "<")
			email = strings.TrimSuffix(email, ">")
			break
		}
	}

	// Find timestamp (should be after email)
	timestampStr := ""
	for i := 3; i < len(fields); i++ {
		if strings.HasPrefix(fields[i], "<") {
			if i+1 < len(fields) {
				timestampStr = fields[i+1]
			}
			break
		}
	}

	timestamp := time.Now()
	if ts, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		timestamp = time.Unix(ts, 0)
	}

	return &Entry{
		OldHash:        oldHash,
		NewHash:        newHash,
		Committer:      committer,
		CommitterEmail: email,
		Timestamp:      timestamp,
		Message:        message,
	}, nil
}
