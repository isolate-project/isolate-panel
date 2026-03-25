#!/bin/bash
# =============================================================================
# Isolate Panel Installation Script
# =============================================================================
# Automated installation script for Docker-based deployment
# Supports Ubuntu, Debian, CentOS, and other major Linux distributions
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Installation directory
INSTALL_DIR="/opt/isolate-panel"

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  Isolate Panel Installation${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
}

print_step() {
    echo ""
    echo -e "${GREEN}>>> $1${NC}"
    echo ""
}

# =============================================================================
# Pre-installation Checks
# =============================================================================

check_root() {
    print_step "Checking privileges"
    
    if [ "$EUID" -ne 0 ]; then 
        log_error "Please run as root (use sudo)"
        exit 1
    fi
    
    log_success "Running as root"
}

check_docker() {
    print_step "Checking Docker"
    
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version | cut -d' ' -f3)
        log_success "Docker is already installed (version $DOCKER_VERSION)"
    else
        log_warning "Docker is not installed"
        install_docker
    fi
    
    # Check if Docker service is running
    if ! systemctl is-active --quiet docker; then
        log_warning "Docker service is not running. Starting..."
        systemctl start docker
    fi
    
    log_success "Docker service is running"
}

install_docker() {
    log_info "Installing Docker..."
    
    # Detect OS
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
    else
        log_error "Cannot detect OS"
        exit 1
    fi
    
    case $OS in
        ubuntu|debian)
            log_info "Installing Docker for $OS..."
            apt-get update
            apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release
            curl -fsSL https://download.docker.com/linux/$OS/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
            echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/$OS $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
            apt-get update
            apt-get install -y docker-ce docker-ce-cli containerd.io
            ;;
        centos|rhel|fedora)
            log_info "Installing Docker for $OS..."
            yum install -y yum-utils
            yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            yum install -y docker-ce docker-ce-cli containerd.io
            systemctl enable docker
            ;;
        *)
            log_warning "Unknown OS ($OS). Trying official Docker install script..."
            curl -fsSL https://get.docker.com | sh
            ;;
    esac
    
    systemctl enable docker
    systemctl start docker
    
    log_success "Docker installed successfully"
}

check_docker_compose() {
    print_step "Checking Docker Compose"
    
    if command -v docker compose &> /dev/null; then
        COMPOSE_VERSION=$(docker compose version | cut -d' ' -f4)
        log_success "Docker Compose is already installed (version $COMPOSE_VERSION)"
    elif command -v docker-compose &> /dev/null; then
        COMPOSE_VERSION=$(docker-compose --version | cut -d' ' -f3)
        log_success "Docker Compose is already installed (version $COMPOSE_VERSION)"
        # Create alias for newer syntax
        alias docker-compose='docker compose'
    else
        log_warning "Docker Compose is not installed"
        install_docker_compose
    fi
}

install_docker_compose() {
    log_info "Installing Docker Compose..."
    
    COMPOSE_VERSION="v2.24.0"
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64)
            ARCH="x86_64"
            ;;
        aarch64|arm64)
            ARCH="aarch64"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    curl -L "https://github.com/docker/compose/releases/download/$COMPOSE_VERSION/docker-compose-$(uname -s)-$ARCH" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
    
    # Also install as docker compose plugin
    mkdir -p /usr/libexec/docker/cli-plugins
    cp /usr/local/bin/docker-compose /usr/libexec/docker/cli-plugins/docker-compose
    
    log_success "Docker Compose installed successfully"
}

# =============================================================================
# Installation
# =============================================================================

create_directories() {
    print_step "Creating directories"
    
    mkdir -p "$INSTALL_DIR"/{data,logs}
    cd "$INSTALL_DIR"
    
    log_success "Directories created at $INSTALL_DIR"
}

download_files() {
    print_step "Downloading configuration files"
    
    # Check if we're running from a cloned repository
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    if [ -f "$SCRIPT_DIR/docker-compose.yml" ]; then
        log_info "Copying from local repository..."
        cp "$SCRIPT_DIR/docker-compose.yml" "$INSTALL_DIR/"
        cp "$SCRIPT_DIR/.env.example" "$INSTALL_DIR/.env.example"
    else
        log_info "Downloading from GitHub..."
        curl -sL "https://raw.githubusercontent.com/your-org/isolate-panel/main/docker/docker-compose.yml" -o "$INSTALL_DIR/docker-compose.yml"
        curl -sL "https://raw.githubusercontent.com/your-org/isolate-panel/main/docker/.env.example" -o "$INSTALL_DIR/.env.example"
    fi
    
    log_success "Configuration files downloaded"
}

