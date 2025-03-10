package lineprocessor

import (
	"bytes"
	"context"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
	"io"
	"sync"
	"time"
)

// import (
//
//	"bytes"
//	"context"
//	"io"
//	"time"
//
//	"github.com/baditaflorin/go_length_similarity/internal/ports"
//
// )
//
// // Constants for line processing
// const (
//
//	// DefaultChunkSize defines the default size of each chunk for reading
//	DefaultChunkSize = 64 * 1024 // 64KB
//
//	// DefaultBatchSize defines how many lines to process in one batch
//	DefaultBatchSize = 100
//
//	// ContextCheckFrequency defines how often to check for context cancellation
//	ContextCheckFrequency = 500 // lines
//
//	// Common newline characters
//	CR = '\r'
//	LF = '\n'
//
// )
//
// Processor implements optimized line processing
type Processor struct {
	logger     ports.Logger
	normalizer ports.Normalizer

	// Buffer pools
	lineBufferPool  *LineBufferPool
	chunkBufferPool *ChunkBufferPool
	batchBufferPool *LineBatchBufferPool

	// Configuration
	chunkSize   int
	batchSize   int
	useParallel bool
}

// // ProcessingConfig defines configuration for line processing
//
//	type ProcessingConfig struct {
//		ChunkSize   int
//		BatchSize   int
//		UseParallel bool
//	}
//
// NewProcessor creates a new optimized line processor
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
		lineBufferPool:  NewLineBufferPool(),
		chunkBufferPool: NewChunkBufferPool(config.ChunkSize),
		batchBufferPool: NewLineBatchBufferPool(config.BatchSize),
		chunkSize:       config.ChunkSize,
		batchSize:       config.BatchSize,
		useParallel:     config.UseParallel,
	}
}

// ProcessLines processes a reader line by line and returns the character count
func (p *Processor) ProcessLines(
	ctx context.Context,
	reader io.Reader,
	writer io.Writer,
) (int, int64, error) {
	if p.useParallel {
		return p.processLinesParallel(ctx, reader, writer)
	}
	return p.processLinesOptimized(ctx, reader, writer)
}

// processLinesParallel implements a parallel line processing algorithm
func (p *Processor) processLinesParallel(
	ctx context.Context,
	reader io.Reader,
	writer io.Writer,
) (int, int64, error) {
	startTime := time.Now()

	// Define the number of workers for parallel processing
	numWorkers := 4 // This could be made configurable or based on runtime.NumCPU()

	// Create channels for communication between workers
	jobs := make(chan []byte, p.batchSize)
	results := make(chan int, numWorkers)
	errChan := make(chan error, 1)
	doneChan := make(chan struct{})

	// Variable to track total bytes processed
	var bytesProcessed int64

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for line := range jobs {
				// Check for context cancellation
				select {
				case <-ctx.Done():
					return
				default:
					// Process the line
					normalized := p.normalizer.Normalize(string(line))
					charCount := len([]rune(normalized))

					// Send result back
					results <- charCount

					// Write normalized output if writer is provided
					if writer != nil {
						writer.Write([]byte(normalized + "\n"))
					}
				}
			}
		}()
	}

	// Close the results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
		close(doneChan)
	}()

	// Start a goroutine to read lines and send them to workers
	go func() {
		chunkBuffer := p.chunkBufferPool.Get()
		defer p.chunkBufferPool.Put(chunkBuffer)

		var partialLine []byte

		for {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				close(jobs)
				errChan <- ctx.Err()
				return
			default:
				// Continue reading
			}

			// Read a chunk
			n, err := reader.Read(chunkBuffer.Bytes)
			if n > 0 {
				bytesProcessed += int64(n)
				chunk := chunkBuffer.Bytes[:n]

				// Process the chunk to find lines
				var lines [][]byte

				// If we have a partial line from the previous chunk, handle it
				if len(partialLine) > 0 {
					// Find the first newline in this chunk
					newlineIdx := bytes.IndexByte(chunk, LF)
					if newlineIdx >= 0 {
						// Complete the partial line
						completeLine := make([]byte, len(partialLine)+newlineIdx)
						copy(completeLine, partialLine)
						copy(completeLine[len(partialLine):], chunk[:newlineIdx])

						// Send this line to be processed
						select {
						case jobs <- completeLine:
							// Continue
						case <-ctx.Done():
							close(jobs)
							errChan <- ctx.Err()
							return
						}

						// Process the rest of the chunk
						lines = bytes.Split(chunk[newlineIdx+1:], []byte{LF})
						partialLine = nil
					} else {
						// No newline found - the entire chunk is part of the partial line
						newPartial := make([]byte, len(partialLine)+n)
						copy(newPartial, partialLine)
						copy(newPartial[len(partialLine):], chunk)
						partialLine = newPartial
						continue
					}
				} else {
					// No partial line, process the whole chunk
					lines = bytes.Split(chunk, []byte{LF})
				}

				// Process complete lines
				if len(lines) > 0 {
					// Check if the last line is complete (ends with newline)
					lastLine := lines[len(lines)-1]
					if n > 0 && chunk[n-1] != LF {
						// Last line is incomplete, save it for the next chunk
						partialLine = lastLine
						lines = lines[:len(lines)-1]
					}

					// Send complete lines to workers
					for _, line := range lines {
						if len(line) > 0 {
							select {
							case jobs <- line:
								// Continue
							case <-ctx.Done():
								close(jobs)
								errChan <- ctx.Err()
								return
							}
						}
					}
				}
			}

			// Handle errors or EOF
			if err != nil {
				// Process any remaining partial line
				if len(partialLine) > 0 {
					select {
					case jobs <- partialLine:
						// Sent successfully
					case <-ctx.Done():
						close(jobs)
						errChan <- ctx.Err()
						return
					}
				}

				// Close the jobs channel to signal no more jobs
				close(jobs)

				if err != io.EOF {
					errChan <- err
				} else {
					errChan <- nil // Normal EOF
				}
				return
			}
		}
	}()

	// Collect results from workers
	charCount := 0
	var processingErr error

	// Process results
	resultsDone := false
	for !resultsDone {
		select {
		case count, ok := <-results:
			if !ok {
				resultsDone = true
			} else {
				charCount += count
			}
		case err := <-errChan:
			if err != nil && err != io.EOF {
				processingErr = err
			}
		case <-ctx.Done():
			return charCount, bytesProcessed, ctx.Err()
		}
	}

	// Wait for processing to complete
	<-doneChan

	// Log completion
	p.logger.Debug("Parallel line processing completed",
		"char_count", charCount,
		"bytes_processed", bytesProcessed,
		"duration", time.Since(startTime),
	)

	return charCount, bytesProcessed, processingErr
}

