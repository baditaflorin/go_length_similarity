// main.go
package main

import (
	"fmt"
	"os"

	"github.com/baditaflorin/go_length_similarity"
	"github.com/baditaflorin/l"
)

func main() {
	// Create a custom logger.
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

	// Initialize the length similarity metric.
	ls := lengthsimilarity.New(
		lengthsimilarity.WithThreshold(0.8),
		lengthsimilarity.WithMaxDiffRatio(0.2),
		lengthsimilarity.WithLogger(logger),
	)

	// Compute the similarity score between two texts.
	result := ls.Compute("This is the original text.", "This is the augmented text!")
	fmt.Printf("Result: %+v\n", result)
}
