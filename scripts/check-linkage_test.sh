#!/usr/bin/env bash
#
# check-linkage_test.sh — tests for scripts/check-linkage.sh.
#
# Hermetic: a stub `gh` sits first on PATH and answers, from fixture files, the only
# calls the script makes (milestone lookup, the milestone's issues and PRs, `pr view`,
# `issue view`), so there is no network and no credentials; REPO is set so the script
# never asks GitHub which repository it is in. What is under test is the milestone rule
# of the gate and the audit, with the exemption for a link to a backlog feature.

# Every `check` condition is a $(... && echo 0 || echo 1) that yields exactly one
# token, so the unquoted substitution cannot split.
# shellcheck disable=SC2046
set -uo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
script="$here/check-linkage.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

command -v jq >/dev/null 2>&1 || { echo "error: 'jq' not found on PATH" >&2; exit 2; }

pass=0; fail=0
ok()  { printf '  ✓ %s\n' "$1"; pass=$((pass + 1)); }
bad() { printf '  ✗ %s\n     %s\n' "$1" "$2"; fail=$((fail + 1)); }
check() { # description, condition-exit-code, detail
  if [ "$2" -eq 0 ]; then ok "$1"; else bad "$1" "$3"; fi
}

# ---- the stub `gh` ---------------------------------------------------------------
# Fixtures live in $FIXTURES: items.ndjson (the milestone's issues and PRs as REST
# returns them), pr-<N>.json and issue-<owner>_<repo>-<N>.json. A call it does not
# know is logged to unexpected.log.
bin="$tmp/bin"; mkdir -p "$bin"
cat > "$bin/gh" <<'STUB'
#!/usr/bin/env bash
fx="${FIXTURES:?}"
unexpected() { echo "gh $*" >> "$fx/unexpected.log"; echo "stub gh: unexpected call: gh $*" >&2; exit 99; }
all="$*"
echo "GH_TOKEN=${GH_TOKEN:-} gh $all" >> "$fx/ghcalls.log"
cmd="${1:-}"; [ $# -gt 0 ] && shift
# a private backlog: only its own token reads it (fixture file backlog-hidden)
backlog_denied() { [ -e "$fx/backlog-hidden" ] && [ "${GH_TOKEN:-}" != "backlog-secret" ]; }
not_found() { echo "gh: Not Found (HTTP 404)" >&2; exit 1; }
expr=""; paginate=0; repo=""; positional=()
while [ $# -gt 0 ]; do
  case "$1" in
    --jq|-q)            shift; expr="${1:-}" ;;
    --paginate|--slurp) paginate=1 ;;
    --repo)             shift; repo="${1:-}" ;;
    --json)             shift ;;
    *)                  positional+=("$1") ;;
  esac
  shift
done
reply() { if [ -n "$expr" ]; then jq -r "$expr" "$1"; else cat "$1"; fi; }
case "$cmd" in
  api)
    path="${positional[0]:-}"
    case "$path" in
      "repos/$REPO/milestones?state=open&per_page=100")
        echo '[{"number":8,"title":"v0.8.0"}]' | jq -r "${expr:-.}" ;;
      "repos/$BACKLOG_REPO")
        backlog_denied && not_found
        echo '{"private":true}' ;;
      "repos/$REPO/milestones/8")
        echo '{"title":"v0.8.0","state":"open"}' ;;
      "repos/$REPO/issues?milestone=8&state=all&per_page=100")
        [ "$paginate" = 1 ] || unexpected "$all"
        jq -sc '[.]' "$fx/items.ndjson" ;;
      *) unexpected "$all" ;;
    esac ;;
  pr)
    [ "${positional[0]:-}" = view ] && [ "$repo" = "$REPO" ] && [ -f "$fx/pr-${positional[1]:-}.json" ] || unexpected "$all"
    reply "$fx/pr-${positional[1]}.json" ;;
  issue)
    [ "$repo" = "$BACKLOG_REPO" ] && backlog_denied && not_found
    f="$fx/issue-${repo//\//_}-${positional[1]:-}.json"
    [ "${positional[0]:-}" = view ] && [ -f "$f" ] || unexpected "$all"
    reply "$f" ;;
  *) unexpected "$all" ;;
esac
STUB
chmod +x "$bin/gh"

