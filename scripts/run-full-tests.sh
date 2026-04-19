#!/bin/bash

set -e

COMPOSE_FILE="docker/docker-compose.fullstack.yml"
SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPTS_DIR")"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SKIP_BACKEND=false
SKIP_FRONTEND=false
NO_CLEANUP=false
VERBOSE=false

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Run full-stack integration tests for isolate-panel"
    echo ""
    echo "Options:"
    echo "  --skip-backend      Skip backend integration tests"
    echo "  --skip-frontend     Skip frontend e2e tests"
    echo "  --no-cleanup        Keep Docker containers running after tests"
    echo "  --verbose           Enable verbose output"
    echo "  --help              Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  ISOLATE_FULLSTACK_TESTS=1  Required to run full-stack tests"
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-backend)
            SKIP_BACKEND=true
            shift
            ;;
        --skip-frontend)
            SKIP_FRONTEND=true
            shift
            ;;
        --no-cleanup)
            NO_CLEANUP=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

log() {
    echo -e "${GREEN}[FULLSTACK]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[FULLSTACK]${NC} $1"
}

error() {
    echo -e "${RED}[FULLSTACK]${NC} $1"
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        error "Docker Compose is not installed"
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        error "Go is not installed"
        exit 1
    fi
    
    if ! command -v npm &> /dev/null; then
        error "npm is not installed"
        exit 1
    fi
    
    log "All prerequisites satisfied"
}

cleanup() {
    if [ "$NO_CLEANUP" = true ]; then
        warn "Skipping cleanup (--no-cleanup specified)"
        warn "Containers are still running. To clean up manually:"
        warn "  docker compose -f $COMPOSE_FILE --profile cores down -v"
        return
    fi
    
    log "Cleaning up..."
    cd "$PROJECT_DIR"
    docker compose -f "$COMPOSE_FILE" --profile cores down -v --remove-orphans 2>/dev/null || true
    rm -f frontend/e2e-full/.auth-tokens.json
}

run_backend_tests() {
    if [ "$SKIP_BACKEND" = true ]; then
        warn "Skipping backend tests (--skip-backend specified)"
        return 0
    fi
    
    log "====================================="
    log "Running Backend Full-Stack Tests"
    log "====================================="
    
    cd "$PROJECT_DIR/backend"
    
    export ISOLATE_FULLSTACK_TESTS=1
    
    if [ "$VERBOSE" = true ]; then
        go test -v ./tests/integration/... -run TestFullStack -timeout 30m
    else
        go test ./tests/integration/... -run TestFullStack -timeout 30m
    fi
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        log "Backend tests PASSED"
    else
        error "Backend tests FAILED with exit code $exit_code"
        return $exit_code
    fi
}

run_frontend_tests() {
    if [ "$SKIP_FRONTEND" = true ]; then
        warn "Skipping frontend tests (--skip-frontend specified)"
        return 0
    fi
    
    log "====================================="
    log "Running Frontend Full-Stack Tests"
    log "====================================="
    
    cd "$PROJECT_DIR/frontend"
    
    if ! command -v npx &> /dev/null; then
        log "Installing Playwright browsers..."
        npx playwright install chromium
    fi
    
    if [ "$VERBOSE" = true ]; then
        npm run test:e2e:full
    else
        npm run test:e2e:full -- --reporter=line
    fi
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        log "Frontend tests PASSED"
    else
        error "Frontend tests FAILED with exit code $exit_code"
        return $exit_code
    fi
}

print_summary() {
    log "====================================="
    log "Full-Stack Test Summary"
    log "====================================="
    
    if [ "$SKIP_BACKEND" = false ]; then
        log "Backend Tests: PASSED"
    else
        warn "Backend Tests: SKIPPED"
    fi
    
    if [ "$SKIP_FRONTEND" = false ]; then
        log "Frontend Tests: PASSED"
    else
        warn "Frontend Tests: SKIPPED"
    fi
    
    log ""
    log "All requested tests completed successfully!"
}

main() {
    log "=========================================="
    log "Isolate-Panel Full-Stack Test Runner"
    log "=========================================="
    
    check_prerequisites
    
    if [ "$NO_CLEANUP" = false ]; then
        trap cleanup EXIT
    fi
    
    local backend_exit=0
    local frontend_exit=0
    
    if [ "$SKIP_BACKEND" = false ]; then
        run_backend_tests
        backend_exit=$?
    fi
    
    if [ "$SKIP_FRONTEND" = false ] && [ $backend_exit -eq 0 ]; then
        run_frontend_tests
        frontend_exit=$?
    fi
    
    if [ $backend_exit -eq 0 ] && [ $frontend_exit -eq 0 ]; then
        print_summary
        exit 0
    else
        error "Some tests failed. Check the output above for details."
        exit 1
    fi
}

main "$@"
