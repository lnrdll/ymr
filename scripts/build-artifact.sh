#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-${ROOT_DIR}/dist}"
BIN_BASENAME="${BIN_BASENAME:-ymr}"
PKG="${PKG:-.}"

if [[ -z "${GOOS:-}" || -z "${GOARCH:-}" ]]; then
	cat <<'EOF' >&2
Error: GOOS and GOARCH must be set.

Example:
  GOOS=linux GOARCH=amd64 bash scripts/build-artifact.sh
EOF
	exit 2
fi

VERSION="${VERSION:-${GITHUB_REF_NAME:-}}"
if [[ -z "${VERSION}" ]]; then
	VERSION="$(git -C "${ROOT_DIR}" describe --tags --always --dirty 2>/dev/null || echo dev)"
fi

COMMIT="${COMMIT:-${GITHUB_SHA:-}}"
if [[ -z "${COMMIT}" ]]; then
	COMMIT="$(git -C "${ROOT_DIR}" rev-parse --short=7 HEAD 2>/dev/null || echo unknown)"
else
	COMMIT="${COMMIT:0:7}"
fi

DATE="${DATE:-$(date -u '+%Y-%m-%d %H:%M:%S UTC')}"

LDFLAGS_DEFAULT="-s -w -X 'github.com/lnrdll/ymr/internal/buildinfo.Version=${VERSION}' -X 'github.com/lnrdll/ymr/internal/buildinfo.Commit=${COMMIT}' -X 'github.com/lnrdll/ymr/internal/buildinfo.Date=${DATE}'"
LDFLAGS="${LDFLAGS:-${LDFLAGS_DEFAULT}}"

case "${GOOS}" in
	darwin) goos_label="Darwin" ;;
	linux) goos_label="Linux" ;;
	windows) goos_label="Windows" ;;
	*) goos_label="${GOOS}" ;;
esac
case "${GOARCH}" in
	amd64) goarch_label="x86_64" ;;
	arm64) goarch_label="arm64" ;;
	*) goarch_label="${GOARCH}" ;;
esac

bin_name="${BIN_BASENAME}"
if [[ "${GOOS}" == "windows" ]]; then
	bin_name="${BIN_BASENAME}.exe"
fi

sha256_first_field() {
	local file
	file="$1"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "${file}" | awk '{print $1}'
		return
	fi
	if command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "${file}" | awk '{print $1}'
		return
	fi
	echo "Error: need sha256sum or shasum" >&2
	exit 1
}

mkdir -p "${DIST_DIR}"

tmpdir="$(mktemp -d)"
cleanup() {
	rm -rf "${tmpdir}"
}
trap cleanup EXIT

echo "Building ${VERSION} (${COMMIT}) for ${GOOS}/${GOARCH}"
(
	cd "${ROOT_DIR}"
	CGO_ENABLED="${CGO_ENABLED:-0}" GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
		-trimpath \
		-ldflags "${LDFLAGS}" \
		-o "${tmpdir}/${bin_name}" \
		"${PKG}"
)

base="${BIN_BASENAME}_${goos_label}_${goarch_label}"

if [[ "${GOOS}" == "windows" ]]; then
	(zip -q -j "${DIST_DIR}/${base}.zip" "${tmpdir}/${bin_name}")
	sha256_first_field "${DIST_DIR}/${base}.zip" > "${DIST_DIR}/${base}.zip.sha256"
else
	(tar -C "${tmpdir}" -czf "${DIST_DIR}/${base}.tar.gz" "${bin_name}")
	sha256_first_field "${DIST_DIR}/${base}.tar.gz" > "${DIST_DIR}/${base}.tar.gz.sha256"
fi
