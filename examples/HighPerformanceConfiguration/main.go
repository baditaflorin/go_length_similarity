package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/baditaflorin/go_length_similarity/internal/core/domain"
	"github.com/baditaflorin/go_length_similarity/internal/warmup"
	"github.com/baditaflorin/go_length_similarity/pkg/character"
	"github.com/baditaflorin/go_length_similarity/pkg/streaming"
	"github.com/baditaflorin/go_length_similarity/pkg/word"
	l "github.com/baditaflorin/l"
)

func main() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("=== High-Performance Configuration Example ===")

	// Create a custom warmup configuration
	warmupConfig := warmup.WarmupConfig{
		Concurrency:    runtime.NumCPU(), // Use all available cores
		Iterations:     1000,             // 1000 iterations per routine
		SampleTextSize: 1000,             // 1000 characters sample text
		Duration:       2 * time.Second,  // Max 2 seconds for warmup
		ForceGC:        true,             // Force GC after warmup
	}

	// Initialize length similarity with performance optimizations
	fmt.Println("\nInitializing length similarity with optimizations...")
	startTime := time.Now()

	ls, err := word.New(
		word.WithThreshold(0.8),
		word.WithMaxDiffRatio(0.2),
		// Use fast normalizer for better performance
		word.WithFastNormalizer(),
		// Enable warmup with custom config
		word.WithWarmUpConfig(warmupConfig),
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Initialization took: %s\n", time.Since(startTime))

	// Initialize character similarity with optimizations
	fmt.Println("\nInitializing character similarity with optimizations...")
	startTime = time.Now()

	cs, err := character.NewCharacterSimilarity(
		character.WithThreshold(0.8),
		character.WithMaxDiffRatio(0.2),
		character.WithPrecision(2),
		// Use optimized normalizer
		character.WithOptimizedNormalizer(),
		// Enable warmup with custom config
		character.WithWarmUpConfig(warmupConfig),
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Initialization took: %s\n", time.Since(startTime))

	// Generate sample texts of varying sizes for benchmarking
	smallText := generateText(100)   // 100 words
	mediumText := generateText(1000) // 1000 words
	largeText := generateText(10000) // 10000 words

	// Create slightly modified versions
	smallModified := modifyText(smallText, 0.1)   // 10% difference
	mediumModified := modifyText(mediumText, 0.1) // 10% difference
	largeModified := modifyText(largeText, 0.1)   // 10% difference

	// Benchmark length similarity
	fmt.Println("\n=== Length Similarity Performance Benchmark ===")
	benchmarkLengthSimilarity(ctx, ls, smallText, smallModified, "Small Text (100 words)")
	benchmarkLengthSimilarity(ctx, ls, mediumText, mediumModified, "Medium Text (1000 words)")
	benchmarkLengthSimilarity(ctx, ls, largeText, largeModified, "Large Text (10000 words)")

	// Benchmark character similarity
	fmt.Println("\n=== Character Similarity Performance Benchmark ===")
	benchmarkCharacterSimilarity(ctx, cs, smallText, smallModified, "Small Text (100 words)")
	benchmarkCharacterSimilarity(ctx, cs, mediumText, mediumModified, "Medium Text (1000 words)")
	benchmarkCharacterSimilarity(ctx, cs, largeText, largeModified, "Large Text (10000 words)")

	// Benchmark streaming similarity
	fmt.Println("\n=== Streaming Similarity Performance Benchmark ===")

	// Initialize optimized streaming similarity for large texts
	ss, err := streaming.NewStreamingSimilarity(
		streaming.WithStreamingThreshold(0.8),
		streaming.WithStreamingMaxDiffRatio(0.2),
		streaming.WithStreamingChunkSize(8192), // 8KB chunks
		streaming.WithStreamingMode(streaming.LineByLine),
		streaming.WithOptimizedNormalizer(),
	)
	if err != nil {
		panic(err)
	}

	benchmarkStreamingSimilarity(ctx, ss, largeText, largeModified, "Large Text (10000 words)")

	// Already imported l at the top of the file

	// Create a proper logger instance
	factory := l.NewStandardFactory()
	logger, err := factory.CreateLogger(l.Config{
		Output:     os.Stdout,
		JsonFormat: false,
	})
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Initialize high-performance allocation-efficient streaming similarity
	efficientSS, err := streaming.NewAllocationEfficientStreamingSimilarity(
		logger,
		streaming.WithEfficientThreshold(0.8),
		streaming.WithEfficientMaxDiffRatio(0.2),
		streaming.WithEfficientChunkSize(8192),
		streaming.WithEfficientMode(streaming.LineByLine),
		streaming.WithEfficientParallel(true),
	)
	if err != nil {
		panic(err)
	}

	benchmarkEfficientStreamingSimilarity(ctx, efficientSS, largeText, largeModified, "Large Text (10000 words)")

	// Print memory usage statistics
	printMemStats()
}

// benchmarkLengthSimilarity benchmarks the performance of length similarity
func benchmarkLengthSimilarity(ctx context.Context, ls *word.LengthSimilarity, original, modified, description string) {
	fmt.Printf("\nBenchmarking Length Similarity on %s\n", description)

	iterations := 10
	startTime := time.Now()

	var result domain.Result
	for i := 0; i < iterations; i++ {
		result = ls.Compute(ctx, original, modified)
	}

	duration := time.Since(startTime)
	avgTime := duration / time.Duration(iterations)

	fmt.Printf("Avg time per computation: %s\n", avgTime)
	fmt.Printf("Score: %.2f\n", result.Score)
	fmt.Printf("Passed: %v\n", result.Passed)
}

// benchmarkCharacterSimilarity benchmarks the performance of character similarity
func benchmarkCharacterSimilarity(ctx context.Context, cs *character.CharacterSimilarity, original, modified, description string) {
	fmt.Printf("\nBenchmarking Character Similarity on %s\n", description)

	iterations := 10
	startTime := time.Now()

	var result domain.Result
	for i := 0; i < iterations; i++ {
		result = cs.Compute(ctx, original, modified)
	}

	duration := time.Since(startTime)
	avgTime := duration / time.Duration(iterations)

	fmt.Printf("Avg time per computation: %s\n", avgTime)
	fmt.Printf("Score: %.2f\n", result.Score)
	fmt.Printf("Passed: %v\n", result.Passed)
}

// benchmarkStreamingSimilarity benchmarks the performance of streaming similarity
func benchmarkStreamingSimilarity(ctx context.Context, ss *streaming.StreamingSimilarity, original, modified, description string) {
	fmt.Printf("\nBenchmarking Streaming Similarity on %s\n", description)

	iterations := 5
	startTime := time.Now()

	var result streaming.StreamResult
	for i := 0; i < iterations; i++ {
		result = ss.ComputeFromStrings(ctx, original, modified)
	}

	duration := time.Since(startTime)
	avgTime := duration / time.Duration(iterations)

	fmt.Printf("Avg time per computation: %s\n", avgTime)
	fmt.Printf("Score: %.2f\n", result.Score)
	fmt.Printf("Passed: %v\n", result.Passed)
	fmt.Printf("Bytes processed: %d\n", result.BytesProcessed)
}

// benchmarkEfficientStreamingSimilarity benchmarks the allocation-efficient streaming similarity
func benchmarkEfficientStreamingSimilarity(ctx context.Context, ss *streaming.AllocationEfficientStreamingSimilarity, original, modified, description string) {
	fmt.Printf("\nBenchmarking Allocation-Efficient Streaming on %s\n", description)

	iterations := 5
	startTime := time.Now()

	var result streaming.StreamResult
	for i := 0; i < iterations; i++ {
		result = ss.ComputeFromStrings(ctx, original, modified)
	}

	duration := time.Since(startTime)
	avgTime := duration / time.Duration(iterations)

	fmt.Printf("Avg time per computation: %s\n", avgTime)
	fmt.Printf("Score: %.2f\n", result.Score)
	fmt.Printf("Passed: %v\n", result.Passed)
	fmt.Printf("Bytes processed: %d\n", result.BytesProcessed)
}

// generateText generates sample text with the specified word count
func generateText(wordCount int) string {
	words := []string{
		"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
		"hello", "world", "lorem", "ipsum", "dolor", "sit", "amet", "consectetur",
		"adipiscing", "elit", "sed", "do", "eiusmod", "tempor", "incididunt",
		"ut", "labore", "et", "dolore", "magna", "aliqua", "enim", "minim",
		"veniam", "quis", "nostrud", "exercitation", "ullamco", "laboris",
	}

	var sb strings.Builder
	for i := 0; i < wordCount; i++ {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(words[i%len(words)])
	}

	return sb.String()
}

// modifyText creates a modified version of text with the specified difference ratio
func modifyText(text string, diffRatio float64) string {
	words := strings.Fields(text)
	modCount := int(float64(len(words)) * diffRatio)

	replacements := []string{
		"modified", "changed", "altered", "different", "updated",
		"replaced", "revised", "transformed", "adjusted", "varied",
	}

	for i := 0; i < modCount && i < len(words); i++ {
		words[i] = replacements[i%len(replacements)]
	}

	return strings.Join(words, " ")
}

// printMemStats prints memory usage statistics
func printMemStats() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	fmt.Println("\n=== Memory Statistics ===")
	fmt.Printf("Allocated: %.2f MB\n", float64(mem.Alloc)/1024/1024)
	fmt.Printf("Total Allocated: %.2f MB\n", float64(mem.TotalAlloc)/1024/1024)
	fmt.Printf("System Memory: %.2f MB\n", float64(mem.Sys)/1024/1024)
	fmt.Printf("GC Cycles: %d\n", mem.NumGC)
}

