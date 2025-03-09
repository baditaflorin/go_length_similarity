// File: internal/adapters/stream/lineprocessor/optimized_processor.go
package lineprocessor

import (
	"context"
	"io"
	"time"

	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

// LineRange represents a line's location in a buffer without copying the line
type LineRange struct {
	Start, End int
	Buffer     *ChunkBuffer // Reference to the chunk buffer containing this line
}

// Constants for line processing
const (
	// DefaultChunkSize defines the default size of each chunk for reading
	DefaultChunkSize = 64 * 1024 // 64KB

	// DefaultBatchSize defines how many lines to process in one batch
	DefaultBatchSize = 100

	// ContextCheckFrequency defines how often to check for context cancellation
	ContextCheckFrequency = 500 // lines

	// Common newline characters
	CR = '\r'
	LF = '\n'
)

// OptimizedProcessor implements allocation-efficient line processing
type OptimizedProcessor struct {
	logger     ports.Logger
	normalizer ports.Normalizer

	// Buffer pools
	lineBufferPool    *LineBufferPool
	chunkBufferPool   *ChunkBufferPool
	lineRangePool     *LineRangePool
	stringBuilderPool *StringBuilderPool

	// Configuration
	chunkSize   int
	batchSize   int
	useParallel bool
}

// ProcessingConfig defines configuration for line processing
type ProcessingConfig struct {
	ChunkSize   int
	BatchSize   int
	UseParallel bool
}

// NewOptimizedProcessor creates a new optimized line processor
func NewOptimizedProcessor(
	logger ports.Logger,
	normalizer ports.Normalizer,
	config ProcessingConfig,
) *OptimizedProcessor {
	// Use defaults if not specified
	if config.ChunkSize <= 0 {
		config.ChunkSize = DefaultChunkSize
	}
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultBatchSize
	}

	return &OptimizedProcessor{
		logger:            logger,
		normalizer:        normalizer,
		lineBufferPool:    NewLineBufferPool(),
		chunkBufferPool:   NewChunkBufferPool(config.ChunkSize),
		lineRangePool:     NewLineRangePool(config.BatchSize * 2), // Double capacity to avoid reallocations
		stringBuilderPool: NewStringBuilderPool(),
		chunkSize:         config.ChunkSize,
		batchSize:         config.BatchSize,
		useParallel:       config.UseParallel,
	}
}

// ProcessLines processes a reader line by line and returns the character count
func (p *OptimizedProcessor) ProcessLines(
	ctx context.Context,
	reader io.Reader,
	writer io.Writer,
) (int, int64, error) {
	if p.useParallel {
		return p.processLinesParallel(ctx, reader, writer)
	}
	return p.processLinesOptimized(ctx, reader, writer)
}

