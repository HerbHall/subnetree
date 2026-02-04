# NetVantage Server - Multi-stage Dockerfile
# Builds the Go server with embedded frontend assets

# ============================================================================
# Stage 1: Build frontend
# ============================================================================
FROM node:22-alpine AS frontend-builder

WORKDIR /build/web

# Install dependencies first (cacheable layer)
COPY web/package*.json ./
RUN npm ci

# Copy frontend source and build
COPY web/ ./
RUN npm run build

# ============================================================================
# Stage 2: Build Go binary
# ============================================================================
FROM golang:1.25-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go.mod/go.sum first (cacheable layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets from previous stage
COPY --from=frontend-builder /build/web/dist ./web/dist

# Build arguments for version injection
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w \
        -X github.com/HerbHall/netvantage/internal/version.Version=${VERSION} \
        -X github.com/HerbHall/netvantage/internal/version.Commit=${COMMIT} \
        -X github.com/HerbHall/netvantage/internal/version.BuildTime=${BUILD_TIME}" \
    -o /netvantage \
    ./cmd/netvantage

# ============================================================================
# Stage 3: Runtime image
# ============================================================================
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    # Network scanning tools (optional, for ICMP without raw sockets)
    iputils

# Create non-root user
RUN addgroup -g 1000 netvantage && \
    adduser -u 1000 -G netvantage -s /bin/sh -D netvantage

# Create data directory
RUN mkdir -p /data && chown netvantage:netvantage /data

# Copy binary from builder
COPY --from=go-builder /netvantage /usr/local/bin/netvantage

# Copy timezone data
COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo

# Set working directory
WORKDIR /data

# Switch to non-root user
USER netvantage

# Expose ports
# 8080 - HTTP (Web UI + REST API)
# 9090 - gRPC (Scout agent communication)
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Default environment
ENV NV_DATABASE_DSN=/data/netvantage.db \
    NV_LOG_LEVEL=info \
    NV_HTTP_ADDRESS=:8080 \
    NV_GRPC_ADDRESS=:9090

# Volume for persistent data
VOLUME ["/data"]

# Run the server
ENTRYPOINT ["netvantage"]
CMD ["serve"]
