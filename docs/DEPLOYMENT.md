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

## 📝 Backup & Disaster Recovery

### Built-in Backup System
The Isolate Panel includes a redundant backup system that performs:
- **Database Snapshots**: Full export of the SQLite database.
- **Configuration**: All core configs, certificates, and WARP keys.
- **Security**: AES-256-GCM streaming encryption (64KB chunks).
- **Optimization**: Gzip compression for minimal storage footprint.

### Retention Policy
By default, the panel keeps the **3 most recent backups**. You can change this in the **Backups -> Schedule** section of the UI. Older backups are automatically rotated to save disk space.

### Manual Backup (External)
Backup the entire persistent `data` directory from the host:
```bash
tar -czf isolate-panel-data-$(date +%Y%m%d).tar.gz ./data
```

### Disaster Recovery: Manual Decryption
If the panel is inaccessible and you need to manually restore from an encrypted `.enc` backup:

1. **Locate your encryption key**: Found at `./data/.backup_key` on the host.
2. **Format**: The file is encrypted in **64KB chunks** [4-byte Length][12-byte Nonce][Ciphertext + 16-byte Tag].
3. **Decryption**: Use the panel UI for easy restoration, or use the `isolate-panel` CLI tool inside the container:
   ```bash
   docker compose exec isolate-panel isolate-panel restore --file /app/data/backups/manual_xyz.enc
   ```

> [!IMPORTANT]
> Always keep a copy of `./data/.backup_key` in a secure, off-site location (e.g., a password manager). Without this key, all your system-generated backups are impossible to recover.
