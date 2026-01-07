.PHONY: build run clean test lint fmt deps release snapshot docker

# Binary name
BINARY_NAME=bmad

# Build directory
BUILD_DIR=./bin

# Main package path
MAIN_PKG=./cmd/bmad

# Version info
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Build the application
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PKG)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build with CGO disabled for portable binaries
build-static:
	@echo "Building static $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PKG)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	$(GORUN) $(MAIN_PKG)

# Run with specific sprint-status.yaml for testing
run-test:
	@echo "Running $(BINARY_NAME) with test data..."
	$(GORUN) $(MAIN_PKG)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Lint the code
lint:
	@echo "Linting..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format the code
fmt:
	@echo "Formatting..."
	$(GOFMT) -s -w .
	@echo "Format complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies updated"

# Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Development: run with live reload (requires air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run..."; \
		$(MAKE) run; \
	fi

# Release with goreleaser
release:
	@echo "Creating release..."
	@if command -v goreleaser > /dev/null; then \
		goreleaser release --clean; \
	else \
		echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
	fi

# Create a snapshot release (for testing)
snapshot:
	@echo "Creating snapshot release..."
	@if command -v goreleaser > /dev/null; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
	fi

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t bmad:$(VERSION) .
	docker tag bmad:$(VERSION) bmad:latest
	@echo "Docker image built: bmad:$(VERSION)"

# Run with API server enabled
run-api:
	@echo "Running $(BINARY_NAME) with API server..."
	$(GORUN) $(MAIN_PKG) --api --port 8080

# Run with watch mode enabled
run-watch:
	@echo "Running $(BINARY_NAME) with watch mode..."
	$(GORUN) $(MAIN_PKG) --watch

# Check if all dependencies compile
check:
	@echo "Checking dependencies..."
	$(GOCMD) build ./...
	@echo "All packages compile successfully"

# Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

# Show help
help:
	@echo "BMAD Automate - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build         Build the application"
	@echo "  build-static  Build with CGO disabled (portable)"
	@echo "  run           Run the application"
	@echo "  run-api       Run with API server enabled"
	@echo "  run-watch     Run with watch mode enabled"
	@echo "  clean         Remove build artifacts"
	@echo "  test          Run tests"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  lint          Run linter"
	@echo "  fmt           Format code"
	@echo "  deps          Download and tidy dependencies"
	@echo "  install       Install binary to GOPATH/bin"
	@echo "  dev           Run with live reload (requires air)"
	@echo "  release       Create release with goreleaser"
	@echo "  snapshot      Create snapshot release (testing)"
	@echo "  docker        Build Docker image"
	@echo "  check         Verify all packages compile"
	@echo "  version       Show version information"
	@echo "  help          Show this help message"
