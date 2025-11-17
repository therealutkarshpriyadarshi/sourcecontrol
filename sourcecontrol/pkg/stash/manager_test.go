package stash

import (
	"testing"
)

func TestParseStashName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantIndex int
		wantErr   bool
	}{
		{
			name:      "stash@{0} format",
			input:     "stash@{0}",
			wantIndex: 0,
			wantErr:   false,
		},
		{
			name:      "stash@{5} format",
			input:     "stash@{5}",
			wantIndex: 5,
			wantErr:   false,
		},
		{
			name:      "numeric format",
			input:     "3",
			wantIndex: 3,
			wantErr:   false,
		},
		{
			name:      "invalid format",
			input:     "stash",
			wantIndex: 0,
			wantErr:   true,
		},
		{
			name:      "invalid number",
			input:     "stash@{abc}",
			wantIndex: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIndex, err := ParseStashName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStashName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotIndex != tt.wantIndex {
				t.Errorf("ParseStashName() = %v, want %v", gotIndex, tt.wantIndex)
			}
		})
	}
}

func TestGetStashName(t *testing.T) {
	tests := []struct {
		name  string
		index int
		want  string
	}{
		{
			name:  "index 0",
			index: 0,
			want:  "stash@{0}",
		},
		{
			name:  "index 5",
			index: 5,
			want:  "stash@{5}",
		},
		{
			name:  "index 10",
			index: 10,
			want:  "stash@{10}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetStashName(tt.index); got != tt.want {
				t.Errorf("GetStashName() = %v, want %v", got, tt.want)
			}
		})
	}
}
