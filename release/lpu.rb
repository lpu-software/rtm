class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.18/lpu-mac.tar.gz"
  sha256 "f74fac6063bc0e8f8806a36d4d14c6a9e45ce4a86044e474dd4559a670f2fd93"
  version "1.0.18"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
