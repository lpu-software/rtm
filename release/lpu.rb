class Lpu < Formula
  desc "LPU - Remote Terminal Management"
  homepage "https://github.com/yatishydv/rtm"
  url "https://github.com/yatishydv/rtm/releases/download/v1.0.0/lpu-mac.tar.gz"
  sha256 "3e242542ec0e4b37ee2967e369192b9713ba349f35c6feb469b65d153fcd91af"
  version "1.0.0"

  def install
    # Rename the universal binary back to just 'lpu' during installation
    bin.install "lpu-mac-universal" => "lpu"
  end
end
