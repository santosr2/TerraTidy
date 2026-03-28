#!/usr/bin/env bash
# Run benchmarks and generate reports

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BENCH_DIR="$PROJECT_ROOT/benchmarks"

# Create benchmarks directory
mkdir -p "$BENCH_DIR"

echo "Running benchmarks..."

# Run benchmarks
cd "$PROJECT_ROOT"
go test -bench=. -benchmem -benchtime=5s -run=^$ ./... \
    | tee "$BENCH_DIR/benchmark-$(date +%Y%m%d-%H%M%S).txt"

# If benchstat is installed, compare with previous run
if command -v benchstat &> /dev/null; then
    LATEST=$(ls -t "$BENCH_DIR"/benchmark-*.txt | head -1)
    PREVIOUS=$(ls -t "$BENCH_DIR"/benchmark-*.txt | head -2 | tail -1)

    if [ "$LATEST" != "$PREVIOUS" ] && [ -f "$PREVIOUS" ]; then
        echo ""
        echo "Comparison with previous run:"
        benchstat "$PREVIOUS" "$LATEST"
    fi
else
    echo ""
    echo "Tip: Install benchstat for benchmark comparison:"
    echo "  go install golang.org/x/perf/cmd/benchstat@latest"
fi

echo ""
echo "✓ Benchmark results saved to: $BENCH_DIR"
