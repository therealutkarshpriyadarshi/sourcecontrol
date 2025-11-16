package commit

import (
	"strings"
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/common"
)

func TestNewCommitPerson(t *testing.T) {
	tests := []struct {
		name        string
		pname       string
		email       string
		when        time.Time
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid person",
			pname:   "John Doe",
			email:   "john@example.com",
			when:    time.Unix(1609459200, 0),
			wantErr: false,
		},
		{
			name:    "valid person with trimming",
			pname:   "  Jane Smith  ",
			email:   "  jane@example.com  ",
			when:    time.Unix(1609459200, 0),
			wantErr: false,
		},
		{
			name:        "empty name",
			pname:       "",
			email:       "test@example.com",
			when:        time.Now(),
			wantErr:     true,
			errContains: "name cannot be empty",
		},
		{
			name:        "whitespace name",
			pname:       "   ",
			email:       "test@example.com",
			when:        time.Now(),
			wantErr:     true,
			errContains: "name cannot be empty",
		},
		{
			name:        "empty email",
			pname:       "John Doe",
			email:       "",
			when:        time.Now(),
			wantErr:     true,
			errContains: "email cannot be empty",
		},
		{
			name:        "invalid email without @",
			pname:       "John Doe",
			email:       "invalidemail.com",
			when:        time.Now(),
			wantErr:     true,
			errContains: "invalid email format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			person, err := NewCommitPerson(tt.pname, tt.email, tt.when)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCommitPerson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewCommitPerson() error = %v, should contain %v", err, tt.errContains)
				}
				return
			}
			if person.Name != strings.TrimSpace(tt.pname) {
				t.Errorf("Name = %v, want %v", person.Name, strings.TrimSpace(tt.pname))
			}
			if person.Email != strings.TrimSpace(tt.email) {
				t.Errorf("Email = %v, want %v", person.Email, strings.TrimSpace(tt.email))
			}
		})
	}
}

func TestCommitPerson_FormatForGit(t *testing.T) {
	tests := []struct {
		name     string
		person   *CommitPerson
		expected string
	}{
		{
			name: "UTC timezone",
			person: &CommitPerson{
				Name:  "John Doe",
				Email: "john@example.com",
				When:  common.NewTimestampFromTime(time.Unix(1609459200, 0).UTC()),
			},
			expected: "John Doe <john@example.com> 1609459200 +0000",
		},
		{
			name: "positive timezone offset",
			person: &CommitPerson{
				Name:  "Jane Smith",
				Email: "jane@example.com",
				When:  common.NewTimestampFromTime(time.Unix(1609459200, 0).In(time.FixedZone("IST", 5*3600+30*60))),
			},
			expected: "Jane Smith <jane@example.com> 1609459200 +0530",
		},
		{
			name: "negative timezone offset",
			person: &CommitPerson{
				Name:  "Bob Johnson",
				Email: "bob@example.com",
				When:  common.NewTimestampFromTime(time.Unix(1609459200, 0).In(time.FixedZone("PST", -8*3600))),
			},
			expected: "Bob Johnson <bob@example.com> 1609459200 -0800",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.person.FormatForGit()
			if result != tt.expected {
				t.Errorf("FormatForGit() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseCommitPerson(t *testing.T) {
	tests := []struct {
		name        string
		gitFormat   string
		wantErr     bool
		errContains string
		checkFunc   func(*testing.T, *CommitPerson)
	}{
		{
			name:      "valid format UTC",
			gitFormat: "John Doe <john@example.com> 1609459200 +0000",
			wantErr:   false,
			checkFunc: func(t *testing.T, p *CommitPerson) {
				if p.Name != "John Doe" {
					t.Errorf("Name = %v, want John Doe", p.Name)
				}
				if p.Email != "john@example.com" {
					t.Errorf("Email = %v, want john@example.com", p.Email)
				}
				if p.When.Seconds != 1609459200 {
					t.Errorf("When.Seconds = %v, want 1609459200", p.When.Seconds)
				}
			},
		},
		{
			name:      "valid format with positive offset",
			gitFormat: "Jane Smith <jane@example.com> 1609459200 +0530",
			wantErr:   false,
			checkFunc: func(t *testing.T, p *CommitPerson) {
				if p.Name != "Jane Smith" {
					t.Errorf("Name = %v, want Jane Smith", p.Name)
				}
				_, offset := p.When.Time().Zone()
				expectedOffset := 5*3600 + 30*60
				if offset != expectedOffset {
					t.Errorf("Timezone offset = %v, want %v", offset, expectedOffset)
				}
			},
		},
		{
			name:      "valid format with negative offset",
			gitFormat: "Bob Johnson <bob@example.com> 1609459200 -0800",
			wantErr:   false,
			checkFunc: func(t *testing.T, p *CommitPerson) {
				_, offset := p.When.Time().Zone()
				expectedOffset := -8 * 3600
				if offset != expectedOffset {
					t.Errorf("Timezone offset = %v, want %v", offset, expectedOffset)
				}
			},
		},
		{
			name:        "invalid format - missing parts",
			gitFormat:   "John Doe <john@example.com>",
			wantErr:     true,
			errContains: "invalid person format",
		},
		{
			name:        "invalid format - missing email brackets",
			gitFormat:   "John Doe john@example.com 1609459200 +0000",
			wantErr:     true,
			errContains: "invalid person format",
		},
		{
			name:        "invalid timestamp",
			gitFormat:   "John Doe <john@example.com> invalid +0000",
			wantErr:     true,
			errContains: "invalid person format",
		},
		{
			name:        "invalid timezone format",
			gitFormat:   "John Doe <john@example.com> 1609459200 +00",
			wantErr:     true,
			errContains: "invalid person format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			person, err := ParseCommitPerson(tt.gitFormat)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommitPerson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseCommitPerson() error = %v, should contain %v", err, tt.errContains)
				}
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, person)
			}
		})
	}
}

