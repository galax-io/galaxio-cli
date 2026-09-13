#!/usr/bin/env bash
# Helpers shared by the release audit and its local regression checks.

# GitHub exposes the Dependabot App as app/dependabot through GraphQL (`gh pr
# view`) and as dependabot[bot] in webhook payloads. Accept only those exact
# identities when applying dependency-bot linkage rules.
is_dependabot_author() {
  case "$1" in
    app/dependabot|"dependabot[bot]") return 0 ;;
    *) return 1 ;;
  esac
}

# List commits since the preceding reachable release tag. Exclude the target tag
# itself so auditing an already-created tag does not produce an empty range.
release_commits() {
  local ref="$1" target_tag="$2" previous
  if [ "$(git rev-parse --is-shallow-repository)" = true ]; then
    echo 'error: release audit requires full history; run git fetch --unshallow --tags' >&2
    return 2
  fi
  git rev-parse --verify "${ref}^{commit}" >/dev/null || return 2
  previous=$(git describe --tags --abbrev=0 --match 'v[0-9]*.[0-9]*.[0-9]*' \
    --exclude "$target_tag" "$ref" 2>/dev/null) || previous=""
  if [ -n "$previous" ]; then
    git rev-list "$previous..$ref"
  else
    git rev-list "$ref"
  fi
}

# Read paginated REST pull responses on stdin, selecting by release membership,
# deliberately without a milestone filter. Include missing/wrong milestones.
release_pr_numbers() {
  local commits="$1"
  jq -r --arg commits "$commits" '
    ($commits | split("\n") | map(select(length > 0)) | INDEX(.)) as $included
    | add // [] | .[]
    | select(.merged_at != null)
    | select($included[.merge_commit_sha] != null)
    | .number'
}
