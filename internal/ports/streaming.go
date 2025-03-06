package ports

import (
	"context"
	"io"
	"time"
)

// StreamingMode represents different modes for processing input streams
type StreamingMode int

const (
	// ChunkByChunk processes the input stream in fixed-size chunks
	ChunkByChunk StreamingMode = iota
	// LineByLine processes the input stream line by line
	LineByLine
	// WordByWord processes the input stream word by word
	WordByWord
)

// StreamProcessor defines the interface for processing text streams
type StreamProcessor interface {
	// ProcessStream processes an input stream and returns the length (character or word count)
	ProcessStream(ctx context.Context, reader io.Reader, mode StreamingMode) (int, error)

	// ProcessStreamWithWriter processes an input stream, potentially transforms it, and writes to the output writer
	ProcessStreamWithWriter(ctx context.Context, reader io.Reader, writer io.Writer, mode StreamingMode) (int, error)
}

// StreamResult holds the outcome of a similarity computation on streams
type StreamResult struct {
	Name            string
	Score           float64
	Passed          bool
	OriginalLength  int
	AugmentedLength int
	LengthRatio     float64
	Threshold       float64
	Details         map[string]interface{}
	// Additional fields relevant to streaming processing
	BytesProcessed int64
	ProcessingTime time.Duration
}
