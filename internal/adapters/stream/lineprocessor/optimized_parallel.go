// File: internal/adapters/stream/lineprocessor/optimized_parallel.go
package lineprocessor

import (
	"context"
	"io"
	"runtime"
	"sync"
	"time"
)

// Constants for parallel processing
const (
	// Default number of workers - use 0 to automatically use runtime.NumCPU()
	DefaultWorkers = 0

	// Maximum job queue size
	MaxJobQueueSize = 16

	// Minimum batch size for efficient parallelization
	MinBatchSize = 8
)

// LineJob represents a batch of lines to be processed by a worker
type LineJob struct {
	ChunkBuffer *ChunkBuffer // The buffer containing the chunk data
	Ranges      *LineRanges  // The line ranges in this job
	ChunkID     int          // ID for ordering results
	IsFinal     bool         // Whether this is the last job
}

// LineJobResult represents the result of processing a line batch
type LineJobResult struct {
	CharCount int
	ChunkID   int
	Error     error
}

// processLinesParallel implements parallel line processing with reduced allocations
func (p *OptimizedProcessor) processLinesParallel(
	ctx context.Context,
	reader io.Reader,
	writer io.Writer,
) (int, int64, error) {
	startTime := time.Now()

	// Determine number of workers
	workers := runtime.NumCPU()
	if workers > 8 {
		// Limit to 8 workers to avoid excessive overhead
		workers = 8
	}

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
		var partialLine []byte

		// Use a pool of chunk buffers for reading
		chunkBuffers := make([]*ChunkBuffer, MaxJobQueueSize)
		lineRangesPool := make([]*LineRanges, MaxJobQueueSize)

		for i := 0; i < MaxJobQueueSize; i++ {
			chunkBuffers[i] = p.chunkBufferPool.Get()
			lineRangesPool[i] = p.lineRangePool.Get()
		}

		// Function to clean up resources
		defer func() {
			for i := 0; i < MaxJobQueueSize; i++ {
				if chunkBuffers[i] != nil {
					p.chunkBufferPool.Put(chunkBuffers[i])
				}
				if lineRangesPool[i] != nil {
					p.lineRangePool.Put(lineRangesPool[i])
				}
			}
		}()

		for {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				// Continue
			}

			// Use the current chunk buffer
			bufferIndex := chunkID % MaxJobQueueSize
			chunkBuffer := chunkBuffers[bufferIndex]
			lineRanges := lineRangesPool[bufferIndex]

			// Reset the line ranges
			lineRanges.Reset()

			// Read a chunk
			n, err := reader.Read(chunkBuffer.Bytes)

			if n > 0 {
				bytesProcessed += int64(n)
				chunk := chunkBuffer.Bytes[:n]

				// Process chunk to find line boundaries
				lineCount := 0

				// If we have a partial line from the previous chunk, handle it
				if len(partialLine) > 0 {
					// Find the first newline in this chunk
					newlineIdx := -1
					for i := 0; i < n; i++ {
						if chunk[i] == LF {
							newlineIdx = i
							break
						}
					}

					if newlineIdx >= 0 {
						// Complete the partial line
						completeLine := make([]byte, len(partialLine)+newlineIdx)
						copy(completeLine, partialLine)
						copy(completeLine[len(partialLine):], chunk[:newlineIdx])

						// Send this as a special single-line job
						singleLineJob := LineJob{
							ChunkBuffer: &ChunkBuffer{Bytes: completeLine},
							Ranges: &LineRanges{
								Ranges: []struct{ Start, End int }{{0, len(completeLine)}},
								Count:  1,
							},
							ChunkID: chunkID,
							IsFinal: false,
						}

						select {
						case jobs <- singleLineJob:
							// Job sent successfully
							chunkID++
						case <-ctx.Done():
							errChan <- ctx.Err()
							return
						}

						// Process the rest of the chunk for line boundaries
						lineCount = p.findLineRanges(chunk[newlineIdx+1:], lineRanges, newlineIdx+1)
						partialLine = nil
					} else {
						// No newline found - the entire chunk is part of the partial line
						newPartial := make([]byte, len(partialLine)+n)
						copy(newPartial, partialLine)
						copy(newPartial[len(partialLine):], chunk)
						partialLine = newPartial

						// Skip sending a job for this chunk
						continue
					}
				} else {
					// No partial line, process the whole chunk
					lineCount = p.findLineRanges(chunk, lineRanges, 0)
				}

				// Check if the last line is complete (ends with newline)
				if lineCount > 0 {
					lastLine := lineRanges.Get(lineCount - 1)
					isPartial := lastLine.End == n || chunk[lastLine.End-1] != LF

					if isPartial && err == nil {
						// Extract the partial line for the next chunk
						partialLine = make([]byte, lastLine.End-lastLine.Start)
						copy(partialLine, chunk[lastLine.Start:lastLine.End])

						// Exclude the partial line from processing
						lineRanges.Count--
						lineCount--
					}
				}

				// Only send a job if we have lines to process
				if lineCount > 0 {
					// Create a job for this chunk and send it to workers
					job := LineJob{
						ChunkBuffer: chunkBuffer,
						Ranges:      lineRanges,
						ChunkID:     chunkID,
						IsFinal:     false,
					}

					select {
					case jobs <- job:
						// Job sent successfully

						// Allocate a new buffer and line ranges for the next chunk
						// since we've sent the current ones to a worker
						chunkBuffers[bufferIndex] = p.chunkBufferPool.Get()
						lineRangesPool[bufferIndex] = p.lineRangePool.Get()
					case <-ctx.Done():
						errChan <- ctx.Err()
						return
					}

					chunkID++
				}
			}

			// Handle end of stream or errors
			if err != nil {
				// Process final partial line if it exists
				if len(partialLine) > 0 {
					finalLineJob := LineJob{
						ChunkBuffer: &ChunkBuffer{Bytes: partialLine},
						Ranges: &LineRanges{
							Ranges: []struct{ Start, End int }{{0, len(partialLine)}},
							Count:  1,
						},
						ChunkID: chunkID,
						IsFinal: true,
					}

					select {
					case jobs <- finalLineJob:
						// Final job sent successfully
					case <-ctx.Done():
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
func (p *OptimizedProcessor) lineWorker(
	ctx context.Context,
	id int,
	jobs <-chan LineJob,
	results chan<- LineJobResult,
	wg *sync.WaitGroup,
	writer io.Writer,
) {
	defer wg.Done()

	// Get a string builder for normalization
	sb := p.stringBuilderPool.Get()
	defer p.stringBuilderPool.Put(sb)

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

		// Get chunk data and line ranges
		chunk := job.ChunkBuffer.Bytes
		lineRanges := job.Ranges

		// Process each line in the batch
		for i := 0; i < lineRanges.Count; i++ {
			lr := lineRanges.Get(i)

			// Get the line text
			line := string(chunk[lr.Start:lr.End])

			// Normalize the line
			normalized := p.normalizer.Normalize(line)
			charCount += len([]rune(normalized))

			// Write normalized output if writer is provided
			if writer != nil {
				// For parallel writer support, we would need synchronization
				// This is simplified and would need additional sync mechanisms
				writer.Write([]byte(normalized + "\n"))
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
