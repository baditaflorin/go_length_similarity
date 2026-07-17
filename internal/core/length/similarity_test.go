package length

import (
	"context"
	"testing"

	"github.com/baditaflorin/go_length_similarity/internal/adapters/normalizer"
)

type discardLogger struct{}

func (discardLogger) Debug(string, ...interface{}) {}
func (discardLogger) Info(string, ...interface{})  {}
func (discardLogger) Warn(string, ...interface{})  {}
func (discardLogger) Error(string, ...interface{}) {}
func (discardLogger) Close() error                 { return nil }

func TestComputeIgnoresSharedChrome(t *testing.T) {
	calculator, err := NewCalculator(DefaultConfig(), discardLogger{}, normalizer.NewDefaultNormalizer())
	if err != nil {
		t.Fatal(err)
	}
	first := `<nav>Products Pricing Sign in</nav><main>alpha beta gamma delta</main>`
	second := `<nav>Products Pricing Sign in</nav><main>one two three four five six seven eight</main>`
	result := calculator.Compute(context.Background(), first, second)
	if result.OriginalLength != 4 || result.AugmentedLength != 8 {
		t.Fatalf("expected only visible main prose to be counted, got %d and %d", result.OriginalLength, result.AugmentedLength)
	}
	if result.Passed {
		t.Fatalf("different page bodies must not pass because shared navigation matched: %#v", result)
	}
}

func TestComputeRejectsShortTemplateEvidence(t *testing.T) {
	calculator, err := NewCalculator(DefaultConfig(), discardLogger{}, normalizer.NewDefaultNormalizer())
	if err != nil {
		t.Fatal(err)
	}
	result := calculator.Compute(context.Background(), "welcome", "welcome")
	if result.Passed || result.Score != 0 {
		t.Fatalf("one-word templates must not produce a similarity finding: %#v", result)
	}
}
