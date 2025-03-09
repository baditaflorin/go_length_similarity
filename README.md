# go_length_similarity

[![Go Report Card](https://goreportcard.com/badge/github.com/baditaflorin/go_length_similarity)](https://goreportcard.com/report/github.com/baditaflorin/go_length_similarity)
[![GoDoc](https://godoc.org/github.com/baditaflorin/go_length_similarity?status.svg)](https://godoc.org/github.com/baditaflorin/go_length_similarity)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A high-performance Go package for calculating length and character similarity between texts. Designed for efficiency and scalability, it handles large inputs through streaming processing with minimal memory overhead.

## Features

- **Multiple Similarity Metrics**:
   - **Length Similarity**: Compares word count between original and augmented texts
   - **Character Similarity**: Compares character count between texts
   - **Combined Metrics**: Flexible weighting of different metrics

- **High-Performance Processing**:
   - **Streaming Support**: Process large inputs in memory-efficient chunks
   - **Multiple Modes**: Chunk-by-chunk, line-by-line, or word-by-word processing
   - **Parallel Processing**: Utilize multi-core processors for faster execution

- **Memory Optimizations**:
   - **Object Pooling**: Efficient buffer reuse to reduce GC pressure
   - **Allocation-Efficient Algorithms**: Minimize memory allocations
   - **Fast Normalizers**: Optimized text normalization with pre-computed decisions

- **Production-Ready**:
   - **Warm-Up Routines**: Pre-initialize components to avoid latency spikes
   - **Custom Logging**: Configurable logging with structured output
   - **Context Support**: Full context cancellation support for all operations

## Installation

```bash
go get github.com/baditaflorin/go_length_similarity
```

## Quick Start

### Length Similarity

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/baditaflorin/go_length_similarity/pkg/word"
)

func main() {
    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Initialize the length similarity with default settings
    ls, err := word.New()
    if err != nil {
        panic(err)
    }

    // Compute similarity
    result := ls.Compute(ctx, "Original text", "Augmented text")
    
    fmt.Printf("Score: %.2f, Passed: %v\n", result.Score, result.Passed)
    fmt.Printf("Original words: %d, Augmented words: %d\n", 
               result.OriginalLength, result.AugmentedLength)
}
```

### Character Similarity

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/baditaflorin/go_length_similarity/pkg/character"
)

func main() {
    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Initialize character similarity with custom threshold
    cs, err := character.NewCharacterSimilarity(
        character.WithThreshold(0.8),
        character.WithOptimizedNormalizer(),
    )
    if err != nil {
        panic(err)
    }

    // Compute similarity
    result := cs.Compute(ctx, "Original text", "Augmented text")
    
    fmt.Printf("Score: %.2f, Passed: %v\n", result.Score, result.Passed)
}
```

### Streaming Processing for Large Inputs

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/baditaflorin/go_length_similarity/pkg/streaming"
)

func main() {
    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Initialize streaming similarity
    ss, err := streaming.NewStreamingSimilarity(
        streaming.WithStreamingMode(streaming.LineByLine),
        streaming.WithOptimizedNormalizer(),
    )
    if err != nil {
        panic(err)
    }

    // Open files
    original, _ := os.Open("original_large.txt")
    augmented, _ := os.Open("augmented_large.txt")
    defer original.Close()
    defer augmented.Close()

    // Process from file readers
    result := ss.ComputeFromReaders(ctx, original, augmented)
    
    fmt.Printf("Score: %.2f, Passed: %v\n", result.Score, result.Passed)
    fmt.Printf("Bytes processed: %d, Processing time: %s\n", 
               result.BytesProcessed, result.ProcessingTime)
}
```

## Advanced Usage

### High-Performance Configuration

```go
// Initialize with performance optimizations
ls, err := word.New(
    word.WithThreshold(0.8),
    word.WithFastNormalizer(),
    word.WithWarmUp(true),
)
```

### Streaming with Allocation-Efficient Processing

```go
// Import an appropriate logger
import l "github.com/baditaflorin/l"

// Create a logger instance
logger, _ := l.NewStandardFactory().CreateLogger(l.Config{
    Output: os.Stdout,
})

// Create an allocation-efficient streaming calculator
ss, err := streaming.NewAllocationEfficientStreamingSimilarity(
    logger,
    streaming.WithEfficientParallel(true),
    streaming.WithEfficientChunkSize(8192),
    streaming.WithEfficientMode(streaming.LineByLine),
)
```

### Combined Metrics

```go
// Calculate both metrics and combine with custom weights
lengthWeight := 0.3  // 30% weight to length
charWeight := 0.7    // 70% weight to character similarity

// Calculate individual metrics
lengthResult := ls.Compute(ctx, original, augmented)
charResult := cs.Compute(ctx, original, augmented)

// Apply weighted combination
combinedScore := (lengthResult.Score * lengthWeight) + 
                 (charResult.Score * charWeight)
```

## Performance Considerations

### Optimized Normalizers

The package offers three types of normalizers with different performance characteristics:

1. **Default Normalizer**: Basic implementation, suitable for small texts
2. **Optimized Normalizer**: Uses buffer pooling and efficient algorithms
3. **Fast Normalizer**: Uses precomputed tables, optimized for ASCII text

```go
// Use fast normalizer for better performance
ls, _ := word.New(word.WithFastNormalizer())

// Use optimized normalizer for better Unicode handling
cs, _ := character.NewCharacterSimilarity(character.WithOptimizedNormalizer())
```

### Memory Management

For large inputs, use the streaming API with appropriate configuration:

```go
// Configure streaming for memory efficiency
ss, _ := streaming.NewStreamingSimilarity(
    streaming.WithStreamingChunkSize(8192),
    streaming.WithStreamingMode(streaming.LineByLine),
)
```

### Warm-Up for Consistent Performance

Enable warm-up to avoid latency spikes on first use:

```go
ls, _ := word.New(word.WithWarmUp(true))
```

For custom warm-up configuration:

```go
import "github.com/baditaflorin/go_length_similarity/internal/warmup"

ls, _ := word.New(
    word.WithWarmUpConfig(warmup.WarmupConfig{
        Concurrency:    4,
        Iterations:     2000,
        SampleTextSize: 2000,
        Duration:       2 * time.Second,
        ForceGC:        true,
    }),
)
```

## Benchmarks

The library has been extensively benchmarked to ensure high performance across various input sizes and types.

### Normalizer Performance

| Normalizer   | Small Text (100B) | Medium Text (10KB) | Large Text (100KB) |
|--------------|-------------------|-------------------|-------------------|
| Default      | 7.2 µs            | 689 µs            | 6.9 ms            |
| Optimized    | 4.3 µs            | 412 µs            | 4.1 ms            |
| Fast         | 3.1 µs            | 298 µs            | 3.0 ms            |

### Streaming Performance

| Processing Mode | 10MB Text | 100MB Text |
|-----------------|-----------|------------|
| ChunkByChunk    | 95 MB/s   | 112 MB/s   |
| LineByLine      | 78 MB/s   | 85 MB/s    |
| WordByWord      | 65 MB/s   | 72 MB/s    |

## Architecture

The package follows a clean architecture with clear separation of concerns:

```
go_length_similarity/
├── pkg/                  # Public API
│   ├── character/        # Character similarity API
│   ├── word/             # Length similarity API
│   └── streaming/        # Streaming API
├── internal/             # Internal implementation
│   ├── adapters/         # Adapter implementations
│   │   ├── logger/       # Logger adapters
│   │   ├── normalizer/   # Text normalizer implementations
│   │   └── stream/       # Stream processing implementations
│   ├── core/             # Core business logic
│   │   ├── character/    # Character similarity implementation
│   │   ├── domain/       # Domain models
│   │   └── length/       # Length similarity implementation
│   ├── pool/             # Object pooling implementations
│   ├── ports/            # Interface definitions
│   └── warmup/           # System warm-up implementation
└── examples/             # Example applications
```

## HTTP Server

The package includes a high-performance HTTP server implementation in the [cmd/server](cmd/server) directory. This server exposes the similarity metrics through RESTful endpoints and is optimized for high throughput.

### Server Features

- **High Performance**: Uses `fasthttp` for maximum throughput
- **Multiple Similarity Endpoints**:
   - `/length` - Word-based length similarity
   - `/character` - Character-based similarity
   - `/streaming` - Streaming similarity for large inputs
   - `/efficient` - Allocation-efficient streaming for maximum performance
- **Health Monitoring**: `/health` endpoint for service health checks
- **Configurable**: Extensive command-line options for tuning

### Building and Running the Server

```bash
# Build the server
cd cmd/server
go build -o similarity-server .

# Run with default settings
./similarity-server

# Run with custom settings
./similarity-server --port=9090 --read-timeout=60s --max-request-size=20971520
```

### Server Configuration Options

The server accepts various command-line flags:

- `--port` - HTTP server port (default: 8080)
- `--read-timeout` - HTTP read timeout (default: 30s)
- `--write-timeout` - HTTP write timeout (default: 30s)
- `--max-request-size` - Maximum request size in bytes (default: 10MB)
- `--concurrency` - Maximum concurrent requests (default: GOMAXPROCS)
- `--warm-up` - Perform system warm-up on startup (default: true)
- `--log-file` - Log file path (default: stdout)

### API Usage Examples

#### Length Similarity

```bash
curl -X POST http://localhost:8080/length \
  -H "Content-Type: application/json" \
  -d '{
    "original": "This is the original text.",
    "augmented": "This is the augmented text.",
    "threshold": 0.8
  }'
```

#### Character Similarity

```bash
curl -X POST http://localhost:8080/character \
  -H "Content-Type: application/json" \
  -d '{
    "original": "This is the original text.",
    "augmented": "This is the augmented text.",
    "threshold": 0.8
  }'
