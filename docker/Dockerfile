# =============================================================================
# Stage 1: Go Builder
# =============================================================================
FROM golang:1.23-alpine AS go-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Copy Go module files (for better layer caching)
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source and build
COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X 'main.Version=0.1.0'" \
    -o /app/server \
    ./cmd/server/main.go

# =============================================================================
# Stage 2: Node.js Builder
# =============================================================================
FROM node:22-alpine AS node-builder

WORKDIR /app

# Copy package files (for better layer caching)
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --only=production

# Copy source and build
COPY frontend/ ./
RUN npm run build

# =============================================================================
# Stage 3: Cores Downloader (separate stage for caching)
# =============================================================================
FROM alpine:3.21 AS cores-downloader

WORKDIR /cores

# Download Xray core
RUN wget -q https://github.com/XTLS/Xray-core/releases/download/v26.2.6/Xray-linux-64.zip -O /tmp/xray.zip && \
    unzip -q /tmp/xray.zip -d /usr/local/bin/ && \
    rm /tmp/xray.zip

# Download Mihomo core
RUN wget -q https://github.com/MetaCubeX/mihomo/releases/download/v1.19.21/mihomo-linux-amd64-v1.19.21.gz -O /tmp/mihomo.gz && \
    gunzip -q /tmp/mihomo.gz && \
    mv mihomo /usr/local/bin/ && \
    chmod +x /usr/local/bin/mihomo

# Download Sing-box core (v1.13.3)
RUN wget -q https://github.com/SagerNet/sing-box/releases/download/v1.13.3/sing-box-1.13.3-linux-amd64.tar.gz -O /tmp/singbox.tar.gz && \
    tar -xzf /tmp/singbox.tar.gz -C /tmp && \
    mv /tmp/sing-box-1.13.3-linux-amd64/sing-box /usr/local/bin/ && \
    chmod +x /usr/local/bin/sing-box && \
    rm -rf /tmp/singbox.tar.gz /tmp/sing-box-1.13.3-linux-amd64

# Download WARP client
RUN wget -q https://github.com/ViRb3/wgcf/releases/download/v2.2.23/wgcf_2.2.23_linux_amd64 -O /usr/local/bin/wgcf && \
    chmod +x /usr/local/bin/wgcf

# =============================================================================
# Stage 4: Production Runtime
# =============================================================================
FROM alpine:3.21

# Install ONLY runtime dependencies (minimal set)
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    sqlite \
    supervisor \
    wget

# Create non-root user for security
RUN addgroup -g 1000 isolate && \
    adduser -D -u 1000 -G isolate isolate

# Create directories
RUN mkdir -p /app/data /var/log/supervisor /var/run && \
    chown -R isolate:isolate /app/data /var/log/supervisor

# Copy binaries from builders
COPY --from=go-builder /app/server /usr/local/bin/isolate-panel
COPY --from=node-builder /app/dist /var/www/html

# Copy cores from downloader stage
COPY --from=cores-downloader /usr/local/bin/xray /usr/local/bin/xray
COPY --from=cores-downloader /usr/local/bin/mihomo /usr/local/bin/mihomo
COPY --from=cores-downloader /usr/local/bin/sing-box /usr/local/bin/sing-box
COPY --from=cores-downloader /usr/local/bin/wgcf /usr/local/bin/wgcf

# Copy supervisord config
COPY docker/supervisord.conf /etc/supervisord.conf

# Copy entrypoint script
COPY docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Set working directory
WORKDIR /app

# Environment variables
ENV APP_ENV=production \
    PORT=8080 \
    DATABASE_PATH=/app/data/isolate-panel.db \
    LOG_LEVEL=info \
    MONITORING_MODE=lite

# Expose ports
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Switch to non-root user
USER isolate

# Start supervisord
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