func TestCommitPerson_RoundTrip(t *testing.T) {
	// Test that FormatForGit and ParseCommitPerson are inverse operations
	original := &CommitPerson{
		Name:  "Test User",
		Email: "test@example.com",
		When:  common.NewTimestampFromTime(time.Unix(1609459200, 0).In(time.FixedZone("IST", 5*3600+30*60))),
	}

	gitFormat := original.FormatForGit()
	parsed, err := ParseCommitPerson(gitFormat)
	if err != nil {
		t.Fatalf("ParseCommitPerson() error = %v", err)
	}

	if parsed.Name != original.Name {
		t.Errorf("Name = %v, want %v", parsed.Name, original.Name)
	}
	if parsed.Email != original.Email {
		t.Errorf("Email = %v, want %v", parsed.Email, original.Email)
	}
	if parsed.When.Seconds != original.When.Seconds {
		t.Errorf("When.Seconds = %v, want %v", parsed.When.Seconds, original.When.Seconds)
	}

	// Check timezone
	_, origOffset := original.When.Time().Zone()
	_, parsedOffset := parsed.When.Time().Zone()
	if origOffset != parsedOffset {
		t.Errorf("Timezone offset = %v, want %v", parsedOffset, origOffset)
	}
}

func TestCommitPerson_Equal(t *testing.T) {
	when := common.NewTimestampFromTime(time.Unix(1609459200, 0).UTC())
	person1 := &CommitPerson{
		Name:  "John Doe",
		Email: "john@example.com",
		When:  when,
	}

	tests := []struct {
		name   string
		other  *CommitPerson
		expect bool
	}{
		{
			name: "equal persons",
			other: &CommitPerson{
				Name:  "John Doe",
				Email: "john@example.com",
				When:  when,
			},
			expect: true,
		},
		{
			name: "different name",
			other: &CommitPerson{
				Name:  "Jane Doe",
				Email: "john@example.com",
				When:  when,
			},
			expect: false,
		},
		{
			name: "different email",
			other: &CommitPerson{
				Name:  "John Doe",
				Email: "jane@example.com",
				When:  when,
			},
			expect: false,
		},
		{
			name: "different time",
			other: &CommitPerson{
				Name:  "John Doe",
				Email: "john@example.com",
				When:  common.NewTimestampFromTime(time.Unix(1609459201, 0).UTC()),
			},
			expect: false,
		},
		{
			name:   "nil other",
			other:  nil,
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := person1.Equal(tt.other)
			if result != tt.expect {
				t.Errorf("Equal() = %v, want %v", result, tt.expect)
			}
		})
	}
}

func TestCommitPerson_String(t *testing.T) {
	person := &CommitPerson{
		Name:  "John Doe",
		Email: "john@example.com",
		When:  common.NewTimestampFromTime(time.Unix(1609459200, 0).UTC()),
	}

	str := person.String()
	if !strings.Contains(str, "John Doe") {
		t.Errorf("String() should contain name, got %v", str)
	}
	if !strings.Contains(str, "john@example.com") {
		t.Errorf("String() should contain email, got %v", str)
	}
}

func TestParseTimezone(t *testing.T) {
	tests := []struct {
		name           string
		tzString       string
		wantErr        bool
		expectedOffset int // in seconds
	}{
		{
			name:           "UTC",
			tzString:       "+0000",
			wantErr:        false,
			expectedOffset: 0,
		},
		{
			name:           "IST +0530",
			tzString:       "+0530",
			wantErr:        false,
			expectedOffset: 5*3600 + 30*60,
		},
		{
			name:           "PST -0800",
			tzString:       "-0800",
			wantErr:        false,
			expectedOffset: -8 * 3600,
		},
		{
			name:     "invalid length",
			tzString: "+00",
			wantErr:  true,
		},
		{
			name:     "invalid sign",
			tzString: "00000",
			wantErr:  true,
		},
		{
			name:     "invalid hours",
			tzString: "+ab00",
			wantErr:  true,
		},
		{
			name:     "invalid minutes",
			tzString: "+00xy",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := parseTimezone(tt.tzString)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimezone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				_, offset := time.Now().In(loc).Zone()
				if offset != tt.expectedOffset {
					t.Errorf("Offset = %v, want %v", offset, tt.expectedOffset)
				}
			}
		})
	}
}
