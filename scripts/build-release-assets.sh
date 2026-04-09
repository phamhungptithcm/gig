#!/usr/bin/env bash

set -euo pipefail

version="${1:?usage: build-release-assets.sh <tag> [output-dir]}"
output_dir="${2:-dist}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
commit="$(git -C "${repo_root}" rev-parse --short HEAD)"
build_date="$(git -C "${repo_root}" log -1 --format=%cI HEAD)"
ldflags="-s -w -buildid= -X gig/internal/buildinfo.Version=${version} -X gig/internal/buildinfo.Commit=${commit} -X gig/internal/buildinfo.Date=${build_date}"

mkdir -p "${repo_root}/${output_dir}"
find "${repo_root}/${output_dir}" -mindepth 1 -maxdepth 1 -delete

targets=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
  "windows arm64"
)

for target in "${targets[@]}"; do
  read -r goos goarch <<<"${target}"

  stage_dir="$(mktemp -d)"
  binary_name="gig"
  archive_name="gig_${version#v}_${goos}_${goarch}"
  stable_archive_name="gig_${goos}_${goarch}"
  archive_path=""
  stable_archive_path=""

  if [[ "${goos}" == "windows" ]]; then
    binary_name="gig.exe"
  fi

  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 \
    go build -trimpath -ldflags "${ldflags}" -o "${stage_dir}/${binary_name}" ./cmd/gig

  cp "${repo_root}/README.md" "${stage_dir}/"

  if [[ "${goos}" == "windows" ]]; then
    archive_path="${repo_root}/${output_dir}/${archive_name}.zip"
    stable_archive_path="${repo_root}/${output_dir}/${stable_archive_name}.zip"
    (
      cd "${stage_dir}"
      zip -qr "${archive_path}" .
    )
  else
    archive_path="${repo_root}/${output_dir}/${archive_name}.tar.gz"
    stable_archive_path="${repo_root}/${output_dir}/${stable_archive_name}.tar.gz"
    tar -C "${stage_dir}" -czf "${archive_path}" .
  fi

  cp "${archive_path}" "${stable_archive_path}"

  rm -rf "${stage_dir}"
done

checksum_file="${repo_root}/${output_dir}/gig_${version#v}_checksums.txt"

if command -v shasum >/dev/null 2>&1; then
  (
    cd "${repo_root}/${output_dir}"
    shasum -a 256 gig_* | grep -v "gig_${version#v}_checksums.txt" > "${checksum_file}"
  )
elif command -v sha256sum >/dev/null 2>&1; then
  (
    cd "${repo_root}/${output_dir}"
    sha256sum gig_* | grep -v "gig_${version#v}_checksums.txt" > "${checksum_file}"
  )
fi
