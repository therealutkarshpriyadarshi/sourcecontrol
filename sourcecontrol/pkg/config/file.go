package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// ConfigFileStructure represents the JSON structure of a configuration file.
//
// It provides a flexible hierarchical data structure that supports:
//   - Nested sections (e.g., "core.user.name")
//   - Single-valued keys (strings)
//   - Multi-valued keys (arrays)
//   - Dynamic schema (keys can be added at any level)
//
// The structure uses a map-based implementation to allow arbitrary nesting
// depth and handles automatic type conversion when values are added.
//
// Example JSON structure:
//
//	{
//	  "core": {
//	    "repositoryformatversion": "0",
//	    "filemode": "false"
//	  },
//	  "remote": {
//	    "origin": {
//	      "url": "https://github.com/user/repo.git",
//	      "fetch": ["+refs/heads/*:refs/remotes/origin/*"]
//	    }
//	  }
//	}
//
// Example usage:
//
//	config := NewConfigFileStructure()
//	config.SetNestedValue("core.user.name", "John Doe")
//	config.SetNestedValue("remote.origin.url", "https://github.com/user/repo")
//	config.SetNestedValue("remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
type ConfigFileStructure struct {
	data map[string]any // Root-level configuration data
}

// NewConfigFileStructure creates a new empty ConfigFileStructure.
//
// Returns:
//   - *ConfigFileStructure: A new instance with an initialized empty data map
func NewConfigFileStructure() *ConfigFileStructure {
	return &ConfigFileStructure{
		data: make(map[string]any),
	}
}

// UnmarshalJSON implements json.Unmarshaler interface for deserializing JSON.
//
// Parameters:
//   - data: Raw JSON bytes to unmarshal
//
// Returns:
//   - error: If JSON unmarshaling fails due to invalid syntax
func (c *ConfigFileStructure) UnmarshalJSON(data []byte) error {
	c.data = make(map[string]any)
	return json.Unmarshal(data, &c.data)
}

// MarshalJSON implements json.Marshaler interface for serializing to JSON.
//
// Returns:
//   - []byte: JSON representation of the configuration
//   - error: If JSON marshaling fails (rare, usually only for invalid types)
func (c *ConfigFileStructure) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.data)
}

// Range iterates over all top-level keys and values in the configuration.
//
// This method is useful for inspecting or processing all top-level sections
// of the configuration. The iteration order is undefined (map iteration).
//
// The callback function can return an error to stop iteration early.
//
// Parameters:
//   - fn: Callback function invoked for each key-value pair
//
// Returns:
//   - error: First error returned by the callback function, or nil if all iterations succeed
func (c *ConfigFileStructure) Range(fn func(key string, value any) error) error {
	for key, value := range c.data {
		if err := fn(key, value); err != nil {
			return err
		}
	}
	return nil
}

// SetNestedValue sets a value in the configuration using dot notation.
//
// The key path uses dots to separate nested levels (e.g., "core.user.name").
// Missing intermediate objects are automatically created as nested maps.
//
// Value handling behavior:
//   - If key doesn't exist: Creates new entry with the value
//   - If key exists with a string: Converts to array [oldValue, newValue]
//   - If key exists with an array: Appends newValue to the array
//   - If key exists with a nested object: No action (preserves structure)
//
// Parameters:
//   - keyPath: Dot-separated path to the configuration key (e.g., "remote.origin.url")
//   - value: The string value to set
//
// Returns:
//   - error: If the key path is empty
//
// Example:
//
//	// Set a simple value
//	config.SetNestedValue("user.name", "John Doe")
//
//	// Set a nested value
//	config.SetNestedValue("remote.origin.url", "https://github.com/user/repo")
//
//	// Add multiple values to same key
//	config.SetNestedValue("remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
//	config.SetNestedValue("remote.origin.fetch", "+refs/tags/*:refs/tags/*")
func (c *ConfigFileStructure) SetNestedValue(keyPath, value string) error {
	pathSegments := strings.Split(keyPath, ".")
	if len(pathSegments) == 0 {
		return NewInvalidValueError(keyPath, fmt.Errorf("empty key path"))
	}

	finalKey := pathSegments[len(pathSegments)-1]
	target := c.navigateToTargetObject(pathSegments[:len(pathSegments)-1])

	c.setValueInObject(target, finalKey, value)
	return nil
}

// navigateToTargetObject navigates through the object hierarchy, creating maps as needed.
//
// This is an internal helper method that ensures all intermediate objects
// in a key path exist. If any intermediate object doesn't exist or isn't
// a map, it creates a new map at that location.
//
// Parameters:
//   - pathSegments: Array of key segments to navigate through
//
// Returns:
//   - map[string]any: The target object where the final key-value should be set
//
// Example:
//   - pathSegments: ["remote", "origin"]
//   - Creates: c.data["remote"]["origin"] if it doesn't exist
//   - Returns: The map at c.data["remote"]["origin"]
func (c *ConfigFileStructure) navigateToTargetObject(pathSegments []string) map[string]any {
	currentObject := c.data

	for _, segment := range pathSegments {
		if !c.hasValidObjectProperty(currentObject, segment) {
			currentObject[segment] = make(map[string]any)
		}
		currentObject = currentObject[segment].(map[string]any)
	}

	return currentObject
}

// hasValidObjectProperty checks if an object has a valid map property for navigation.
//
// This is an internal helper method that verifies a property exists and is
// a nested map (not a string or array), making it suitable for further navigation.
//
// Parameters:
//   - obj: The object to check
//   - propertyKey: The property name to verify
//
// Returns:
//   - bool: true if the property exists and is a map[string]any, false otherwise
func (c *ConfigFileStructure) hasValidObjectProperty(obj map[string]any, propertyKey string) bool {
	_, exists := obj[propertyKey]
	return exists && reflect.TypeOf(obj[propertyKey]).Kind() == reflect.Map
}

// setValueInObject sets a value in an object, handling existing values intelligently.
//
// This method implements smart value merging:
//   - No existing value: Sets the new value directly
//   - Existing string value: Converts to array [oldValue, newValue]
//   - Existing array: Appends newValue to the array
//   - Existing nested object: Preserves the object, ignores new value
//   - Other types: Creates array [oldValue, newValue]
//
// This behavior allows configuration keys to naturally transition from
// single-valued to multi-valued as needed.
//
// Parameters:
//   - targetObject: The map object to modify
//   - key: The key within the object to set
//   - newValue: The string value to set or add
func (c *ConfigFileStructure) setValueInObject(targetObject map[string]any, key, newValue string) {
	existingValue, exists := targetObject[key]
	if !exists {
		targetObject[key] = newValue
		return
	}

	if arr, ok := existingValue.([]any); ok {
		targetObject[key] = append(arr, newValue)
		return
	}

	if strVal, ok := existingValue.(string); ok {
		targetObject[key] = []any{strVal, newValue}
		return
	}

	if _, ok := existingValue.(map[string]any); ok {
		return
	}

	targetObject[key] = []any{existingValue, newValue}
}
