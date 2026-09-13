#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
workflow="$root/.github/workflows/dependabot-metadata.yml"
failures=0

require_text() {
  local expected="$1"
  if ! grep -Fq -- "$expected" "$workflow"; then
    printf 'FAIL missing workflow invariant: %s\n' "$expected"
    failures=$((failures + 1))
  fi
}

require_text 'pull_request_target:'
require_text "github.event.pull_request.user.login == 'dependabot[bot]'"
require_text 'issues: write'
require_text "state: 'open'"
require_text 'left.number - right.number'
require_text 'github.rest.issues.update'

if grep -Fq 'actions/checkout' "$workflow"; then
  echo 'FAIL trusted workflow must not check out pull-request content'
  failures=$((failures + 1))
fi

[ "$failures" = 0 ] || exit 1
echo 'PASS Dependabot metadata workflow checks'
