.PHONY: build build-all test clean run lint release-snapshot install help

# Variables
BINARY_NAME=pm-agents
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=bin
GO=go

# Build the application (single binary for local development)
build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/pm-agents

# Build all platform-specific binaries
build-all:
	@mkdir -p $(BUILD_DIR)
	@echo "Building platform-specific binaries..."
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/pm-agents
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/pm-agents
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/pm-agents
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/pm-agents
	@chmod +x $(BUILD_DIR)/$(BINARY_NAME)-*
	@echo "All platform binaries built in $(BUILD_DIR)/ directory"

# Install to GOPATH/bin
install:
	$(GO) install -ldflags "-X main.Version=$(VERSION)" ./cmd/pm-agents

# Run tests
test:
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run tests with coverage report
test-coverage: test
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf outputs/
	rm -f $(BINARY_NAME)
	rm -f coverage.txt coverage.html

# Run the application in development mode
run-dev:
	$(GO) run ./cmd/pm-agents

# Run the built binary
run:
	./$(BUILD_DIR)/$(BINARY_NAME)

# Create a release snapshot using GoReleaser
release-snapshot:
	goreleaser release --snapshot --clean

# Run linting checks (same as CI)
lint:
	@echo "Running linters..."
	@$(GO) mod tidy
	@if ! git diff --quiet go.mod go.sum 2>/dev/null; then echo "go.mod or go.sum is not tidy, run 'go mod tidy'"; git diff go.mod go.sum; exit 1; fi
	@if ! command -v golangci-lint &> /dev/null; then echo "Installing golangci-lint..." && $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; fi
	@golangci-lint run --timeout=5m

# Format code
fmt:
	$(GO) fmt ./...

# Download and verify dependencies
deps:
	$(GO) mod download
	$(GO) mod verify
	$(GO) mod tidy

# Initialize config in current directory
init: build
	./$(BUILD_DIR)/$(BINARY_NAME) init

# Check if binary works
verify: build
	./$(BUILD_DIR)/$(BINARY_NAME) --help
	./$(BUILD_DIR)/$(BINARY_NAME) agents list

# Default target
all: clean deps lint test build

# Show help
help:
	@echo "PM Agent Workflow - Makefile targets"
	@echo ""
	@echo "Build:"
	@echo "  build          Build the binary for current platform"
	@echo "  build-all      Build for all platforms (darwin, linux)"
	@echo "  install        Install to GOPATH/bin"
	@echo ""
	@echo "Development:"
	@echo "  run-dev        Run with go run"
	@echo "  run            Run the built binary"
	@echo "  fmt            Format code"
	@echo "  lint           Run linters"
	@echo "  deps           Download and verify dependencies"
	@echo ""
	@echo "Testing:"
	@echo "  test           Run tests with race detection"
	@echo "  test-coverage  Run tests and generate coverage report"
	@echo ""
	@echo "Release:"
	@echo "  release-snapshot  Create snapshot release with goreleaser"
	@echo ""
	@echo "Other:"
	@echo "  clean          Clean build artifacts"
	@echo "  init           Initialize pm-agents config"
	@echo "  verify         Build and verify binary works"
	@echo "  all            Clean, lint, test, and build"
	@echo "  help           Show this help"
