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
	// MinWords prevents boilerplate snippets and one-word templates from
	// being reported as high-confidence content similarity.
	MinWords int
}

// DefaultConfig returns a default configuration.
func DefaultConfig() SimilarityConfig {
	return SimilarityConfig{
		Threshold:    0.7,
		MaxDiffRatio: 0.3,
		MinWords:     3,
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
	if c.MinWords < 1 {
		return errors.New("minWords must be at least 1")
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

	normalizedOriginal := c.normalizer.Normalize(visibleComparisonText(original))
	normalizedAugmented := c.normalizer.Normalize(visibleComparisonText(augmented))

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
	if origLen < c.config.MinWords || augLen < c.config.MinWords {
		details["error"] = "insufficient normalized text"
		details["minimum_words"] = c.config.MinWords
		details["original_length"] = origLen
		details["augmented_length"] = augLen
		return domain.Result{
			Name:            "length_similarity",
			Score:           0,
			Passed:          false,
			OriginalLength:  origLen,
			AugmentedLength: augLen,
			Threshold:       c.config.Threshold,
			Details:         details,
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

// visibleComparisonText removes markup and page-chrome blocks before the
// normalizer counts words. Length similarity is used as content evidence, so
// shared navigation, cookie banners, and scripts must not make unrelated
// pages appear similar.
func visibleComparisonText(input string) string {
	var out strings.Builder
	for i := 0; i < len(input); {
		if strings.HasPrefix(input[i:], "<!--") {
			if end := strings.Index(input[i+4:], "-->"); end >= 0 {
				i += end + 7
				continue
			}
			break
		}
		if input[i] != '<' {
			out.WriteByte(input[i])
			i++
			continue
		}
		end := htmlTagEnd(input, i+1)
		if end < 0 {
			// A literal '<' in prose is still content; don't discard the rest
			// of a malformed response.
			out.WriteByte(input[i])
			i++
			continue
		}
		tag := htmlTagName(input[i+1 : end])
		i = end + 1
		if tag == "" || strings.HasPrefix(tag, "/") {
			continue
		}
		if nonContentHTMLTags[tag] {
			closing := "</" + tag
			if closeStart := strings.Index(strings.ToLower(input[i:]), closing); closeStart >= 0 {
				if closeEnd := htmlTagEnd(input, i+closeStart+len(closing)); closeEnd >= 0 {
					i = closeEnd + 1
					continue
				}
			}
		}
	}
	return out.String()
}

var nonContentHTMLTags = map[string]bool{
	"script": true, "style": true, "noscript": true, "template": true,
	"nav": true, "footer": true, "header": true, "aside": true,
}

func htmlTagName(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	raw = strings.TrimPrefix(raw, "/")
	if raw == "" || raw[0] == '!' || raw[0] == '?' {
		return ""
	}
	for i, r := range raw {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '/' {
			return raw[:i]
		}
	}
	return raw
}

func htmlTagEnd(input string, start int) int {
	var quote byte
	for i := start; i < len(input); i++ {
		if quote != 0 {
			if input[i] == quote {
				quote = 0
			}
			continue
		}
		if input[i] == '\'' || input[i] == '"' {
			quote = input[i]
		} else if input[i] == '>' {
			return i
		}
	}
	return -1
}
