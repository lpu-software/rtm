class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.7/lpu-mac.tar.gz"
  sha256 "37e6b59fec629766e8d165b8818f647339f2f20c6a2cc773279b81d7afc1a03d"
  version "1.0.7"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
