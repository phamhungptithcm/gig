#!/usr/bin/env bash

set -euo pipefail

version="${1:?usage: generate-package-manager-files.sh <tag> [dist-dir|--from-release]}"
source_arg="${2:---from-release}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo="phamhungptithcm/gig"
package_name="gig-cli"
version_without_v="${version#v}"

darwin_amd64_sha=""
darwin_arm64_sha=""
linux_amd64_sha=""
linux_arm64_sha=""
windows_amd64_sha=""
windows_arm64_sha=""

require_file() {
  local path="${1}"
  if [[ ! -f "${path}" ]]; then
    echo "missing required asset: ${path}" >&2
    exit 1
  fi
}

sha256_for() {
  local path="${1}"

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${path}" | awk '{print $1}'
    return
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${path}" | awk '{print $1}'
    return
  fi

  echo "shasum or sha256sum is required" >&2
  exit 1
}

require_command() {
  local name="${1}"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "${name} is required" >&2
    exit 1
  fi
}

load_shas_from_dist() {
  local dist_dir="${1}"
  local dist_path="${repo_root}/${dist_dir}"
  local darwin_amd64="${dist_path}/gig_${version_without_v}_darwin_amd64.tar.gz"
  local darwin_arm64="${dist_path}/gig_${version_without_v}_darwin_arm64.tar.gz"
  local linux_amd64="${dist_path}/gig_${version_without_v}_linux_amd64.tar.gz"
  local linux_arm64="${dist_path}/gig_${version_without_v}_linux_arm64.tar.gz"
  local windows_amd64="${dist_path}/gig_${version_without_v}_windows_amd64.zip"
  local windows_arm64="${dist_path}/gig_${version_without_v}_windows_arm64.zip"

  require_file "${darwin_amd64}"
  require_file "${darwin_arm64}"
  require_file "${linux_amd64}"
  require_file "${linux_arm64}"
  require_file "${windows_amd64}"
  require_file "${windows_arm64}"

  darwin_amd64_sha="$(sha256_for "${darwin_amd64}")"
  darwin_arm64_sha="$(sha256_for "${darwin_arm64}")"
  linux_amd64_sha="$(sha256_for "${linux_amd64}")"
  linux_arm64_sha="$(sha256_for "${linux_arm64}")"
  windows_amd64_sha="$(sha256_for "${windows_amd64}")"
  windows_arm64_sha="$(sha256_for "${windows_arm64}")"
}

release_asset_sha() {
  local release_json="${1}"
  local asset_name="${2}"

  RELEASE_JSON="${release_json}" python3 - "${asset_name}" <<'PY'
import json
import os
import sys

asset_name = sys.argv[1]
release = json.loads(os.environ["RELEASE_JSON"])

for asset in release.get("assets", []):
    if asset.get("name") != asset_name:
        continue
    digest = asset.get("digest", "")
    if digest.startswith("sha256:"):
        print(digest.split(":", 1)[1])
        sys.exit(0)
    raise SystemExit(f"missing sha256 digest for {asset_name}")

raise SystemExit(f"missing asset {asset_name}")
PY
}

load_shas_from_release() {
  require_command curl
  require_command python3

  local release_json
  release_json="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/tags/${version}")"

  darwin_amd64_sha="$(release_asset_sha "${release_json}" "gig_${version_without_v}_darwin_amd64.tar.gz")"
  darwin_arm64_sha="$(release_asset_sha "${release_json}" "gig_${version_without_v}_darwin_arm64.tar.gz")"
  linux_amd64_sha="$(release_asset_sha "${release_json}" "gig_${version_without_v}_linux_amd64.tar.gz")"
  linux_arm64_sha="$(release_asset_sha "${release_json}" "gig_${version_without_v}_linux_arm64.tar.gz")"
  windows_amd64_sha="$(release_asset_sha "${release_json}" "gig_${version_without_v}_windows_amd64.zip")"
  windows_arm64_sha="$(release_asset_sha "${release_json}" "gig_${version_without_v}_windows_arm64.zip")"
}

if [[ "${source_arg}" == "--from-release" ]]; then
  load_shas_from_release
else
  load_shas_from_dist "${source_arg}"
fi

mkdir -p "${repo_root}/Formula" "${repo_root}/bucket"

cat > "${repo_root}/Formula/${package_name}.rb" <<EOF
class GigCli < Formula
  desc "CLI for tracking ticket-related commits across multiple repositories"
  homepage "https://github.com/${repo}"
  version "${version_without_v}"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/${repo}/releases/download/${version}/gig_${version_without_v}_darwin_arm64.tar.gz"
      sha256 "${darwin_arm64_sha}"
    else
      url "https://github.com/${repo}/releases/download/${version}/gig_${version_without_v}_darwin_amd64.tar.gz"
      sha256 "${darwin_amd64_sha}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/${repo}/releases/download/${version}/gig_${version_without_v}_linux_arm64.tar.gz"
      sha256 "${linux_arm64_sha}"
    else
      url "https://github.com/${repo}/releases/download/${version}/gig_${version_without_v}_linux_amd64.tar.gz"
      sha256 "${linux_amd64_sha}"
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
EOF

cat > "${repo_root}/bucket/${package_name}.json" <<EOF
{
  "version": "${version_without_v}",
  "description": "CLI for tracking ticket-related commits across multiple repositories",
  "homepage": "https://github.com/${repo}",
  "notes": "Package name is '${package_name}', but the installed command is 'gig'. Run 'gig version' to confirm the installed build.",
  "architecture": {
    "64bit": {
      "url": "https://github.com/${repo}/releases/download/${version}/gig_${version_without_v}_windows_amd64.zip",
      "hash": "${windows_amd64_sha}"
    },
    "arm64": {
      "url": "https://github.com/${repo}/releases/download/${version}/gig_${version_without_v}_windows_arm64.zip",
      "hash": "${windows_arm64_sha}"
    }
  },
  "bin": "gig.exe",
  "checkver": {
    "github": "https://github.com/${repo}"
  },
  "autoupdate": {
    "architecture": {
      "64bit": {
        "url": "https://github.com/${repo}/releases/download/v\$version/gig_\$version_windows_amd64.zip"
      },
      "arm64": {
        "url": "https://github.com/${repo}/releases/download/v\$version/gig_\$version_windows_arm64.zip"
      }
    }
  }
}
EOF
