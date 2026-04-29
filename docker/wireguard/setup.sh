#!/bin/bash
set -euo pipefail

# setup-wireguard.sh — Generate WireGuard keys for panel-to-node tunnel
# Run once on the panel server, then distribute client configs to nodes.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="${SCRIPT_DIR}/config"
mkdir -p "${CONFIG_DIR}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Isolate Panel WireGuard Setup ===${NC}"
echo ""

# Panel (server) key pair
PANEL_PRIVATE_KEY="$(wg genkey)"
PANEL_PUBLIC_KEY="$(echo "${PANEL_PRIVATE_KEY}" | wg pubkey)"

# Generate panel server config
cat > "${CONFIG_DIR}/wg0-panel.conf" << PANELCFG
[Interface]
Address = 10.200.200.1/24
ListenPort = 51820
PrivateKey = ${PANEL_PRIVATE_KEY}
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
DNS = 1.1.1.1, 1.0.0.1

# Node 1
[Peer]
PublicKey = <NODE1_PUBLIC_KEY>
AllowedIPs = 10.200.200.2/32
PersistentKeepalive = 25

# Node 2
[Peer]
PublicKey = <NODE2_PUBLIC_KEY>
AllowedIPs = 10.200.200.3/32
PersistentKeepalive = 25

# Node 3
[Peer]
PublicKey = <NODE3_PUBLIC_KEY>
AllowedIPs = 10.200.200.4/32
PersistentKeepalive = 25
PANELCFG

# Generate node configs
for i in 1 2 3; do
    NODE_PRIVATE_KEY="$(wg genkey)"
    NODE_PUBLIC_KEY="$(echo "${NODE_PRIVATE_KEY}" | wg pubkey)"
    NODE_IP="10.200.200.$((i + 1))"

    cat > "${CONFIG_DIR}/wg0-node${i}.conf" << NODECFG
[Interface]
Address = ${NODE_IP}/32
PrivateKey = ${NODE_PRIVATE_KEY}
DNS = 1.1.1.1, 1.0.0.1

[Peer]
PublicKey = ${PANEL_PUBLIC_KEY}
AllowedIPs = 10.200.200.0/24
Endpoint = <PANEL_PUBLIC_IP>:51820
PersistentKeepalive = 25
NODECFG

    echo -e "${YELLOW}Node ${i} public key:${NC} ${NODE_PUBLIC_KEY}"
    echo "  → Add this to panel config as Peer #${i}"
    echo ""
done

echo -e "${GREEN}Panel public key:${NC} ${PANEL_PUBLIC_KEY}"
echo ""
echo -e "${RED}IMPORTANT:${NC}"
echo "1. Edit ${CONFIG_DIR}/wg0-panel.conf"
echo "   Replace <NODE1_PUBLIC_KEY>, <NODE2_PUBLIC_KEY>, <NODE3_PUBLIC_KEY>"
echo "   with the actual public keys shown above."
echo ""
echo "2. Edit each node config and set <PANEL_PUBLIC_IP> to your VPS IP."
echo ""
echo "3. For each node, copy its config to /etc/wireguard/wg0.conf and run:"
echo "   wg-quick up wg0"
echo ""
echo "4. On the panel server, place wg0-panel.conf and run:"
echo "   wg-quick up wg0"
echo ""
echo "Generated configs in: ${CONFIG_DIR}/"
ls -la "${CONFIG_DIR}/"
