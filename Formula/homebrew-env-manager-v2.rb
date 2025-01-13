# typed: true
# frozen_string_literal: true

# This file was generated by Homebrew Releaser. DO NOT EDIT.
class HomebrewEnvManagerV2 < Formula
  desc "Environment variable manager for aws amplify, dgo apps and oci object storage,"
  homepage "https://github.com/stanyzra/homebrew-env-manager-v2"
  url "https://github.com/stanyzra/homebrew-env-manager-v2/archive/refs/tags/v1.2.1.tar.gz"
  sha256 "5aa4890ed32d3e34d85df3658ebd71608f0619a003f5820d57442a6e0536f708"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    system "go", "build", "-o", bin/"env-manager-v2"
  end

  test do
    assert_match "1.1.0", shell_output("#{bin}/env-manager-v2 --version")
  end
end
