package ignore

import (
	"testing"
)

func TestNewPatternConfig(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		wantNegation bool
		wantDirOnly  bool
		wantRooted   bool
		wantCleaned  string
	}{
		{
			name:         "simple pattern",
			pattern:      "*.log",
			wantNegation: false,
			wantDirOnly:  false,
			wantRooted:   false,
			wantCleaned:  "*.log",
		},
		{
			name:         "negation pattern",
			pattern:      "!important.log",
			wantNegation: true,
			wantDirOnly:  false,
			wantRooted:   false,
			wantCleaned:  "important.log",
		},
		{
			name:         "directory only pattern",
			pattern:      "build/",
			wantNegation: false,
			wantDirOnly:  true,
			wantRooted:   false,
			wantCleaned:  "build",
		},
		{
			name:         "rooted pattern",
			pattern:      "/TODO",
			wantNegation: false,
			wantDirOnly:  false,
			wantRooted:   true,
			wantCleaned:  "TODO",
		},
		{
			name:         "negation + directory",
			pattern:      "!temp/",
			wantNegation: true,
			wantDirOnly:  true,
			wantRooted:   false,
			wantCleaned:  "temp",
		},
		{
			name:         "rooted + directory",
			pattern:      "/build/",
			wantNegation: false,
			wantDirOnly:  true,
			wantRooted:   true,
			wantCleaned:  "build",
		},
		{
			name:         "negation + rooted + directory",
			pattern:      "!/dist/",
			wantNegation: true,
			wantDirOnly:  true,
			wantRooted:   true,
			wantCleaned:  "dist",
		},
		{
			name:         "pattern with whitespace",
			pattern:      "  test.txt  ",
			wantNegation: false,
			wantDirOnly:  false,
			wantRooted:   false,
			wantCleaned:  "test.txt",
		},
		{
			name:         "wildcard pattern",
			pattern:      "**/node_modules",
			wantNegation: false,
			wantDirOnly:  false,
			wantRooted:   false,
			wantCleaned:  "**/node_modules",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewPatternConfig(tt.pattern)

			if config.IsNegation != tt.wantNegation {
				t.Errorf("IsNegation = %v, want %v", config.IsNegation, tt.wantNegation)
			}
			if config.IsDirOnly != tt.wantDirOnly {
				t.Errorf("IsDirOnly = %v, want %v", config.IsDirOnly, tt.wantDirOnly)
			}
			if config.IsRooted != tt.wantRooted {
				t.Errorf("IsRooted = %v, want %v", config.IsRooted, tt.wantRooted)
			}
			if config.CleanedPattern != tt.wantCleaned {
				t.Errorf("CleanedPattern = %v, want %v", config.CleanedPattern, tt.wantCleaned)
			}
		})
	}
}

func TestNewIgnorePattern(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		source         string
		lineNumber     int
		wantPattern    string
		wantOriginal   string
		wantIsNegation bool
		wantIsDirOnly  bool
		wantIsRooted   bool
		wantSource     string
	}{
		{
			name:           "simple pattern",
			pattern:        "*.log",
			source:         ".sourceignore",
			lineNumber:     1,
			wantPattern:    "*.log",
			wantOriginal:   "*.log",
			wantIsNegation: false,
			wantIsDirOnly:  false,
			wantIsRooted:   false,
			wantSource:     ".sourceignore",
		},
		{
			name:           "negated pattern",
			pattern:        "!important.txt",
			source:         ".sourceignore",
			lineNumber:     5,
			wantPattern:    "important.txt",
			wantOriginal:   "!important.txt",
			wantIsNegation: true,
			wantIsDirOnly:  false,
			wantIsRooted:   false,
			wantSource:     ".sourceignore",
		},
		{
			name:           "default source",
			pattern:        "test",
			source:         "",
			lineNumber:     1,
			wantPattern:    "test",
			wantOriginal:   "test",
			wantIsNegation: false,
			wantIsDirOnly:  false,
			wantIsRooted:   false,
			wantSource:     DefaultSource,
		},
		{
			name:           "escaped pattern",
			pattern:        `\*.txt`,
			source:         ".sourceignore",
			lineNumber:     1,
			wantPattern:    "*.txt",
			wantOriginal:   `\*.txt`,
			wantIsNegation: false,
			wantIsDirOnly:  false,
			wantIsRooted:   false,
			wantSource:     ".sourceignore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := NewIgnorePattern(tt.pattern, tt.source, tt.lineNumber)

			if pattern.Pattern != tt.wantPattern {
				t.Errorf("Pattern = %v, want %v", pattern.Pattern, tt.wantPattern)
			}
			if pattern.OriginalPattern != tt.wantOriginal {
				t.Errorf("OriginalPattern = %v, want %v", pattern.OriginalPattern, tt.wantOriginal)
			}
			if pattern.IsNegation != tt.wantIsNegation {
				t.Errorf("IsNegation = %v, want %v", pattern.IsNegation, tt.wantIsNegation)
			}
			if pattern.IsDirOnly != tt.wantIsDirOnly {
				t.Errorf("IsDirOnly = %v, want %v", pattern.IsDirOnly, tt.wantIsDirOnly)
			}
			if pattern.IsRooted != tt.wantIsRooted {
				t.Errorf("IsRooted = %v, want %v", pattern.IsRooted, tt.wantIsRooted)
			}
			if pattern.Source != tt.wantSource {
				t.Errorf("Source = %v, want %v", pattern.Source, tt.wantSource)
			}
			if pattern.LineNumber != tt.lineNumber {
				t.Errorf("LineNumber = %v, want %v", pattern.LineNumber, tt.lineNumber)
			}
		})
	}
}

func TestFromLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		source      string
		lineNumber  int
		wantNil     bool
		wantPattern string
	}{
		{
			name:        "valid pattern",
			line:        "*.log",
			source:      ".sourceignore",
			lineNumber:  1,
			wantNil:     false,
			wantPattern: "*.log",
		},
		{
			name:       "empty line",
			line:       "",
			source:     ".sourceignore",
			lineNumber: 2,
			wantNil:    true,
		},
		{
			name:       "whitespace only",
			line:       "   ",
			source:     ".sourceignore",
			lineNumber: 3,
			wantNil:    true,
		},
		{
			name:       "comment line",
			line:       "# This is a comment",
			source:     ".sourceignore",
			lineNumber: 4,
			wantNil:    true,
		},
		{
			name:        "pattern with trailing whitespace",
			line:        "test.txt   ",
			source:      ".sourceignore",
			lineNumber:  5,
			wantNil:     false,
			wantPattern: "test.txt",
		},
		{
			name:        "pattern with escaped trailing space",
			line:        `test.txt\ `,
			source:      ".sourceignore",
			lineNumber:  6,
			wantNil:     false,
			wantPattern: "test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := FromLine(tt.line, tt.source, tt.lineNumber)

			if tt.wantNil {
				if pattern != nil {
					t.Errorf("FromLine() = %v, want nil", pattern)
				}
				return
			}

			if pattern == nil {
				t.Fatalf("FromLine() = nil, want non-nil")
			}

			if pattern.Pattern != tt.wantPattern {
				t.Errorf("Pattern = %v, want %v", pattern.Pattern, tt.wantPattern)
			}
		})
	}
}

