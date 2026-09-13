#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
dockerfile="$root/Dockerfile"
dependabot="$root/.github/dependabot.yml"

module_go=$(awk '$1 == "go" { print $2; exit }' "$root/go.mod")
image_go=$(sed -nE 's/^FROM .*golang:([0-9]+\.[0-9]+\.[0-9]+)-bookworm AS build$/\1/p' "$dockerfile")

if [[ -z "$module_go" || -z "$image_go" || "$module_go" != "$image_go" ]]; then
  printf 'FAIL go.mod version %s does not match Docker builder version %s\n' \
    "${module_go:-missing}" "${image_go:-missing}" >&2
  exit 1
fi

grep -Fq 'FROM --platform=$BUILDPLATFORM golang:' "$dockerfile"
grep -Fq 'FROM gcr.io/distroless/static-debian12:nonroot' "$dockerfile"

if grep -Eq '^FROM .*\$\{' "$dockerfile"; then
  echo 'FAIL Docker base images must be literal so Dependabot can discover them' >&2
  exit 1
fi

grep -Fq 'multi-ecosystem-groups:' "$dependabot"
grep -Fq 'multi-ecosystem-group: go-runtime' "$dependabot"
grep -Fq 'package-ecosystem: gomod' "$dependabot"
grep -Fq 'package-ecosystem: docker' "$dependabot"
grep -Fq 'gcr.io/distroless/static-debian12' "$dependabot"

awk '
  $0 == "  go-runtime:" { in_group = 1; next }
  in_group && /^updates:/ { exit !found }
  in_group && $0 == "    open-pull-requests-limit: 5" { found = 1 }
  END { exit !found }
' "$dependabot"

if awk '
  /^  - package-ecosystem: (gomod|docker)$/ { in_grouped_update = 1; next }
  /^  - package-ecosystem:/ { in_grouped_update = 0 }
  in_grouped_update && /open-pull-requests-limit:/ { found = 1 }
  END { exit !found }
' "$dependabot"; then
  echo 'FAIL grouped updates must set open-pull-requests-limit on the multi-ecosystem group' >&2
  exit 1
fi

echo 'PASS Docker dependency update policy checks'
