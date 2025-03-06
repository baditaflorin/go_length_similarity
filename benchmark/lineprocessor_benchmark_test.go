package benchmark

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/baditaflorin/go_length_similarity/internal/adapters/normalizer"
	"github.com/baditaflorin/go_length_similarity/internal/adapters/stream"
	"github.com/baditaflorin/go_length_similarity/internal/adapters/stream/lineprocessor"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

// mockLogger implements a minimal logger for testing
// (Defined here rather than relying on wordprocessor_benchmark_test.go)
type mockLogger struct{}

func (l *mockLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (l *mockLogger) Info(msg string, keysAndValues ...interface{})  {}
func (l *mockLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (l *mockLogger) Error(msg string, keysAndValues ...interface{}) {}
func (l *mockLogger) Close() error                                   { return nil }

// generateLineTestText creates a text sample optimized for line processing benchmarks
func generateLineTestText(lineCount int, avgLineLen int) string {
	// Sample sentences for generating lines
	sentences := []string{
		"The quick brown fox jumps over the lazy dog.",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
		"Sed ut perspiciatis unde omnis iste natus error sit voluptatem.",
		"At vero eos et accusamus et iusto odio dignissimos ducimus.",
		"Excepteur sint occaecat cupidatat non proident.",
		"Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore.",
		"Hello world, this is a test sentence with some typical content.",
		"Programming languages like Go provide excellent performance characteristics.",
		"Each line in this test will have a reasonable but varying length.",
		"Performance optimization is key to building efficient software systems.",
	}

	var sb strings.Builder
	totalLen := lineCount * (avgLineLen + 1) // Add newline after each line
	sb.Grow(totalLen)

	for i := 0; i < lineCount; i++ {
		if i > 0 {
			sb.WriteString("\n")
		}

		// Select a sentence from the list
		sentence := sentences[i%len(sentences)]

		// Add some variation to line length by repeating parts
		repetitions := (i % 3) + 1
		for j := 0; j < repetitions; j++ {
			if j > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(sentence)
		}
	}

	return sb.String()
}

// generateMixedLineText creates a text with a mix of short and long lines
func generateMixedLineText(lineCount int) string {
	shortLines := []string{
		"Short line.",
		"Another short one.",
		"Brevity is key.",
		"Small text.",
		"Concise.",
	}

	longLines := []string{
		"This is a much longer line that contains significantly more text to process. It should have a different memory and processing pattern compared to short lines, which helps to test the efficiency of line processing algorithms under varying conditions. The line length should be substantially greater to ensure we're testing a different performance profile.",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.",
		"Performance testing is essential to identify bottlenecks and optimize code execution. Long lines like this one test how efficiently the processor handles larger chunks of text in a single line, which can be particularly challenging for line-by-line processing algorithms that need to buffer entire lines before processing them. Efficient memory management becomes critical in these scenarios.",
	}

	var sb strings.Builder
	sb.Grow(lineCount * 100) // Rough estimate

	for i := 0; i < lineCount; i++ {
		if i > 0 {
			sb.WriteString("\n")
		}

		// Alternate between short and long lines, with short lines being more common
		if i%4 == 0 {
			// Long line (every 4th line)
			sb.WriteString(longLines[i%len(longLines)])
		} else {
			// Short line (3 out of 4 lines)
			sb.WriteString(shortLines[i%len(shortLines)])
		}
	}

	return sb.String()
}

// BenchmarkLineProcessing runs benchmarks comparing different line processing implementations
func BenchmarkLineProcessing(b *testing.B) {
	// Create test samples of different sizes
	smallLines := generateLineTestText(50, 60)   // ~50 lines, ~3KB
	mediumLines := generateLineTestText(500, 60) // ~500 lines, ~30KB
	largeLines := generateLineTestText(5000, 60) // ~5000 lines, ~300KB
	mixedLines := generateMixedLineText(500)     // ~500 mixed lines, ~30-50KB

	// Context for processing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create logger
	logger := &mockLogger{}

	// Create normalizer
	normFactory := normalizer.NewNormalizerFactory()
	norm := normFactory.CreateNormalizer(normalizer.OptimizedNormalizerType)

	// Create processors
	// 1. Original processor (using DefaultProcessor with old implementation)
	originalProc := stream.NewDefaultProcessor(logger, norm)

	// 2. Optimized line processor (non-parallel)
	optimizedProc := lineprocessor.NewProcessor(logger, norm, lineprocessor.ProcessingConfig{
		ChunkSize:   64 * 1024,
		BatchSize:   100,
		UseParallel: false,
	})

	// 3. Parallel line processor
	parallelProc := lineprocessor.NewProcessor(logger, norm, lineprocessor.ProcessingConfig{
		ChunkSize:   64 * 1024,
		BatchSize:   100,
		UseParallel: true,
	})

	// Define benchmark cases
	benchmarks := []struct {
		name      string
		proc      interface{} // Either DefaultProcessor or Processor
		input     string
		inputDesc string
	}{
		{"Original-Small-Lines", originalProc, smallLines, "50 lines"},
		{"Original-Medium-Lines", originalProc, mediumLines, "500 lines"},
		{"Original-Large-Lines", originalProc, largeLines, "5000 lines"},
		{"Original-Mixed-Lines", originalProc, mixedLines, "500 mixed lines"},

		{"Optimized-Small-Lines", optimizedProc, smallLines, "50 lines"},
		{"Optimized-Medium-Lines", optimizedProc, mediumLines, "500 lines"},
		{"Optimized-Large-Lines", optimizedProc, largeLines, "5000 lines"},
		{"Optimized-Mixed-Lines", optimizedProc, mixedLines, "500 mixed lines"},

		{"Parallel-Small-Lines", parallelProc, smallLines, "50 lines"},
		{"Parallel-Medium-Lines", parallelProc, mediumLines, "500 lines"},
		{"Parallel-Large-Lines", parallelProc, largeLines, "5000 lines"},
		{"Parallel-Mixed-Lines", parallelProc, mixedLines, "500 mixed lines"},
	}

	// Run benchmarks
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(bm.input)))

			for i := 0; i < b.N; i++ {
				// Create a fresh reader for each iteration
				reader := strings.NewReader(bm.input)

				// Process based on the processor type
				switch p := bm.proc.(type) {
				case *stream.DefaultProcessor:
					// Use the original implementation with processLines
					_, err := p.ProcessStream(ctx, reader, ports.LineByLine)
					if err != nil && err != io.EOF {
						b.Fatalf("Error processing: %v", err)
					}
				case *lineprocessor.Processor:
					// Use the optimized line processor
					_, _, err := p.ProcessLines(ctx, reader, nil)
					if err != nil && err != io.EOF {
						b.Fatalf("Error processing: %v", err)
					}
				}
			}
		})
	}
}
