# Multi-stage Dockerfile for Database MCP Server
# Build stage
FROM golang:1.21-alpine AS builder

# Install git for version info
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev') -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.date=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
    -trimpath \
    -o database-mcp \
    ./cmd/database-mcp

# Final stage
FROM gcr.io/distroless/static:nonroot

# Copy binary from builder
COPY --from=builder /app/database-mcp /usr/local/bin/database-mcp

# Use non-root user
USER nonroot:nonroot

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/database-mcp"]

# Default command shows help
CMD ["--help"]

# Labels
LABEL org.opencontainers.image.title="Database MCP Server"
LABEL org.opencontainers.image.description="A secure database interface for AI assistants via Model Context Protocol"
LABEL org.opencontainers.image.source="https://github.com/iva198/database-mcp"
LABEL org.opencontainers.image.documentation="https://github.com/iva198/database-mcp/blob/main/README.md"