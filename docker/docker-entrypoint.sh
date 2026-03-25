#!/bin/sh
# =============================================================================
# Isolate Panel Docker Entrypoint Script
# =============================================================================
# This script performs initialization tasks before starting the application
# =============================================================================

set -e

echo "=== Isolate Panel Docker Entrypoint ==="

# -----------------------------------------------------------------------------
# Environment Validation
# -----------------------------------------------------------------------------
echo "Validating environment variables..."

if [ -z "$JWT_SECRET" ] || [ "$JWT_SECRET" = "change-this-in-production-use-a-strong-random-secret" ]; then
    echo "ERROR: JWT_SECRET is not set or uses default value!"
    echo "Please set JWT_SECRET in your .env file or docker-compose.yml"
    echo "Generate a strong secret: openssl rand -base64 64"
    exit 1
fi

if [ -z "$DATABASE_PATH" ]; then
    echo "WARNING: DATABASE_PATH not set, using default: /app/data/isolate-panel.db"
    export DATABASE_PATH="/app/data/isolate-panel.db"
fi

if [ -z "$LOG_LEVEL" ]; then
    export LOG_LEVEL="info"
fi

if [ -z "$MONITORING_MODE" ]; then
    export MONITORING_MODE="lite"
fi

echo "Environment validation passed"

# -----------------------------------------------------------------------------
# Directory Setup
# -----------------------------------------------------------------------------
echo "Setting up directories..."

# Create data directories
mkdir -p /app/data/cores/xray
mkdir -p /app/data/cores/singbox
mkdir -p /app/data/cores/mihomo
mkdir -p /app/data/backups
mkdir -p /app/data/certificates
mkdir -p /app/data/geo

# Create log directories
mkdir -p /var/log/supervisor
mkdir -p /var/run

# Set permissions
chown -R isolate:isolate /app/data
chown -R isolate:isolate /var/log/supervisor
chown -R isolate:isolate /var/run

echo "Directories setup complete"

# -----------------------------------------------------------------------------
# Core Binaries Check
# -----------------------------------------------------------------------------
echo "Checking core binaries..."

for core in xray sing-box mihomo wgcf; do
    if ! command -v "$core" >/dev/null 2>&1; then
        echo "ERROR: Core binary '$core' not found!"
        exit 1
    fi
    echo "  - $core: $(command -v "$core")"
done

echo "Core binaries check passed"

# -----------------------------------------------------------------------------
# Database Check
# -----------------------------------------------------------------------------
if [ ! -f "$DATABASE_PATH" ]; then
    echo "Database not found at $DATABASE_PATH"
    echo "Database will be created on first run"
fi

# -----------------------------------------------------------------------------
# Display Configuration
# -----------------------------------------------------------------------------
echo ""
echo "=== Configuration ==="
echo "APP_ENV:        ${APP_ENV:-production}"
echo "PORT:           ${PORT:-8080}"
echo "LOG_LEVEL:      ${LOG_LEVEL}"
echo "MONITORING_MODE:${MONITORING_MODE}"
echo "DATABASE_PATH:  ${DATABASE_PATH}"
echo "TZ:             ${TZ:-UTC}"
echo ""

# -----------------------------------------------------------------------------
# Start Application
# -----------------------------------------------------------------------------
echo "=== Starting Isolate Panel ==="

# Execute the main command (CMD from Dockerfile)
exec "$@"
