#!/usr/bin/env bash
# Install the valguard binary from GitHub Releases (verified with checksums.txt).
#   curl -fsSL https://raw.githubusercontent.com/valguard-ai/valguard/main/scripts/install.sh | bash
#
# Optional env:
#   VALGUARD_VERSION=v0.1.3   (release tag)
#   VALGUARD_INSTALL_DIR=~/.local/bin
set -euo pipefail

VERSION="${VALGUARD_VERSION:-v0.1.3}"
VER="${VERSION#v}"
REPO="valguard-ai/valguard"
BASE="https://github.com/${REPO}/releases/download/${VERSION}"
INSTALL_DIR="${VALGUARD_INSTALL_DIR:-${HOME}/.local/bin}"

case "$(uname -s)" in
  Darwin) GOOS=darwin ;;
  Linux) GOOS=linux ;;
  MINGW*|MSYS*|CYGWIN*)
    echo "valguard install: on Windows use:" >&2
    echo "  go install github.com/valguard-ai/valguard/cmd/valguard@${VERSION}" >&2
    echo "  or pip install valguard-cli" >&2
    exit 1
    ;;
  *)
    echo "valguard install: unsupported OS; try pip install valguard-cli or go install" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) GOARCH=amd64 ;;
  aarch64 | arm64) GOARCH=arm64 ;;
  *)
    echo "valguard install: unsupported CPU arch $(uname -m)" >&2
    exit 1
    ;;
esac

ASSET="valguard_${VER}_${GOOS}_${GOARCH}.tar.gz"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "valguard install: ${VERSION} ${GOOS}/${GOARCH} -> ${INSTALL_DIR}"
curl -fsSL "${BASE}/checksums.txt" -o "${TMP}/checksums.txt"
curl -fsSL "${BASE}/${ASSET}" -o "${TMP}/${ASSET}"
line="$(grep -F "${ASSET}" "${TMP}/checksums.txt" | head -1 || true)"
if [[ -z "${line}" ]]; then
  echo "valguard install: ${ASSET} missing from checksums.txt" >&2
  exit 1
fi
want="${line%% *}"
got="$(sha256sum "${TMP}/${ASSET}" | awk '{print $1}')"
if [[ "${want}" != "${got}" ]]; then
  echo "valguard install: checksum mismatch for ${ASSET}" >&2
  exit 1
fi
tar -xzf "${TMP}/${ASSET}" -C "${TMP}" valguard
mkdir -p "${INSTALL_DIR}"
install -m 0755 "${TMP}/valguard" "${INSTALL_DIR}/valguard"

if ! command -v valguard >/dev/null 2>&1; then
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
      echo "Add ${INSTALL_DIR} to PATH, for example:"
      echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
      ;;
  esac
fi

"${INSTALL_DIR}/valguard" version