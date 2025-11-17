package merge

import (
	"context"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

// TestMergeConfig tests merge configuration building
func TestMergeConfig(t *testing.T) {
	tests := []struct {
		name     string
		opts     []MergeOption
		expected *Config
	}{
		{
			name: "default config",
			opts: []MergeOption{},
			expected: &Config{
				Strategy:                StrategyRecursive,
				Mode:                    ModeDefault,
				ConflictResolution:      ConflictFail,
				AllowUnrelatedHistories: false,
				Verbose:                 false,
			},
		},
		{
			name: "squash merge",
			opts: []MergeOption{WithSquash()},
			expected: &Config{
				Strategy:                StrategyRecursive,
				Mode:                    ModeSquash,
				ConflictResolution:      ConflictFail,
				AllowUnrelatedHistories: false,
				Verbose:                 false,
			},
		},
		{
			name: "fast-forward only",
			opts: []MergeOption{WithFastForwardOnly()},
			expected: &Config{
				Strategy:                StrategyRecursive,
				Mode:                    ModeFastForwardOnly,
				ConflictResolution:      ConflictFail,
				AllowUnrelatedHistories: false,
				Verbose:                 false,
			},
		},
		{
			name: "no fast-forward",
			opts: []MergeOption{WithNoFastForward()},
			expected: &Config{
				Strategy:                StrategyRecursive,
				Mode:                    ModeNoFastForward,
				ConflictResolution:      ConflictFail,
				AllowUnrelatedHistories: false,
				Verbose:                 false,
			},
		},
		{
			name: "custom message and verbose",
			opts: []MergeOption{
				WithMessage("Custom merge message"),
				WithVerbose(),
			},
			expected: &Config{
				Strategy:                StrategyRecursive,
				Mode:                    ModeDefault,
				ConflictResolution:      ConflictFail,
				Message:                 "Custom merge message",
				AllowUnrelatedHistories: false,
				Verbose:                 true,
			},
		},
		{
			name: "octopus strategy",
			opts: []MergeOption{WithStrategy(StrategyOctopus)},
			expected: &Config{
				Strategy:                StrategyOctopus,
				Mode:                    ModeDefault,
				ConflictResolution:      ConflictFail,
				AllowUnrelatedHistories: false,
				Verbose:                 false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			for _, opt := range tt.opts {
				opt(cfg)
			}

			if cfg.Strategy != tt.expected.Strategy {
				t.Errorf("Strategy = %v, want %v", cfg.Strategy, tt.expected.Strategy)
			}
			if cfg.Mode != tt.expected.Mode {
				t.Errorf("Mode = %v, want %v", cfg.Mode, tt.expected.Mode)
			}
			if cfg.Message != tt.expected.Message {
				t.Errorf("Message = %v, want %v", cfg.Message, tt.expected.Message)
			}
			if cfg.Verbose != tt.expected.Verbose {
				t.Errorf("Verbose = %v, want %v", cfg.Verbose, tt.expected.Verbose)
			}
		})
	}
}

// TestStrategyString tests strategy string representation
func TestStrategyString(t *testing.T) {
	tests := []struct {
		strategy Strategy
		want     string
	}{
		{StrategyRecursive, "recursive"},
		{StrategyOctopus, "octopus"},
		{StrategyOurs, "ours"},
		{StrategySubtree, "subtree"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.want {
				t.Errorf("Strategy.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMergeModeString tests merge mode string representation
func TestMergeModeString(t *testing.T) {
	tests := []struct {
		mode MergeMode
		want string
	}{
		{ModeDefault, "default"},
		{ModeNoCommit, "no-commit"},
		{ModeSquash, "squash"},
		{ModeFastForwardOnly, "ff-only"},
		{ModeNoFastForward, "no-ff"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("MergeMode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConflictString tests conflict string representation
func TestConflictString(t *testing.T) {
	path, _ := scpath.NewRelativePath("test/file.txt")
	ourSHA, _ := objects.NewObjectHashFromString("0123456789abcdef0123456789abcdef01234567")
	theirSHA, _ := objects.NewObjectHashFromString("fedcba9876543210fedcba9876543210fedcba98")
	baseSHA, _ := objects.NewObjectHashFromString("1111111111111111111111111111111111111111")

	conflict := &Conflict{
		Path:     path,
		OurSHA:   ourSHA,
		TheirSHA: theirSHA,
		BaseSHA:  baseSHA,
	}

	str := conflict.String()
	if str == "" {
		t.Error("Conflict.String() returned empty string")
	}

	// Check that the string contains key information
	if !contains(str, "test/file.txt") {
		t.Error("Conflict.String() should contain file path")
	}
}

// TestMergeBaseCalculator tests merge base calculation
func TestMergeBaseCalculator(t *testing.T) {
	// This is a placeholder test - in a real implementation,
	// we'd set up a test repository with commits and test merge base finding
	t.Skip("Requires test repository setup")
}

// TestFastForwardMerger tests fast-forward merge strategy
func TestFastForwardMerger(t *testing.T) {
	// This is a placeholder test - in a real implementation,
	// we'd set up a test repository and test fast-forward merges
	t.Skip("Requires test repository setup")
}

// TestThreeWayMerger tests three-way merge strategy
func TestThreeWayMerger(t *testing.T) {
	// This is a placeholder test - in a real implementation,
	// we'd set up a test repository and test three-way merges
	t.Skip("Requires test repository setup")
}

// TestRecursiveMerger tests recursive merge strategy
func TestRecursiveMerger(t *testing.T) {
	// This is a placeholder test - in a real implementation,
	// we'd set up a test repository and test recursive merges
	t.Skip("Requires test repository setup")
}

// TestOctopusMerger tests octopus merge strategy
func TestOctopusMerger(t *testing.T) {
	// This is a placeholder test - in a real implementation,
	// we'd set up a test repository and test octopus merges
	t.Skip("Requires test repository setup")
}

// TestSquashMerger tests squash merge
func TestSquashMerger(t *testing.T) {
	// This is a placeholder test - in a real implementation,
	// we'd set up a test repository and test squash merges
	t.Skip("Requires test repository setup")
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestMergeContextCreation tests merge context creation
func TestMergeContextCreation(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	// Create mock commits
	ourCommit := &commit.Commit{
		Message: "Our commit",
	}

	theirCommit := &commit.Commit{
		Message: "Their commit",
	}

	baseCommit := &commit.Commit{
		Message: "Base commit",
	}

	mergeCtx := &MergeContext{
		Ctx:          ctx,
		OurCommit:    ourCommit,
		TheirCommits: []*commit.Commit{theirCommit},
		BaseCommit:   baseCommit,
		Config:       config,
	}

	if mergeCtx.OurCommit != ourCommit {
		t.Error("MergeContext.OurCommit not set correctly")
	}

	if len(mergeCtx.TheirCommits) != 1 || mergeCtx.TheirCommits[0] != theirCommit {
		t.Error("MergeContext.TheirCommits not set correctly")
	}

	if mergeCtx.BaseCommit != baseCommit {
		t.Error("MergeContext.BaseCommit not set correctly")
	}

	if mergeCtx.Config != config {
		t.Error("MergeContext.Config not set correctly")
	}
}

// TestMergeResult tests merge result structure
func TestMergeResult(t *testing.T) {
	sha, _ := objects.NewObjectHashFromString("0123456789abcdef0123456789abcdef01234567")

	result := &MergeResult{
		Success:      true,
		FastForward:  true,
		CommitSHA:    sha,
		Conflicts:    []string{},
		FilesChanged: 5,
		Insertions:   100,
		Deletions:    50,
		Message:      "Fast-forward merge",
	}

	if !result.Success {
		t.Error("MergeResult.Success should be true")
	}

	if !result.FastForward {
		t.Error("MergeResult.FastForward should be true")
	}

	if result.CommitSHA != sha {
		t.Error("MergeResult.CommitSHA not set correctly")
	}

	if len(result.Conflicts) != 0 {
		t.Error("MergeResult.Conflicts should be empty")
	}

	if result.FilesChanged != 5 {
		t.Error("MergeResult.FilesChanged should be 5")
	}

	if result.Message == "" {
		t.Error("MergeResult.Message should not be empty")
	}
}

// TestConflictResolver tests conflict resolution
func TestConflictResolver(t *testing.T) {
	tests := []struct {
		name     string
		strategy ConflictResolution
	}{
		{"fail on conflict", ConflictFail},
		{"use ours", ConflictOurs},
		{"use theirs", ConflictTheirs},
		{"manual resolution", ConflictManual},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewConflictResolver(tt.strategy)
			if resolver == nil {
				t.Fatal("NewConflictResolver returned nil")
			}

			if resolver.HasConflicts() {
				t.Error("New resolver should not have conflicts")
			}

			// Add a conflict
			path, _ := scpath.NewRelativePath("test.txt")
			conflict := Conflict{
				Path:         path,
				OurVersion:   []byte("our content"),
				TheirVersion: []byte("their content"),
			}

			resolver.AddConflict(conflict)

			if !resolver.HasConflicts() {
				t.Error("Resolver should have conflicts after adding one")
			}

			conflicts := resolver.GetConflicts()
			if len(conflicts) != 1 {
				t.Errorf("Expected 1 conflict, got %d", len(conflicts))
			}
		})
	}
}

// TestMergeContent tests simple content merging
func TestMergeContent(t *testing.T) {
	tests := []struct {
		name     string
		base     []byte
		ours     []byte
		theirs   []byte
		wantData []byte
		wantOk   bool
	}{
		{
			name:     "no changes",
			base:     []byte("content"),
			ours:     []byte("content"),
			theirs:   []byte("content"),
			wantData: []byte("content"),
			wantOk:   true,
		},
		{
			name:     "only we changed",
			base:     []byte("base"),
			ours:     []byte("modified"),
			theirs:   []byte("base"),
			wantData: []byte("modified"),
			wantOk:   true,
		},
		{
			name:     "only they changed",
			base:     []byte("base"),
			ours:     []byte("base"),
			theirs:   []byte("modified"),
			wantData: []byte("modified"),
			wantOk:   true,
		},
		{
			name:     "same changes",
			base:     []byte("base"),
			ours:     []byte("modified"),
			theirs:   []byte("modified"),
			wantData: []byte("modified"),
			wantOk:   true,
		},
		{
			name:     "different changes - conflict",
			base:     []byte("base"),
			ours:     []byte("our change"),
			theirs:   []byte("their change"),
			wantData: nil,
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotData, gotOk := MergeContent(tt.base, tt.ours, tt.theirs)

			if gotOk != tt.wantOk {
				t.Errorf("MergeContent() ok = %v, want %v", gotOk, tt.wantOk)
			}

			if tt.wantOk && string(gotData) != string(tt.wantData) {
				t.Errorf("MergeContent() data = %v, want %v", string(gotData), string(tt.wantData))
			}
		})
	}
}
