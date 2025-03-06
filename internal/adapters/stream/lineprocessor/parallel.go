package lineprocessor

import (
	"bytes"
	"context"
	"io"
	"runtime"
	"sync"
	"time"
)

// Constants for parallel processing
const (
	// DefaultWorkers is the default number of worker goroutines
	DefaultWorkers = 0 // 0 means use runtime.NumCPU()

	// MaxJobQueueSize limits the number of pending jobs
	MaxJobQueueSize = 32

	// MinLinesPerBatch is the minimum number of lines that should be processed in a batch
	MinLinesPerBatch = 10
)

// LineJob represents a chunk of text to be processed by a worker
type LineJob struct {
	// Lines in this job
	Lines   [][]byte
	ChunkID int
	IsFinal bool
}

// LineJobResult represents the result of processing a chunk
type LineJobResult struct {
	CharCount int
	ChunkID   int
	Error     error
}

// processLinesParallel implements parallel line processing using worker pools
func (p *Processor) processLinesParallel(
	ctx context.Context,
	reader io.Reader,
	writer io.Writer,
) (int, int64, error) {
	startTime := time.Now()

	// Determine number of workers
	workers := runtime.NumCPU()

	// Create channels for job distribution and result collection
	jobs := make(chan LineJob, MaxJobQueueSize)
	results := make(chan LineJobResult, workers)

	// Create a wait group to track worker completion
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go p.lineWorker(ctx, i, jobs, results, &wg, writer)
	}

	// Create a goroutine to close the results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Create a goroutine to read and split into lines
	errChan := make(chan error, 1)
	bytesProcessedChan := make(chan int64, 1)

	go func() {
		var chunkID int
		var bytesProcessed int64
		var lineBuffers [][]byte
		var pendingLines [][]byte

		// Create a buffered reader
		chunkBuffer := p.chunkBufferPool.Get()
		defer p.chunkBufferPool.Put(chunkBuffer)

		// Buffer to hold partial lines between chunks
		partialLine := p.lineBufferPool.Get()
		defer p.lineBufferPool.Put(partialLine)

		// Utility function to send a batch of lines to workers
		sendBatch := func(final bool) error {
			// Only send if we have lines or it's the final batch
			if len(pendingLines) > 0 || final {
				// Convert to immutable copy that can be safely sent
				lineBatch := make([][]byte, len(pendingLines))
				for i, line := range pendingLines {
					lineCopy := make([]byte, len(line))
					copy(lineCopy, line)
					lineBatch[i] = lineCopy
				}

				// Send to a worker
				select {
				case jobs <- LineJob{
					Lines:   lineBatch,
					ChunkID: chunkID,
					IsFinal: final,
				}:
					// Job sent successfully
					chunkID++
					pendingLines = pendingLines[:0] // Clear pending lines
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		}

		for {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				// Continue
			}

			// Read a chunk
			n, err := reader.Read(chunkBuffer.Bytes)

			if n > 0 {
				bytesProcessed += int64(n)
				chunk := chunkBuffer.Bytes[:n]

				// If we have a partial line from the previous chunk, append this chunk to it
				if len(partialLine.Bytes) > 0 {
					// Find the first newline in this chunk
					nlIndex := -1
					for i := 0; i < n; i++ {
						if chunk[i] == '\n' {
							nlIndex = i
							break
						}
					}

					if nlIndex >= 0 {
						// Complete the partial line
						partialLine.Bytes = append(partialLine.Bytes, chunk[:nlIndex]...)
						pendingLines = append(pendingLines, partialLine.Bytes)
						partialLine.Bytes = partialLine.Bytes[:0] // Reset

						// Process the rest of the chunk
						remainingChunk := chunk[nlIndex+1:]
						if len(remainingChunk) > 0 {
							// Split the remaining chunk into lines
							lineBuffers = bytes.Split(remainingChunk, []byte{'\n'})

							// Add complete lines to pending batch
							if len(lineBuffers) > 0 {
								// If the last line doesn't end with a newline, it's incomplete
								if err == nil || err != io.EOF {
									partialLine.Bytes = append(partialLine.Bytes, lineBuffers[len(lineBuffers)-1]...)
									lineBuffers = lineBuffers[:len(lineBuffers)-1]
								}

								// Add complete lines to pending
								pendingLines = append(pendingLines, lineBuffers...)
							}
						}
					} else {
						// No newline in this chunk, just append to partial line
						partialLine.Bytes = append(partialLine.Bytes, chunk...)
					}
				} else {
					// No partial line, process the whole chunk
					lineBuffers = bytes.Split(chunk, []byte{'\n'})

					// If the last line doesn't end with a newline, it's incomplete
					if len(lineBuffers) > 0 && (err == nil || err != io.EOF) {
						partialLine.Bytes = append(partialLine.Bytes, lineBuffers[len(lineBuffers)-1]...)
						lineBuffers = lineBuffers[:len(lineBuffers)-1]
					}

					// Add complete lines to pending
					pendingLines = append(pendingLines, lineBuffers...)
				}

				// Send a batch if we have enough lines
				if len(pendingLines) >= p.batchSize {
					if err := sendBatch(false); err != nil {
						errChan <- err
						return
					}
				}
			}

			// Handle end of stream or errors
			if err != nil {
				// Add any remaining partial line as a complete line
				if len(partialLine.Bytes) > 0 {
					pendingLines = append(pendingLines, partialLine.Bytes)
					partialLine.Bytes = partialLine.Bytes[:0]
				}

				// Send any remaining lines
				if err := sendBatch(true); err != nil {
					errChan <- err
					return
				}

				// Close the jobs channel to signal no more jobs
				close(jobs)

				if err != io.EOF {
					errChan <- err
				} else {
					errChan <- nil // Normal EOF
				}

				bytesProcessedChan <- bytesProcessed
				return
			}
		}
	}()

	// Collect and process results
	charCount := 0
	resultMap := make(map[int]LineJobResult)
	nextChunkID := 0

	// Wait for all results and order them by chunk ID
	for result := range results {
		resultMap[result.ChunkID] = result

		// Process results in order
		for {
			result, exists := resultMap[nextChunkID]
			if !exists {
				break
			}

			// Check for errors
			if result.Error != nil {
				return charCount, <-bytesProcessedChan, result.Error
			}

			// Add to character count
			charCount += result.CharCount

			// Remove processed result and move to next
			delete(resultMap, nextChunkID)
			nextChunkID++
		}
	}

	// Get the final error (if any) and bytes processed
	var err error
	select {
	case err = <-errChan:
		// Error retrieved
	default:
		// No error
	}

	bytesProcessed := <-bytesProcessedChan

	// Log completion
	p.logger.Debug("Parallel line processing completed",
		"char_count", charCount,
		"bytes_processed", bytesProcessed,
		"workers", workers,
		"duration", time.Since(startTime),
	)

	return charCount, bytesProcessed, err
}

