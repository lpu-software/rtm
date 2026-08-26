class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.13/lpu-mac.tar.gz"
  sha256 "82eb697cf1e7a5e08a732a440af04a76f58da7efc346ec3a1edb2ee0265f92d7"
  version "1.0.13"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
