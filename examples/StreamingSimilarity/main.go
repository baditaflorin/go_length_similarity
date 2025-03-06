package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/baditaflorin/go_length_similarity/pkg/streaming"
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

	// Initialize the streaming similarity metric with optimized settings
	ss, err := streaming.NewStreamingSimilarity(
		streaming.WithStreamingThreshold(0.8),
		streaming.WithStreamingMaxDiffRatio(0.2),
		streaming.WithStreamingLogger(logger),
		streaming.WithStreamingChunkSize(4096), // 4KB chunks
		streaming.WithStreamingMode(streaming.LineByLine),
		// Use optimized normalizer for better performance
		streaming.WithOptimizedNormalizer(),
	)
	if err != nil {
		panic(err)
	}

	// Sample texts to compare
	original := generateLargeText(100000) // 100K words
	modified := modifyText(original, 0.1) // 10% difference

	// Create readers from strings (in real world, might be files)
	originalReader := strings.NewReader(original)
	modifiedReader := strings.NewReader(modified)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Measure performance
	start := time.Now()

	// Compute similarity using streaming
	result := ss.ComputeFromReaders(ctx, originalReader, modifiedReader)

	duration := time.Since(start)

	// Print results
	fmt.Printf("Streaming Similarity Result:\n")
	fmt.Printf("  Score: %.4f\n", result.Score)
	fmt.Printf("  Passed: %v\n", result.Passed)
	fmt.Printf("  Original Length: %d\n", result.OriginalLength)
	fmt.Printf("  Augmented Length: %d\n", result.AugmentedLength)
	fmt.Printf("  Bytes Processed: %d\n", result.BytesProcessed)
	fmt.Printf("  Processing Time: %s\n", result.ProcessingTime)
	fmt.Printf("  Performance: %.2f MB/s\n", float64(result.BytesProcessed)/1024/1024/duration.Seconds())
}

// generateLargeText creates a large sample text with the specified word count
func generateLargeText(wordCount int) string {
	// Sample vocabulary for generating text
	words := []string{
		"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
		"hello", "world", "lorem", "ipsum", "dolor", "sit", "amet", "consectetur",
		"adipiscing", "elit", "sed", "do", "eiusmod", "tempor", "incididunt",
		"ut", "labore", "et", "dolore", "magna", "aliqua", "enim", "minim",
		"veniam", "quis", "nostrud", "exercitation", "ullamco", "laboris",
		"nisi", "aliquip", "ex", "ea", "commodo", "consequat", "duis", "aute",
		"irure", "dolor", "reprehenderit", "voluptate", "velit", "esse", "cillum",
	}

	var sb strings.Builder
	sb.Grow(wordCount * 6) // Assume average word length of 5 + space

	for i := 0; i < wordCount; i++ {
		if i > 0 {
			sb.WriteString(" ")
		}
		wordIndex := i % len(words)
		sb.WriteString(words[wordIndex])
	}

	return sb.String()
}

// modifyText alters a percentage of words in the original text
func modifyText(original string, modifyRatio float64) string {
	words := strings.Fields(original)
	wordsToModify := int(float64(len(words)) * modifyRatio)

	// Replacement vocabulary
	replacements := []string{
		"modified", "changed", "altered", "different", "unique",
		"new", "fresh", "novel", "replaced", "updated",
	}

	// Make a copy of the original words
	result := make([]string, len(words))
	copy(result, words)

	// Modify a percentage of words
	for i := 0; i < wordsToModify && i < len(words); i++ {
		result[i] = replacements[i%len(replacements)]
	}

	return strings.Join(result, " ")
}
