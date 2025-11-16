package ignore

import (
	"testing"
)

func TestNewPatternSet(t *testing.T) {
	ps := NewPatternSet()

	if ps == nil {
		t.Fatal("NewPatternSet() returned nil")
	}

	if ps.patterns != nil {
		t.Errorf("patterns should be nil, got %v", ps.patterns)
	}

	if ps.negationPatterns != nil {
		t.Errorf("negationPatterns should be nil, got %v", ps.negationPatterns)
	}
}

func TestPatternSet_Add(t *testing.T) {
	tests := []struct {
		name                      string
		patterns                  []string
		wantPatternsCount         int
		wantNegationPatternsCount int
	}{
		{
			name:                      "add single ignore pattern",
			patterns:                  []string{"*.log"},
			wantPatternsCount:         1,
			wantNegationPatternsCount: 0,
		},
		{
			name:                      "add single negation pattern",
			patterns:                  []string{"!important.log"},
			wantPatternsCount:         0,
			wantNegationPatternsCount: 1,
		},
		{
			name:                      "add mixed patterns",
			patterns:                  []string{"*.log", "*.tmp", "!important.log", "!keep.tmp"},
			wantPatternsCount:         2,
			wantNegationPatternsCount: 2,
		},
		{
			name:                      "add directory patterns",
			patterns:                  []string{"build/", "!dist/"},
			wantPatternsCount:         1,
			wantNegationPatternsCount: 1,
		},
		{
			name:                      "add rooted patterns",
			patterns:                  []string{"/README.md", "!/LICENSE"},
			wantPatternsCount:         1,
			wantNegationPatternsCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPatternSet()

			for _, p := range tt.patterns {
				pattern := NewIgnorePattern(p, ".sourceignore", 1)
				ps.Add(&pattern)
			}

			if len(ps.patterns) != tt.wantPatternsCount {
				t.Errorf("patterns count = %d, want %d", len(ps.patterns), tt.wantPatternsCount)
			}

			if len(ps.negationPatterns) != tt.wantNegationPatternsCount {
				t.Errorf("negationPatterns count = %d, want %d", len(ps.negationPatterns), tt.wantNegationPatternsCount)
			}
		})
	}
}

func TestPatternSet_AddPatternsFromText(t *testing.T) {
	tests := []struct {
		name                      string
		text                      string
		source                    string
		wantPatternsCount         int
		wantNegationPatternsCount int
	}{
		{
			name:                      "single pattern",
			text:                      "*.log",
			source:                    ".sourceignore",
			wantPatternsCount:         1,
			wantNegationPatternsCount: 0,
		},
		{
			name:                      "multiple patterns",
			text:                      "*.log\n*.tmp\n*.bak",
			source:                    ".sourceignore",
			wantPatternsCount:         3,
			wantNegationPatternsCount: 0,
		},
		{
			name:                      "patterns with comments",
			text:                      "# This is a comment\n*.log\n# Another comment\n*.tmp",
			source:                    ".sourceignore",
			wantPatternsCount:         2,
			wantNegationPatternsCount: 0,
		},
		{
			name:                      "patterns with empty lines",
			text:                      "*.log\n\n*.tmp\n\n\n*.bak",
			source:                    ".sourceignore",
			wantPatternsCount:         3,
			wantNegationPatternsCount: 0,
		},
		{
			name:                      "mixed patterns and negations",
			text:                      "*.log\n!important.log\n*.tmp\n!keep.tmp",
			source:                    ".sourceignore",
			wantPatternsCount:         2,
			wantNegationPatternsCount: 2,
		},
		{
			name:                      "default source",
			text:                      "*.log",
			source:                    "",
			wantPatternsCount:         1,
			wantNegationPatternsCount: 0,
		},
		{
			name:                      "complex patterns",
			text:                      "# Build outputs\nbuild/\ndist/\n*.o\n*.so\n\n# But keep important builds\n!dist/important/\n\n# Logs\n*.log\n!debug.log",
			source:                    ".sourceignore",
			wantPatternsCount:         5,
			wantNegationPatternsCount: 2,
		},
		{
			name:                      "patterns with whitespace",
			text:                      "  *.log  \n\t*.tmp\t\n  \n*.bak",
			source:                    ".sourceignore",
			wantPatternsCount:         3,
			wantNegationPatternsCount: 0,
		},
		{
			name:                      "empty text",
			text:                      "",
			source:                    ".sourceignore",
			wantPatternsCount:         0,
			wantNegationPatternsCount: 0,
		},
		{
			name:                      "only comments and empty lines",
			text:                      "# Comment 1\n\n# Comment 2\n\n",
			source:                    ".sourceignore",
			wantPatternsCount:         0,
			wantNegationPatternsCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPatternSet()
			ps.AddPatternsFromText(tt.text, tt.source)

			if len(ps.patterns) != tt.wantPatternsCount {
				t.Errorf("patterns count = %d, want %d", len(ps.patterns), tt.wantPatternsCount)
			}

			if len(ps.negationPatterns) != tt.wantNegationPatternsCount {
				t.Errorf("negationPatterns count = %d, want %d", len(ps.negationPatterns), tt.wantNegationPatternsCount)
			}

			// Verify source is set correctly
			expectedSource := tt.source
			if expectedSource == "" {
				expectedSource = DefaultSource
			}

			for _, p := range ps.patterns {
				if p.Source != expectedSource {
					t.Errorf("pattern source = %s, want %s", p.Source, expectedSource)
				}
			}

			for _, p := range ps.negationPatterns {
				if p.Source != expectedSource {
					t.Errorf("negation pattern source = %s, want %s", p.Source, expectedSource)
				}
			}
		})
	}
}

