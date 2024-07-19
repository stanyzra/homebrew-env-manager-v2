class EnvManagerV2 < Formula
  desc "CLI Application to manage environment variables in Kubernetes Cluster"
  homepage "https://github.com/stanyzra/homebrew-env-manager-v2"
  url "https://github.com/stanyzra/homebrew-env-manager-v2/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "106ade5880897e43643c9c831091f530b79c64ca4968e3d31569fd249c2fad2d"
  license "Apache-2.0"
  head "https://github.com/stanyzra/homebrew-env-manager-v2.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", "-o", bin/"env-manager-v2"
  end

  test do
    assert_match "0.1.0", shell_output("#{bin}/env-manager-v2 --version")
  end
end
