#!/usr/bin/env bash
set -uo pipefail

# Fuzz test runner with auto-discovery
# Usage: fuzz.sh [fuzztime]
# Example: fuzz.sh 30s    # Run each target for 30s
#          fuzz.sh 5m     # Run each target for 5m (scheduled runs)

FUZZTIME="${1:-30s}"
FAILED=0
TOTAL=0

echo "Discovering fuzz targets..."

# Collect all fuzz targets (package:target pairs)
TARGETS=""
for pkg in $(go list ./...); do
  # Get fuzz targets in this package
  for target in $(go test -list 'Fuzz.*' "$pkg" 2>/dev/null | grep '^Fuzz' || true); do
    TARGETS="$TARGETS $pkg:$target"
    TOTAL=$((TOTAL + 1))
  done
done

# Trim leading space
TARGETS="${TARGETS# }"

if [ -z "$TARGETS" ]; then
  echo "No fuzz targets found"
  exit 0
fi

echo "Found $TOTAL fuzz targets"
echo ""

# Run each fuzz target
for entry in $TARGETS; do
  pkg="${entry%:*}"
  target="${entry#*:}"

  echo "=== Fuzzing $target in $pkg for $FUZZTIME ==="

  if go test -fuzz="$target" -fuzztime="$FUZZTIME" "$pkg"; then
    echo "PASS"
  else
    echo "FAIL"
    FAILED=1
  fi
  echo ""
done

if [ "$FAILED" -eq 1 ]; then
  echo "Some fuzz tests failed"
  exit 1
fi

echo "All fuzz tests passed"
