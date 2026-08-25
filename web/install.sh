#!/bin/bash
set -e

echo "=============================================="
echo "       LPU Zero-Installation Setup            "
echo "=============================================="

# Detect OS and Arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case $ARCH in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
esac

BINARY_NAME="lpu-${OS}-${ARCH}"
# Assuming the script is downloaded from the same host serving the binaries
DOWNLOAD_URL="https://heavy-towns-deny.loca.lt/bin/${BINARY_NAME}"

if [[ "$OS" != "darwin" && "$OS" != "linux" ]]; then
    echo "Unsupported OS: $OS"
    exit 1
fi

echo "Detected Platform: $OS ($ARCH)"
echo "Downloading LPU..."

# Download to temp directory
curl -sSL "$DOWNLOAD_URL" -o /tmp/lpu

if [ ! -s /tmp/lpu ]; then
    echo "Error: Failed to download the executable from $DOWNLOAD_URL"
    exit 1
fi

chmod +x /tmp/lpu

echo "Installing LPU globally (may require password)..."
sudo mv /tmp/lpu /usr/local/bin/lpu

echo "=============================================="
echo "Installation complete!"
echo "You can now run LPU from anywhere by typing:"
echo "  lpu lele    (to share this computer)"
echo "  lpu dede    (to connect to another computer)"
echo "=============================================="

# Optionally start it immediately:
# lpu lele
