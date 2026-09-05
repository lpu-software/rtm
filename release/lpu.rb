class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.26/lpu-mac.tar.gz"
  sha256 "0a4d6f633a8c1e889b87034555da3a850016c4039d0b75a44a1361c52f20d31b"
  version "1.0.26"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
