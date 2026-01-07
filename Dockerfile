# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bmad ./cmd/bmad

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    openssh-client

# Copy binary from builder
COPY --from=builder /app/bmad /usr/local/bin/bmad

# Create non-root user
RUN adduser -D -g '' bmad
USER bmad

# Default working directory for projects
WORKDIR /workspace

# Entry point
ENTRYPOINT ["bmad"]
CMD ["--help"]
