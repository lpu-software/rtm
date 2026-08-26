class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.16/lpu-mac.tar.gz"
  sha256 "1fbaba2db6cb6a56e92516de09ae2c5397c54e0d9ac1e041ca56d9c8626d492b"
  version "1.0.16"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
