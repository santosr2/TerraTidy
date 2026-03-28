#!/usr/bin/env bash
# Generate test coverage report

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
COVERAGE_DIR="$PROJECT_ROOT/coverage"

# Create coverage directory
mkdir -p "$COVERAGE_DIR"

echo "Running tests with coverage..."

# Run tests with coverage
cd "$PROJECT_ROOT"
go test -v -race -coverprofile="$COVERAGE_DIR/coverage.out" -covermode=atomic ./...

# Generate HTML report
go tool cover -html="$COVERAGE_DIR/coverage.out" -o "$COVERAGE_DIR/coverage.html"

# Generate function-level coverage
go tool cover -func="$COVERAGE_DIR/coverage.out" > "$COVERAGE_DIR/coverage.txt"

# Calculate total coverage
TOTAL_COVERAGE=$(go tool cover -func="$COVERAGE_DIR/coverage.out" | grep total | awk '{print $3}')

echo ""
echo "✓ Coverage report generated:"
echo "  - HTML: $COVERAGE_DIR/coverage.html"
echo "  - Text: $COVERAGE_DIR/coverage.txt"
echo "  - Total Coverage: $TOTAL_COVERAGE"

# Check if coverage meets minimum
MIN_COVERAGE=80
COVERAGE_NUM=$(echo "$TOTAL_COVERAGE" | sed 's/%//')
if (( $(echo "$COVERAGE_NUM < $MIN_COVERAGE" | bc -l) )); then
    echo ""
    echo "⚠️  Warning: Coverage ($TOTAL_COVERAGE) is below minimum ($MIN_COVERAGE%)"
    exit 1
fi

echo ""
echo "✓ Coverage meets minimum requirement ($MIN_COVERAGE%)"
