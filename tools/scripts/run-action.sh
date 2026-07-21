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

  # Validate against safe pattern (includes backslash and colon for Windows paths)
  if ! [[ "$value" =~ ^[a-zA-Z0-9._/\\:\ -]*$ ]]; then
    echo "::error::Invalid $name: contains unsafe characters. Only alphanumeric, dots, underscores, slashes, backslashes, colons, hyphens, and spaces are allowed."
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

# Build command as array (safer than string + eval)
CMD=(terratidy check)

# Config file
if [ -n "${INPUT_CONFIG:-}" ] && [ -f "${INPUT_CONFIG:-}" ]; then
  CMD+=(--config "$INPUT_CONFIG")
fi

# Profile
if [ -n "${INPUT_PROFILE:-}" ]; then
  CMD+=(--profile "$INPUT_PROFILE")
fi

# Skip flags
if [ "${INPUT_SKIP_FMT:-false}" = "true" ]; then
  CMD+=(--skip-fmt)
fi
if [ "${INPUT_SKIP_STYLE:-false}" = "true" ]; then
  CMD+=(--skip-style)
fi
if [ "${INPUT_SKIP_LINT:-false}" = "true" ]; then
  CMD+=(--skip-lint)
fi
if [ "${INPUT_SKIP_POLICY:-false}" = "true" ]; then
  CMD+=(--skip-policy)
fi

# Parallel mode. Pass an explicit flag either way: without --no-parallel, setting
# parallel: false has no effect because the config default (parallel: true) wins.
if [ "${INPUT_PARALLEL:-false}" = "true" ]; then
  CMD+=(--parallel)
else
  CMD+=(--no-parallel)
fi

# Exclude patterns (comma-separated)
if [ -n "${INPUT_EXCLUDE:-}" ]; then
  CMD+=(--exclude "$INPUT_EXCLUDE")
fi

# No-recurse flag
if [ "${INPUT_NO_RECURSE:-false}" = "true" ]; then
  CMD+=(--no-recurse)
fi

# Absolute paths flag
if [ "${INPUT_ABSOLUTE_PATHS:-false}" = "true" ]; then
  CMD+=(--absolute-paths)
fi

# Changed files only
if [ "${INPUT_CHANGED:-false}" = "true" ]; then
  CMD+=(--changed)
fi

# Severity threshold
if [ -n "${INPUT_SEVERITY_THRESHOLD:-}" ]; then
  CMD+=(--severity-threshold "$INPUT_SEVERITY_THRESHOLD")
fi

# Output format
FORMAT="${INPUT_FORMAT:-text}"

# When fail-on-warning is enabled with non-JSON formats, we need to run twice:
# once with JSON to get accurate warning counts, then with user's format for display.
NEEDS_JSON_FOR_WARNINGS=false
if [ "${INPUT_FAIL_ON_WARNING:-false}" = "true" ]; then
  if [ "$FORMAT" != "json" ] && [ "$FORMAT" != "json-compact" ]; then
    NEEDS_JSON_FOR_WARNINGS=true
  fi
fi

CMD+=(--format "$FORMAT")

# SARIF output file
SARIF_FILE=""
SARIF_OUTPUT_PATH=""
if [ "$FORMAT" = "sarif" ]; then
  SARIF_FILE="terratidy-results.sarif"
  SARIF_OUTPUT_PATH="${INPUT_WORKING_DIRECTORY:-}/terratidy-results.sarif"
  SARIF_OUTPUT_PATH="${SARIF_OUTPUT_PATH#./}"
  echo "sarif-file=$SARIF_OUTPUT_PATH" >> "$GITHUB_OUTPUT"
fi

# Run TerraTidy and capture output
# Use array expansion instead of eval for safety
_TMPDIR="${RUNNER_TEMP:-/tmp}"

