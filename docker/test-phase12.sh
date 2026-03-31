#!/bin/bash
# =============================================================================
# Phase 12 Automated Validation Script
# Tests all Docker deployment changes without actually building containers
# (since we don't have the cores binaries and full source to build)
# =============================================================================

set -e

DOCKER_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$DOCKER_DIR")"

PASS=0
FAIL=0
WARN=0

pass() { echo "  ✅ PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  ❌ FAIL: $1"; FAIL=$((FAIL + 1)); }
warn() { echo "  ⚠️  WARN: $1"; WARN=$((WARN + 1)); }

echo "============================================="
echo "  Phase 12 Docker Deployment Validation"
echo "============================================="
echo ""

# -----------------------------------------------
# 12.1: Version checks
# -----------------------------------------------
echo "📋 12.1 — Version checks"

# Dockerfile.dev Go version
if grep -q "golang:1.25-alpine" "$DOCKER_DIR/Dockerfile.dev"; then
    pass "Dockerfile.dev uses Go 1.25"
else
    fail "Dockerfile.dev does NOT use Go 1.25"
fi

# Dockerfile Go version
if grep -q "golang:1.25-alpine" "$DOCKER_DIR/Dockerfile"; then
    pass "Dockerfile uses Go 1.25"
else
    fail "Dockerfile does NOT use Go 1.25"
fi

# Sing-box version sync
if grep -q "v1.13.3" "$DOCKER_DIR/Dockerfile.dev"; then
    pass "Dockerfile.dev uses Sing-box v1.13.3 (matches seeds.go)"
else
    fail "Dockerfile.dev Sing-box version mismatch"
fi

# ARG VERSION in Dockerfile
if grep -q "ARG VERSION" "$DOCKER_DIR/Dockerfile"; then
    pass "Dockerfile has ARG VERSION"
else
    fail "Dockerfile missing ARG VERSION"
fi

if grep -q '${VERSION}' "$DOCKER_DIR/Dockerfile"; then
    pass "Dockerfile uses \${VERSION} in ldflags"
else
    fail "Dockerfile does not use \${VERSION} in ldflags"
fi

echo ""

# -----------------------------------------------
# 12.2: supervisord.dev.conf
# -----------------------------------------------
echo "📋 12.2 — supervisord.dev.conf"

if [ -f "$DOCKER_DIR/supervisord.dev.conf" ]; then
    pass "supervisord.dev.conf exists"
else
    fail "supervisord.dev.conf does NOT exist"
fi

if grep -q "program:backend" "$DOCKER_DIR/supervisord.dev.conf"; then
    pass "supervisord.dev.conf has [program:backend] (air)"
else
    fail "supervisord.dev.conf missing [program:backend]"
fi

if grep -q "air" "$DOCKER_DIR/supervisord.dev.conf"; then
    pass "supervisord.dev.conf uses air for hot-reload"
else
    fail "supervisord.dev.conf does NOT use air"
fi

if grep -q "program:frontend" "$DOCKER_DIR/supervisord.dev.conf"; then
    pass "supervisord.dev.conf has [program:frontend] (vite)"
else
    fail "supervisord.dev.conf missing [program:frontend]"
fi

if grep -q "vite" "$DOCKER_DIR/supervisord.dev.conf"; then
    pass "supervisord.dev.conf uses vite dev server"
else
    fail "supervisord.dev.conf does NOT use vite"
fi

# Dockerfile.dev copies dev config
if grep -q "supervisord.dev.conf" "$DOCKER_DIR/Dockerfile.dev"; then
    pass "Dockerfile.dev copies supervisord.dev.conf"
else
    fail "Dockerfile.dev still copies supervisord.conf"
fi

echo ""

# -----------------------------------------------
# 12.3: Health check
# -----------------------------------------------
echo "📋 12.3 — Health check for cores"

