package config

// ConfigLevel represents the hierarchy level of a configuration entry
// Ordered by precedence (highest to lowest)
type ConfigLevel int

const (
	// CommandLineLevel represents command-line configuration (highest precedence)
	// Example: --config.user.name="John"
	CommandLineLevel ConfigLevel = iota

	// RepositoryLevel represents repository-specific configuration
	// Location: .source/config.json
	RepositoryLevel

	// UserLevel represents user-specific configuration
	// Location: ~/.config/sourcecontrol/config.json
	UserLevel

	// SystemLevel represents system-wide configuration
	// Location: /etc/sourcecontrol/config.json (Unix) or C:\ProgramData\SourceControl\config.json (Windows)
	SystemLevel

	// BuiltinLevel represents hardcoded default values (lowest precedence)
	BuiltinLevel
)

// String returns the string representation of the configuration level
func (l ConfigLevel) String() string {
	switch l {
	case CommandLineLevel:
		return "command-line"
	case RepositoryLevel:
		return "repository"
	case UserLevel:
		return "user"
	case SystemLevel:
		return "system"
	case BuiltinLevel:
		return "builtin"
	default:
		return "unknown"
	}
}

// IsValid returns true if the configuration level is valid
func (l ConfigLevel) IsValid() bool {
	return l >= CommandLineLevel && l <= BuiltinLevel
}

// CanWrite returns true if the configuration level is writable
func (l ConfigLevel) CanWrite() bool {
	return l == RepositoryLevel || l == UserLevel || l == SystemLevel
}

// ParseLevel converts a string to a ConfigLevel
func ParseLevel(s string) (ConfigLevel, error) {
	switch s {
	case "command-line":
		return CommandLineLevel, nil
	case "repository":
		return RepositoryLevel, nil
	case "user":
		return UserLevel, nil
	case "system":
		return SystemLevel, nil
	case "builtin":
		return BuiltinLevel, nil
	default:
		return 0, NewConfigError("parse", CodeInvalidLevelErr, "", "", s, ErrInvalidLevel)
	}
}