# ---- fixtures --------------------------------------------------------------------
REPO=galax-io/test
BACKLOG=galax-io/backlog
new_case() { fx="$tmp/fx-$1"; mkdir -p "$fx"; : > "$fx/items.ndjson"; }
issue_in() { # owner/repo, number, milestone title, [milestone description]
  jq -nc --arg ms "$3" --arg desc "${4:-}" '{milestone: (if $ms == "" then null else {title: $ms, description: $desc} end)}' \
    > "$fx/issue-${1//\//_}-$2.json"
}
add_issue() { # number, state — an issue of this repository in v0.8.0
  printf '{"number":%s,"state":"%s"}\n' "$1" "$2" >> "$fx/items.ndjson"
  issue_in "$REPO" "$1" v0.8.0
}
pr_json() { # number, state, closing link owner/repo#N or "", body, milestone title or ""
  jq -nc --argjson n "$1" --arg state "$2" --arg link "$3" --arg body "$4" --arg ms "$5" --arg author "${PR_AUTHOR:-human}" --arg bot "${PR_BOT:-0}" --arg head "${PR_HEAD:-feat/x}" '{
    number: $n, title: "PR \($n)", state: $state, body: $body,
    author: {login: $author, is_bot: ($bot == "1")}, headRefName: $head,
    milestone: (if $ms == "" then null else {title: $ms} end),
    closingIssuesReferences: [$link | select(. != "")
      | capture("^(?<owner>[^/]+)/(?<name>[^#]+)#(?<n>[0-9]+)$")
      | {number: (.n | tonumber), repository: {name, owner: {login: .owner}}}]
  }' > "$fx/pr-$1.json"
}
add_pr() { # number, state, closing link or "", [body] — a PR in v0.8.0
  printf '{"number":%s,"state":"%s","pull_request":{}}\n' "$1" "$(echo "$2" | tr '[:upper:]' '[:lower:]')" >> "$fx/items.ndjson"
  pr_json "$1" "$2" "$3" "${4:-}" v0.8.0
}
run() { # the script's arguments; sets $out and $rc. The cases below 15 test the other rules, so the
        # requirement of a feature link is off for them; the strict cases use run_strict.
  out="$(PATH="$bin:$PATH" REPO="$REPO" BACKLOG_REPO="$BACKLOG" FEATURE_LINK=optional FIXTURES="$fx" bash "$script" "$@" 2>&1)"; rc=$?
}
run_strict() { # as run, with the requirement of a feature link in force (the default)
  out="$(PATH="$bin:$PATH" REPO="$REPO" BACKLOG_REPO="$BACKLOG" FIXTURES="$fx" bash "$script" "$@" 2>&1)"; rc=$?
}
has() { grep -qF -- "$1" <<<"$out"; }
only_known_calls() { [ ! -s "$fx/unexpected.log" ]; }
BUSINESS='Release: business'

echo "check-linkage.sh"

# 1. gate: the local rule, a PR closes an issue of its own milestone
new_case gate-local
pr_json 3 OPEN "$REPO#1" "Closes #1" v0.8.0
issue_in "$REPO" 1 v0.8.0
run --pr 3
check "gate: closing an issue of the same milestone passes" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"
check "gate: it says the milestone is the same" $(has '✓ closes #1 (same milestone)' && echo 0 || echo 1) "out=$out"

# 2. gate: a human PR without a closing link fails
new_case gate-no-link
pr_json 3 OPEN "" "no link" v0.8.0
run --pr 3
check "gate: a human PR that closes nothing fails" $([ "$rc" -eq 1 ] && echo 0 || echo 1) "rc=$rc out=$out"

# 3. gate: closing an issue of another milestone fails
new_case gate-other-ms
pr_json 3 OPEN "$REPO#1" "Closes #1" v0.8.0
issue_in "$REPO" 1 v0.7.0
run --pr 3
check "gate: closing an issue of another milestone fails" $([ "$rc" -eq 1 ] && echo 0 || echo 1) "rc=$rc out=$out"
check "gate: it names both milestones" $(has '✗ closes #1 but its milestone is "v0.7.0", not "v0.8.0"' && echo 0 || echo 1) "out=$out"

# 4. gate: Part of a backlog feature next to a local closing link passes
new_case gate-feature
pr_json 3 OPEN "$REPO#1" "Closes #1
Part of $BACKLOG#30" v0.8.0
issue_in "$REPO" 1 v0.8.0
issue_in "$BACKLOG" 30 'M1 — first run' "$BUSINESS"
run --pr 3
check "gate: Part of a backlog feature in a business milestone passes" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"
check "gate: it names the business milestone" $(has "✓ links $BACKLOG#30 (feature of business milestone \"M1 — first run\")" && echo 0 || echo 1) "out=$out"
check "gate: the script made only the calls the stub knows" $(only_known_calls && echo 0 || echo 1) "$(cat "$fx/unexpected.log" 2>&1)"

