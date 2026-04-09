#!/usr/bin/env sh

set -eu

repo="${GIG_REPO:-phamhungptithcm/gig}"
install_dir="${GIG_INSTALL_DIR:-}"
version="${GIG_VERSION:-latest}"

if [ "${version}" != "latest" ]; then
  case "${version}" in
    v*) ;;
    *) version="v${version}" ;;
  esac
fi

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

asset="gig_${os}_${arch}.tar.gz"

if [ "${version}" = "latest" ]; then
  url="https://github.com/${repo}/releases/latest/download/${asset}"
else
  url="https://github.com/${repo}/releases/download/${version}/${asset}"
fi

download() {
  destination="${1}"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${url}" -o "${destination}"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO "${destination}" "${url}"
    return
  fi

  echo "curl or wget is required to install gig." >&2
  exit 1
}

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

archive_path="${tmpdir}/${asset}"
download "${archive_path}"

install_dir="$(choose_install_dir)"

mkdir -p "${tmpdir}/extract"
tar -xzf "${archive_path}" -C "${tmpdir}/extract"

mkdir -p "${install_dir}"
cp "${tmpdir}/extract/gig" "${install_dir}/gig"
chmod 755 "${install_dir}/gig"

echo "gig installed to ${install_dir}/gig"
echo
"${install_dir}/gig" version
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
