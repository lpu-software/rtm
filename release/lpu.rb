class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.21/lpu-mac.tar.gz"
  sha256 "2c176b0d01a782b8ed2d322e78a8bbc75158ca8fc6d47123b7130120bdab0a59"
  version "1.0.21"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
