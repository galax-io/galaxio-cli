#!/usr/bin/env bash
# Local regression checks; uses real Git and jq, with no GitHub writes or mocks.
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
failures=0
check_hook() {
  local expected="$1" cmd="$2" rc=0
  jq -cn --arg cmd "$cmd" '{tool_input:{command:$cmd}}' |
    CLAUDE_PROJECT_DIR="$tmp" bash "$root/.claude/hooks/linkage-guard.sh" >"$tmp/out" 2>&1 || rc=$?
  if [ "$rc" != "$expected" ]; then
    printf 'FAIL hook (%s != %s): %s\n' "$rc" "$expected" "$cmd"
    failures=$((failures + 1))
  fi
}
# A release operation must reach the missing-checker guard, including in a compound command.
check_hook 2 'git tag v1.2.0'
check_hook 2 'git log -1 && git tag v1.2.0'
check_hook 2 'git show HEAD && git push origin v1.2.0'
check_hook 2 'git commit -m fix && git push origin v1.2.0'
check_hook 0 'git log -1'
check_hook 0 'git show v1.2.0'
check_hook 0 'git push origin chore/speckit-bootstrap'

if [ -f "$root/scripts/lib/linkage.sh" ]; then
  source "$root/scripts/lib/linkage.sh"
  is_dependabot_author app/dependabot \
    || { echo 'FAIL GraphQL Dependabot identity rejected'; failures=$((failures + 1)); }
  is_dependabot_author 'dependabot[bot]' \
    || { echo 'FAIL webhook Dependabot identity rejected'; failures=$((failures + 1)); }
  if is_dependabot_author dependabot || is_dependabot_author app/renovate; then
    echo 'FAIL non-Dependabot identity accepted'
    failures=$((failures + 1))
  fi
  git -C "$tmp" init -q -b review-fixture
  git -C "$tmp" -c user.name=Review -c user.email=review@example.invalid commit -qm initial --allow-empty
  git -C "$tmp" tag v1.0.0
  old=$(git -C "$tmp" rev-parse HEAD)
  git -C "$tmp" -c user.name=Review -c user.email=review@example.invalid commit -qm change --allow-empty
  current=$(git -C "$tmp" rev-parse HEAD)
  git -C "$tmp" tag v1.1.0
  commits=$(cd "$tmp" && release_commits HEAD v1.1.0)
  [ "$commits" = "$current" ] || { echo 'FAIL release range includes previous release'; failures=$((failures + 1)); }
  commits=$(cd "$tmp" && release_commits HEAD v1.2.0)
  [ -z "$commits" ] || { echo 'FAIL unreleased range at latest tag'; failures=$((failures + 1)); }
  commits=$(cd "$tmp" && release_commits v1.0.0 v1.0.0)
  [ "$commits" = "$old" ] || { echo 'FAIL first release range'; failures=$((failures + 1)); }
  commits="$current"
  empty=$(printf '[]' | release_pr_numbers "$commits")
  [ -z "$empty" ] || { echo 'FAIL empty PR response'; failures=$((failures + 1)); }
  git clone -q --depth=1 "file://$tmp" "$tmp/shallow"
  rc=0
  (cd "$tmp/shallow" && release_commits HEAD v1.1.0) >"$tmp/out" 2>&1 || rc=$?
  [ "$rc" = 2 ] || { echo 'FAIL shallow history accepted'; failures=$((failures + 1)); }
  # GitHub response data is input to a pure selection function, not a mocked API.
  prs=$(jq -cn --arg old "$old" --arg current "$current" '[[
    {number:1, merged_at:"date", merge_commit_sha:$old, milestone:{number:1}},
    {number:2, merged_at:"date", merge_commit_sha:$current, milestone:null},
    {number:3, merged_at:"date", merge_commit_sha:$current, milestone:{number:99}},
    {number:4, merged_at:null, merge_commit_sha:$current}
  ]]' | release_pr_numbers "$commits")
  [ "$prs" = $'2\n3' ] || { echo 'FAIL missing/wrong milestone PRs excluded'; failures=$((failures + 1)); }
else
  echo 'FAIL release range helpers missing'
  failures=$((failures + 1))
fi
[ "$failures" = 0 ] || exit 1
echo 'PASS linkage regression checks'
