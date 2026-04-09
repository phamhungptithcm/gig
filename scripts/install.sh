#!/usr/bin/env sh

set -eu

repo="${GIG_REPO:-phamhungptithcm/gig}"
install_dir="${GIG_INSTALL_DIR:-}"
version="${GIG_VERSION:-latest}"

usage() {
  cat <<'EOF'
Usage: install.sh [--version vYYYY.MM.DD] [--repo owner/name] [--install-dir /path/to/bin]
EOF
}

normalize_version() {
  input="${1:-latest}"

  if [ -z "${input}" ] || [ "${input}" = "latest" ]; then
    printf '%s\n' "latest"
    return
  fi

  case "${input}" in
    v*) printf '%s\n' "${input}" ;;
    V*) printf 'v%s\n' "${input#?}" ;;
    *) printf 'v%s\n' "${input}" ;;
  esac
}

fetch_text() {
  request_url="${1}"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${request_url}"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO- "${request_url}"
    return
  fi

  echo "curl or wget is required to install gig." >&2
  exit 1
}

download_file() {
  request_url="${1}"
  destination="${2}"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${request_url}" -o "${destination}"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO "${destination}" "${request_url}"
    return
  fi

  echo "curl or wget is required to install gig." >&2
  exit 1
}

sha256_for() {
  path="${1}"

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${path}" | awk '{print $1}'
    return
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${path}" | awk '{print $1}'
    return
  fi

  echo "shasum or sha256sum is required to verify gig." >&2
  exit 1
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || {
        echo "--version requires a value" >&2
        exit 1
      }
      version="$2"
      shift 2
      ;;
    --repo)
      [ "$#" -ge 2 ] || {
        echo "--repo requires a value" >&2
        exit 1
      }
      repo="$2"
      shift 2
      ;;
    --install-dir)
      [ "$#" -ge 2 ] || {
        echo "--install-dir requires a value" >&2
        exit 1
      }
      install_dir="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

version="$(normalize_version "${version}")"

uname_s="$(uname -s)"
uname_m="$(uname -m)"

case "${uname_s}" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "Unsupported operating system: ${uname_s}" >&2
    exit 1
    ;;
esac

case "${uname_m}" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "Unsupported architecture: ${uname_m}" >&2
    exit 1
    ;;
esac

release_api="https://api.github.com/repos/${repo}/releases/latest"
if [ "${version}" != "latest" ]; then
  release_api="https://api.github.com/repos/${repo}/releases/tags/${version}"
fi

release_json="$(fetch_text "${release_api}")"
resolved_version="$(printf '%s' "${release_json}" | tr -d '\n' | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p')"

if [ -z "${resolved_version}" ]; then
  echo "Failed to resolve the requested gig release from GitHub." >&2
  exit 1
fi

stable_asset="gig_${os}_${arch}.tar.gz"
versioned_asset="gig_${resolved_version#v}_${os}_${arch}.tar.gz"
checksums_url="https://github.com/${repo}/releases/download/${resolved_version}/gig_${resolved_version#v}_checksums.txt"
checksums="$(fetch_text "${checksums_url}")"

choose_install_dir() {
  if [ -n "${install_dir}" ]; then
    printf '%s\n' "${install_dir}"
    return
  fi

  for candidate in "/usr/local/bin" "/opt/homebrew/bin" "$HOME/.local/bin" "$HOME/bin"; do
    case ":$PATH:" in
      *":${candidate}:"*)
        if [ -d "${candidate}" ] && [ -w "${candidate}" ]; then
          printf '%s\n' "${candidate}"
          return
        fi
        ;;
    esac
  done

  old_ifs="${IFS}"
  IFS=":"
  for candidate in $PATH; do
    if [ -n "${candidate}" ] && [ -d "${candidate}" ] && [ -w "${candidate}" ]; then
      IFS="${old_ifs}"
      printf '%s\n' "${candidate}"
      return
    fi
  done
  IFS="${old_ifs}"

  printf '%s\n' "$HOME/.local/bin"
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT INT TERM

archive_asset="${stable_asset}"
archive_url="https://github.com/${repo}/releases/download/${resolved_version}/${archive_asset}"
archive_path="${tmpdir}/${archive_asset}"

if ! download_file "${archive_url}" "${archive_path}"; then
  archive_asset="${versioned_asset}"
  archive_url="https://github.com/${repo}/releases/download/${resolved_version}/${archive_asset}"
  archive_path="${tmpdir}/${archive_asset}"
  download_file "${archive_url}" "${archive_path}"
fi

expected_sha="$(printf '%s\n' "${checksums}" | awk -v target="${archive_asset}" '$2 == target {print $1; exit}')"
if [ -z "${expected_sha}" ]; then
  echo "Failed to find a checksum for ${archive_asset} in ${checksums_url}." >&2
  exit 1
fi

actual_sha="$(sha256_for "${archive_path}")"
if [ "${actual_sha}" != "${expected_sha}" ]; then
  echo "Checksum verification failed for ${archive_asset}." >&2
  echo "Expected: ${expected_sha}" >&2
  echo "Actual:   ${actual_sha}" >&2
  exit 1
fi

install_dir="$(choose_install_dir)"
target_path="${install_dir}/gig"
action="installed"
if [ -f "${target_path}" ]; then
  action="updated"
fi

mkdir -p "${tmpdir}/extract"
tar -xzf "${archive_path}" -C "${tmpdir}/extract"

mkdir -p "${install_dir}"
cp "${tmpdir}/extract/gig" "${target_path}"
chmod 755 "${target_path}"

echo "gig ${action} to ${target_path}"
echo
"${target_path}" version
echo

case ":$PATH:" in
  *":${install_dir}:"*)
    echo "Run: gig scan --path ."
    ;;
  *)
    echo "Add ${install_dir} to your PATH to run 'gig' from anywhere."
    echo "Example:"
    echo "  export PATH=\"${install_dir}:\$PATH\""
    ;;
esac
