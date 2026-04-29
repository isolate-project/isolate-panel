#!/bin/sh
set -e

echo "🚀 Isolate Panel Starting..."

chown -R isolate:isolate /app/data /var/log/isolate-panel 2>/dev/null || true

# ── Auto-generate core API keys ──────────────────────────────────
SECRETS_FILE="/app/data/.core-secrets"
if [ ! -f "$SECRETS_FILE" ]; then
    echo "🔑 Generating core API keys..."
    SINGBOX_API_KEY=$(head -c 32 /dev/urandom | xxd -p -c 64 | head -c 64)
    MIHOMO_API_KEY=$(head -c 32 /dev/urandom | xxd -p -c 64 | head -c 64)
    echo "SINGBOX_API_KEY=$SINGBOX_API_KEY" > "$SECRETS_FILE"
    echo "MIHOMO_API_KEY=$MIHOMO_API_KEY" >> "$SECRETS_FILE"
    chmod 600 "$SECRETS_FILE"
    echo "  ✅ API keys generated and saved to $SECRETS_FILE"
fi
# shellcheck disable=SC1090
. "$SECRETS_FILE"
export CORES_SINGBOX_API_KEY="${SINGBOX_API_KEY}"
export CORES_MIHOMO_API_KEY="${MIHOMO_API_KEY}"

# ── Auto-generate JWT secret ──────────────────────────────────
if [ -z "$JWT_SECRET" ] || [ "$JWT_SECRET" = "change-this-in-production-use-a-strong-random-secret" ]; then
    if [ -f "$SECRETS_FILE" ]; then
        JWT_SECRET=$(grep "^JWT_SECRET=" "$SECRETS_FILE" | cut -d'=' -f2-)
    fi
    if [ -z "$JWT_SECRET" ]; then
        echo "🔐 Generating JWT secret..."
        JWT_SECRET=$(head -c 64 /dev/urandom | xxd -p -c 128 | head -c 128)
        if [ -f "$SECRETS_FILE" ]; then
            echo "JWT_SECRET=$JWT_SECRET" >> "$SECRETS_FILE"
        else
            echo "JWT_SECRET=$JWT_SECRET" > "$SECRETS_FILE"
            chmod 600 "$SECRETS_FILE"
        fi
        echo "  ✅ JWT secret generated and saved to $SECRETS_FILE"
    else
        echo "  ✅ JWT secret loaded from $SECRETS_FILE"
    fi
fi
export JWT_SECRET

# ── Auto-generate password pepper ──────────────────────────────────
if [ -z "$PASSWORD_PEPPER" ]; then
    if [ -f "$SECRETS_FILE" ]; then
        PASSWORD_PEPPER=$(grep "^PASSWORD_PEPPER=" "$SECRETS_FILE" | cut -d'=' -f2-)
    fi
    if [ -z "$PASSWORD_PEPPER" ]; then
        echo "🔐 Generating password pepper..."
        PASSWORD_PEPPER=$(head -c 32 /dev/urandom | xxd -p -c 64 | head -c 64)
        echo "PASSWORD_PEPPER=$PASSWORD_PEPPER" >> "$SECRETS_FILE"
        echo "  ✅ Password pepper generated and saved to $SECRETS_FILE"
    else
        echo "  ✅ Password pepper loaded from $SECRETS_FILE"
    fi
fi
export PASSWORD_PEPPER

# ── Auto-generate data encryption key ──────────────────────────────────
if [ -z "$DATA_ENCRYPTION_KEY" ]; then
    if [ -f "$SECRETS_FILE" ]; then
        DATA_ENCRYPTION_KEY=$(grep "^DATA_ENCRYPTION_KEY=" "$SECRETS_FILE" | cut -d'=' -f2-)
    fi
    if [ -z "$DATA_ENCRYPTION_KEY" ]; then
        echo "🔐 Generating data encryption key..."
        DATA_ENCRYPTION_KEY=$(head -c 32 /dev/urandom | xxd -p -c 64 | head -c 64)
        echo "DATA_ENCRYPTION_KEY=$DATA_ENCRYPTION_KEY" >> "$SECRETS_FILE"
        echo "  ✅ Data encryption key generated and saved to $SECRETS_FILE"
    else
        echo "  ✅ Data encryption key loaded from $SECRETS_FILE"
    fi
fi
export DATA_ENCRYPTION_KEY