# 5. gate: the marker may sit among other lines of the milestone description
new_case gate-feature-multiline
pr_json 3 OPEN "$REPO#1" "Closes #1
Part of $BACKLOG#30" v0.8.0
issue_in "$REPO" 1 v0.8.0
issue_in "$BACKLOG" 30 'M1 — first run' $'First real run of the process.\nRelease: business\nDue: autumn'
run --pr 3
check "gate: the marker on its own line of a longer description counts" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"

# 6. gate: a backlog issue in an ordinary milestone is not a feature
new_case gate-backlog-plain
pr_json 3 OPEN "$REPO#1" "Closes #1
Part of $BACKLOG#31" v0.8.0
issue_in "$REPO" 1 v0.8.0
issue_in "$BACKLOG" 31 'v0.7.0' 'Tooling of the backlog itself.'
run --pr 3
check "gate: a backlog issue in an ordinary milestone still has to match" $([ "$rc" -eq 1 ] && echo 0 || echo 1) "rc=$rc out=$out"
check "gate: it names both milestones" $(has "✗ links $BACKLOG#31 but its milestone is \"v0.7.0\", not \"v0.8.0\"" && echo 0 || echo 1) "out=$out"

# 7. gate: the marker only counts in the backlog repository
new_case gate-marker-elsewhere
pr_json 3 OPEN "$REPO#1" "Closes #1" v0.8.0
issue_in "$REPO" 1 'M1 — first run' "$BUSINESS"
run --pr 3
check "gate: the marker in another repository does not make a feature" $([ "$rc" -eq 1 ] && echo 0 || echo 1) "rc=$rc out=$out"

# 8. gate: a backlog issue without a milestone is not a feature
new_case gate-feature-no-milestone
pr_json 3 OPEN "$REPO#1" "Closes #1
Part of $BACKLOG#32" v0.8.0
issue_in "$REPO" 1 v0.8.0
issue_in "$BACKLOG" 32 ""
run --pr 3
check "gate: a backlog issue without a milestone fails" $([ "$rc" -eq 1 ] && echo 0 || echo 1) "rc=$rc out=$out"

# 9. gate: the PR itself still needs its version milestone
new_case gate-feature-no-pr-milestone
pr_json 3 OPEN "$REPO#1" "Closes #1
Part of $BACKLOG#30" ""
issue_in "$REPO" 1 v0.8.0
issue_in "$BACKLOG" 30 'M1 — first run' "$BUSINESS"
run --pr 3
check "gate: a feature link does not replace the PR's own milestone" $([ "$rc" -eq 1 ] && echo 0 || echo 1) "rc=$rc out=$out"
check "gate: it says to assign one" $(has '✗ no milestone' && echo 0 || echo 1) "out=$out"
check "gate: the feature link is still reported" $(has "✓ links $BACKLOG#30 (feature of business milestone" && echo 0 || echo 1) "out=$out"

# 10. gate: a closing link straight to a backlog feature is the PR's closing link
new_case gate-closes-feature
pr_json 3 OPEN "$BACKLOG#30" "Closes $BACKLOG#30" v0.8.0
issue_in "$BACKLOG" 30 'M1 — first run' "$BUSINESS"
run --pr 3
check "gate: Closes a backlog feature passes" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"
check "gate: it names the business milestone" $(has "✓ closes $BACKLOG#30 (feature of business milestone" && echo 0 || echo 1) "out=$out"

# 11. audit: a PR of the milestone that closes an issue and is part of a feature is fine
new_case audit-feature
add_issue 1 open
add_pr 2 OPEN "$REPO#1" "Part of $BACKLOG#30"
issue_in "$BACKLOG" 30 'M1 — first run' "$BUSINESS"
run
check "audit: linking a backlog feature passes the audit" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"
check "audit: the PR is listed with its closing link" $(has '✓ PR #2 (OPEN) → closes #1' && echo 0 || echo 1) "out=$out"

# 12. audit: a part of a backlog issue in an ordinary milestone is an error
new_case audit-backlog-plain
add_issue 1 open
add_pr 2 OPEN "$REPO#1" "Part of $BACKLOG#31"
issue_in "$BACKLOG" 31 'v0.7.0' 'Tooling.'
run
check "audit: a backlog issue in another ordinary milestone fails" $([ "$rc" -eq 1 ] && echo 0 || echo 1) "rc=$rc out=$out"

