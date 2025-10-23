# fh - Fast History
# Makefile for development tasks

.PHONY: help build test coverage lint install clean run fmt vet

# Default Go version
GO := go
BINARY_NAME := fh
MAIN_PATH := ./cmd/fh

# Build variables
BUILD_DIR := build
DIST_DIR := dist
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

## help: Display this help message
help:
	@echo "fh - Fast History"
	@echo ""
	@echo "Available targets:"
	@awk '/^##/ { \
		sub(/^## /, "", $$0); \
		printf "  \033[36m%-15s\033[0m %s\n", $$1, substr($$0, index($$0, $$2)); \
	}' $(MAKEFILE_LIST)

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v -race ./...

## coverage: Generate coverage report
coverage:
	@echo "Generating coverage report..."
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run linters
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	gofmt -s -w .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## install: Install binary to $GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installed: $(GOPATH)/bin/$(BINARY_NAME)"

## clean: Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	rm -f coverage.txt coverage.html
	$(GO) clean
	@echo "Clean complete"

## run: Build and run the binary
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"
