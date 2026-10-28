#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
VERSION="${1:-dev}"
OUT="${ROOT}/dist"
mkdir -p "$OUT"
export GOGC=25
export GOMAXPROCS=1
for os in linux darwin windows; do
  for arch in amd64 arm64; do
    ext=""
    if [[ "$os" == windows ]]; then ext=".exe"; fi
    env GOOS="$os" GOARCH="$arch" go build -p 1 -ldflags "-X github.com/swtormy-agents/keep-cli/internal/cli.Version=${VERSION}" -o "${OUT}/keep-${VERSION}-${os}-${arch}${ext}" ./cmd/keep
  done
done
( cd "$OUT" && sha256sum keep-${VERSION}-* > "SHA256SUMS-${VERSION}.txt" )