// processLinesOptimized implements an optimized single-threaded line processing algorithm
func (p *Processor) processLinesOptimized(
	ctx context.Context,
	reader io.Reader,
	writer io.Writer,
) (int, int64, error) {
	startTime := time.Now()

	// Get buffers from pools
	chunkBuffer := p.chunkBufferPool.Get()
	defer p.chunkBufferPool.Put(chunkBuffer)

	lineBuffer := p.lineBufferPool.Get()
	defer p.lineBufferPool.Put(lineBuffer)

	// Count characters (runes) and bytes
	charCount := 0
	var bytesProcessed int64 = 0

	// Line tracking state
	inLine := false
	carryoverCR := false
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

			// Process the chunk line by line
			var lineStart int = 0

			for i := 0; i < n; i++ {
				b := chunk[i]

				// Handle different newline sequences:
				// LF: \n (Unix)
				// CRLF: \r\n (Windows)
				// CR: \r (Old Mac)
				if b == LF || b == CR {
					// We found a line break

					// For Windows-style CRLF, we need to handle the sequence as one line break
					// If we see a CR, we set a flag and continue to the next byte
					if b == CR {
						// Extract the line so far (excluding CR)
						line := chunk[lineStart:i]

						// Check for CRLF sequence
						if i+1 < n && chunk[i+1] == LF {
							// This is a CRLF sequence, skip the CR and let the LF handling process it
							carryoverCR = true
							continue
						}

						// Process the line that ends with a CR
						p.processLine(line, writer, &charCount)
						lineStart = i + 1
						carryoverCR = false
					} else if b == LF {
						// Handle LF (Unix style) or second part of CRLF (Windows style)
						var line []byte

						if carryoverCR {
							// This LF is part of a CRLF sequence, the CR was already handled
							carryoverCR = false
							line = chunk[lineStart:i]
						} else {
							// This is a standalone LF
							line = chunk[lineStart:i]
						}

						// Process the line
						p.processLine(line, writer, &charCount)
						lineStart = i + 1
					}
				}
			}

			// Handle any partial line at the end of the chunk
			if lineStart < n {
				// We have a partial line, store it for the next chunk
				lineBuffer.Bytes = append(lineBuffer.Bytes, chunk[lineStart:]...)
				inLine = true
			}
		}

		// Handle errors or EOF
		if err != nil {
			if err != io.EOF {
				p.logger.Warn("Error reading from input", "error", err)
				return charCount, bytesProcessed, err
			}

			// Handle final line if there's buffered data
			if inLine && len(lineBuffer.Bytes) > 0 {
				p.processLine(lineBuffer.Bytes, writer, &charCount)
				lineBuffer.Bytes = lineBuffer.Bytes[:0]
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

// processLine handles a single line of text
func (p *Processor) processLine(line []byte, writer io.Writer, charCount *int) {
	if len(line) == 0 {
		return
	}

	// Normalize the line
	normalized := p.normalizer.Normalize(string(line))

	// Count characters (runes)
	*charCount += len([]rune(normalized))

	// Write normalized output if writer is provided
	if writer != nil {
		writer.Write([]byte(normalized + "\n"))
	}
}

// findLines locates lines in a byte slice and adds them to the provided buffer
// Returns the number of complete lines found
func (p *Processor) findLines(data []byte, lines *[][]byte) int {
	lineCount := 0

	// Find lines using bytes.Split which is more efficient than Scanner
	// for pre-loaded data
	*lines = bytes.Split(data, []byte{'\n'})
	lineCount = len(*lines)

	// If the data ended with a newline, the last element will be empty
	// We want to exclude it from the count
	if lineCount > 0 && len((*lines)[lineCount-1]) == 0 {
		*lines = (*lines)[:lineCount-1]
		lineCount--
	}

	return lineCount
}

// ProcessResult holds the result of a parallel line processing operation
type ProcessResult struct {
	CharCount int
	Error     error
}
