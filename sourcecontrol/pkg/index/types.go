package index

import (
	"github.com/utkarsh5026/SourceControl/pkg/common"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

// NewTimestampFromMillis creates a Timestamp from milliseconds.
var NewTimestampFromMillis = common.NewTimestampFromMillis

// FileMode is an alias for objects.FileMode for backward compatibility.
// Use objects.FileMode directly in new code.
type FileMode = objects.FileMode

// Re-export FileMode constants for backward compatibility.
const (
	FileModeTypeMask = objects.FileModeTypeMask
	FileModePermMask = objects.FileModePermMask
	FileModeExecMask = objects.FileModeExecMask

	FileModeTypeRegular = objects.FileModeTypeRegular
	FileModeTypeSymlink = objects.FileModeTypeSymlink
	FileModeTypeGitlink = objects.FileModeTypeGitlink
	FileModeTypeDir     = objects.FileModeTypeDir

	FileModeRegular    = objects.FileModeRegular
	FileModeExecutable = objects.FileModeExecutable
	FileModeSymlink    = objects.FileModeSymlink
	FileModeGitlink    = objects.FileModeGitlink
)

// EntryFlags represents the flags field in an index entry.
// The flags field contains several pieces of information packed into 16 bits:
// - Bit 15: assume-valid flag
// - Bit 14: extended flag (must be 0 for version 2)
// - Bits 13-12: stage number (0-3, for merge conflicts)
// - Bits 11-0: filename length (max 4095)
type EntryFlags uint16

const (
	// Flag bit positions and masks
	FlagAssumeValidBit                = 15
	FlagAssumeValidMask    EntryFlags = 0x8000
	FlagExtendedBit                   = 14
	FlagExtendedMask       EntryFlags = 0x4000
	FlagStageShift                    = 12
	FlagStageMask          EntryFlags = 0x3000
	FlagFilenameLengthMask EntryFlags = 0x0FFF
	MaxFilenameLength                 = 0x0FFF
)

// NewEntryFlags creates EntryFlags from components.
func NewEntryFlags(assumeValid bool, stage uint8, filenameLen int) EntryFlags {
	var flags EntryFlags

	if assumeValid {
		flags |= FlagAssumeValidMask
	}

	flags |= EntryFlags(stage&0x3) << FlagStageShift

	cappedLen := filenameLen
	if cappedLen > MaxFilenameLength {
		cappedLen = MaxFilenameLength
	}
	flags |= EntryFlags(cappedLen)

	return flags
}

// AssumeValid returns the assume-valid flag.
func (f EntryFlags) AssumeValid() bool {
	return (f & FlagAssumeValidMask) != 0
}

// Extended returns the extended flag.
func (f EntryFlags) Extended() bool {
	return (f & FlagExtendedMask) != 0
}

// Stage returns the stage number (0-3).
func (f EntryFlags) Stage() uint8 {
	return uint8((f & FlagStageMask) >> FlagStageShift)
}

// FilenameLength returns the filename length from the flags.
func (f EntryFlags) FilenameLength() int {
	return int(f & FlagFilenameLengthMask)
}

// Binary layout constants for index entries
const (
	FixedHeaderSize   = 62 // Everything before filename
	SHALength         = 20 // SHA-1 is always 20 bytes
	FlagsLength       = 2  // Flags are 2 bytes
	FieldSize         = 4  // Most fields are 4 bytes
	AlignmentBoundary = 8  // Entries are padded to 8-byte boundaries
)

// Index file format constants
const (
	IndexSignature    = "DIRC"
	IndexVersion      = 2
	IndexHeaderSize   = 12 // Signature (4) + Version (4) + Entry count (4)
	IndexChecksumSize = 20 // SHA-1 checksum
)
