package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/baditaflorin/go_length_similarity/pkg/streaming"
	l "github.com/baditaflorin/l"
)

func main() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Example using strings (simple case)
	simpleStreamingExample(ctx)

	// Example using file readers (for large files)
	fileStreamingExample(ctx)

	// Example using high-performance optimized streaming
	highPerformanceStreamingExample(ctx)
}

func simpleStreamingExample(ctx context.Context) {
	fmt.Println("=== Simple Streaming Example ===")

	// Original and augmented texts
	original := "This is a longer text that we'll process in a streaming fashion. " +
		"Streaming is useful when working with large texts that might not fit in memory. " +
		"The streaming calculator processes text chunk by chunk, line by line, or word by word."

	augmented := "This is a longer example that we'll analyze in a streaming manner. " +
		"Streaming is beneficial when processing large texts that might not fit in memory. " +
		"The streaming similarity calculator processes input chunk by chunk, line by line, or word by word."

	// Initialize streaming similarity with default settings
	ss, err := streaming.NewStreamingSimilarity()
	if err != nil {
		panic(err)
	}

	// Compute similarity from strings
	result := ss.ComputeFromStrings(ctx, original, augmented)

	// Display results
	fmt.Printf("Score: %.2f\n", result.Score)
	fmt.Printf("Passed: %v\n", result.Passed)
	fmt.Printf("Original length: %d\n", result.OriginalLength)
	fmt.Printf("Augmented length: %d\n", result.AugmentedLength)
	fmt.Printf("Length ratio: %.2f\n", result.LengthRatio)
	fmt.Printf("Processing time: %s\n", result.ProcessingTime)
	fmt.Printf("Bytes processed: %d\n", result.BytesProcessed)
}

func fileStreamingExample(ctx context.Context) {
	fmt.Println("\n=== File Streaming Example ===")

	// Create temporary files for demonstration
	originalFile, err := os.CreateTemp("", "original-*.txt")
	if err != nil {
		panic(err)
	}
	defer os.Remove(originalFile.Name())
	defer originalFile.Close()

	augmentedFile, err := os.CreateTemp("", "augmented-*.txt")
	if err != nil {
		panic(err)
	}
	defer os.Remove(augmentedFile.Name())
	defer augmentedFile.Close()

	// Write sample content to files
	originalContent := "Line 1: This is an example file.\nLine 2: It contains multiple lines.\nLine 3: We'll process it using streaming.\n"
	augmentedContent := "Line 1: This is a sample file.\nLine 2: It has several lines.\nLine 3: We'll analyze it using streaming.\n"

	originalFile.WriteString(originalContent)
	augmentedFile.WriteString(augmentedContent)

	// Reset file pointers to beginning
	originalFile.Seek(0, 0)
	augmentedFile.Seek(0, 0)

	// Initialize streaming similarity with line-by-line mode
	ss, err := streaming.NewStreamingSimilarity(
		streaming.WithStreamingMode(streaming.LineByLine),
		streaming.WithStreamingChunkSize(1024), // 1KB chunks
	)
	if err != nil {
		panic(err)
	}

	// Compute similarity from file readers
	result := ss.ComputeFromReaders(ctx, originalFile, augmentedFile)

	// Display results
	fmt.Printf("Score: %.2f\n", result.Score)
	fmt.Printf("Passed: %v\n", result.Passed)
	fmt.Printf("Original length: %d\n", result.OriginalLength)
	fmt.Printf("Augmented length: %d\n", result.AugmentedLength)
	fmt.Printf("Length ratio: %.2f\n", result.LengthRatio)
	fmt.Printf("Processing time: %s\n", result.ProcessingTime)
	fmt.Printf("Bytes processed: %d\n", result.BytesProcessed)
}

func highPerformanceStreamingExample(ctx context.Context) {
	fmt.Println("\n=== High-Performance Streaming Example ===")

	// Original and augmented texts
	original := "This is a sample for testing the high-performance streaming processor.\n" +
		"It uses the allocation-efficient implementation for better performance.\n" +
		"This is particularly useful for large texts and memory-constrained environments."

	augmented := "This is an example for testing the optimized streaming processor.\n" +
		"It uses the allocation-efficient implementation for improved performance.\n" +
		"This is especially beneficial for large texts and memory-constrained systems."

	// Create a proper logger instance
	factory := l.NewStandardFactory()
	logger, err := factory.CreateLogger(l.Config{
		Output:     os.Stdout,
		JsonFormat: false,
		AsyncWrite: true,
	})
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Initialize the allocation-efficient streaming similarity
	ss, err := streaming.NewAllocationEfficientStreamingSimilarity(
		logger,
		streaming.WithEfficientMode(streaming.LineByLine),
		streaming.WithEfficientChunkSize(4096), // 4KB chunks
		streaming.WithEfficientParallel(true),  // Enable parallel processing
		streaming.WithEfficientThreshold(0.75), // Custom threshold
	)
	if err != nil {
		panic(err)
	}

	// Compute similarity
	result := ss.ComputeFromStrings(ctx, original, augmented)

	// Display results
	fmt.Printf("Score: %.2f\n", result.Score)
	fmt.Printf("Passed: %v\n", result.Passed)
	fmt.Printf("Original length: %d\n", result.OriginalLength)
	fmt.Printf("Augmented length: %d\n", result.AugmentedLength)
	fmt.Printf("Length ratio: %.2f\n", result.LengthRatio)
	fmt.Printf("Processing time: %s\n", result.ProcessingTime)
	fmt.Printf("Bytes processed: %d\n", result.BytesProcessed)
}

/*
Sample output:

=== Simple Streaming Example ===
Score: 0.85
Passed: true
Original length: 36
Augmented length: 39
Length ratio: 0.92
Processing time: 1.032ms
Bytes processed: 498

=== File Streaming Example ===
Score: 0.89
Passed: true
Original length: 16
Augmented length: 14
Length ratio: 0.88
Processing time: 1.245ms
Bytes processed: 186

=== High-Performance Streaming Example ===
Score: 0.91
Passed: true
Original length: 30
Augmented length: 31
Length ratio: 0.97
Processing time: 0.873ms
Bytes processed: 431
*/
