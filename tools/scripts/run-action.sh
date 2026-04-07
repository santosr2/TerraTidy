#!/usr/bin/env bash
set -euo pipefail

# TerraTidy GitHub Action runner script
# Called by action.yml with all inputs passed as environment variables.

# Validate input to prevent shell injection.
# Only allows alphanumeric characters, dots, underscores, slashes, hyphens, and spaces.
# Paths with special characters ($, `, !, etc.) are rejected for security.
validate_input() {
  local name="$1"
  local value="$2"

  # Empty values are allowed
  if [ -z "$value" ]; then
    return 0
  fi

  # Validate against safe pattern
  if ! [[ "$value" =~ ^[a-zA-Z0-9._/\ -]*$ ]]; then
    echo "::error::Invalid $name: contains unsafe characters. Only alphanumeric, dots, underscores, slashes, hyphens, and spaces are allowed."
    exit 1
  fi
}

# Validate format against allowed values
validate_format() {
  local value="$1"
  case "$value" in
    text|table|json|json-compact|sarif|html|junit|markdown|github)
      return 0
      ;;
    *)
      echo "::error::Invalid format: '$value'. Allowed values: text, table, json, json-compact, sarif, html, junit, markdown, github"
      exit 1
      ;;
  esac
}

# Validate glob patterns (allows alphanumeric, dots, underscores, slashes, hyphens, asterisks, and commas)
validate_glob_patterns() {
  local name="$1"
  local value="$2"

  # Empty values are allowed
  if [ -z "$value" ]; then
    return 0
  fi

  # Validate against safe pattern for glob patterns
  if ! [[ "$value" =~ ^[a-zA-Z0-9._/*,\ -]*$ ]]; then
    echo "::error::Invalid $name: contains unsafe characters. Only alphanumeric, dots, underscores, slashes, hyphens, asterisks, and commas are allowed."
    exit 1
  fi
}

# Validate user-provided inputs before any execution
validate_input "config" "${INPUT_CONFIG:-}"
validate_input "profile" "${INPUT_PROFILE:-}"
validate_input "working-directory" "${INPUT_WORKING_DIRECTORY:-}"
validate_glob_patterns "exclude" "${INPUT_EXCLUDE:-}"
validate_format "${INPUT_FORMAT:-text}"

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

# Exclude patterns (comma-separated)
if [ -n "${INPUT_EXCLUDE:-}" ]; then
  ARGS="$ARGS --exclude $INPUT_EXCLUDE"
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
TMPDIR="${RUNNER_TEMP:-/tmp}"
set +e
if [ "$FORMAT" = "sarif" ]; then
  terratidy check $ARGS > "$SARIF_FILE" 2>"$TMPDIR/terratidy-stderr.txt"
  EXIT_CODE=$?
  cat "$TMPDIR/terratidy-stderr.txt" || true
  OUTPUT=$(cat "$SARIF_FILE")
elif [ "$FORMAT" = "json" ] || [ "$FORMAT" = "json-compact" ]; then
  terratidy check $ARGS > "$TMPDIR/terratidy-output.json" 2>"$TMPDIR/terratidy-stderr.txt"
  EXIT_CODE=$?
  cat "$TMPDIR/terratidy-stderr.txt" || true
  OUTPUT=$(cat "$TMPDIR/terratidy-output.json")
else
  OUTPUT=$(terratidy check $ARGS 2>&1)
  EXIT_CODE=$?
fi
set -e

echo "$OUTPUT"

# Parse findings counts from JSON output.
# Accurate counts are only available for json/json-compact formats.
# For other formats, findings-count reflects the exit code (0 = clean,
# non-zero = issues found) and errors-count/warnings-count are estimates.
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