if [ -f "$DOCKER_DIR/docker-healthcheck.sh" ]; then
    pass "docker-healthcheck.sh exists"
else
    fail "docker-healthcheck.sh does NOT exist"
fi

if [ -x "$DOCKER_DIR/docker-healthcheck.sh" ]; then
    pass "docker-healthcheck.sh is executable"
else
    fail "docker-healthcheck.sh is NOT executable"
fi

if grep -q "FATAL\|BACKOFF" "$DOCKER_DIR/docker-healthcheck.sh"; then
    pass "Healthcheck detects FATAL/BACKOFF core states"
else
    fail "Healthcheck does NOT check core process states"
fi

if grep -q "localhost:8080/health" "$DOCKER_DIR/docker-healthcheck.sh"; then
    pass "Healthcheck checks panel HTTP endpoint"
else
    fail "Healthcheck does NOT check panel HTTP"
fi

# Dockerfile references healthcheck script
if grep -q "docker-healthcheck.sh" "$DOCKER_DIR/Dockerfile"; then
    pass "Dockerfile copies healthcheck script"
else
    fail "Dockerfile does NOT copy healthcheck script"
fi

# Dockerfile HEALTHCHECK uses healthcheck script
if grep -q "HEALTHCHECK" "$DOCKER_DIR/Dockerfile" && grep -q "docker-healthcheck" "$DOCKER_DIR/Dockerfile"; then
    pass "Dockerfile HEALTHCHECK uses healthcheck script"
else
    fail "Dockerfile HEALTHCHECK does NOT use healthcheck script"
fi

# docker-compose.yml uses healthcheck script
if grep -q '"/docker-healthcheck.sh"' "$DOCKER_DIR/docker-compose.yml"; then
    pass "docker-compose.yml uses healthcheck script"
else
    fail "docker-compose.yml does NOT use healthcheck script"
fi

echo ""

# -----------------------------------------------
# 12.4: Log rotation
# -----------------------------------------------
echo "📋 12.4 — Log rotation"

# Production supervisord.conf
LOGROTATE_COUNT_PROD=$(grep -c "logfile_maxbytes" "$DOCKER_DIR/supervisord.conf" 2>/dev/null || echo "0")
if [ "$LOGROTATE_COUNT_PROD" -ge 8 ]; then
    pass "supervisord.conf has log rotation ($LOGROTATE_COUNT_PROD directives)"
else
    fail "supervisord.conf missing log rotation (only $LOGROTATE_COUNT_PROD directives, expected ≥8)"
fi

if grep -q "logfile_backups" "$DOCKER_DIR/supervisord.conf"; then
    pass "supervisord.conf has logfile_backups"
else
    fail "supervisord.conf missing logfile_backups"
fi

# Dev supervisord.dev.conf
LOGROTATE_COUNT_DEV=$(grep -c "logfile_maxbytes" "$DOCKER_DIR/supervisord.dev.conf" 2>/dev/null || echo "0")
if [ "$LOGROTATE_COUNT_DEV" -ge 8 ]; then
    pass "supervisord.dev.conf has log rotation ($LOGROTATE_COUNT_DEV directives)"
else
    fail "supervisord.dev.conf missing log rotation (only $LOGROTATE_COUNT_DEV directives, expected ≥8)"
fi

# Supervisord global log rotation
if grep -A2 "\[supervisord\]" "$DOCKER_DIR/supervisord.conf" | grep -q "logfile_maxbytes"; then
    pass "supervisord.conf has global log rotation"
else
    warn "supervisord.conf global [supervisord] log rotation not directly after section (may be OK)"
fi

echo ""

# -----------------------------------------------
# 12.5: Non-root user
# -----------------------------------------------
echo "📋 12.5 — Non-root user (security hardening)"

# Dockerfile creates isolate user
if grep -q "adduser.*isolate" "$DOCKER_DIR/Dockerfile"; then
    pass "Dockerfile creates 'isolate' user"
