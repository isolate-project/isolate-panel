#!/bin/sh
set -e

echo "🚀 Isolate Panel Starting..."

# Create necessary directories
mkdir -p /app/data/cores/xray
mkdir -p /app/data/cores/mihomo
mkdir -p /app/data/cores/singbox
mkdir -p /var/log/isolate-panel
mkdir -p /var/log/supervisor
mkdir -p /app/configs

echo ""
echo "📦 Checking cores..."

# Function to copy core from image to volume if missing
copy_core_if_missing() {
    local core_name=$1
    local core_binary=$2
    local source_path="/usr/local/bin/cores/${core_binary}"
    local dest_path="/app/data/cores/${core_name}/${core_binary}"
    
    if [ ! -x "${dest_path}" ]; then
        echo "  📥 Installing ${core_name}..."
        if [ -f "${source_path}" ]; then
            cp "${source_path}" "${dest_path}"
            chmod +x "${dest_path}"
            echo "     ✅ ${core_name} installed"
        else
            echo "     ❌ ${core_name} not found in image"
        fi
    else
        echo "  ✅ ${core_name} already installed"
    fi
}

# Copy cores from image to volume if missing
copy_core_if_missing "xray" "xray"
copy_core_if_missing "mihomo" "mihomo"
copy_core_if_missing "singbox" "sing-box"

echo ""

# Create symlink for config file (viper looks in /app by default)
if [ -f /app/configs/config.yaml ] && [ ! -f /app/config.yaml ]; then
    echo "📋 Creating config symlink..."
    ln -sf /app/configs/config.yaml /app/config.yaml
    echo "  ✅ Config symlink created: /app/config.yaml -> /app/configs/config.yaml"
fi

# Initialize database
echo "📊 Checking database..."
DB_PATH="/app/data/isolate-panel.db"

# Check if admins table exists (better than checking file existence)
TABLES_EXIST=$(sqlite3 "$DB_PATH" "SELECT name FROM sqlite_master WHERE type='table' AND name='admins';" 2>/dev/null || echo "")

if [ -z "$TABLES_EXIST" ]; then
    echo "  📥 Initializing database tables..."
    
    sqlite3 "$DB_PATH" <<'SQL'
-- Admins table
CREATE TABLE IF NOT EXISTS admins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    email TEXT,
    is_super_admin INTEGER DEFAULT 0,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default admin (password: admin, Argon2id hash of 'admin')
INSERT INTO admins (username, password_hash, is_super_admin, is_active) 
VALUES ('admin', '95d28c9edeec6ae4f12da84a0a3955a8:6ed66e173f13a582cc599a3f8632f3c9c30dc60c75756ea2b34a59f24d20a298', 1, 1);

-- Cores table
CREATE TABLE IF NOT EXISTS cores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    version TEXT,
    is_enabled INTEGER DEFAULT 1,
    is_running INTEGER DEFAULT 0,
    config_path TEXT,
    log_path TEXT,
    pid INTEGER,
    uptime_seconds INTEGER DEFAULT 0,
    restart_count INTEGER DEFAULT 0,
    last_error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    email TEXT,
    uuid TEXT UNIQUE,
    subscription_token TEXT UNIQUE,
    is_active INTEGER DEFAULT 1,
    traffic_used_bytes INTEGER DEFAULT 0,
    traffic_limit_bytes INTEGER,
    expiry_date DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Inbounds table
CREATE TABLE IF NOT EXISTS inbounds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    protocol TEXT NOT NULL,
    port INTEGER NOT NULL,
    core_id INTEGER,
    listen_address TEXT DEFAULT '0.0.0.0',
    is_enabled INTEGER DEFAULT 1,
    tls_enabled INTEGER DEFAULT 0,
    reality_enabled INTEGER DEFAULT 0,
    config_json TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (core_id) REFERENCES cores(id)
);

-- Outbounds table
CREATE TABLE IF NOT EXISTS outbounds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL,
    type TEXT,
    server_address TEXT,
    server_port INTEGER,
    is_enabled INTEGER DEFAULT 1,
    priority INTEGER DEFAULT 0,
    config_json TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default cores
INSERT INTO cores (name, type, config_path, log_path) VALUES 
('singbox', 'sing-box', '/app/data/cores/singbox/config.json', '/var/log/isolate-panel/singbox.log'),
('xray', 'xray', '/app/data/cores/xray/config.json', '/var/log/isolate-panel/xray.log'),
('mihomo', 'mihomo', '/app/data/cores/mihomo/config.yaml', '/var/log/isolate-panel/mihomo.log');

-- Login attempts table (for rate limiting)
CREATE TABLE IF NOT EXISTS login_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip_address TEXT NOT NULL,
    username TEXT,
    success INTEGER DEFAULT 0,
    attempted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    user_agent TEXT
);

-- Refresh tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    admin_id INTEGER NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    revoked INTEGER DEFAULT 0,
    user_agent TEXT,
    ip_address TEXT,
    FOREIGN KEY (admin_id) REFERENCES admins(id)
);

-- Settings table
CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT UNIQUE NOT NULL,
    value TEXT NOT NULL,
    value_type TEXT DEFAULT 'string',
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Active connections table
CREATE TABLE IF NOT EXISTS active_connections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    inbound_id INTEGER,
    connection_id TEXT,
    source_ip TEXT,
    destination TEXT,
    protocol TEXT,
    upload_bytes INTEGER DEFAULT 0,
    download_bytes INTEGER DEFAULT 0,
    start_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (inbound_id) REFERENCES inbounds(id)
);

-- Certificates table
CREATE TABLE IF NOT EXISTS certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT UNIQUE NOT NULL,
    cert_path TEXT,
    key_path TEXT,
    issuer TEXT,
    expires_at DATETIME,
    is_auto_renew INTEGER DEFAULT 1,
    status TEXT DEFAULT 'pending',
    last_error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Backups table
CREATE TABLE IF NOT EXISTS backups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    path TEXT,
    size_bytes INTEGER DEFAULT 0,
    status TEXT DEFAULT 'pending',
    schedule TEXT,
    last_run DATETIME,
    next_run DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Notification settings table
CREATE TABLE IF NOT EXISTS notification_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT UNIQUE NOT NULL,
    is_enabled INTEGER DEFAULT 1,
    config_json TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Geo rules table
CREATE TABLE IF NOT EXISTS geo_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    action TEXT,
    rules_json TEXT,
    is_enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- WARP routes table
CREATE TABLE IF NOT EXISTS warp_routes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    destination TEXT,
    via_warp INTEGER DEFAULT 0,
    priority INTEGER DEFAULT 0,
    is_enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token TEXT UNIQUE NOT NULL,
    last_used DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

SQL
    
    echo "  ✅ Database initialized"
else
    echo "  ✅ Database tables exist"
fi

# Set proper permissions
chown -R root:root /app/data
chmod 755 /app/data

echo ""
echo "🌐 Starting Isolate Panel..."
echo "   Panel URL: http://localhost:8080"
echo "   Default login: admin / admin"
echo "   Logs: /var/log/isolate-panel/"
echo ""

# Execute the main command
exec "$@"
