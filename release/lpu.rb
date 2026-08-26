class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.15/lpu-mac.tar.gz"
  sha256 "c8e35084265026867afd42265874924377b36ae2897cdacab92386f486efbe1b"
  version "1.0.15"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
