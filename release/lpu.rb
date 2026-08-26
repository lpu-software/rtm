class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.19/lpu-mac.tar.gz"
  sha256 "f276db10a1b9b460b404f59244a3b1e201d3294f2651dc2a7b5385db6a8e05c9"
  version "1.0.19"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
