# Makefile for Timeline2SVG

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build parameters
BINARY_NAME=timeline2svg
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe

# Build flags
LDFLAGS=-ldflags="-s -w"

.PHONY: all build clean test deps fmt lint help

# Default target
all: clean deps test build

# Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v ./...

# Build for multiple platforms
build-all: build-linux build-windows build-darwin

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) -v ./...

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_WINDOWS) -v ./...

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)_darwin -v ./...

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_WINDOWS)
	rm -f $(BINARY_NAME)_darwin
	rm -f *.svg
	rm -f test_*.csv

# Run tests
test:
	$(GOTEST) -v ./...

# Test with coverage
test-coverage:
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	$(GOFMT) ./...

# Run go vet
vet:
	$(GOVET) ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Create test data
test-data:
	@echo "timestamp,title,notes" > test_timeline.csv
	@echo "2024-01-01 09:00,Project Start,Initial project kickoff meeting" >> test_timeline.csv
	@echo "2024-01-01 10:30,Team Meeting,Daily standup with development team" >> test_timeline.csv
	@echo "2024-01-01 14:00,Code Review,Review of authentication module" >> test_timeline.csv
	@echo "2024-01-01 16:00,Testing,Unit tests for user registration" >> test_timeline.csv
	@echo "2024-01-01 17:30,Deploy,Deploy to staging environment" >> test_timeline.csv

# Quick test run
quick-test: build test-data
	./$(BINARY_NAME) --csv test_timeline.csv
	@echo "Generated: test_timeline.svg"

# Debug test run
debug-test: build test-data
	./$(BINARY_NAME) --debug --csv test_timeline.csv --output debug_timeline.svg
	@echo "Generated: debug_timeline.svg with debug output"

# Install development tools
install-tools:
	$(GOGET) honnef.co/go/tools/cmd/staticcheck@latest
	$(GOGET) golang.org/x/vuln/cmd/govulncheck@latest
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run security scan
security:
	govulncheck ./...

# Run static analysis
static:
	staticcheck ./...

# Full quality check
quality: fmt vet lint static security test

# Development workflow
dev: clean deps quality build quick-test

# Release preparation
release-prep: clean deps test quality build-all
	@echo "Release binaries built successfully"
	@echo "Linux: $(BINARY_UNIX)"
	@echo "Windows: $(BINARY_WINDOWS)"
	@echo "Darwin: $(BINARY_NAME)_darwin"

# Help
help:
	@echo "Available targets:"
	@echo "  all          - Clean, download deps, test, and build"
	@echo "  build        - Build the binary"
	@echo "  build-all    - Build for all platforms (Linux, Windows, Darwin)"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run golangci-lint"
	@echo "  test-data    - Create test CSV file"
	@echo "  quick-test   - Build and run quick test"
	@echo "  debug-test   - Build and run debug test"
	@echo "  quality      - Run full quality checks"
	@echo "  dev          - Full development workflow"
	@echo "  release-prep - Prepare release binaries"
	@echo "  help         - Show this help message"
