# Benchmarking the Similarity CLI Tool

This document provides examples of how to use the `benchmark_similarity.sh` script to test the performance of the similarity CLI tool under different configurations and loads.

## Preparing for Benchmarking

1. First, compile the similarity tool:
   ```bash
   go build -o similarity main.go
   ```

2. Make the benchmark script executable:
   ```bash
   chmod +x benchmark_similarity.sh
   ```

3. Ensure you have basic Unix utilities installed:
    - `bc` for floating-point calculations
    - `/usr/bin/time` for resource usage measurement (standard on most Linux distributions)

## Basic Usage

The benchmark script accepts two optional parameters:
```bash
./benchmark_similarity.sh [iterations] [mode]
```

- `iterations`: Number of times to run the test (default: 100)
- `mode`: Benchmark mode to run (default: "simple")

## Benchmark Modes

### 1. Simple Sequential Benchmark

Runs the tool multiple times in sequence and reports the average execution time:

```bash
# Run 100 iterations (default)
./benchmark_similarity.sh

# Run 1000 iterations
./benchmark_similarity.sh 1000 simple
```

### 2. Comparative Benchmark

Compares different configurations of the tool:

```bash
./benchmark_similarity.sh 100 compare
```

This will test various configurations:
- Default settings
- Length-based similarity
- Character-based similarity
- Optimized performance
- Line-by-line streaming
- Word-by-word streaming

### 3. Parallel Benchmark

Tests throughput by running many instances in parallel:

```bash
./benchmark_similarity.sh 500 parallel
```

This is useful for measuring how the tool performs under concurrent load, such as in a server environment.

### 4. Progressive Load Benchmark

Increases the load progressively to identify performance bottlenecks:

```bash
./benchmark_similarity.sh 0 load
```

The iteration count is ignored in this mode as it uses predefined batch sizes.

### 5. Detailed Profiling

Collects detailed performance metrics including memory usage:

```bash
./benchmark_similarity.sh 50 profile
```

This generates a CSV file with detailed measurements that you can analyze in a spreadsheet or plotting tool.

### 6. Run All Benchmark Types

To run all benchmark modes in sequence:

```bash
./benchmark_similarity.sh 100 all
```

## Real-World Examples

### Testing with Large Files

```bash
# Create larger test files
for i in {1..10000}; do 
  echo "Line $i: $(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 100 | head -n 1)" >> large_original.txt
done

# Create a slightly modified version
cp large_original.txt large_modified.txt
sed -i -e 's/a/e/g' -e 's/t/r/g' large_modified.txt

# Update the benchmark script to use these files
sed -i 's/ORIGINAL_FILE="sample_original.txt"/ORIGINAL_FILE="large_original.txt"/' benchmark_similarity.sh
sed -i 's/AUGMENTED_FILE="sample_augmented.txt"/AUGMENTED_FILE="large_modified.txt"/' benchmark_similarity.sh

# Run benchmark with large files
./benchmark_similarity.sh 50 compare
```

### Continuous Integration Testing

Add this to your CI pipeline to track performance over time:

```bash
#!/bin/bash
# ci_benchmark.sh

# Run benchmark and save results
./benchmark_similarity.sh 100 profile

# Extract average time and memory usage
AVG_TIME=$(awk -F, 'NR>1 {sum+=$2} END {print sum/(NR-1)}' similarity_benchmark.csv)
AVG_MEM=$(awk -F, 'NR>1 {sum+=$3} END {print sum/(NR-1)}' similarity_benchmark.csv)

# Compare with previous build (example)
PREV_TIME=$(cat previous_benchmark.txt | grep "time" | awk '{print $2}')
PREV_MEM=$(cat previous_benchmark.txt | grep "mem" | awk '{print $2}')

# Calculate percentage change
TIME_CHANGE=$(echo "scale=2; 100 * ($AVG_TIME - $PREV_TIME) / $PREV_TIME" | bc)
MEM_CHANGE=$(echo "scale=2; 100 * ($AVG_MEM - $PREV_MEM) / $PREV_MEM" | bc)

# Save current results for next build
echo "time $AVG_TIME" > previous_benchmark.txt
echo "mem $AVG_MEM" >> previous_benchmark.txt

# Report changes
echo "Performance change: ${TIME_CHANGE}% execution time, ${MEM_CHANGE}% memory usage"

# Fail if performance degraded significantly
if (( $(echo "$TIME_CHANGE > 10" | bc -l) )); then
  echo "Performance regression detected!"
  exit 1
fi
```

### Quick A/B Testing

Test the impact of a specific change:

```bash
# Measure baseline
./benchmark_similarity.sh 100 simple > baseline.log

# Make your changes to the code and recompile
# ...

# Measure new performance
./benchmark_similarity.sh 100 simple > new.log

# Compare results
echo "Baseline:"
grep "Average time" baseline.log
echo "New version:"
grep "Average time" new.log
```

## Tips for Accurate Benchmarking

1. **Run multiple times**: For important measurements, run the benchmark multiple times to account for system variability.

2. **Minimize system load**: Close other applications and services that might affect benchmark results.

3. **Use consistent hardware**: Always compare benchmarks run on the same hardware.

4. **Warm up**: The first few runs might be slower due to caching, JIT compilation, etc. Consider adding a warm-up phase.

5. **Test with realistic data**: Use data that closely resembles what you'll be processing in production.

6. **Monitor system resources**: Use tools like `top`, `htop`, or `sar` to monitor system resource usage during benchmarks.