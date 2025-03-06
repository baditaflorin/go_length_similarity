package wordprocessor

import (
	"sync"
)

// WordBuffer represents a reusable buffer for word processing
type WordBuffer struct {
	// Buffer to store word bytes
	Bytes []byte
}

// WordBufferPool implements a pool of word buffers for efficient reuse
type WordBufferPool struct {
	pool sync.Pool
}

// NewWordBufferPool creates a new word buffer pool
func NewWordBufferPool() *WordBufferPool {
	return &WordBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				// Most words are under 64 bytes
				buf := make([]byte, 0, 64)
				return &WordBuffer{Bytes: buf}
			},
		},
	}
}

// Get retrieves a word buffer from the pool
func (wbp *WordBufferPool) Get() *WordBuffer {
	return wbp.pool.Get().(*WordBuffer)
}

// Put returns a word buffer to the pool
func (wbp *WordBufferPool) Put(wb *WordBuffer) {
	// Reset length but keep capacity
	wb.Bytes = wb.Bytes[:0]
	wbp.pool.Put(wb)
}

// ChunkBuffer represents a larger buffer for processing chunks of text
type ChunkBuffer struct {
	// Buffer to store chunk bytes
	Bytes []byte
}

// ChunkBufferPool implements a pool of chunk buffers
type ChunkBufferPool struct {
	pool      sync.Pool
	chunkSize int
}

// NewChunkBufferPool creates a new chunk buffer pool
func NewChunkBufferPool(chunkSize int) *ChunkBufferPool {
	return &ChunkBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, chunkSize)
				return &ChunkBuffer{Bytes: buf}
			},
		},
		chunkSize: chunkSize,
	}
}

// Get retrieves a chunk buffer from the pool
func (cbp *ChunkBufferPool) Get() *ChunkBuffer {
	buffer := cbp.pool.Get().(*ChunkBuffer)

	// Ensure buffer has correct size (in case chunkSize changed)
	if cap(buffer.Bytes) < cbp.chunkSize {
		buffer.Bytes = make([]byte, cbp.chunkSize)
	} else {
		buffer.Bytes = buffer.Bytes[:cbp.chunkSize]
	}

	return buffer
}

// Put returns a chunk buffer to the pool
func (cbp *ChunkBufferPool) Put(cb *ChunkBuffer) {
	cbp.pool.Put(cb)
}

// WordBatchBuffer holds a batch of words for batch processing
type WordBatchBuffer struct {
	// Slice of word slices
	Words [][]byte
}

// WordBatchBufferPool implements a pool of word batch buffers
type WordBatchBufferPool struct {
	pool      sync.Pool
	batchSize int
}

// NewWordBatchBufferPool creates a new word batch buffer pool
func NewWordBatchBufferPool(batchSize int) *WordBatchBufferPool {
	return &WordBatchBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				words := make([][]byte, 0, batchSize)
				return &WordBatchBuffer{Words: words}
			},
		},
		batchSize: batchSize,
	}
}

// Get retrieves a word batch buffer from the pool
func (wbbp *WordBatchBufferPool) Get() *WordBatchBuffer {
	buffer := wbbp.pool.Get().(*WordBatchBuffer)

	// Ensure buffer has correct capacity
	if cap(buffer.Words) < wbbp.batchSize {
		buffer.Words = make([][]byte, 0, wbbp.batchSize)
	} else {
		buffer.Words = buffer.Words[:0]
	}

	return buffer
}

// Put returns a word batch buffer to the pool
func (wbbp *WordBatchBufferPool) Put(wbb *WordBatchBuffer) {
	// Clear all word references to avoid memory leaks
	for i := range wbb.Words {
		wbb.Words[i] = nil
	}
	wbb.Words = wbb.Words[:0]
	wbbp.pool.Put(wbb)
}
