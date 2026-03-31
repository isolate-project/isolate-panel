# =============================================================================
# Stage 1: Go Builder
# =============================================================================
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

RUN apk add --no-cache git gcc musl-dev sqlite-dev
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X 'main.Version=0.1.0'" \
    -o /app/server \
    ./cmd/server/main.go
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/isolate-migrate \
    ./cmd/migrate/main.go

# =============================================================================
# Stage 2: Node.js Builder
# =============================================================================
FROM node:22-alpine AS node-builder

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm install

COPY frontend/ ./
RUN npm run build

# =============================================================================
# Stage 3: Final Image
# =============================================================================
FROM alpine:3.21

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    sqlite \
    supervisor \
    wget \
    curl \
    bash

# Copy compiled Go binary
COPY --from=go-builder /app/server /usr/local/bin/isolate-panel
COPY --from=go-builder /app/isolate-migrate /usr/local/bin/isolate-migrate

# Copy compiled frontend
COPY --from=node-builder /app/dist /var/www/html

# Copy pre-downloaded cores to system directory (not volume)
COPY docker/cores/xray/xray /usr/local/bin/cores/xray
COPY docker/cores/mihomo/mihomo /usr/local/bin/cores/mihomo
COPY docker/cores/singbox/sing-box /usr/local/bin/cores/sing-box

# Copy default config
COPY docker/config.yaml /app/configs/config.yaml

# Set default environment variables
ENV PORT=8080
ENV CONFIG_PATH=/app/configs/config.yaml
ENV DATABASE_PATH=/app/data/isolate-panel.db
ENV LOG_LEVEL=info
ENV APP_ENV=production

# Set executable permissions for cores
RUN mkdir -p /app/configs && \
    chmod +x /usr/local/bin/cores/xray \
    && chmod +x /usr/local/bin/cores/mihomo \
    && chmod +x /usr/local/bin/cores/sing-box \
    && echo "✅ Cores pre-installed in image"

# Copy configuration files
COPY docker/supervisord.conf /etc/supervisord.conf
COPY docker/docker-entrypoint.sh /docker-entrypoint.sh

# Create directories and set permissions
RUN mkdir -p /app/data /var/log/isolate-panel /var/log/supervisor && \
    chmod +x /docker-entrypoint.sh

# Expose ports
EXPOSE 8080 443 8443

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