else
    fail "Dockerfile does NOT create 'isolate' user"
fi

if grep -q "addgroup.*isolate" "$DOCKER_DIR/Dockerfile"; then
    pass "Dockerfile creates 'isolate' group"
else
    fail "Dockerfile does NOT create 'isolate' group"
fi

# setcap for binding ports < 1024
if grep -q "setcap cap_net_bind_service" "$DOCKER_DIR/Dockerfile"; then
    pass "Dockerfile sets cap_net_bind_service on binaries"
else
    fail "Dockerfile does NOT set cap_net_bind_service"
fi

SETCAP_COUNT=$(grep -c "setcap cap_net_bind_service" "$DOCKER_DIR/Dockerfile" 2>/dev/null || echo "0")
if [ "$SETCAP_COUNT" -ge 4 ]; then
    pass "setcap applied to $SETCAP_COUNT binaries (3 cores + panel)"
else
    fail "setcap only applied to $SETCAP_COUNT binaries (expected ≥4)"
fi

# libcap installed
if grep -q "libcap" "$DOCKER_DIR/Dockerfile"; then
    pass "Dockerfile installs libcap (for setcap)"
else
    fail "Dockerfile does NOT install libcap"
fi

# supervisord.conf uses isolate user
if grep -q "user=isolate" "$DOCKER_DIR/supervisord.conf"; then
    pass "supervisord.conf runs programs as 'isolate' user"
else
    fail "supervisord.conf still runs programs as root"
fi

ROOT_USER_COUNT=$(grep -c "user=root" "$DOCKER_DIR/supervisord.conf" || true)
if [ "$ROOT_USER_COUNT" -eq 0 ]; then
    pass "supervisord.conf has no user=root references"
else
    fail "supervisord.conf still has $ROOT_USER_COUNT user=root references"
fi

# entrypoint uses isolate ownership
if grep -q "chown.*isolate:isolate" "$DOCKER_DIR/docker-entrypoint.sh"; then
    pass "docker-entrypoint.sh sets ownership to isolate:isolate"
else
    fail "docker-entrypoint.sh still uses root ownership"
fi

if grep -q "chown.*root:root" "$DOCKER_DIR/docker-entrypoint.sh"; then
    fail "docker-entrypoint.sh still has root:root chown"
else
    pass "docker-entrypoint.sh has no root:root chown"
fi

# Dockerfile chown
if grep -q "chown.*isolate:isolate" "$DOCKER_DIR/Dockerfile"; then
    pass "Dockerfile sets ownership to isolate:isolate"
else
    fail "Dockerfile does NOT set ownership to isolate"
fi

echo ""

# -----------------------------------------------
# 12.6: Deprecated version field
# -----------------------------------------------
echo "📋 12.6 — Remove deprecated version field"

if grep -q "^version:" "$DOCKER_DIR/docker-compose.yml"; then
    fail "docker-compose.yml still has deprecated 'version:' field"
else
    pass "docker-compose.yml has no deprecated 'version:' field"
fi

if grep -q "^name:" "$DOCKER_DIR/docker-compose.yml"; then
    pass "docker-compose.yml has 'name:' field"
else
    warn "docker-compose.yml missing 'name:' field (optional but recommended)"
fi

if grep -q "^version:" "$DOCKER_DIR/docker-compose.dev.yml"; then
    fail "docker-compose.dev.yml still has deprecated 'version:' field"
else
    pass "docker-compose.dev.yml has no deprecated 'version:' field"
fi

if grep -q "^name:" "$DOCKER_DIR/docker-compose.dev.yml"; then
    pass "docker-compose.dev.yml has 'name:' field"
else
    warn "docker-compose.dev.yml missing 'name:' field"
fi

echo ""

# -----------------------------------------------
# 12.7: Admin password hardcoding
# -----------------------------------------------
echo "📋 12.7 — Admin password message fix"

