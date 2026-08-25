class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.5/lpu-mac.tar.gz"
  sha256 "ce47cac54b38584a6b8d42289272b35a6d37a5b00a0377518c3b0a34efb46314"
  version "1.0.5"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
