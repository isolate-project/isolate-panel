#!/bin/bash
# =============================================================================
# Isolate Panel Installation Script
# =============================================================================
# Automated installation script for Docker-based deployment
# Supports Ubuntu, Debian, CentOS, and other major Linux distributions
#
# Usage:
#   Install:    bash install.sh
#   Update:     bash install.sh --update
#   Uninstall:  bash install.sh --uninstall
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

# GitHub repository
GITHUB_REPO="isolate-project/isolate-panel"
GITHUB_RAW="https://raw.githubusercontent.com/${GITHUB_REPO}/master"

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
            echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/$OS $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
            apt-get update
            apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
            ;;
        centos|rhel|fedora|rocky|alma)
            log_info "Installing Docker for $OS..."
            yum install -y yum-utils
            yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
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
    
    if docker compose version &> /dev/null; then
        COMPOSE_VERSION=$(docker compose version --short)
        log_success "Docker Compose is available (version $COMPOSE_VERSION)"
    elif command -v docker-compose &> /dev/null; then
        COMPOSE_VERSION=$(docker-compose --version | cut -d' ' -f3)
        log_success "Docker Compose (standalone) is available (version $COMPOSE_VERSION)"
    else
        log_error "Docker Compose is not available. Please install Docker Compose v2."
        log_info "Try: apt-get install docker-compose-plugin"
        exit 1
    fi
}

# Helper: run docker compose (v2 plugin or standalone)
compose_cmd() {
    if docker compose version &> /dev/null; then
        docker compose "$@"
    else
        docker-compose "$@"
    fi
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
    
    if [ -f "$SCRIPT_DIR/docker-compose.production.yml" ]; then
        log_info "Copying from local repository..."
        cp "$SCRIPT_DIR/docker-compose.production.yml" "$INSTALL_DIR/docker-compose.yml"
        cp "$SCRIPT_DIR/.env.example" "$INSTALL_DIR/.env.example"
    else
        log_info "Downloading from GitHub..."
        curl -sL "${GITHUB_RAW}/docker/docker-compose.production.yml" -o "$INSTALL_DIR/docker-compose.yml"
        curl -sL "${GITHUB_RAW}/docker/.env.example" -o "$INSTALL_DIR/.env.example"
    fi
    
    log_success "Configuration files downloaded"
}

generate_env() {
    print_step "Generating environment configuration"
    
    # Generate JWT secret
    JWT_SECRET=$(openssl rand -base64 64 | tr -d '\n')
    
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
    echo -e "${GREEN}Admin Credentials:${NC}"
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
    
    # Pull image and start container
    compose_cmd pull
    compose_cmd up -d
    
    # Wait for container to be healthy
    log_info "Waiting for container to start..."
    
    local max_attempts=30
    local attempt=1
    while [ $attempt -le $max_attempts ]; do
        if [ "$(docker inspect -f '{{.State.Health.Status}}' isolate-panel 2>/dev/null)" = "healthy" ]; then
            break
        fi
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    echo ""
    
    # Check status
    compose_cmd ps
    
    if [ "$(docker inspect -f '{{.State.Health.Status}}' isolate-panel 2>/dev/null)" = "healthy" ]; then
        log_success "Isolate Panel is running and healthy"
    else
        log_warning "Container is starting but not yet healthy. Check logs: docker logs isolate-panel"
    fi
}

# =============================================================================
# Update
# =============================================================================

update_panel() {
    print_step "Updating Isolate Panel"
    
    if [ ! -d "$INSTALL_DIR" ]; then
        log_error "Isolate Panel is not installed at $INSTALL_DIR"
        exit 1
    fi
    
    cd "$INSTALL_DIR"
    
    # Download latest docker-compose
    log_info "Downloading latest configuration..."
    curl -sL "${GITHUB_RAW}/docker/docker-compose.production.yml" -o "$INSTALL_DIR/docker-compose.yml.new"
    mv "$INSTALL_DIR/docker-compose.yml.new" "$INSTALL_DIR/docker-compose.yml"
    
    # Pull and restart
    log_info "Pulling latest image..."
    compose_cmd pull
    
    log_info "Restarting..."
    compose_cmd up -d
    
    log_success "Isolate Panel updated to latest version"
    compose_cmd ps
}

# =============================================================================
# Uninstall
# =============================================================================

