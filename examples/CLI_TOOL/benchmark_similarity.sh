#!/bin/bash
# benchmark_similarity.sh - Performance testing script for the similarity CLI tool
# Usage: ./benchmark_similarity.sh [iterations] [mode]

# Default values
ITERATIONS=${1:-100}
MODE=${2:-"simple"}
TOOL="./similarity"  # Path to your compiled similarity tool
ORIGINAL_FILE="sample_original.txt"
AUGMENTED_FILE="sample_augmented.txt"

# Create sample files if they don't exist
if [ ! -f "$ORIGINAL_FILE" ]; then
    echo "Creating sample original file..."
    for i in $(seq 1 100); do
        echo "This is line $i of the original test file for benchmarking similarity calculations." >> "$ORIGINAL_FILE"
    done
fi

if [ ! -f "$AUGMENTED_FILE" ]; then
    echo "Creating sample augmented file..."
    for i in $(seq 1 100); do
        # Make it slightly different
        echo "This is line $i of the modified test file for benchmarking similarity measurements." >> "$AUGMENTED_FILE"
    done
fi

# Function to run a single test and return time in seconds
run_single_test() {
    local config=$1
    local start=$(date +%s.%N)
    eval "$config" >/dev/null 2>&1
    local end=$(date +%s.%N)
    echo $(echo "$end - $start" | bc)
}

# Simple sequential benchmark
run_simple_benchmark() {
    echo "Running simple sequential benchmark ($ITERATIONS iterations)..."
    local total_time=0
    local min_time=999999
    local max_time=0

    for i in $(seq 1 $ITERATIONS); do
        local elapsed=$(run_single_test "$TOOL --original-file=$ORIGINAL_FILE --augmented-file=$AUGMENTED_FILE")
        total_time=$(echo "$total_time + $elapsed" | bc)

        # Update min/max times
        min_time=$(echo "if ($elapsed < $min_time) $elapsed else $min_time" | bc)
        max_time=$(echo "if ($elapsed > $max_time) $elapsed else $max_time" | bc)

        # Show progress
        if [ $((i % 10)) -eq 0 ]; then
            echo -ne "Completed: $i/$ITERATIONS\r"
        fi
    done

    echo -e "\nCompleted $ITERATIONS iterations"
    echo "Total time: $(echo "scale=2; $total_time" | bc) seconds"
    echo "Average time: $(echo "scale=4; $total_time / $ITERATIONS" | bc) seconds per run"
    echo "Min time: $min_time seconds"
    echo "Max time: $max_time seconds"
}

# Comparative benchmark (comparing different configurations)
run_comparative_benchmark() {
    echo "Running comparative benchmark ($ITERATIONS iterations each)..."

    # Define configurations to test
    local configs=(
        "Default:$TOOL --original-file=$ORIGINAL_FILE --augmented-file=$AUGMENTED_FILE"
        "Length:$TOOL --original-file=$ORIGINAL_FILE --augmented-file=$AUGMENTED_FILE --metric=length"
        "Character:$TOOL --original-file=$ORIGINAL_FILE --augmented-file=$AUGMENTED_FILE --metric=character"
        "Optimized:$TOOL --original-file=$ORIGINAL_FILE --augmented-file=$AUGMENTED_FILE --optimize-speed"
        "Streaming:$TOOL --original-file=$ORIGINAL_FILE --augmented-file=$AUGMENTED_FILE --streaming --streaming-mode=line"
        "WordStream:$TOOL --original-file=$ORIGINAL_FILE --augmented-file=$AUGMENTED_FILE --streaming --streaming-mode=word"
    )

    # Run benchmark for each configuration
    for config in "${configs[@]}"; do
        local name="${config%%:*}"
        local cmd="${config#*:}"

        echo -e "\nTesting configuration: $name"
        local total_time=0

        for i in $(seq 1 $ITERATIONS); do
            local elapsed=$(run_single_test "$cmd")
            total_time=$(echo "$total_time + $elapsed" | bc)

            # Show progress
            if [ $((i % 10)) -eq 0 ]; then
                echo -ne "Completed: $i/$ITERATIONS\r"
            fi
        done

        echo -e "\nAverage time for $name: $(echo "scale=4; $total_time / $ITERATIONS" | bc) seconds per run"
    done
}

