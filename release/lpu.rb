class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.1/lpu-mac.tar.gz"
  sha256 "a9677a8f270210acf8f0d60e19be43c03843f1b4ada70dc42db861f31868a5d2"
  version "1.0.1"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
