// File: pkg/streaming/optimized_streaming.go
package streaming

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/baditaflorin/go_length_similarity/internal/adapters/normalizer"
	"github.com/baditaflorin/go_length_similarity/internal/adapters/stream/lineprocessor"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
	"github.com/baditaflorin/l"
)

// AllocationEfficientStreamingSimilarity provides a highly optimized streaming similarity implementation
type AllocationEfficientStreamingSimilarity struct {
	logger         ports.Logger
	normalizer     ports.Normalizer
	byteNormalizer normalizer.ByteNormalizer
	lineProcessor  *lineprocessor.OptimizedProcessor
	config         AllocationEfficientConfig
}

// AllocationEfficientConfig holds configuration for the allocation-efficient streaming similarity
type AllocationEfficientConfig struct {
	Threshold    float64
	MaxDiffRatio float64
	ChunkSize    int
	Mode         ports.StreamingMode
	UseParallel  bool
	BatchSize    int
}

// AllocationEfficientOption defines a functional option for configuring AllocationEfficientStreamingSimilarity
type AllocationEfficientOption func(*AllocationEfficientConfig)

// WithEfficientThreshold sets a custom threshold
func WithEfficientThreshold(th float64) AllocationEfficientOption {
	return func(cfg *AllocationEfficientConfig) {
		cfg.Threshold = th
	}
}

// WithEfficientMaxDiffRatio sets a custom maximum difference ratio
func WithEfficientMaxDiffRatio(ratio float64) AllocationEfficientOption {
	return func(cfg *AllocationEfficientConfig) {
		cfg.MaxDiffRatio = ratio
	}
}

// WithEfficientChunkSize sets a custom chunk size
func WithEfficientChunkSize(size int) AllocationEfficientOption {
	return func(cfg *AllocationEfficientConfig) {
		cfg.ChunkSize = size
	}
}

// WithEfficientMode sets a custom streaming mode
func WithEfficientMode(mode StreamingMode) AllocationEfficientOption {
	return func(cfg *AllocationEfficientConfig) {
		cfg.Mode = ports.StreamingMode(mode)
	}
}

// WithEfficientParallel enables or disables parallel processing
func WithEfficientParallel(enable bool) AllocationEfficientOption {
	return func(cfg *AllocationEfficientConfig) {
		cfg.UseParallel = enable
	}
}

// WithEfficientBatchSize sets a custom batch size for line processing
func WithEfficientBatchSize(size int) AllocationEfficientOption {
	return func(cfg *AllocationEfficientConfig) {
		cfg.BatchSize = size
	}
}

// NewAllocationEfficientStreamingSimilarity creates a new allocation-efficient streaming similarity calculator
func NewAllocationEfficientStreamingSimilarity(logger l.Logger, opts ...AllocationEfficientOption) (*AllocationEfficientStreamingSimilarity, error) {
	// Default configuration
	config := &AllocationEfficientConfig{
		Threshold:    0.7,
		MaxDiffRatio: 0.3,
		ChunkSize:    64 * 1024, // 64KB chunks
		Mode:         ports.LineByLine,
		UseParallel:  true,
		BatchSize:    100,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Create the allocation-efficient normalizer
	normFactory := normalizer.NewNormalizerFactory()
	byteNorm := normFactory.CreateAllocationEfficientNormalizer()

	// Create the optimized line processor
	lineProc := lineprocessor.NewOptimizedProcessor(
		logger,
		byteNorm.(ports.Normalizer),
		lineprocessor.ProcessingConfig{
			ChunkSize:   config.ChunkSize,
			BatchSize:   config.BatchSize,
			UseParallel: config.UseParallel,
		},
	)

	return &AllocationEfficientStreamingSimilarity{
		logger:         logger,
		normalizer:     byteNorm.(ports.Normalizer),
		byteNormalizer: byteNorm,
		lineProcessor:  lineProc,
		config:         *config,
	}, nil
}

// ComputeFromReaders calculates the streaming similarity between two text readers
func (aes *AllocationEfficientStreamingSimilarity) ComputeFromReaders(ctx context.Context, original io.Reader, augmented io.Reader) StreamResult {
	startTime := time.Now()

	// Process original text stream
	origCount, origBytes, err := aes.lineProcessor.ProcessLines(ctx, original, nil)
	if err != nil && err != io.EOF {
		aes.logger.Error("Error processing original stream", "error", err)
		return StreamResult{
			Name:           "streaming_similarity",
			Score:          0,
			Passed:         false,
			Details:        map[string]interface{}{"error": "error processing original stream: " + err.Error()},
			ProcessingTime: time.Since(startTime).String(),
		}
	}

	// Process augmented text stream
	augCount, augBytes, err := aes.lineProcessor.ProcessLines(ctx, augmented, nil)
	if err != nil && err != io.EOF {
		aes.logger.Error("Error processing augmented stream", "error", err)
		return StreamResult{
			Name:           "streaming_similarity",
			Score:          0,
			Passed:         false,
			Details:        map[string]interface{}{"error": "error processing augmented stream: " + err.Error()},
			ProcessingTime: time.Since(startTime).String(),
		}
	}

	// Calculate similarity using the similar algorithm as the regular version
	var lengthRatio float64
	var score float64
	var passed bool

	// Special case: both empty texts
	if origCount == 0 && augCount == 0 {
		lengthRatio = 1.0
		score = 1.0
		passed = true
	} else if origCount == 0 {
		// Original text is empty
		lengthRatio = 0.0
		score = 0.0
		passed = false
	} else {
		// Standard calculation
		if origCount > augCount {
			lengthRatio = float64(augCount) / float64(origCount)
		} else {
			lengthRatio = float64(origCount) / float64(augCount)
		}

		diff := float64(origCount - augCount)
		if diff < 0 {
			diff = -diff
		}

		diffRatio := diff / (float64(origCount) * aes.config.MaxDiffRatio)
		if diffRatio > 1.0 {
			diffRatio = 1.0
		}

		score = 1.0 - diffRatio
		passed = score >= aes.config.Threshold
	}

	// Create detailed result
	details := map[string]interface{}{
		"original_length":           origCount,
		"augmented_length":          augCount,
		"length_ratio":              lengthRatio,
		"threshold":                 aes.config.Threshold,
		"mode":                      aes.config.Mode,
		"parallel":                  aes.config.UseParallel,
		"bytes_processed_original":  origBytes,
		"bytes_processed_augmented": augBytes,
	}

	totalBytes := origBytes + augBytes
	duration := time.Since(startTime)

	aes.logger.Debug("Computed allocation-efficient streaming similarity",
		"score", score,
		"passed", passed,
		"details", details,
		"duration", duration,
	)

	return StreamResult{
		Name:            "streaming_similarity",
		Score:           score,
		Passed:          passed,
		OriginalLength:  origCount,
		AugmentedLength: augCount,
		LengthRatio:     lengthRatio,
		Threshold:       aes.config.Threshold,
		ProcessingTime:  duration.String(),
		BytesProcessed:  totalBytes,
		Details:         details,
	}
}

// ComputeFromStrings calculates the streaming similarity between two strings
// This is a convenience method that wraps the strings in readers
func (aes *AllocationEfficientStreamingSimilarity) ComputeFromStrings(ctx context.Context, original, augmented string) StreamResult {
	originalReader := strings.NewReader(original)
	augmentedReader := strings.NewReader(augmented)

	return aes.ComputeFromReaders(ctx, originalReader, augmentedReader)
}
