# Build configuration
BINARY_NAME=database-mcp
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./cmd/database-mcp

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)"
BUILD_FLAGS=-trimpath $(LDFLAGS)

# Go configuration
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

.PHONY: help build build-static test test-verbose lint clean dev docker docker-build docker-run fmt vet deps tidy

# Default target
help: ## Show this help message
	@echo "Database MCP Server - Build System"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILD_FLAGS) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Built: $(BINARY_PATH)"

build-static: ## Build static binary (Linux)
	@echo "Building static binary for Linux..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_PATH)-linux-amd64 $(MAIN_PATH)

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_PATH)-linux-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_PATH)-darwin-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BINARY_PATH)-darwin-arm64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_PATH)-windows-amd64.exe $(MAIN_PATH)

test: ## Run tests
	@echo "Running tests..."
	go test -race -cover ./...

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	go test -race -cover -v ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Development environment
dev: ## Start development environment with Docker Compose
	@echo "Starting development environment..."
	docker-compose up -d postgres clickhouse
	@echo "Databases are starting up..."
	@echo "PostgreSQL: localhost:5432 (user: postgres, pass: password, db: testdb)"
	@echo "ClickHouse: localhost:9000 (user: default, db: default)"

dev-stop: ## Stop development environment
	@echo "Stopping development environment..."
	docker-compose down

dev-logs: ## Show development environment logs
	docker-compose logs -f

# Docker targets
docker: docker-build ## Build and run Docker container

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) -t $(BINARY_NAME):latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -it \
		-e DB_PRIMARY_URL="postgres://postgres:password@host.docker.internal:5432/testdb?sslmode=disable" \
		-e DB_ANALYTICS_URL="clickhouse://host.docker.internal:9000/default" \
		$(BINARY_NAME):latest

# Install target for local development
install: build ## Install binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	cp $(BINARY_PATH) $(GOPATH)/bin/

# Check all the things
check: fmt vet lint test ## Run all checks (format, vet, lint, test)

.DEFAULT_GOAL := help