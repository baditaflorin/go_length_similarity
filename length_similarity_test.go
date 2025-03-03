// length_similarity_test.go
package lengthsimilarity

import (
	"testing"
)

func TestComputeWithDefaults(t *testing.T) {
	// Test cases with varying word counts.
	tests := []struct {
		name     string
		orig     string
		aug      string
		expected bool // whether the result should pass based on default threshold
	}{
		{
			name: "Identical texts",
			orig: "The quick brown fox jumps over the lazy dog.",
			aug:  "The quick brown fox jumps over the lazy dog.",
			// Identical lengths should pass.
			expected: true,
		},
		{
			name:     "Slightly shorter augmented text",
			orig:     "The quick brown fox jumps over the lazy dog.",
			aug:      "The quick brown fox jumps over dog.",
			expected: true,
		},
		{
			name:     "Much shorter augmented text",
			orig:     "The quick brown fox jumps over the lazy dog.",
			aug:      "Quick fox jumps.",
			expected: false,
		},
		{
			name: "Empty original text",
			orig: "",
			aug:  "Some text here.",
			// This should fail because the original text is empty.
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ComputeWithDefaults(tc.orig, tc.aug)
			if result.Passed != tc.expected {
				t.Errorf("expected passed=%v, got %v, details: %v", tc.expected, result.Passed, result.Details)
			}
		})
	}
}