# 13. audit: a PR closing a backlog feature directly is not an error
new_case audit-closes-feature
add_pr 2 OPEN "$BACKLOG#30"
issue_in "$BACKLOG" 30 'M1 — first run' "$BUSINESS"
run
check "audit: closing a backlog feature passes the audit" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"

# 14. audit: the local rule stays, a linked issue of another milestone fails
new_case audit-other-ms
add_issue 1 open
add_pr 2 OPEN "$REPO#4"
issue_in "$REPO" 4 v0.7.0
run
check "audit: a closed issue of another milestone fails" $([ "$rc" -eq 1 ] && echo 0 || echo 1) "rc=$rc out=$out"

# 15. gate: every PR links a feature of the backlog; a PR with no link to one fails
new_case strict-none
pr_json 3 OPEN "$REPO#1" "Closes #1" v0.8.0
issue_in "$REPO" 1 v0.8.0
run_strict --pr 3
check "strict: a PR with no link to a feature fails" $([ "$rc" -eq 1 ] && echo 0 || echo 1) "rc=$rc out=$out"
check "strict: it says which link to add" $(has "✗ links no feature of the backlog" && has "Part of $BACKLOG#" && has 'the issue with the label standing' && echo 0 || echo 1) "out=$out"

# 16. gate: Part of a feature passes
new_case strict-feature
pr_json 3 OPEN "$BACKLOG#30" "Closes $BACKLOG#30" v0.8.0
issue_in "$BACKLOG" 30 'M1 — first run' "$BUSINESS"
run_strict --pr 3
check "strict: a closing link to a feature passes" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"
new_case strict-part
pr_json 3 OPEN "$REPO#1" $'Closes #1\nPart of '"$BACKLOG"'#30' v0.8.0
issue_in "$REPO" 1 v0.8.0
issue_in "$BACKLOG" 30 'M1 — first run' "$BUSINESS"
run_strict --pr 3
check "strict: a closing link to a local issue does not hide the Part of a feature" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"
check "strict: the feature is reported" $(has "✓ links $BACKLOG#30 (feature of business milestone" && echo 0 || echo 1) "out=$out"

# 17. gate: a backlog issue that is not a feature does not satisfy the requirement
new_case strict-not-feature
pr_json 3 OPEN "$REPO#1" $'Closes #1\nPart of '"$BACKLOG"'#31' v0.8.0
issue_in "$REPO" 1 v0.8.0
issue_in "$BACKLOG" 31 'v0.8.0' 'Tooling of the backlog itself.'
run_strict --pr 3
check "strict: Part of a backlog issue in an ordinary milestone is no feature" $([ "$rc" -eq 1 ] && has "✗ links no feature of the backlog" && echo 0 || echo 1) "rc=$rc out=$out"

# 18. gate: a bot's PR is exempt (a bot author, or Dependabot's or Scala Steward's branch)
new_case strict-bot
PR_AUTHOR='app/dependabot' PR_BOT=1 pr_json 3 OPEN "" "Bumps a library." v0.8.0
run_strict --pr 3
check "strict: a bot's PR needs no feature" $([ "$rc" -eq 0 ] && has 'needs no feature' && echo 0 || echo 1) "rc=$rc out=$out"
new_case strict-update-branch
PR_HEAD='update/scalafmt-core-3.9.0' pr_json 3 OPEN "$REPO#1" "Closes #1" v0.8.0
issue_in "$REPO" 1 v0.8.0
run_strict --pr 3
check "strict: an update/ branch is exempt" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"
new_case strict-dependabot-branch
PR_HEAD='dependabot/go_modules/x' pr_json 3 OPEN "$REPO#1" "Closes #1" v0.8.0
issue_in "$REPO" 1 v0.8.0
run_strict --pr 3
check "strict: a dependabot/ branch is exempt" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"
new_case strict-human-update
PR_HEAD='feat/not-a-bot' pr_json 3 OPEN "$REPO#1" "Closes #1" v0.8.0
issue_in "$REPO" 1 v0.8.0
run_strict --pr 3
check "strict: a human's branch is not exempt" $([ "$rc" -eq 1 ] && echo 0 || echo 1) "rc=$rc out=$out"

# 19. gate: the backlog's own PRs link nothing above them
new_case strict-backlog-itself
pr_json 3 OPEN "" "Adds a script." v0.8.0
out="$(PATH="$bin:$PATH" REPO="$BACKLOG" BACKLOG_REPO="$BACKLOG" FIXTURES="$fx" bash "$script" --pr 3 2>&1)"; rc=$?
check "strict: a PR in the backlog repository itself needs no feature" $(! has 'links no feature' && echo 0 || echo 1) "rc=$rc out=$out"

