# =============================================================================
# Caddy Reverse Proxy Dockerfile for Isolate Panel
# =============================================================================
# This Dockerfile builds a custom Caddy image with:
# - Official Caddy base image
# - Custom Caddyfile configuration
# - Additional modules (rate limiting, etc.)
# =============================================================================

FROM caddy:2.8-builder AS builder

# Build Caddy with additional modules
# - http.rate-limit: Rate limiting support
# - http.ip_source.cloudflare: Cloudflare IP source for rate limiting
RUN xcaddy build \
    --with github.com/mholt/caddy-ratelimit \
    --with github.com/WeidiDeng/caddy-cloudflare-ip

# =============================================================================
# Final Image
# =============================================================================
FROM caddy:2.8-alpine

# Copy custom-built Caddy binary
COPY --from=builder /usr/bin/caddy /usr/bin/caddy

# Install additional tools for health checks
RUN apk add --no-cache curl ca-certificates tzdata

# Create log directory
RUN mkdir -p /var/log/caddy && \
    chown -R caddy:caddy /var/log/caddy

# Copy Caddyfile configuration
COPY Caddyfile /etc/caddy/Caddyfile

# Set proper permissions
RUN chown caddy:caddy /etc/caddy/Caddyfile

# Switch to non-root user (Caddy runs as caddy by default in official image)
USER caddy

# Expose ports
EXPOSE 80 443 443/udp 2019

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["caddy", "version"] || curl -f http://localhost:2019/config/ || exit 1

# Default command
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
