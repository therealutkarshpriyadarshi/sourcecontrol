package scpath

import (
	"path/filepath"
	"testing"
)

func TestRepositoryPath_IsValid(t *testing.T) {
	// Use platform-appropriate absolute paths
	var validAbsPath string
	if filepath.Separator == '\\' {
		// Windows
		validAbsPath = "C:\\Users\\user\\repo"
	} else {
		// Unix-like (Linux, macOS)
		validAbsPath = "/home/user/repo"
	}

	tests := []struct {
		name  string
		path  RepositoryPath
		valid bool
	}{
		{
			name:  "valid absolute path",
			path:  RepositoryPath(validAbsPath),
			valid: true,
		},
		{
			name:  "invalid relative path",
			path:  RepositoryPath("relative/path"),
			valid: false,
		},
		{
			name:  "empty path",
			path:  RepositoryPath(""),
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v for path %q", got, tt.valid, tt.path)
			}
		})
	}
}

func TestRepositoryPath_SourcePath(t *testing.T) {
	repo := RepositoryPath("/home/user/repo")
	sourcePath := repo.SourcePath()

	expected := filepath.Join("/home/user/repo", SourceDir)
	if sourcePath.String() != expected {
		t.Errorf("SourcePath() = %v, want %v", sourcePath, expected)
	}
}

func TestRepositoryPath_ObjectsPath(t *testing.T) {
	repo := RepositoryPath("/home/user/repo")
	objectsPath := repo.SourcePath().ObjectsPath()

	expected := filepath.Join("/home/user/repo", SourceDir, "objects")
	if objectsPath.String() != expected {
		t.Errorf("ObjectsPath() = %v, want %v", objectsPath, expected)
	}
}

