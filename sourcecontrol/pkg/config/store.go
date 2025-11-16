package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/utkarsh5026/SourceControl/pkg/common/fileops"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// Store handles reading and writing JSON configuration files.
// It provides atomic write operations to prevent corruption and manages
// configuration entries at different levels (system, global, local).
//
// The Store uses a map of keys to slices of ConfigEntry to support
// multi-valued configuration keys. All read operations return copies
// to prevent external mutation of internal state.
//
// Example usage:
//
//	store := NewStore(path, ConfigLevelLocal)
//	if err := store.Load(); err != nil {
//	    return err
//	}
//	store.Set("user.name", "John Doe")
//	store.Add("remote.origin.url", "https://github.com/user/repo")
//	if err := store.Save(); err != nil {
//	    return err
//	}
type Store struct {
	path    scpath.AbsolutePath       // Absolute path to the configuration file
	level   ConfigLevel               // Configuration level (system/global/local)
	entries map[string][]*ConfigEntry // Key-value pairs supporting multi-valued keys
	parser  *Parser                   // Parser for JSON serialization/deserialization
}

// NewStore creates a new configuration store for a specific file and level.
//
// Parameters:
//   - path: Absolute path where the configuration file will be stored
//   - level: Configuration level (system, global, or local)
//
// Returns:
//   - *Store: A new Store instance with empty entries
func NewStore(path scpath.AbsolutePath, level ConfigLevel) *Store {
	return &Store{
		path:    path,
		level:   level,
		entries: make(map[string][]*ConfigEntry),
		parser:  &Parser{},
	}
}

// Load reads and parses the configuration file from disk.
//
// If the file doesn't exist, it initializes an empty configuration without
// returning an error. This is expected behavior for new configurations.
//
// If the file exists but contains invalid JSON, it logs warnings to stderr
// and initializes an empty configuration. This ensures the system remains
// operational even with corrupted configuration files.
//
// Returns:
//   - error: Only for actual file system or critical parsing failures
//     Returns nil for non-existent files or invalid JSON (after warnings)
func (s *Store) Load() error {
	content, err := fileops.ReadBytes(s.path)
	if err != nil {
		return NewConfigError("load", CodeNotFoundErr, "", s.path.String(), "", err)
	}

	if content == nil {
		s.entries = make(map[string][]*ConfigEntry)
		return nil
	}

	if v := s.parser.Validate(string(content)); !v.Valid {
		fmt.Fprintf(os.Stderr, "Warning: Invalid configuration in %s:\n", s.path.String())
		fmt.Fprintf(os.Stderr, "  %s\n", strings.Join(v.Errors, "\n  "))
		s.entries = make(map[string][]*ConfigEntry)
		return nil
	}

	entries, err := s.parser.Parse(string(content), NewFileSource(s.path), s.level)
	if err != nil {
		return NewInvalidFormatError("load", s.path.String(), err)
	}

	s.entries = entries
	return nil
}

// Save writes the configuration to disk atomically.
//
// Uses the write-to-temp-then-rename pattern to ensure atomic updates.
// This prevents configuration corruption if the write operation is interrupted.
// The parent directory is automatically created if it doesn't exist.
//
// The file is written with permissions 0644 (readable by all, writable by owner).
//
// Returns:
//   - error: If serialization fails, parent directory creation fails,
//     or the atomic write operation fails
func (s *Store) Save() error {
	content, err := s.parser.Serialize(s.entries)
	if err != nil {
		return NewInvalidFormatError("save", s.path.String(), err)
	}

	if err := fileops.EnsureParentDir(s.path); err != nil {
		return NewInvalidFormatError("save", s.path.String(), err)
	}

	if err := fileops.AtomicWrite(s.path, []byte(content), 0644); err != nil {
		return NewInvalidFormatError("save", s.path.String(), err)
	}

	return nil
}

// GetEntries returns all entries for a specific key.
//
// Returns a deep copy of entries to prevent external mutation.
// If the key doesn't exist, returns an empty slice (not nil).
//
// Parameters:
//   - key: The configuration key to retrieve
//
// Returns:
//   - []*ConfigEntry: Copy of all entries for the key, or empty slice if not found
func (s *Store) GetEntries(key string) []*ConfigEntry {
	entries, exists := s.entries[key]
	if !exists {
		return []*ConfigEntry{}
	}

	result := make([]*ConfigEntry, len(entries))
	for i, entry := range entries {
		result[i] = entry.Clone()
	}
	return result
}

