package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/baditaflorin/go_length_similarity"
	"github.com/baditaflorin/l"
)

func main() {
	// Create a custom logger using the original l package
	factory := l.NewStandardFactory()
	logger, err := factory.CreateLogger(l.Config{
		Output:      os.Stdout,
		JsonFormat:  true,
		AsyncWrite:  true,
		BufferSize:  1024 * 1024,
		MaxFileSize: 10 * 1024 * 1024,
		MaxBackups:  5,
		AddSource:   true,
		Metrics:     true,
	})
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Initialize the character similarity metric using our new public API
	cs, err := lengthsimilarity.NewCharacterSimilarity(
		lengthsimilarity.WithThreshold(0.8),
		lengthsimilarity.WithMaxDiffRatio(0.2),
		lengthsimilarity.WithLogger(logger),
		lengthsimilarity.WithPrecision(2),
	)
	if err != nil {
		panic(err)
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Compute the character similarity score between two texts
	result := cs.Compute(ctx, "This is the original text.", "This is the augmented text!")
	fmt.Printf("Character Similarity Result: %+v\n", result)
}
