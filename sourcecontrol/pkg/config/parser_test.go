package config

import (
	"encoding/json"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	parser := &Parser{}

	tests := []struct {
		name       string
		content    string
		wantKeys   []string
		wantValues map[string]string
		wantErr    bool
	}{
		{
			name:       "empty content",
			content:    "",
			wantKeys:   []string{},
			wantValues: map[string]string{},
			wantErr:    false,
		},
		{
			name: "simple key-value",
			content: `{
				"core": {
					"filemode": "true",
					"bare": "false"
				}
			}`,
			wantKeys: []string{"core.filemode", "core.bare"},
			wantValues: map[string]string{
				"core.filemode": "true",
				"core.bare":     "false",
			},
			wantErr: false,
		},
		{
			name: "nested sections",
			content: `{
				"remote": {
					"origin": {
						"url": "https://github.com/user/repo.git"
					}
				}
			}`,
			wantKeys: []string{"remote.origin.url"},
			wantValues: map[string]string{
				"remote.origin.url": "https://github.com/user/repo.git",
			},
			wantErr: false,
		},
		{
			name: "array values",
			content: `{
				"remote": {
					"origin": {
						"fetch": [
							"+refs/heads/*:refs/remotes/origin/*",
							"+refs/tags/*:refs/tags/*"
						]
					}
				}
			}`,
			wantKeys:   []string{"remote.origin.fetch"},
			wantValues: map[string]string{},
			wantErr:    false,
		},
		{
			name:       "invalid JSON",
			content:    `{invalid json}`,
			wantKeys:   []string{},
			wantValues: map[string]string{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.content, ConfigSource("test.json"), UserLevel)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Check that expected keys exist
			for _, key := range tt.wantKeys {
				if _, exists := result[key]; !exists {
					t.Errorf("Parse() missing key %q", key)
				}
			}

			// Check values
			for key, wantValue := range tt.wantValues {
				entries, exists := result[key]
				if !exists {
					t.Errorf("Parse() missing key %q", key)
					continue
				}
				if len(entries) == 0 {
					t.Errorf("Parse() key %q has no entries", key)
					continue
				}
				if entries[0].Value != wantValue {
					t.Errorf("Parse() key %q = %q, want %q", key, entries[0].Value, wantValue)
				}
			}
		})
	}
}

func TestParser_ParseArrayValues(t *testing.T) {
	parser := &Parser{}
	content := `{
		"remote": {
			"origin": {
				"fetch": [
					"+refs/heads/*:refs/remotes/origin/*",
					"+refs/tags/*:refs/tags/*"
				]
			}
		}
	}`

	result, err := parser.Parse(content, ConfigSource("test.json"), UserLevel)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	entries, exists := result["remote.origin.fetch"]
	if !exists {
		t.Fatal("Parse() missing key remote.origin.fetch")
	}

	if len(entries) != 2 {
		t.Errorf("Parse() remote.origin.fetch has %d entries, want 2", len(entries))
	}

	expectedValues := []string{
		"+refs/heads/*:refs/remotes/origin/*",
		"+refs/tags/*:refs/tags/*",
	}

	for i, entry := range entries {
		if entry.Value != expectedValues[i] {
			t.Errorf("Parse() entry[%d] = %q, want %q", i, entry.Value, expectedValues[i])
		}
	}
}

func TestParser_Serialize(t *testing.T) {
	parser := &Parser{}

	tests := []struct {
		name    string
		entries map[string][]*ConfigEntry
		wantErr bool
	}{
		{
			name:    "empty entries",
			entries: map[string][]*ConfigEntry{},
			wantErr: false,
		},
		{
			name: "simple entries",
			entries: map[string][]*ConfigEntry{
				"core.filemode": {
					NewEntry("core.filemode", "true", UserLevel, ConfigSource("test"), 0),
				},
				"user.name": {
					NewEntry("user.name", "John Doe", UserLevel, ConfigSource("test"), 0),
				},
			},
			wantErr: false,
		},
		{
			name: "multi-value entries",
			entries: map[string][]*ConfigEntry{
				"remote.origin.fetch": {
					NewEntry("remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*", UserLevel, ConfigSource("test"), 0),
					NewEntry("remote.origin.fetch", "+refs/tags/*:refs/tags/*", UserLevel, ConfigSource("test"), 0),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Serialize(tt.entries)
			if (err != nil) != tt.wantErr {
				t.Errorf("Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Verify the result is valid JSON
			var parsed interface{}
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Errorf("Serialize() produced invalid JSON: %v", err)
			}
		})
	}
}

func TestParser_Validate(t *testing.T) {
	parser := &Parser{}

	tests := []struct {
		name      string
		content   string
		wantValid bool
	}{
		{
			name:      "valid simple config",
			content:   `{"core": {"filemode": "true"}}`,
			wantValid: true,
		},
		{
			name:      "valid nested config",
			content:   `{"remote": {"origin": {"url": "https://github.com/user/repo.git"}}}`,
			wantValid: true,
		},
		{
			name:      "valid array config",
			content:   `{"remote": {"origin": {"fetch": ["+refs/heads/*:refs/remotes/origin/*"]}}}`,
			wantValid: true,
		},
		{
			name:      "invalid JSON syntax",
			content:   `{invalid}`,
			wantValid: false,
		},
		{
			name:      "non-object root",
			content:   `["array"]`,
			wantValid: false,
		},
		{
			name:      "array with objects",
			content:   `{"remote": {"origin": {"fetch": [{"key": "value"}]}}}`,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Validate(tt.content)
			if result.Valid != tt.wantValid {
				t.Errorf("Validate() valid = %v, want %v. Errors: %v", result.Valid, tt.wantValid, result.Errors)
			}
		})
	}
}

func TestParser_RoundTrip(t *testing.T) {
	parser := &Parser{}

	original := map[string][]*ConfigEntry{
		"core.filemode": {
			NewEntry("core.filemode", "true", UserLevel, ConfigSource("test"), 0),
		},
		"user.name": {
			NewEntry("user.name", "John Doe", UserLevel, ConfigSource("test"), 0),
		},
		"remote.origin.url": {
			NewEntry("remote.origin.url", "https://github.com/user/repo.git", UserLevel, ConfigSource("test"), 0),
		},
	}

	// Serialize
	serialized, err := parser.Serialize(original)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Parse back
	parsed, err := parser.Parse(serialized, "test.json", UserLevel)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Compare
	for key, originalEntries := range original {
		parsedEntries, exists := parsed[key]
		if !exists {
			t.Errorf("Round-trip lost key %q", key)
			continue
		}
		if len(parsedEntries) != len(originalEntries) {
			t.Errorf("Round-trip key %q has %d entries, want %d", key, len(parsedEntries), len(originalEntries))
			continue
		}
		for i := range originalEntries {
			if parsedEntries[i].Value != originalEntries[i].Value {
				t.Errorf("Round-trip key %q entry[%d] = %q, want %q",
					key, i, parsedEntries[i].Value, originalEntries[i].Value)
			}
		}
	}
}
