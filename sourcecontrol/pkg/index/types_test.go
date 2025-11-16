package index

import (
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/common"
)

func TestFileMode(t *testing.T) {
	tests := []struct {
		name      string
		mode      FileMode
		wantType  FileMode
		wantPerms FileMode
		isRegular bool
		isSymlink bool
		isGitlink bool
		isDir     bool
		isExec    bool
	}{
		{
			name:      "regular file",
			mode:      FileModeRegular,
			wantType:  FileModeTypeRegular,
			wantPerms: 0o644,
			isRegular: true,
			isSymlink: false,
			isGitlink: false,
			isDir:     false,
			isExec:    false,
		},
		{
			name:      "executable file",
			mode:      FileModeExecutable,
			wantType:  FileModeTypeRegular,
			wantPerms: 0o755,
			isRegular: true,
			isSymlink: false,
			isGitlink: false,
			isDir:     false,
			isExec:    true,
		},
		{
			name:      "symbolic link",
			mode:      FileModeSymlink,
			wantType:  FileModeTypeSymlink,
			wantPerms: 0,
			isRegular: false,
			isSymlink: true,
			isGitlink: false,
			isDir:     false,
			isExec:    false,
		},
		{
			name:      "gitlink",
			mode:      FileModeGitlink,
			wantType:  FileModeTypeGitlink,
			wantPerms: 0,
			isRegular: false,
			isSymlink: false,
			isGitlink: true,
			isDir:     false,
			isExec:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.Type(); got != tt.wantType {
				t.Errorf("Type() = %v, want %v", got, tt.wantType)
			}
			if got := tt.mode.Permissions(); got != tt.wantPerms {
				t.Errorf("Permissions() = %o, want %o", got, tt.wantPerms)
			}
			if got := tt.mode.IsRegular(); got != tt.isRegular {
				t.Errorf("IsRegular() = %v, want %v", got, tt.isRegular)
			}
			if got := tt.mode.IsSymlink(); got != tt.isSymlink {
				t.Errorf("IsSymlink() = %v, want %v", got, tt.isSymlink)
			}
			if got := tt.mode.IsGitlink(); got != tt.isGitlink {
				t.Errorf("IsGitlink() = %v, want %v", got, tt.isGitlink)
			}
			if got := tt.mode.IsDirectory(); got != tt.isDir {
				t.Errorf("IsDirectory() = %v, want %v", got, tt.isDir)
			}
			if got := tt.mode.IsExecutable(); got != tt.isExec {
				t.Errorf("IsExecutable() = %v, want %v", got, tt.isExec)
			}
		})
	}
}

func TestTimestamp(t *testing.T) {
	t.Run("NewTimestamp", func(t *testing.T) {
		now := time.Now()
		ts := common.NewTimestampFromTime(now)

		// Check that conversion round-trips correctly
		gotTime := ts.Time()
		if gotTime.Unix() != now.Unix() {
			t.Errorf("Time() unix = %v, want %v", gotTime.Unix(), now.Unix())
		}
	})

	t.Run("NewTimestampFromMillis", func(t *testing.T) {
		millis := int64(1234567890123)
		ts := NewTimestampFromMillis(millis)

		wantSeconds := uint32(1234567890)
		wantNanos := uint32(123000000)

		if ts.Seconds != wantSeconds {
			t.Errorf("Seconds = %v, want %v", ts.Seconds, wantSeconds)
		}
		if ts.Nanoseconds != wantNanos {
			t.Errorf("Nanoseconds = %v, want %v", ts.Nanoseconds, wantNanos)
		}
	})

	t.Run("IsZero", func(t *testing.T) {
		zero := common.Timestamp{}
		if !zero.IsZero() {
			t.Error("zero timestamp should be zero")
		}

		nonZero := common.Timestamp{Seconds: 1}
		if nonZero.IsZero() {
			t.Error("non-zero timestamp should not be zero")
		}
	})
}

func TestEntryFlags(t *testing.T) {
	t.Run("NewEntryFlags", func(t *testing.T) {
		tests := []struct {
			name        string
			assumeValid bool
			stage       uint8
			filenameLen int
			wantValid   bool
			wantStage   uint8
			wantLen     int
		}{
			{
				name:        "normal entry",
				assumeValid: false,
				stage:       0,
				filenameLen: 256,
				wantValid:   false,
				wantStage:   0,
				wantLen:     256,
			},
			{
				name:        "assume valid",
				assumeValid: true,
				stage:       0,
				filenameLen: 100,
				wantValid:   true,
				wantStage:   0,
				wantLen:     100,
			},
			{
				name:        "merge conflict stage 1",
				assumeValid: false,
				stage:       1,
				filenameLen: 50,
				wantValid:   false,
				wantStage:   1,
				wantLen:     50,
			},
			{
				name:        "max filename length",
				assumeValid: false,
				stage:       0,
				filenameLen: 5000, // Over max
				wantValid:   false,
				wantStage:   0,
				wantLen:     MaxFilenameLength, // Should be capped
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				flags := NewEntryFlags(tt.assumeValid, tt.stage, tt.filenameLen)

				if got := flags.AssumeValid(); got != tt.wantValid {
					t.Errorf("AssumeValid() = %v, want %v", got, tt.wantValid)
				}
				if got := flags.Stage(); got != tt.wantStage {
					t.Errorf("Stage() = %v, want %v", got, tt.wantStage)
				}
				if got := flags.FilenameLength(); got != tt.wantLen {
					t.Errorf("FilenameLength() = %v, want %v", got, tt.wantLen)
				}
				if flags.Extended() {
					t.Error("Extended flag should not be set")
				}
			})
		}
	})

	t.Run("round trip", func(t *testing.T) {
		original := NewEntryFlags(true, 2, 512)

		assumeValid := original.AssumeValid()
		stage := original.Stage()
		filenameLen := original.FilenameLength()

		reconstructed := NewEntryFlags(assumeValid, stage, filenameLen)

		if original != reconstructed {
			t.Errorf("Round trip failed: original = %v, reconstructed = %v", original, reconstructed)
		}
	})
}
