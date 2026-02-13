# ============================================================================
# Poly-Go Makefile
# ============================================================================

# Binary name
BINARY := poly

# Version from git tags, fallback to "dev"
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")

# Go build flags
LDFLAGS := -X main.version=$(VERSION)

# Default target
.DEFAULT_GOAL := build

# ============================================================================
# Build targets
# ============================================================================

## build: Compile the binary (default)
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## dev: Build with race detector enabled
dev:
	go build -race -ldflags "$(LDFLAGS)" -o $(BINARY) .

## release: Optimized build with stripped symbols + version
release:
	go build -ldflags "$(LDFLAGS) -s -w" -o $(BINARY) .

## install: Install binary to ~/go/bin/
install: build
	cp $(BINARY) ~/go/bin/

# ============================================================================
# Test targets
# ============================================================================

## test: Run all tests
test:
	go test ./...

## test-verbose: Run tests with verbose output
test-verbose:
	go test -v ./...

## test-coverage: Run tests with coverage report (text + HTML)
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ============================================================================
# Code quality
# ============================================================================

## lint: Run go vet on all packages
lint:
	go vet ./...

## fmt: Format all Go source files
fmt:
	gofmt -w .

# ============================================================================
# Utility targets
# ============================================================================

## clean: Remove binary and temporary files
clean:
	rm -f $(BINARY)
	rm -f coverage.out coverage.html
	rm -rf tmp/

## setup-42: Run 42 campus setup script
setup-42:
	bash scripts/setup-42.sh

## sandbox-setup: Pull the sandbox container image
sandbox-setup:
	@RT=$$(command -v podman 2>/dev/null || command -v docker 2>/dev/null); \
	if [ -z "$$RT" ]; then echo "Error: install podman or docker first"; exit 1; fi; \
	echo "Pulling sandbox image (alpine:latest)..."; \
	$$RT pull alpine:latest && echo "Sandbox ready."

## version: Print the current version
version:
	@echo $(VERSION)

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'

# ============================================================================
# Phony declarations
# ============================================================================

.PHONY: build dev release install test test-verbose test-coverage lint fmt clean setup-42 sandbox-setup version help
