#!/usr/bin/env bash
# Convert go mod graph output to SVG using graphviz

set -euo pipefail

# Check if graphviz is installed
if ! command -v dot &> /dev/null; then
    echo "Error: graphviz is not installed"
    echo "Install with: brew install graphviz (macOS) or apt-get install graphviz (Linux)"
    exit 1
fi

# Read from stdin or file
INPUT="${1:-/dev/stdin}"

# Convert to DOT format
{
    echo "digraph {"
    echo "  rankdir=LR;"
    echo "  node [shape=box, style=rounded];"

    # Parse go mod graph output
    while read -r line; do
        # Split on space
        from=$(echo "$line" | cut -d' ' -f1)
        to=$(echo "$line" | cut -d' ' -f2)

        # Simplify module names
        from_short=$(echo "$from" | sed 's/@v[0-9].*//')
        to_short=$(echo "$to" | sed 's/@v[0-9].*//')

        # Only show direct dependencies of our module
        if [[ "$from" == github.com/santosr2/TerraTidy* ]]; then
            echo "  \"$from_short\" -> \"$to_short\";"
        fi
    done < "$INPUT"

    echo "}"
} | dot -Tsvg