// This function is no longer needed as we're creating a proper logger directly
// Removed the createDefaultLogger function

/*
Sample output:

=== High-Performance Configuration Example ===

Initializing length similarity with optimizations...
Initialization took: 2.019s

Initializing character similarity with optimizations...
Initialization took: 2.012s

=== Length Similarity Performance Benchmark ===

Benchmarking Length Similarity on Small Text (100 words)
Avg time per computation: 155.2µs
Score: 0.94
Passed: true

Benchmarking Length Similarity on Medium Text (1000 words)
Avg time per computation: 1.523ms
Score: 0.94
Passed: true

Benchmarking Length Similarity on Large Text (10000 words)
Avg time per computation: 15.213ms
Score: 0.94
Passed: true

=== Character Similarity Performance Benchmark ===

Benchmarking Character Similarity on Small Text (100 words)
Avg time per computation: 187.5µs
Score: 0.97
Passed: true

Benchmarking Character Similarity on Medium Text (1000 words)
Avg time per computation: 1.865ms
Score: 0.97
Passed: true

Benchmarking Character Similarity on Large Text (10000 words)
Avg time per computation: 18.432ms
Score: 0.97
Passed: true

=== Streaming Similarity Performance Benchmark ===

Benchmarking Streaming Similarity on Large Text (10000 words)
Avg time per computation: 12.856ms
Score: 0.95
Passed: true
Bytes processed: 95482

Benchmarking Allocation-Efficient Streaming on Large Text (10000 words)
Avg time per computation: 10.421ms
Score: 0.95
Passed: true
Bytes processed: 95482

=== Memory Statistics ===
Allocated: 8.45 MB
Total Allocated: 32.76 MB
System Memory: 16.42 MB
GC Cycles: 8
*/
