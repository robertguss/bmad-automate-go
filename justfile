# Binary name
binary_name := "bmad"

# Build directory
build_dir := "./bin"

# Main package path
main_pkg := "./cmd/bmad"

# Version info
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
commit := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
date := `date -u +"%Y-%m-%dT%H:%M:%SZ"`

# Build flags
ldflags := "-s -w -X main.version=" + version + " -X main.commit=" + commit + " -X main.date=" + date

# Default recipe to show help
default: help

# Build the application
build:
    @echo "Building {{binary_name}} {{version}}..."
    @mkdir -p {{build_dir}}
    go build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}} {{main_pkg}}
    @echo "Build complete: {{build_dir}}/{{binary_name}}"

# Build with CGO disabled for portable binaries
build-static:
    @echo "Building static {{binary_name}} {{version}}..."
    @mkdir -p {{build_dir}}
    CGO_ENABLED=0 go build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}} {{main_pkg}}
    @echo "Build complete: {{build_dir}}/{{binary_name}}"

# Run the application
run:
    @echo "Running {{binary_name}}..."
    go run {{main_pkg}}

# Run with specific sprint-status.yaml for testing
run-test:
    @echo "Running {{binary_name}} with test data..."
    go run {{main_pkg}}

# Clean build artifacts
clean:
    @echo "Cleaning..."
    go clean
    rm -rf {{build_dir}}
    @echo "Clean complete"

# Run tests
test:
    @echo "Running tests..."
    go test -v ./...

# Run tests with coverage
test-coverage:
    @echo "Running tests with coverage..."
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
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
    gofmt -s -w .
    @echo "Format complete"

# Download dependencies
deps:
    @echo "Downloading dependencies..."
    go mod download
    go mod tidy
    @echo "Dependencies updated"

# Install the binary to GOPATH/bin
install: build
    @echo "Installing {{binary_name}}..."
    cp {{build_dir}}/{{binary_name}} "$(go env GOPATH)/bin/"
    @echo "Installed to $(go env GOPATH)/bin/{{binary_name}}"

# Development: run with live reload (requires air)
dev:
    @if command -v air > /dev/null; then \
        air; \
    else \
        echo "air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
        echo "Falling back to regular run..."; \
        just run; \
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
    docker build -t bmad:{{version}} .
    docker tag bmad:{{version}} bmad:latest
    @echo "Docker image built: bmad:{{version}}"

# Run with API server enabled
run-api:
    @echo "Running {{binary_name}} with API server..."
    go run {{main_pkg}} --api --port 8080

# Run with watch mode enabled
run-watch:
    @echo "Running {{binary_name}} with watch mode..."
    go run {{main_pkg}} --watch

# Check if all dependencies compile
check:
    @echo "Checking dependencies..."
    go build ./...
    @echo "All packages compile successfully"

# Show version info
version:
    @echo "Version: {{version}}"
    @echo "Commit:  {{commit}}"
    @echo "Date:    {{date}}"

# Show help
help:
    @echo "BMAD Automate - Justfile Commands"
    @echo ""
    @echo "Usage: just [recipe]"
    @echo ""
    @echo "Recipes:"
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
