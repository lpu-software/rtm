class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.9/lpu-mac.tar.gz"
  sha256 "536e7dbf61cb32e1276aaadddc13d574c9b2ba7105ad2381d56fe0cd690fa91c"
  version "1.0.9"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