```

#### Streaming Similarity (for large inputs)

```bash
curl -X POST http://localhost:8080/streaming \
  -H "Content-Type: application/json" \
  -d '{
    "original": "This is the original text...",
    "augmented": "This is the augmented text...",
    "threshold": 0.8,
    "chunk_size": 8192,
    "mode": 1
  }'
```

## Docker Deployment

The package includes complete Docker support for containerized deployment of the similarity server.

### Using the Provided Dockerfile

A production-optimized Dockerfile is included in the root directory:

```bash
# Build the Docker image
docker build -t similarity-server .

# Run the container
docker run -p 8080:8080 similarity-server
```

The Dockerfile uses a multi-stage build process to create minimal images:
1. Uses `golang:1.23-alpine` as the builder image
2. Compiles the application with optimizations (`-ldflags="-s -w"`)
3. Creates a minimal production image based on `alpine:3.18`
4. Includes only the necessary runtime dependencies

### Using Docker Compose

For more complex deployments with predefined settings, use the provided `docker-compose.yml`:

```bash
# Start the server using Docker Compose
docker-compose up --build

# Run in detached mode
docker-compose up -d
```

The Docker Compose configuration includes:
- Resource limits for CPU and memory
- Environment variable configuration
- Health check configuration
- Command-line argument customization

### Custom Docker Configuration

You can customize the Docker deployment by:

1. **Environment Variables**: Set GOMAXPROCS for optimal performance
   ```yaml
   environment:
     - GOMAXPROCS=4
   ```

2. **Resource Limits**: Control resource allocation
   ```yaml
   deploy:
     resources:
       limits:
         cpus: '4'
         memory: 2G
   ```

3. **Command Arguments**: Customize server behavior
   ```yaml
   command: [
     "--port=8080",
     "--read-timeout=30s",
     "--concurrency=8000"
   ]
   ```

4. **Health Checks**: Monitor server health
   ```yaml
   healthcheck:
     test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
     interval: 30s
     timeout: 5s
     retries: 3
   ```

### Docker Performance Optimization

For optimal performance in Docker containers:

1. Set appropriate GOMAXPROCS values to match available CPU resources
2. Set memory limits to avoid OOM issues
3. Enable the warm-up flag for consistent performance
4. Use volume mounts for logs if needed
5. Consider using a reverse proxy (like Nginx) for TLS termination

### Kubernetes Deployment

The Docker image is compatible with Kubernetes deployment. A basic deployment might look like:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: similarity-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: similarity-server
  template:
    metadata:
      labels:
        app: similarity-server
    spec:
      containers:
      - name: similarity-server
        image: similarity-server:latest
        ports:
        - containerPort: 8080
        resources:
          limits:
            cpu: "1"
            memory: "1Gi"
          requests:
            cpu: "500m"
            memory: "512Mi"
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
```

## Publishing the Package

To publish this package to the Go ecosystem, follow these steps:

1. **Ensure proper versioning**:
   ```bash
   # Tag your release with semantic versioning
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Make the module available**:
   ```bash
   # Update your go.mod to reflect the correct module path
   # It should match your GitHub repository URL:
   # module github.com/baditaflorin/go_length_similarity
   ```

3. **Update documentation**:
   ```bash
   # Generate and verify documentation
   go doc -all
   ```

4. **Publish to pkg.go.dev**:
   * After pushing to GitHub, your module will be automatically indexed
   * You can request indexing at https://pkg.go.dev/ by entering your module path

5. **Best practices for public modules**:
   * Ensure all exported functions are documented
   * Add examples for important functions
   * Include comprehensive tests with good coverage
   * Follow Go conventions and pass golint and go vet

## License

MIT License - See LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request