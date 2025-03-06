package streaming

import (
	"context"
	"github.com/baditaflorin/go_length_similarity/internal/adapters/logger"
	"github.com/baditaflorin/go_length_similarity/internal/adapters/normalizer"
	"github.com/baditaflorin/go_length_similarity/internal/adapters/stream"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
	"github.com/baditaflorin/l"
	"io"
	"strings"
)

// StreamingMode represents different modes for processing input streams
type StreamingMode int

const (
	// ChunkByChunk processes the input stream in fixed-size chunks
	ChunkByChunk StreamingMode = iota
	// LineByLine processes the input stream line by line
	LineByLine
	// WordByWord processes the input stream word by word
	WordByWord
)

// StreamResult represents the result of a streaming similarity computation
type StreamResult struct {
	Name            string
	Score           float64
	Passed          bool
	OriginalLength  int
	AugmentedLength int
	LengthRatio     float64
	Threshold       float64
	ProcessingTime  string // Duration as string for easy display
	BytesProcessed  int64
	Details         map[string]interface{}
}

// StreamingSimilarity provides methods for streaming similarity computation
type StreamingSimilarity struct {
	calculator *stream.StreamingCalculator
	logger     ports.Logger
}

// StreamingOption defines a functional option for configuring StreamingSimilarity
type StreamingOption func(*streamingConfig)

type streamingConfig struct {
	Threshold    float64
	MaxDiffRatio float64
	ChunkSize    int
	Mode         ports.StreamingMode
	Logger       ports.Logger
	Normalizer   ports.Normalizer
}

// WithStreamingThreshold sets a custom threshold for streaming similarity
func WithStreamingThreshold(th float64) StreamingOption {
	return func(cfg *streamingConfig) {
		cfg.Threshold = th
	}
}

// WithStreamingMaxDiffRatio sets a custom maximum difference ratio for streaming similarity
func WithStreamingMaxDiffRatio(ratio float64) StreamingOption {
	return func(cfg *streamingConfig) {
		cfg.MaxDiffRatio = ratio
	}
}

// WithStreamingChunkSize sets a custom chunk size for streaming
func WithStreamingChunkSize(size int) StreamingOption {
	return func(cfg *streamingConfig) {
		cfg.ChunkSize = size
	}
}

// WithStreamingMode sets a custom streaming mode
func WithStreamingMode(mode StreamingMode) StreamingOption {
	return func(cfg *streamingConfig) {
		cfg.Mode = ports.StreamingMode(mode)
	}
}

// WithStreamingLogger sets a custom logger for streaming similarity
func WithStreamingLogger(l l.Logger) StreamingOption {
	return func(cfg *streamingConfig) {
		cfg.Logger = logger.FromExisting(l)
	}
}

// WithStreamingNormalizer sets a custom normalizer for streaming similarity
func WithStreamingNormalizer(normalizer ports.Normalizer) StreamingOption {
	return func(cfg *streamingConfig) {
		cfg.Normalizer = normalizer
	}
}

// WithOptimizedNormalizer sets the optimized normalizer.
func WithOptimizedNormalizer() StreamingOption {
	return func(cfg *streamingConfig) {
		normFactory := normalizer.NewNormalizerFactory()
		cfg.Normalizer = normFactory.CreateNormalizer(normalizer.OptimizedNormalizerType)
	}
}

// NewStreamingSimilarity creates a new StreamingSimilarity instance
func NewStreamingSimilarity(opts ...StreamingOption) (*StreamingSimilarity, error) {
	// Default configuration
	config := &streamingConfig{
		Threshold:    0.7,
		MaxDiffRatio: 0.3,
		ChunkSize:    8192,
		Mode:         ports.LineByLine,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Set up logger if not provided
	if config.Logger == nil {
		var err error
		config.Logger, err = logger.NewStdLogger()
		if err != nil {
			return nil, err
		}
	}

	// Set up normalizer if not provided
	if config.Normalizer == nil {
		normFactory := normalizer.NewNormalizerFactory()
		config.Normalizer = normFactory.CreateNormalizer(normalizer.OptimizedNormalizerType)
	}

	// Create core calculator
	streamingConfig := stream.StreamingConfig{
		Threshold:    config.Threshold,
		MaxDiffRatio: config.MaxDiffRatio,
		ChunkSize:    config.ChunkSize,
		Mode:         config.Mode,
	}
	calculator, err := stream.NewStreamingCalculator(streamingConfig, config.Logger, config.Normalizer)
	if err != nil {
		return nil, err
	}

	return &StreamingSimilarity{
		calculator: calculator,
		logger:     config.Logger,
	}, nil
}

// ComputeFromReaders calculates the streaming similarity between two text readers
func (ss *StreamingSimilarity) ComputeFromReaders(ctx context.Context, original io.Reader, augmented io.Reader) StreamResult {
	result := ss.calculator.ComputeStreaming(ctx, original, augmented)

	// Convert internal result to public result
	return StreamResult{
		Name:            result.Name,
		Score:           result.Score,
		Passed:          result.Passed,
		OriginalLength:  result.OriginalLength,
		AugmentedLength: result.AugmentedLength,
		LengthRatio:     result.LengthRatio,
		Threshold:       result.Threshold,
		ProcessingTime:  result.ProcessingTime.String(),
		BytesProcessed:  result.BytesProcessed,
		Details:         result.Details,
	}
}

// ComputeFromStrings calculates the streaming similarity between two strings
// This is a convenience method that wraps the strings in readers
func (ss *StreamingSimilarity) ComputeFromStrings(ctx context.Context, original, augmented string) StreamResult {
	originalReader := strings.NewReader(original)
	augmentedReader := strings.NewReader(augmented)

	return ss.ComputeFromReaders(ctx, originalReader, augmentedReader)
}
