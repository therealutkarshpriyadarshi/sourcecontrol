package config

// TypedConfig provides type-safe access to common configuration values
// It wraps a Manager and provides convenient getter methods
type TypedConfig struct {
	manager *Manager
}

// NewTypedConfig creates a new TypedConfig wrapper around a Manager
func NewTypedConfig(manager *Manager) *TypedConfig {
	return &TypedConfig{
		manager: manager,
	}
}

// RepositoryFormatVersion returns the repository format version
func (tc *TypedConfig) RepositoryFormatVersion() int {
	entry := tc.manager.Get("core.repositoryformatversion")
	if entry == nil {
		return 0
	}
	val, err := entry.AsInt()
	if err != nil {
		return 0
	}
	return val
}

// FileMode returns whether file mode should be tracked
func (tc *TypedConfig) FileMode() bool {
	entry := tc.manager.Get("core.filemode")
	if entry == nil {
		return true
	}
	val, err := entry.AsBoolean()
	if err != nil {
		return true
	}
	return val
}

// Bare returns whether the repository is bare
func (tc *TypedConfig) Bare() bool {
	entry := tc.manager.Get("core.bare")
	if entry == nil {
		return false
	}
	val, err := entry.AsBoolean()
	if err != nil {
		return false
	}
	return val
}

// IgnoreCase returns whether to ignore case in file names
func (tc *TypedConfig) IgnoreCase() bool {
	entry := tc.manager.Get("core.ignorecase")
	if entry == nil {
		return false
	}
	val, err := entry.AsBoolean()
	if err != nil {
		return false
	}
	return val
}

// AutoCRLF returns the line ending conversion setting
func (tc *TypedConfig) AutoCRLF() string {
	entry := tc.manager.Get("core.autocrlf")
	if entry == nil {
		return "input"
	}
	return entry.AsString()
}

// LogAllRefUpdates returns whether to log all ref updates
func (tc *TypedConfig) LogAllRefUpdates() bool {
	entry := tc.manager.Get("core.logallrefupdates")
	if entry == nil {
		return true
	}
	val, err := entry.AsBoolean()
	if err != nil {
		return true
	}
	return val
}

// User configuration

// UserName returns the configured user name
func (tc *TypedConfig) UserName() string {
	entry := tc.manager.Get("user.name")
	if entry == nil {
		return ""
	}
	return entry.AsString()
}

// UserEmail returns the configured user email
func (tc *TypedConfig) UserEmail() string {
	entry := tc.manager.Get("user.email")
	if entry == nil {
		return ""
	}
	return entry.AsString()
}

// Init configuration

// DefaultBranch returns the default branch name for new repositories
func (tc *TypedConfig) DefaultBranch() string {
	entry := tc.manager.Get("init.defaultbranch")
	if entry == nil {
		return "main"
	}
	return entry.AsString()
}

// Color configuration

// ColorUI returns the color UI setting
func (tc *TypedConfig) ColorUI() string {
	entry := tc.manager.Get("color.ui")
	if entry == nil {
		return "auto"
	}
	return entry.AsString()
}

// Remote configuration

// GetRemoteURL returns the URL for a specific remote
func (tc *TypedConfig) GetRemoteURL(remoteName string) string {
	entry := tc.manager.Get("remote." + remoteName + ".url")
	if entry == nil {
		return ""
	}
	return entry.AsString()
}

// GetRemoteFetch returns all fetch refspecs for a remote
func (tc *TypedConfig) GetRemoteFetch(remoteName string) []string {
	entries := tc.manager.GetAll("remote." + remoteName + ".fetch")
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.AsString())
	}
	return result
}

// GetRemotePushURL returns the push URL for a remote (falls back to URL if not set)
func (tc *TypedConfig) GetRemotePushURL(remoteName string) string {
	entry := tc.manager.Get("remote." + remoteName + ".pushurl")
	if entry == nil {
		return tc.GetRemoteURL(remoteName)
	}
	return entry.AsString()
}

// Branch configuration

// GetBranchRemote returns the remote for a specific branch
func (tc *TypedConfig) GetBranchRemote(branchName string) string {
	entry := tc.manager.Get("branch." + branchName + ".remote")
	if entry == nil {
		return ""
	}
	return entry.AsString()
}

// GetBranchMerge returns the merge ref for a specific branch
func (tc *TypedConfig) GetBranchMerge(branchName string) string {
	entry := tc.manager.Get("branch." + branchName + ".merge")
	if entry == nil {
		return ""
	}
	return entry.AsString()
}

// GetBranchRebase returns whether to rebase when pulling for a specific branch
func (tc *TypedConfig) GetBranchRebase(branchName string) bool {
	entry := tc.manager.Get("branch." + branchName + ".rebase")
	if entry == nil {
		return false
	}
	val, err := entry.AsBoolean()
	if err != nil {
		return false
	}
	return val
}

func (tc *TypedConfig) DiffRenames() bool {
	entry := tc.manager.Get("diff.renames")
	if entry == nil {
		return true
	}
	val, err := entry.AsBoolean()
	if err != nil {
		return true
	}
	return val
}

// PullRebase returns the default rebase strategy for pulls
func (tc *TypedConfig) PullRebase() string {
	entry := tc.manager.Get("pull.rebase")
	if entry == nil {
		return "false"
	}
	return entry.AsString()
}

// PushDefault returns the default push strategy
func (tc *TypedConfig) PushDefault() string {
	entry := tc.manager.Get("push.default")
	if entry == nil {
		return "simple"
	}
	return entry.AsString()
}

// GetString returns a configuration value as a string
func (tc *TypedConfig) GetString(key string) string {
	entry := tc.manager.Get(key)
	if entry == nil {
		return ""
	}
	return entry.AsString()
}

// GetInt returns a configuration value as an integer
func (tc *TypedConfig) GetInt(key string) (int, error) {
	entry := tc.manager.Get(key)
	if entry == nil {
		return 0, NewNotFoundError(key, "")
	}
	return entry.AsInt()
}

// GetBool returns a configuration value as a boolean
func (tc *TypedConfig) GetBool(key string) (bool, error) {
	entry := tc.manager.Get(key)
	if entry == nil {
		return false, NewNotFoundError(key, "")
	}
	return entry.AsBoolean()
}

// GetList returns a configuration value as a list of strings
func (tc *TypedConfig) GetList(key string) []string {
	entry := tc.manager.Get(key)
	if entry == nil {
		return []string{}
	}
	return entry.AsList()
}

// GetAll returns all values for a multi-value configuration key
func (tc *TypedConfig) GetAll(key string) []string {
	entries := tc.manager.GetAll(key)
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.AsString())
	}
	return result
}
