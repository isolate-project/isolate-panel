# Isolate Panel - Deployment Guide

## 🚀 Quick Start with Docker

### Prerequisites
- Docker and Docker Compose installed
- Ports 8080, 443 available

### 1. Clone and Configure
```bash
cd /path/to/isolate-panel
cp docker/.env.example docker/.env
```

Edit `docker/.env` and set:
```bash
JWT_SECRET=your-super-secret-key-here-use-long-random-string
ADMIN_USERNAME=admin
ADMIN_PASSWORD=your-secure-password
```

### 2. Start the container
```bash
docker compose -f docker/docker-compose.yml up -d --build
```

### 3. Access the panel
- **URL:** http://localhost:8080
- **Username:** admin
- **Password:** (from your .env)

## 🔐 Production Deployment (VPS)

### Security Recommendations

1. **Use SSH tunnel for panel access:**
   ```bash
   # In docker-compose.yml, change:
   ports:
     - "127.0.0.1:8080:8080"  # Only localhost
   
   # Then access via SSH tunnel:
   ssh -L 8080:localhost:8080 user@your-vps
   ```

2. **Enable HTTPS with reverse proxy:**
   - Use Nginx/Caddy as reverse proxy
   - Configure Let's Encrypt SSL

3. **Configure firewall:**
   ```bash
   # Only allow necessary ports
   ufw allow 22/tcp      # SSH
   ufw allow 443/tcp     # HTTPS
   ufw allow 443/udp     # QUIC/H2
   ufw enable
   ```

4. **Set strong JWT_SECRET:**
   ```bash
   # Generate secure random string
   openssl rand -hex 32
   ```

## 📊 Proxy Ports Configuration

Uncomment proxy ports in `docker-compose.yml` based on your needs:

```yaml
ports:
  - "443:443/tcp"      # Standard HTTPS
  - "443:443/udp"      # QUIC/H2
  - "8443:8443/tcp"    # Alternative HTTPS
  - "9000:9000/tcp"    # SOCKS5
  - "9000:9000/udp"    # SOCKS5 UDP
```

## 🔧 Troubleshooting

### View logs
```bash
docker compose logs -f isolate-panel
```

### Restart container
```bash
docker compose restart
```

### Check core status
```bash
docker compose exec isolate-panel supervisorctl status
```

### Database location
- Path: `/app/data/isolate-panel.db` (inside container)
- Mapped to: `./data/isolate-panel.db` (on host)

## 📝 Backup

Backup the `data` directory:
```bash
tar -czf isolate-panel-backup-$(date +%Y%m%d).tar.gz ./data
```

To restore:
```bash
tar -xzf isolate-panel-backup-*.tar.gz -C /path/to/isolate-panel/
```
