package wordprocessor

import (
	"context"
	"io"
	"time"

	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

// Constants for word processing
const (
	// DefaultChunkSize defines the default size of each chunk for reading
	DefaultChunkSize = 64 * 1024 // 64KB

	// DefaultBatchSize defines how many words to process in one batch
	DefaultBatchSize = 1000

	// ContextCheckFrequency defines how often to check for context cancellation
	ContextCheckFrequency = 5000 // words
)

// Processor implements optimized word processing
type Processor struct {
	logger     ports.Logger
	normalizer ports.Normalizer

	// Buffer pools
	wordBufferPool  *WordBufferPool
	chunkBufferPool *ChunkBufferPool
	batchBufferPool *WordBatchBufferPool

	// Configuration
	chunkSize   int
	batchSize   int
	useParallel bool
}

// ProcessingConfig defines configuration for word processing
type ProcessingConfig struct {
	ChunkSize   int
	BatchSize   int
	UseParallel bool
}

// NewProcessor creates a new optimized word processor
func NewProcessor(
	logger ports.Logger,
	normalizer ports.Normalizer,
	config ProcessingConfig,
) *Processor {
	// Use defaults if not specified
	if config.ChunkSize <= 0 {
		config.ChunkSize = DefaultChunkSize
	}
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultBatchSize
	}

	return &Processor{
		logger:          logger,
		normalizer:      normalizer,
		wordBufferPool:  NewWordBufferPool(),
		chunkBufferPool: NewChunkBufferPool(config.ChunkSize),
		batchBufferPool: NewWordBatchBufferPool(config.BatchSize),
		chunkSize:       config.ChunkSize,
		batchSize:       config.BatchSize,
		useParallel:     config.UseParallel,
	}
}

// ProcessWords processes a reader word by word and returns the word count
func (p *Processor) ProcessWords(
	ctx context.Context,
	reader io.Reader,
	writer io.Writer,
) (int, int64, error) {
	if p.useParallel {
		return p.processWordsParallel(ctx, reader, writer)
	}
	return p.processWordsOptimized(ctx, reader, writer)
}

// processWordsOptimized implements an optimized single-threaded word processing algorithm
func (p *Processor) processWordsOptimized(
	ctx context.Context,
	reader io.Reader,
	writer io.Writer,
) (int, int64, error) {
	startTime := time.Now()

	// Get buffers from pools
	chunkBuffer := p.chunkBufferPool.Get()
	defer p.chunkBufferPool.Put(chunkBuffer)

	// Count words and bytes
	wordCount := 0
	var bytesProcessed int64 = 0

	// Track word boundary information
	inWord := false
	wordStart := 0
	lastWordChar := false
	contextCheckCounter := 0

	// Loop until we're done or encounter an error
	for {
		// Periodically check context for cancellation
		contextCheckCounter++
		if contextCheckCounter >= ContextCheckFrequency {
			select {
			case <-ctx.Done():
				p.logger.Warn("Processing cancelled by context", "error", ctx.Err())
				return wordCount, bytesProcessed, ctx.Err()
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

			// Determine if we can use the fast ASCII path
			asciiOnly := IsASCIIOnly(chunk)

			// Process the chunk
			if asciiOnly {
				// Fast path for ASCII
				for i := 0; i < n; i++ {
					b := chunk[i]
					isChar := IsASCIIWordChar(b)

					if isChar {
						// Start of a word
						if !inWord {
							wordStart = i
							inWord = true
						}
					} else {
						// End of a word
						if inWord {
							// Found a complete word
							wordCount++

							// Write the word if needed
							if writer != nil {
								wb := p.wordBufferPool.Get()
								wb.Bytes = append(wb.Bytes, chunk[wordStart:i]...)
								normalized := p.normalizer.Normalize(string(wb.Bytes))
								writer.Write([]byte(normalized + " "))
								p.wordBufferPool.Put(wb)
							}

							inWord = false
						}
					}
				}

				// Update for the next chunk
				lastWordChar = n > 0 && IsASCIIWordChar(chunk[n-1])
			} else {
				// Slower path for non-ASCII
				i := 0
				for i < n {
					// Fix: Use blank identifier for unused variable
					_, size, isChar := HandleUTF8(chunk, i)

					if isChar {
						// Start of a word
						if !inWord {
							wordStart = i
							inWord = true
						}
					} else {
						// End of a word
						if inWord {
							// Found a complete word
							wordCount++

							// Write the word if needed
							if writer != nil {
								wb := p.wordBufferPool.Get()
								wb.Bytes = append(wb.Bytes, chunk[wordStart:i]...)
								normalized := p.normalizer.Normalize(string(wb.Bytes))
								writer.Write([]byte(normalized + " "))
								p.wordBufferPool.Put(wb)
							}

							inWord = false
						}
					}

					i += size
				}

				// Update for the next chunk
				if n > 0 {
					// Fix: Use blank identifier for unused variables
					_, _, isChar := HandleUTF8(chunk, n-1)
					lastWordChar = isChar
				}
			}

			// Handle word that spans chunks
			if inWord && !lastWordChar {
				// Word ended at chunk boundary
				wordCount++

				// Write the word if needed
				if writer != nil {
					wb := p.wordBufferPool.Get()
					wb.Bytes = append(wb.Bytes, chunk[wordStart:n]...)
					normalized := p.normalizer.Normalize(string(wb.Bytes))
					writer.Write([]byte(normalized + " "))
					p.wordBufferPool.Put(wb)
				}

				inWord = false
			}
		}

		// Handle errors or EOF
		if err != nil {
			if err != io.EOF {
				p.logger.Warn("Error reading from input", "error", err)
				return wordCount, bytesProcessed, err
			}

			// Handle final word if necessary
			if inWord {
				wordCount++
			}

			break
		}
	}

	// Final logging
	p.logger.Debug("Word processing completed",
		"word_count", wordCount,
		"bytes_processed", bytesProcessed,
		"duration", time.Since(startTime),
	)

	return wordCount, bytesProcessed, nil
}

// ProcessResult holds the result of a parallel word processing operation
type ProcessResult struct {
	WordCount int
	Error     error
}

// WordProcessor defines a function type for processing words
type WordProcessor func([]byte, int, int) (int, error)
