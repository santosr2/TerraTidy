#!/usr/bin/env bash
set -euo pipefail

# TerraTidy GitHub Action runner script
# Called by action.yml with all inputs passed as environment variables.

# Build command arguments
ARGS=""

# Config file
if [ -f "$INPUT_CONFIG" ]; then
  ARGS="$ARGS --config $INPUT_CONFIG"
fi

# Profile
if [ -n "$INPUT_PROFILE" ]; then
  ARGS="$ARGS --profile $INPUT_PROFILE"
fi

# Skip flags
if [ "$INPUT_SKIP_FMT" = "true" ]; then
  ARGS="$ARGS --skip-fmt"
fi
if [ "$INPUT_SKIP_STYLE" = "true" ]; then
  ARGS="$ARGS --skip-style"
fi
if [ "$INPUT_SKIP_LINT" = "true" ]; then
  ARGS="$ARGS --skip-lint"
fi
if [ "$INPUT_SKIP_POLICY" = "true" ]; then
  ARGS="$ARGS --skip-policy"
fi

# Parallel mode
if [ "$INPUT_PARALLEL" = "true" ]; then
  ARGS="$ARGS --parallel"
fi

# Output format
FORMAT="$INPUT_FORMAT"
ARGS="$ARGS --format $FORMAT"

# SARIF output file
SARIF_FILE=""
SARIF_OUTPUT_PATH=""
if [ "$FORMAT" = "sarif" ]; then
  SARIF_FILE="terratidy-results.sarif"
  SARIF_OUTPUT_PATH="${INPUT_WORKING_DIRECTORY}/terratidy-results.sarif"
  SARIF_OUTPUT_PATH="${SARIF_OUTPUT_PATH#./}"
  echo "sarif-file=$SARIF_OUTPUT_PATH" >> "$GITHUB_OUTPUT"
fi

# Run TerraTidy and capture output
set +e
if [ "$FORMAT" = "sarif" ]; then
  terratidy check $ARGS > "$SARIF_FILE" 2>/tmp/terratidy-stderr.txt
  EXIT_CODE=$?
  cat /tmp/terratidy-stderr.txt || true
  OUTPUT=$(cat "$SARIF_FILE")
elif [ "$FORMAT" = "json" ] || [ "$FORMAT" = "json-compact" ]; then
  terratidy check $ARGS > /tmp/terratidy-output.json 2>/tmp/terratidy-stderr.txt
  EXIT_CODE=$?
  cat /tmp/terratidy-stderr.txt || true
  OUTPUT=$(cat /tmp/terratidy-output.json)
else
  OUTPUT=$(terratidy check $ARGS 2>&1)
  EXIT_CODE=$?
fi
set -e

echo "$OUTPUT"

# Parse findings counts from JSON output
if [ "$FORMAT" = "json" ] || [ "$FORMAT" = "json-compact" ]; then
  FINDINGS=$(echo "$OUTPUT" | jq -r '.summary.total // 0')
  ERRORS=$(echo "$OUTPUT" | jq -r '.summary.errors // 0')
  WARNINGS=$(echo "$OUTPUT" | jq -r '.summary.warnings // 0')
else
  FINDINGS=$EXIT_CODE
  ERRORS=0
  WARNINGS=0
  if [ $EXIT_CODE -ne 0 ]; then
    ERRORS=$EXIT_CODE
  fi
fi

echo "findings-count=$FINDINGS" >> "$GITHUB_OUTPUT"
echo "errors-count=$ERRORS" >> "$GITHUB_OUTPUT"
echo "warnings-count=$WARNINGS" >> "$GITHUB_OUTPUT"

# Determine if we should fail
SHOULD_FAIL=false

if [ "$INPUT_FAIL_ON_ERROR" = "true" ] && [ "$ERRORS" -gt 0 ]; then
  SHOULD_FAIL=true
fi

if [ "$INPUT_FAIL_ON_WARNING" = "true" ] && [ "$WARNINGS" -gt 0 ]; then
  SHOULD_FAIL=true
fi

if [ "$SHOULD_FAIL" = "true" ]; then
  exit 1
fi