func TestPatternSet_IsIgnored(t *testing.T) {
	tests := []struct {
		name          string
		patterns      string
		filePath      string
		isDirectory   bool
		fromDirectory string
		want          bool
	}{
		// Simple ignore patterns
		{
			name:        "ignore log files",
			patterns:    "*.log",
			filePath:    "error.log",
			isDirectory: false,
			want:        true,
		},
		{
			name:        "don't ignore txt files when ignoring log",
			patterns:    "*.log",
			filePath:    "readme.txt",
			isDirectory: false,
			want:        false,
		},
		{
			name:        "ignore directory",
			patterns:    "build/",
			filePath:    "build",
			isDirectory: true,
			want:        true,
		},
		{
			name:        "don't ignore file with directory pattern",
			patterns:    "build/",
			filePath:    "build",
			isDirectory: false,
			want:        false,
		},

		// Negation patterns
		{
			name:        "negation overrides ignore",
			patterns:    "*.log\n!important.log",
			filePath:    "important.log",
			isDirectory: false,
			want:        false,
		},
		{
			name:        "negation doesn't affect other files",
			patterns:    "*.log\n!important.log",
			filePath:    "error.log",
			isDirectory: false,
			want:        true,
		},
		{
			name:        "no match no ignore",
			patterns:    "*.log\n!important.log",
			filePath:    "readme.txt",
			isDirectory: false,
			want:        false,
		},

		// Multiple patterns
		{
			name:        "multiple ignore patterns",
			patterns:    "*.log\n*.tmp\n*.bak",
			filePath:    "error.log",
			isDirectory: false,
			want:        true,
		},
		{
			name:        "multiple ignore patterns - match second",
			patterns:    "*.log\n*.tmp\n*.bak",
			filePath:    "cache.tmp",
			isDirectory: false,
			want:        true,
		},
		{
			name:        "multiple ignore patterns - match third",
			patterns:    "*.log\n*.tmp\n*.bak",
			filePath:    "old.bak",
			isDirectory: false,
			want:        true,
		},

		// Complex negation scenarios
		{
			name:        "multiple negations",
			patterns:    "*.log\n!important.log\n!debug.log",
			filePath:    "important.log",
			isDirectory: false,
			want:        false,
		},
		{
			name:        "multiple negations - second negation",
			patterns:    "*.log\n!important.log\n!debug.log",
			filePath:    "debug.log",
			isDirectory: false,
			want:        false,
		},
		{
			name:        "multiple negations - still ignore others",
			patterns:    "*.log\n!important.log\n!debug.log",
			filePath:    "error.log",
			isDirectory: false,
			want:        true,
		},

		// Rooted patterns
		{
			name:        "rooted pattern - match at root",
			patterns:    "/README.md",
			filePath:    "README.md",
			isDirectory: false,
			want:        true,
		},
		{
			name:        "rooted pattern - match full path",
			patterns:    "/src/main.go",
			filePath:    "src/main.go",
			isDirectory: false,
			want:        true,
		},

		// Directory-specific patterns
		{
			name:          "from directory - match within",
			patterns:      "*.log",
			filePath:      "src/error.log",
			isDirectory:   false,
			fromDirectory: "src",
			want:          true,
		},
		{
			name:          "from directory - don't match outside",
			patterns:      "*.log",
			filePath:      "docs/error.log",
			isDirectory:   false,
			fromDirectory: "src",
			want:          false,
		},

		// Wildcard patterns
		{
			name:        "double asterisk - nested match",
			patterns:    "**/node_modules",
			filePath:    "project/node_modules",
			isDirectory: true,
			want:        true,
		},
		{
			name:        "double asterisk - deeply nested match",
			patterns:    "**/node_modules",
			filePath:    "a/b/c/node_modules",
			isDirectory: true,
			want:        true,
		},
		{
			name:        "double asterisk with extension",
			patterns:    "**/*.test.js",
			filePath:    "src/utils/helper.test.js",
			isDirectory: false,
			want:        true,
		},

		// Comments and empty lines (should be ignored)
		{
			name:        "with comments",
			patterns:    "# Ignore logs\n*.log\n# End",
			filePath:    "error.log",
			isDirectory: false,
			want:        true,
		},
		{
			name:        "with empty lines",
			patterns:    "*.log\n\n*.tmp\n\n",
			filePath:    "cache.tmp",
			isDirectory: false,
			want:        true,
		},

		// Edge cases
		{
			name:        "empty pattern set",
			patterns:    "",
			filePath:    "anything.txt",
			isDirectory: false,
			want:        false,
		},
		{
			name:        "only comments",
			patterns:    "# Comment 1\n# Comment 2",
			filePath:    "anything.txt",
			isDirectory: false,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPatternSet()
			ps.AddPatternsFromText(tt.patterns, ".sourceignore")

			got := ps.IsIgnored(tt.filePath, tt.isDirectory, tt.fromDirectory)

			if got != tt.want {
				t.Errorf("IsIgnored(%q, %v, %q) = %v, want %v",
					tt.filePath, tt.isDirectory, tt.fromDirectory, got, tt.want)
			}
		})
	}
}