func TestRelativePath_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		path     RelativePath
		expected RelativePath
	}{
		{
			name:     "already normalized",
			path:     RelativePath("src/main.go"),
			expected: RelativePath("src/main.go"),
		},
		{
			name:     "with leading ./",
			path:     RelativePath("./src/main.go"),
			expected: RelativePath("src/main.go"),
		},
		{
			name:     "with backslashes",
			path:     RelativePath("src\\main.go"),
			expected: RelativePath("src/main.go"),
		},
		{
			name:     "with redundant slashes",
			path:     RelativePath("src//main.go"),
			expected: RelativePath("src/main.go"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.path.Normalize()
			if got != tt.expected {
				t.Errorf("Normalize() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRelativePath_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		path  RelativePath
		valid bool
	}{
		{
			name:  "valid relative path",
			path:  RelativePath("src/main.go"),
			valid: true,
		},
		{
			name:  "absolute path",
			path:  RelativePath("/home/user/file.go"),
			valid: false,
		},
		{
			name:  "contains ..",
			path:  RelativePath("../other/file.go"),
			valid: false,
		},
		{
			name:  "empty path",
			path:  RelativePath(""),
			valid: false,
		},
		{
			name:  "single file",
			path:  RelativePath("README.md"),
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v for path %q", got, tt.valid, tt.path)
			}
		})
	}
}

func TestRelativePath_Components(t *testing.T) {
	tests := []struct {
		name     string
		path     RelativePath
		expected []string
	}{
		{
			name:     "simple path",
			path:     RelativePath("src/main.go"),
			expected: []string{"src", "main.go"},
		},
		{
			name:     "single component",
			path:     RelativePath("README.md"),
			expected: []string{"README.md"},
		},
		{
			name:     "deep path",
			path:     RelativePath("a/b/c/d/file.txt"),
			expected: []string{"a", "b", "c", "d", "file.txt"},
		},
		{
			name:     "empty path",
			path:     RelativePath(""),
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.path.Components()
			if len(got) != len(tt.expected) {
				t.Errorf("Components() length = %v, want %v", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("Components()[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestRelativePath_Depth(t *testing.T) {
	tests := []struct {
		name  string
		path  RelativePath
		depth int
	}{
		{
			name:  "single file",
			path:  RelativePath("README.md"),
			depth: 1,
		},
		{
			name:  "nested path",
			path:  RelativePath("src/main.go"),
			depth: 2,
		},
		{
			name:  "deep path",
			path:  RelativePath("a/b/c/d/e.txt"),
			depth: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.Depth(); got != tt.depth {
				t.Errorf("Depth() = %v, want %v", got, tt.depth)
			}
		})
	}
}

func TestRelativePath_Join(t *testing.T) {
	tests := []struct {
		name     string
		base     RelativePath
		elements []string
		expected RelativePath
	}{
		{
			name:     "join single element",
			base:     RelativePath("src"),
			elements: []string{"main.go"},
			expected: RelativePath("src/main.go"),
		},
		{
			name:     "join multiple elements",
			base:     RelativePath("src"),
			elements: []string{"utils", "helper.go"},
			expected: RelativePath("src/utils/helper.go"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.base.Join(tt.elements...)
			if got != tt.expected {
				t.Errorf("Join() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSourcePath_ObjectFilePath(t *testing.T) {
	objectsPath := SourcePath(filepath.Join("/repo", SourceDir, "objects"))

	tests := []struct {
		name         string
		hash         string
		expectEmpty  bool
		expectPrefix string
		expectSuffix string
	}{
		{
			name:         "valid hash",
			hash:         "abcdef0123456789abcdef0123456789abcdef01",
			expectEmpty:  false,
			expectPrefix: "ab",
			expectSuffix: "cdef0123456789abcdef0123456789abcdef01",
		},
		{
			name:         "different hash",
			hash:         "1234567890abcdef1234567890abcdef12345678",
			expectEmpty:  false,
			expectPrefix: "12",
			expectSuffix: "34567890abcdef1234567890abcdef12345678",
		},
		{
			name:        "invalid hash - too short",
			hash:        "abcdef",
			expectEmpty: true,
		},
		{
			name:        "invalid hash - too long",
			hash:        "abcdef0123456789abcdef0123456789abcdef012345",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := objectsPath.ObjectFilePath(tt.hash)

			if tt.expectEmpty {
				if got != "" {
					t.Errorf("ObjectFilePath() = %v, want empty string", got)
				}
				return
			}

			// Check if the path contains the expected components
			expected := filepath.Join(objectsPath.String(), tt.expectPrefix, tt.expectSuffix)
			if got.String() != expected {
				t.Errorf("ObjectFilePath() = %v, want %v", got, expected)
			}
		})
	}
}

func TestIsPathSafe(t *testing.T) {
	tests := []struct {
		name string
		path string
		safe bool
	}{
		{
			name: "safe relative path",
			path: "src/main.go",
			safe: true,
		},
		{
			name: "contains ..",
			path: "../../../etc/passwd",
			safe: false,
		},
		{
			name: "absolute path",
			path: "/etc/passwd",
			safe: false,
		},
		{
			name: "backslashes",
			path: "src\\main.go",
			safe: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPathSafe(tt.path); got != tt.safe {
				t.Errorf("IsPathSafe() = %v, want %v for path %q", got, tt.safe, tt.path)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "already normalized",
			path:     "src/main.go",
			expected: "src/main.go",
		},
		{
			name:     "with leading ./",
			path:     "./src/main.go",
			expected: "src/main.go",
		},
		{
			name:     "with backslashes",
			path:     "src\\main.go",
			expected: "src/main.go",
		},
		{
			name:     "with trailing slash",
			path:     "src/",
			expected: "src",
		},
		{
			name:     "with redundant slashes",
			path:     "src//main.go",
			expected: "src/main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizePath(tt.path); got != tt.expected {
				t.Errorf("NormalizePath() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRelativePath_IsInSubdir(t *testing.T) {
	tests := []struct {
		name   string
		path   RelativePath
		subdir string
		result bool
	}{
		{
			name:   "file in subdir",
			path:   RelativePath("src/main.go"),
			subdir: "src",
			result: true,
		},
		{
			name:   "file not in subdir",
			path:   RelativePath("docs/README.md"),
			subdir: "src",
			result: false,
		},
		{
			name:   "exact match",
			path:   RelativePath("src"),
			subdir: "src",
			result: true,
		},
		{
			name:   "deep nesting",
			path:   RelativePath("src/utils/helper.go"),
			subdir: "src",
			result: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.IsInSubdir(tt.subdir); got != tt.result {
				t.Errorf("IsInSubdir() = %v, want %v", got, tt.result)
			}
		})
	}
}

func TestRelativePath_Base(t *testing.T) {
	tests := []struct {
		name     string
		path     RelativePath
		expected string
	}{
		{
			name:     "simple path",
			path:     RelativePath("src/main.go"),
			expected: "main.go",
		},
		{
			name:     "single file",
			path:     RelativePath("README.md"),
			expected: "README.md",
		},
		{
			name:     "deep path",
			path:     RelativePath("a/b/c/file.txt"),
			expected: "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.Base(); got != tt.expected {
				t.Errorf("Base() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRelativePath_Dir(t *testing.T) {
	tests := []struct {
		name     string
		path     RelativePath
		expected RelativePath
	}{
		{
			name:     "simple path",
			path:     RelativePath("src/main.go"),
			expected: RelativePath("src"),
		},
		{
			name:     "single file",
			path:     RelativePath("README.md"),
			expected: RelativePath(""),
		},
		{
			name:     "deep path",
			path:     RelativePath("a/b/c/file.txt"),
			expected: RelativePath("a/b/c"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.Dir(); got != tt.expected {
				t.Errorf("Dir() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRepositoryPath_JoinRelative(t *testing.T) {
	// Use a platform-appropriate repository path
	repo := RepositoryPath(filepath.Join("C:", "repos", "myproject"))

	tests := []struct {
		name        string
		repo        RepositoryPath
		relPath     RelativePath
		expectError bool
		checkResult func(AbsolutePath) bool
	}{
		{
			name:        "simple file",
			repo:        repo,
			relPath:     RelativePath("README.md"),
			expectError: false,
			checkResult: func(ap AbsolutePath) bool {
				return ap.String() == filepath.Join(string(repo), "README.md")
			},
		},
		{
			name:        "nested path",
			repo:        repo,
			relPath:     RelativePath("src/main.go"),
			expectError: false,
			checkResult: func(ap AbsolutePath) bool {
				return ap.String() == filepath.Join(string(repo), "src", "main.go")
			},
		},
		{
			name:        "deep nested path",
			repo:        repo,
			relPath:     RelativePath("a/b/c/d/file.txt"),
			expectError: false,
			checkResult: func(ap AbsolutePath) bool {
				return ap.String() == filepath.Join(string(repo), "a", "b", "c", "d", "file.txt")
			},
		},
		{
			name:        "dot path returns repo root",
			repo:        repo,
			relPath:     RelativePath("."),
			expectError: false,
			checkResult: func(ap AbsolutePath) bool {
				return ap.String() == string(repo)
			},
		},
		{
			name:        "invalid relative path - contains parent dir",
			repo:        repo,
			relPath:     RelativePath("../etc/passwd"),
			expectError: true,
		},
		{
			name:        "invalid relative path - absolute path",
			repo:        repo,
			relPath:     RelativePath("/etc/passwd"),
			expectError: true,
		},
		{
			name:        "normalized path with ./ prefix",
			repo:        repo,
			relPath:     RelativePath("./src/main.go").Normalize(),
			expectError: false,
			checkResult: func(ap AbsolutePath) bool {
				return ap.String() == filepath.Join(string(repo), "src", "main.go")
			},
		},
		{
			name:        "path stays within repository",
			repo:        repo,
			relPath:     RelativePath("src/../docs/README.md").Normalize(),
			expectError: false,
			checkResult: func(ap AbsolutePath) bool {
				// After normalization, this becomes "docs/README.md"
				return ap.String() == filepath.Join(string(repo), "docs", "README.md")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.repo.JoinRelative(tt.relPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("JoinRelative() expected error but got none, result = %v", result)
				}
				return
			}

			if err != nil {
				t.Errorf("JoinRelative() unexpected error: %v", err)
				return
			}

			if tt.checkResult != nil && !tt.checkResult(result) {
				t.Errorf("JoinRelative() = %v, failed validation check", result)
			}
		})
	}
}

func TestAbsolutePath_IsValid(t *testing.T) {
	// Use platform-appropriate absolute paths
	var validAbsPath, volumeRootPath string
	if filepath.Separator == '\\' {
		// Windows
		validAbsPath = "C:\\Users\\user\\repo"
		volumeRootPath = "C:\\"
	} else {
		// Unix-like (Linux, macOS)
		validAbsPath = "/home/user/repo"
		volumeRootPath = "/"
	}

	tests := []struct {
		name  string
		path  AbsolutePath
		valid bool
	}{
		{
			name:  "valid absolute path",
			path:  AbsolutePath(validAbsPath),
			valid: true,
		},
		{
			name:  "invalid relative path",
			path:  AbsolutePath("relative/path"),
			valid: false,
		},
		{
			name:  "empty path",
			path:  AbsolutePath(""),
			valid: false,
		},
		{
			name:  "volume root path",
			path:  AbsolutePath(volumeRootPath),
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v for path %q", got, tt.valid, tt.path)
			}
		})
	}
}

func TestNewAbsolutePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		validate    func(AbsolutePath) bool
	}{
		{
			name:        "empty path",
			input:       "",
			expectError: true,
		},
		{
			name:        "absolute path stays absolute",
			input:       filepath.Join("C:", "Users", "user", "repo"),
			expectError: false,
			validate: func(ap AbsolutePath) bool {
				return ap.IsValid() && filepath.IsAbs(string(ap))
			},
		},
		{
			name:        "relative path becomes absolute",
			input:       "relative/path",
			expectError: false,
			validate: func(ap AbsolutePath) bool {
				// Should be converted to absolute
				return ap.IsValid() && filepath.IsAbs(string(ap))
			},
		},
		{
			name:        "dot path becomes absolute",
			input:       ".",
			expectError: false,
			validate: func(ap AbsolutePath) bool {
				return ap.IsValid() && filepath.IsAbs(string(ap))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewAbsolutePath(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewAbsolutePath() expected error but got none, result = %v", result)
				}
				return
			}

			if err != nil {
				t.Errorf("NewAbsolutePath() unexpected error: %v", err)
				return
			}

			if tt.validate != nil && !tt.validate(result) {
				t.Errorf("NewAbsolutePath() = %v, failed validation", result)
			}
		})
	}
}
