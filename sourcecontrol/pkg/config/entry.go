package config

import (
	"strconv"
	"strings"
)

// ConfigEntry represents a single configuration entry with its value and metadata
type ConfigEntry struct {
	Key        string       // Configuration key (e.g., "user.name")
	Value      string       // String value
	Level      ConfigLevel  // Configuration level
	Source     ConfigSource // Source of this configuration (command-line, builtin, or file path)
	LineNumber int          // Line number in source file (0 if not applicable)
}

// NewEntry creates a new configuration entry
func NewEntry(key, value string, level ConfigLevel, source ConfigSource, lineNumber int) *ConfigEntry {
	return &ConfigEntry{
		Key:        key,
		Value:      value,
		Level:      level,
		Source:     source,
		LineNumber: lineNumber,
	}
}

func NewCommandLineEntry(key, value string) *ConfigEntry {
	return &ConfigEntry{
		Key:        key,
		Value:      value,
		Level:      CommandLineLevel,
		Source:     CommandLineSource,
		LineNumber: 0,
	}
}

func NewBuiltinEntry(key, value string) *ConfigEntry {
	return &ConfigEntry{
		Key:        key,
		Value:      value,
		Level:      BuiltinLevel,
		Source:     BuiltinSource,
		LineNumber: 0,
	}
}

// AsString returns the value as a string
func (e *ConfigEntry) AsString() string {
	return e.Value
}

// AsInt converts the value to an integer
func (e *ConfigEntry) AsInt() (int, error) {
	val, err := strconv.Atoi(e.Value)
	if err != nil {
		return 0, NewConversionError(e.Key, err)
	}
	return val, nil
}

// AsInt64 converts the value to an int64
func (e *ConfigEntry) AsInt64() (int64, error) {
	val, err := strconv.ParseInt(e.Value, 10, 64)
	if err != nil {
		return 0, NewConversionError(e.Key, err)
	}
	return val, nil
}

// AsFloat64 converts the value to a float64
func (e *ConfigEntry) AsFloat64() (float64, error) {
	val, err := strconv.ParseFloat(e.Value, 64)
	if err != nil {
		return 0, NewConversionError(e.Key, err)
	}
	return val, nil
}

// AsBoolean converts the value to a boolean
// Accepts: "true", "yes", "1", "on" (case-insensitive) as true
// Accepts: "false", "no", "0", "off" (case-insensitive) as false
func (e *ConfigEntry) AsBoolean() (bool, error) {
	lower := strings.ToLower(strings.TrimSpace(e.Value))
	switch lower {
	case "true", "yes", "1", "on":
		return true, nil
	case "false", "no", "0", "off":
		return false, nil
	default:
		return false, NewConversionError(e.Key, ErrConversion)
	}
}

// AsList converts the value to a list of strings by splitting on commas
// Trims whitespace from each element and filters out empty strings
func (e *ConfigEntry) AsList() []string {
	if e.Value == "" {
		return []string{}
	}

	parts := strings.Split(e.Value, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// Clone creates a deep copy of the configuration entry
func (e *ConfigEntry) Clone() *ConfigEntry {
	return &ConfigEntry{
		Key:        e.Key,
		Value:      e.Value,
		Level:      e.Level,
		Source:     e.Source,
		LineNumber: e.LineNumber,
	}
}
