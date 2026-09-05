class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.24/lpu-mac.tar.gz"
  sha256 "b266fc3aad9b3a45bdd41403fc7ada4af5256f9561f41ad49643d06fe8c9331f"
  version "1.0.24"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
