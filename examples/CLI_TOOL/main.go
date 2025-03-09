package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/baditaflorin/go_length_similarity/internal/core/domain"
	"github.com/baditaflorin/go_length_similarity/pkg/character"
	"github.com/baditaflorin/go_length_similarity/pkg/streaming"
	"github.com/baditaflorin/go_length_similarity/pkg/word"
)

// Command-line flags
var (
	originalFile  string
	augmentedFile string
	originalText  string
	augmentedText string
	metric        string
	threshold     float64
	maxDiffRatio  float64
	useStreaming  bool
	streamingMode string
	optimizeSpeed bool
	outputFormat  string
	verbose       bool
)

func init() {
	// File inputs
	flag.StringVar(&originalFile, "original-file", "", "Path to the original text file")
	flag.StringVar(&augmentedFile, "augmented-file", "", "Path to the augmented text file")

	// Direct text inputs
	flag.StringVar(&originalText, "original", "", "Original text content")
	flag.StringVar(&augmentedText, "augmented", "", "Augmented text content")

	// Metric configuration
	flag.StringVar(&metric, "metric", "length", "Similarity metric to use: 'length', 'character', or 'both'")
	flag.Float64Var(&threshold, "threshold", 0.7, "Similarity threshold (0.0-1.0)")
	flag.Float64Var(&maxDiffRatio, "max-diff-ratio", 0.3, "Maximum difference ratio")

	// Streaming options
	flag.BoolVar(&useStreaming, "streaming", false, "Use streaming processing (for large files)")
	flag.StringVar(&streamingMode, "streaming-mode", "line", "Streaming mode: 'chunk', 'line', or 'word'")

	// Performance options
	flag.BoolVar(&optimizeSpeed, "optimize-speed", false, "Enable performance optimizations")

	// Output options
	flag.StringVar(&outputFormat, "output", "text", "Output format: 'text' or 'json'")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")

	// Add help text
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --original-file=orig.txt --augmented-file=aug.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --original=\"Hello world\" --augmented=\"Hello there\" --metric=character\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --original-file=large.txt --augmented-file=large_mod.txt --streaming --optimize-speed\n", os.Args[0])
	}
}

func main() {
	// Parse command-line flags
	flag.Parse()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Validate inputs
	if err := validateInputs(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// Load inputs
	original, augmented, err := loadInputs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading inputs: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Println("=== Similarity Comparison ===")
		fmt.Printf("Original (%d chars): %s\n", len(original), truncateText(original, 100))
		fmt.Printf("Augmented (%d chars): %s\n", len(augmented), truncateText(augmented, 100))
	}

	// Process based on configuration
	switch metric {
	case "length":
		processLengthSimilarity(ctx, original, augmented)
	case "character":
		processCharacterSimilarity(ctx, original, augmented)
	case "both":
		processLengthSimilarity(ctx, original, augmented)
		processCharacterSimilarity(ctx, original, augmented)
	}
}

// validateInputs validates the command-line inputs
func validateInputs() error {
	// Check that we have at least one input method
	hasFileInput := originalFile != "" && augmentedFile != ""
	hasTextInput := originalText != "" && augmentedText != ""

	if !hasFileInput && !hasTextInput {
		return fmt.Errorf("must provide either file inputs or direct text inputs")
	}

	// Validate metric choice
	validMetrics := map[string]bool{
		"length":    true,
		"character": true,
		"both":      true,
	}

	if !validMetrics[metric] {
		return fmt.Errorf("invalid metric: %s. Must be 'length', 'character', or 'both'", metric)
	}

	// Validate threshold
	if threshold < 0 || threshold > 1 {
		return fmt.Errorf("threshold must be between 0.0 and 1.0")
	}

	// Validate maxDiffRatio
	if maxDiffRatio <= 0 {
		return fmt.Errorf("max-diff-ratio must be greater than 0")
	}

	// Validate streaming mode
	if useStreaming {
		validModes := map[string]bool{
			"chunk": true,
			"line":  true,
			"word":  true,
		}

		if !validModes[streamingMode] {
			return fmt.Errorf("invalid streaming mode: %s. Must be 'chunk', 'line', or 'word'", streamingMode)
		}
	}

	// Validate output format
	validFormats := map[string]bool{
		"text": true,
		"json": true,
	}

	if !validFormats[outputFormat] {
		return fmt.Errorf("invalid output format: %s. Must be 'text' or 'json'", outputFormat)
	}

	return nil
}