# ── Auto-generate admin password (if not provided) ──────────────────────────────────
if [ -z "$ADMIN_PASSWORD" ]; then
    if [ -f "$SECRETS_FILE" ]; then
        ADMIN_PASSWORD=$(grep "^ADMIN_PASSWORD=" "$SECRETS_FILE" | cut -d'=' -f2-)
    fi
    if [ -z "$ADMIN_PASSWORD" ]; then
        echo "🔐 Generating admin password..."
        ADMIN_PASSWORD=$(head -c 24 /dev/urandom | xxd -p -c 48 | head -c 48)
        echo "ADMIN_PASSWORD=$ADMIN_PASSWORD" >> "$SECRETS_FILE"
        echo ""
        echo "  ┌─────────────────────────────────────────────────────────────┐"
        echo "  │  ⚠️  AUTO-GENERATED ADMIN PASSWORD                           │"
        echo "  │                                                             │"
        echo "  │  $ADMIN_PASSWORD                                           │"
        echo "  │                                                             │"
        echo "  │  Save this password now. It will not be shown again.       │"
        echo "  │  Store it in a password manager.                           │"
        echo "  └─────────────────────────────────────────────────────────────┘"
        echo ""
    else
        echo "  ✅ Admin password loaded from $SECRETS_FILE"
    fi
fi
export ADMIN_PASSWORD

# ── Auto-generate user credential encryption key ──────────────────────────────────
USER_CRED_KEY="/app/data/.user_cred_key"
if [ ! -f "$USER_CRED_KEY" ]; then
    echo "🔐 Generating user credential encryption key..."
    head -c 32 /dev/urandom > "$USER_CRED_KEY"
    chmod 600 "$USER_CRED_KEY"
    echo "  ✅ Encryption key generated and saved to $USER_CRED_KEY"
fi

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
            setcap cap_net_bind_service+ep "${dest_path}" 2>/dev/null || true
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
    cat > /app/data/cores/singbox/config.json << SINGBOX_EOF
{
  "log": {
    "level": "warn"
  },
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:9090",
      "secret": "${SINGBOX_API_KEY}"
    }
  },
  "inbounds": [],
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    }
  ],
  "route": {
    "final": "direct",
    "auto_detect_interface": true
  }
}
SINGBOX_EOF
    echo "  ✅ Sing-box config created"
else
    echo "  ✅ Sing-box config exists"
fi

# Mihomo initial config
if [ ! -f "/app/data/cores/mihomo/config.yaml" ]; then
    echo "  📝 Creating initial Mihomo config..."
    cat > /app/data/cores/mihomo/config.yaml << MIHOMO_EOF
mixed-port: 0
allow-lan: false
mode: rule
log-level: warning
external-controller: 127.0.0.1:9091
secret: "${MIHOMO_API_KEY}"

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
# Run migrations only if needed (check current version first)
if command -v isolate-migrate &> /dev/null; then
    CURRENT_VERSION=$(isolate-migrate -db "$DB_PATH" -cmd version 2>/dev/null | grep "Current version:" | awk '{print $3}' || echo "0")
    # If version command not supported or returns 0, always run migrations (safe, idempotent)
    if [ "$CURRENT_VERSION" = "0" ]; then
        isolate-migrate -db "$DB_PATH" -cmd up
    else
        echo "  ✅ Migrations already at version $CURRENT_VERSION, skipping"
    fi
else
    echo "  ⚠️  isolate-migrate not found, skipping migrations"
fi

echo "  🔐 Encrypting existing user passwords..."
isolate-migrate -db "$DB_PATH" -cmd encrypt-passwords

echo "  ⚙️ Setting up application defaults..."
isolate-migrate -db "$DB_PATH" -cmd setup

echo "  ✅ Database initialization complete"

# Set proper permissions
chown -R isolate:isolate /app/data
chmod 755 /app/data

echo ""
echo "🌐 Starting Isolate Panel..."
echo "   Panel URL: http://localhost:8080"
if [ -n "$ADMIN_PASSWORD" ]; then
    echo "   Login: admin / <ADMIN_PASSWORD from .env>"
else
    echo "   Default login: admin / admin"
    echo "   ⚠️  WARNING: Change default password immediately!"
fi
echo "   Logs: /var/log/isolate-panel/"
echo ""

# Execute the main command
exec "$@"
