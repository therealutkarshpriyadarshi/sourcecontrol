package merge

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateConflictMarkers(t *testing.T) {
	tests := []struct {
		name        string
		base        []byte
		ours        []byte
		theirs      []byte
		oursLabel   string
		theirsLabel string
		diff3Style  bool
		wantContain []string
	}{
		{
			name:        "simple conflict without diff3",
			base:        []byte("original content\n"),
			ours:        []byte("our changes\n"),
			theirs:      []byte("their changes\n"),
			oursLabel:   "HEAD",
			theirsLabel: "feature",
			diff3Style:  false,
			wantContain: []string{
				"<<<<<<< HEAD",
				"our changes",
				"=======",
				"their changes",
				">>>>>>> feature",
			},
		},
		{
			name:        "conflict with diff3 style",
			base:        []byte("original content\n"),
			ours:        []byte("our changes\n"),
			theirs:      []byte("their changes\n"),
			oursLabel:   "HEAD",
			theirsLabel: "feature",
			diff3Style:  true,
			wantContain: []string{
				"<<<<<<< HEAD",
				"our changes",
				"||||||| base",
				"original content",
				"=======",
				"their changes",
				">>>>>>> feature",
			},
		},
		{
			name:        "conflict without newline at end",
			base:        []byte("base"),
			ours:        []byte("ours"),
			theirs:      []byte("theirs"),
			oursLabel:   "HEAD",
			theirsLabel: "branch",
			diff3Style:  false,
			wantContain: []string{
				"<<<<<<< HEAD",
				"ours",
				"=======",
				"theirs",
				">>>>>>> branch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateConflictMarkers(tt.base, tt.ours, tt.theirs, tt.oursLabel, tt.theirsLabel, tt.diff3Style)
			resultStr := string(result)

			for _, want := range tt.wantContain {
				assert.Contains(t, resultStr, want, "result should contain: %s", want)
			}
		})
	}
}

func TestParseConflictMarkers(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int // number of conflicts
		wantErr bool
	}{
		{
			name: "single conflict",
			content: `line 1
<<<<<<< HEAD
our version
=======
their version
>>>>>>> branch
line 2`,
			want:    1,
			wantErr: false,
		},
		{
			name: "multiple conflicts",
			content: `line 1
<<<<<<< HEAD
conflict 1 ours
=======
conflict 1 theirs
>>>>>>> branch
line 2
<<<<<<< HEAD
conflict 2 ours
=======
conflict 2 theirs
>>>>>>> branch
line 3`,
			want:    2,
			wantErr: false,
		},
		{
			name: "conflict with diff3 style",
			content: `line 1
<<<<<<< HEAD
our version
||||||| base
base version
=======
their version
>>>>>>> branch
line 2`,
			want:    1,
			wantErr: false,
		},
		{
			name: "unclosed conflict",
			content: `line 1
<<<<<<< HEAD
our version
=======
their version
line 2`,
			want:    0,
			wantErr: true,
		},
		{
			name: "nested conflict markers",
			content: `line 1
<<<<<<< HEAD
<<<<<<< HEAD
nested
=======
their version
>>>>>>> branch
line 2`,
			want:    0,
			wantErr: true,
		},
		{
			name:    "no conflicts",
			content: "just regular content\nno conflicts here\n",
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts, err := ParseConflictMarkers([]byte(tt.content))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, len(conflicts), "number of conflicts should match")

			// Verify conflict structure
			for _, conflict := range conflicts {
				assert.NotEmpty(t, conflict.OursContent, "ours content should not be empty")
				assert.NotEmpty(t, conflict.TheirsContent, "theirs content should not be empty")
				assert.Greater(t, conflict.EndLine, conflict.StartLine, "end line should be after start line")
			}
		})
	}
}