uninstall_panel() {
    echo ""
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}  Isolate Panel Uninstaller${NC}"
    echo -e "${RED}========================================${NC}"
    echo ""
    
    if [ ! -d "$INSTALL_DIR" ]; then
        log_error "Isolate Panel is not installed at $INSTALL_DIR"
        exit 1
    fi
    
    # Confirm
    read -p "Are you sure you want to uninstall Isolate Panel? [y/N] " confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        echo "Cancelled."
        exit 0
    fi
    
    # Ask about data
    read -p "Delete ALL data (database, configs, cores, backups)? [y/N] " delete_data
    
    # Stop containers
    log_info "Stopping containers..."
    cd "$INSTALL_DIR" 2>/dev/null && compose_cmd down --remove-orphans 2>/dev/null || true
    
    if [[ "$delete_data" == "y" || "$delete_data" == "Y" ]]; then
        log_warning "Removing all data..."
        rm -rf "$INSTALL_DIR"
        log_success "All data removed"
    else
        log_info "Removing config files (keeping data/)..."
        rm -f "$INSTALL_DIR/docker-compose.yml"
        rm -f "$INSTALL_DIR/.env"
        rm -f "$INSTALL_DIR/.env.example"
        echo -e "${GREEN}Data preserved at: $INSTALL_DIR/data/${NC}"
        echo "To fully remove: rm -rf $INSTALL_DIR"
    fi
    
    # Remove Docker image
    docker rmi ghcr.io/isolate-project/isolate-panel:latest 2>/dev/null || true
    
    echo ""
    log_success "Isolate Panel has been uninstalled."
}

# =============================================================================
# Post-installation
# =============================================================================

print_summary() {
    local SERVER_IP
    SERVER_IP=$(curl -s4 ifconfig.me 2>/dev/null || echo "your-server-ip")
    
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Installation Complete! 🎉${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "${BLUE}Access via SSH tunnel:${NC}"
    echo "  ssh -L 8080:localhost:8080 root@${SERVER_IP}"
    echo "  Then open: http://localhost:8080"
    echo ""
    echo -e "${BLUE}Management commands:${NC}"
    echo "  cd $INSTALL_DIR"
    echo "  docker compose ps            # Check status"
    echo "  docker compose logs -f       # View logs"
    echo "  docker compose restart       # Restart panel"
    echo ""
    echo -e "${BLUE}Update:${NC}"
    echo "  bash <(curl -sL ${GITHUB_RAW}/docker/install.sh) --update"
    echo ""
    echo -e "${BLUE}Uninstall:${NC}"
    echo "  bash <(curl -sL ${GITHUB_RAW}/docker/install.sh) --uninstall"
    echo ""
    echo -e "${BLUE}Configuration:${NC}"
    echo "  $INSTALL_DIR/.env               # Environment variables"
    echo "  $INSTALL_DIR/docker-compose.yml  # Docker configuration"
    echo "  $INSTALL_DIR/data/               # Persistent data"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo "  1. Connect via SSH tunnel (command above)"
    echo "  2. Login with admin credentials (shown above)"
    echo "  3. Change the default password"
    echo "  4. Start proxy cores (Cores → Start)"
    echo "  5. Create users and inbounds"
    echo ""
    echo -e "${YELLOW}⚠️  Firewall: open ports for your inbounds:${NC}"
    echo "  ufw allow 2000:2100/tcp"
    echo "  ufw allow 2000:2100/udp"
    echo "  ufw allow 443/tcp"
    echo "  ufw allow 443/udp"
    echo ""
    echo -e "${GREEN}========================================${NC}"
}

# =============================================================================
# Main
# =============================================================================

main() {
    # Handle flags
    case "${1:-}" in
        --update|-u)
            check_root
            update_panel
            exit 0
            ;;
        --uninstall|--remove)
            check_root
            uninstall_panel
            exit 0
            ;;
        --help|-h)
            echo "Isolate Panel Installer"
            echo ""
            echo "Usage:"
            echo "  bash install.sh              Install Isolate Panel"
            echo "  bash install.sh --update     Update to latest version"
            echo "  bash install.sh --uninstall  Remove Isolate Panel"
            echo "  bash install.sh --help       Show this help"
            exit 0
            ;;
    esac

    print_header
    
    # Pre-installation checks
    check_root
    check_docker
    check_docker_compose
    
    # Check if already installed
    if [ -f "$INSTALL_DIR/docker-compose.yml" ]; then
        log_warning "Isolate Panel is already installed at $INSTALL_DIR"
        read -p "Reinstall? This will NOT delete your data. [y/N] " confirm
        if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
            echo "Use --update to update to the latest version."
            exit 0
        fi
    fi
    
    # Installation
    create_directories
    download_files
    generate_env
    start_services
    
    # Post-installation
    print_summary
}

# Run
main "$@"
