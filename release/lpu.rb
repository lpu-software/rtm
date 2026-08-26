class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.17/lpu-mac.tar.gz"
  sha256 "bd03984b7eda249eb83e3b7173eab4ecdaed13cacfb9bf5410151920382536f2"
  version "1.0.17"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