// processLinesOptimized implements an allocation-efficient line processing algorithm
func (p *OptimizedProcessor) processLinesOptimized(
	ctx context.Context,
	reader io.Reader,
	writer io.Writer,
) (int, int64, error) {
	startTime := time.Now()

	// Get a chunk buffer from the pool
	chunkBuffer := p.chunkBufferPool.Get()
	defer p.chunkBufferPool.Put(chunkBuffer)

	// Get a line ranges slice from the pool
	lineRanges := p.lineRangePool.Get()
	defer p.lineRangePool.Put(lineRanges)

	// Get a string builder for normalization
	sb := p.stringBuilderPool.Get()
	defer p.stringBuilderPool.Put(sb)

	// Count characters (runes) and bytes
	charCount := 0
	var bytesProcessed int64 = 0

	// Track the previous chunk's unused data
	var partialLine []byte
	contextCheckCounter := 0

	// Loop until we're done or encounter an error
	for {
		// Periodically check context for cancellation
		contextCheckCounter++
		if contextCheckCounter >= ContextCheckFrequency {
			select {
			case <-ctx.Done():
				p.logger.Warn("Processing cancelled by context", "error", ctx.Err())
				return charCount, bytesProcessed, ctx.Err()
			default:
				// Continue processing
			}
			contextCheckCounter = 0
		}

		// Read a chunk
		n, err := reader.Read(chunkBuffer.Bytes)
		if n > 0 {
			bytesProcessed += int64(n)
			chunk := chunkBuffer.Bytes[:n]

			// Process the chunk to find line boundaries (without copying each line)
			lineCount := 0
			lineRanges.Reset() // Clear previous line ranges without reallocating

			// If we have a partial line from the previous chunk, handle it
			if len(partialLine) > 0 {
				// Find the first newline in the current chunk
				newlineIndex := -1
				for i := 0; i < n; i++ {
					if chunk[i] == LF {
						newlineIndex = i
						break
					}
				}

				if newlineIndex >= 0 {
					// We found a newline - complete the partial line
					completeLine := make([]byte, len(partialLine)+newlineIndex)
					copy(completeLine, partialLine)
					copy(completeLine[len(partialLine):], chunk[:newlineIndex])

					// Process this complete line
					line := string(completeLine)
					normalized := p.normalizer.Normalize(line)
					charCount += len([]rune(normalized))

					if writer != nil {
						writer.Write([]byte(normalized + "\n"))
					}

					// Start processing the rest of the chunk
					lineCount = p.findLineRanges(chunk[newlineIndex+1:], lineRanges, newlineIndex+1)
					partialLine = nil
				} else {
					// No newline - the entire chunk is part of the partial line
					newPartialLine := make([]byte, len(partialLine)+n)
					copy(newPartialLine, partialLine)
					copy(newPartialLine[len(partialLine):], chunk)
					partialLine = newPartialLine
				}
			} else {
				// No partial line, process the whole chunk
				lineCount = p.findLineRanges(chunk, lineRanges, 0)
			}

			// Process the lines in the current chunk
			for i := 0; i < lineCount; i++ {
				lr := lineRanges.Get(i)

				// Check if this is the last line and doesn't end with a newline
				isPartialLine := i == lineCount-1 &&
					(lr.End == len(chunk) || chunk[lr.End-1] != LF)

				if isPartialLine && err == nil {
					// This is a partial line - save it for the next chunk
					partialLine = make([]byte, lr.End-lr.Start)
					copy(partialLine, chunk[lr.Start:lr.End])
					continue
				}

				// Process a complete line
				line := string(chunk[lr.Start:lr.End])
				normalized := p.normalizer.Normalize(line)
				charCount += len([]rune(normalized))

				if writer != nil {
					writer.Write([]byte(normalized + "\n"))
				}
			}
		}

		// Handle errors or EOF
		if err != nil {
			if err != io.EOF {
				p.logger.Warn("Error reading from input", "error", err)
				return charCount, bytesProcessed, err
			}

			// Handle final line if there's a partial line
			if len(partialLine) > 0 {
				line := string(partialLine)
				normalized := p.normalizer.Normalize(line)
				charCount += len([]rune(normalized))

				if writer != nil {
					writer.Write([]byte(normalized + "\n"))
				}
			}

			break
		}
	}

	// Final logging
	p.logger.Debug("Line processing completed",
		"char_count", charCount,
		"bytes_processed", bytesProcessed,
		"duration", time.Since(startTime),
	)

	return charCount, bytesProcessed, nil
}

// findLineRanges locates line boundaries in a byte slice without copying each line
// Returns the number of lines found
func (p *OptimizedProcessor) findLineRanges(data []byte, ranges *LineRanges, offset int) int {
	lineCount := 0
	lineStart := 0

	for i := 0; i < len(data); i++ {
		if data[i] == LF {
			// Found a line boundary
			ranges.Add(lineStart+offset, i+1+offset)
			lineStart = i + 1
			lineCount++
		}
	}

	// Add the last line if it doesn't end with a newline
	if lineStart < len(data) {
		ranges.Add(lineStart+offset, len(data)+offset)
		lineCount++
	}

	return lineCount
}

// BatchProcessLines processes multiple lines at once to reduce normalization overhead
func (p *OptimizedProcessor) BatchProcessLines(
	chunk []byte,
	lineRanges *LineRanges,
	startIndex, endIndex int,
	sb *StringBuilder,
) int {
	if startIndex >= endIndex {
		return 0
	}

	// Reset the string builder
	sb.Reset()

	// Concatenate all lines in the batch
	for i := startIndex; i < endIndex; i++ {
		lr := lineRanges.Get(i)
		sb.WriteString(string(chunk[lr.Start:lr.End]))
		sb.WriteRune('\n')
	}

	// Normalize the entire batch at once
	normalized := p.normalizer.Normalize(sb.String())

	// Count characters in the normalized text
	return len([]rune(normalized))
}
