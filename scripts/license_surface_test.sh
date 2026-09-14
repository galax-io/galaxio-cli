#!/usr/bin/env bash

set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
EXPECTED_LICENSE_SHA256=8177f97513213526df2cf6184d8ff986c675afb514d4e68a404010521b880643

if command -v sha256sum >/dev/null 2>&1; then
  actual_license_sha256=$(sha256sum "$ROOT/LICENSE" | awk '{print $1}')
else
  actual_license_sha256=$(shasum -a 256 "$ROOT/LICENSE" | awk '{print $1}')
fi

failures=0

check() {
  local description=$1
  shift
  if "$@"; then
    printf '  PASS %s\n' "$description"
  else
    printf '  FAIL %s\n' "$description" >&2
    failures=$((failures + 1))
  fi
}

check 'LICENSE is the canonical GNU GPL Version 2 text' \
  test "$actual_license_sha256" = "$EXPECTED_LICENSE_SHA256"
check 'README declares GPL-2.0-only' \
  grep -Fq '`GPL-2.0-only`' "$ROOT/README.md"
check 'Docker image declares GPL-2.0-only' \
  grep -Fq 'org.opencontainers.image.licenses="GPL-2.0-only"' "$ROOT/Dockerfile"

printf '%s licence-surface checks failed\n' "$failures"
test "$failures" -eq 0