func TestIgnorePattern_Matches(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		filePath      string
		isDirectory   bool
		fromDirectory string
		want          bool
	}{
		// Simple pattern matching
		{
			name:        "exact match file",
			pattern:     "test.txt",
			filePath:    "test.txt",
			isDirectory: false,
			want:        true,
		},
		{
			name:        "no match",
			pattern:     "test.txt",
			filePath:    "other.txt",
			isDirectory: false,
			want:        false,
		},
		{
			name:        "wildcard match",
			pattern:     "*.log",
			filePath:    "error.log",
			isDirectory: false,
			want:        true,
		},
		{
			name:        "wildcard no match",
			pattern:     "*.log",
			filePath:    "test.txt",
			isDirectory: false,
			want:        false,
		},
		// Directory patterns
		{
			name:        "directory only - matches dir",
			pattern:     "build/",
			filePath:    "build",
			isDirectory: true,
			want:        true,
		},
		{
			name:        "directory only - no match file",
			pattern:     "build/",
			filePath:    "build",
			isDirectory: false,
			want:        false,
		},
		{
			name:        "directory only - matches child dir",
			pattern:     "build/",
			filePath:    "build/dist",
			isDirectory: true,
			want:        true,
		},
		// Rooted patterns
		{
			name:        "rooted pattern - match at root",
			pattern:     "/README.md",
			filePath:    "README.md",
			isDirectory: false,
			want:        true,
		},
		{
			name:        "rooted pattern - matches only full path",
			pattern:     "/src/main.go",
			filePath:    "src/main.go",
			isDirectory: false,
			want:        true,
		},
		// Non-rooted patterns (match anywhere)
		{
			name:        "non-rooted - match in subdir",
			pattern:     "test.txt",
			filePath:    "src/test.txt",
			isDirectory: false,
			want:        true,
		},
		{
			name:        "non-rooted - match nested",
			pattern:     "test.txt",
			filePath:    "a/b/c/test.txt",
			isDirectory: false,
			want:        true,
		},
		// Double asterisk patterns
		{
			name:        "double asterisk - match nested",
			pattern:     "**/node_modules",
			filePath:    "project/node_modules",
			isDirectory: true,
			want:        true,
		},
		{
			name:        "double asterisk - match deeply nested",
			pattern:     "**/node_modules",
			filePath:    "a/b/c/node_modules",
			isDirectory: true,
			want:        true,
		},
		{
			name:        "double asterisk with extension",
			pattern:     "**/*.test.js",
			filePath:    "src/utils/helper.test.js",
			isDirectory: false,
			want:        true,
		},
		// From directory
		{
			name:          "from directory - match within",
			pattern:       "*.txt",
			filePath:      "src/test.txt",
			isDirectory:   false,
			fromDirectory: "src",
			want:          true,
		},
		{
			name:          "from directory - no match outside",
			pattern:       "*.txt",
			filePath:      "docs/test.txt",
			isDirectory:   false,
			fromDirectory: "src",
			want:          false,
		},
		{
			name:          "from directory - relative match",
			pattern:       "utils/helper.js",
			filePath:      "src/utils/helper.js",
			isDirectory:   false,
			fromDirectory: "src",
			want:          true,
		},
		// Path normalization
		{
			name:        "path with backslashes",
			pattern:     "src/main.go",
			filePath:    "src\\main.go",
			isDirectory: false,
			want:        true,
		},
		{
			name:        "path with mixed slashes",
			pattern:     "src/utils/helper.js",
			filePath:    "src/utils\\helper.js",
			isDirectory: false,
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := NewIgnorePattern(tt.pattern, ".sourceignore", 1)
			got := pattern.Matches(tt.filePath, tt.isDirectory, tt.fromDirectory)

			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIgnorePattern_Matches_Security(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		filePath    string
		isDirectory bool
		wantBlocked bool
	}{
		{
			name:        "directory traversal attempt with ..",
			pattern:     "*.txt",
			filePath:    "../etc/passwd",
			isDirectory: false,
			wantBlocked: true,
		},
		{
			name:        "absolute path attempt",
			pattern:     "*.txt",
			filePath:    "/etc/passwd",
			isDirectory: false,
			wantBlocked: true,
		},
		{
			name:        "valid relative path",
			pattern:     "*.txt",
			filePath:    "src/test.txt",
			isDirectory: false,
			wantBlocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := NewIgnorePattern(tt.pattern, ".sourceignore", 1)
			got := pattern.Matches(tt.filePath, tt.isDirectory, "")

			// If wantBlocked is true, Matches should return false (path is blocked/rejected)
			// If wantBlocked is false, we're testing normal matching behavior
			if tt.wantBlocked && got {
				t.Errorf("Matches() = true for unsafe path %q, should be blocked", tt.filePath)
			}
		})
	}
}

