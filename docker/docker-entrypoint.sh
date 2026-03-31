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

# Create initial core configurations if missing
echo "📋 Checking core configurations..."

# Xray initial config
if [ ! -f "/app/data/cores/xray/config.json" ]; then
    echo "  📝 Creating initial Xray config..."
    cat > /app/data/cores/xray/config.json << 'XRAY_EOF'
{
  "log": {
    "loglevel": "warning"
  },
  "api": {
    "tag": "api",
    "services": ["HandlerService", "StatsService"]
  },
  "stats": {},
  "policy": {
    "levels": {
      "0": {
        "statsUserUplink": true,
        "statsUserDownlink": true
      }
    },
    "system": {
      "statsInboundUplink": true,
      "statsInboundDownlink": true
    }
  },
  "inbounds": [
    {
      "tag": "api",
      "listen": "127.0.0.1",
      "port": 10085,
      "protocol": "dokodemo-door",
      "settings": {
        "address": "127.0.0.1"
      }
    }
  ],
  "outbounds": [
    {
      "tag": "direct",
      "protocol": "freedom"
    },
    {
      "tag": "blocked",
      "protocol": "blackhole"
    }
  ],
  "routing": {
    "rules": [
      {
        "type": "field",
        "inboundTag": ["api"],
        "outboundTag": "api"
      }
    ]
  }
}
XRAY_EOF
    echo "  ✅ Xray config created"
else
    echo "  ✅ Xray config exists"
fi

# Sing-box initial config
if [ ! -f "/app/data/cores/singbox/config.json" ]; then
    echo "  📝 Creating initial Sing-box config..."
    cat > /app/data/cores/singbox/config.json << 'SINGBOX_EOF'
{
  "log": {
    "level": "warn"
  },
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:9090",
      "secret": "isolate-singbox-key"
    }
  },
  "inbounds": [],
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    },
    {
      "type": "block",
      "tag": "block"
    }
  ]
}
SINGBOX_EOF
    echo "  ✅ Sing-box config created"
else
    echo "  ✅ Sing-box config exists"
fi

# Mihomo initial config
if [ ! -f "/app/data/cores/mihomo/config.yaml" ]; then
    echo "  📝 Creating initial Mihomo config..."
    cat > /app/data/cores/mihomo/config.yaml << 'MIHOMO_EOF'
mixed-port: 0
allow-lan: false
mode: rule
log-level: warning
external-controller: 127.0.0.1:9091
secret: "isolate-mihomo-key"

proxies: []

proxy-groups: []

rules:
  - MATCH,DIRECT
MIHOMO_EOF
    echo "  ✅ Mihomo config created"
else
    echo "  ✅ Mihomo config exists"
fi

# Create symlink for config file (viper looks in /app by default)
if [ -f /app/configs/config.yaml ] && [ ! -f /app/config.yaml ]; then
    echo "📋 Creating config symlink..."
    ln -sf /app/configs/config.yaml /app/config.yaml
    echo "  ✅ Config symlink created: /app/config.yaml -> /app/configs/config.yaml"
fi

# Initialize database
echo "📊 Initializing database..."
DB_PATH="/app/data/isolate-panel.db"

echo "  📥 Running migrations..."
isolate-migrate -db "$DB_PATH" -cmd up

echo "  ⚙️ Setting up application defaults..."
isolate-migrate -db "$DB_PATH" -cmd setup

echo "  ✅ Database initialization complete"

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
