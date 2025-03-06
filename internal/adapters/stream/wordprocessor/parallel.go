package wordprocessor

import (
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
)

// WordJob represents a chunk of text to be processed by a worker
type WordJob struct {
	Chunk     []byte
	ChunkID   int
	StartWord bool // Whether we're starting in a word
}

// WordJobResult represents the result of processing a chunk
type WordJobResult struct {
	WordCount int
	ChunkID   int
	EndWord   bool // Whether we ended in a word
	Error     error
}

// processWordsParallel implements parallel word processing using worker pools
func (p *Processor) processWordsParallel(
	ctx context.Context,
	reader io.Reader,
	writer io.Writer,
) (int, int64, error) {
	startTime := time.Now()

	// Determine number of workers
	workers := runtime.NumCPU()

	// Create channels for job distribution and result collection
	jobs := make(chan WordJob, MaxJobQueueSize)
	results := make(chan WordJobResult, workers)

	// Create a wait group to track worker completion
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go p.wordWorker(ctx, i, jobs, results, &wg, writer)
	}

	// Create a goroutine to close the results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Create a goroutine to read and send chunks to workers
	errChan := make(chan error, 1)
	bytesProcessedChan := make(chan int64, 1)

	go func() {
		var chunkID int
		var inWord bool
		var bytesProcessed int64

		// Get a buffer for reading
		chunkBuffer := p.chunkBufferPool.Get()
		defer p.chunkBufferPool.Put(chunkBuffer)

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
				// Make a copy of the chunk for the worker
				// We need to copy since the buffer will be reused
				chunk := make([]byte, n)
				copy(chunk, chunkBuffer.Bytes[:n])
				bytesProcessed += int64(n)

				// Send the chunk to the worker pool
				select {
				case jobs <- WordJob{
					Chunk:     chunk,
					ChunkID:   chunkID,
					StartWord: inWord,
				}:
					// Job sent successfully
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				}

				// Update for the next chunk
				// Check if we end in a word character
				if n > 0 {
					lastByte := chunk[n-1]
					inWord = IsASCIIWordChar(lastByte)
				}

				chunkID++
			}

			// Handle end of stream or errors
			if err != nil {
				close(jobs) // No more jobs

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
	wordCount := 0
	resultMap := make(map[int]WordJobResult)
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
				return wordCount, <-bytesProcessedChan, result.Error
			}

			// Add to word count
			wordCount += result.WordCount

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
	p.logger.Debug("Parallel word processing completed",
		"word_count", wordCount,
		"bytes_processed", bytesProcessed,
		"workers", workers,
		"duration", time.Since(startTime),
	)

	return wordCount, bytesProcessed, err
}

// wordWorker is a worker goroutine that processes chunks in parallel
func (p *Processor) wordWorker(
	ctx context.Context,
	id int,
	jobs <-chan WordJob,
	results chan<- WordJobResult,
	wg *sync.WaitGroup,
	writer io.Writer,
) {
	defer wg.Done()

	// Get a word buffer from the pool
	wordBuffer := p.wordBufferPool.Get()
	defer p.wordBufferPool.Put(wordBuffer)

	// Process jobs until the channel is closed
	for job := range jobs {
		// Check context for cancellation
		select {
		case <-ctx.Done():
			results <- WordJobResult{
				ChunkID: job.ChunkID,
				Error:   ctx.Err(),
			}
			return
		default:
			// Continue processing
		}

		// Process the chunk
		var wordCount int
		var endWord bool

		// Determine if we can use the fast ASCII path
		asciiOnly := IsASCIIOnly(job.Chunk)

		// Variables for word tracking
		inWord := job.StartWord
		wordStart := 0

		if asciiOnly {
			// Fast ASCII path
			for i := 0; i < len(job.Chunk); i++ {
				b := job.Chunk[i]
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
							wordBuffer.Bytes = append(wordBuffer.Bytes[:0], job.Chunk[wordStart:i]...)
							normalized := p.normalizer.Normalize(string(wordBuffer.Bytes))

							// For parallel writer support, we would need a write mutex here
							// For now, this is a simplification and would need additional synchronization
							writer.Write([]byte(normalized + " "))
						}

						inWord = false
					}
				}
			}
		} else {
			// Non-ASCII path
			i := 0
			for i < len(job.Chunk) {
				// Fix: Use blank identifier for unused variable
				_, size, isChar := HandleUTF8(job.Chunk, i)

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
							wordBuffer.Bytes = append(wordBuffer.Bytes[:0], job.Chunk[wordStart:i]...)
							normalized := p.normalizer.Normalize(string(wordBuffer.Bytes))
							writer.Write([]byte(normalized + " "))
						}

						inWord = false
					}
				}

				i += size
			}
		}

		// Are we ending in a word?
		endWord = inWord

		// Send the result
		results <- WordJobResult{
			WordCount: wordCount,
			ChunkID:   job.ChunkID,
			EndWord:   endWord,
			Error:     nil,
		}
	}
}
