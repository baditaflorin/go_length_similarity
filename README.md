# go_length_similarity

A high-performance Go package for calculating length and character similarity between texts.

## Key Features

- **Length Similarity**: Compares the word count between original and augmented texts.
- **Character Similarity**: Compares the character count between original and augmented texts.
- **Streaming Processing**: Efficiently processes large inputs in configurable chunks.
- **Performance Optimizations**: Includes buffer pooling, optimized normalizers, and system warm-up.
- **Flexible Configuration**: Provides functional options for customizing behavior.

## Performance & Scalability Optimizations

1. **Streaming Processing for Large Inputs**
    - Process input in chunks to reduce memory overhead
    - Support different modes: chunk-by-chunk, line-by-line, word-by-word

2. **Optimized Memory Allocations**
    - Improved normalization algorithms with pre-allocated buffers
    - Fast normalizer with pre-computed decisions for ASCII characters

3. **Object Pooling for Reusable Buffers**
    - Buffer pools for byte slices, rune slices, and string builders
    - Reduces GC pressure and improves throughput

4. **Warm-up Routines for Performance Optimization**
    - Pre-initialize caches and pools to avoid latency spikes
    - Configurable warm-up with options for iterations and concurrency

## Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/baditaflorin/go_length_similarity/pkg"
)

func main() {
    // Initialize the length similarity with default settings
    ls, err := lengthsimilarity.New()
    if err != nil {
        panic(err)
    }

    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    // Compute similarity
    result := ls.Compute(ctx, "Original text", "Augmented text")
    
    fmt.Printf("Score: %.2f, Passed: %v\n", result.Score, result.Passed)
}
```

### High-Performance Configuration

```go
// Initialize with performance optimizations
ls, err := lengthsimilarity.New(
    lengthsimilarity.WithThreshold(0.8),
    lengthsimilarity.WithFastNormalizer(),
    lengthsimilarity.WithWarmUp(true),
)
```

### Streaming Processing for Large Inputs

```go
// Initialize streaming similarity
ss, err := lengthsimilarity.NewStreamingSimilarity(
    lengthsimilarity.WithStreamingMode(lengthsimilarity.LineByLine),
    lengthsimilarity.WithStreamingChunkSize(8192),
    lengthsimilarity.WithOptimizedNormalizer(),
)

// Process from readers (e.g., files)
result := ss.ComputeFromReaders(ctx, originalReader, augmentedReader)
```

## Running Benchmarks

```bash
go test -bench=. -benchmem ./benchmark/...
```

## Project Structure

```
go_length_similarity/
├── benchmark/                  # Performance benchmarks
├── examples/                   # Example applications
│   ├── CharacterSimilarity/    # Character similarity examples
│   ├── HighPerformance/        # High-performance examples
│   ├── LengthSimilarity/       # Length similarity examples
│   └── StreamingSimilarity/    # Streaming processing examples
├── internal/                   # Internal implementation
│   ├── adapters/               # Adapter implementations
│   │   ├── logger/             # Logger adapters
│   │   ├── normalizer/         # Text normalizer implementations
│   │   └── stream/             # Stream processing implementations
│   ├── core/                   # Core business logic
│   │   ├── character/          # Character similarity implementation
│   │   ├── domain/             # Domain models
│   │   └── length/             # Length similarity implementation
│   ├── pool/                   # Object pooling implementations
│   ├── ports/                  # Interface definitions
│   └── warmup/                 # System warm-up implementation
├── pkg/                        # Public API
│   ├── character_similarity.go # Character similarity API
│   ├── length_similarity.go    # Length similarity API
│   └── streaming.go            # Streaming API
└── README.md                   # Documentation
```

## License

MIT License - See LICENSE file for details