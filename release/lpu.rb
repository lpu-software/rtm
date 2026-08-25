class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.8/lpu-mac.tar.gz"
  sha256 "224cc19303ff2ea9f360a846bc7d939dbe21d13ea536fb7871ae8d02c49d8be7"
  version "1.0.8"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
