package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Parser handles parsing and serialization of JSON configuration files
type Parser struct{}

// ValidationResult contains validation results
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// parseContext holds common parsing parameters
type parseContext struct {
	result map[string][]*ConfigEntry
	source ConfigSource
	level  ConfigLevel
}

// validationContext holds validation state
type validationContext struct {
	errors *[]string
}

// Parse parses JSON configuration content into a map of entries
func (p *Parser) Parse(content string, source ConfigSource, level ConfigLevel) (map[string][]*ConfigEntry, error) {
	result := make(map[string][]*ConfigEntry)

	if strings.TrimSpace(content) == "" {
		return result, nil
	}

	configData := NewConfigFileStructure()
	if err := json.Unmarshal([]byte(content), configData); err != nil {
		return nil, NewInvalidFormatError("parse", source.String(), fmt.Errorf("%w: %v", ErrInvalidFormat, err))
	}

	ctx := &parseContext{
		result: result,
		source: source,
		level:  level,
	}

	if err := p.parseSection(configData, ctx, ""); err != nil {
		return nil, err
	}

	return result, nil
}

// Serialize converts configuration entries to JSON format
func (p *Parser) Serialize(entries map[string][]*ConfigEntry) (string, error) {
	configData := NewConfigFileStructure()

	for fullKey, entryList := range entries {
		for _, entry := range entryList {
			if err := configData.SetNestedValue(fullKey, entry.Value); err != nil {
				return "", err
			}
		}
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return "", NewInvalidFormatError("serialize", "", fmt.Errorf("%w: %v", ErrInvalidFormat, err))
	}

	return string(data), nil
}

// Validate validates JSON configuration structure
func (p *Parser) Validate(content string) ValidationResult {
	errors := []string{}

	var parsed any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		errors = append(errors, fmt.Sprintf("Invalid JSON: %v", err))
		return ValidationResult{Valid: false, Errors: errors}
	}

	configMap, ok := parsed.(map[string]any)
	if !ok {
		errors = append(errors, "Configuration must be a JSON object")
		return ValidationResult{Valid: false, Errors: errors}
	}

	ctx := &validationContext{errors: &errors}
	p.validateSection(configMap, "", ctx)
	return ValidationResult{Valid: len(errors) == 0, Errors: errors}
}

// FormatForDisplay creates a pretty-formatted JSON string for display
// Only includes the last (effective) value for each key
func (p *Parser) FormatForDisplay(entries map[string][]*ConfigEntry) (string, error) {
	cfg := NewConfigFileStructure()

	for fullKey, entryList := range entries {
		if len(entryList) > 0 {
			effectiveEntry := entryList[len(entryList)-1]
			if err := cfg.SetNestedValue(fullKey, effectiveEntry.Value); err != nil {
				return "", err
			}
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", NewInvalidFormatError("format", "", fmt.Errorf("%w: %v", ErrInvalidFormat, err))
	}

	return string(data), nil
}

// parseSection recursively parses sections and subsections
func (p *Parser) parseSection(cfg *ConfigFileStructure, ctx *parseContext, keyPrefix string) error {
	return cfg.Range(func(k string, v any) error {
		fullKey := p.buildFullKey(keyPrefix, k)
		return p.processConfigValue(fullKey, v, ctx)
	})
}

// buildFullKey builds the full configuration key from prefix and section key
func (p *Parser) buildFullKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// processConfigValue processes a configuration value based on its type
func (p *Parser) processConfigValue(key string, value any, ctx *parseContext) error {
	switch v := value.(type) {
	case []any:
		return p.processArrayValue(key, v, ctx)
	case map[string]any:
		nestedConfig := &ConfigFileStructure{data: v}
		return p.parseSection(nestedConfig, ctx, key)
	case string:
		p.addEntry(ctx, key, v)
		return nil
	default:
		p.addEntry(ctx, key, fmt.Sprintf("%v", v))
		return nil
	}
}

// processArrayValue processes array configuration values
func (p *Parser) processArrayValue(key string, values []any, ctx *parseContext) error {
	for _, item := range values {
		if strVal, ok := item.(string); ok {
			p.addEntry(ctx, key, strVal)
		} else {
			p.addEntry(ctx, key, fmt.Sprintf("%v", item))
		}
	}
	return nil
}

// addEntry adds a configuration entry to the result map
func (p *Parser) addEntry(ctx *parseContext, configKey string, configValue string) {
	if _, exists := ctx.result[configKey]; !exists {
		ctx.result[configKey] = []*ConfigEntry{}
	}

	entry := NewEntry(configKey, configValue, ctx.level, ctx.source, 0)
	ctx.result[configKey] = append(ctx.result[configKey], entry)
}

// validateSection validates a configuration section
func (p *Parser) validateSection(configSection map[string]any, currentPath string, ctx *validationContext) {
	for key, value := range configSection {
		valuePath := p.buildFullKey(currentPath, key)
		p.validateConfigValue(valuePath, value, ctx)
	}
}

// validateConfigValue validates a single configuration value
func (p *Parser) validateConfigValue(path string, value any, ctx *validationContext) {
	switch v := value.(type) {
	case []any:
		p.validateArrayValue(path, v, ctx)
	case map[string]any:
		p.validateSection(v, path, ctx)
	case string:
		// Valid
	default:
		// Other types are allowed and will be converted to strings
	}
}

// validateArrayValue validates array configuration values
func (p *Parser) validateArrayValue(path string, values []any, ctx *validationContext) {
	for _, item := range values {
		switch item.(type) {
		case string:
			// Valid
		case map[string]any:
			*ctx.errors = append(*ctx.errors, fmt.Sprintf("Configuration array at '%s' cannot contain objects", path))
		default:
			// Other types are allowed and will be converted to strings
		}
	}
}