if grep -q 'ADMIN_PASSWORD' "$DOCKER_DIR/docker-entrypoint.sh"; then
    pass "docker-entrypoint.sh checks ADMIN_PASSWORD env"
else
    fail "docker-entrypoint.sh does NOT check ADMIN_PASSWORD"
fi

# Check that hardcoded "admin / admin" is now conditional
if grep -B1 'Default login: admin / admin' "$DOCKER_DIR/docker-entrypoint.sh" | grep -q 'else'; then
    pass "Default login message is conditional (in else branch)"
else
    fail "Default login message is NOT conditional"
fi

if grep -q 'WARNING.*Change default password' "$DOCKER_DIR/docker-entrypoint.sh"; then
    pass "Warning message shown when using default password"
else
    fail "No warning message for default password"
fi

echo ""

# -----------------------------------------------
# Bonus: docker-compose.dev.yml improvements
# -----------------------------------------------
echo "📋 Bonus — docker-compose.dev.yml improvements"

if grep -q "env_file" "$DOCKER_DIR/docker-compose.dev.yml"; then
    pass "docker-compose.dev.yml has env_file"
else
    fail "docker-compose.dev.yml missing env_file"
fi

if grep -q "frontend_node_modules" "$DOCKER_DIR/docker-compose.dev.yml"; then
    pass "docker-compose.dev.yml has named volume for node_modules"
else
    warn "docker-compose.dev.yml missing node_modules named volume"
fi

echo ""

# -----------------------------------------------
# Bonus: Syntax validity
# -----------------------------------------------
echo "📋 Bonus — Syntax validity"

# docker-compose.yml syntax
if docker compose -f "$DOCKER_DIR/docker-compose.yml" config --quiet 2>/dev/null; then
    pass "docker-compose.yml is valid YAML"
else
    fail "docker-compose.yml has syntax errors"
fi

# docker-compose.dev.yml syntax
if docker compose -f "$DOCKER_DIR/docker-compose.dev.yml" config --quiet 2>/dev/null; then
    pass "docker-compose.dev.yml is valid YAML"
else
    fail "docker-compose.dev.yml has syntax errors"
fi

# Dockerfile syntax (basic check — hadolint would be better)
if grep -q "^FROM" "$DOCKER_DIR/Dockerfile" && grep -q "^ENTRYPOINT" "$DOCKER_DIR/Dockerfile"; then
    pass "Dockerfile has required FROM and ENTRYPOINT"
else
    fail "Dockerfile missing FROM or ENTRYPOINT"
fi

if grep -q "^FROM" "$DOCKER_DIR/Dockerfile.dev" && grep -q "^CMD" "$DOCKER_DIR/Dockerfile.dev"; then
    pass "Dockerfile.dev has required FROM and CMD"
else
    fail "Dockerfile.dev missing FROM or CMD"
fi

# Check shell scripts for syntax
if bash -n "$DOCKER_DIR/docker-entrypoint.sh" 2>/dev/null; then
    pass "docker-entrypoint.sh has valid bash syntax"
else
    fail "docker-entrypoint.sh has syntax errors"
fi

if sh -n "$DOCKER_DIR/docker-healthcheck.sh" 2>/dev/null; then
    pass "docker-healthcheck.sh has valid sh syntax"
else
    fail "docker-healthcheck.sh has syntax errors"
fi

echo ""

# -----------------------------------------------
# Summary
# -----------------------------------------------
echo "============================================="
echo "  RESULTS"
echo "============================================="
echo "  ✅ Passed: $PASS"
echo "  ❌ Failed: $FAIL"
echo "  ⚠️  Warnings: $WARN"
echo "============================================="

if [ "$FAIL" -gt 0 ]; then
    echo "  STATUS: ❌ SOME TESTS FAILED"
    exit 1
else
    echo "  STATUS: ✅ ALL TESTS PASSED"
    exit 0
fi
