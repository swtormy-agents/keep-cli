#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go-licenses >/dev/null 2>&1; then
  go install github.com/google/go-licenses/v2@v2.0.1
fi

export PATH="$(go env GOPATH)/bin:$PATH"

go-licenses report ./... > THIRD_PARTY_NOTICES

mkdir -p third_party/apache-notices
find third_party/apache-notices -type f ! -name .gitkeep -delete 2>/dev/null || true

while IFS=, read -r pkg lic _; do
  pkg="${pkg//\"/}"
  lic="${lic//\"/}"
  [[ "$lic" == "Apache-2.0" ]] || continue
  dir="$(go list -m -f '{{.Dir}}' "${pkg%%/...*}" 2>/dev/null || true)"
  if [[ -z "$dir" || ! -d "$dir" ]]; then
    base="${pkg%%/*}"
    dir="$(go list -m -f '{{.Dir}}' "$base" 2>/dev/null || true)"
  fi
  if [[ -n "$dir" && -f "$dir/NOTICE" ]]; then
    safe="${pkg//\//_}"
    safe="${safe//\./_}"
    cp "$dir/NOTICE" "third_party/apache-notices/${safe}.NOTICE"
  fi
done < <(go-licenses csv ./...)

echo "Wrote THIRD_PARTY_NOTICES and third_party/apache-notices/"
