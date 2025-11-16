package objects

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
)

// ObjectHash represents a SHA-1 hash of a Git object (40-character hex string)
// Example: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"
type ObjectHash string

// ShortHash represents an abbreviated hash (typically 7 characters)
// Example: "e69de29"
type ShortHash string

// RawHash represents a SHA-1 hash as a 20-byte array
type RawHash [20]byte

const (
	// HashLength is the length of a full SHA-1 hash in hex (40 characters)
	HashLength = 40
	// ShortHashLength is the default length for abbreviated hashes (7 characters)
	ShortHashLength = 7
	// RawHashLength is the length of a SHA-1 hash in bytes (20 bytes)
	RawHashLength = 20
)

// ZeroHash returns an all-zero hash (used for uninitialized or null references)
func ZeroHash() ObjectHash {
	return ObjectHash("0000000000000000000000000000000000000000")
}

// NewObjectHash creates a new ObjectHash from a byte slice
func NewObjectHash(data []byte) ObjectHash {
	hash := sha1.Sum(data)
	return ObjectHash(hex.EncodeToString(hash[:]))
}

// NewObjectHashFromRaw creates an ObjectHash from a 20-byte array
func NewObjectHashFromRaw(raw RawHash) ObjectHash {
	return ObjectHash(hex.EncodeToString(raw[:]))
}

// NewObjectHashFromString creates an ObjectHash from a hex string
// Returns an error if the string is not a valid hash
func NewObjectHashFromString(s string) (ObjectHash, error) {
	hash := ObjectHash(strings.ToLower(s))
	if err := hash.Validate(); err != nil {
		return "", err
	}
	return hash, nil
}

// ParseObjectHash is an alias for NewObjectHashFromString
func ParseObjectHash(s string) (ObjectHash, error) {
	return NewObjectHashFromString(s)
}

// String returns the hash as a string
func (h ObjectHash) String() string {
	return string(h)
}

// IsValid returns true if this is a valid SHA-1 hash
func (h ObjectHash) IsValid() bool {
	return h.Validate() == nil
}

// Validate checks if the hash is valid
func (h ObjectHash) Validate() error {
	if len(h) != HashLength {
		return fmt.Errorf("hash must be %d characters long, got %d", HashLength, len(h))
	}

	for _, c := range h {
		if !isHexChar(c) {
			return fmt.Errorf("hash must contain only hex characters, found '%c'", c)
		}
	}

	return nil
}

// IsZero returns true if this is the zero hash
func (h ObjectHash) IsZero() bool {
	return h == ZeroHash()
}

// Short returns the abbreviated version of the hash
func (h ObjectHash) Short() ShortHash {
	if len(h) >= ShortHashLength {
		return ShortHash(h[:ShortHashLength])
	}
	return ShortHash(h)
}

// ShortN returns the first n characters of the hash
func (h ObjectHash) ShortN(n int) ShortHash {
	if n <= 0 {
		n = ShortHashLength
	}
	if n > len(h) {
		n = len(h)
	}
	return ShortHash(h[:n])
}

// Bytes returns the hash as a byte slice (decoded from hex)
func (h ObjectHash) Bytes() ([]byte, error) {
	if err := h.Validate(); err != nil {
		return nil, err
	}
	return hex.DecodeString(string(h))
}

// Raw returns the hash as a 20-byte array
func (h ObjectHash) Raw() (RawHash, error) {
	bytes, err := h.Bytes()
	if err != nil {
		return RawHash{}, err
	}

	var raw RawHash
	copy(raw[:], bytes)
	return raw, nil
}

// Equal compares two hashes for equality (case-insensitive)
func (h ObjectHash) Equal(other ObjectHash) bool {
	return strings.EqualFold(string(h), string(other))
}

// HasPrefix returns true if the hash starts with the given prefix
func (h ObjectHash) HasPrefix(prefix string) bool {
	return strings.HasPrefix(string(h), strings.ToLower(prefix))
}

// String returns the short hash as a string
func (sh ShortHash) String() string {
	return string(sh)
}

// IsValid returns true if this is a valid short hash (hex characters only)
func (sh ShortHash) IsValid() bool {
	if len(sh) == 0 || len(sh) > HashLength {
		return false
	}
	for _, c := range sh {
		if !isHexChar(c) {
			return false
		}
	}
	return true
}

// Matches returns true if the full hash starts with this short hash
func (sh ShortHash) Matches(hash ObjectHash) bool {
	return hash.HasPrefix(string(sh))
}

// Length returns the length of the short hash
func (sh ShortHash) Length() int {
	return len(sh)
}

// RawHash methods

// Hash converts RawHash to ObjectHash
func (rh RawHash) Hash() ObjectHash {
	return NewObjectHashFromRaw(rh)
}

// String returns the hash as a hex string
func (rh RawHash) String() string {
	return hex.EncodeToString(rh[:])
}

// Short returns the abbreviated version
func (rh RawHash) Short() ShortHash {
	return rh.Hash().Short()
}

// IsZero returns true if this is a zero hash
func (rh RawHash) IsZero() bool {
	for _, b := range rh {
		if b != 0 {
			return false
		}
	}
	return true
}

// Equal compares two raw hashes for equality
func (rh RawHash) Equal(other RawHash) bool {
	return rh == other
}

// Helper functions

// isHexChar returns true if the character is a valid hex character
func isHexChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// ComputeHash computes the SHA-1 hash of the given data
func ComputeHash(data []byte) RawHash {
	return sha1.Sum(data)
}

// ComputeObjectHash computes the object hash from type and content
// This follows Git's format: hash("<type> <size>\0<content>")
func ComputeObjectHash(objType ObjectType, content ObjectContent) ObjectHash {
	serialized := NewSerializedObject(objType, content)
	return NewObjectHash(serialized.Bytes())
}
