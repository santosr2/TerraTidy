#!/usr/bin/env bash
# Uninstall all TerraTidy development tools

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if force mode
FORCE=false
if [[ "${1:-}" == "--force" ]]; then
    FORCE=true
fi

echo -e "${BLUE}TerraTidy Tool Uninstaller${NC}"
echo ""

# Confirmation unless force mode
if [[ "$FORCE" == "false" ]]; then
    echo -e "${YELLOW}⚠️  Warning: This will uninstall all development tools installed by TerraTidy${NC}"
    echo ""
    echo "This includes:"
    echo "  - Go tools in \$(go env GOPATH)/bin"
    echo "  - Go tools (air, benchstat, gofumpt, staticcheck, etc.)"
    echo ""
    echo "Note: This will NOT uninstall:"
    echo "  - Go itself (managed by mise)"
    echo "  - Node.js itself (managed by mise)"
    echo "  - Python itself (system package)"
    echo "  - mise itself"
    echo "  - MCP server configurations in Claude Desktop"
    echo ""
    read -p "Do you want to continue? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${GREEN}Cancelled. No changes made.${NC}"
        exit 0
    fi
    echo ""
fi

# Function to remove Go binary if it exists
remove_go_bin() {
    local bin_name="$1"
    local bin_path="$(go env GOPATH)/bin/$bin_name"

    if [[ -f "$bin_path" ]]; then
        rm -f "$bin_path"
        echo -e "${GREEN}✓${NC} Removed: $bin_name"
        return 0
    else
        echo -e "${YELLOW}○${NC} Not found: $bin_name"
        return 1
    fi
}

# Uninstall Go tools
echo -e "${BLUE}Removing Go tools...${NC}"

GO_TOOLS=(
    "benchstat"
    "gofumpt"
    "staticcheck"
    "revive"
    "govulncheck"
)

for tool in "${GO_TOOLS[@]}"; do
    remove_go_bin "$tool"
done

echo ""

# Clean generated files
echo -e "${BLUE}Cleaning generated files...${NC}"

CLEAN_DIRS=(
    "coverage"
    "benchmarks"
    "tmp"
    "bin"
    "dist"
)

for dir in "${CLEAN_DIRS[@]}"; do
    if [[ -d "$PROJECT_ROOT/$dir" ]]; then
        rm -rf "$PROJECT_ROOT/$dir"
        echo -e "${GREEN}✓${NC} Removed directory: $dir/"
    fi
done

CLEAN_FILES=(
    "coverage.out"
    "coverage.html"
    "coverage.txt"
)

for file in "${CLEAN_FILES[@]}"; do
    if [[ -f "$PROJECT_ROOT/$file" ]]; then
        rm -f "$PROJECT_ROOT/$file"
        echo -e "${GREEN}✓${NC} Removed file: $file"
    fi
done

# Clean Go caches
echo ""
echo -e "${BLUE}Cleaning Go caches...${NC}"
go clean -cache -testcache 2>/dev/null || true
echo -e "${GREEN}✓${NC} Go caches cleaned"

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✓ Uninstall complete!${NC}"
echo ""
echo "What was NOT removed (intentionally):"
echo "  • Go (managed by mise)"
echo "  • mise itself"
echo "  • golangci-lint, terraform, terragrunt (managed by mise)"
echo "  • Project source code and documentation"
echo ""
echo "To reinstall everything:"
echo "  ${BLUE}mise install && mise run setup${NC}"
echo ""
echo "To remove mise-managed tools:"
echo "  ${BLUE}mise uninstall go@1.25${NC}"
echo "  ${BLUE}# etc.${NC}"
echo ""