func TestHasConflictMarkers(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name: "has conflict markers",
			content: `line 1
<<<<<<< HEAD
our version
=======
their version
>>>>>>> branch`,
			want: true,
		},
		{
			name: "has separator only",
			content: `line 1
=======
line 2`,
			want: true,
		},
		{
			name: "no conflict markers",
			content: `line 1
line 2
line 3`,
			want: false,
		},
		{
			name: "partial marker in middle of line",
			content: `this line has <<<<<<< in the middle
not at the start`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasConflictMarkers([]byte(tt.content))
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRemoveConflictMarkers(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		keepVersion string
		want        string
		wantErr     bool
	}{
		{
			name: "keep ours",
			content: `line 1
<<<<<<< HEAD
our version
=======
their version
>>>>>>> branch
line 2`,
			keepVersion: "ours",
			want: `line 1
our version
line 2
`,
			wantErr: false,
		},
		{
			name: "keep theirs",
			content: `line 1
<<<<<<< HEAD
our version
=======
their version
>>>>>>> branch
line 2`,
			keepVersion: "theirs",
			want: `line 1
their version
line 2
`,
			wantErr: false,
		},
		{
			name: "keep base with diff3",
			content: `line 1
<<<<<<< HEAD
our version
||||||| base
base version
=======
their version
>>>>>>> branch
line 2`,
			keepVersion: "base",
			want: `line 1
base version
line 2
`,
			wantErr: false,
		},
		{
			name: "invalid keep version",
			content: `line 1
<<<<<<< HEAD
our version
=======
their version
>>>>>>> branch
line 2`,
			keepVersion: "invalid",
			want:        "",
			wantErr:     true,
		},
		{
			name: "multiple conflicts keep ours",
			content: `line 1
<<<<<<< HEAD
conflict 1 ours
=======
conflict 1 theirs
>>>>>>> branch
line 2
<<<<<<< HEAD
conflict 2 ours
=======
conflict 2 theirs
>>>>>>> branch
line 3`,
			keepVersion: "ours",
			want: `line 1
conflict 1 ours
line 2
conflict 2 ours
line 3
`,
			wantErr: false,
		},
		{
			name:        "no conflicts",
			content:     "just regular content\nno conflicts here\n",
			keepVersion: "ours",
			want:        "just regular content\nno conflicts here\n",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RemoveConflictMarkers([]byte(tt.content), tt.keepVersion)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, string(result))
		})
	}
}

func TestConflictMarkerLabels(t *testing.T) {
	content := `<<<<<<< my-label
ours
=======
theirs
>>>>>>> their-label`

	conflicts, err := ParseConflictMarkers([]byte(content))
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	assert.Equal(t, "my-label", conflicts[0].OursLabel)
	assert.Equal(t, "their-label", conflicts[0].TheirsLabel)
}

func TestConflictMarkerContent(t *testing.T) {
	content := `before
<<<<<<< HEAD
line 1 ours
line 2 ours
=======
line 1 theirs
line 2 theirs
line 3 theirs
>>>>>>> branch
after`

	conflicts, err := ParseConflictMarkers([]byte(content))
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	conflict := conflicts[0]
	assert.Equal(t, 2, len(conflict.OursContent))
	assert.Equal(t, 3, len(conflict.TheirsContent))
	assert.Equal(t, "line 1 ours", conflict.OursContent[0])
	assert.Equal(t, "line 2 ours", conflict.OursContent[1])
	assert.Equal(t, "line 1 theirs", conflict.TheirsContent[0])
	assert.Equal(t, "line 2 theirs", conflict.TheirsContent[1])
	assert.Equal(t, "line 3 theirs", conflict.TheirsContent[2])
}

func TestConflictMarkerLineNumbers(t *testing.T) {
	lines := []string{
		"line 0",
		"line 1",
		"<<<<<<< HEAD",
		"ours",
		"=======",
		"theirs",
		">>>>>>> branch",
		"line 7",
	}
	content := strings.Join(lines, "\n")

	conflicts, err := ParseConflictMarkers([]byte(content))
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	conflict := conflicts[0]
	assert.Equal(t, 2, conflict.StartLine, "start line should be 2 (0-indexed)")
	assert.Equal(t, 6, conflict.EndLine, "end line should be 6 (0-indexed)")
}
