package commit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/common"
)

// CommitPerson represents author or committer information in a Git commit.
//
// Commit Person Structure:
// ┌─────────────────────────────────────────────────────────────────┐
// │ Name <email> timestamp timezone                                 │
// └─────────────────────────────────────────────────────────────────┘
//
// Example: "John Doe <john@example.com> 1609459200 +0000"
type CommitPerson struct {
	Name  string
	Email string
	When  common.Timestamp
}

// personPattern is the regex pattern for parsing Git person format
// Pattern: "Name <email> timestamp timezone"
var personPattern = regexp.MustCompile(`^(.+) <([^>]+)> (\d+) ([+-]\d{4})$`)

// NewCommitPerson creates a new CommitPerson with validation
func NewCommitPerson(name, email string, when time.Time) (*CommitPerson, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}

	if err := validateEmail(email); err != nil {
		return nil, err
	}

	return &CommitPerson{
		Name:  strings.TrimSpace(name),
		Email: strings.TrimSpace(email),
		When:  common.NewTimestampFromTime(when),
	}, nil
}

// NewCommitPersonWithTimestamp creates a new CommitPerson with a Timestamp
func NewCommitPersonWithTimestamp(name, email string, when common.Timestamp) (*CommitPerson, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}

	if err := validateEmail(email); err != nil {
		return nil, err
	}

	return &CommitPerson{
		Name:  strings.TrimSpace(name),
		Email: strings.TrimSpace(email),
		When:  when,
	}, nil
}

// FormatForGit formats person information in Git's standard format:
// "Name <email> timestamp timezone"
func (p *CommitPerson) FormatForGit() string {
	when := p.When.Time()
	timestamp := when.Unix()
	_, offset := when.Zone()

	// Format timezone as +HHMM or -HHMM
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60

	return fmt.Sprintf("%s <%s> %d %s%02d%02d",
		p.Name, p.Email, timestamp, sign, hours, minutes)
}

// ParseCommitPerson parses person information from Git's format.
// Format: "Name <email> timestamp timezone"
// Example: "John Doe <john@example.com> 1609459200 +0000"
func ParseCommitPerson(gitFormat string) (*CommitPerson, error) {
	matches := personPattern.FindStringSubmatch(gitFormat)
	if matches == nil {
		return nil, fmt.Errorf("invalid person format: %s", gitFormat)
	}

	name := matches[1]
	email := matches[2]
	timestampStr := matches[3]
	timezoneStr := matches[4]

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	location, err := parseTimezone(timezoneStr)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	when := time.Unix(timestamp, 0).In(location)

	return NewCommitPersonWithTimestamp(name, email, common.NewTimestampFromTime(when))
}

// String returns a human-readable representation
func (p *CommitPerson) String() string {
	return fmt.Sprintf("%s <%s> at %s", p.Name, p.Email, p.When.String())
}

// Equal compares two CommitPerson instances for equality
func (p *CommitPerson) Equal(other *CommitPerson) bool {
	if other == nil {
		return false
	}
	return p.Name == other.Name &&
		p.Email == other.Email &&
		p.When.Equal(other.When)
}

// validateName validates the person name
func validateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
}

// validateEmail validates the email address
func validateEmail(email string) error {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if !strings.Contains(trimmed, "@") {
		return fmt.Errorf("invalid email format: %s", email)
	}
	return nil
}

// parseTimezone parses timezone string like "+0530" or "-0800" and returns a Location
func parseTimezone(tzString string) (*time.Location, error) {
	if len(tzString) != 5 {
		return nil, fmt.Errorf("invalid timezone length: %s", tzString)
	}

	sign := tzString[0]
	if sign != '+' && sign != '-' {
		return nil, fmt.Errorf("invalid timezone sign: %c", sign)
	}

	hours, err := strconv.Atoi(tzString[1:3])
	if err != nil {
		return nil, fmt.Errorf("invalid timezone hours: %w", err)
	}

	minutes, err := strconv.Atoi(tzString[3:5])
	if err != nil {
		return nil, fmt.Errorf("invalid timezone minutes: %w", err)
	}

	offsetSeconds := hours*3600 + minutes*60
	if sign == '-' {
		offsetSeconds = -offsetSeconds
	}

	return time.FixedZone(tzString, offsetSeconds), nil
}
