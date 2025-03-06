package wordprocessor

import (
	"unicode"
)

// Constants for ASCII processing
const (
	// ASCIITableSize is the size of the ASCII lookup table (0-127)
	ASCIITableSize = 128
)

// ASCIIWordChar is a lookup table for ASCII characters that can be part of a word
// This is much faster than calling unicode functions for ASCII characters
var ASCIIWordChar [ASCIITableSize]bool

// Initialize the ASCII lookup table
func init() {
	// Initialize the lookup table
	for i := 0; i < ASCIITableSize; i++ {
		r := rune(i)

		// Characters that can be part of a word
		ASCIIWordChar[i] = (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-' || r == '\''
	}
}

// IsASCIIWordChar returns true if the byte is part of a word (for ASCII characters)
func IsASCIIWordChar(b byte) bool {
	if b < ASCIITableSize {
		return ASCIIWordChar[b]
	}
	return false
}

// IsWordChar determines if a rune is part of a word
func IsWordChar(r rune) bool {
	if r < ASCIITableSize {
		return ASCIIWordChar[r]
	}

	// For non-ASCII, use Unicode package
	return unicode.IsLetter(r) || unicode.IsNumber(r) ||
		r == '_' || r == '-' || r == '\''
}

// IsASCIIOnly checks if a byte slice contains only ASCII characters
func IsASCIIOnly(data []byte) bool {
	for _, b := range data {
		if b >= ASCIITableSize {
			return false
		}
	}
	return true
}

// HandleUTF8 handles multi-byte UTF-8 sequences
// Returns:
// - The rune value
// - The number of bytes in the UTF-8 sequence
// - Whether the rune is a word character
func HandleUTF8(data []byte, pos int) (rune, int, bool) {
	// Fast path for ASCII
	if data[pos] < ASCIITableSize {
		return rune(data[pos]), 1, ASCIIWordChar[data[pos]]
	}

	// Handle UTF-8 sequences
	r, size := DecodeRune(data[pos:])
	return r, size, IsWordChar(r)
}

// DecodeRune decodes a UTF-8 sequence to a rune
func DecodeRune(b []byte) (rune, int) {
	if len(b) == 0 {
		return 0, 0
	}

	// ASCII
	if b[0] < 0x80 {
		return rune(b[0]), 1
	}

	// Invalid UTF-8 sequence starter
	if b[0] < 0xC0 || b[0] > 0xF7 {
		return unicode.ReplacementChar, 1
	}

	// 2-byte sequence
	if b[0] < 0xE0 {
		if len(b) < 2 || b[1]&0xC0 != 0x80 {
			return unicode.ReplacementChar, 1
		}
		return rune((uint32(b[0]&0x1F) << 6) | uint32(b[1]&0x3F)), 2
	}

	// 3-byte sequence
	if b[0] < 0xF0 {
		if len(b) < 3 || b[1]&0xC0 != 0x80 || b[2]&0xC0 != 0x80 {
			return unicode.ReplacementChar, 1
		}
		return rune((uint32(b[0]&0x0F) << 12) | (uint32(b[1]&0x3F) << 6) | uint32(b[2]&0x3F)), 3
	}

	// 4-byte sequence
	if len(b) < 4 || b[1]&0xC0 != 0x80 || b[2]&0xC0 != 0x80 || b[3]&0xC0 != 0x80 {
		return unicode.ReplacementChar, 1
	}

	return rune((uint32(b[0]&0x07) << 18) | (uint32(b[1]&0x3F) << 12) | (uint32(b[2]&0x3F) << 6) | uint32(b[3]&0x3F)), 4
}

// RuneLen returns the length of a UTF-8 encoded rune
func RuneLen(r rune) int {
	if r < 0x80 {
		return 1
	} else if r < 0x800 {
		return 2
	} else if r < 0x10000 {
		return 3
	}
	return 4
}
