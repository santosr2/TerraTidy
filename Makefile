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

lint: ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: mise install golangci-lint"; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@if command -v goimports > /dev/null; then \
		goimports -w .; \
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
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf dist/
	@rm -f coverage.out coverage.html
	@go clean

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

# Build for multiple platforms
build-all: ## Build for all platforms
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/terratidy
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/terratidy
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/terratidy
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/terratidy
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/terratidy
	@echo "Binaries built in bin/"

