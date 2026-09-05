class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.27/lpu-mac.tar.gz"
  sha256 "e3a0dc26e2119f65c8a11afb3ad49ba3f51ddbffed5503f667328f1ad6664319"
  version "1.0.27"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
