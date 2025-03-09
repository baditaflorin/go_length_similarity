#!/bin/bash
# fixed-benchmark.sh - Performance testing script for the similarity HTTP server
# Usage: ./fixed-benchmark.sh [endpoint] [concurrency] [requests]

# Default values
ENDPOINT=${1:-"/length"}
CONCURRENCY=${2:-100}
REQUESTS=${3:-10000}
HOST=${4:-"http://localhost:8080"}
TEST_FILE="benchmark_payload.json"

# Create a test payload if it doesn't exist
if [ ! -f "$TEST_FILE" ]; then
    echo "Creating test payload..."
    cat > "$TEST_FILE" <<EOF
{
    "original": "This is a sample text that we'll use to benchmark the similarity server.",
    "augmented": "This is a modified text that we'll use to test the similarity server."
}
EOF
fi

echo "=== Similarity Server Benchmark ==="
echo "Endpoint: $HOST$ENDPOINT"
echo "Concurrency: $CONCURRENCY"
echo "Requests: $REQUESTS"
echo

# Read the payload file directly - this fixes the issue
PAYLOAD=$(cat "$TEST_FILE")

# Run hey benchmark with the payload content directly
if command -v hey &> /dev/null; then
    echo "Running benchmark with hey..."
    hey -n $REQUESTS -c $CONCURRENCY -m POST \
        -H "Content-Type: application/json" \
        -d "$PAYLOAD" \
        "$HOST$ENDPOINT"
# Try Apache Benchmark if hey isn't available
elif command -v ab &> /dev/null; then
    echo "Running benchmark with Apache Benchmark (ab)..."
    ab -n $REQUESTS -c $CONCURRENCY \
       -T 'application/json' \
       -p "$TEST_FILE" \
       "$HOST$ENDPOINT"
# Fallback to curl for basic benchmarking
elif command -v curl &> /dev/null; then
    echo "Running benchmark with curl (limited concurrency support)..."

    # Temporary file for timing results
    TIMING_FILE=$(mktemp)

    # Start time
    START_TIME=$(date +%s.%N)

    # Run requests in batches
    for ((i=1; i<=$REQUESTS; i+=$CONCURRENCY)); do
        # Calculate batch size (handle the last batch that may be smaller)
        BATCH_SIZE=$CONCURRENCY
        if [ $(($i + $CONCURRENCY - 1)) -gt $REQUESTS ]; then
            BATCH_SIZE=$(($REQUESTS - $i + 1))
        fi

        # Launch concurrent requests
        for ((j=0; j<$BATCH_SIZE; j++)); do
            curl -s -X POST "$HOST$ENDPOINT" \
                -H "Content-Type: application/json" \
                -d "$PAYLOAD" > /dev/null &
        done

        # Wait for all requests in this batch to complete
        wait

        # Progress update every 10% of total
        if [ $((i % (REQUESTS/10 + 1))) -eq 0 ]; then
            echo "Progress: $i/$REQUESTS requests completed"
        fi
    done

    # End time
    END_TIME=$(date +%s.%N)
    DURATION=$(echo "$END_TIME - $START_TIME" | bc)
    RPS=$(echo "$REQUESTS / $DURATION" | bc)

    echo
    echo "Total requests: $REQUESTS"
    echo "Duration: $DURATION seconds"
    echo "Requests per second: $RPS"

    # Clean up
    rm -f "$TIMING_FILE"
else
    echo "Error: No suitable benchmarking tool found."
    echo "Please install one of these tools:"
    echo "  - hey: https://github.com/rakyll/hey"
    echo "  - ab (Apache Benchmark): Install Apache HTTP Server tools package"
    echo "  - curl: Available on most systems"
    exit 1
fi

echo
echo "Benchmark complete."