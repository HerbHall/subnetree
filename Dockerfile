# SubNetree Server - Multi-stage Dockerfile
# Builds the Go server with embedded frontend assets

# ============================================================================
# Stage 1: Build frontend
# ============================================================================
FROM node:22-alpine AS frontend-builder

# Enable corepack for pnpm
RUN corepack enable && corepack prepare pnpm@9 --activate

WORKDIR /build/web

# Install dependencies first (cacheable layer)
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Copy frontend source and build
COPY web/ ./
RUN pnpm run build

# ============================================================================
# Stage 2: Build Go binary
# ============================================================================
FROM golang:1.25.9-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go.mod/go.sum first (cacheable layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets to where Go embed expects them
COPY --from=frontend-builder /build/web/dist ./internal/dashboard/dist

# Build arguments for version injection
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Build the server binary (cache Go modules and build cache for faster rebuilds)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w \
        -X github.com/HerbHall/subnetree/internal/version.Version=${VERSION} \
        -X github.com/HerbHall/subnetree/internal/version.GitCommit=${COMMIT} \
        -X github.com/HerbHall/subnetree/internal/version.BuildDate=${BUILD_TIME}" \
    -o /subnetree \
    ./cmd/subnetree

# ============================================================================
# Stage 3: Runtime image
# ============================================================================
FROM alpine:3.21

# OCI image labels
LABEL org.opencontainers.image.title="SubNetree" \
      org.opencontainers.image.description="Network monitoring and management platform" \
      org.opencontainers.image.source="https://github.com/HerbHall/subnetree" \
      org.opencontainers.image.licenses="BSL-1.1"

# Install runtime dependencies, create user and data directory (single layer)
RUN apk add --no-cache ca-certificates tzdata iputils && \
    addgroup -g 1000 subnetree && \
    adduser -u 1000 -G subnetree -s /bin/sh -D subnetree && \
    mkdir -p /data && chown subnetree:subnetree /data

# Copy binary from builder
COPY --from=go-builder /subnetree /usr/local/bin/subnetree

# Set working directory
WORKDIR /data

# Switch to non-root user
USER subnetree

# Expose ports
# 8080 - HTTP (Web UI + REST API)
# 9090 - gRPC (Scout agent communication)
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Default environment
ENV NV_DATABASE_DSN=/data/subnetree.db \
    NV_LOG_LEVEL=info \
    NV_HTTP_ADDRESS=:8080 \
    NV_GRPC_ADDRESS=:9090

# Volume for persistent data
VOLUME ["/data"]

# Run the server
ENTRYPOINT ["subnetree"]
