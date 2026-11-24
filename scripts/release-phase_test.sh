#!/usr/bin/env bash
#
# release-phase_test.sh — tests for scripts/release-phase.sh (BCKLG-1030), the logic of the
# phases `open`, `prepare` and `publish` of .github/workflows/release-build.yml.
#
# Hermetic: a bare repository stands for `origin`, a clone for the runner's checkout, and a stub `gh` for
# the GitHub Release. Real `git rebase`, `git merge --ff-only` and `git push --force-with-lease` run.
# What is under test is the promise of the phases: the branch of a release is the repository's version,
# release/X.Y.0 for vX.Y.0, made from `main`;
# `prepare` rebases it onto `main`, checks it and says what it holds, and on a conflict changes
# nothing; `publish` fast-forwards `main` to the branch only after a green `prepare` of that very
# head, tags `main`, deletes the branch, and a second run makes nothing new.
#
# The script is run in the clone as: release-phase.sh <phase> --release <id> --version vX.Y.0 [--result <file>];
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

# FR-002 — open: the branch of the release is made from main; a second run changes nothing
new_repo
phase open --release m1 --version v1.3.0
check "FR-002: open makes release/1.3.0, the version, on origin" "$([ "$rc" -eq 0 ] && has_ref refs/heads/release/1.3.0 && echo 0 || echo 1)" "rc=$rc out=$out"
check "FR-002: the branch starts at main" "$([ "$(ref refs/heads/release/1.3.0)" = "$(ref refs/heads/main)" ] && echo 0 || echo 1)" ""
commit_on release/1.3.0 b.txt b "feat: b (#31)"
before="$(ref refs/heads/release/1.3.0)"
phase open --release m1 --version v1.3.0
check "FR-002: a second open changes nothing" "$([ "$rc" -eq 0 ] && [ "$(ref refs/heads/release/1.3.0)" = "$before" ] && echo 0 || echo 1)" "rc=$rc out=$out"

