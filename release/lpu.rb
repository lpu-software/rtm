class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/lpu-software/rtm"
  url "https://github.com/lpu-software/rtm/releases/download/v1.0.22/lpu-mac.tar.gz"
  sha256 "b7852c9ff712cd07511115f3a58c747b526eda9b24d35e26dc7dee9ab90de0d7"
  version "1.0.22"

  def install
    # Install the binary
    bin.install "lpu-mac" => "lpu"
  end
end
