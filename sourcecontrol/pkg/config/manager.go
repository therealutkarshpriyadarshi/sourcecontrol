package config

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"golang.org/x/sync/errgroup"
)

// Default configuration paths for different platforms
const (
	WindowsProgramFilesPath = `C:\ProgramData\SourceControl`
	UnixProgramFilesPath    = "/etc/sourcecontrol"
	ConfigFileName          = "config.json"
)

// Manager is the central configuration manager that handles the hierarchy of config files.
//
// The Manager implements a hierarchical configuration system with the following precedence
// (highest to lowest):
//  1. Command-line arguments (-c flag)
//  2. Repository-level configuration (.sourcecontrol/config.json)
//  3. User-level configuration (~/.config/sourcecontrol/config.json)
//  4. System-level configuration (/etc/sourcecontrol/config.json or C:\ProgramData\SourceControl\config.json)
//  5. Built-in defaults (hardcoded values)
//
// The Manager is thread-safe and can be used concurrently. It provides atomic read
// and write operations through internal mutex locking.
//
// Example usage:
//
//	// Create manager for a repository
//	manager := NewManager(repoPath)
//
//	// Load all configuration files
//	if err := manager.Load(ctx); err != nil {
//	    return err
//	}
//
//	// Get configuration value (respects hierarchy)
//	if entry := manager.Get("user.name"); entry != nil {
//	    fmt.Println(entry.Value)
//	}
//
//	// Set configuration at specific level
//	manager.Set("user.email", "user@example.com", UserLevel)
type Manager struct {
	mu              sync.RWMutex           // Protects concurrent access to all fields
	stores          map[ConfigLevel]*Store // Configuration stores for each level
	commandLine     map[string]string      // Command-line overrides (highest precedence)
	builtinDefaults map[string]string      // Built-in default values (lowest precedence)
	parser          *Parser                // JSON parser for serialization
}

// NewManager creates a new configuration manager.
//
// The manager is initialized with stores for system and user levels. If a
// repository path is provided, a repository-level store is also created.
//
// Built-in defaults are loaded automatically during initialization.
//
// Parameters:
//   - repositoryPath: Path to the repository root (empty for non-repository contexts)
//
// Returns:
//   - *Manager: A new Manager instance ready for use
//
// Example:
//
//	// Manager for repository context
//	manager := NewManager(scpath.RepositoryPath("/path/to/repo"))
//
//	// Manager for global context (no repository)
//	manager := NewManager("")
func NewManager(repositoryPath scpath.RepositoryPath) *Manager {
	m := &Manager{
		stores:          make(map[ConfigLevel]*Store),
		commandLine:     make(map[string]string),
		builtinDefaults: make(map[string]string),
		parser:          &Parser{},
	}

	m.initializeStores(repositoryPath)
	m.loadBuiltinDefaults()

	return m
}

// Load loads all configuration files from disk concurrently.
//
// This method should be called once during initialization. It loads all
// available configuration stores in parallel using an error group. If any
// store fails to load, the error is returned immediately.
//
// Non-existent configuration files are handled gracefully and don't cause
// errors. Only actual I/O failures or malformed JSON will result in errors.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//
// Returns:
//   - error: If any configuration file fails to load
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	if err := manager.Load(ctx); err != nil {
//	    log.Fatalf("Failed to load configuration: %v", err)
//	}
func (m *Manager) Load(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	g, _ := errgroup.WithContext(ctx)

	for _, store := range m.stores {
		s := store // Capture loop variable
		g.Go(func() error {
			return s.Load()
		})
	}

	return g.Wait()
}

// Get retrieves a configuration value, respecting the hierarchy.
//
// Returns the highest precedence value for the key according to the hierarchy:
// command-line > repository > user > system > built-in defaults.
//
// If the key doesn't exist at any level, returns nil.
//
// Parameters:
//   - key: Configuration key in dot notation (e.g., "user.name")
//
// Returns:
//   - *ConfigEntry: The highest precedence entry, or nil if not found
//
// Example:
//
//	// Get user name
//	if entry := manager.Get("user.name"); entry != nil {
//	    fmt.Printf("User: %s (from %s)\n", entry.Value, entry.Source())
//	}
func (m *Manager) Get(key string) *ConfigEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.getUnsafe(key)
}