func TestTrimTrailingWhitespace(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "no whitespace",
			line: "test.txt",
			want: "test.txt",
		},
		{
			name: "trailing spaces",
			line: "test.txt   ",
			want: "test.txt",
		},
		{
			name: "trailing tabs",
			line: "test.txt\t\t",
			want: "test.txt",
		},
		{
			name: "trailing mixed whitespace",
			line: "test.txt  \t ",
			want: "test.txt",
		},
		{
			name: "trailing backslash (odd)",
			line: `test.txt\`,
			want: `test.txt\`,
		},
		{
			name: "trailing backslashes (even)",
			line: `test.txt\\`,
			want: `test.txt\\`,
		},
		{
			name: "trailing backslashes (odd) with space",
			line: `test.txt\\\`,
			want: `test.txt\\\`,
		},
		{
			name: "leading whitespace preserved",
			line: "  test.txt",
			want: "  test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimTrailingWhitespace(tt.line)
			if got != tt.want {
				t.Errorf("trimTrailingWhitespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnescapePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "no escapes",
			pattern: "test.txt",
			want:    "test.txt",
		},
		{
			name:    "escaped asterisk",
			pattern: `\*.txt`,
			want:    "*.txt",
		},
		{
			name:    "escaped question mark",
			pattern: `test\?.txt`,
			want:    "test?.txt",
		},
		{
			name:    "escaped backslash",
			pattern: `test\\file`,
			want:    `test\file`,
		},
		{
			name:    "multiple escapes",
			pattern: `\*\?\[test\]`,
			want:    "*?[test]",
		},
		{
			name:    "escaped space",
			pattern: `test\ file`,
			want:    "test file",
		},
		{
			name:    "double backslash",
			pattern: `\\`,
			want:    `\`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unescapePattern(tt.pattern)
			if got != tt.want {
				t.Errorf("unescapePattern() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContainsWildcard(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{
			name:    "no wildcard",
			pattern: "test.txt",
			want:    false,
		},
		{
			name:    "single asterisk",
			pattern: "*.txt",
			want:    true,
		},
		{
			name:    "double asterisk",
			pattern: "**/node_modules",
			want:    true,
		},
		{
			name:    "question mark",
			pattern: "test?.txt",
			want:    true,
		},
		{
			name:    "square brackets",
			pattern: "test[0-9].txt",
			want:    true,
		},
		{
			name:    "curly braces",
			pattern: "*.{js,ts}",
			want:    true,
		},
		{
			name:    "multiple wildcards",
			pattern: "src/**/*.test.js",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsWildcard(tt.pattern)
			if got != tt.want {
				t.Errorf("containsWildcard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		pattern   string
		isDirOnly bool
		want      bool
	}{
		// Exact matching (no wildcards)
		{
			name:    "exact match",
			path:    "test.txt",
			pattern: "test.txt",
			want:    true,
		},
		{
			name:    "exact match - basename",
			path:    "src/test.txt",
			pattern: "test.txt",
			want:    true,
		},
		{
			name:    "exact match - full path",
			path:    "src/utils/helper.js",
			pattern: "src/utils/helper.js",
			want:    true,
		},
		{
			name:    "no match - different name",
			path:    "test.txt",
			pattern: "other.txt",
			want:    false,
		},
		// Wildcard matching
		{
			name:    "single asterisk - extension",
			path:    "test.log",
			pattern: "*.log",
			want:    true,
		},
		{
			name:    "single asterisk - prefix",
			path:    "test.txt",
			pattern: "test.*",
			want:    true,
		},
		{
			name:    "question mark",
			path:    "test1.txt",
			pattern: "test?.txt",
			want:    true,
		},
		// Double asterisk
		{
			name:    "double asterisk - nested",
			path:    "a/b/c/test.txt",
			pattern: "**/test.txt",
			want:    true,
		},
		{
			name:    "double asterisk - extension",
			path:    "src/utils/helper.test.js",
			pattern: "**/*.test.js",
			want:    true,
		},
		// Directory only
		{
			name:      "directory only - match dir",
			path:      "build",
			pattern:   "build",
			isDirOnly: true,
			want:      true,
		},
		{
			name:      "directory only - match children",
			path:      "build/output.js",
			pattern:   "build",
			isDirOnly: true,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPattern(tt.path, tt.pattern, tt.isDirOnly)
			if got != tt.want {
				t.Errorf("matchPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchAnySubpath(t *testing.T) {
	tests := []struct {
		name      string
		testPath  string
		pattern   string
		isDirOnly bool
		want      bool
	}{
		{
			name:     "match at root",
			testPath: "test.txt",
			pattern:  "test.txt",
			want:     true,
		},
		{
			name:     "match in subdir",
			testPath: "src/test.txt",
			pattern:  "test.txt",
			want:     true,
		},
		{
			name:     "match nested",
			testPath: "a/b/c/test.txt",
			pattern:  "test.txt",
			want:     true,
		},
		{
			name:     "no match",
			testPath: "src/other.txt",
			pattern:  "test.txt",
			want:     false,
		},
		{
			name:     "match partial path",
			testPath: "src/utils/helper.js",
			pattern:  "utils/helper.js",
			want:     true,
		},
		{
			name:     "match with wildcard",
			testPath: "src/test.log",
			pattern:  "*.log",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchAnySubpath(tt.testPath, tt.pattern, tt.isDirOnly)
			if got != tt.want {
				t.Errorf("matchAnySubpath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "double asterisk",
			pattern: "**/test",
			want:    "^.*/test$",
		},
		{
			name:    "single asterisk",
			pattern: "*.txt",
			want:    "^[^/]*\\.txt$",
		},
		{
			name:    "question mark",
			pattern: "test?.txt",
			want:    "^test[^/]\\.txt$",
		},
		{
			name:    "complex pattern",
			pattern: "src/**/*.test.js",
			want:    "^src/.*/[^/]*\\.test\\.js$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := globToRegex(tt.pattern)
			if got != tt.want {
				t.Errorf("globToRegex() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Benchmark tests
func BenchmarkMatches(b *testing.B) {
	pattern := NewIgnorePattern("**/*.test.js", ".sourceignore", 1)
	filePath := "src/components/Button/Button.test.js"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pattern.Matches(filePath, false, "")
	}
}

func BenchmarkMatchPattern(b *testing.B) {
	path := "src/components/Button.test.js"
	pattern := "**/*.test.js"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchPattern(path, pattern, false)
	}
}

func BenchmarkNewPatternConfig(b *testing.B) {
	pattern := "!/build/"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewPatternConfig(pattern)
	}
}
