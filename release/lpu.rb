class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.29/lpu-mac.tar.gz"
  sha256 "87c83c8bb425fd8a08143876f5b2870e064953975df7f270d36c8a47dee5107e"
  version "1.0.29"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