// loadInputs loads the input texts from files or direct input
func loadInputs() (string, string, error) {
	// If we have file inputs, use those
	if originalFile != "" && augmentedFile != "" {
		// Read original file
		origBytes, err := ioutil.ReadFile(originalFile)
		if err != nil {
			return "", "", fmt.Errorf("error reading original file: %v", err)
		}

		// Read augmented file
		augBytes, err := ioutil.ReadFile(augmentedFile)
		if err != nil {
			return "", "", fmt.Errorf("error reading augmented file: %v", err)
		}

		return string(origBytes), string(augBytes), nil
	}

	// Otherwise use direct text inputs
	return originalText, augmentedText, nil
}

// processLengthSimilarity calculates and outputs length similarity
func processLengthSimilarity(ctx context.Context, original, augmented string) {
	if useStreaming && len(original) > 10000 {
		// Use streaming for large inputs
		processStreamingSimilarity(ctx, original, augmented, "Length similarity")
		return
	}

	// Configure length similarity options
	var opts []word.LengthSimilarityOption
	opts = append(opts, word.WithThreshold(threshold))
	opts = append(opts, word.WithMaxDiffRatio(maxDiffRatio))

	if optimizeSpeed {
		opts = append(opts, word.WithFastNormalizer())
	}

	// Initialize the calculator
	ls, err := word.New(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing length similarity: %v\n", err)
		os.Exit(1)
	}

	// Compute similarity
	startTime := time.Now()
	result := ls.Compute(ctx, original, augmented)
	duration := time.Since(startTime)

	// Output results
	outputResult("Length Similarity", result, duration)
}

// processCharacterSimilarity calculates and outputs character similarity
func processCharacterSimilarity(ctx context.Context, original, augmented string) {
	if useStreaming && len(original) > 10000 {
		// Use streaming for large inputs
		processStreamingSimilarity(ctx, original, augmented, "Character similarity")
		return
	}

	// Configure character similarity options
	var opts []character.CharacterSimilarityOption
	opts = append(opts, character.WithThreshold(threshold))
	opts = append(opts, character.WithMaxDiffRatio(maxDiffRatio))

	if optimizeSpeed {
		opts = append(opts, character.WithOptimizedNormalizer())
	}

	// Initialize the calculator
	cs, err := character.NewCharacterSimilarity(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing character similarity: %v\n", err)
		os.Exit(1)
	}

	// Compute similarity
	startTime := time.Now()
	result := cs.Compute(ctx, original, augmented)
	duration := time.Since(startTime)

	// Output results
	outputResult("Character Similarity", result, duration)
}

// processStreamingSimilarity uses streaming processing for similarity calculation
func processStreamingSimilarity(ctx context.Context, original, augmented, title string) {
	// Determine streaming mode
	var mode streaming.StreamingMode
	switch streamingMode {
	case "chunk":
		mode = streaming.ChunkByChunk
	case "word":
		mode = streaming.WordByWord
	default:
		mode = streaming.LineByLine
	}

	// Configure streaming options
	var opts []streaming.StreamingOption
	opts = append(opts, streaming.WithStreamingThreshold(threshold))
	opts = append(opts, streaming.WithStreamingMaxDiffRatio(maxDiffRatio))
	opts = append(opts, streaming.WithStreamingMode(mode))

	if optimizeSpeed {
		opts = append(opts, streaming.WithOptimizedNormalizer())
	}

	// Initialize streaming similarity
	ss, err := streaming.NewStreamingSimilarity(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing streaming similarity: %v\n", err)
		os.Exit(1)
	}

	// Compute similarity
	startTime := time.Now()
	result := ss.ComputeFromStrings(ctx, original, augmented)
	duration := time.Since(startTime)

	// Output results
	outputStreamingResult(title+" (streaming)", result, duration)
}

// outputResult formats and outputs the similarity result
func outputResult(title string, result domain.Result, duration time.Duration) {
	if outputFormat == "json" {
		// Output JSON format
		fmt.Printf("{\n")
		fmt.Printf("  \"metric\": \"%s\",\n", title)
		fmt.Printf("  \"score\": %.4f,\n", result.Score)
		fmt.Printf("  \"passed\": %v,\n", result.Passed)
		fmt.Printf("  \"original_length\": %d,\n", result.OriginalLength)
		fmt.Printf("  \"augmented_length\": %d,\n", result.AugmentedLength)
		fmt.Printf("  \"length_ratio\": %.4f,\n", result.LengthRatio)
		fmt.Printf("  \"threshold\": %.4f,\n", result.Threshold)
		fmt.Printf("  \"duration_ms\": %.2f\n", float64(duration.Microseconds())/1000)
		fmt.Printf("}\n")
	} else {
		// Output text format
		fmt.Printf("\n=== %s ===\n", title)
		fmt.Printf("Score: %.4f\n", result.Score)
		fmt.Printf("Result: %s\n", getPassFailString(result.Passed))
		fmt.Printf("Original length: %d\n", result.OriginalLength)
		fmt.Printf("Augmented length: %d\n", result.AugmentedLength)
		fmt.Printf("Length ratio: %.4f\n", result.LengthRatio)
		fmt.Printf("Threshold: %.4f\n", result.Threshold)
		fmt.Printf("Processing time: %.2f ms\n", float64(duration.Microseconds())/1000)
	}
}