// GetAllEntries returns a deep copy of all configuration entries.
//
// This is useful for exporting or inspecting the entire configuration.
// The returned map is a complete copy and can be safely modified without
// affecting the Store's internal state.
//
// Returns:
//   - map[string][]*ConfigEntry: Copy of all entries indexed by key
func (s *Store) GetAllEntries() map[string][]*ConfigEntry {
	result := make(map[string][]*ConfigEntry, len(s.entries))
	for key, entries := range s.entries {
		result[key] = make([]*ConfigEntry, len(entries))
		for i, entry := range entries {
			result[key][i] = entry.Clone()
		}
	}
	return result
}

// Set replaces all values for a key with a single value.
//
// Any existing values for the key are removed. Use Add() to append
// to multi-valued keys instead.
//
// Parameters:
//   - key: The configuration key to set
//   - value: The new value for the key
func (s *Store) Set(key, value string) {
	entry := NewEntry(key, value, s.level, NewFileSource(s.path), 0)
	s.entries[key] = []*ConfigEntry{entry}
}

// Add appends a value to a multi-value key.
//
// If the key doesn't exist, it will be created. This is useful for
// configuration keys that support multiple values, such as remote URLs.
//
// Parameters:
//   - key: The configuration key to add to
//   - value: The value to append
func (s *Store) Add(key, value string) {
	if _, exists := s.entries[key]; !exists {
		s.entries[key] = []*ConfigEntry{}
	}
	entry := NewEntry(key, value, s.level, NewFileSource(s.path), 0)
	s.entries[key] = append(s.entries[key], entry)
}

// Unset removes all values for a key from the configuration.
//
// If the key doesn't exist, this is a no-op.
//
// Parameters:
//   - key: The configuration key to remove
func (s *Store) Unset(key string) {
	delete(s.entries, key)
}

// ToJSON exports the configuration as a formatted JSON string.
//
// The output is formatted for human readability with proper indentation.
//
// Returns:
//   - string: JSON representation of the configuration
//   - error: If serialization fails
func (s *Store) ToJSON() (string, error) {
	return s.parser.FormatForDisplay(s.entries)
}

// FromJSON imports configuration from a JSON string.
//
// The JSON is validated before parsing. If validation fails, an error
// is returned and the Store's existing entries are not modified.
//
// Parameters:
//   - jsonContent: JSON string containing configuration data
//
// Returns:
//   - error: If JSON is invalid or parsing fails
func (s *Store) FromJSON(jsonContent string) error {
	v := s.parser.Validate(jsonContent)
	if !v.Valid {
		return NewInvalidFormatError("import", "", fmt.Errorf("invalid JSON configuration: %s", strings.Join(v.Errors, "\n  ")))
	}

	entries, err := s.parser.Parse(jsonContent, NewFileSource(s.path), s.level)
	if err != nil {
		return NewInvalidFormatError("import", "", err)
	}

	s.entries = entries
	return nil
}

// Path returns the file path for this configuration store.
//
// Returns:
//   - scpath.AbsolutePath: The absolute path to the configuration file
func (s *Store) Path() scpath.AbsolutePath {
	return s.path
}

// Level returns the configuration level for this store.
//
// Returns:
//   - ConfigLevel: The level (system, global, or local)
func (s *Store) Level() ConfigLevel {
	return s.level
}

// HasKey returns true if the store has any entries for the given key.
//
// Parameters:
//   - key: The configuration key to check
//
// Returns:
//   - bool: true if the key exists with at least one entry, false otherwise
func (s *Store) HasKey(key string) bool {
	entries, exists := s.entries[key]
	return exists && len(entries) > 0
}

// Clear removes all entries from the store.
//
// The configuration is cleared in memory only. Call Save() to persist
// the empty configuration to disk.
func (s *Store) Clear() {
	s.entries = make(map[string][]*ConfigEntry)
}
