#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

targets=(
	"linux/amd64"
	"linux/arm64"
	"darwin/amd64"
	"darwin/arm64"
	"windows/amd64"
	"windows/arm64"
)

for t in "${targets[@]}"; do
	GOOS="${t%/*}" GOARCH="${t#*/}" bash "${ROOT_DIR}/scripts/build-artifact.sh"
done
