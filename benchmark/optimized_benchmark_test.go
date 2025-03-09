// File: benchmark/optimized_benchmark_test.go
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

// mockLogger (re-implement just to be sure it's available)
//type mockLogger struct{}
//
//func (l *mockLogger) Debug(msg string, keysAndValues ...interface{}) {}
//func (l *mockLogger) Info(msg string, keysAndValues ...interface{})  {}
//func (l *mockLogger) Warn(msg string, keysAndValues ...interface{})  {}
//func (l *mockLogger) Error(msg string, keysAndValues ...interface{}) {}
//func (l *mockLogger) Close() error                                   { return nil }

// BenchmarkOptimizedLineProcessing compares our new allocation-efficient line processing
func BenchmarkOptimizedLineProcessing(b *testing.B) {
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

	// Create normalizer factory and normalizers
	normFactory := normalizer.NewNormalizerFactory()
	optimizedNorm := normFactory.CreateNormalizer(normalizer.OptimizedNormalizerType)

	// Create an allocation-efficient normalizer
	allocationEfficientNorm := normFactory.CreateAllocationEfficientNormalizer()

	// Create processors
	// 1. Original processor (using DefaultProcessor with old implementation)
	originalProc := stream.NewDefaultProcessor(logger, optimizedNorm)

	// 2. Standard optimized line processor (non-parallel)
	optimizedProc := lineprocessor.NewProcessor(logger, optimizedNorm, lineprocessor.ProcessingConfig{
		ChunkSize:   64 * 1024,
		BatchSize:   100,
		UseParallel: false,
	})

	// 3. Optimized processor with allocation-efficient normalizer
	allocationEfficientProc := lineprocessor.NewOptimizedProcessor(logger, allocationEfficientNorm.(ports.Normalizer), lineprocessor.ProcessingConfig{
		ChunkSize:   64 * 1024,
		BatchSize:   100,
		UseParallel: false,
	})

	// 4. Parallel allocation-efficient processor
	parallelAllocationEfficientProc := lineprocessor.NewOptimizedProcessor(logger, allocationEfficientNorm.(ports.Normalizer), lineprocessor.ProcessingConfig{
		ChunkSize:   64 * 1024,
		BatchSize:   100,
		UseParallel: true,
	})

	// Define benchmark cases
	benchmarks := []struct {
		name      string
		proc      interface{} // Either DefaultProcessor, Processor, or OptimizedProcessor
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

		{"AllocationEfficient-Small-Lines", allocationEfficientProc, smallLines, "50 lines"},
		{"AllocationEfficient-Medium-Lines", allocationEfficientProc, mediumLines, "500 lines"},
		{"AllocationEfficient-Large-Lines", allocationEfficientProc, largeLines, "5000 lines"},
		{"AllocationEfficient-Mixed-Lines", allocationEfficientProc, mixedLines, "500 mixed lines"},

		{"ParallelAllocationEfficient-Small-Lines", parallelAllocationEfficientProc, smallLines, "50 lines"},
		{"ParallelAllocationEfficient-Medium-Lines", parallelAllocationEfficientProc, mediumLines, "500 lines"},
		{"ParallelAllocationEfficient-Large-Lines", parallelAllocationEfficientProc, largeLines, "5000 lines"},
		{"ParallelAllocationEfficient-Mixed-Lines", parallelAllocationEfficientProc, mixedLines, "500 mixed lines"},
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
					// Use the original implementation
					_, err := p.ProcessStream(ctx, reader, ports.LineByLine)
					if err != nil && err != io.EOF {
						b.Fatalf("Error processing: %v", err)
					}

				case *lineprocessor.Processor:
					// Use the current optimized processor
					_, _, err := p.ProcessLines(ctx, reader, nil)
					if err != nil && err != io.EOF {
						b.Fatalf("Error processing: %v", err)
					}

				case *lineprocessor.OptimizedProcessor:
					// Use the allocation-efficient processor
					_, _, err := p.ProcessLines(ctx, reader, nil)
					if err != nil && err != io.EOF {
						b.Fatalf("Error processing: %v", err)
					}
				}
			}
		})
	}

	// Run a special benchmark to measure allocation impact of different normalizers
	b.Run("NormalizerComparison", func(b *testing.B) {
		// Subsection for normalizer comparison
		normalizers := []struct {
			name string
			norm ports.Normalizer
		}{
			{"Default", normFactory.CreateNormalizer(normalizer.DefaultNormalizerType)},
			{"Optimized", normFactory.CreateNormalizer(normalizer.OptimizedNormalizerType)},
			{"Fast", normFactory.CreateNormalizer(normalizer.FastNormalizerType)},
			{"AllocationEfficient", allocationEfficientNorm.(ports.Normalizer)},
		}

		for _, norm := range normalizers {
			b.Run(norm.name, func(b *testing.B) {
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					// Normalize a medium-sized text
					_ = norm.norm.Normalize(mediumLines)
				}
			})
		}
	})
}