// GetAll retrieves all values for a configuration key across all levels.
//
// Useful for multi-value keys like remote.origin.fetch where each level
// can contribute values. Returns entries from all levels, not just the
// highest precedence one.
//
// Parameters:
//   - key: Configuration key in dot notation
//
// Returns:
//   - []*ConfigEntry: All entries for the key across all levels (may be empty)
//
// Example:
//
//	// Get all fetch refspecs
//	entries := manager.GetAll("remote.origin.fetch")
//	for _, entry := range entries {
//	    fmt.Printf("Fetch: %s (from %s)\n", entry.Value, entry.Level)
//	}
func (m *Manager) GetAll(key string) []*ConfigEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allEntries []*ConfigEntry

	// Command-line overrides
	if value, exists := m.commandLine[key]; exists {
		allEntries = append(allEntries, NewCommandLineEntry(key, value))
	}

	// Store entries (repository, user, system)
	allEntries = append(allEntries, m.findInStores(key)...)

	// Built-in defaults
	if value, exists := m.builtinDefaults[key]; exists {
		allEntries = append(allEntries, NewBuiltinEntry(key, value))
	}

	return allEntries
}

// Set sets a configuration value at a specific level.
//
// The value is immediately written to disk. This operation is atomic - either
// the entire operation succeeds or it fails with an error.
//
// Parameters:
//   - key: Configuration key in dot notation
//   - value: Value to set
//   - level: Configuration level to write to (must be writable)
//
// Returns:
//   - error: If the level doesn't exist, is read-only, or write fails
//
// Example:
//
//	// Set user email at user level
//	if err := manager.Set("user.email", "user@example.com", UserLevel); err != nil {
//	    log.Fatalf("Failed to set email: %v", err)
//	}
func (m *Manager) Set(key, value string, level ConfigLevel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	store, err := m.validateStore("set", key, level)
	if err != nil {
		return err
	}

	store.Set(key, value)
	return store.Save()
}

// Add adds a value to a multi-value configuration key.
//
// Unlike Set, which replaces all values, Add appends to existing values.
// This is useful for keys that support multiple values.
//
// The change is immediately written to disk.
//
// Parameters:
//   - key: Configuration key in dot notation
//   - value: Value to append
//   - level: Configuration level to write to (must be writable)
//
// Returns:
//   - error: If the level doesn't exist, is read-only, or write fails
//
// Example:
//
//	// Add a fetch refspec
//	err := manager.Add("remote.origin.fetch", "+refs/tags/*:refs/tags/*", RepositoryLevel)
func (m *Manager) Add(key, value string, level ConfigLevel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	store, err := m.validateStore("add", key, level)
	if err != nil {
		return err
	}

	store.Add(key, value)
	return store.Save()
}

// Unset removes a configuration key at a specific level.
//
// Only removes the key from the specified level, not from other levels.
// The change is immediately written to disk.
//
// Parameters:
//   - key: Configuration key to remove
//   - level: Configuration level to remove from (must be writable)
//
// Returns:
//   - error: If the level doesn't exist, is read-only, or write fails
//
// Example:
//
//	// Remove user email from user level
//	if err := manager.Unset("user.email", UserLevel); err != nil {
//	    log.Printf("Failed to unset email: %v", err)
//	}
func (m *Manager) Unset(key string, level ConfigLevel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	store, err := m.validateStore("unset", key, level)
	if err != nil {
		return err
	}

	store.Unset(key)
	return store.Save()
}

// validateStore validates that a store exists and is writable.
//
// This is an internal helper method that performs common validation
// for write operations.
//
// Parameters:
//   - operation: Name of the operation (for error messages)
//   - key: Configuration key (for error messages)
//   - level: Configuration level to validate
//
// Returns:
//   - *Store: The validated store
//   - error: If validation fails
func (m *Manager) validateStore(operation string, key string, level ConfigLevel) (*Store, error) {
	if !level.CanWrite() {
		return nil, NewConfigError(operation, CodeReadOnlyErr, key, "", level.String(), ErrReadOnly)
	}

	store, exists := m.stores[level]
	if !exists {
		return nil, NewNotFoundError(key, level.String())
	}

	return store, nil
}

// SetCommandLine sets a command-line configuration value.
//
// Command-line values have the highest precedence and override all other
// configuration sources. These values are not persisted to disk.
//
// This method is typically used to implement the -c flag functionality.
//
// Parameters:
//   - key: Configuration key in dot notation
//   - value: Value to set
//
// Example:
//
//	// Set temporary configuration override
//	manager.SetCommandLine("core.editor", "vim")
func (m *Manager) SetCommandLine(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandLine[key] = value
}

