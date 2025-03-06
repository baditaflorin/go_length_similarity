// Fix for benchmark/benchmark_test.go
// To address the issues in the benchmarking tests

package benchmark

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/baditaflorin/go_length_similarity/internal/adapters/normalizer"
	"github.com/baditaflorin/go_length_similarity/pkg/streaming"
	"github.com/baditaflorin/go_length_similarity/pkg/word"
)

// generateText creates a text of the specified size by repeating a sample text
func generateText(size int) string {
	// Return empty string for size 0
	if size <= 0 {
		return ""
	}

	sample := "The quick brown fox jumps over the lazy dog. This sentence contains all letters of the English alphabet and is commonly used for testing text processing algorithms and systems."
	var sb strings.Builder
	sb.Grow(size)

	for sb.Len() < size {
		sb.WriteString(sample)
		sb.WriteString(" ")
	}

	// Ensure we don't return more than requested size
	if sb.Len() > size {
		return sb.String()[:size]
	}

	return sb.String()
}

// SafeReader wraps a strings.Reader with protection against token-too-long errors
type SafeReader struct {
	reader *strings.Reader
	maxLen int
}

func NewSafeReader(s string, maxLen int) *SafeReader {
	return &SafeReader{
		reader: strings.NewReader(s),
		maxLen: maxLen,
	}
}

func (sr *SafeReader) Read(p []byte) (n int, err error) {
	// Limit read size to prevent token-too-long issues
	if len(p) > sr.maxLen {
		p = p[:sr.maxLen]
	}
	return sr.reader.Read(p)
}

// Implementing other io.Reader methods
func (sr *SafeReader) Seek(offset int64, whence int) (int64, error) {
	return sr.reader.Seek(offset, whence)
}

func (sr *SafeReader) WriteTo(w io.Writer) (n int64, err error) {
	return sr.reader.WriteTo(w)
}

// BenchmarkNormalizers compares the performance of different normalizers
func BenchmarkNormalizers(b *testing.B) {
	// Create text samples of different sizes
	smallText := generateText(100)    // 100 bytes
	mediumText := generateText(10000) // 10 KB
	largeText := generateText(100000) // 100 KB (reduced from 1MB to avoid memory issues)

	// Create the normalizer factory
	factory := normalizer.NewNormalizerFactory()

	// Define benchmark cases for each normalizer type
	benchmarks := []struct {
		name      string
		normType  normalizer.NormalizerType
		input     string
		inputSize string
	}{
		{"Default-Small", normalizer.DefaultNormalizerType, smallText, "100B"},
		{"Default-Medium", normalizer.DefaultNormalizerType, mediumText, "10KB"},
		{"Default-Large", normalizer.DefaultNormalizerType, largeText, "100KB"},

		{"Optimized-Small", normalizer.OptimizedNormalizerType, smallText, "100B"},
		{"Optimized-Medium", normalizer.OptimizedNormalizerType, mediumText, "10KB"},
		{"Optimized-Large", normalizer.OptimizedNormalizerType, largeText, "100KB"},

		{"Fast-Small", normalizer.FastNormalizerType, smallText, "100B"},
		{"Fast-Medium", normalizer.FastNormalizerType, mediumText, "10KB"},
		{"Fast-Large", normalizer.FastNormalizerType, largeText, "100KB"},
	}

	// Run benchmarks
	for _, bm := range benchmarks {
		norm := factory.CreateNormalizer(bm.normType)

		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(bm.input)))

			for i := 0; i < b.N; i++ {
				_ = norm.Normalize(bm.input)
			}
		})
	}
}

// BenchmarkLengthSimilarity benchmarks the length similarity with different configurations
func BenchmarkLengthSimilarity(b *testing.B) {
	// Create text samples
	original := generateText(10000)                      // 10 KB
	similar := strings.Replace(original, "the", "a", 10) // Similar with minor changes
	different := generateText(5000)                      // 5 KB

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Benchmark standard configuration
	b.Run("Standard", func(b *testing.B) {
		ls, _ := word.New()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = ls.Compute(ctx, original, similar)
		}
	})

	// Benchmark with FastNormalizer
	b.Run("FastNormalizer", func(b *testing.B) {
		ls, _ := word.New(
			word.WithFastNormalizer(),
		)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = ls.Compute(ctx, original, similar)
		}
	})

	// Benchmark with WarmUp
	b.Run("WithWarmUp", func(b *testing.B) {
		ls, _ := word.New(
			word.WithFastNormalizer(),
			word.WithWarmUp(true),
		)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = ls.Compute(ctx, original, similar)
		}
	})

	// Benchmark different similarity levels
	b.Run("Similar", func(b *testing.B) {
		ls, _ := word.New(word.WithFastNormalizer())
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = ls.Compute(ctx, original, similar)
		}
	})

	b.Run("Different", func(b *testing.B) {
		ls, _ := word.New(word.WithFastNormalizer())
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = ls.Compute(ctx, original, different)
		}
	})
}

// BenchmarkStreamingSimilarity benchmarks the streaming similarity implementation
func BenchmarkStreamingSimilarity(b *testing.B) {
	// Create text samples
	original := generateText(100000)                      // 100 KB (reduced from 1MB)
	similar := strings.Replace(original, "the", "a", 100) // Similar with changes

	// Common timeout for all tests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Benchmark different streaming modes
	modes := []struct {
		name string
		mode streaming.StreamingMode
	}{
		{"ChunkByChunk", streaming.ChunkByChunk},
		{"LineByLine", streaming.LineByLine},
		{"WordByWord", streaming.WordByWord},
	}

	for _, mode := range modes {
		b.Run(mode.name, func(b *testing.B) {
			ss, _ := streaming.NewStreamingSimilarity(
				streaming.WithStreamingMode(mode.mode),
				streaming.WithOptimizedNormalizer(),
				streaming.WithStreamingChunkSize(8192), // Use a reasonable chunk size
			)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Use SafeReader to prevent token-too-long errors
				originalReader := NewSafeReader(original, 8192)
				similarReader := NewSafeReader(similar, 8192)
				_ = ss.ComputeFromReaders(ctx, originalReader, similarReader)
			}
		})
	}

	// Benchmark different chunk sizes
	chunkSizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"8KB", 8192},
		{"32KB", 32768}, // Reduced from 64KB
	}

	for _, cs := range chunkSizes {
		b.Run("ChunkSize-"+cs.name, func(b *testing.B) {
			ss, _ := streaming.NewStreamingSimilarity(
				streaming.WithStreamingMode(streaming.ChunkByChunk),
				streaming.WithStreamingChunkSize(cs.size),
				streaming.WithOptimizedNormalizer(),
			)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				originalReader := strings.NewReader(original)
				similarReader := strings.NewReader(similar)
				_ = ss.ComputeFromReaders(ctx, originalReader, similarReader)
			}
		})
	}

	// Add test for empty input
	b.Run("EmptyInput", func(b *testing.B) {
		ss, _ := streaming.NewStreamingSimilarity(
			streaming.WithStreamingMode(streaming.ChunkByChunk),
			streaming.WithOptimizedNormalizer(),
		)
		b.ResetTimer()

		emptyText := ""
		for i := 0; i < b.N; i++ {
			originalReader := strings.NewReader(emptyText)
			similarReader := strings.NewReader(emptyText)
			_ = ss.ComputeFromReaders(ctx, originalReader, similarReader)
		}
	})
}
