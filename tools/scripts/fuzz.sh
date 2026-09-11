#!/usr/bin/env bash
set -uo pipefail

# Fuzz test runner with auto-discovery
# Usage: fuzz.sh [fuzztime] [package]
# Example: fuzz.sh 30s                          # Every target, 30s each
#          fuzz.sh 5m                           # Every target, 5m each
#          fuzz.sh 5m ./internal/cst            # Only that package
#
# `go test -fuzz` fuzzes one target per invocation, so the full sweep is
# sequential and its wall time is (targets x fuzztime). Pass a package to shard
# that across parallel CI jobs instead of serialising all of them into one.

FUZZTIME="${1:-30s}"
SCOPE="${2:-./...}"
FAILED=0
TOTAL=0

echo "Discovering fuzz targets in $SCOPE..."

# Collect all fuzz targets (package:target pairs)
TARGETS=""
for pkg in $(go list "$SCOPE"); do
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

  if go test -fuzz="^${target}\$" -fuzztime="$FUZZTIME" "$pkg"; then
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
