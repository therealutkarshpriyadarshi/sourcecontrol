package common

import (
	"time"
)

// Timestamp represents a Git timestamp with nanosecond precision.
// Git stores timestamps as [seconds_since_epoch, nanoseconds] pairs.
//
// This is used in:
// - Index entries (creation and modification times)
// - Commit objects (author and committer timestamps)
//
// Binary representation:
//   - Seconds: 4 bytes (uint32)
//   - Nanoseconds: 4 bytes (uint32)
//
// Note: This type stores Unix timestamp (UTC) but preserves timezone
// information for proper conversion back to time.Time.
type Timestamp struct {
	Seconds     uint32
	Nanoseconds uint32
	location    *time.Location // timezone for proper time conversion
}

func NewTimestamp(millis uint32, nanos uint32) Timestamp {
	return Timestamp{
		Seconds:     millis,
		Nanoseconds: nanos,
	}
}

// NewTimestamp creates a Timestamp from a time.Time.
func NewTimestampFromTime(t time.Time) Timestamp {
	unix := t.Unix()
	nanos := t.Nanosecond()
	return Timestamp{
		Seconds:     uint32(unix),
		Nanoseconds: uint32(nanos),
		location:    t.Location(),
	}
}

// NewTimestampFromMillis creates a Timestamp from milliseconds.
func NewTimestampFromMillis(millis int64) Timestamp {
	seconds := millis / 1000
	nanos := (millis % 1000) * 1_000_000
	return Timestamp{
		Seconds:     uint32(seconds),
		Nanoseconds: uint32(nanos),
	}
}

// Time converts the Timestamp to a time.Time.
// If a location was preserved, it uses that; otherwise defaults to UTC.
func (t Timestamp) Time() time.Time {
	if t.location != nil {
		return time.Unix(int64(t.Seconds), int64(t.Nanoseconds)).In(t.location)
	}
	return time.Unix(int64(t.Seconds), int64(t.Nanoseconds)).UTC()
}

// IsZero returns true if the timestamp is zero.
func (t Timestamp) IsZero() bool {
	return t.Seconds == 0 && t.Nanoseconds == 0
}

// String returns a human-readable representation.
func (t Timestamp) String() string {
	if t.IsZero() {
		return "0"
	}
	return t.Time().Format(time.RFC3339)
}

// Equal compares two timestamps for equality.
func (t Timestamp) Equal(other Timestamp) bool {
	return t.Seconds == other.Seconds && t.Nanoseconds == other.Nanoseconds
}

// Before returns true if this timestamp is before the other.
func (t Timestamp) Before(other Timestamp) bool {
	if t.Seconds != other.Seconds {
		return t.Seconds < other.Seconds
	}
	return t.Nanoseconds < other.Nanoseconds
}

// After returns true if this timestamp is after the other.
func (t Timestamp) After(other Timestamp) bool {
	if t.Seconds != other.Seconds {
		return t.Seconds > other.Seconds
	}
	return t.Nanoseconds > other.Nanoseconds
}
