class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.6/lpu-mac.tar.gz"
  sha256 "fe5f6c68614a54a287352b35dbb9b67b6e0a56cfc855abb996768e8f219f4b76"
  version "1.0.6"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
