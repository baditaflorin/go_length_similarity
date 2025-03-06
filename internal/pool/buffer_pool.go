package pool

import (
	"strings"
	"sync"
)

// BufferPool implements a pool of byte slices for efficient memory reuse
type BufferPool struct {
	pool sync.Pool
	size int
}

// NewBufferPool creates a new buffer pool with buffers of the specified size
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				buffer := make([]byte, 0, size)
				return &buffer
			},
		},
		size: size,
	}
}

// Get retrieves a buffer from the pool or creates a new one if none are available
func (bp *BufferPool) Get() *[]byte {
	return bp.pool.Get().(*[]byte)
}

// Put returns a buffer to the pool for reuse
func (bp *BufferPool) Put(buffer *[]byte) {
	// Reset buffer length but keep capacity
	*buffer = (*buffer)[:0]
	bp.pool.Put(buffer)
}

// StringBuilderPool implements a pool of strings.Builder for efficient string building
type StringBuilderPool struct {
	pool sync.Pool
}

// NewStringBuilderPool creates a new strings.Builder pool
func NewStringBuilderPool() *StringBuilderPool {
	return &StringBuilderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(StringBuilder)
			},
		},
	}
}

// Get retrieves a StringBuilder from the pool or creates a new one if none are available
func (sbp *StringBuilderPool) Get() *StringBuilder {
	return sbp.pool.Get().(*StringBuilder)
}

// Put returns a StringBuilder to the pool for reuse
func (sbp *StringBuilderPool) Put(sb *StringBuilder) {
	sb.Reset()
	sbp.pool.Put(sb)
}

// StringBuilder wraps strings.Builder with additional functionality
type StringBuilder struct {
	builder    strings.Builder
	runeBuffer []rune
}

// WriteRune writes a rune to the builder
func (sb *StringBuilder) WriteRune(r rune) {
	sb.builder.WriteRune(r)
}

// WriteString writes a string to the builder
func (sb *StringBuilder) WriteString(s string) {
	sb.builder.WriteString(s)
}

// String returns the accumulated string
func (sb *StringBuilder) String() string {
	return sb.builder.String()
}

// Reset resets the builder for reuse
func (sb *StringBuilder) Reset() {
	sb.builder.Reset()
	sb.runeBuffer = sb.runeBuffer[:0]
}

// RuneBufferPool implements a pool of rune slices
type RuneBufferPool struct {
	pool sync.Pool
	size int
}

// NewRuneBufferPool creates a new pool of rune slices with the specified size
func NewRuneBufferPool(size int) *RuneBufferPool {
	return &RuneBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				buffer := make([]rune, 0, size)
				return &buffer
			},
		},
		size: size,
	}
}

// Get retrieves a rune buffer from the pool
func (rbp *RuneBufferPool) Get() *[]rune {
	return rbp.pool.Get().(*[]rune)
}

// Put returns a rune buffer to the pool
func (rbp *RuneBufferPool) Put(buffer *[]rune) {
	*buffer = (*buffer)[:0]
	rbp.pool.Put(buffer)
}
