class GigCli < Formula
  desc "CLI for tracking ticket-related commits across multiple repositories"
  homepage "https://github.com/phamhungptithcm/gig"
  version "0.1.5"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/phamhungptithcm/gig/releases/download/v0.1.5/gig_0.1.5_darwin_arm64.tar.gz"
      sha256 "cdb57fd0c4eb511bda07026d807925f2eae057e944ddf40628fdcd4209bbae49"
    else
      url "https://github.com/phamhungptithcm/gig/releases/download/v0.1.5/gig_0.1.5_darwin_amd64.tar.gz"
      sha256 "391b2c35452c2c4c7d62214c0e2dafdd82303b81deea18bc01430264b925d6ff"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/phamhungptithcm/gig/releases/download/v0.1.5/gig_0.1.5_linux_arm64.tar.gz"
      sha256 "1e8a59a9748241e77d1eaf22d03e450800312b8326ca28cf903731b12e6a9696"
    else
      url "https://github.com/phamhungptithcm/gig/releases/download/v0.1.5/gig_0.1.5_linux_amd64.tar.gz"
      sha256 "439a41451b7c65ea1c6737cd4a22adad6dd09b0bb6bd4f98e651303d61785491"
    end
  end

  def install
    bin.install "gig"
    doc.install "README.md"
  end

  test do
    output = shell_output("#{bin}/gig version")
    assert_match "gig 0.1.5", output
  end
end
