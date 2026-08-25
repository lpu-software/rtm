class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.12/lpu-mac.tar.gz"
  sha256 "ee75d47825fec3f0f5868184271a35b0d768c350226724bb0d0c44e71d30982e"
  version "1.0.12"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
