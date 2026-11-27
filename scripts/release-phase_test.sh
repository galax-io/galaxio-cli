#!/usr/bin/env bash
#
# release-phase_test.sh — tests for scripts/release-phase.sh (BCKLG-1038, trunk-based), the logic of
# the phases `prepare` and `publish` of .github/workflows/release-build.yml.
#
# Hermetic: a bare repository stands for `origin`, a clone for the runner's checkout, and a stub `gh` for
# the GitHub Release. Real `git` runs. What is under test is the promise of the phases: all work is on
# `main`; `prepare` of vX.Y.0 cuts release/X.Y.0 from `main`, of a patch vX.Y.Z needs that branch, never
# moves a branch, checks its head and marks it green; `publish` tags only that green head, makes the
# GitHub Release, keeps the branch and never touches `main`; a second run makes nothing new; a patch is a
# fix of `main` cherry-picked onto the branch.
#
# The script is run in the clone as: release-phase.sh <phase> --release <id> --version vX.Y.Z [--result <file>];
# the checks of the repository are the command in $RELEASE_CHECKS.

set -uo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
script="$here/release-phase.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

pass=0; fail=0
ok()  { printf '  ✓ %s\n' "$1"; pass=$((pass + 1)); }
bad() { printf '  ✗ %s\n     %s\n' "$1" "$2"; fail=$((fail + 1)); }
check() { # a check passes only when the script exists: a negative assertion proves nothing without it
  if [ -f "$script" ] && [ "$2" -eq 0 ]; then ok "$1"; else bad "$1" "${3:-}"; fi
}

export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@example.test GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@example.test
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null

bin="$tmp/bin"; mkdir -p "$bin"
printf '#!/usr/bin/env bash\necho "gh $*" >> "$GH_LOG"\n' > "$bin/gh"; chmod +x "$bin/gh"

n=0
new_repo() { # sets $origin and $work: main has one commit, file a.txt
  n=$((n + 1)); origin="$tmp/origin$n.git"; work="$tmp/work$n"; export GH_LOG="$tmp/gh$n.log"; : > "$GH_LOG"
  git init -q --bare -b main "$origin"
  git clone -q "$origin" "$work" 2>/dev/null
  ( cd "$work" && git checkout -q -b main && printf 'one\n' > a.txt && git add a.txt && git commit -q -m "chore: init" && git push -q origin main )
}
g() { git -C "$work" "$@"; }
commit_on() { # branch, file, content, subject — a commit on a branch of origin, through the clone
  ( cd "$work" && git fetch -q origin && { git checkout -q "$1" 2>/dev/null || git checkout -q -b "$1" "origin/$1"; } && git reset -q --hard "origin/$1" && printf '%s\n' "$3" > "$2" && git add "$2" && git commit -q -m "$4" && git push -q origin "$1" )
}
tag_main() { ( cd "$work" && git checkout -q main && git pull -q origin main && git tag "$1" && git push -q origin "$1" ); }
main_commit() { ( cd "$work" && git checkout -q main && git pull -q origin main && printf '%s\n' "$2" > "$1" && git add "$1" && git commit -q -m "$3" && git push -q origin main ); }
phase() { # phase, then the arguments; sets $out and $rc; the checks are $CHECKS (default: true)
  local p="$1"; shift
  out="$( cd "$work" && git checkout -q main 2>/dev/null; PATH="$bin:$PATH" RELEASE_CHECKS="${CHECKS:-true}" bash "$script" "$p" --result "$work/../result$n.json" "$@" 2>&1 )"; rc=$?
}
result() { jq -r "$1" "$tmp/result$n.json" 2>/dev/null; }
ref() { git --git-dir="$origin" rev-parse -q --verify "$1" 2>/dev/null || echo none; }
has_ref() { git --git-dir="$origin" rev-parse -q --verify "$1" >/dev/null 2>&1; }
log_of() { git --git-dir="$origin" log --format=%s "$1" 2>/dev/null; }
CHECKS=true

echo "release-phase.sh"

