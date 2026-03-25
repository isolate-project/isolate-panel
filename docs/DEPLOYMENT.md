# Isolate Panel - Deployment Guide

Complete guide for deploying Isolate Panel to production using Docker.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Installation Script](#installation-script)
4. [Configuration](#configuration)
5. [Production Deployment](#production-deployment)
6. [Security Hardening](#security-hardening)
7. [Monitoring & Maintenance](#monitoring--maintenance)
8. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### System Requirements

- **OS**: Linux (Ubuntu 20.04+, Debian 11+, CentOS 8+)
- **CPU**: 2+ cores (4+ recommended)
- **RAM**: 512MB minimum (1GB+ recommended)
- **Storage**: 10GB+ free space
- **Docker**: 20.10+ with Docker Compose v2.0+

### Network Requirements

- Public IP address (for proxy connections)
- Firewall configured to allow proxy ports
- SSH access for management

---

## Quick Start

### Option 1: Automated Installation (Recommended)

```bash
# Download installation script
curl -fsSL https://raw.githubusercontent.com/your-org/isolate-panel/main/docker/install.sh -o install.sh

# Make executable and run
chmod +x install.sh
sudo ./install.sh
```

The script will:
- Install Docker and Docker Compose if not present
- Create installation directory (`/opt/isolate-panel`)
- Generate secure JWT secret and admin password
- Start Isolate Panel container
- Display access credentials

### Option 2: Manual Installation

#### 1. Install Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# CentOS/RHEL
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install docker-ce docker-ce-cli containerd.io
sudo systemctl start docker
sudo usermod -aG docker $USER
```

#### 2. Clone and configure

```bash
# Clone repository
git clone https://github.com/your-org/isolate-panel.git
cd isolate-panel/docker

# Copy environment template
cp .env.example .env

# Generate JWT secret
JWT_SECRET=$(openssl rand -base64 64)
sed -i "s/change-this-in-production.*/$JWT_SECRET/" .env
```

#### 3. Start the Panel

```bash
# Build and start
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f
```

### 2. Clone and Configure

```bash
# Clone repository
git clone https://github.com/your-org/isolate-panel.git
cd isolate-panel/docker

# Copy environment template
cp .env.example .env

# Generate JWT secret
JWT_SECRET=$(openssl rand -base64 64)
sed -i "s/change-this-in-production.*/$JWT_SECRET/" .env
```

### 3. Start the Panel

```bash
# Build and start
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f
```

### 4. Access the Panel

```bash
# Via SSH tunnel (recommended)
ssh -L 8080:localhost:8080 user@your-server

# Then open in browser
http://localhost:8080
```

**Default credentials:**
- Username: `admin`
- Password: Set during first login

---

## Installation Script

The automated installation script (`install.sh`) simplifies deployment to production servers.

### Features

- ✅ Automatic Docker and Docker Compose installation
- ✅ OS detection (Ubuntu, Debian, CentOS, RHEL, Fedora)
- ✅ Secure JWT secret generation
- ✅ Random admin password generation
- ✅ Directory structure creation
- ✅ Service startup and health check

### Usage

```bash
# Download script
curl -fsSL https://raw.githubusercontent.com/your-org/isolate-panel/main/docker/install.sh -o install.sh

# Make executable
chmod +x install.sh

# Run installation
sudo ./install.sh
```

### What the Script Does

1. **Pre-installation checks:**
   - Verifies root privileges
   - Checks if Docker is installed
   - Checks if Docker Compose is installed
   - Installs missing dependencies

2. **Installation:**
   - Creates `/opt/isolate-panel` directory
   - Downloads `docker-compose.yml` and `.env.example`
   - Generates `.env` with secure secrets
   - Sets file permissions (600 for .env)

3. **Post-installation:**
   - Starts Docker containers
   - Displays access credentials
   - Shows management commands

### Generated Credentials

The script generates and displays:
- **JWT Secret**: 64-character base64 random string
- **Admin Password**: 16-character alphanumeric random string

**Important:** Save these credentials immediately! The password is shown only once during installation.

### Manual Installation Directory

If you prefer manual setup, the script uses:
- **Install directory:** `/opt/isolate-panel`
- **Data directory:** `/opt/isolate-panel/data`
- **Logs directory:** `/opt/isolate-panel/logs`
- **Environment file:** `/opt/isolate-panel/.env` (permissions: 600)

---

## Configuration

### Environment Variables

Edit `.env` file to configure:

```bash
# Required
JWT_SECRET=your-secret-key-here

# Optional (with defaults)
APP_ENV=production
PORT=8080
LOG_LEVEL=info
MONITORING_MODE=lite
TZ=UTC
DATABASE_PATH=/app/data/isolate-panel.db
```

### Environment Variable Descriptions

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | **Yes** | - | Secret key for JWT tokens (generate with `openssl rand -base64 64`) |
| `APP_ENV` | No | `production` | Environment: `development` or `production` |
| `PORT` | No | `8080` | Panel API port |
| `LOG_LEVEL` | No | `info` | Logging level: `debug`, `info`, `warn`, `error` |
| `MONITORING_MODE` | No | `lite` | Traffic collection: `lite` (60s) or `full` (10s) |
| `TZ` | No | `UTC` | Timezone (e.g., `UTC`, `Europe/Moscow`, `Asia/Shanghai`) |
| `DATABASE_PATH` | No | `/app/data/isolate-panel.db` | SQLite database path inside container |

### Monitoring Modes

| Mode | Interval | RAM Usage | Use Case |
|------|----------|-----------|----------|
| `lite` | 60 seconds | ~30MB | Low-resource servers, basic monitoring |
| `full` | 10 seconds | ~100MB | Production servers, real-time stats |

---

## Production Deployment

### 1. Configure Proxy Ports

Edit `docker-compose.yml` to expose proxy ports:

```yaml
ports:
  - "127.0.0.1:8080:8080"  # Panel (localhost only)
  
  # Uncomment ports you need:
  - "443:443/tcp"          # HTTPS proxy
  - "443:443/udp"          # QUIC/H2
  - "8443:8443/tcp"        # Alternative HTTPS
```

**Important:** Only expose ports you actually use for proxies.

### 2. Configure Firewall

```bash
# UFW (Ubuntu)
sudo ufw allow 22/tcp      # SSH
sudo ufw allow 443/tcp     # HTTPS proxy
sudo ufw allow 443/udp     # QUIC
sudo ufw enable

# firewalld (CentOS)
sudo firewall-cmd --permanent --add-service=ssh
sudo firewall-cmd --permanent --add-port=443/tcp
sudo firewall-cmd --permanent --add-port=443/udp
sudo firewall-cmd --reload
```

### 3. Set Up Auto-Start

```bash
# Enable Docker service
sudo systemctl enable docker
sudo systemctl restart docker

# Container auto-restart is configured in docker-compose.yml
# restart: unless-stopped
```

### 4. Backup Configuration

```bash
# Backup data directory
tar -czf isolate-panel-backup-$(date +%Y%m%d).tar.gz ./data

# Backup to remote server
scp isolate-panel-backup-*.tar.gz user@backup-server:/backups/
```

---

## Security Hardening

### 1. Non-Root User

The container runs as non-root user (`isolate:isolate`, UID:1000) by default.

### 2. Network Isolation

Panel is only accessible via localhost by default. Use SSH tunnel:

```bash
ssh -L 8080:localhost:8080 -N user@your-server
```

### 3. Capability Dropping

Container drops all capabilities except `NET_BIND_SERVICE`:

```yaml
security_opt:
  - no-new-privileges:true
cap_drop:
  - ALL
cap_add:
  - NET_BIND_SERVICE
```

### 4. Read-Only Filesystem (Optional)

For enhanced security, enable read-only root filesystem:

```yaml
read_only: true
tmpfs:
  - /tmp
  - /var/run
```

### 5. Update JWT Secret

**Never use the default JWT secret in production!**

```bash
# Generate strong secret
openssl rand -base64 64

# Update .env file
JWT_SECRET="your-generated-secret"

# Restart container
docker compose restart
```

---

## Monitoring & Maintenance

### Health Check

Container includes built-in health check:

```bash
# Check container health
docker inspect --format='{{.State.Health.Status}}' isolate-panel

# Or via API
curl http://localhost:8080/health
```

**Health response:**
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime": "2h30m15s",
  "database": "connected",
  "timestamp": "2026-03-25T12:00:00Z"
}
```

### Logs

```bash
# View all logs
docker compose logs

# Follow logs in real-time
docker compose logs -f

# Last 100 lines
docker compose logs --tail=100

# Supervisor logs (inside container)
docker exec isolate-panel tail -f /var/log/supervisor/isolate-panel.log
```

### Resource Monitoring

```bash
# Container stats
docker stats isolate-panel

# Memory usage
docker exec isolate-panel free -h

# Disk usage
docker exec isolate-panel df -h /app/data
```

### Updates

```bash
# Pull latest image
docker compose pull

# Recreate container
docker compose up -d --force-recreate

# Remove old images
docker image prune -f
```

---

## Troubleshooting

### Container Won't Start

**Check logs:**
```bash
docker compose logs
```

**Common issues:**

1. **JWT_SECRET not set:**
   ```
   ERROR: JWT_SECRET is not set or uses default value!
   ```
   **Solution:** Generate and set JWT_SECRET in `.env`

2. **Port already in use:**
   ```
   Error starting userland proxy: listen tcp4 0.0.0.0:8080: bind: address already in use
   ```
   **Solution:** Change PORT in `.env` or stop conflicting service

3. **Permission denied:**
   ```
   permission denied while trying to connect to Docker daemon socket
   ```
   **Solution:** Add user to docker group: `sudo usermod -aG docker $USER`

### Database Issues

**Reset database (WARNING: deletes all data):**
```bash
docker compose down
rm ./data/isolate-panel.db
docker compose up -d
```

### Core Issues

**Check core status:**
```bash
docker exec isolate-panel supervisorctl status
```

**Restart specific core:**
```bash
docker exec isolate-panel supervisorctl restart singbox
docker exec isolate-panel supervisorctl restart xray
docker exec isolate-panel supervisorctl restart mihomo
```

### High Memory Usage

**Switch to lite monitoring mode:**
1. Open Settings page
2. Change "Monitoring Mode" to "Lite"
3. Or set `MONITORING_MODE=lite` in `.env` and restart

---

## Architecture Overview

### Container Structure

```
isolate-panel container
├── /usr/local/bin/isolate-panel  # Backend binary
├── /var/www/html                  # Frontend (Preact)
├── /usr/local/bin/{xray,sing-box,mihomo,wgcf}  # Cores
├── /app/data                      # Persistent data
│   ├── isolate-panel.db          # SQLite database
│   ├── cores/                    # Core configurations
│   ├── backups/                  # Backups
│   └── certificates/             # TLS certificates
└── /var/log/supervisor           # Logs
```

### Process Management

Supervisord manages all processes:
- **isolate-panel** (backend + frontend) - autostart: true
- **singbox** - autostart: false (lazy loading)
- **xray** - autostart: false (lazy loading)
- **mihomo** - autostart: false (lazy loading)

Cores are started on-demand when inbounds are created.

### Network Flow

```
Internet
    │
    ├── Port 443 ──► Proxy Core (Xray/Sing-box/Mihomo)
    │
    └── Port 8080 ──► Isolate Panel (localhost only)
                          │
                          └── SSH Tunnel ──► Admin Browser
```

---

## Support

For issues and feature requests, please open an issue on GitHub.

**Documentation version:** 0.1.0  
**Last updated:** March 2026
