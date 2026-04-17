# ===============================================================================
# Gore Lint Docker Image
# Multi-stage build for minimal image size
# ===============================================================================

# -------------------------------------------------------------------------------
# Stage 1: Builder
# -------------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Pre-copy go mod files for better layer caching
COPY go.mod go.sum* ./

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Build arguments
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

# Build the binary with version info
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags=" \
        -s -w \
        -X main.version=${VERSION} \
        -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') \
        -X main.date=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
    " -o gore-lint ./cmd/gore-lint

# -------------------------------------------------------------------------------
# Stage 2: Runner
# -------------------------------------------------------------------------------
FROM alpine:3.19 AS runner

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    mysql-client \
    && addgroup -g 1000 gore \
    && adduser -u 1000 -G gore -s /bin/sh -D gore

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/gore-lint /app/

# Set ownership
RUN chown -R gore:gore /app

# Switch to non-root user
USER gore

# Set environment variables
ENV GORE_ENV=production

# Expose port (for future HTTP server mode)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
ENTRYPOINT ["/app/gore-lint"]
CMD ["--help"]

# ===============================================================================
# Labels
# ===============================================================================
LABEL org.opencontainers.image.title="Gore Lint"
LABEL org.opencontainers.image.description="SQL index advisor and linter for Go"
LABEL org.opencontainers.image.source="https://github.com/gore-io/gore"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="latest"
LABEL org.opencontainers.image.created=""
