// The main issues in the stream processor need to be fixed:
// 1. Handling empty input text
// 2. Fixing the "token too long" error in the scanner
// 3. Better error handling for cancelled operations

// Fix for internal/adapters/stream/processor.go

package stream

import (
	"bufio"
	"context"
	"github.com/baditaflorin/go_length_similarity/internal/adapters/stream/wordprocessor"
	"io"
	"math"
	"time"

	"github.com/baditaflorin/go_length_similarity/internal/pool"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

const (
	// DefaultChunkSize defines the default size of each chunk when processing in chunks
	DefaultChunkSize = 8192 // 8KB

	// MaxScannerBufferSize defines the maximum buffer size for the scanner
	// This helps prevent "token too long" errors
	MaxScannerBufferSize = 1024 * 1024 // 1MB
)

// DefaultProcessor implements streaming processing for text input
type DefaultProcessor struct {
	logger      ports.Logger
	normalizer  ports.Normalizer
	bufferPool  *pool.BufferPool
	runePool    *pool.RuneBufferPool
	builderPool *pool.StringBuilderPool
	chunkSize   int

	// Word processor for optimized word-by-word processing
	wordProcessor *wordprocessor.Processor
}

// NewDefaultProcessor creates a new default stream processor
func NewDefaultProcessor(logger ports.Logger, normalizer ports.Normalizer) *DefaultProcessor {
	// Initialize the optimized word processor
	wordProc := wordprocessor.NewProcessor(logger, normalizer, wordprocessor.ProcessingConfig{
		ChunkSize:   DefaultChunkSize * 8, // Larger chunks for word processing
		BatchSize:   1000,                 // Process words in batches of 1000
		UseParallel: false,                // Disable parallel processing by default
	})

	return &DefaultProcessor{
		logger:        logger,
		normalizer:    normalizer,
		bufferPool:    pool.NewBufferPool(DefaultChunkSize),
		runePool:      pool.NewRuneBufferPool(DefaultChunkSize),
		builderPool:   pool.NewStringBuilderPool(),
		chunkSize:     DefaultChunkSize,
		wordProcessor: wordProc,
	}
}

// WithChunkSize sets a custom chunk size for the processor
func (p *DefaultProcessor) WithChunkSize(size int) *DefaultProcessor {
	p.chunkSize = size
	return p
}

// WithParallelWordProcessing enables parallel word processing
func (p *DefaultProcessor) WithParallelWordProcessing(enable bool) *DefaultProcessor {
	// Create a new word processor with parallel enabled/disabled
	p.wordProcessor = wordprocessor.NewProcessor(p.logger, p.normalizer, wordprocessor.ProcessingConfig{
		ChunkSize:   p.chunkSize * 8, // Larger chunks for word processing
		BatchSize:   1000,            // Process words in batches of 1000
		UseParallel: enable,          // Set parallel processing as requested
	})

	return p
}

func (p *DefaultProcessor) ProcessStream(ctx context.Context, reader io.Reader, mode ports.StreamingMode) (int, error) {
	startTime := time.Now()

	// Check if reader is nil
	if reader == nil {
		p.logger.Error("Nil reader provided")
		return 0, io.ErrUnexpectedEOF
	}

	var count int
	var bytesProcessed int64
	var err error

	switch mode {
	case ports.ChunkByChunk:
		count, bytesProcessed, err = p.processChunks(ctx, reader, nil)
	case ports.LineByLine:
		count, bytesProcessed, err = p.processLines(ctx, reader, nil)
	case ports.WordByWord:
		// Use optimized word processor
		count, bytesProcessed, err = p.wordProcessor.ProcessWords(ctx, reader, nil)
	}

	if err != nil && err != io.EOF {
		p.logger.Error("Stream processing error", "error", err, "mode", mode)
		return count, err
	}

	// Even if we processed zero bytes, don't return an error if it was just an empty stream
	if bytesProcessed == 0 && err == io.EOF {
		p.logger.Debug("Empty stream processed", "mode", mode)
		return 0, nil
	}

	p.logger.Debug("Stream processing completed",
		"mode", mode,
		"count", count,
		"bytes_processed", bytesProcessed,
		"duration", time.Since(startTime),
	)

	return count, nil
}

// ProcessStreamWithWriter processes an input stream, potentially transforms it, and writes to the output writer
func (p *DefaultProcessor) ProcessStreamWithWriter(ctx context.Context, reader io.Reader, writer io.Writer, mode ports.StreamingMode) (int, error) {
	startTime := time.Now()

	// Check if reader or writer is nil
	if reader == nil {
		p.logger.Error("Nil reader provided")
		return 0, io.ErrUnexpectedEOF
	}
	if writer == nil {
		p.logger.Error("Nil writer provided")
		return 0, io.ErrUnexpectedEOF
	}

	var count int
	var bytesProcessed int64
	var err error

	switch mode {
	case ports.ChunkByChunk:
		count, bytesProcessed, err = p.processChunks(ctx, reader, writer)
	case ports.LineByLine:
		count, bytesProcessed, err = p.processLines(ctx, reader, writer)
	case ports.WordByWord:
		// Use optimized word processor
		count, bytesProcessed, err = p.wordProcessor.ProcessWords(ctx, reader, writer)
	}

	if err != nil && err != io.EOF {
		p.logger.Error("Stream processing with writer error", "error", err, "mode", mode)
		return count, err
	}

	p.logger.Debug("Stream processing with writer completed",
		"mode", mode,
		"count", count,
		"bytes_processed", bytesProcessed,
		"duration", time.Since(startTime),
	)

	return count, nil
}

// processChunks processes the input in fixed-size chunks
func (p *DefaultProcessor) processChunks(ctx context.Context, reader io.Reader, writer io.Writer) (int, int64, error) {
	buffer := p.bufferPool.Get()
	defer p.bufferPool.Put(buffer)

	// Ensure buffer has sufficient capacity
	if cap(*buffer) < p.chunkSize {
		*buffer = make([]byte, p.chunkSize)
	} else {
		*buffer = (*buffer)[:p.chunkSize]
	}

	count := 0
	var totalBytes int64 = 0
	var lastErr error

	for {
		// Check context for cancellation
		select {
		case <-ctx.Done():
			p.logger.Warn("Processing cancelled by context", "error", ctx.Err())
			return count, totalBytes, ctx.Err()
		default:
			// Continue processing
		}

		// Reset buffer length but keep capacity
		*buffer = (*buffer)[:p.chunkSize]

		n, err := reader.Read(*buffer)
		*buffer = (*buffer)[:n]
		totalBytes += int64(n)

		if n > 0 {
			// Process chunk
			normalized := p.normalizer.Normalize(string(*buffer))
			count += len([]rune(normalized))

			// Write normalized output if writer is provided
			if writer != nil {
				_, werr := writer.Write([]byte(normalized))
				if werr != nil {
					p.logger.Error("Error writing to output", "error", werr)
					return count, totalBytes, werr
				}
			}
		}

		if err != nil {
			if err != io.EOF {
				p.logger.Warn("Error reading from input", "error", err)
				lastErr = err
			} else {
				lastErr = io.EOF
			}
			break
		}
	}

	return count, totalBytes, lastErr
}

// processLines processes the input line by line
func (p *DefaultProcessor) processLines(ctx context.Context, reader io.Reader, writer io.Writer) (int, int64, error) {
	scanner := bufio.NewScanner(reader)

	// Increase scanner buffer size to handle longer lines
	// This should fix the "token too long" error
	scannerBuffer := make([]byte, MaxScannerBufferSize)
	scanner.Buffer(scannerBuffer, MaxScannerBufferSize)

	count := 0
	var totalBytes int64 = 0

	for scanner.Scan() {
		// Check context for cancellation
		select {
		case <-ctx.Done():
			p.logger.Warn("Processing cancelled by context", "error", ctx.Err())
			return count, totalBytes, ctx.Err()
		default:
			// Continue processing
		}

		line := scanner.Text()
		lineLen := len(line)
		totalBytes += int64(lineLen + 1) // +1 for the newline

		// Process line
		normalized := p.normalizer.Normalize(line)
		count += len([]rune(normalized))

		// Write normalized output if writer is provided
		if writer != nil {
			_, err := writer.Write([]byte(normalized + "\n"))
			if err != nil {
				p.logger.Error("Error writing to output", "error", err)
				return count, totalBytes, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		p.logger.Warn("Error scanning input", "error", err)
		return count, totalBytes, err
	}

	return count, totalBytes, nil
}

// processWords processes the input word by word
func (p *DefaultProcessor) processWords(ctx context.Context, reader io.Reader, writer io.Writer) (int, int64, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanWords)

	// Increase scanner buffer to handle longer words
	scannerBuffer := make([]byte, MaxScannerBufferSize)
	scanner.Buffer(scannerBuffer, MaxScannerBufferSize)

	count := 0
	var totalBytes int64 = 0

	for scanner.Scan() {
		// Check context for cancellation
		select {
		case <-ctx.Done():
			p.logger.Warn("Processing cancelled by context", "error", ctx.Err())
			return count, totalBytes, ctx.Err()
		default:
			// Continue processing
		}

		word := scanner.Text()
		wordLen := len(word)
		totalBytes += int64(wordLen + 1) // +1 for the whitespace

		// Process word (count is just word count here)
		count++

		// Write normalized output if writer is provided
		if writer != nil {
			normalized := p.normalizer.Normalize(word)
			_, err := writer.Write([]byte(normalized + " "))
			if err != nil {
				p.logger.Error("Error writing to output", "error", err)
				return count, totalBytes, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		p.logger.Warn("Error scanning input", "error", err)
		return count, totalBytes, err
	}

	return count, totalBytes, nil
}

// StreamingCalculator extends the regular calculator with streaming capabilities
type StreamingCalculator struct {
	config     StreamingConfig
	logger     ports.Logger
	normalizer ports.Normalizer
	processor  *DefaultProcessor
}

// StreamingConfig holds configuration for streaming similarity calculation
type StreamingConfig struct {
	Threshold    float64
	MaxDiffRatio float64
	ChunkSize    int
	Mode         ports.StreamingMode
}

// NewStreamingCalculator creates a new streaming calculator
func NewStreamingCalculator(config StreamingConfig, logger ports.Logger, normalizer ports.Normalizer) (*StreamingCalculator, error) {
	processor := NewDefaultProcessor(logger, normalizer).WithChunkSize(config.ChunkSize)

	return &StreamingCalculator{
		config:     config,
		logger:     logger,
		normalizer: normalizer,
		processor:  processor,
	}, nil
}

// ComputeStreaming calculates the similarity between two text streams
func (sc *StreamingCalculator) ComputeStreaming(ctx context.Context, original io.Reader, augmented io.Reader) ports.StreamResult {
	startTime := time.Now()

	details := make(map[string]interface{})

	// Process original text stream
	origCount, err := sc.processor.ProcessStream(ctx, original, sc.config.Mode)
	if err != nil && err != io.EOF {
		sc.logger.Error("Error processing original stream", "error", err)
		details["error"] = "error processing original stream: " + err.Error()
		return ports.StreamResult{
			Name:           "streaming_similarity",
			Score:          0,
			Passed:         false,
			Details:        details,
			ProcessingTime: time.Since(startTime),
		}
	}

	// Process augmented text stream
	augCount, err := sc.processor.ProcessStream(ctx, augmented, sc.config.Mode)
	if err != nil && err != io.EOF {
		sc.logger.Error("Error processing augmented stream", "error", err)
		details["error"] = "error processing augmented stream: " + err.Error()
		return ports.StreamResult{
			Name:           "streaming_similarity",
			Score:          0,
			Passed:         false,
			Details:        details,
			ProcessingTime: time.Since(startTime),
		}
	}

	// Special case: if both texts are empty, consider them identical
	if origCount == 0 && augCount == 0 {
		sc.logger.Debug("Both texts are empty, considering them identical")
		details["note"] = "both texts are empty, considered identical"
		return ports.StreamResult{
			Name:            "streaming_similarity",
			Score:           1.0,
			Passed:          true,
			OriginalLength:  0,
			AugmentedLength: 0,
			LengthRatio:     1.0,
			Threshold:       sc.config.Threshold,
			Details:         details,
			ProcessingTime:  time.Since(startTime),
		}
	}

	// Handle case where original text is empty but augmented is not
	if origCount == 0 {
		sc.logger.Warn("Original text has zero length, considering maximum difference")
		details["warning"] = "original text has zero length"
		return ports.StreamResult{
			Name:            "streaming_similarity",
			Score:           0.0,
			Passed:          false,
			OriginalLength:  0,
			AugmentedLength: augCount,
			LengthRatio:     0.0,
			Threshold:       sc.config.Threshold,
			Details:         details,
			ProcessingTime:  time.Since(startTime),
		}
	}

	// Calculate similarity using the same algorithm as the non-streaming version
	var lengthRatio float64
	if origCount > augCount {
		lengthRatio = float64(augCount) / float64(origCount)
	} else {
		lengthRatio = float64(origCount) / float64(augCount)
	}

	diff := math.Abs(float64(origCount - augCount))
	diffRatio := diff / (float64(origCount) * sc.config.MaxDiffRatio)
	if diffRatio > 1.0 {
		diffRatio = 1.0
	}

	scaledScore := 1.0 - diffRatio
	passed := scaledScore >= sc.config.Threshold

	details["original_length"] = origCount
	details["augmented_length"] = augCount
	details["length_ratio"] = lengthRatio
	details["threshold"] = sc.config.Threshold
	details["mode"] = sc.config.Mode

	sc.logger.Debug("Computed streaming similarity",
		"score", scaledScore,
		"passed", passed,
		"details", details,
		"duration", time.Since(startTime),
	)

	return ports.StreamResult{
		Name:            "streaming_similarity",
		Score:           scaledScore,
		Passed:          passed,
		OriginalLength:  origCount,
		AugmentedLength: augCount,
		LengthRatio:     lengthRatio,
		Threshold:       sc.config.Threshold,
		Details:         details,
		ProcessingTime:  time.Since(startTime),
	}
}
