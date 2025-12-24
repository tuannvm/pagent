.PHONY: build build-all test test-coverage clean run run-dev lint release-snapshot install help docker-build docker-run docker-test fmt deps init verify all

# Variables
BINARY_NAME=pagent
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=bin
GO=go
REGISTRY=ghcr.io
IMAGE_NAME=$(shell echo "$(REGISTRY)/tuannvm/pagent" | tr '[:upper:]' '[:lower:]')

# Build the application (single binary for local development)
build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/pagent

# Build all platform-specific binaries
build-all:
	@mkdir -p $(BUILD_DIR)
	@echo "Building platform-specific binaries..."
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/pagent
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/pagent
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/pagent
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/pagent
	@chmod +x $(BUILD_DIR)/$(BINARY_NAME)-*
	@echo "All platform binaries built in $(BUILD_DIR)/ directory"

# Install to GOPATH/bin
install:
	$(GO) install -ldflags "-X main.Version=$(VERSION)" ./cmd/pagent

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
	$(GO) run ./cmd/pagent

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
	@echo "Verifying pagent binary..."
	./$(BUILD_DIR)/$(BINARY_NAME) --help
	./$(BUILD_DIR)/$(BINARY_NAME) agents list

# Build Docker image
docker-build:
	docker build -t $(IMAGE_NAME):$(VERSION) -t $(IMAGE_NAME):latest --build-arg VERSION=$(VERSION) .

# Run Docker container
docker-run: docker-build
	docker run --rm -it $(IMAGE_NAME):$(VERSION) --help

# Run tests in Docker
docker-test:
	docker build -f Dockerfile --target builder -t $(BINARY_NAME)-test:$(VERSION) .
	docker run --rm $(BINARY_NAME)-test:$(VERSION) go test ./...

# Security scan
security:
	@echo "Running security checks..."
	@if ! command -v govulncheck &> /dev/null; then echo "Installing govulncheck..." && $(GO) install golang.org/x/vuln/cmd/govulncheck@latest; fi
	@govulncheck ./...

# Default target
all: clean deps lint test build

# CI target (what runs in GitHub Actions)
ci: deps lint security test build verify

# Show help
help:
	@echo "pagent - Pagent Workflow"
	@echo ""
	@echo "Build:"
	@echo "  build            Build the binary for current platform"
	@echo "  build-all        Build for all platforms (darwin, linux)"
	@echo "  install          Install to GOPATH/bin"
	@echo ""
	@echo "Development:"
	@echo "  run-dev          Run with go run"
	@echo "  run              Run the built binary"
	@echo "  fmt              Format code"
	@echo "  lint             Run linters (golangci-lint)"
	@echo "  deps             Download and verify dependencies"
	@echo ""
	@echo "Testing:"
	@echo "  test             Run tests with race detection"
	@echo "  test-coverage    Run tests and generate HTML coverage report"
	@echo "  security         Run security vulnerability scan (govulncheck)"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build     Build Docker image"
	@echo "  docker-run       Build and run Docker container"
	@echo "  docker-test      Run tests inside Docker container"
	@echo ""
	@echo "Release:"
	@echo "  release-snapshot Create snapshot release with goreleaser"
	@echo ""
	@echo "CI/CD:"
	@echo "  ci               Run full CI pipeline (deps, lint, security, test, build, verify)"
	@echo "  all              Clean, deps, lint, test, and build"
	@echo ""
	@echo "Other:"
	@echo "  clean            Clean build artifacts"
	@echo "  init             Initialize pagent config"
	@echo "  verify           Build and verify binary works"
	@echo "  help             Show this help"
