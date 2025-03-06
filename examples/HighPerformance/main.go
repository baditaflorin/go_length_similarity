package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/baditaflorin/go_length_similarity/internal/warmup"
	"github.com/baditaflorin/go_length_similarity/pkg/character"
	"github.com/baditaflorin/go_length_similarity/pkg/word"
	"github.com/baditaflorin/l"
)

func main() {
	// Create a custom logger
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

	// Create a custom warmup configuration
	warmupConfig := warmup.WarmupConfig{
		Concurrency:    4,               // Use 4 concurrent routines
		Iterations:     2000,            // 2000 iterations per routine
		SampleTextSize: 2000,            // 2000 characters sample text
		Duration:       2 * time.Second, // Max 2 seconds for warmup
		ForceGC:        true,            // Force GC after warmup
	}

	// Initialize length similarity with performance optimizations
	ls, err := word.New(
		word.WithThreshold(0.8),
		word.WithMaxDiffRatio(0.2),
		word.WithLogger(logger),
		// Use fast normalizer for better performance
		word.WithFastNormalizer(),
		// Enable warmup with custom config
		word.WithWarmUpConfig(warmupConfig),
	)
	if err != nil {
		panic(err)
	}

	// Initialize character similarity with optimizations
	cs, err := character.NewCharacterSimilarity(
		character.WithThreshold(0.8),
		character.WithMaxDiffRatio(0.2),
		character.WithLogger(logger),
		character.WithPrecision(2),
		// Use optimized normalizer
		character.WithOptimizedNormalizer(),
		// Enable warmup with custom config
		character.WithWarmUpConfig(warmupConfig),
	)
	if err != nil {
		panic(err)
	}

	// Sample texts to compare
	original := "This is the original text with multiple sentences. It contains several paragraphs of text that will be used to test the performance of our similarity metrics. The metrics should efficiently calculate how similar two texts are based on their length or character count."

	// Create three variants with different levels of similarity
	similar := "This is the similar text with multiple sentences. It contains several paragraphs of content that will be used to test the performance of our similarity metrics. The metrics should efficiently measure how similar two texts are based on their length or character count."
	different := "This text is quite different from the original. It has a completely different structure and uses different vocabulary. The content has been changed significantly to test how the similarity metrics handle dissimilar texts."
	veryDifferent := "Short unrelated text."

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Benchmark the performance
	fmt.Println("=== Performance Benchmark ===")

	// Length similarity
	fmt.Println("\nLength Similarity:")
	benchmarkSimilarity(ctx, ls, original, similar, different, veryDifferent)

	// Character similarity
	fmt.Println("\nCharacter Similarity:")
	benchmarkSimilarity(ctx, cs, original, similar, different, veryDifferent)
}

// benchmarkSimilarity runs performance tests for a similarity calculator
func benchmarkSimilarity(ctx context.Context, calculator interface{}, original, similar, different, veryDifferent string) {
	// Run each comparison multiple times to get stable results
	iterations := 1000

	// Functions to benchmark different scenarios
	scenarios := []struct {
		name  string
		text1 string
		text2 string
	}{
		{"Identical", original, original},
		{"Similar", original, similar},
		{"Different", original, different},
		{"Very Different", original, veryDifferent},
	}

	// Run benchmarks for each scenario
	for _, scenario := range scenarios {
		start := time.Now()

		// Run multiple iterations
		for i := 0; i < iterations; i++ {
			switch c := calculator.(type) {
			case *word.LengthSimilarity:
				c.Compute(ctx, scenario.text1, scenario.text2)
			case *character.CharacterSimilarity:
				c.Compute(ctx, scenario.text1, scenario.text2)
			}
		}

		elapsed := time.Since(start)
		opsPerSec := float64(iterations) / elapsed.Seconds()

		fmt.Printf("  %s: %.2f ops/sec (%.2f µs/op)\n",
			scenario.name,
			opsPerSec,
			float64(elapsed.Microseconds())/float64(iterations),
		)
	}
}