// lineWorker is a worker goroutine that processes lines in parallel
func (p *Processor) lineWorker(
	ctx context.Context,
	id int,
	jobs <-chan LineJob,
	results chan<- LineJobResult,
	wg *sync.WaitGroup,
	writer io.Writer,
) {
	defer wg.Done()

	// Get a buffer from the pool
	lineBuffer := p.lineBufferPool.Get()
	defer p.lineBufferPool.Put(lineBuffer)

	// A mutex for safe writing if we're writing to an output
	var writerMutex sync.Mutex

	// Process jobs until the channel is closed
	for job := range jobs {
		// Check context for cancellation
		select {
		case <-ctx.Done():
			results <- LineJobResult{
				ChunkID: job.ChunkID,
				Error:   ctx.Err(),
			}
			return
		default:
			// Continue processing
		}

		// Process the lines in this job
		charCount := 0

		for _, line := range job.Lines {
			// Skip empty lines
			if len(line) == 0 {
				continue
			}

			// Normalize the line
			normalized := p.normalizer.Normalize(string(line))
			charCount += len([]rune(normalized))

			// Write normalized output if writer is provided
			if writer != nil {
				writerMutex.Lock()
				writer.Write([]byte(normalized + "\n"))
				writerMutex.Unlock()
			}
		}

		// Send the result
		results <- LineJobResult{
			CharCount: charCount,
			ChunkID:   job.ChunkID,
			Error:     nil,
		}
	}
}
