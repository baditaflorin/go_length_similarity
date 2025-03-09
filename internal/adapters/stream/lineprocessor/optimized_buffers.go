// File: internal/adapters/stream/lineprocessor/optimized_buffers.go
package lineprocessor

import (
	"strings"
	"sync"
)

// LineRanges represents a collection of line boundaries without storing line content
type LineRanges struct {
	Ranges []struct{ Start, End int }
	Count  int
}

// LineRangePool implements a pool of LineRanges for efficient reuse
type LineRangePool struct {
	pool     sync.Pool
	capacity int
}

// NewLineRangePool creates a new line range pool
func NewLineRangePool(capacity int) *LineRangePool {
	return &LineRangePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &LineRanges{
					Ranges: make([]struct{ Start, End int }, capacity),
					Count:  0,
				}
			},
		},
		capacity: capacity,
	}
}

// Get retrieves a LineRanges from the pool
func (lrp *LineRangePool) Get() *LineRanges {
	return lrp.pool.Get().(*LineRanges)
}

// Put returns a LineRanges to the pool
func (lrp *LineRangePool) Put(lr *LineRanges) {
	lr.Count = 0 // Reset count without allocating
	lrp.pool.Put(lr)
}

// Add adds a line range to the collection
func (lr *LineRanges) Add(start, end int) {
	if lr.Count >= len(lr.Ranges) {
		// Grow the slice if needed, avoiding frequent reallocations
		newCap := len(lr.Ranges) * 2
		if newCap == 0 {
			newCap = 8
		}
		newRanges := make([]struct{ Start, End int }, newCap)
		copy(newRanges, lr.Ranges)
		lr.Ranges = newRanges
	}

	lr.Ranges[lr.Count].Start = start
	lr.Ranges[lr.Count].End = end
	lr.Count++
}

// Get retrieves a line range at the specified index
func (lr *LineRanges) Get(index int) struct{ Start, End int } {
	if index < 0 || index >= lr.Count {
		return struct{ Start, End int }{0, 0}
	}
	return lr.Ranges[index]
}

// Reset clears all line ranges without reallocating
func (lr *LineRanges) Reset() {
	lr.Count = 0
}

// StringBuilderPool implements a pool of strings.Builder for efficient string building
type StringBuilderPool struct {
	pool sync.Pool
}

// StringBuilder wraps strings.Builder for pooling
type StringBuilder struct {
	builder strings.Builder
}

// NewStringBuilderPool creates a new string builder pool
func NewStringBuilderPool() *StringBuilderPool {
	return &StringBuilderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &StringBuilder{}
			},
		},
	}
}

// Get retrieves a StringBuilder from the pool
func (sbp *StringBuilderPool) Get() *StringBuilder {
	return sbp.pool.Get().(*StringBuilder)
}

// Put returns a StringBuilder to the pool
func (sbp *StringBuilderPool) Put(sb *StringBuilder) {
	sb.Reset()
	sbp.pool.Put(sb)
}

// WriteString writes a string to the builder
func (sb *StringBuilder) WriteString(s string) {
	sb.builder.WriteString(s)
}

// WriteRune writes a rune to the builder
func (sb *StringBuilder) WriteRune(r rune) {
	sb.builder.WriteRune(r)
}

// String returns the accumulated string
func (sb *StringBuilder) String() string {
	return sb.builder.String()
}

// Reset resets the builder for reuse
func (sb *StringBuilder) Reset() {
	sb.builder.Reset()
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

// LineBuffer represents a reusable buffer for storing a line of text
type LineBuffer struct {
	// Buffer to store line bytes
	Bytes []byte
}

// LineBufferPool implements a pool of line buffers for efficient reuse
type LineBufferPool struct {
	pool sync.Pool
}

// NewLineBufferPool creates a new line buffer pool
func NewLineBufferPool() *LineBufferPool {
	return &LineBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				// Most lines are under 256 bytes
				buf := make([]byte, 0, 256)
				return &LineBuffer{Bytes: buf}
			},
		},
	}
}

// Get retrieves a line buffer from the pool
func (lbp *LineBufferPool) Get() *LineBuffer {
	return lbp.pool.Get().(*LineBuffer)
}

// Put returns a line buffer to the pool
func (lbp *LineBufferPool) Put(lb *LineBuffer) {
	// Reset length but keep capacity
	lb.Bytes = lb.Bytes[:0]
	lbp.pool.Put(lb)
}