# Parallel benchmark (testing throughput)
run_parallel_benchmark() {
    echo "Running parallel benchmark ($ITERATIONS iterations)..."

    # Determine level of parallelism based on CPU cores
    local cores=$(nproc)
    local parallelism=$((cores * 2))  # 2x number of cores
    echo "Using parallelism level: $parallelism"

    # Create temporary directory for results
    local tempdir=$(mktemp -d)

    # Launch parallel processes
    local start=$(date +%s.%N)
    for i in $(seq 1 $ITERATIONS); do
        $TOOL --original-file=$ORIGINAL_FILE --augmented-file=$AUGMENTED_FILE > "$tempdir/result_$i.txt" 2>/dev/null &

        # Limit concurrent processes
        if [ $((i % parallelism)) -eq 0 ]; then
            wait
        fi

        # Show progress
        if [ $((i % 10)) -eq 0 ]; then
            echo -ne "Launched: $i/$ITERATIONS\r"
        fi
    done

    # Wait for all processes to finish
    wait
    local end=$(date +%s.%N)

    local total_time=$(echo "$end - $start" | bc)
    echo -e "\nAll processes completed."
    echo "Total wall-clock time: $total_time seconds"
    echo "Throughput: $(echo "scale=2; $ITERATIONS / $total_time" | bc) operations per second"

    # Clean up
    rm -rf "$tempdir"
}

# Progressive load benchmark (increasing load)
run_progressive_benchmark() {
    echo "Running progressive load benchmark..."

    local batch_sizes=(1 10 50 100 500 1000)

    for batch in "${batch_sizes[@]}"; do
        echo -e "\nTesting batch size: $batch"
        local start=$(date +%s.%N)

        # Run tests in parallel batches
        for i in $(seq 1 $batch); do
            $TOOL --original-file=$ORIGINAL_FILE --augmented-file=$AUGMENTED_FILE >/dev/null 2>&1 &
        done
        wait

        local end=$(date +%s.%N)
        local total_time=$(echo "$end - $start" | bc)
        echo "Total time: $total_time seconds"
        echo "Average time per operation: $(echo "scale=4; $total_time / $batch" | bc) seconds"
        echo "Throughput: $(echo "scale=2; $batch / $total_time" | bc) operations per second"
    done
}

# Detailed profiling (CPU, memory usage)
run_profiling_benchmark() {
    echo "Running detailed profiling benchmark ($ITERATIONS iterations)..."

    # Create output CSV file
    local csv_file="similarity_benchmark.csv"
    echo "Iteration,ElapsedTime,MaxMemory" > "$csv_file"

    for i in $(seq 1 $ITERATIONS); do
        echo -ne "Running iteration $i/$ITERATIONS\r"

        # Use /usr/bin/time to capture resource usage
        local output_file=$(mktemp)
        local start=$(date +%s.%N)

        /usr/bin/time -f "%M" $TOOL --original-file=$ORIGINAL_FILE --augmented-file=$AUGMENTED_FILE >/dev/null 2>"$output_file"

        local end=$(date +%s.%N)
        local elapsed=$(echo "$end - $start" | bc)
        local memory=$(cat "$output_file")

        # Append to CSV
        echo "$i,$elapsed,$memory" >> "$csv_file"
        rm "$output_file"
    done

    echo -e "\nProfiling data saved to $csv_file"

    # Calculate and display summary statistics
    echo "Calculating summary statistics..."
    awk -F, '
    BEGIN { sum=0; count=0; min=999999; max=0; sum_mem=0; min_mem=999999999; max_mem=0; }
    NR>1 {
        sum+=$2; count++;
        if($2<min) min=$2;
        if($2>max) max=$2;
        sum_mem+=$3;
        if($3<min_mem) min_mem=$3;
        if($3>max_mem) max_mem=$3;
    }
    END {
        printf "Time (seconds) - Avg: %.4f, Min: %.4f, Max: %.4f\n", sum/count, min, max;
        printf "Memory (KB) - Avg: %.2f, Min: %d, Max: %d\n", sum_mem/count, min_mem, max_mem;
    }' "$csv_file"

    echo "For detailed analysis, import $csv_file into a spreadsheet application or use a tool like gnuplot."
}

# Main execution
case "$MODE" in
    "simple")
        run_simple_benchmark
        ;;
    "compare")
        run_comparative_benchmark
        ;;
    "parallel")
        run_parallel_benchmark
        ;;
    "load")
        run_progressive_benchmark
        ;;
    "profile")
        run_profiling_benchmark
        ;;
    "all")
        run_simple_benchmark
        run_comparative_benchmark
        run_parallel_benchmark
        run_progressive_benchmark
        run_profiling_benchmark
        ;;
    *)
        echo "Unknown mode: $MODE"
        echo "Available modes: simple, compare, parallel, load, profile, all"
        exit 1
        ;;
esac

echo "Benchmark completed."