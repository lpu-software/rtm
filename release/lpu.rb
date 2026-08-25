class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.2/lpu-mac.tar.gz"
  sha256 "bf6f4dd094c0bfe97d530575d0c9232d2f72448165712bad36db20c486a67762"
  version "1.0.2"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
