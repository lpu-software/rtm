class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.14/lpu-mac.tar.gz"
  sha256 "00bdaec737d358d22ddf7164cebfef09c47ccc9b76f64cfc448da3e6838ffc82"
  version "1.0.14"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
