# Production Deployment Guide

Complete guide for deploying Isolate Panel on a fresh Ubuntu 24.04 VPS with Docker Compose, Caddy reverse proxy, automated backups, and security hardening.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Initial Server Setup](#initial-server-setup)
3. [Docker Installation](#docker-installation)
4. [User Namespace Remapping](#user-namespace-remapping)
5. [Firewall Configuration](#firewall-configuration)
6. [Isolate Panel Installation](#isolate-panel-installation)
7. [SSH Tunnel Access](#ssh-tunnel-access)
8. [Backup and Recovery](#backup-and-recovery)
9. [Monitoring and Maintenance](#monitoring-and-maintenance)
10. [WireGuard Panel-to-Node Tunnel](#wireguard-panel-to-node-tunnel-optional)
11. [Troubleshooting](#troubleshooting)

---

## Prerequisites

- Fresh Ubuntu 24.04 LTS VPS (1GB+ RAM recommended)
- Root or sudo access
- Domain name (optional, for Caddy auto HTTPS)
- SSH key-based authentication configured

---

## Initial Server Setup

### 1. Update System

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y curl wget git ufw fail2ban
```

### 2. Create Dedicated User

```bash
# Create a non-root user for running Docker
sudo useradd -m -s /bin/bash isolate
sudo usermod -aG sudo isolate

# Set up SSH key for the new user
sudo mkdir -p /home/isolate/.ssh
sudo cp ~/.ssh/authorized_keys /home/isolate/.ssh/
sudo chown -R isolate:isolate /home/isolate/.ssh
sudo chmod 700 /home/isolate/.ssh
sudo chmod 600 /home/isolate/.ssh/authorized_keys
```

### 3. Harden SSH Configuration

```bash
sudo nano /etc/ssh/sshd_config
```

Make these changes:
```
PermitRootLogin no
PasswordAuthentication no
MaxAuthTries 3
ClientAliveInterval 300
ClientAliveCountMax 2
```

Restart SSH:
```bash
sudo systemctl restart sshd
```

---

## Docker Installation

### 1. Install Docker Engine

```bash
# Remove old versions
sudo apt remove -y docker docker-engine docker.io containerd runc

# Install prerequisites
sudo apt install -y ca-certificates gnupg lsb-release

# Add Docker's official GPG key
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# Set up repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Enable Docker service
sudo systemctl enable docker
sudo systemctl start docker
```

### 2. Configure Docker for Non-Root User

```bash
sudo usermod -aG docker isolate
newgrp docker
```

---

## User Namespace Remapping

User namespace remapping adds an extra layer of security by mapping container root to a non-root user on the host.

### 1. Enable User Namespace Remapping

```bash
# Create dockremap user
sudo useradd -u 100000 -U dockremap

# Configure subordinate UIDs/GIDs
sudo sh -c 'echo "dockremap:100000:65536" > /etc/subuid'
sudo sh -c 'echo "dockremap:100000:65536" > /etc/subgid'

# Create Docker daemon configuration
sudo mkdir -p /etc/docker
sudo nano /etc/docker/daemon.json
```

Add this configuration:
```json
{
  "userns-remap": "default",
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "3"
  },
  "live-restore": true,
  "no-new-privileges": true,
  "seccomp-profile": "/etc/docker/seccomp-default.json"
}
```

### 2. Create Seccomp Profile

```bash
sudo curl -fsSL https://raw.githubusercontent.com/moby/moby/master/profiles/seccomp/default.json \
  -o /etc/docker/seccomp-default.json
```

### 3. Restart Docker

```bash
sudo systemctl restart docker
```

---

## Firewall Configuration

### 1. Configure UFW

```bash
# Reset UFW to defaults
sudo ufw --force reset

# Set default policies
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH (adjust port if needed)
sudo ufw allow 22/tcp

# Allow HTTP and HTTPS (for Caddy)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 443/udp  # HTTP/3 (QUIC)

# Allow VPN inbound ports (adjust range as needed)
sudo ufw allow 2000:2050/tcp
sudo ufw allow 2000:2050/udp

# Enable UFW
sudo ufw enable
```

### 2. Configure DOCKER-USER Chain

Docker bypasses UFW by default. We need to configure the DOCKER-USER chain:

```bash
# Create a script to configure DOCKER-USER chain
sudo nano /usr/local/bin/docker-user-rules.sh
```

Add this content:
```bash
#!/bin/bash
# Configure DOCKER-USER chain for additional security

# Flush existing rules
iptables -F DOCKER-USER

# Allow established connections
iptables -A DOCKER-USER -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

# Allow loopback
iptables -A DOCKER-USER -i lo -j ACCEPT

# Block external access to Docker internal networks
iptables -A DOCKER-USER -s 172.16.0.0/12 -j DROP

# Allow specific external access (adjust as needed)
# Example: Allow access to port 8080 only from localhost
iptables -A DOCKER-USER -p tcp --dport 8080 -s 127.0.0.1 -j ACCEPT
iptables -A DOCKER-USER -p tcp --dport 8080 -j DROP

# Default: return to calling chain
iptables -A DOCKER-USER -j RETURN
```

Make it executable and run:
```bash
sudo chmod +x /usr/local/bin/docker-user-rules.sh
sudo /usr/local/bin/docker-user-rules.sh
```

Create a systemd service to persist rules:
```bash
sudo nano /etc/systemd/system/docker-user-rules.service
```

Add:
```ini
[Unit]
Description=Apply DOCKER-USER iptables rules
After=docker.service
Wants=docker.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/docker-user-rules.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
```

Enable the service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable docker-user-rules
sudo systemctl start docker-user-rules
```

---

## Isolate Panel Installation

### 1. Create Installation Directory

```bash
sudo mkdir -p /opt/isolate-panel
sudo chown isolate:isolate /opt/isolate-panel
su - isolate
cd /opt/isolate-panel
```

### 2. Download Configuration Files

```bash
# Download docker-compose files
curl -fsSL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/docker-compose.yml -o docker-compose.yml
curl -fsSL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/docker-compose.prod.yml -o docker-compose.prod.yml
curl -fsSL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/.env.example -o .env

# Create necessary directories
mkdir -p data logs backups secrets caddy

# Download Caddy configuration
curl -fsSL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/caddy/Caddyfile -o caddy/Caddyfile
curl -fsSL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/caddy/Dockerfile -o caddy/Dockerfile
```

### 3. Configure Environment (Optional)

```bash
nano .env
```

All secrets (JWT, password pepper, data encryption key, admin password) are **auto-generated on first run** and persisted in `/app/data/.core-secrets` inside the container volume. You only need `.env` for non-secret settings:

```
# Application
APP_ENV=production
TZ=UTC

# Admin credentials (password auto-generated if left empty)
ADMIN_USERNAME=admin

# Backup
BACKUP_PASSWORD=$(openssl rand -base64 32)

# Monitoring
MONITORING_MODE=lite
```

If you want to set a custom admin password instead of auto-generated:
```
ADMIN_PASSWORD=your-secure-password-here
```

### 5. Start Services

```bash
# Start with production configuration
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Or set COMPOSE_FILE environment variable
export COMPOSE_FILE=docker-compose.yml:docker-compose.prod.yml
docker compose up -d
```

### 6. Verify Installation

```bash
# Check container status
docker compose ps

# Check logs
docker compose logs -f

# Test health endpoints
curl http://localhost:8080/healthz
curl http://localhost:2019/config/  # Caddy admin API
```

---

## SSH Tunnel Access

The admin panel is only accessible via SSH tunnel (binds to 127.0.0.1:8080).

### From Your Local Machine

```bash
# Basic tunnel
ssh -L 8080:localhost:8080 isolate@your-server-ip

# Background tunnel (detached)
ssh -fNL 8080:localhost:8080 isolate@your-server-ip

# Tunnel with specific SSH key
ssh -i ~/.ssh/your-key -L 8080:localhost:8080 isolate@your-server-ip

# Via jump host (bastion)
ssh -J bastion-user@bastion-host -L 8080:localhost:8080 isolate@your-server-ip
```

### Access the Panel

Once the tunnel is established:
1. Open browser
2. Navigate to `http://localhost:8080`
3. Login with credentials from `.env`

### Persistent Tunnel with AutoSSH

For a persistent tunnel that reconnects automatically:

```bash
# Install autossh
sudo apt install autossh

# Create systemd service
sudo nano /etc/systemd/system/isolate-tunnel.service
```

Add:
```ini
[Unit]
Description=Isolate Panel SSH Tunnel
After=network.target

[Service]
User=your-local-user
ExecStart=/usr/bin/autossh -M 0 -N -o "ServerAliveInterval 60" -o "ServerAliveCountMax 3" -o "ExitOnForwardFailure=yes" -L 8080:localhost:8080 isolate@your-server-ip
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable:
```bash
sudo systemctl daemon-reload
sudo systemctl enable isolate-tunnel
sudo systemctl start isolate-tunnel
```

---

## Backup and Recovery

### Automated Backups

Backups run daily at 2 AM via the `backup` service:
- SQLite database
- Configuration files
- Logs
- Encrypted with password from secrets

### Manual Backup

```bash
# Trigger immediate backup
docker compose exec backup backup

# Check backup status
docker compose logs backup
```

### Restore from Backup

```bash
# List available backups
ls -la backups/

# Stop services
docker compose down

# Extract backup
cd /opt/isolate-panel
tar -xzf backups/isolate-panel-backup-YYYY-MM-DDTHH-MM-SS.tar.gz

# Restore data
cp -r backup/data/* data/
cp -r backup/logs/* logs/

# Restart services
docker compose up -d
```

### Backup Retention

Backups are automatically cleaned up after 7 days (configurable via `BACKUP_RETENTION_DAYS`).

---

## Monitoring and Maintenance

### Image Update Notifications

Diun checks for Docker image updates daily at 6 AM. Configure notifications in `.env`:

```bash
# Discord
DIUN_DISCORD_WEBHOOK=https://discord.com/api/webhooks/...

# Slack
DIUN_SLACK_WEBHOOK=https://hooks.slack.com/services/...

# Telegram
DIUN_TELEGRAM_BOT_TOKEN=your-bot-token
DIUN_TELEGRAM_CHAT_ID=your-chat-id
```

### View Diun Logs

```bash
docker compose logs diun
```

### Update Isolate Panel

```bash
cd /opt/isolate-panel

# Pull latest images
docker compose pull

# Restart with new images
docker compose up -d

# Verify update
docker compose ps
```

### Monitor Resource Usage

```bash
# Container stats
docker stats

# Disk usage
docker system df

# Clean up unused resources
docker system prune -a
```

### Log Rotation

Logs are automatically rotated by Docker:
- Max size: 100MB per file
- Max files: 3
- JSON format with labels

View logs:
```bash
# All services
docker compose logs

# Specific service
docker compose logs isolate-panel

# Follow logs
docker compose logs -f

# Last 100 lines
docker compose logs --tail 100
```

---

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker compose logs isolate-panel

# Check for port conflicts
sudo netstat -tlnp | grep -E '8080|443|80'

# Check auto-generated secrets
docker compose exec isolate-panel cat /app/data/.core-secrets
```

### Health Check Failures

```bash
# Test health endpoint manually
docker compose exec isolate-panel wget -qO- http://localhost:8080/healthz

# Check Caddy health
curl http://localhost:2019/config/

# Restart specific service
docker compose restart isolate-panel
```

### Database Issues

```bash
# Check database file
ls -la data/isolate-panel.db

# Verify permissions
docker compose exec isolate-panel ls -la /app/data/

# Run migrations manually
docker compose exec isolate-panel isolate-migrate -db /app/data/isolate-panel.db -cmd up
```

### Backup Failures

```bash
# Check backup logs
docker compose logs backup

# Verify backup password
cat secrets/backup_password.txt

# Test backup manually
docker compose exec backup backup
```

### Caddy Certificate Issues

```bash
# Check Caddy logs
docker compose logs caddy

# Verify Caddy data volume
docker compose exec caddy ls -la /data/

# Force certificate renewal
docker compose exec caddy caddy reload --config /etc/caddy/Caddyfile
```

### SSH Tunnel Issues

```bash
# Test SSH connection
ssh -v isolate@your-server-ip

# Check if port 8080 is listening on server
sudo netstat -tlnp | grep 8080

# Verify panel is running
docker compose ps
```

### Firewall Issues

```bash
# Check UFW status
sudo ufw status verbose

# Check DOCKER-USER chain
sudo iptables -L DOCKER-USER -v -n

# Temporarily disable UFW for testing
sudo ufw disable
# ... test ...
sudo ufw enable
```

---

## WireGuard Panel-to-Node Tunnel (Optional)

Use WireGuard to encrypt all API communication between the panel and remote proxy cores over the internet. This is **recommended** if you have nodes on different servers.

### 1. Generate WireGuard Keys

```bash
cd /opt/isolate-panel
# Run the setup script (generates keys for panel + 3 nodes)
./wireguard/setup.sh
```

### 2. Configure the Panel (Server)

Edit `wireguard/config/wg0-panel.conf`:
1. Replace `<NODE1_PUBLIC_KEY>`, etc. with the actual public keys from the script output
2. Set your VPS public IP or domain as the endpoint

Copy the config and start WireGuard:

```bash
sudo cp wireguard/config/wg0-panel.conf /etc/wireguard/wg0.conf
sudo wg-quick up wg0
sudo systemctl enable wg-quick@wg0
```

### 3. Configure Remote Nodes

On each remote node server:

```bash
# Install WireGuard
sudo apt install -y wireguard

# Copy node config from panel server
scp isolate@panel-server:/opt/isolate-panel/wireguard/config/wg0-node1.conf /etc/wireguard/wg0.conf

# Start WireGuard
sudo wg-quick up wg0
sudo systemctl enable wg-quick@wg0
```

### 4. Update Core Configuration

In the panel, cores now communicate via WireGuard internal IPs:
- Panel API: `10.200.200.1:8080`
- Node 1: `10.200.200.2`
- Node 2: `10.200.200.3`
- Node 3: `10.200.200.4`

Set the core API addresses in your `.env`:

```bash
# Example for a remote node using WireGuard
SINGBOX_API_URL=http://10.200.200.2:9090
MIHOMO_API_URL=http://10.200.200.3:9090
```

### 5. Enable WireGuard in Docker Compose

```bash
# Add WireGuard to your compose stack
docker compose -f docker-compose.yml -f docker-compose.prod.yml \
               -f docker-compose.wireguard.yml up -d
```

### Security Notes

- WireGuard uses **Curve25519** for key exchange — modern, fast, secure
- All panel-to-node traffic is encrypted end-to-end
- UDP port 51820 must be open in UFW:
  ```bash
  sudo ufw allow 51820/udp comment 'WireGuard VPN'
  ```
- WireGuard has **no listening ports** on the client side — stealthy
- The tunnel is stateless — no need to manage connections

---

## Security Checklist

- [ ] SSH key-based authentication only (no passwords)
- [ ] Root login disabled
- [ ] UFW firewall enabled with minimal ports open
- [ ] DOCKER-USER chain configured
- [ ] Docker user namespace remapping enabled
- [ ] Auto-generated secrets persisted in data volume
- [ ] Admin password saved from first-run logs (or set explicitly in .env)
- [ ] Backup encryption password generated
- [ ] Auto-updates configured (Diun)
- [ ] Fail2ban installed and configured
- [ ] Unattended security updates enabled
- [ ] Server timezone configured correctly
- [ ] WireGuard tunnel configured (if multi-node)
- [ ] Database field encryption enabled
- [ ] Per-node API keys generated and distributed

---

## Additional Resources

- [Docker Security Best Practices](https://docs.docker.com/engine/security/)
- [Caddy Documentation](https://caddyserver.com/docs/)
- [UFW Documentation](https://help.ubuntu.com/community/UFW)
- [Isolate Panel Wiki](https://github.com/isolate-project/isolate-panel/wiki)