# 20. audit: a PR of the milestone without a feature link is a warning, not an error
new_case strict-audit
add_issue 1 open
add_pr 2 OPEN "$REPO#1"
run_strict
check "strict audit: a PR without a feature is a warning" $([ "$rc" -eq 0 ] && has '! PR #2' && has 'links no feature of the backlog' && echo 0 || echo 1) "rc=$rc out=$out"

# 21. the requirement can be switched off by name only
new_case strict-optional
pr_json 3 OPEN "$REPO#1" "Closes #1" v0.8.0
issue_in "$REPO" 1 v0.8.0
run --pr 3
check "optional: FEATURE_LINK=optional lets a PR without a feature pass" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"
out="$(PATH="$bin:$PATH" REPO="$REPO" BACKLOG_REPO="$BACKLOG" FEATURE_LINK=nonsense FIXTURES="$fx" bash "$script" --pr 3 2>&1)"; rc=$?
check "optional: an unknown FEATURE_LINK value is refused" $([ "$rc" -eq 2 ] && echo 0 || echo 1) "rc=$rc out=$out"

# 22. gate: a private backlog the job's token cannot read is said so, not mistaken for "no feature"
new_case strict-hidden
pr_json 3 OPEN "$REPO#1" $'Closes #1\nPart of '"$BACKLOG"'#30' v0.8.0
issue_in "$REPO" 1 v0.8.0
issue_in "$BACKLOG" 30 'M1 — first run' "$BUSINESS"
: > "$fx/backlog-hidden"
run_strict --pr 3
check "hidden: a backlog the token cannot read fails the gate" $([ "$rc" -eq 1 ] && echo 0 || echo 1) "rc=$rc out=$out"
check "hidden: it names the missing token" $(has "cannot read the backlog $BACKLOG" && has 'BACKLOG_READ_TOKEN' && echo 0 || echo 1) "out=$out"
check "hidden: it does not claim the PR links no feature" $(! has 'links no feature of the backlog' && echo 0 || echo 1) "out=$out"
check "hidden: it does not call the milestone of the feature \"none\"" $(! has 'its milestone is "none"' && ! has 'but its milestone is "none"' && echo 0 || echo 1) "out=$out"

# 23. gate: with BACKLOG_READ_TOKEN the backlog is read with that token, and only the backlog
new_case strict-token
pr_json 3 OPEN "$REPO#1" $'Closes #1\nPart of '"$BACKLOG"'#30' v0.8.0
issue_in "$REPO" 1 v0.8.0
issue_in "$BACKLOG" 30 'M1 — first run' "$BUSINESS"
: > "$fx/backlog-hidden"
out="$(PATH="$bin:$PATH" REPO="$REPO" BACKLOG_REPO="$BACKLOG" GH_TOKEN=job-token BACKLOG_READ_TOKEN=backlog-secret FIXTURES="$fx" bash "$script" --pr 3 2>&1)"; rc=$?
check "token: with BACKLOG_READ_TOKEN the feature is seen and the gate passes" $([ "$rc" -eq 0 ] && has "✓ links $BACKLOG#30 (feature of business milestone" && echo 0 || echo 1) "rc=$rc out=$out"
check "token: the backlog token is used for calls about the backlog" $(grep -q "GH_TOKEN=backlog-secret gh .*$BACKLOG" "$fx/ghcalls.log" && echo 0 || echo 1) "$(cat "$fx/ghcalls.log")"
check "token: the backlog token never reaches a call about this repository" $(! grep "GH_TOKEN=backlog-secret" "$fx/ghcalls.log" | grep -v "$BACKLOG" | grep -q . && echo 0 || echo 1) "$(grep backlog-secret "$fx/ghcalls.log")"
check "token: the job's own token still reads the pull request" $(grep -q "GH_TOKEN=job-token gh pr view" "$fx/ghcalls.log" && echo 0 || echo 1) "$(cat "$fx/ghcalls.log")"

# 24. gate: an unset BACKLOG_READ_TOKEN changes nothing for a readable backlog
new_case strict-readable
pr_json 3 OPEN "$REPO#1" $'Closes #1\nPart of '"$BACKLOG"'#30' v0.8.0
issue_in "$REPO" 1 v0.8.0
issue_in "$BACKLOG" 30 'M1 — first run' "$BUSINESS"
run_strict --pr 3
check "readable: without BACKLOG_READ_TOKEN a readable backlog is read with the job's token" $([ "$rc" -eq 0 ] && echo 0 || echo 1) "rc=$rc out=$out"

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
