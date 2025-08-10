# Go build configuration
GO := go
GOFMT := gofmt
GOLINT := golangci-lint

# Project configuration
PROJECT_NAME := etcd-fuzzing
BINARY_NAME := etcd-fuzzer
PKG := github.com/ds-testing-user/etcd-fuzzing
BUILD_DIR := bin
RESULTS_DIR := results
TRACES_DIR := traces

# Build information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go build flags
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME)
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

# Default target
.DEFAULT_GOAL := build

# Help target
.PHONY: help
help: ## Show this help message
	@echo "$(PROJECT_NAME) - Makefile targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build targets
.PHONY: build
build: ## Build the main binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

.PHONY: build-all
build-all: build build-examples ## Build all binaries including examples

.PHONY: build-examples
build-examples: ## Build example binaries
	@echo "Building pubsub example..."
	@mkdir -p $(BUILD_DIR)/examples
	$(GO) build -o $(BUILD_DIR)/examples/pubsub-example ./pubsub/example

# Development targets
.PHONY: run
run: build ## Build and run the fuzzer with default parameters
	./$(BUILD_DIR)/$(BINARY_NAME) fuzz

.PHONY: run-compare
run-compare: build ## Build and run the comparison mode
	./$(BUILD_DIR)/$(BINARY_NAME) compare

# Test targets
.PHONY: test
test: ## Run unit tests
	$(GO) test -v ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	$(GO) test -race -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@mkdir -p $(BUILD_DIR)/coverage
	$(GO) test -coverprofile=$(BUILD_DIR)/coverage/coverage.out ./...
	$(GO) tool cover -html=$(BUILD_DIR)/coverage/coverage.out -o $(BUILD_DIR)/coverage/coverage.html
	@echo "Coverage report generated: $(BUILD_DIR)/coverage/coverage.html"

.PHONY: test-integration
test-integration: ## Run integration tests
	$(GO) test -tags=integration -v ./pubsub/example/

# Code quality targets
.PHONY: fmt
fmt: ## Format Go code
	$(GOFMT) -s -w .

.PHONY: fmt-check
fmt-check: ## Check if code is formatted
	@test -z "$$($(GOFMT) -l .)" || (echo "Code is not formatted. Run 'make fmt' to fix." && exit 1)

.PHONY: lint
lint: ## Run linter
	$(GOLINT) run

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: check
check: fmt-check vet lint ## Run all code quality checks

# Dependencies
.PHONY: deps
deps: ## Download dependencies
	$(GO) mod download

.PHONY: deps-update
deps-update: ## Update dependencies
	$(GO) mod tidy
	$(GO) get -u ./...

.PHONY: deps-verify
deps-verify: ## Verify dependencies
	$(GO) mod verify

# Clean targets
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

.PHONY: clean-all
clean-all: clean clean-results ## Clean everything including results and traces

.PHONY: clean-results
clean-results: ## Clean results and traces
	@echo "Cleaning results and traces..."
	rm -rf $(RESULTS_DIR)/*
	rm -rf $(TRACES_DIR)/*

# Docker targets (if needed)
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(PROJECT_NAME):$(VERSION) .

# Installation targets
.PHONY: install
install: build ## Install binary to GOPATH/bin
	$(GO) install $(BUILD_FLAGS) .

# Development setup
.PHONY: setup
setup: deps ## Setup development environment
	@echo "Setting up development environment..."
	@command -v golangci-lint >/dev/null 2>&1 || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@mkdir -p $(BUILD_DIR)
	@mkdir -p $(RESULTS_DIR)
	@mkdir -p $(TRACES_DIR)

# Benchmarking
.PHONY: bench
bench: build ## Run benchmarks
	./$(BUILD_DIR)/$(BINARY_NAME) fuzz --episodes 1000 --horizon 20

# Debug build
.PHONY: debug
debug: ## Build with debug information
	@echo "Building debug version..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-debug .

# Cross-platform builds
.PHONY: build-linux
build-linux: ## Build for Linux
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux .

.PHONY: build-windows
build-windows: ## Build for Windows
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows.exe .

.PHONY: build-darwin
build-darwin: ## Build for macOS
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin .

# Protobuf generation (if needed)
.PHONY: proto
proto: ## Generate protobuf files
	$(GO) generate ./...

# Version information
.PHONY: version
version: ## Show version information
	@echo "Project: $(PROJECT_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Build Time: $(BUILD_TIME)"
