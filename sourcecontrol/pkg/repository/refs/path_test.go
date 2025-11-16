package refs

import "testing"

func TestRefPath_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		path  RefPath
		valid bool
	}{
		{
			name:  "valid branch ref",
			path:  RefPath("refs/heads/main"),
			valid: true,
		},
		{
			name:  "valid tag ref",
			path:  RefPath("refs/tags/v1.0.0"),
			valid: true,
		},
		{
			name:  "HEAD",
			path:  RefPath("HEAD"),
			valid: true,
		},
		{
			name:  "contains space",
			path:  RefPath("refs/heads/my branch"),
			valid: false,
		},
		{
			name:  "contains ..",
			path:  RefPath("refs/../heads/main"),
			valid: false,
		},
		{
			name:  "ends with .lock",
			path:  RefPath("refs/heads/main.lock"),
			valid: false,
		},
		{
			name:  "starts with .",
			path:  RefPath(".refs/heads/main"),
			valid: false,
		},
		{
			name:  "empty",
			path:  RefPath(""),
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

func TestRefPath_IsBranch(t *testing.T) {
	tests := []struct {
		name     string
		path     RefPath
		isBranch bool
	}{
		{
			name:     "branch ref",
			path:     RefPath("refs/heads/main"),
			isBranch: true,
		},
		{
			name:     "tag ref",
			path:     RefPath("refs/tags/v1.0.0"),
			isBranch: false,
		},
		{
			name:     "HEAD",
			path:     RefPath("HEAD"),
			isBranch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.IsBranch(); got != tt.isBranch {
				t.Errorf("IsBranch() = %v, want %v", got, tt.isBranch)
			}
		})
	}
}

func TestRefPath_IsTag(t *testing.T) {
	tests := []struct {
		name  string
		path  RefPath
		isTag bool
	}{
		{
			name:  "tag ref",
			path:  RefPath("refs/tags/v1.0.0"),
			isTag: true,
		},
		{
			name:  "branch ref",
			path:  RefPath("refs/heads/main"),
			isTag: false,
		},
		{
			name:  "HEAD",
			path:  RefPath("HEAD"),
			isTag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.IsTag(); got != tt.isTag {
				t.Errorf("IsTag() = %v, want %v", got, tt.isTag)
			}
		})
	}
}

func TestRefPath_ShortName(t *testing.T) {
	tests := []struct {
		name      string
		path      RefPath
		shortName string
	}{
		{
			name:      "branch ref",
			path:      RefPath("refs/heads/main"),
			shortName: "main",
		},
		{
			name:      "tag ref",
			path:      RefPath("refs/tags/v1.0.0"),
			shortName: "v1.0.0",
		},
		{
			name:      "HEAD",
			path:      RefPath("HEAD"),
			shortName: "HEAD",
		},
		{
			name:      "branch with slashes",
			path:      RefPath("refs/heads/feature/new-feature"),
			shortName: "feature/new-feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.ShortName(); got != tt.shortName {
				t.Errorf("ShortName() = %v, want %v", got, tt.shortName)
			}
		})
	}
}

func TestNewBranchRef(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		wantErr  bool
		expected RefPath
	}{
		{
			name:     "valid branch name",
			branch:   "main",
			wantErr:  false,
			expected: RefPath("refs/heads/main"),
		},
		{
			name:     "branch with slashes",
			branch:   "feature/new-feature",
			wantErr:  false,
			expected: RefPath("refs/heads/feature/new-feature"),
		},
		{
			name:    "empty name",
			branch:  "",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			branch:  "my branch",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBranchRef(tt.branch)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBranchRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("NewBranchRef() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewTagRef(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		wantErr  bool
		expected RefPath
	}{
		{
			name:     "valid tag name",
			tag:      "v1.0.0",
			wantErr:  false,
			expected: RefPath("refs/tags/v1.0.0"),
		},
		{
			name:    "empty name",
			tag:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTagRef(tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTagRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("NewTagRef() = %v, want %v", got, tt.expected)
			}
		})
	}
}
