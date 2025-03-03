package normalizer

import (
	"strings"
	"unicode"

	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

// DefaultNormalizer implements the default text normalization strategy.
type DefaultNormalizer struct{}

// NewDefaultNormalizer creates a new default normalizer.
func NewDefaultNormalizer() ports.Normalizer {
	return &DefaultNormalizer{}
}

// Normalize converts the input text to lower case and replaces punctuation with spaces.
func (n *DefaultNormalizer) Normalize(text string) string {
	text = strings.ToLower(text)
	var sb strings.Builder
	for _, r := range text {
		if unicode.IsPunct(r) {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
