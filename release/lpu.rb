class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.11/lpu-mac.tar.gz"
  sha256 "97be8418e029f3b9a1ce2cb808479a7c09b6ab90f11003cef455d8e11398a94f"
  version "1.0.11"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
