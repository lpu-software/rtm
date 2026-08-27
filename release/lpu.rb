class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.20/lpu-mac.tar.gz"
  sha256 "b65f8fc916940407483aa4a30cc5d0a6ab5399970922c0a7282d6e5430b27f8e"
  version "1.0.20"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
