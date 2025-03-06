package normalizer

import (
	"unicode"

	"github.com/baditaflorin/go_length_similarity/internal/pool"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

// OptimizedNormalizer implements an optimized text normalization strategy with buffer pooling
type OptimizedNormalizer struct {
	runePool    *pool.RuneBufferPool
	builderPool *pool.StringBuilderPool
}

// NewOptimizedNormalizer creates a new optimized normalizer
func NewOptimizedNormalizer() ports.Normalizer {
	return &OptimizedNormalizer{
		runePool:    pool.NewRuneBufferPool(8192), // 8K runes initial capacity
		builderPool: pool.NewStringBuilderPool(),
	}
}

// Normalize converts the input text to lower case and replaces punctuation with spaces efficiently
func (n *OptimizedNormalizer) Normalize(text string) string {
	// No processing needed for empty strings
	if len(text) == 0 {
		return ""
	}

	// Get reusable buffers from the pools
	runeBuffer := n.runePool.Get()
	defer n.runePool.Put(runeBuffer)

	sb := n.builderPool.Get()
	defer n.builderPool.Put(sb)

	// Pre-allocate capacity if needed by appending to the slice
	textRunes := []rune(text)
	if cap(*runeBuffer) < len(textRunes) {
		// Expand capacity if needed
		*runeBuffer = make([]rune, 0, len(textRunes))
	}

	// Process the text in a single pass
	for _, r := range textRunes {
		if unicode.IsPunct(r) {
			sb.WriteRune(' ')
		} else {
			// Convert to lowercase as we go
			sb.WriteRune(unicode.ToLower(r))
		}
	}

	return sb.String()
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
