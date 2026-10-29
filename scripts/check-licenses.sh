#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Match go.mod toolchain so go-licenses sees the same GOROOT as `go list` (avoids
# "does not have module info" on stdlib when go-licenses was built with another Go).
GO_LICENSES_VERSION="v2.0.1"
GO_TOOLCHAIN="${GO_TOOLCHAIN:-go1.27.3}"
SCAN="./cmd/keep"

export GOTOOLCHAIN="$GO_TOOLCHAIN"
if ! command -v go-licenses >/dev/null 2>&1; then
  go install "github.com/google/go-licenses/v2@${GO_LICENSES_VERSION}"
fi
export PATH="$(go env GOPATH)/bin:$PATH"

deny='GPL|AGPL|GNU General Public|GNU Affero'
found=0

while IFS=, read -r pkg lic _; do
  pkg="${pkg//\"/}"
  lic="${lic//\"/}"
  if echo "$lic" | grep -qiE "$deny"; then
    echo "disallowed license: $pkg ($lic)" >&2
    found=1
  fi
done < <(go-licenses csv "$SCAN")

if ! go-licenses check --allowed_licenses=MIT,Apache-2.0,BSD-3-Clause,BSD-2-Clause,ISC "$SCAN"; then
  found=1
fi

if [[ "$found" -ne 0 ]]; then
  echo "dependency license check failed (GPL/AGPL not allowed)" >&2
  exit 1
fi

echo "dependency license check passed (no GPL/AGPL)"