# Pre-run: If fail-on-warning is enabled with non-JSON format, run with JSON first
# to get accurate warning/error counts for the fail decision.
JSON_COUNTS_OUTPUT=""
JSON_PRERUN_EXIT=0
if [ "$NEEDS_JSON_FOR_WARNINGS" = "true" ]; then
  # Build JSON command (same as CMD but with --format json)
  JSON_CMD=("${CMD[@]}")
  # Replace the format argument (--format must exist and have a value after it)
  for i in "${!JSON_CMD[@]}"; do
    if [ "${JSON_CMD[$i]}" = "--format" ]; then
      JSON_CMD[i+1]="json"
      break
    fi
  done
  set +e
  JSON_COUNTS_OUTPUT=$("${JSON_CMD[@]}" 2>"$_TMPDIR/terratidy-prerun-stderr.txt")
  JSON_PRERUN_EXIT=$?
  set -e
  # Log pre-run errors as warning (pre-run failure degrades to exit-code fallback)
  if [ -s "$_TMPDIR/terratidy-prerun-stderr.txt" ]; then
    echo "::warning::TerraTidy pre-run for warning counts produced stderr:"
    cat "$_TMPDIR/terratidy-prerun-stderr.txt"
  fi
fi

set +e
if [ "$FORMAT" = "sarif" ]; then
  "${CMD[@]}" > "$SARIF_FILE" 2>"$_TMPDIR/terratidy-stderr.txt"
  EXIT_CODE=$?
  cat "$_TMPDIR/terratidy-stderr.txt" || true
  OUTPUT=$(cat "$SARIF_FILE")
elif [ "$FORMAT" = "json" ] || [ "$FORMAT" = "json-compact" ]; then
  "${CMD[@]}" > "$_TMPDIR/terratidy-output.json" 2>"$_TMPDIR/terratidy-stderr.txt"
  EXIT_CODE=$?
  cat "$_TMPDIR/terratidy-stderr.txt" || true
  OUTPUT=$(cat "$_TMPDIR/terratidy-output.json")
else
  OUTPUT=$("${CMD[@]}" 2>&1)
  EXIT_CODE=$?
fi
set -e

echo "$OUTPUT"

# Tool-level failures (config error = 2, internal error = 3) are not findings the
# user can opt out of via fail-on-error / fail-on-warning. Without this guard a
# JSON run that exits 2/3 produces no parseable summary, so the counts below fall
# back to 0 and the step reports a false success. Fail outright on any code >= 2.
if [ "$EXIT_CODE" -ge 2 ]; then
  echo "::error::TerraTidy exited with code $EXIT_CODE (configuration or internal error); see the log above."
  exit "$EXIT_CODE"
fi

# Parse findings counts from JSON output.
# Accurate counts are only available for json/json-compact formats, or when
# fail-on-warning is enabled (pre-run provides accurate counts for any format).
# Otherwise, fall back to exit code based estimates.
if [ "$FORMAT" = "json" ] || [ "$FORMAT" = "json-compact" ]; then
  FINDINGS=$(echo "$OUTPUT" | jq -r '.summary.total // 0' 2>/dev/null || echo 0)
  ERRORS=$(echo "$OUTPUT" | jq -r '.summary.errors // 0' 2>/dev/null || echo 0)
  WARNINGS=$(echo "$OUTPUT" | jq -r '.summary.warnings // 0' 2>/dev/null || echo 0)
elif [ -n "$JSON_COUNTS_OUTPUT" ] && [ "$JSON_PRERUN_EXIT" -le 1 ]; then
  # Use counts from pre-run JSON output (fail-on-warning with non-JSON format)
  # Trust exit 0 (clean) and 1 (findings exist); fall back to estimates on 2 (config) or 3 (internal)
  FINDINGS=$(echo "$JSON_COUNTS_OUTPUT" | jq -r '.summary.total // 0' 2>/dev/null || echo 0)
  ERRORS=$(echo "$JSON_COUNTS_OUTPUT" | jq -r '.summary.errors // 0' 2>/dev/null || echo 0)
  WARNINGS=$(echo "$JSON_COUNTS_OUTPUT" | jq -r '.summary.warnings // 0' 2>/dev/null || echo 0)
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

if [ "${INPUT_FAIL_ON_ERROR:-true}" = "true" ] && [ "$ERRORS" -gt 0 ]; then
  SHOULD_FAIL=true
fi

if [ "${INPUT_FAIL_ON_WARNING:-false}" = "true" ] && [ "$WARNINGS" -gt 0 ]; then
  SHOULD_FAIL=true
fi

if [ "$SHOULD_FAIL" = "true" ]; then
  exit 1
fi
