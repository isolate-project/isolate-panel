#!/bin/sh
# Initialize database with required tables

DB_PATH="/app/data/isolate-panel.db"

if [ ! -f "$DB_PATH" ] || [ ! -s "$DB_PATH" ]; then
    echo "📊 Initializing database tables..."
    
    sqlite3 "$DB_PATH" <<SQL
-- Admins table
CREATE TABLE IF NOT EXISTS admins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    email TEXT,
    is_super_admin INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default admin (password: admin)
INSERT INTO admins (username, password_hash, is_super_admin) 
VALUES ('admin', '\$2a\$10\$X.vZKHCpS7kDqJjJzJh8W.FqQxQxQxQxQxQxQxQxQxQxQxQxQxQxQ', 1);

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

-- Insert default cores
INSERT INTO cores (name, type, config_path, log_path) VALUES 
('singbox', 'sing-box', '/app/data/cores/singbox/config.json', '/var/log/isolate-panel/singbox.log'),
('xray', 'xray', '/app/data/cores/xray/config.json', '/var/log/isolate-panel/xray.log'),
('mihomo', 'mihomo', '/app/data/cores/mihomo/config.yaml', '/var/log/isolate-panel/mihomo.log');

SQL
    
    echo "  ✅ Database initialized"
else
    echo "  ✅ Database already exists"
fi
