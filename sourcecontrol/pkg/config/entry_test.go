package config

import (
	"testing"
)

func TestConfigEntry_AsString(t *testing.T) {
	entry := NewEntry("test.key", "hello world", UserLevel, ConfigSource("test"), 0)
	if got := entry.AsString(); got != "hello world" {
		t.Errorf("AsString() = %q, want %q", got, "hello world")
	}
}

func TestConfigEntry_AsInt(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{"valid positive", "42", 42, false},
		{"valid negative", "-10", -10, false},
		{"valid zero", "0", 0, false},
		{"invalid string", "not a number", 0, true},
		{"invalid float", "3.14", 0, true},
		{"invalid empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test.key", tt.value, UserLevel, ConfigSource("test"), 0)
			got, err := entry.AsInt()
			if (err != nil) != tt.wantErr {
				t.Errorf("AsInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AsInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigEntry_AsBoolean(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    bool
		wantErr bool
	}{
		// True values
		{"true lowercase", "true", true, false},
		{"true uppercase", "TRUE", true, false},
		{"true mixed case", "True", true, false},
		{"yes lowercase", "yes", true, false},
		{"yes uppercase", "YES", true, false},
		{"one", "1", true, false},
		{"on lowercase", "on", true, false},
		{"on uppercase", "ON", true, false},

		// False values
		{"false lowercase", "false", false, false},
		{"false uppercase", "FALSE", false, false},
		{"no lowercase", "no", false, false},
		{"no uppercase", "NO", false, false},
		{"zero", "0", false, false},
		{"off lowercase", "off", false, false},
		{"off uppercase", "OFF", false, false},

		// Invalid values
		{"invalid string", "maybe", false, true},
		{"empty string", "", false, true},
		{"number", "42", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test.key", tt.value, UserLevel, ConfigSource("test"), 0)
			got, err := entry.AsBoolean()
			if (err != nil) != tt.wantErr {
				t.Errorf("AsBoolean() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AsBoolean() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigEntry_AsList(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"single item", "item1", []string{"item1"}},
		{"multiple items", "item1,item2,item3", []string{"item1", "item2", "item3"}},
		{"items with spaces", "item1, item2 , item3", []string{"item1", "item2", "item3"}},
		{"empty string", "", []string{}},
		{"trailing comma", "item1,item2,", []string{"item1", "item2"}},
		{"leading comma", ",item1,item2", []string{"item1", "item2"}},
		{"multiple commas", "item1,,item2", []string{"item1", "item2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test.key", tt.value, UserLevel, ConfigSource("test"), 0)
			got := entry.AsList()
			if len(got) != len(tt.want) {
				t.Errorf("AsList() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("AsList()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestConfigEntry_Clone(t *testing.T) {
	original := NewEntry("test.key", "value", UserLevel, ConfigSource("test.json"), 42)
	clone := original.Clone()

	// Check all fields are equal
	if clone.Key != original.Key {
		t.Errorf("Clone() Key = %q, want %q", clone.Key, original.Key)
	}
	if clone.Value != original.Value {
		t.Errorf("Clone() Value = %q, want %q", clone.Value, original.Value)
	}
	if clone.Level != original.Level {
		t.Errorf("Clone() Level = %v, want %v", clone.Level, original.Level)
	}
	if clone.Source != original.Source {
		t.Errorf("Clone() Source = %q, want %q", clone.Source, original.Source)
	}
	if clone.LineNumber != original.LineNumber {
		t.Errorf("Clone() LineNumber = %d, want %d", clone.LineNumber, original.LineNumber)
	}

	// Ensure it's a different instance
	if clone == original {
		t.Error("Clone() returned same instance, want different instance")
	}
}
