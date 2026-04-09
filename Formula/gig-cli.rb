class GigCli < Formula
  desc "CLI for tracking ticket-related commits across multiple repositories"
  homepage "https://github.com/phamhungptithcm/gig"
  version "0.1.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/phamhungptithcm/gig/releases/download/v0.1.4/gig_0.1.4_darwin_arm64.tar.gz"
      sha256 "3b847e540fa9d77ab892c27f43b85b78af77083b8d043fe450e1c06e2a9f74b8"
    else
      url "https://github.com/phamhungptithcm/gig/releases/download/v0.1.4/gig_0.1.4_darwin_amd64.tar.gz"
      sha256 "3cd3d07901e301a792caa388f5c171510fd435c37716a216e344ff99860532bc"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/phamhungptithcm/gig/releases/download/v0.1.4/gig_0.1.4_linux_arm64.tar.gz"
      sha256 "e3237673f88cfe042bed33ebab3ed6068b49f64b49b037037dc938c569b76984"
    else
      url "https://github.com/phamhungptithcm/gig/releases/download/v0.1.4/gig_0.1.4_linux_amd64.tar.gz"
      sha256 "55f677cfe6d10c2e5c1e70f2171e0e66a1253442d661ccc8dd525536a71fd904"
    end
  end

  def install
    bin.install "gig"
    doc.install "README.md"
  end

  test do
    output = shell_output("#{bin}/gig version")
    assert_match "gig 0.1.4", output
  end
end
