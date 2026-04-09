#!/usr/bin/env bash

set -euo pipefail

version="${1:?usage: build-release-assets.sh <tag> [output-dir]}"
output_dir="${2:-dist}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

mkdir -p "${repo_root}/${output_dir}"
find "${repo_root}/${output_dir}" -mindepth 1 -maxdepth 1 -delete

targets=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
)

for target in "${targets[@]}"; do
  read -r goos goarch <<<"${target}"

  stage_dir="$(mktemp -d)"
  binary_name="gig"
  archive_name="gig_${version#v}_${goos}_${goarch}"

  if [[ "${goos}" == "windows" ]]; then
    binary_name="gig.exe"
  fi

  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 \
    go build -trimpath -o "${stage_dir}/${binary_name}" ./cmd/gig

  cp "${repo_root}/README.md" "${stage_dir}/"

  if [[ "${goos}" == "windows" ]]; then
    (
      cd "${stage_dir}"
      zip -qr "${repo_root}/${output_dir}/${archive_name}.zip" .
    )
  else
    tar -C "${stage_dir}" -czf "${repo_root}/${output_dir}/${archive_name}.tar.gz" .
  fi

  rm -rf "${stage_dir}"
done
