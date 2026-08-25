#!/bin/bash
set -e

echo "Packaging Mac binaries for Homebrew Release..."

mkdir -p release
cd web/bin

# We will package the arm64 (Apple Silicon) binary for the tap as default,
# or we can package a universal binary if we use lipo. Let's just create a universal one!
echo "Creating Universal Binary for Mac..."
lipo -create -output lpu-mac-universal lpu-darwin-amd64 lpu-darwin-arm64

# Compress it
tar -czvf ../../release/lpu-mac.tar.gz lpu-mac-universal
cd ../../release

# Calculate SHA256 checksum
SHA=$(shasum -a 256 lpu-mac.tar.gz | awk '{print $1}')

echo "=============================================="
echo "Release Tarball Created: release/lpu-mac.tar.gz"
echo "SHA256 Checksum: $SHA"
echo "=============================================="

# Write the formula template
cat > lpu.rb <<EOF
class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/yatishydv/rtm"
  url "https://github.com/yatishydv/rtm/releases/download/v1.0.0/lpu-mac.tar.gz"
  sha256 "${SHA}"
  version "1.0.0"

  def install
    # Rename the universal binary back to just 'lpu' during installation
    bin.install "lpu-mac-universal" => "lpu"
  end
end
EOF

echo "Homebrew Formula Template generated at release/lpu.rb"