func TestPatternSet_Clear(t *testing.T) {
	ps := NewPatternSet()
	ps.AddPatternsFromText("*.log\n!important.log\n*.tmp", ".sourceignore")

	// Verify patterns were added
	if len(ps.patterns) == 0 {
		t.Fatal("patterns should not be empty before Clear")
	}
	if len(ps.negationPatterns) == 0 {
		t.Fatal("negationPatterns should not be empty before Clear")
	}

	// Clear the patterns
	ps.Clear()

	// Verify patterns were cleared
	if ps.patterns != nil {
		t.Errorf("patterns should be nil after Clear, got %v", ps.patterns)
	}
	if ps.negationPatterns != nil {
		t.Errorf("negationPatterns should be nil after Clear, got %v", ps.negationPatterns)
	}

	// Verify IsIgnored returns false after clear
	if ps.IsIgnored("error.log", false, "") {
		t.Error("IsIgnored should return false after Clear")
	}
}

func TestPatternSet_IgnoredPatterns(t *testing.T) {
	ps := NewPatternSet()
	ps.AddPatternsFromText("*.log\n*.tmp\n!important.log", ".sourceignore")

	patterns := ps.IgnoredPatterns()

	if len(patterns) != 2 {
		t.Errorf("IgnoredPatterns() count = %d, want 2", len(patterns))
	}

	// Verify the patterns are the ignore patterns, not negation patterns
	for _, p := range patterns {
		if p.IsNegation {
			t.Errorf("IgnoredPatterns() returned negation pattern: %s", p.Pattern)
		}
	}
}

func TestPatternSet_UnignoredPatterns(t *testing.T) {
	ps := NewPatternSet()
	ps.AddPatternsFromText("*.log\n*.tmp\n!important.log\n!debug.log", ".sourceignore")

	patterns := ps.UnignoredPatterns()

	if len(patterns) != 2 {
		t.Errorf("UnignoredPatterns() count = %d, want 2", len(patterns))
	}

	// Verify the patterns are negation patterns
	for _, p := range patterns {
		if !p.IsNegation {
			t.Errorf("UnignoredPatterns() returned non-negation pattern: %s", p.Pattern)
		}
	}
}

