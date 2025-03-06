package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/baditaflorin/go_length_similarity/pkg/character"
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
	cs, err := character.NewCharacterSimilarity(
		character.WithThreshold(0.8),
		character.WithMaxDiffRatio(0.2),
		character.WithLogger(logger),
		character.WithPrecision(2),
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
