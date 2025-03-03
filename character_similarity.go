// character_similarity.go
// Package lengthsimilarity provides an alternative character-level similarity metric between two texts.
// The metric calculates a score between 0 and 1 based on the difference in character counts.
// A score of 1 indicates identical lengths, while lower scores indicate a larger difference.
// The computation uses the formula:
//
//	scaledScore = 1.0 - min(1.0, abs(origLen - augLen) / (origLen * maxDiffRatio))
//
// This version improves error handling, configuration validation, and adds support for context cancellation.
package lengthsimilarity

import (
	"context"
	"errors"
	"math"
)

// CharacterSimilarity provides methods to compute a character-level similarity metric using configurable parameters.
type CharacterSimilarity struct {
	config Config
}

// NewCharacterSimilarity creates a new CharacterSimilarity instance with the provided functional options.
// Returns an error if the configuration is invalid.
func NewCharacterSimilarity(opts ...Option) (*CharacterSimilarity, error) {
	cfg := Config{
		Threshold:    DefaultThreshold,
		MaxDiffRatio: DefaultMaxDiffRatio,
		Precision:    2,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	// Validate config parameters.
	if cfg.Threshold < 0 || cfg.Threshold > 1 {
		return nil, errors.New("threshold must be between 0 and 1")
	}
	if cfg.MaxDiffRatio <= 0 {
		return nil, errors.New("maxDiffRatio must be greater than 0")
	}
	// Create a default logger if one is not provided.
	if cfg.Logger == nil {
		var err error
		cfg.Logger, err = createDefaultLogger()
		if err != nil {
			return nil, err
		}
	}
	return &CharacterSimilarity{config: cfg}, nil
}

// WithPrecision sets a custom precision for rounding computed float values.
func WithPrecision(p int) Option {
	return func(cfg *Config) {
		cfg.Precision = p
	}
}

// Compute calculates the character-level similarity metric for the given texts using the configured parameters.
// It accepts a context for cancellation and logs key steps of the computation.
func (cs *CharacterSimilarity) Compute(ctx context.Context, original, augmented string) Result {
	cs.config.Logger.Debug("Starting character similarity computation",
		"original", original,
		"augmented", augmented,
	)

	details := make(map[string]interface{})

	normalizeFn := cs.config.Normalizer
	if normalizeFn == nil {
		normalizeFn = defaultNormalize
	}

	normalizedOriginal := normalizeFn(original)
	normalizedAugmented := normalizeFn(augmented)
	cs.config.Logger.Debug("Normalized texts",
		"normalizedOriginal", normalizedOriginal,
		"normalizedAugmented", normalizedAugmented,
	)

	// Check context cancellation.
	select {
	case <-ctx.Done():
		cs.config.Logger.Error("Computation cancelled", "error", ctx.Err())
		details["error"] = "computation cancelled"
		return Result{
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
	cs.config.Logger.Debug("Computed character counts",
		"original_length", origLen,
		"augmented_length", augLen,
	)

	if origLen == 0 {
		cs.config.Logger.Error("Original text has zero characters", "original", original)
		details["error"] = "original text has zero characters"
		return Result{
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
	diffRatio := diff / (float64(origLen) * cs.config.MaxDiffRatio)
	if diffRatio > 1.0 {
		diffRatio = 1.0
	}

	scaledScore := 1.0 - diffRatio
	// Round the score to the configured precision.
	factor := math.Pow(10, float64(cs.config.Precision))
	scaledScore = math.Round(scaledScore*factor) / factor
	lengthRatio = math.Round(lengthRatio*factor) / factor

	passed := scaledScore >= cs.config.Threshold

	details["original_length"] = origLen
	details["augmented_length"] = augLen
	details["length_ratio"] = lengthRatio
	details["threshold"] = cs.config.Threshold

	cs.config.Logger.Debug("Computed character similarity",
		"score", scaledScore,
		"passed", passed,
		"details", details,
	)

	return Result{
		Name:            "character_similarity",
		Score:           scaledScore,
		Passed:          passed,
		OriginalLength:  origLen,
		AugmentedLength: augLen,
		LengthRatio:     lengthRatio,
		Threshold:       cs.config.Threshold,
		Details:         details,
	}
}
