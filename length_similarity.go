// length_similarity.go
// Package lengthsimilarity computes a length similarity metric between two texts.
// The metric calculates a score between 0 and 1 based on the difference in word count.
// A score of 1 indicates identical lengths, while lower scores indicate a larger difference.
// The computation uses the formula:
//
//	scaledScore = 1.0 - min(1.0, abs(origLen - augLen) / (origLen * maxDiffRatio))
//
// This version incorporates improved error handling, configuration validation, and support for context.
package lengthsimilarity

import (
	"context"
	"errors"
	"math"
	"strings"
	"unicode"

	"github.com/baditaflorin/l"
)

// NormalizerFunc defines the signature for a text normalization function.
type NormalizerFunc func(text string) string

// defaultNormalize converts the input text to lower case and replaces punctuation with spaces.
func defaultNormalize(text string) string {
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

// Result holds the outcome of the length similarity computation.
type Result struct {
	Name            string
	Score           float64
	Passed          bool
	OriginalLength  int
	AugmentedLength int
	LengthRatio     float64
	Threshold       float64
	Details         map[string]interface{}
}

// Config holds configuration options for the metric.
type Config struct {
	Threshold    float64
	MaxDiffRatio float64
	Logger       l.Logger
	Normalizer   NormalizerFunc
	Precision    int
}

// Option defines a functional option for configuring the metric.
type Option func(*Config)

// WithThreshold sets a custom threshold.
func WithThreshold(th float64) Option {
	return func(cfg *Config) {
		cfg.Threshold = th
	}
}

// WithMaxDiffRatio sets a custom maximum difference ratio.
func WithMaxDiffRatio(ratio float64) Option {
	return func(cfg *Config) {
		cfg.MaxDiffRatio = ratio
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger l.Logger) Option {
	return func(cfg *Config) {
		cfg.Logger = logger
	}
}

// WithNormalizer sets a custom normalization function.
func WithNormalizer(normalizer NormalizerFunc) Option {
	return func(cfg *Config) {
		cfg.Normalizer = normalizer
	}
}

// Default configuration values.
const (
	DefaultThreshold    = 0.7
	DefaultMaxDiffRatio = 0.3
)

// LengthSimilarity provides methods to compute the length similarity metric.
type LengthSimilarity struct {
	config Config
}

// New creates a new LengthSimilarity instance with the provided options.
// Returns an error if configuration validation fails.
func New(opts ...Option) (*LengthSimilarity, error) {
	cfg := Config{
		Threshold:    DefaultThreshold,
		MaxDiffRatio: DefaultMaxDiffRatio,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Threshold < 0 || cfg.Threshold > 1 {
		return nil, errors.New("threshold must be between 0 and 1")
	}
	if cfg.MaxDiffRatio <= 0 {
		return nil, errors.New("maxDiffRatio must be greater than 0")
	}
	if cfg.Logger == nil {
		var err error
		cfg.Logger, err = createDefaultLogger()
		if err != nil {
			return nil, err
		}
	}
	return &LengthSimilarity{config: cfg}, nil
}

// Compute calculates the length similarity metric for the given texts using the configured parameters.
// It accepts a context for cancellation and logs each major step of the computation.
func (ls *LengthSimilarity) Compute(ctx context.Context, original, augmented string) Result {
	ls.config.Logger.Debug("Starting length similarity computation",
		"original", original,
		"augmented", augmented,
	)

	details := make(map[string]interface{})
	normalizeFn := ls.config.Normalizer
	if normalizeFn == nil {
		normalizeFn = defaultNormalize
	}

	normalizedOriginal := normalizeFn(original)
	normalizedAugmented := normalizeFn(augmented)
	ls.config.Logger.Debug("Normalized texts",
		"normalizedOriginal", normalizedOriginal,
		"normalizedAugmented", normalizedAugmented,
	)

	// Check for context cancellation.
	select {
	case <-ctx.Done():
		ls.config.Logger.Error("Computation cancelled", "error", ctx.Err())
		details["error"] = "computation cancelled"
		return Result{
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
	ls.config.Logger.Debug("Computed word counts",
		"original_length", origLen,
		"augmented_length", augLen,
	)

	if origLen == 0 {
		ls.config.Logger.Error("Original text has zero words", "original", original)
		details["error"] = "original text has zero words"
		return Result{
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
	diffRatio := diff / (float64(origLen) * ls.config.MaxDiffRatio)
	if diffRatio > 1.0 {
		diffRatio = 1.0
	}

	scaledScore := 1.0 - diffRatio
	passed := scaledScore >= ls.config.Threshold

	details["original_length"] = origLen
	details["augmented_length"] = augLen
	details["length_ratio"] = lengthRatio
	details["threshold"] = ls.config.Threshold

	ls.config.Logger.Debug("Computed length similarity",
		"score", scaledScore,
		"passed", passed,
		"details", details,
	)

	return Result{
		Name:            "length_similarity",
		Score:           scaledScore,
		Passed:          passed,
		OriginalLength:  origLen,
		AugmentedLength: augLen,
		LengthRatio:     lengthRatio,
		Threshold:       ls.config.Threshold,
		Details:         details,
	}
}
