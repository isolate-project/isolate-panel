#!/bin/bash
set -euo pipefail

# download-cores.sh - Download and verify proxy core binaries with SHA256 checksums
# Usage: ./download-cores.sh [version]
# Example: ./download-cores.sh v25.3.3

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CORES_DIR="${SCRIPT_DIR}/../cores"
CHECKSUMS_FILE="${CORES_DIR}/checksums.txt"

# Versions (pin to specific releases - NEVER use "latest")
XRAY_VERSION="${1:-v25.3.3}"
MIHOMO_VERSION="${1:-v1.19.10}"
SINGBOX_VERSION="${1:-v1.11.8}"

ARCH="amd64"
OS="linux"

mkdir -p "${CORES_DIR}/xray" "${CORES_DIR}/mihomo" "${CORES_DIR}/singbox"

download_and_verify() {
    local name="$1"
    local url="$2"
    local dest="$3"
    local expected_hash="$4"

    echo "Downloading ${name}..."
    wget -q --show-progress "${url}" -O "${dest}.tmp"

    echo "Verifying SHA256 checksum for ${name}..."
    local actual_hash
    actual_hash=$(sha256sum "${dest}.tmp" | awk '{print $1}')

    if [ "${actual_hash}" != "${expected_hash}" ]; then
        echo "ERROR: SHA256 mismatch for ${name}!" >&2
        echo "  Expected: ${expected_hash}" >&2
        echo "  Actual:   ${actual_hash}" >&2
        rm -f "${dest}.tmp"
        exit 1
    fi

    mv "${dest}.tmp" "${dest}"
    chmod +x "${dest}"
    echo "Verified ${name} (${actual_hash})"
}

# Xray Core
download_and_verify \
    "Xray-core" \
    "https://github.com/XTLS/Xray-core/releases/download/${XRAY_VERSION}/Xray-${OS}-${ARCH}.zip" \
    "${CORES_DIR}/xray/xray.zip" \
    "PLACEHOLDER_XRAY_SHA256"

unzip -q "${CORES_DIR}/xray/xray.zip" -d "${CORES_DIR}/xray/"
rm -f "${CORES_DIR}/xray/xray.zip"

# Mihomo Core
download_and_verify \
    "Mihomo" \
    "https://github.com/MetaCubeX/mihomo/releases/download/${MIHOMO_VERSION}/mihomo-${OS}-${ARCH}-${MIHOMO_VERSION}.gz" \
    "${CORES_DIR}/mihomo/mihomo.gz" \
    "PLACEHOLDER_MIHOMO_SHA256"

gunzip -c "${CORES_DIR}/mihomo/mihomo.gz" > "${CORES_DIR}/mihomo/mihomo"
rm -f "${CORES_DIR}/mihomo/mihomo.gz"
chmod +x "${CORES_DIR}/mihomo/mihomo"

# Sing-box Core
download_and_verify \
    "Sing-box" \
    "https://github.com/SagerNet/sing-box/releases/download/${SINGBOX_VERSION}/sing-box-${SINGBOX_VERSION}-${OS}-${ARCH}.tar.gz" \
    "${CORES_DIR}/singbox/sing-box.tar.gz" \
    "PLACEHOLDER_SINGBOX_SHA256"

tar -xzf "${CORES_DIR}/singbox/sing-box.tar.gz" -C "${CORES_DIR}/singbox/" --strip-components=1
rm -f "${CORES_DIR}/singbox/sing-box.tar.gz"

echo "All cores downloaded and verified successfully."
echo ""
echo "IMPORTANT: Update checksums.txt with actual SHA256 hashes before first use:"
echo "  sha256sum ${CORES_DIR}/xray/xray ${CORES_DIR}/mihomo/mihomo ${CORES_DIR}/singbox/sing-box > ${CHECKSUMS_FILE}"
