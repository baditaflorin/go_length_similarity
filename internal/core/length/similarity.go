package length

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/baditaflorin/go_length_similarity/internal/core/domain"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

// SimilarityConfig holds configuration for the length similarity calculator.
type SimilarityConfig struct {
	Threshold    float64
	MaxDiffRatio float64
}

// DefaultConfig returns a default configuration.
func DefaultConfig() SimilarityConfig {
	return SimilarityConfig{
		Threshold:    0.7,
		MaxDiffRatio: 0.3,
	}
}

// Validate checks if the configuration is valid.
func (c SimilarityConfig) Validate() error {
	if c.Threshold < 0 || c.Threshold > 1 {
		return errors.New("threshold must be between 0 and 1")
	}
	if c.MaxDiffRatio <= 0 {
		return errors.New("maxDiffRatio must be greater than 0")
	}
	return nil
}

// Calculator implements the word-level length similarity calculation.
type Calculator struct {
	config     SimilarityConfig
	logger     ports.Logger
	normalizer ports.Normalizer
}

// NewCalculator creates a new length similarity calculator.
func NewCalculator(config SimilarityConfig, logger ports.Logger, normalizer ports.Normalizer) (*Calculator, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Calculator{
		config:     config,
		logger:     logger,
		normalizer: normalizer,
	}, nil
}

// Compute calculates the word-level length similarity between two texts.
func (c *Calculator) Compute(ctx context.Context, original, augmented string) domain.Result {
	c.logger.Debug("Starting length similarity computation",
		"original", original,
		"augmented", augmented,
	)

	details := make(map[string]interface{})

	normalizedOriginal := c.normalizer.Normalize(original)
	normalizedAugmented := c.normalizer.Normalize(augmented)

	c.logger.Debug("Normalized texts",
		"normalizedOriginal", normalizedOriginal,
		"normalizedAugmented", normalizedAugmented,
	)

	// Check for context cancellation.
	select {
	case <-ctx.Done():
		c.logger.Error("Computation cancelled", "error", ctx.Err())
		details["error"] = "computation cancelled"
		return domain.Result{
			Name:    "length_similarity",
			Score:   0,
			Passed:  false,
			Details: details,
		}
	default:
		// continue
	}

	origWords := strings.Fields(normalizedOriginal)
	augWords := strings.Fields(normalizedAugmented)
	origLen := len(origWords)
	augLen := len(augWords)

	c.logger.Debug("Computed word counts",
		"original_length", origLen,
		"augmented_length", augLen,
	)

	if origLen == 0 {
		c.logger.Error("Original text has zero words", "original", original)
		details["error"] = "original text has zero words"
		return domain.Result{
			Name:    "length_similarity",
			Score:   0,
			Passed:  false,
			Details: details,
		}
	}

	var lengthRatio float64
	if origLen > augLen {
		lengthRatio = float64(augLen) / float64(origLen)
	} else {
		lengthRatio = float64(origLen) / float64(augLen)
	}

	diff := math.Abs(float64(origLen - augLen))
	diffRatio := diff / (float64(origLen) * c.config.MaxDiffRatio)
	if diffRatio > 1.0 {
		diffRatio = 1.0
	}

	scaledScore := 1.0 - diffRatio
	passed := scaledScore >= c.config.Threshold

	details["original_length"] = origLen
	details["augmented_length"] = augLen
	details["length_ratio"] = lengthRatio
	details["threshold"] = c.config.Threshold

	c.logger.Debug("Computed length similarity",
		"score", scaledScore,
		"passed", passed,
		"details", details,
	)

	return domain.Result{
		Name:            "length_similarity",
		Score:           scaledScore,
		Passed:          passed,
		OriginalLength:  origLen,
		AugmentedLength: augLen,
		LengthRatio:     lengthRatio,
		Threshold:       c.config.Threshold,
		Details:         details,
	}
}
