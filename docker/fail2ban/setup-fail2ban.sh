#!/bin/bash
set -euo pipefail

# setup-fail2ban.sh - Install and configure fail2ban for Isolate Panel
# Run this on the HOST (not inside Docker container)
# Requires: Ubuntu/Debian with systemd

echo "Installing fail2ban..."
apt-get update && apt-get install -y fail2ban

echo "Copying Isolate Panel filter and jail..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cp "${SCRIPT_DIR}/filter.d/isolate-panel.conf" /etc/fail2ban/filter.d/
cp "${SCRIPT_DIR}/jail.d/isolate-panel.conf" /etc/fail2ban/jail.d/

echo "Creating log directory symlink..."
mkdir -p /var/log/isolate-panel

# If using Docker, the panel logs go to Docker's JSON log driver
# We need to extract them. Create a simple script for that.
cat > /etc/fail2ban/extract-isolate-logs.sh << 'INNEREOF'
#!/bin/bash
# Extract Isolate Panel auth failures from Docker logs
CONTAINER=$(docker ps -q -f name=isolate-panel)
if [ -n "$CONTAINER" ]; then
    docker logs "$CONTAINER" 2>&1 | grep '"event":"auth_failure"'
fi
INNEREOF
chmod +x /etc/fail2ban/extract-isolate-logs.sh

echo "Reloading fail2ban..."
systemctl restart fail2ban
systemctl enable fail2ban

echo "Checking fail2ban status for isolate-panel jail..."
fail2ban-client status isolate-panel || true

echo ""
echo "fail2ban configured for Isolate Panel."
echo "Max retry: 5 attempts in 15 minutes"
echo "Ban time: 1 hour"
echo ""
echo "To test: fail2ban-client status isolate-panel"