# the input is checked before anything is written (release ^[a-z0-9.]+$, version ^v[0-9]+\.[0-9]+\.[0-9]+$)
new_repo
phase prepare --release 'M1 x' --version v1.3.0
check "a release id with capitals and a space is refused, nothing is made" "$([ "$rc" -ne 0 ] && [ "$(git --git-dir="$origin" branch --list 'release/*' | wc -l | tr -d ' ')" -eq 0 ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase prepare --release 'm1;touch-x' --version v1.3.0
check "a release id with a shell character is refused" "$([ "$rc" -ne 0 ] && [ ! -e "$work/touch-x" ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase prepare --release m1
check "prepare without a version is refused" "$([ "$rc" -eq 2 ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase prepare --release m1 --version 'v1.3.0 x'
check "a version with a space is refused" "$([ "$rc" -eq 2 ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase prepare --release m1 --version v1.3
check "a version that is not vX.Y.Z is refused" "$([ "$rc" -eq 2 ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase open --release m1 --version v1.3.0
check "FR-005: there is no phase open" "$([ "$rc" -eq 2 ] && [ "$(git --git-dir="$origin" branch --list 'release/*' | wc -l | tr -d ' ')" -eq 0 ] && echo 0 || echo 1)" "rc=$rc out=$out"

# FR-002, FR-003 — prepare of vX.Y.0 cuts release/X.Y.0 from main, checks it, marks it; no tag, main unchanged
new_repo
tag_main v1.2.0
main_commit b.txt b "feat: b (#31)"; main_commit c.txt c "feat: c (#32)"
main_before="$(ref refs/heads/main)"
phase prepare --release m1 --version v1.3.0
check "FR-002: prepare cuts release/1.3.0 on origin" "$([ "$rc" -eq 0 ] && has_ref refs/heads/release/1.3.0 && echo 0 || echo 1)" "rc=$rc out=$out"
check "FR-002: the branch starts at main" "$([ "$(ref refs/heads/release/1.3.0)" = "$main_before" ] && echo 0 || echo 1)" ""
check "FR-003: the result is green and names the PRs after the last tag" "$([ "$(result .status)" = green ] && [ "$(result '.included | sort | join(",")')" = "31,32" ] && [ "$(result .branch)" = release/1.3.0 ] && echo 0 || echo 1)" "$(cat "$tmp/result$n.json" 2>&1)"
check "FR-003: prepare sets no tag and does not move main" "$([ "$(git --git-dir="$origin" tag | tr '\n' ' ')" = "v1.2.0 " ] && [ "$(ref refs/heads/main)" = "$main_before" ] && echo 0 || echo 1)" "$(git --git-dir="$origin" tag)"

# FR-002 — an existing branch is never moved: main went on after the cut
main_commit d.txt d "feat: d (#33)"
cut="$(ref refs/heads/release/1.3.0)"
phase prepare --release m1 --version v1.3.0
check "FR-002: a second prepare keeps the branch where it was cut" "$([ "$rc" -eq 0 ] && [ "$(ref refs/heads/release/1.3.0)" = "$cut" ] && echo 0 || echo 1)" "rc=$rc out=$out"

# FR-003 — red checks: the result is red, nothing is marked
new_repo
CHECKS='echo "FAILED unit" >&2; exit 1' phase prepare --release m1 --version v1.3.0
check "FR-003: red checks stop prepare" "$([ "$rc" -ne 0 ] && [ "$(result .status)" = red ] && ! has_ref refs/release-prepared/1.3.0 && echo 0 || echo 1)" "rc=$rc $(cat "$tmp/result$n.json" 2>&1)"

# FR-002 — a tagged version and a patch without its branch are refused
new_repo
tag_main v1.3.0
phase prepare --release m1 --version v1.3.0
check "FR-002: prepare of a version that is tagged already is refused, no branch is made" "$([ "$rc" -ne 0 ] && ! has_ref refs/heads/release/1.3.0 && echo 0 || echo 1)" "rc=$rc out=$out"
phase prepare --release m1 --version v1.4.1
check "FR-002: a patch whose release branch does not exist is refused, no branch is made" "$([ "$rc" -ne 0 ] && ! has_ref refs/heads/release/1.4.0 && echo 0 || echo 1)" "rc=$rc out=$out"

