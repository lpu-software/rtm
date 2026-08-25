class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.10/lpu-mac.tar.gz"
  sha256 "8511864d936b0a02dacd8bdf80db94c14d82a30760f6a78c9f0a83bc2a103d5a"
  version "1.0.10"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
