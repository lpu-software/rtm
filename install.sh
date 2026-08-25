#!/bin/bash
set -e

echo "Installing RTM (Remote Terminal Access)..."

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case $ARCH in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
esac

# In a real deployment, this would point to GitHub Releases
# e.g., URL="https://github.com/yatishydv/rtm/releases/latest/download/rtm-${OS}-${ARCH}"
# For this prototype, we'll assume the binary is already built locally, or we provide instructions.

BIN_DIR="/usr/local/bin"
if [ ! -w "$BIN_DIR" ]; then
    echo "Requires sudo to install to $BIN_DIR"
    SUDO="sudo"
else
    SUDO=""
fi

echo "For this prototype, we'll assume the binary is placed in your path."
echo "RTM command is ready. Run 'rtm host' to start."
