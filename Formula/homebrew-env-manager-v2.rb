# typed: true
# frozen_string_literal: true

# This file was generated by Homebrew Releaser. DO NOT EDIT.
class HomebrewEnvManagerV2 < Formula
  desc "Environment variable manager for aws amplify, dgo apps and oci object storage,"
  homepage "https://github.com/stanyzra/homebrew-env-manager-v2"
  url "https://github.com/stanyzra/homebrew-env-manager-v2/archive/refs/tags/v2.1.0.tar.gz"
  sha256 "81ace25150bd7e19cf08b8bc102c06e743030cd25f290cbfc5f505d26c741881"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    system "go", "build", "-o", bin/"env-manager-v2"
  end

  test do
    assert_match "env-manager-v2 version 2.1.0", shell_output("#{bin}/env-manager-v2 --version")
  end
end
