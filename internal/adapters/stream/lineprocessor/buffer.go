package lineprocessor

import "sync"

// import (
//
//	"sync"
//
// )
//
// // LineBuffer represents a reusable buffer for storing a line of text
//
//	type LineBuffer struct {
//		// Buffer to store line bytes
//		Bytes []byte
//	}
//
// // LineBufferPool implements a pool of line buffers for efficient reuse
//
//	type LineBufferPool struct {
//		pool sync.Pool
//	}
//
// // NewLineBufferPool creates a new line buffer pool
//
//	func NewLineBufferPool() *LineBufferPool {
//		return &LineBufferPool{
//			pool: sync.Pool{
//				New: func() interface{} {
//					// Most lines are under 256 bytes
//					buf := make([]byte, 0, 256)
//					return &LineBuffer{Bytes: buf}
//				},
//			},
//		}
//	}
//
// // Get retrieves a line buffer from the pool
//
//	func (lbp *LineBufferPool) Get() *LineBuffer {
//		return lbp.pool.Get().(*LineBuffer)
//	}
//
// // Put returns a line buffer to the pool
//
//	func (lbp *LineBufferPool) Put(lb *LineBuffer) {
//		// Reset length but keep capacity
//		lb.Bytes = lb.Bytes[:0]
//		lbp.pool.Put(lb)
//	}
//
// // ChunkBuffer represents a larger buffer for processing chunks of text
//
//	type ChunkBuffer struct {
//		// Buffer to store chunk bytes
//		Bytes []byte
//	}
//
// // ChunkBufferPool implements a pool of chunk buffers
//
//	type ChunkBufferPool struct {
//		pool      sync.Pool
//		chunkSize int
//	}
//
// // NewChunkBufferPool creates a new chunk buffer pool
//
//	func NewChunkBufferPool(chunkSize int) *ChunkBufferPool {
//		return &ChunkBufferPool{
//			pool: sync.Pool{
//				New: func() interface{} {
//					buf := make([]byte, chunkSize)
//					return &ChunkBuffer{Bytes: buf}
//				},
//			},
//			chunkSize: chunkSize,
//		}
//	}
//
// // Get retrieves a chunk buffer from the pool
//
//	func (cbp *ChunkBufferPool) Get() *ChunkBuffer {
//		buffer := cbp.pool.Get().(*ChunkBuffer)
//
//		// Ensure buffer has correct size (in case chunkSize changed)
//		if cap(buffer.Bytes) < cbp.chunkSize {
//			buffer.Bytes = make([]byte, cbp.chunkSize)
//		} else {
//			buffer.Bytes = buffer.Bytes[:cbp.chunkSize]
//		}
//
//		return buffer
//	}
//
// // Put returns a chunk buffer to the pool
//
//	func (cbp *ChunkBufferPool) Put(cb *ChunkBuffer) {
//		cbp.pool.Put(cb)
//	}
//
// LineBatchBuffer holds a batch of lines for batch processing
type LineBatchBuffer struct {
	// Slice of line slices
	Lines [][]byte
}

// LineBatchBufferPool implements a pool of line batch buffers
type LineBatchBufferPool struct {
	pool      sync.Pool
	batchSize int
}

// NewLineBatchBufferPool creates a new line batch buffer pool
func NewLineBatchBufferPool(batchSize int) *LineBatchBufferPool {
	return &LineBatchBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				lines := make([][]byte, 0, batchSize)
				return &LineBatchBuffer{Lines: lines}
			},
		},
		batchSize: batchSize,
	}
}

//// Get retrieves a line batch buffer from the pool
//func (lbbp *LineBatchBufferPool) Get() *LineBatchBuffer {
//	buffer := lbbp.pool.Get().(*LineBatchBuffer)
//
//	// Ensure buffer has correct capacity
//	if cap(buffer.Lines) < lbbp.batchSize {
//		buffer.Lines = make([][]byte, 0, lbbp.batchSize)
//	} else {
//		buffer.Lines = buffer.Lines[:0]
//	}
//
//	return buffer
//}
//
//// Put returns a line batch buffer to the pool
//func (lbbp *LineBatchBufferPool) Put(lbb *LineBatchBuffer) {
//	// Clear all line references to avoid memory leaks
//	for i := range lbb.Lines {
//		lbb.Lines[i] = nil
//	}
//	lbb.Lines = lbb.Lines[:0]
//	lbbp.pool.Put(lbb)
//}