// List returns all effective configuration entries respecting hierarchy.
//
// For each unique key, returns only the highest precedence value.
// The entries are sorted by key for consistent output.
//
// Returns:
//   - []*ConfigEntry: Sorted list of all effective configuration entries
//
// Example:
//
//	// List all configuration
//	for _, entry := range manager.List() {
//	    fmt.Printf("%s = %s (from %s)\n", entry.Key, entry.Value, entry.Source())
//	}
func (m *Manager) List() []*ConfigEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.listUnsafe()
}

// collectAllKeys collects all unique keys across all configuration sources.
//
// This is an internal helper method used by List to find all keys
// that exist in any configuration source.
//
// Returns:
//   - map[string]bool: Set of all unique configuration keys
func (m *Manager) collectAllKeys() map[string]bool {
	allKeys := make(map[string]bool)

	// Collect from command-line
	for key := range m.commandLine {
		allKeys[key] = true
	}

	// Collect from all stores
	for _, store := range m.stores {
		for key := range store.GetAllEntries() {
			allKeys[key] = true
		}
	}

	// Collect from built-in defaults
	for key := range m.builtinDefaults {
		allKeys[key] = true
	}

	return allKeys
}

// ExportJSON exports configuration as JSON string.
//
// If a level is specified, exports only that level's configuration.
// If level is nil, exports all effective configuration (respecting hierarchy).
//
// Parameters:
//   - level: Optional pointer to specific level to export (nil for all)
//
// Returns:
//   - string: JSON representation of the configuration
//   - error: If serialization fails
//
// Example:
//
//	// Export user-level configuration
//	userLevel := UserLevel
//	json, err := manager.ExportJSON(&userLevel)
//
//	// Export all effective configuration
//	json, err := manager.ExportJSON(nil)
func (m *Manager) ExportJSON(level *ConfigLevel) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if level != nil {
		store, exists := m.stores[*level]
		if !exists {
			return "{}", nil
		}
		return store.ToJSON()
	}

	// Export all effective configuration
	entries := m.listUnsafe()
	entriesMap := make(map[string][]*ConfigEntry)

	for _, entry := range entries {
		if _, exists := entriesMap[entry.Key]; !exists {
			entriesMap[entry.Key] = []*ConfigEntry{}
		}
		entriesMap[entry.Key] = append(entriesMap[entry.Key], entry)
	}

	return m.parser.Serialize(entriesMap)
}

// GetStore returns the store for a specific level.
//
// This method is useful for direct access to a specific configuration level,
// bypassing the hierarchy. Returns nil if the store doesn't exist.
//
// Parameters:
//   - level: Configuration level to retrieve
//
// Returns:
//   - *Store: The store for the level, or nil if not found
//
// Example:
//
//	// Direct access to user store
//	if store := manager.GetStore(UserLevel); store != nil {
//	    entries := store.GetAllEntries()
//	}
func (m *Manager) GetStore(level ConfigLevel) *Store {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stores[level]
}

// initializeStores creates stores for different configuration levels.
//
// This is an internal helper method called during Manager initialization.
// It creates stores for system and user levels always, and repository
// level only if a repository path is provided.
//
// Parameters:
//   - repositoryPath: Path to repository root (empty for non-repository contexts)
func (m *Manager) initializeStores(repositoryPath scpath.RepositoryPath) {
	// System-level configuration
	systemPath := m.getSystemConfigPath()
	m.stores[SystemLevel] = NewStore(systemPath, SystemLevel)

	// User-level configuration
	userPath := m.getUserConfigPath()
	m.stores[UserLevel] = NewStore(userPath, UserLevel)

	// Repository-level configuration (if applicable)
	if repositoryPath != "" {
		repoPath := scpath.AbsolutePath(filepath.Join(string(repositoryPath), ConfigFileName))
		m.stores[RepositoryLevel] = NewStore(repoPath, RepositoryLevel)
	}
}

// getSystemConfigPath returns the system-wide configuration path.
//
// The path is platform-specific:
//   - Windows: C:\ProgramData\SourceControl\config.json
//   - Unix/Linux: /etc/sourcecontrol/config.json
//
// Returns:
//   - scpath.AbsolutePath: Absolute path to system configuration
func (m *Manager) getSystemConfigPath() scpath.AbsolutePath {
	var path string
	if runtime.GOOS == "windows" {
		path = filepath.Join(WindowsProgramFilesPath, ConfigFileName)
	} else {
		path = filepath.Join(UnixProgramFilesPath, ConfigFileName)
	}
	return scpath.AbsolutePath(path)
}

