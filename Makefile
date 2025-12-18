.PHONY: help build test lint clean install setup integration dev

# Variables
BINARY_NAME=terratidy
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/terratidy

install: build ## Install the binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	@go install $(LDFLAGS) ./cmd/terratidy

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

coverage-html: ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	@mkdir -p coverage
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage/index.html
	@echo "✓ Coverage report: coverage/index.html"

lint: ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: mise install golangci-lint"; \
	fi

staticcheck: ## Run staticcheck (advanced static analysis)
	@echo "Running staticcheck..."
	@if command -v staticcheck > /dev/null; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not found. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi

revive: ## Run revive linter
	@echo "Running revive..."
	@if command -v revive > /dev/null; then \
		revive -config revive.toml -formatter friendly ./...; \
	else \
		echo "revive not found. Install with: go install github.com/mgechev/revive@latest"; \
	fi

vuln: ## Check for security vulnerabilities
	@echo "Checking for vulnerabilities..."
	@if command -v govulncheck > /dev/null; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not found. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@if command -v goimports > /dev/null; then \
		goimports -w .; \
	fi

fmt-strict: ## Format code with gofumpt (stricter than gofmt)
	@echo "Formatting code with gofumpt..."
	@if command -v gofumpt > /dev/null; then \
		gofumpt -l -w .; \
	else \
		echo "gofumpt not found. Install with: go install mvdan.cc/gofumpt@latest"; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

integration: build ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

dev: ## Run in development mode (with hot reload)
	@echo "Running in dev mode..."
	@go run $(LDFLAGS) ./cmd/terratidy dev

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf dist/
	@rm -f coverage.out coverage.html
	@go clean

clean-all: ## Clean all generated files and artifacts
	@echo "Cleaning all generated files..."
	@rm -rf bin/
	@rm -rf dist/
	@rm -rf context/
	@rm -rf coverage/
	@rm -rf benchmarks/
	@rm -rf tmp/
	@rm -rf test_fixtures/
	@rm -f coverage.out coverage.html coverage.txt
	@rm -f *.test
	@rm -f *.prof
	@go clean -cache -testcache -modcache
	@echo "✓ All generated files cleaned"

uninstall-tools: ## Uninstall all development tools
	@echo "⚠️  This will uninstall all TerraTidy development tools"
	@echo "The following will be removed:"
	@echo "  - Go tools (air, benchstat, gofumpt, staticcheck, etc.)"
	@echo "  - Node tools (repomix)"
	@echo "  - Python tools (beads-mcp)"
	@echo ""
	@read -p "Continue? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		bash tools/scripts/uninstall.sh; \
	else \
		echo "Cancelled."; \
	fi

uninstall-tools-force: ## Force uninstall tools without confirmation
	@bash tools/scripts/uninstall.sh --force

# Rule development helpers
init-rule: ## Initialize a new rule (use: make init-rule NAME=my-rule TYPE=go|yaml|bash)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make init-rule NAME=my-rule TYPE=go"; \
		exit 1; \
	fi
	@./bin/$(BINARY_NAME) init-rule --name $(NAME) --type $(TYPE)

test-rule: ## Test a specific rule (use: make test-rule RULE=custom.my-rule)
	@if [ -z "$(RULE)" ]; then \
		echo "Error: RULE is required. Usage: make test-rule RULE=custom.my-rule"; \
		exit 1; \
	fi
	@./bin/$(BINARY_NAME) test-rule $(RULE)

# Config management
config-split: build ## Split config into modular structure
	@./bin/$(BINARY_NAME) config split

config-show: build ## Show resolved configuration
	@./bin/$(BINARY_NAME) config show

config-validate: build ## Validate configuration
	@./bin/$(BINARY_NAME) config validate

# Quality checks (run all)
check: fmt vet lint test ## Run all quality checks

check-strict: fmt-strict vet lint staticcheck vuln test ## Run all checks with strict formatting and security scan

# Build for multiple platforms
build-all: ## Build for all platforms
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/terratidy
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/terratidy
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/terratidy
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/terratidy
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/terratidy
	@echo "Binaries built in bin/"

# Development tools
context: ## Generate AI context for LLMs
	@bash tools/scripts/generate-context.sh

coverage-report: ## Generate detailed coverage report
	@bash tools/scripts/coverage-report.sh

benchmark: ## Run benchmarks
	@bash tools/scripts/benchmark.sh

deps-graph: ## Generate dependency graph
	@go mod graph | bash tools/scripts/mod-graph.sh > docs/dependencies.svg
	@echo "Dependency graph saved to docs/dependencies.svg"

hot-reload: ## Run with hot reload (requires air)
	@if command -v air > /dev/null; then \
		air -c tools/air.toml; \
	else \
		echo "air not found. Install with: go install github.com/cosmtrek/air@latest"; \
	fi