# the input is checked before anything is written (release ^[a-z0-9.]+$, version ^v[0-9]+\.[0-9]+\.0$)
new_repo
phase open --release 'M1 x' --version v1.3.0
check "FR-002: a release id with capitals and a space is refused, nothing is made" "$([ "$rc" -ne 0 ] && [ "$(git --git-dir="$origin" branch --list 'release/*' | wc -l | tr -d ' ')" -eq 0 ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase open --release 'm1;touch-x' --version v1.3.0
check "FR-002: a release id with a shell character is refused" "$([ "$rc" -ne 0 ] && [ ! -e "$work/touch-x" ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase open --release m1 --version v1.3.0; commit_on release/1.3.0 b.txt b "feat: b (#31)"
phase prepare --release m1 --version v1.3.1
check "FR-005: a version that is not vX.Y.0 is refused" "$([ "$rc" -ne 0 ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase prepare --release m1 --version 'v1.3.0 x'
check "FR-005: a version with a space is refused" "$([ "$rc" -ne 0 ] && echo 0 || echo 1)" "rc=$rc out=$out"

# the branch is named by the version: no phase runs without one, and open refuses a version already tagged
new_repo
phase open --release m1
check "FR-002: open without a version is refused, nothing is made" "$([ "$rc" -eq 2 ] && [ "$(git --git-dir="$origin" branch --list 'release/*' | wc -l | tr -d ' ')" -eq 0 ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase prepare --release m1
check "FR-005: prepare without a version is refused" "$([ "$rc" -eq 2 ] && echo 0 || echo 1)" "rc=$rc out=$out"
git -C "$work" tag v1.3.0 && git -C "$work" push -q origin v1.3.0
phase open --release m1 --version v1.3.0
check "FR-002: open of a version that is tagged already is refused, no branch is made" "$([ "$rc" -ne 0 ] && ! has_ref refs/heads/release/1.3.0 && echo 0 || echo 1)" "rc=$rc out=$out"
phase open --release m2 --version v1.4.0
check "FR-002: the branch of each release is its own version" "$([ "$rc" -eq 0 ] && has_ref refs/heads/release/1.4.0 && [ "$(result .branch)" = release/1.4.0 ] && echo 0 || echo 1)" "rc=$rc out=$out"

# FR-005 — prepare: the branch is rebased onto main, checked, and the result says what it holds; no tag
new_repo
phase open --release m1 --version v1.3.0
commit_on release/1.3.0 b.txt b "feat: b (#31)"; commit_on release/1.3.0 c.txt c "feat: c (#32)"
main_commit bot.txt bot "chore: bump (#40)"
phase prepare --release m1 --version v1.3.0
check "FR-005: prepare of a branch behind main succeeds" "$([ "$rc" -eq 0 ] && echo 0 || echo 1)" "rc=$rc out=$out"
check "FR-005: main is an ancestor of the branch after prepare" "$(git --git-dir="$origin" merge-base --is-ancestor refs/heads/main refs/heads/release/1.3.0 && echo 0 || echo 1)" ""
check "FR-005: the branch keeps its own commits on top of main" "$([ "$(log_of release/1.3.0 | head -3 | tr '\n' '|')" = 'feat: c (#32)|feat: b (#31)|chore: bump (#40)|' ] && echo 0 || echo 1)" "$(log_of release/1.3.0)"
check "FR-005: the result is green and names the PRs of the branch" "$([ "$(result .status)" = green ] && [ "$(result '.included | sort | join(",")')" = "31,32" ] && [ "$(result .version)" = v1.3.0 ] && echo 0 || echo 1)" "$(cat "$tmp/result$n.json" 2>&1)"
check "FR-005: prepare sets no tag" "$([ -z "$(git --git-dir="$origin" tag)" ] && echo 0 || echo 1)" ""
check "FR-005: prepare does not move main" "$([ "$(log_of main | head -1)" = "chore: bump (#40)" ] && echo 0 || echo 1)" ""

# FR-005 — a conflict of the rebase: the branch is not changed, the files are named
new_repo
phase open --release m1 --version v1.3.0
commit_on release/1.3.0 a.txt branch-side "feat: a (#31)"
main_commit a.txt main-side "fix: a (#41)"
before="$(ref refs/heads/release/1.3.0)"
phase prepare --release m1 --version v1.3.0
check "FR-005: a conflicting rebase stops prepare" "$([ "$rc" -ne 0 ] && echo 0 || echo 1)" "rc=$rc out=$out"
check "FR-005: the branch on origin is not changed by a conflict" "$([ "$(ref refs/heads/release/1.3.0)" = "$before" ] && echo 0 || echo 1)" ""
check "FR-005: the result is a conflict that names the file" "$([ "$(result .status)" = conflict ] && [ "$(result '.conflict | join(",")')" = a.txt ] && echo 0 || echo 1)" "$(cat "$tmp/result$n.json" 2>&1)"

# FR-005 — red checks of the branch: the result is red, with the failed check named
new_repo
phase open --release m1 --version v1.3.0; commit_on release/1.3.0 b.txt b "feat: b (#31)"
CHECKS='echo "FAILED unit" >&2; exit 1' phase prepare --release m1 --version v1.3.0
check "FR-005: red checks stop prepare" "$([ "$rc" -ne 0 ] && [ "$(result .status)" = red ] && echo 0 || echo 1)" "rc=$rc $(cat "$tmp/result$n.json" 2>&1)"

# FR-005 — publish only after a green prepare of the same head
new_repo
phase open --release m1 --version v1.3.0; commit_on release/1.3.0 b.txt b "feat: b (#31)"
main_before="$(ref refs/heads/main)"
phase publish --release m1 --version v1.3.0
check "FR-005: publish with no prepare is refused, main and tags are untouched" "$([ "$rc" -ne 0 ] && [ "$(ref refs/heads/main)" = "$main_before" ] && [ -z "$(git --git-dir="$origin" tag)" ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase prepare --release m1 --version v1.3.0
commit_on release/1.3.0 d.txt d "feat: d (#33)"
phase publish --release m1 --version v1.3.0
check "FR-005: publish after the head moved past prepare is refused" "$([ "$rc" -ne 0 ] && [ "$(ref refs/heads/main)" = "$main_before" ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase prepare --release m1 --version v1.3.0
main_commit bot.txt bot "chore: bump (#40)"
phase publish --release m1 --version v1.3.0
check "FR-005: publish when main moved on after prepare is refused" "$([ "$rc" -ne 0 ] && [ -z "$(git --git-dir="$origin" tag)" ] && echo 0 || echo 1)" "rc=$rc out=$out"

# US1, FR-005 — only the release's own content reaches main: two branches, the first published
new_repo
phase open --release r1 --version v1.3.0; phase open --release r2 --version v1.4.0
commit_on release/1.3.0 one.txt 1 "feat: one (#31)"; commit_on release/1.4.0 two.txt 2 "feat: two (#51)"
phase prepare --release r1 --version v1.3.0
phase publish --release r1 --version v1.3.0
check "FR-005: publish succeeds after a green prepare" "$([ "$rc" -eq 0 ] && [ "$(result .status)" = published ] && echo 0 || echo 1)" "rc=$rc out=$out"
# The log is read once: grep -q stops at its match, and under pipefail the log's SIGPIPE would fail a pipe into it.
main_log="$(log_of main)"
check "SC-001: main holds the PR of r1 and not the PR of r2" "$(grep -qF 'feat: one (#31)' <<<"$main_log" && ! grep -qF 'feat: two (#51)' <<<"$main_log" && echo 0 || echo 1)" "$main_log"
check "FR-005: main is the branch's head, fast-forwarded without a merge commit" "$([ "$(log_of main | head -1)" = "feat: one (#31)" ] && [ -z "$(git --git-dir="$origin" log --merges --format=%h main)" ] && echo 0 || echo 1)" ""
check "FR-005: the tag stands on main" "$([ "$(ref 'refs/tags/v1.3.0^{commit}')" = "$(ref refs/heads/main)" ] && echo 0 || echo 1)" "$(git --git-dir="$origin" tag)"
check "FR-005: the release branch is deleted, the other release's branch stays" "$(! has_ref refs/heads/release/1.3.0 && has_ref refs/heads/release/1.4.0 && echo 0 || echo 1)" ""
check "FR-005: a GitHub Release is made for the tag" "$(grep -q 'release create v1.3.0' "$GH_LOG" && echo 0 || echo 1)" "$(cat "$GH_LOG")"
tag_before="$(ref refs/tags/v1.3.0)"; main_after="$(ref refs/heads/main)"; releases_before="$(grep -c 'release create' "$GH_LOG")"
phase publish --release r1 --version v1.3.0
check "SC-006: a second publish changes no tag, no main and makes no second release" "$([ "$(ref refs/tags/v1.3.0)" = "$tag_before" ] && [ "$(ref refs/heads/main)" = "$main_after" ] && [ "$(grep -c 'release create' "$GH_LOG")" = "$releases_before" ] && echo 0 || echo 1)" "rc=$rc out=$out"
phase prepare --release r2 --version v1.4.0
check "US2: the next release's prepare rebases its branch onto the published main" "$(git --git-dir="$origin" merge-base --is-ancestor refs/heads/main refs/heads/release/1.4.0 && echo 0 || echo 1)" "rc=$rc out=$out"

# FR-008 — an urgent release: publish with no branch tags the current main
new_repo
main_commit fix.txt fix "fix: urgent (#60)"
main_before="$(ref refs/heads/main)"
phase publish --release m1.1 --version v1.4.0
check "FR-008: publish with no release branch tags main" "$([ "$rc" -eq 0 ] && [ "$(ref 'refs/tags/v1.4.0^{commit}')" = "$main_before" ] && [ "$(ref refs/heads/main)" = "$main_before" ] && echo 0 || echo 1)" "rc=$rc out=$out"
check "FR-008: the urgent publish makes a GitHub Release" "$(grep -q 'release create v1.4.0' "$GH_LOG" && echo 0 || echo 1)" "$(cat "$GH_LOG")"

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
