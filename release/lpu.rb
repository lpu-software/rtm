class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.25/lpu-mac.tar.gz"
  sha256 "51ebb814d24ca6f66868d64682e4351cc7162f1f3be271b1910e29b640a52347"
  version "1.0.25"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