// getUserConfigPath returns the user-specific configuration path.
//
// The path is typically: ~/.config/sourcecontrol/config.json
//
// If the home directory cannot be determined, falls back to current directory.
//
// Returns:
//   - scpath.AbsolutePath: Absolute path to user configuration
func (m *Manager) getUserConfigPath() scpath.AbsolutePath {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home dir can't be determined
		homeDir = "."
	}
	return scpath.AbsolutePath(filepath.Join(homeDir, ".config", "sourcecontrol", ConfigFileName))
}

// loadBuiltinDefaults initializes hardcoded default values.
//
// This is an internal helper method that loads sensible defaults for
// common configuration keys. Platform-specific defaults are set based
// on the operating system.
//
// Defaults include:
//   - Core repository settings
//   - Branch initialization defaults
//   - UI and display settings
//   - Platform-specific settings (line endings, case sensitivity)
func (m *Manager) loadBuiltinDefaults() {
	// Core repository settings
	m.builtinDefaults["core.repositoryformatversion"] = "0"
	m.builtinDefaults["core.filemode"] = "true"
	m.builtinDefaults["core.bare"] = "false"
	m.builtinDefaults["core.logallrefupdates"] = "true"

	// Branch and workflow settings
	m.builtinDefaults["init.defaultbranch"] = "main"
	m.builtinDefaults["pull.rebase"] = "false"
	m.builtinDefaults["push.default"] = "simple"

	// UI and display settings
	m.builtinDefaults["color.ui"] = "auto"
	m.builtinDefaults["diff.renames"] = "true"

	// Platform-specific defaults
	if runtime.GOOS == "windows" {
		m.builtinDefaults["core.ignorecase"] = "true"
		m.builtinDefaults["core.autocrlf"] = "true"
	} else {
		m.builtinDefaults["core.ignorecase"] = "false"
		m.builtinDefaults["core.autocrlf"] = "input"
	}
}

// getUnsafe is the internal implementation of Get without locking.
//
// This method must only be called when the caller already holds at least
// a read lock. It implements the hierarchy search logic.
//
// Parameters:
//   - key: Configuration key to retrieve
//
// Returns:
//   - *ConfigEntry: The highest precedence entry, or nil if not found
func (m *Manager) getUnsafe(key string) *ConfigEntry {
	// Check command-line (highest precedence)
	if value, exists := m.commandLine[key]; exists {
		return NewCommandLineEntry(key, value)
	}

	// Check stores (repository > user > system)
	entries := m.findInStores(key)
	if len(entries) > 0 {
		return entries[len(entries)-1]
	}

	// Check built-in defaults (lowest precedence)
	if value, exists := m.builtinDefaults[key]; exists {
		return NewBuiltinEntry(key, value)
	}

	return nil
}

// listUnsafe is the internal implementation of List without locking.
//
// This method must only be called when the caller already holds at least
// a read lock. It collects all unique keys and returns their highest
// precedence values.
//
// Returns:
//   - []*ConfigEntry: Sorted list of all effective configuration entries
func (m *Manager) listUnsafe() []*ConfigEntry {
	allKeys := m.collectAllKeys()
	var entries []*ConfigEntry

	// Get effective value for each key
	for key := range allKeys {
		if entry := m.getUnsafe(key); entry != nil {
			entries = append(entries, entry)
		}
	}

	// Sort by key for consistent output
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})

	return entries
}

// findInStores searches for a key in stores following hierarchy.
//
// This is an internal helper that searches stores in precedence order:
// repository > user > system. Returns entries from the first store
// that contains the key.
//
// Parameters:
//   - key: Configuration key to search for
//
// Returns:
//   - []*ConfigEntry: Entries from the highest precedence store containing the key
func (m *Manager) findInStores(key string) []*ConfigEntry {
	levels := []ConfigLevel{RepositoryLevel, UserLevel, SystemLevel}

	for _, level := range levels {
		store, exists := m.stores[level]
		if !exists {
			continue
		}

		entries := store.GetEntries(key)
		if len(entries) > 0 {
			return entries
		}
	}

	return nil
}
