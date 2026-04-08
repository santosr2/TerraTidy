#!/usr/bin/env bash
set -euo pipefail

# Validate argument
if [ $# -lt 1 ] || [ -z "${1:-}" ]; then
  echo '{"findings": []}' >&2
  echo "Usage: no-hardcoded-account-id.sh <file>" >&2
  exit 1
fi

FILE="$1"

# Match 12-digit AWS account IDs (standalone, not inside a variable reference)
PATTERN='[^a-zA-Z_$][0-9]{12}[^0-9]'

findings="[]"

while IFS= read -r match; do
  line=$(echo "$match" | cut -d: -f1)
  findings=$(echo "$findings" | jq --arg file "$FILE" --arg line "$line" \
    '. + [{"file": $file, "line": ($line | tonumber), "message": "Hardcoded AWS account ID detected; use a variable or data source", "severity": "warning"}]')
done < <(grep -nE "$PATTERN" "$FILE" 2>/dev/null || true)

echo "{\"findings\": $findings}"
