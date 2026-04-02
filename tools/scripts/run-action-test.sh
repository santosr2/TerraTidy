#!/usr/bin/env bash
# Test script for run-action.sh input validation
# Run: ./tools/scripts/run-action-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_ACTION="$SCRIPT_DIR/run-action.sh"

# Test counters
PASSED=0
FAILED=0

# Create a mock terratidy
MOCK_DIR="$SCRIPT_DIR/.testmocks"
mkdir -p "$MOCK_DIR"
cat > "$MOCK_DIR/terratidy" << 'EOF'
#!/bin/bash
echo '{"summary":{"total":0,"errors":0,"warnings":0}}'
exit 0
EOF
chmod +x "$MOCK_DIR/terratidy"

cleanup() {
  rm -rf "$MOCK_DIR"
}
trap cleanup EXIT

# Test helper: expect script to fail with specific error
expect_fail() {
  local test_name="$1"
  local expected_error="$2"
  shift 2

  output=$("$@" 2>&1) && exit_code=0 || exit_code=$?

  if [ "$exit_code" -eq 0 ]; then
    echo "FAIL: $test_name - expected failure but got success"
    FAILED=$((FAILED + 1))
    return 1
  fi

  if [[ "$output" != *"$expected_error"* ]]; then
    echo "FAIL: $test_name - expected error containing '$expected_error' but got: $output"
    FAILED=$((FAILED + 1))
    return 1
  fi

  echo "PASS: $test_name"
  PASSED=$((PASSED + 1))
  return 0
}

# Test helper: expect validation to pass
expect_pass() {
  local test_name="$1"
  shift

  output=$(PATH="$MOCK_DIR:$PATH" "$@" 2>&1) && exit_code=0 || exit_code=$?

  if [ "$exit_code" -ne 0 ]; then
    echo "FAIL: $test_name - expected success but got failure: $output"
    FAILED=$((FAILED + 1))
    return 1
  fi

  echo "PASS: $test_name"
  PASSED=$((PASSED + 1))
  return 0
}

echo "Testing input validation in run-action.sh"
echo "=========================================="

export GITHUB_OUTPUT="/dev/null"

# Test: Valid inputs should pass
INPUT_CONFIG=".terratidy.yaml" \
INPUT_PROFILE="production" \
INPUT_WORKING_DIRECTORY="./terraform" \
INPUT_FORMAT="json" \
INPUT_SKIP_FMT="false" \
INPUT_SKIP_STYLE="false" \
INPUT_SKIP_LINT="false" \
INPUT_SKIP_POLICY="false" \
INPUT_PARALLEL="false" \
INPUT_FAIL_ON_ERROR="false" \
INPUT_FAIL_ON_WARNING="false" \
expect_pass "valid inputs" bash "$RUN_ACTION"

# Test: Empty inputs should pass
INPUT_CONFIG="" \
INPUT_PROFILE="" \
INPUT_WORKING_DIRECTORY="" \
INPUT_FORMAT="text" \
INPUT_SKIP_FMT="false" \
INPUT_SKIP_STYLE="false" \
INPUT_SKIP_LINT="false" \
INPUT_SKIP_POLICY="false" \
INPUT_PARALLEL="false" \
INPUT_FAIL_ON_ERROR="false" \
INPUT_FAIL_ON_WARNING="false" \
expect_pass "empty inputs" bash "$RUN_ACTION"

# Test: Shell metacharacters in config should fail
# shellcheck disable=SC2016 # Single quotes intentional - testing literal $() rejection
INPUT_CONFIG='$(whoami).yaml' \
INPUT_PROFILE="" \
INPUT_WORKING_DIRECTORY="" \
INPUT_FORMAT="text" \
expect_fail "command substitution in config" "Invalid config" bash "$RUN_ACTION"

# Test: Backticks in profile should fail
# shellcheck disable=SC2016 # Single quotes intentional - testing literal backtick rejection
INPUT_CONFIG="" \
INPUT_PROFILE='`id`' \
INPUT_WORKING_DIRECTORY="" \
INPUT_FORMAT="text" \
expect_fail "backticks in profile" "Invalid profile" bash "$RUN_ACTION"

# Test: Semicolon in working-directory should fail
INPUT_CONFIG="" \
INPUT_PROFILE="" \
INPUT_WORKING_DIRECTORY="; rm -rf /" \
INPUT_FORMAT="text" \
expect_fail "semicolon in working-directory" "Invalid working-directory" bash "$RUN_ACTION"

# Test: Pipe in config should fail
INPUT_CONFIG="config.yaml | cat /etc/passwd" \
INPUT_PROFILE="" \
INPUT_WORKING_DIRECTORY="" \
INPUT_FORMAT="text" \
expect_fail "pipe in config" "Invalid config" bash "$RUN_ACTION"

# Test: Invalid format should fail
INPUT_CONFIG="" \
INPUT_PROFILE="" \
INPUT_WORKING_DIRECTORY="" \
INPUT_FORMAT="malicious" \
expect_fail "invalid format" "Invalid format" bash "$RUN_ACTION"

# Test: Valid format variations
for fmt in text table json json-compact sarif html junit markdown github; do
  INPUT_CONFIG="" \
  INPUT_PROFILE="" \
  INPUT_WORKING_DIRECTORY="" \
  INPUT_FORMAT="$fmt" \
  INPUT_SKIP_FMT="false" \
  INPUT_SKIP_STYLE="false" \
  INPUT_SKIP_LINT="false" \
  INPUT_SKIP_POLICY="false" \
  INPUT_PARALLEL="false" \
  INPUT_FAIL_ON_ERROR="false" \
  INPUT_FAIL_ON_WARNING="false" \
  expect_pass "valid format: $fmt" bash "$RUN_ACTION"
done

# Test: Paths with spaces should pass
INPUT_CONFIG="path with spaces/config.yaml" \
INPUT_PROFILE="my profile" \
INPUT_WORKING_DIRECTORY="./my directory/terraform" \
INPUT_FORMAT="text" \
INPUT_SKIP_FMT="false" \
INPUT_SKIP_STYLE="false" \
INPUT_SKIP_LINT="false" \
INPUT_SKIP_POLICY="false" \
INPUT_PARALLEL="false" \
INPUT_FAIL_ON_ERROR="false" \
INPUT_FAIL_ON_WARNING="false" \
expect_pass "paths with spaces" bash "$RUN_ACTION"

# Test: Paths with dots and underscores should pass
INPUT_CONFIG=".config/terra_tidy.yaml" \
INPUT_PROFILE="prod_v2.1" \
INPUT_WORKING_DIRECTORY="./infra_v2/terraform.d" \
INPUT_FORMAT="text" \
INPUT_SKIP_FMT="false" \
INPUT_SKIP_STYLE="false" \
INPUT_SKIP_LINT="false" \
INPUT_SKIP_POLICY="false" \
INPUT_PARALLEL="false" \
INPUT_FAIL_ON_ERROR="false" \
INPUT_FAIL_ON_WARNING="false" \
expect_pass "paths with dots and underscores" bash "$RUN_ACTION"

echo ""
echo "=========================================="
echo "Results: $PASSED passed, $FAILED failed"

if [ $FAILED -gt 0 ]; then
  exit 1
fi
