.PHONY: build install test clean help run

# Binary name
BINARY_NAME=inferencehub

# Version information
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "\
	-X github.com/Vinay-Venkatesh/inferencehub-cli/pkg/version.Version=$(VERSION) \
	-X github.com/Vinay-Venkatesh/inferencehub-cli/pkg/version.Commit=$(COMMIT) \
	-X github.com/Vinay-Venkatesh/inferencehub-cli/pkg/version.BuildDate=$(BUILD_DATE)"

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/inferencehub/main.go
	@echo "Build complete: bin/$(BINARY_NAME)"

# Install to $GOPATH/bin (adds to PATH if GOPATH/bin is already on PATH)
install:
	@echo "Installing $(BINARY_NAME) to $(shell go env GOPATH)/bin/..."
	go install $(LDFLAGS) ./cmd/inferencehub
	@echo ""
	@echo "✓ Installed: $(shell go env GOPATH)/bin/$(BINARY_NAME)"
	@if echo "$$PATH" | grep -q "$(shell go env GOPATH)/bin"; then \
		echo "✓ $(shell go env GOPATH)/bin is already in PATH — run: $(BINARY_NAME) --help"; \
	else \
		echo ""; \
		echo "⚠  Add GOPATH/bin to your PATH to use '$(BINARY_NAME)' directly:"; \
		echo "   echo 'export PATH=\"\$$PATH:$(shell go env GOPATH)/bin\"' >> ~/.zshrc && source ~/.zshrc"; \
		echo "   (or ~/.bashrc if you use bash)"; \
		echo ""; \
		echo "   Or install system-wide with: make install-system"; \
	fi

# Install to /usr/local/bin (available to all users without PATH change)
install-system:
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/inferencehub/main.go
	sudo cp bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "✓ Installed: /usr/local/bin/$(BINARY_NAME)"
	@echo "✓ Run: $(BINARY_NAME) --help"

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# Run the CLI (development)
run:
	go run cmd/inferencehub/main.go

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 cmd/inferencehub/main.go
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 cmd/inferencehub/main.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 cmd/inferencehub/main.go
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe cmd/inferencehub/main.go
	@echo "Built binaries for all platforms in bin/"

# Help
help:
	@echo "InferenceHub CLI - Makefile targets:"
	@echo ""
	@echo "  build          - Build the binary to bin/"
	@echo "  install        - Install to \$$GOPATH/bin (with PATH guidance)"
	@echo "  install-system - Install to /usr/local/bin (requires sudo)"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  clean          - Remove build artifacts"
	@echo "  run            - Run the CLI in development mode"
	@echo "  fmt            - Format code"
	@echo "  lint           - Run linter"
	@echo "  deps           - Download dependencies"
	@echo "  deps-update    - Update dependencies"
	@echo "  build-all      - Build for all platforms"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
