package normalizer

import (
	"unicode"

	"github.com/baditaflorin/go_length_similarity/internal/pool"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

// OptimizedNormalizer implements an optimized text normalization strategy with buffer pooling
type OptimizedNormalizer struct {
	// Pre-computed decision table for ASCII characters (0-127)
	asciiTable [128]byte

	// Reusable buffer pool - only need one buffer type
	bytePool *pool.BufferPool
}

// NewOptimizedNormalizer creates a new optimized normalizer
func NewOptimizedNormalizer() ports.Normalizer {
	n := &OptimizedNormalizer{
		bytePool: pool.NewBufferPool(8192), // 8K bytes initial capacity
	}

	// Initialize lookup table for ASCII characters
	// 0 = keep as is
	// 1 = replace with space
	// 2 = convert to lowercase
	for i := 0; i < 128; i++ {
		r := rune(i)
		if unicode.IsPunct(r) || unicode.IsSpace(r) {
			// Replace punctuation and whitespace with space
			n.asciiTable[i] = 1
		} else if unicode.IsUpper(r) {
			// Convert uppercase to lowercase
			n.asciiTable[i] = 2
		} else {
			// Keep as is
			n.asciiTable[i] = 0
		}
	}

	return n
}

// Normalize converts the input text to lower case and replaces punctuation with spaces efficiently
func (n *OptimizedNormalizer) Normalize(text string) string {
	// Fast path for empty strings
	if len(text) == 0 {
		return ""
	}

	// Check for ASCII-only string first (optimization)
	asciiOnly := true
	for i := 0; i < len(text); i++ {
		if text[i] >= 128 {
			asciiOnly = false
			break
		}
	}

	// Get a reusable buffer from the pool
	buffer := n.bytePool.Get()
	defer n.bytePool.Put(buffer)

	// Ensure the buffer has adequate capacity
	if cap(*buffer) < len(text) {
		*buffer = make([]byte, 0, len(text))
	}
	*buffer = (*buffer)[:0] // Reset length while keeping capacity

	if asciiOnly {
		// Fast path for ASCII-only strings
		var lastWasSpace bool
		for i := 0; i < len(text); i++ {
			b := text[i]
			switch n.asciiTable[b] {
			case 0: // Keep as is
				*buffer = append(*buffer, b)
				lastWasSpace = false
			case 1: // Replace with space
				// Avoid consecutive spaces
				if !lastWasSpace {
					*buffer = append(*buffer, ' ')
					lastWasSpace = true
				}
			case 2: // Convert to lowercase (ASCII)
				*buffer = append(*buffer, b+('a'-'A'))
				lastWasSpace = false
			}
		}

		return string(*buffer)
	}

	// Slower path for mixed ASCII/Unicode strings
	var lastWasSpace bool
	for _, r := range text {
		if r < 128 {
			// ASCII character - use lookup table
			switch n.asciiTable[r] {
			case 0: // Keep as is
				*buffer = append(*buffer, byte(r))
				lastWasSpace = false
			case 1: // Replace with space
				// Avoid consecutive spaces
				if !lastWasSpace {
					*buffer = append(*buffer, ' ')
					lastWasSpace = true
				}
			case 2: // Convert to lowercase (ASCII)
				*buffer = append(*buffer, byte(r)+('a'-'A'))
				lastWasSpace = false
			}
		} else {
			// Non-ASCII character
			if unicode.IsPunct(r) || unicode.IsSpace(r) {
				// Replace punctuation with space
				if !lastWasSpace {
					*buffer = append(*buffer, ' ')
					lastWasSpace = true
				}
			} else {
				// Convert to lowercase and append the UTF-8 bytes
				lower := unicode.ToLower(r)
				runeBytes := []byte(string(lower))
				*buffer = append(*buffer, runeBytes...)
				lastWasSpace = false
			}
		}
	}

	return string(*buffer)
}

// FastNormalizer offers an even faster normalization with pre-cached decisions
// for ASCII characters and minimal allocations
type FastNormalizer struct {
	// Pre-computed decision table for ASCII characters (0-127)
	asciiTable [128]struct {
		replace bool
		char    rune
	}

	// Pools for reusing buffers
	runePool    *pool.RuneBufferPool
	builderPool *pool.StringBuilderPool
}

// NewFastNormalizer creates a new fast normalizer with precomputed tables
func NewFastNormalizer() ports.Normalizer {
	n := &FastNormalizer{
		runePool:    pool.NewRuneBufferPool(8192),
		builderPool: pool.NewStringBuilderPool(),
	}

	// Initialize the decision table for ASCII characters
	for i := 0; i < 128; i++ {
		r := rune(i)
		if unicode.IsPunct(r) {
			n.asciiTable[i] = struct {
				replace bool
				char    rune
			}{
				replace: true,
				char:    ' ',
			}
		} else if unicode.IsUpper(r) {
			n.asciiTable[i] = struct {
				replace bool
				char    rune
			}{
				replace: true,
				char:    unicode.ToLower(r),
			}
		} else {
			n.asciiTable[i] = struct {
				replace bool
				char    rune
			}{
				replace: false,
			}
		}
	}

	return n
}

// Normalize performs fast normalization with pre-computed decisions for ASCII
func (n *FastNormalizer) Normalize(text string) string {
	// Fast path for empty strings
	if len(text) == 0 {
		return ""
	}

	// Get a buffer from the pool
	sb := n.builderPool.Get()
	defer n.builderPool.Put(sb)

	// Fast path for ASCII-only strings
	asciiOnly := true
	for _, r := range text {
		if r >= 128 {
			asciiOnly = false
			break
		}
	}

	if asciiOnly {
		// Use the precomputed table for ASCII
		for _, r := range text {
			entry := n.asciiTable[r]
			if entry.replace {
				sb.WriteRune(entry.char)
			} else {
				sb.WriteRune(r)
			}
		}
	} else {
		// Fallback for non-ASCII characters
		for _, r := range text {
			if r < 128 {
				// Use precomputed table for ASCII
				entry := n.asciiTable[r]
				if entry.replace {
					sb.WriteRune(entry.char)
				} else {
					sb.WriteRune(r)
				}
			} else {
				// Process non-ASCII characters
				if unicode.IsPunct(r) {
					sb.WriteRune(' ')
				} else {
					sb.WriteRune(unicode.ToLower(r))
				}
			}
		}
	}

	return sb.String()
}

// NormalizerFactory creates the appropriate normalizer based on performance requirements
type NormalizerFactory struct{}

// NewNormalizerFactory creates a new normalizer factory
func NewNormalizerFactory() *NormalizerFactory {
	return &NormalizerFactory{}
}

// Type of normalizer to create
type NormalizerType int

const (
	// DefaultNormalizerType is the original normalizer
	DefaultNormalizerType NormalizerType = iota
	// OptimizedNormalizerType uses buffer pooling and optimized algorithms
	OptimizedNormalizerType
	// FastNormalizerType uses precomputed tables and is optimized for ASCII
	FastNormalizerType
)

// CreateNormalizer creates a normalizer of the specified type
func (f *NormalizerFactory) CreateNormalizer(normalizerType NormalizerType) ports.Normalizer {
	switch normalizerType {
	case OptimizedNormalizerType:
		return NewOptimizedNormalizer()
	case FastNormalizerType:
		return NewFastNormalizer()
	default:
		return NewDefaultNormalizer()
	}
}
