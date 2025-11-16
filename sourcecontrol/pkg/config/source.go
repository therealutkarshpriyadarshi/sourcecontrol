package config

import "github.com/utkarsh5026/SourceControl/pkg/repository/scpath"

// ConfigSource represents the source of a configuration entry
// Can be either a special source (command-line, builtin) or a file path
type ConfigSource string

const (
	// CommandLineSource represents configuration from command-line flags
	CommandLineSource ConfigSource = "command-line"

	// BuiltinSource represents hardcoded default configuration
	BuiltinSource ConfigSource = "builtin"
)

// NewFileSource creates a ConfigSource from a file path
func NewFileSource(path scpath.AbsolutePath) ConfigSource {
	return ConfigSource(path.String())
}

// String returns the string representation of the source
func (s ConfigSource) String() string {
	return string(s)
}

// IsCommandLine returns true if this is a command-line source
func (s ConfigSource) IsCommandLine() bool {
	return s == CommandLineSource
}

// IsBuiltin returns true if this is a builtin source
func (s ConfigSource) IsBuiltin() bool {
	return s == BuiltinSource
}

// IsFile returns true if this is a file-based source
func (s ConfigSource) IsFile() bool {
	return !s.IsCommandLine() && !s.IsBuiltin()
}

// IsValid returns true if the source is valid (non-empty)
func (s ConfigSource) IsValid() bool {
	return s != ""
}