func TestPatternSet_MultipleOperations(t *testing.T) {
	ps := NewPatternSet()

	// Add patterns from text
	ps.AddPatternsFromText("*.log\n*.tmp", ".sourceignore")

	// Add individual patterns
	pattern1 := NewIgnorePattern("*.bak", ".sourceignore", 10)
	ps.Add(&pattern1)

	pattern2 := NewIgnorePattern("!important.bak", ".sourceignore", 11)
	ps.Add(&pattern2)

	// Verify all patterns were added
	if len(ps.patterns) != 3 {
		t.Errorf("patterns count = %d, want 3", len(ps.patterns))
	}
	if len(ps.negationPatterns) != 1 {
		t.Errorf("negationPatterns count = %d, want 1", len(ps.negationPatterns))
	}

	// Test matching
	if !ps.IsIgnored("error.log", false, "") {
		t.Error("should ignore error.log")
	}
	if !ps.IsIgnored("cache.tmp", false, "") {
		t.Error("should ignore cache.tmp")
	}
	if !ps.IsIgnored("old.bak", false, "") {
		t.Error("should ignore old.bak")
	}
	if ps.IsIgnored("important.bak", false, "") {
		t.Error("should not ignore important.bak (negated)")
	}

	// Clear and verify
	ps.Clear()
	if ps.IsIgnored("error.log", false, "") {
		t.Error("should not ignore after Clear")
	}
}

func TestPatternSet_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name     string
		patterns string
		checks   []struct {
			filePath    string
			isDirectory bool
			wantIgnored bool
		}
	}{
		{
			name: "gitignore-like patterns",
			patterns: `# Dependencies
node_modules/
*.lock

# Build outputs
build/
dist/
*.o

# But keep some files
!dist/README.md
!build/important/`,
			checks: []struct {
				filePath    string
				isDirectory bool
				wantIgnored bool
			}{
				{"node_modules", true, true},
				{"package.lock", false, true},
				{"build", true, true},
				{"dist", true, true},
				{"test.o", false, true},
				{"dist/README.md", false, false},
				{"build/important", true, false},
				{"src/main.js", false, false},
			},
		},
		{
			name: "nested patterns with wildcards",
			patterns: `**/test/**
*.test.js
**/*.test.js
!important.test.js
!**/important.test.js`,
			checks: []struct {
				filePath    string
				isDirectory bool
				wantIgnored bool
			}{
				{"src/test/file.js", false, true},
				{"utils.test.js", false, true},
				{"src/utils.test.js", false, true},
				{"important.test.js", false, false},
				{"src/important.test.js", false, false},
				{"main.js", false, false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPatternSet()
			ps.AddPatternsFromText(tt.patterns, ".sourceignore")

			for _, check := range tt.checks {
				got := ps.IsIgnored(check.filePath, check.isDirectory, "")
				if got != check.wantIgnored {
					t.Errorf("IsIgnored(%q, %v) = %v, want %v",
						check.filePath, check.isDirectory, got, check.wantIgnored)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkPatternSet_Add(b *testing.B) {
	pattern := NewIgnorePattern("*.log", ".sourceignore", 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps := NewPatternSet()
		ps.Add(&pattern)
	}
}

func BenchmarkPatternSet_AddPatternsFromText(b *testing.B) {
	text := `# Build outputs
build/
dist/
*.o
*.so

# Logs
*.log
*.tmp

# Dependencies
node_modules/
vendor/`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps := NewPatternSet()
		ps.AddPatternsFromText(text, ".sourceignore")
	}
}

func BenchmarkPatternSet_IsIgnored_Simple(b *testing.B) {
	ps := NewPatternSet()
	ps.AddPatternsFromText("*.log\n*.tmp", ".sourceignore")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.IsIgnored("error.log", false, "")
	}
}

func BenchmarkPatternSet_IsIgnored_WithNegation(b *testing.B) {
	ps := NewPatternSet()
	ps.AddPatternsFromText("*.log\n!important.log", ".sourceignore")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.IsIgnored("important.log", false, "")
	}
}

func BenchmarkPatternSet_IsIgnored_Complex(b *testing.B) {
	ps := NewPatternSet()
	ps.AddPatternsFromText(`**/test/**
**/*.test.js
!**/important.test.js
build/
dist/
*.o`, ".sourceignore")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.IsIgnored("src/utils/helper.test.js", false, "")
	}
}

func BenchmarkPatternSet_IsIgnored_ManyPatterns(b *testing.B) {
	ps := NewPatternSet()
	text := ""
	for i := 0; i < 100; i++ {
		text += "*.log\n"
	}
	ps.AddPatternsFromText(text, ".sourceignore")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.IsIgnored("error.log", false, "")
	}
}
