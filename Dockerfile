# Multi-stage build for optimal image size
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /build

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o qwelli ./cmd/qwelli

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/qwelli .

# Copy web UI assets if they exist
COPY --from=builder /build/web/dist ./web/dist

# Create non-root user
RUN addgroup -g 1000 qwelli && \
    adduser -D -u 1000 -G qwelli qwelli && \
    chown -R qwelli:qwelli /app

USER qwelli

# Expose port for web UI
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command - start server
CMD ["./qwelli", "serve", "--port", "8080"]