// outputStreamingResult formats and outputs streaming similarity results
func outputStreamingResult(title string, result streaming.StreamResult, duration time.Duration) {
	if outputFormat == "json" {
		// Output JSON format
		fmt.Printf("{\n")
		fmt.Printf("  \"metric\": \"%s\",\n", title)
		fmt.Printf("  \"score\": %.4f,\n", result.Score)
		fmt.Printf("  \"passed\": %v,\n", result.Passed)
		fmt.Printf("  \"original_length\": %d,\n", result.OriginalLength)
		fmt.Printf("  \"augmented_length\": %d,\n", result.AugmentedLength)
		fmt.Printf("  \"length_ratio\": %.4f,\n", result.LengthRatio)
		fmt.Printf("  \"threshold\": %.4f,\n", result.Threshold)
		fmt.Printf("  \"bytes_processed\": %d,\n", result.BytesProcessed)
		fmt.Printf("  \"duration_ms\": %.2f\n", float64(duration.Microseconds())/1000)
		fmt.Printf("}\n")
	} else {
		// Output text format
		fmt.Printf("\n=== %s ===\n", title)
		fmt.Printf("Score: %.4f\n", result.Score)
		fmt.Printf("Result: %s\n", getPassFailString(result.Passed))
		fmt.Printf("Original length: %d\n", result.OriginalLength)
		fmt.Printf("Augmented length: %d\n", result.AugmentedLength)
		fmt.Printf("Length ratio: %.4f\n", result.LengthRatio)
		fmt.Printf("Threshold: %.4f\n", result.Threshold)
		fmt.Printf("Bytes processed: %d\n", result.BytesProcessed)
		fmt.Printf("Processing time: %.2f ms\n", float64(duration.Microseconds())/1000)
	}
}

// getPassFailString returns a human-readable pass/fail string
func getPassFailString(passed bool) string {
	if passed {
		return "PASS"
	}
	return "FAIL"
}

// truncateText truncates text to the specified length and adds ellipsis if needed
func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}

/*
Sample usage:

1. Basic comparison using files:
   ./similarity --original-file=original.txt --augmented-file=augmented.txt

2. Direct text comparison:
   ./similarity --original="The quick brown fox jumps over the lazy dog." --augmented="The swift brown fox leaps over the sleepy dog." --metric=character

3. Performance-optimized streaming for large files:
   ./similarity --original-file=large.txt --augmented-file=large_mod.txt --streaming --optimize-speed --streaming-mode=line

4. Combined metrics with JSON output:
   ./similarity --original-file=doc.txt --augmented-file=doc_revised.txt --metric=both --output=json

5. Custom threshold and detailed output:
   ./similarity --original-file=doc.txt --augmented-file=doc_revised.txt --threshold=0.85 --verbose

6. Very strict comparison (high threshold):
   ./similarity --original-file=report.txt --augmented-file=report_updated.txt --threshold=0.95 --metric=length

7. More lenient comparison with custom difference ratio:
   ./similarity --original-file=essay.txt --augmented-file=essay_edited.txt --threshold=0.6 --max-diff-ratio=0.5

8. Word-by-word streaming comparison for large documents:
   ./similarity --original-file=book.txt --augmented-file=book_revised.txt --streaming --streaming-mode=word --optimize-speed

9. Chunk-by-chunk streaming for binary-like content:
   ./similarity --original-file=data.bin --augmented-file=data_modified.bin --streaming --streaming-mode=chunk --chunk-size=4096

10. Compare files with performance optimization and detailed output:
    ./similarity --original-file=large_report.txt --augmented-file=large_report_v2.txt --optimize-speed --verbose

11. Use both metrics with different thresholds in JSON format:
    ./similarity --original-file=paper.txt --augmented-file=paper_reviewed.txt --metric=both --threshold=0.75 --output=json

12. Simple character-based comparison between short texts:
    ./similarity --original="Hello, world!" --augmented="Hello, universe!" --metric=character

13. Performance optimized comparison with all details:
    ./similarity --original-file=code.txt --augmented-file=code_new.txt --optimize-speed --verbose --output=json

14. Streaming with custom threshold for specific content type:
    ./similarity --original-file=transcript.txt --augmented-file=transcript_edited.txt --streaming --threshold=0.82 --streaming-mode=line

15. Advanced configuration with all custom parameters:
    ./similarity --original-file=document.txt --augmented-file=document_revised.txt --metric=both --threshold=0.85 --max-diff-ratio=0.4 --streaming --streaming-mode=word --optimize-speed --output=json --verbose
*/
