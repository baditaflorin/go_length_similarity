// length_similarity.go
// Package lengthsimilarity computes a length similarity metric between two texts.
// The metric calculates a score between 0 and 1 based on the difference in word count.
// A score of 1 indicates identical lengths, while lower scores indicate a larger difference.
// The computation uses the formula:
//
//	scaledScore = 1.0 - min(1.0, abs(origLen - augLen) / (origLen * maxDiffRatio))
//
// This version uses the functional options pattern to allow configuration of parameters
// like threshold, maxDiffRatio, and logging.
package lengthsimilarity

import (
	"math"
	"os"
	"strings"
	"unicode"

	"github.com/baditaflorin/l"
)

// normalize converts the input text to lower case and replaces punctuation characters with spaces.
// This helps ensure that punctuation does not interfere with word splitting while preserving Unicode.
func normalize(text string) string {
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
	// Name of the metric.
	Name string
	// Score is the computed similarity score between 0 and 1.
	Score float64
	// Passed indicates whether the computed score meets or exceeds the threshold.
	Passed bool
	// OriginalLength is the word count of the original text.
	OriginalLength int
	// AugmentedLength is the word count of the augmented text.
	AugmentedLength int
	// LengthRatio is the ratio of the smaller length to the larger length.
	LengthRatio float64
	// Threshold used to determine pass/fail.
	Threshold float64
	// Details holds additional diagnostic information.
	Details map[string]interface{}
}

// Config holds configuration options for the length similarity metric.
type Config struct {
	Threshold    float64
	MaxDiffRatio float64
	// Logger for tracing computation steps.
	Logger l.Logger
}

// Option defines a functional option for configuring the metric.
type Option func(*Config)

// WithThreshold sets a custom threshold.
func WithThreshold(th float64) Option {
	return func(cfg *Config) {
		cfg.Threshold = th
	}
}

// WithMaxDiffRatio sets a custom maxDiffRatio.
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

// Default configuration values.
const (
	DefaultThreshold    = 0.7
	DefaultMaxDiffRatio = 0.3
)

// LengthSimilarity provides methods to compute the length similarity metric
// using configurable parameters.
type LengthSimilarity struct {
	config Config
}

// New creates a new LengthSimilarity with the provided functional options.
// If no logger is provided, a default logger is created.
func New(opts ...Option) *LengthSimilarity {
	cfg := Config{
		Threshold:    DefaultThreshold,
		MaxDiffRatio: DefaultMaxDiffRatio,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	// If no logger is set, create a default logger.
	if cfg.Logger == nil {
		logger, err := l.NewStandardFactory().CreateLogger(l.Config{
			Output:      os.Stdout,
			JsonFormat:  false,
			AsyncWrite:  true,
			BufferSize:  1024 * 1024,      // 1MB buffer
			MaxFileSize: 10 * 1024 * 1024, // 10MB max file size
			MaxBackups:  5,
			AddSource:   true,
			Metrics:     true,
		})
		if err != nil {
			panic(err)
		}
		cfg.Logger = logger
	}
	return &LengthSimilarity{config: cfg}
}

// Compute calculates the length similarity metric for the given texts using the configured parameters.
// It logs key steps of the computation. If the original text contains zero words, it returns a score of 0 and marks it as failed.
func (ls *LengthSimilarity) Compute(original, augmented string) Result {
	ls.config.Logger.Info("Starting length similarity computation",
		"original", original,
		"augmented", augmented,
	)

	details := make(map[string]interface{})

	// Normalize texts.
	normalizedOriginal := normalize(original)
	normalizedAugmented := normalize(augmented)
	ls.config.Logger.Info("Normalized texts",
		"normalizedOriginal", normalizedOriginal,
		"normalizedAugmented", normalizedAugmented,
	)

	// Split texts into words.
	origWords := strings.Fields(normalizedOriginal)
	augWords := strings.Fields(normalizedAugmented)
	origLen := len(origWords)
	augLen := len(augWords)
	ls.config.Logger.Info("Computed word counts",
		"original_length", origLen,
		"augmented_length", augLen,
	)

	// Validate that original text is not empty.
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

	// Compute length ratio: (smaller length / larger length).
	var lengthRatio float64
	if origLen > augLen {
		lengthRatio = float64(augLen) / float64(origLen)
	} else {
		lengthRatio = float64(origLen) / float64(augLen)
	}

	// Calculate the absolute difference in word counts.
	diff := math.Abs(float64(origLen - augLen))
	// Normalize the difference using the product of original length and maxDiffRatio.
	diffRatio := diff / (float64(origLen) * ls.config.MaxDiffRatio)
	// Cap the difference ratio to 1.0.
	if diffRatio > 1.0 {
		diffRatio = 1.0
	}

	// Compute the scaled score (1 means identical lengths).
	scaledScore := 1.0 - diffRatio
	// Determine if the score meets the threshold.
	passed := scaledScore >= ls.config.Threshold

	// Record additional details.
	details["original_length"] = origLen
	details["augmented_length"] = augLen
	details["length_ratio"] = lengthRatio
	details["threshold"] = ls.config.Threshold

	ls.config.Logger.Info("Computed length similarity",
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
