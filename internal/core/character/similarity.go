package character

import (
	"context"
	"errors"
	"math"

	"github.com/baditaflorin/go_length_similarity/internal/core/domain"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

// SimilarityConfig holds configuration for the character similarity calculator.
type SimilarityConfig struct {
	Threshold    float64
	MaxDiffRatio float64
	Precision    int
}

// DefaultConfig returns a default configuration.
func DefaultConfig() SimilarityConfig {
	return SimilarityConfig{
		Threshold:    0.7,
		MaxDiffRatio: 0.3,
		Precision:    2,
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

// Calculator implements the character-level similarity calculation.
type Calculator struct {
	config     SimilarityConfig
	logger     ports.Logger
	normalizer ports.Normalizer
}

// NewCalculator creates a new character similarity calculator.
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

// Compute calculates the character-level similarity between two texts.
func (c *Calculator) Compute(ctx context.Context, original, augmented string) domain.Result {
	c.logger.Debug("Starting character similarity computation",
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

	// Check context cancellation.
	select {
	case <-ctx.Done():
		c.logger.Error("Computation cancelled", "error", ctx.Err())
		details["error"] = "computation cancelled"
		return domain.Result{
			Name:    "character_similarity",
			Score:   0,
			Passed:  false,
			Details: details,
		}
	default:
		// continue
	}

	origRunes := []rune(normalizedOriginal)
	augRunes := []rune(normalizedAugmented)
	origLen := len(origRunes)
	augLen := len(augRunes)

	c.logger.Debug("Computed character counts",
		"original_length", origLen,
		"augmented_length", augLen,
	)

	if origLen == 0 {
		c.logger.Error("Original text has zero characters", "original", original)
		details["error"] = "original text has zero characters"
		return domain.Result{
			Name:    "character_similarity",
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
	// Round the score to the configured precision.
	factor := math.Pow(10, float64(c.config.Precision))
	scaledScore = math.Round(scaledScore*factor) / factor
	lengthRatio = math.Round(lengthRatio*factor) / factor

	passed := scaledScore >= c.config.Threshold

	details["original_length"] = origLen
	details["augmented_length"] = augLen
	details["length_ratio"] = lengthRatio
	details["threshold"] = c.config.Threshold

	c.logger.Debug("Computed character similarity",
		"score", scaledScore,
		"passed", passed,
		"details", details,
	)

	return domain.Result{
		Name:            "character_similarity",
		Score:           scaledScore,
		Passed:          passed,
		OriginalLength:  origLen,
		AugmentedLength: augLen,
		LengthRatio:     lengthRatio,
		Threshold:       c.config.Threshold,
		Details:         details,
	}
}
