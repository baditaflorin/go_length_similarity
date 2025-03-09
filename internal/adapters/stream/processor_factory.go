// File: internal/adapters/stream/processor_factory.go
package stream

import (
	"context"
	"github.com/baditaflorin/go_length_similarity/internal/adapters/normalizer"
	"github.com/baditaflorin/go_length_similarity/internal/adapters/stream/lineprocessor"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
	"io"
)

// ProcessorMode defines different processor implementations
type ProcessorMode int

const (
	// StandardProcessor uses the original implementation
	StandardProcessor ProcessorMode = iota

	// OptimizedProcessor uses the current optimized implementation
	OptimizedProcessor

	// AllocationEfficientProcessor uses the new allocation-efficient implementation
	AllocationEfficientProcessor
)

// ProcessorFactory creates the appropriate stream processor based on requirements
type ProcessorFactory struct {
	logger ports.Logger
}

// NewProcessorFactory creates a new processor factory
func NewProcessorFactory(logger ports.Logger) *ProcessorFactory {
	return &ProcessorFactory{
		logger: logger,
	}
}

// CreateProcessor creates a processor with the specified configuration
func (f *ProcessorFactory) CreateProcessor(
	mode ProcessorMode,
	normalizerType normalizer.NormalizerType,
	config ProcessorConfig,
) ports.StreamProcessor {
	// Create normalizer based on type
	normFactory := normalizer.NewNormalizerFactory()
	var norm ports.Normalizer

	// If allocation efficient processor is requested with non-AllocationEfficient normalizer,
	// force the allocation efficient normalizer
	if mode == AllocationEfficientProcessor {
		byteNorm := normFactory.CreateAllocationEfficientNormalizer()
		norm = byteNorm.(ports.Normalizer)
	} else {
		norm = normFactory.CreateNormalizer(normalizerType)
	}

	// Create processor based on mode
	switch mode {
	case OptimizedProcessor:
		// Create the current optimized processor
		lineProc := lineprocessor.NewProcessor(
			f.logger,
			norm,
			lineprocessor.ProcessingConfig{
				ChunkSize:   config.ChunkSize,
				BatchSize:   config.BatchSize,
				UseParallel: config.UseParallel,
			},
		)

		// Create a stream processor adapter that uses the line processor
		return NewStreamProcessorWithLineProcessor(f.logger, lineProc)

	case AllocationEfficientProcessor:
		// Create the allocation-efficient processor
		efficientProc := lineprocessor.NewOptimizedProcessor(
			f.logger,
			norm,
			lineprocessor.ProcessingConfig{
				ChunkSize:   config.ChunkSize,
				BatchSize:   config.BatchSize,
				UseParallel: config.UseParallel,
			},
		)

		// Create a stream processor adapter that uses the allocation-efficient processor
		return NewStreamProcessorWithOptimizedLineProcessor(f.logger, efficientProc)

	default: // StandardProcessor
		// Create the standard processor
		processor := NewDefaultProcessor(f.logger, norm)
		if config.ChunkSize > 0 {
			processor.WithChunkSize(config.ChunkSize)
		}
		if config.UseParallel {
			processor.WithParallelProcessing(true)
		}
		return processor
	}
}

// ProcessorConfig defines configuration for creating processors
type ProcessorConfig struct {
	ChunkSize   int
	BatchSize   int
	UseParallel bool
}

// StreamProcessorWithLineProcessor adapts a line processor to the StreamProcessor interface
type StreamProcessorWithLineProcessor struct {
	logger    ports.Logger
	processor *lineprocessor.Processor
}

// NewStreamProcessorWithLineProcessor creates a new stream processor that uses a line processor
func NewStreamProcessorWithLineProcessor(logger ports.Logger, processor *lineprocessor.Processor) *StreamProcessorWithLineProcessor {
	return &StreamProcessorWithLineProcessor{
		logger:    logger,
		processor: processor,
	}
}

// ProcessStream processes an input stream and returns the length
func (sp *StreamProcessorWithLineProcessor) ProcessStream(ctx context.Context, reader io.Reader, mode ports.StreamingMode) (int, error) {
	count, _, err := sp.processor.ProcessLines(ctx, reader, nil)
	return count, err
}

// ProcessStreamWithWriter processes an input stream and writes to the output writer
func (sp *StreamProcessorWithLineProcessor) ProcessStreamWithWriter(ctx context.Context, reader io.Reader, writer io.Writer, mode ports.StreamingMode) (int, error) {
	count, _, err := sp.processor.ProcessLines(ctx, reader, writer)
	return count, err
}

// StreamProcessorWithOptimizedLineProcessor adapts an optimized line processor to the StreamProcessor interface
type StreamProcessorWithOptimizedLineProcessor struct {
	logger    ports.Logger
	processor *lineprocessor.OptimizedProcessor
}

// NewStreamProcessorWithOptimizedLineProcessor creates a new stream processor with an optimized line processor
func NewStreamProcessorWithOptimizedLineProcessor(logger ports.Logger, processor *lineprocessor.OptimizedProcessor) *StreamProcessorWithOptimizedLineProcessor {
	return &StreamProcessorWithOptimizedLineProcessor{
		logger:    logger,
		processor: processor,
	}
}

// ProcessStream processes an input stream and returns the length
func (sp *StreamProcessorWithOptimizedLineProcessor) ProcessStream(ctx context.Context, reader io.Reader, mode ports.StreamingMode) (int, error) {
	count, _, err := sp.processor.ProcessLines(ctx, reader, nil)
	return count, err
}

// ProcessStreamWithWriter processes an input stream and writes to the output writer
func (sp *StreamProcessorWithOptimizedLineProcessor) ProcessStreamWithWriter(ctx context.Context, reader io.Reader, writer io.Writer, mode ports.StreamingMode) (int, error) {
	count, _, err := sp.processor.ProcessLines(ctx, reader, writer)
	return count, err
}