# FR-004 — publish only after a green prepare of the same head
new_repo
phase publish --release m1 --version v1.3.0
check "FR-004: publish with no prepare is refused, no tag" "$([ "$rc" -ne 0 ] && [ -z "$(git --git-dir="$origin" tag)" ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase prepare --release m1 --version v1.3.0
commit_on release/1.3.0 e.txt e "fix: e (#34)"
phase publish --release m1 --version v1.3.0
check "FR-004: publish after the head moved past prepare is refused" "$([ "$rc" -ne 0 ] && [ -z "$(git --git-dir="$origin" tag)" ] && echo 0 || echo 1)" "rc=$rc out=$out"

# FR-004, SC-001 — publish tags the green head of the branch; main is not touched, the branch stays
new_repo
tag_main v1.2.0; main_commit b.txt b "feat: b (#31)"
phase prepare --release m1 --version v1.3.0
head="$(ref refs/heads/release/1.3.0)"
main_commit bot.txt bot "chore: bump (#40)"
main_before="$(ref refs/heads/main)"
phase publish --release m1 --version v1.3.0
check "FR-004: publish succeeds after a green prepare, though main moved on" "$([ "$rc" -eq 0 ] && [ "$(result .status)" = published ] && echo 0 || echo 1)" "rc=$rc out=$out"
check "SC-001: the tag stands on the head prepare checked" "$([ "$(ref 'refs/tags/v1.3.0^{commit}')" = "$head" ] && echo 0 || echo 1)" ""
check "SC-001: main is not touched by publish" "$([ "$(ref refs/heads/main)" = "$main_before" ] && echo 0 || echo 1)" ""
check "FR-004: the release branch stays, the mark is gone" "$(has_ref refs/heads/release/1.3.0 && ! has_ref refs/release-prepared/1.3.0 && echo 0 || echo 1)" ""
check "FR-004: a GitHub Release is made for the tag" "$(grep -q 'release create v1.3.0' "$GH_LOG" && echo 0 || echo 1)" "$(cat "$GH_LOG")"
releases_before="$(grep -c 'release create' "$GH_LOG")"
phase publish --release m1 --version v1.3.0
check "FR-004: a second publish changes no tag and makes no second release" "$([ "$rc" -eq 0 ] && [ "$(ref 'refs/tags/v1.3.0^{commit}')" = "$head" ] && [ "$(grep -c 'release create' "$GH_LOG")" = "$releases_before" ] && echo 0 || echo 1)" "rc=$rc out=$out"

# US3, SC-002 — a patch: the fix lands on main, is cherry-picked onto the branch and is tagged there
main_commit fix.txt fix "fix: crash (#60)"; fix="$(ref refs/heads/main)"
main_commit f.txt f "feat: next (#61)"
( cd "$work" && git fetch -q origin && git checkout -q -B release/1.3.0 origin/release/1.3.0 && git cherry-pick -x "$fix" >/dev/null && git push -q origin release/1.3.0 )
main_before="$(ref refs/heads/main)"
phase prepare --release m1 --version v1.3.1
check "US3: prepare of the patch checks the branch and names the cherry-picked PR only" "$([ "$rc" -eq 0 ] && [ "$(result '.included | join(",")')" = "60" ] && echo 0 || echo 1)" "rc=$rc $(cat "$tmp/result$n.json" 2>&1)"
phase publish --release m1 --version v1.3.1
check "SC-002: the patch is tagged on the release branch" "$([ "$rc" -eq 0 ] && [ "$(ref 'refs/tags/v1.3.1^{commit}')" = "$(ref refs/heads/release/1.3.0)" ] && echo 0 || echo 1)" "rc=$rc out=$out"
check "SC-002: main gets no commit of the branch and is not touched" "$([ "$(ref refs/heads/main)" = "$main_before" ] && ! git --git-dir="$origin" merge-base --is-ancestor refs/heads/release/1.3.0 refs/heads/main && echo 0 || echo 1)" ""
check "SC-002: the feature after the fix is not in the patch" "$(! git --git-dir="$origin" log --format=%s refs/tags/v1.3.1 | grep -qF 'feat: next (#61)' && echo 0 || echo 1)" ""

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
