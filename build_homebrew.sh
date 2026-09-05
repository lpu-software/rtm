#!/bin/bash
set -e

echo "Packaging Mac binaries for Homebrew Release..."

mkdir -p release
mkdir -p bin

# Compile for native architecture (arm64)
echo "Compiling native binary..."
go build -o bin/lpu-mac ./cmd/rtm

cd bin

# Compress it
tar -czvf ../release/lpu-mac.tar.gz lpu-mac
cd ../release

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
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.27/lpu-mac.tar.gz"
  sha256 "${SHA}"
  version "1.0.27"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
EOF

echo "Homebrew Formula Template generated at release/lpu.rb"
