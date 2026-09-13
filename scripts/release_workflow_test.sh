#!/usr/bin/env bash

set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
CI="$ROOT/.github/workflows/ci.yml"
RELEASE="$ROOT/.github/workflows/release.yml"

pass=0
fail=0

check() {
  local description=$1
  shift
  if "$@"; then
    printf '  PASS %s\n' "$description"
    pass=$((pass + 1))
  else
    printf '  FAIL %s\n' "$description" >&2
    fail=$((fail + 1))
  fi
}

contains() {
  local file=$1
  local pattern=$2
  grep -Eq "$pattern" "$file"
}

lacks() {
  local file=$1
  local pattern=$2
  ! grep -Eq "$pattern" "$file"
}

job_needs() {
  local file=$1
  local job=$2
  local required=$3
  awk -v job="$job" -v required="$required" '
    $0 == "  " job ":" { in_job = 1; next }
    in_job && /^  [[:alnum:]-]+:$/ { exit }
    in_job && $0 == "      - " required { found = 1 }
    END { exit !found }
  ' "$file"
}

check 'CI does not create releases' lacks "$CI" 'goreleaser|git tag|git push origin.*refs/tags|^[[:space:]]*release:'
check 'tag workflow exists' test -f "$RELEASE"
check 'release workflow is triggered by semantic version tags' contains "$RELEASE" "tags:[[:space:]]*\[['\"]v\*\.\*\.\*['\"]\]"
check 'release validates the tag milestone before publishing' contains "$RELEASE" 'scripts/check-linkage\.sh --for-tag'
check 'release verifies the tagged source' contains "$RELEASE" 'go test -race -coverprofile=coverage\.out ./\.\.\.'
check 'release verifies integration tests' contains "$RELEASE" 'go test -tags=integration -race -count=1 ./\.\.\.'
check 'release publishes GitHub assets after the guards' contains "$RELEASE" 'goreleaser/goreleaser-action'
check 'release publishes the Docker image after the GitHub release' contains "$RELEASE" 'docker/login-action'
check 'Docker publication waits for the GitHub Release' job_needs "$RELEASE" publish-image publish-release

printf '%s passed, %s failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
