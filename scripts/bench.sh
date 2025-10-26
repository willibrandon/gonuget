#!/bin/bash
set -e

echo "===================="
echo "gonuget Benchmarks"
echo "===================="
echo ""

# Build with optimizations
echo "Building optimized binary..."
go build -o gonuget -ldflags="-s -w" ./cmd/gonuget

echo ""
echo "===================="
echo "Startup Benchmarks"
echo "===================="
echo ""

# Run startup benchmarks
go test -tags=benchmark -bench=BenchmarkStartup -benchmem -benchtime=100x ./cmd/gonuget | tee startup_bench.txt

echo ""
echo "===================="
echo "Command Benchmarks"
echo "===================="
echo ""

# Run command benchmarks
go test -tags=benchmark -bench=BenchmarkVersion -benchmem ./cmd/gonuget
go test -tags=benchmark -bench=BenchmarkConfig -benchmem ./cmd/gonuget
go test -tags=benchmark -bench=Benchmark.*Source -benchmem ./cmd/gonuget
go test -tags=benchmark -bench=BenchmarkHelp -benchmem ./cmd/gonuget

echo ""
echo "===================="
echo "Package Benchmarks"
echo "===================="
echo ""

# Run package-level benchmarks
go test -tags=benchmark -bench=. -benchmem ./cmd/gonuget/output
go test -tags=benchmark -bench=. -benchmem ./cmd/gonuget/config

echo ""
echo "===================="
echo "Performance Report"
echo "===================="
echo ""

# Extract startup time from benchmark results
if [ -f startup_bench.txt ]; then
    startup_time=$(grep "BenchmarkStartup" startup_bench.txt | awk '{print $3}')
    startup_ms=$(echo "$startup_time" | sed 's/μs\/op//' | awk '{print $1/1000}')

    echo "Startup Time: ${startup_ms}ms"

    # Check against target (<50ms P50)
    if (( $(echo "$startup_ms < 50" | bc -l) )); then
        echo "✅ PASS: Startup time is below 50ms target"
    else
        echo "❌ FAIL: Startup time exceeds 50ms target"
        exit 1
    fi
fi

echo ""
echo "===================="
echo "Benchmark complete!"
echo "===================="
