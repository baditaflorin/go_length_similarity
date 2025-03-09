// File: internal/adapters/normalizer/allocation_efficient.go
package normalizer

import (
	"sync"
	"unicode"

	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

// ByteNormalizer extends the Normalizer interface with byte-level operations
type ByteNormalizer interface {
	ports.Normalizer
	NormalizeBytes([]byte, []byte) []byte
}

// AllocationEfficientNormalizer implements an optimized normalizer with minimal allocations
type AllocationEfficientNormalizer struct {
	// Pre-computed decision table for ASCII characters (0-127)
	asciiTable [128]struct {
		replace bool
		char    byte
	}

	// Buffer pool for reusable output buffers
	bufferPool sync.Pool
}

// NewAllocationEfficientNormalizer creates a new allocation-efficient normalizer
func NewAllocationEfficientNormalizer() ByteNormalizer {
	n := &AllocationEfficientNormalizer{
		bufferPool: sync.Pool{
			New: func() interface{} {
				buffer := make([]byte, 0, 1024)
				return &buffer
			},
		},
	}

	// Initialize the decision table for ASCII characters
	for i := 0; i < 128; i++ {
		r := rune(i)
		if unicode.IsPunct(r) {
			n.asciiTable[i] = struct {
				replace bool
				char    byte
			}{
				replace: true,
				char:    ' ',
			}
		} else if unicode.IsUpper(r) {
			n.asciiTable[i] = struct {
				replace bool
				char    byte
			}{
				replace: true,
				char:    byte(unicode.ToLower(r)),
			}
		} else {
			n.asciiTable[i] = struct {
				replace bool
				char    byte
			}{
				replace: false,
			}
		}
	}

	return n
}

// Normalize implements the standard Normalizer interface
func (n *AllocationEfficientNormalizer) Normalize(text string) string {
	if len(text) == 0 {
		return ""
	}

	// Get a buffer from the pool
	buffer := n.bufferPool.Get().(*[]byte)

	// Ensure the buffer has adequate capacity
	if cap(*buffer) < len(text)*2 {
		*buffer = make([]byte, 0, len(text)*2)
	}
	*buffer = (*buffer)[:0] // Reset length while keeping capacity

	// Fast path for ASCII-only strings
	asciiOnly := true
	for i := 0; i < len(text); i++ {
		if text[i] >= 128 {
			asciiOnly = false
			break
		}
	}

	var result []byte
	if asciiOnly {
		// Fast path for ASCII
		result = n.normalizeASCII([]byte(text), *buffer)
	} else {
		// Fallback for non-ASCII text
		result = n.normalizeUnicode([]byte(text), *buffer)
	}

	// Create a string from the result
	s := string(result)

	// Reset and return the buffer to the pool
	*buffer = (*buffer)[:0]
	n.bufferPool.Put(buffer)

	return s
}

// NormalizeBytes normalizes a byte slice directly without string conversion
func (n *AllocationEfficientNormalizer) NormalizeBytes(src []byte, dest []byte) []byte {
	if len(src) == 0 {
		return dest[:0]
	}

	// Check for ASCII-only input
	asciiOnly := true
	for i := 0; i < len(src); i++ {
		if src[i] >= 128 {
			asciiOnly = false
			break
		}
	}

	if asciiOnly {
		// Fast path for ASCII text
		return n.normalizeASCII(src, dest)
	}

	// Fallback for non-ASCII text
	return n.normalizeUnicode(src, dest)
}

// normalizeASCII performs fast normalization of ASCII-only text
func (n *AllocationEfficientNormalizer) normalizeASCII(src []byte, dest []byte) []byte {
	dest = dest[:0]

	// Reserve capacity if needed
	if cap(dest) < len(src) {
		newDest := make([]byte, 0, len(src))
		dest = newDest
	}

	var lastWasSpace bool

	for i := 0; i < len(src); i++ {
		b := src[i]

		if b < 128 {
			entry := n.asciiTable[b]
			if entry.replace {
				// Special handling for spaces to avoid duplicates
				if entry.char == ' ' {
					if !lastWasSpace {
						dest = append(dest, ' ')
						lastWasSpace = true
					}
				} else {
					dest = append(dest, entry.char)
					lastWasSpace = false
				}
			} else {
				dest = append(dest, b)
				lastWasSpace = false
			}
		} else {
			// This shouldn't happen for ASCII-only text,
			// but handle it just in case
			dest = append(dest, b)
			lastWasSpace = false
		}
	}

	return dest
}

// normalizeUnicode normalizes text that may contain unicode characters
func (n *AllocationEfficientNormalizer) normalizeUnicode(src []byte, dest []byte) []byte {
	dest = dest[:0]

	// Reserve capacity if needed - unicode may expand
	if cap(dest) < len(src)*2 {
		newDest := make([]byte, 0, len(src)*2)
		dest = newDest
	}

	var lastWasSpace bool

	// Process byte by byte, handling UTF-8 sequences
	i := 0
	for i < len(src) {
		// Check if ASCII
		if src[i] < 128 {
			// Fast path for ASCII
			entry := n.asciiTable[src[i]]
			if entry.replace {
				// Special handling for spaces
				if entry.char == ' ' {
					if !lastWasSpace {
						dest = append(dest, ' ')
						lastWasSpace = true
					}
				} else {
					dest = append(dest, entry.char)
					lastWasSpace = false
				}
			} else {
				dest = append(dest, src[i])
				lastWasSpace = false
			}
			i++
		} else {
			// Handle UTF-8 multibyte sequence
			r, size := decodeRune(src[i:])

			if unicode.IsPunct(r) || unicode.IsSpace(r) {
				// Replace punctuation with space
				if !lastWasSpace {
					dest = append(dest, ' ')
					lastWasSpace = true
				}
			} else if unicode.IsUpper(r) {
				// Convert to lowercase
				lower := unicode.ToLower(r)

				// Encode lowercase rune to UTF-8
				if lower < 128 {
					dest = append(dest, byte(lower))
				} else if lower < 2048 {
					dest = append(dest, byte(0xC0|(lower>>6)))
					dest = append(dest, byte(0x80|(lower&0x3F)))
				} else if lower < 65536 {
					dest = append(dest, byte(0xE0|(lower>>12)))
					dest = append(dest, byte(0x80|((lower>>6)&0x3F)))
					dest = append(dest, byte(0x80|(lower&0x3F)))
				} else {
					dest = append(dest, byte(0xF0|(lower>>18)))
					dest = append(dest, byte(0x80|((lower>>12)&0x3F)))
					dest = append(dest, byte(0x80|((lower>>6)&0x3F)))
					dest = append(dest, byte(0x80|(lower&0x3F)))
				}
				lastWasSpace = false
			} else {
				// Keep other characters as is
				dest = append(dest, src[i:i+size]...)
				lastWasSpace = false
			}

			i += size
		}
	}

	return dest
}

// decodeRune decodes a UTF-8 sequence to a rune
func decodeRune(b []byte) (rune, int) {
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

// NormalizerFactory extension to support the allocation efficient normalizer
func (f *NormalizerFactory) CreateAllocationEfficientNormalizer() ByteNormalizer {
	return NewAllocationEfficientNormalizer()
}
