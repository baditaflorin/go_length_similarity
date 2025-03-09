# High-Performance Similarity Server

This project provides a high-performance HTTP server for text similarity calculations using the `go_length_similarity` library with `fasthttp`.

## Features

- **High Performance**: Uses `fasthttp` for maximum throughput
- **Multiple Similarity Metrics**: Length, character, and streaming similarity
- **Optimized Memory Usage**: Employs allocation-efficient algorithms
- **Configurable**: Tunable parameters for different workloads
- **Docker Support**: Easy deployment with Docker and Docker Compose
- **Benchmarking Tools**: Scripts for performance testing

## Server Architecture

The server exposes several endpoints:

- `/health` - Health check endpoint
- `/length` - Word-based length similarity
- `/character` - Character-based similarity
- `/streaming` - Streaming similarity for large inputs
- `/efficient` - Allocation-efficient streaming for maximum performance

## Getting Started

### Prerequisites

- Go 1.18 or higher
- Docker and Docker Compose (optional)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/similarity-server.git
   cd similarity-server
   ```

2. Build the server:
   ```bash
   go build -o similarity-server ./cmd/server
   ```

3. Run the server:
   ```bash
   ./similarity-server --port=8080
   ```

### Docker Deployment

Use Docker Compose for an optimized containerized deployment:

```bash
docker-compose up --build
```

## Configuration Options

The server accepts various command-line flags for tuning:

- `--port` - HTTP server port (default: 8080)
- `--read-timeout` - HTTP read timeout (default: 30s)
- `--write-timeout` - HTTP write timeout (default: 30s)
- `--max-request-size` - Maximum request size in bytes (default: 10MB)
- `--concurrency` - Maximum concurrent requests (default: GOMAXPROCS)
- `--warm-up` - Perform system warm-up on startup (default: true)
- `--log-file` - Log file path (default: stdout)

## Performance Tuning

For optimal performance:

1. Set GOMAXPROCS to match your CPU count
2. Enable warm-up for better initial performance
3. Tune concurrency based on your workload
4. Adjust read/write timeouts for your use case
5. Set appropriate max request size for your data

## API Usage

### Length Similarity

```bash
curl -X POST http://localhost:8080/length \
  -H "Content-Type: application/json" \
  -d '{
    "original": "This is the original text.",
    "augmented": "This is the augmented text.",
    "threshold": 0.8
  }'
```

### Character Similarity

```bash
curl -X POST http://localhost:8080/character \
  -H "Content-Type: application/json" \
  -d '{
    "original": "This is the original text.",
    "augmented": "This is the augmented text.",
    "threshold": 0.8
  }'
```

### Streaming Similarity (for large inputs)

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

The `mode` parameter can be:
- 0 = ChunkByChunk
- 1 = LineByLine
- 2 = WordByWord

### Allocation-Efficient Streaming (for maximum performance)

```bash
curl -X POST http://localhost:8080/efficient \
  -H "Content-Type: application/json" \
  -d '{
    "original": "This is the original text...",
    "augmented": "This is the augmented text...",
    "threshold": 0.8
  }'
```

## Benchmarking

Use the provided script to benchmark server performance:

```bash
./benchmark.sh [endpoint] [concurrency] [requests]
```

Example:
```bash
./benchmark.sh /efficient 200 50000
```

## Architecture Details

The server uses a clean architecture with several optimized components:

1. **FastHTTP Server**: High-performance HTTP server with optimized memory usage
2. **Optimized Similarity Calculators**:
    - Length similarity with fast normalizer
    - Character similarity with optimized normalizer
    - Streaming similarity for large inputs
    - Allocation-efficient streaming for maximum performance
3. **Efficient JSON Handling**: Minimize allocations during request parsing
4. **Parallel Processing**: Efficient use of goroutines for concurrent requests
5. **Memory Pooling**: Reuse of memory buffers to reduce GC pressure

## Production Deployment Recommendations

1. **Horizontal Scaling**: Deploy multiple instances behind a load balancer
2. **Resource Limits**: Set appropriate CPU and memory limits
3. **Monitoring**: Implement health checks and metrics collection
4. **Rate Limiting**: Protect against abuse with appropriate rate limits
5. **TLS Termination**: Use a reverse proxy for TLS termination

## Performance Metrics

On a standard 4-core machine, the server can handle:

- ~20,000 requests/second for length similarity
- ~15,000 requests/second for character similarity
- ~5,000 requests/second for standard streaming
- ~8,000 requests/second for allocation-efficient streaming

Actual performance will vary based on input size and system resources.