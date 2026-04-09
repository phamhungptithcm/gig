class GigCli < Formula
  desc "CLI for tracking ticket-related commits across multiple repositories"
  homepage "https://github.com/phamhungptithcm/gig"
  version "2026.04.09"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/phamhungptithcm/gig/releases/download/v2026.04.09/gig_2026.04.09_darwin_arm64.tar.gz"
      sha256 "b974dd2549e1f18207998b0d2d3d7a1136d274d251d9ca826b27d7619b2bba2a"
    else
      url "https://github.com/phamhungptithcm/gig/releases/download/v2026.04.09/gig_2026.04.09_darwin_amd64.tar.gz"
      sha256 "f79b8550a11b3d2b43228dfd54bf00bdb065e15bbf54a80a5aca1e5232034d22"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/phamhungptithcm/gig/releases/download/v2026.04.09/gig_2026.04.09_linux_arm64.tar.gz"
      sha256 "7b47861143f9cbfb2f334d7d7373e250cdf71bf36c41b040850903ddb28b51fc"
    else
      url "https://github.com/phamhungptithcm/gig/releases/download/v2026.04.09/gig_2026.04.09_linux_amd64.tar.gz"
      sha256 "0e34570042464a83d5a52f6bf770844b079827b7731ebe1f2d6e8300511d06b5"
    end
  end

  def install
    bin.install "gig"
    doc.install "README.md"
  end

  test do
    output = shell_output("#{bin}/gig version")
    assert_match "gig v#{version}", output
  end
end
