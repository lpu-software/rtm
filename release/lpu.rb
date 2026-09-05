class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.23/lpu-mac.tar.gz"
  sha256 "af74901ef042a323353ab525175f609862a5add39ad20967a4cd450ad8009fe4"
  version "1.0.23"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