generate_env() {
    print_step "Generating environment configuration"
    
    # Generate JWT secret
    JWT_SECRET=$(openssl rand -base64 64)
    
    # Generate admin password
    ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9' | head -c 16)
    
    # Create .env file
    cat > "$INSTALL_DIR/.env" << EOL
# Isolate Panel Environment Configuration
# Generated on $(date)

# JWT Authentication (REQUIRED)
JWT_SECRET=${JWT_SECRET}

# Application Settings
APP_ENV=production
PORT=8080
TZ=UTC

# Logging
LOG_LEVEL=info

# Monitoring Mode: lite (60s) or full (10s)
MONITORING_MODE=lite

# Database
DATABASE_PATH=/app/data/isolate-panel.db

# Admin Credentials (CHANGE AFTER FIRST LOGIN!)
ADMIN_USERNAME=admin
ADMIN_PASSWORD=${ADMIN_PASSWORD}
EOL

    # Set secure permissions
    chmod 600 "$INSTALL_DIR/.env"
    
    log_success "Environment configuration generated"
    
    # Display credentials
    echo ""
    log_warning "IMPORTANT: Save these credentials securely!"
    echo ""
    echo -e "${GREEN}Default Admin Credentials:${NC}"
    echo "  Username: ${BLUE}admin${NC}"
    echo "  Password: ${BLUE}${ADMIN_PASSWORD}${NC}"
    echo ""
    log_warning "Change the password after first login!"
}

# =============================================================================
# Start Services
# =============================================================================

start_services() {
    print_step "Starting Isolate Panel"
    
    cd "$INSTALL_DIR"
    
    # Pull and start containers
    if command -v docker-compose &> /dev/null; then
        docker-compose up -d
    else
        docker compose up -d
    fi
    
    # Wait for container to be healthy
    log_info "Waiting for container to start..."
    sleep 5
    
    # Check status
    if command -v docker-compose &> /dev/null; then
        docker-compose ps
    else
        docker compose ps
    fi
    
    log_success "Isolate Panel started successfully"
}

# =============================================================================
# Post-installation
# =============================================================================

print_summary() {
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Installation Complete!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "${BLUE}Panel URL:${NC} http://localhost:8080"
    echo ""
    echo -e "${BLUE}Access via SSH tunnel:${NC}"
    echo "  ssh -L 8080:localhost:8080 user@your-server"
    echo ""
    echo -e "${BLUE}Management commands:${NC}"
    echo "  cd $INSTALL_DIR"
    if command -v docker-compose &> /dev/null; then
        echo "  docker-compose ps          # Check status"
        echo "  docker-compose logs -f     # View logs"
        echo "  docker-compose stop        # Stop panel"
        echo "  docker-compose start       # Start panel"
        echo "  docker-compose restart     # Restart panel"
    else
        echo "  docker compose ps          # Check status"
        echo "  docker compose logs -f     # View logs"
        echo "  docker compose stop        # Stop panel"
        echo "  docker compose start       # Start panel"
        echo "  docker compose restart     # Restart panel"
    fi
    echo ""
    echo -e "${BLUE}Configuration files:${NC}"
    echo "  $INSTALL_DIR/.env              # Environment variables"
    echo "  $INSTALL_DIR/docker-compose.yml # Docker configuration"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo "  1. Set up SSH tunnel to access the panel"
    echo "  2. Login with admin credentials (shown above)"
    echo "  3. Change the default password immediately"
    echo "  4. Configure proxy ports in docker-compose.yml if needed"
    echo ""
    echo -e "${GREEN}========================================${NC}"
}

# =============================================================================
# Main Installation Flow
# =============================================================================

main() {
    print_header
    
    # Pre-installation checks
    check_root
    check_docker
    check_docker_compose
    
    # Installation
    create_directories
    download_files
    generate_env
    start_services
    
    # Post-installation
    print_summary
}

# Run installation
main "$@"
