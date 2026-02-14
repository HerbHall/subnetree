#!/usr/bin/env bash
set -euo pipefail

# SubNetree Scout Agent Installer
# Usage: sudo ./install.sh --server http://your-subnetree-server:8080

BINARY_NAME="scout"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/scout"
SERVICE_FILE="/etc/systemd/system/scout.service"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Defaults.
SERVER_URL=""
START_NOW="false"

usage() {
    echo "Usage: $0 --server <URL> [--start]"
    echo ""
    echo "Options:"
    echo "  --server <URL>   SubNetree server URL (required)"
    echo "  --start          Start the service immediately after install"
    echo "  -h, --help       Show this help message"
    exit 1
}

# Parse arguments.
while [[ $# -gt 0 ]]; do
    case "$1" in
        --server)
            SERVER_URL="$2"
            shift 2
            ;;
        --start)
            START_NOW="true"
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

if [[ -z "$SERVER_URL" ]]; then
    echo "Error: --server is required."
    echo ""
    usage
fi

# Must run as root.
if [[ $EUID -ne 0 ]]; then
    echo "Error: This script must be run as root (use sudo)."
    exit 1
fi

echo "==> Installing SubNetree Scout Agent"

# Detect architecture.
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac
echo "    Architecture: $ARCH"

# Create scout user and group if they don't exist.
if ! id -u scout &>/dev/null; then
    echo "==> Creating scout user"
    useradd --system --no-create-home --shell /usr/sbin/nologin scout
fi

# Copy binary.
echo "==> Installing binary to $INSTALL_DIR/$BINARY_NAME"
if [[ -f "$SCRIPT_DIR/$BINARY_NAME" ]]; then
    cp "$SCRIPT_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
elif [[ -f "./$BINARY_NAME" ]]; then
    cp "./$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
else
    echo "Error: $BINARY_NAME binary not found in $SCRIPT_DIR or current directory."
    echo "Place the scout binary next to this script and try again."
    exit 1
fi
chmod 755 "$INSTALL_DIR/$BINARY_NAME"

# Create config directory.
echo "==> Creating config directory at $CONFIG_DIR"
mkdir -p "$CONFIG_DIR"
chown scout:scout "$CONFIG_DIR"

# Install systemd service file with the server URL substituted.
echo "==> Installing systemd service"
if [[ -f "$SCRIPT_DIR/scout.service" ]]; then
    sed "s|\${SERVER_URL}|$SERVER_URL|g" "$SCRIPT_DIR/scout.service" > "$SERVICE_FILE"
elif [[ -f "./scout.service" ]]; then
    sed "s|\${SERVER_URL}|$SERVER_URL|g" "./scout.service" > "$SERVICE_FILE"
else
    # Generate service file inline as fallback.
    cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=SubNetree Scout Agent
Documentation=https://github.com/HerbHall/subnetree
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=scout
Group=scout
ExecStart=$INSTALL_DIR/$BINARY_NAME --server $SERVER_URL
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
ProtectSystem=strict
ProtectHome=true
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF
fi

# Reload systemd and enable the service.
echo "==> Enabling scout service"
systemctl daemon-reload
systemctl enable scout.service

if [[ "$START_NOW" == "true" ]]; then
    echo "==> Starting scout service"
    systemctl start scout.service
    echo ""
    echo "Scout agent is running. Check status with:"
    echo "  systemctl status scout"
else
    echo ""
    echo "Scout agent installed and enabled. Start it with:"
    echo "  sudo systemctl start scout"
fi

echo ""
echo "View logs with:"
echo "  journalctl -u scout -f"
echo ""
echo "Done."
