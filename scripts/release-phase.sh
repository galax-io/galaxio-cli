#!/usr/bin/env bash
#
# release-phase.sh — the phases `open`, `prepare` and `publish` of .github/workflows/release-build.yml
# (BCKLG-1030). A release of the backlog has one branch in the repository, named by the repository's
# version in it: release/X.Y.0 for vX.Y.0. The branch is the content of that version.
#
#   open     the branch release/X.Y.0 from origin/main; an existing branch is left as it is; a version
#            that is tagged already is refused.
#   prepare  the branch is rebased onto origin/main (pushed with --force-with-lease), the checks of the
#            repository run on it, the result says what it holds. A conflict changes nothing. No tag.
#            A green prepare marks the head it checked in refs/release-prepared/X.Y.0.
#   publish  only when that mark is the head of the branch and origin/main is an ancestor of it:
#            main is fast-forwarded to the branch, the version is tagged on main, a GitHub Release is
#            made, the branch and the mark are deleted. With no branch (an urgent release) the version
#            is tagged on the current main. A version that is already tagged on main changes nothing.
#
# Usage: release-phase.sh <open|prepare|publish> --release <id> --version vX.Y.0 [--result <file>]
#
# --release  ^[a-z0-9.]+$, the id of the backlog's release (for the result)
# --version  ^v[0-9]+\.[0-9]+\.0$, required by every phase: it names the branch
# --result   where the JSON result is written (default release-result.json)
# RELEASE_CHECKS  the checks to run on the branch; default: the `# lint` and `# test` lines of the
#                 Commands block of AGENTS.md.
#
# The input is checked before anything is written. Only the maintainer starts the workflow that runs it.
# Exit: 0 done | 1 the phase did not complete (conflict, red checks, refused) | 2 wrong usage.

set -uo pipefail

phase="${1:-}"; [ $# -gt 0 ] && shift
release=""; version=""; result="release-result.json"
while [ $# -gt 0 ]; do
  case "$1" in
    --release) release="${2:-}"; shift 2 ;;
    --version) version="${2:-}"; shift 2 ;;
    --result) result="${2:-}"; shift 2 ;;
    *) echo "release-phase: unknown argument '$1'" >&2; exit 2 ;;
  esac
done

case "$phase" in open|prepare|publish) ;; *) echo "release-phase: the phase is open, prepare or publish" >&2; exit 2 ;; esac
[[ "$release" =~ ^[a-z0-9.]+$ ]] || { echo "release-phase: --release must match ^[a-z0-9.]+\$" >&2; exit 2; }
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.0$ ]]; then
  echo "release-phase: --version vX.Y.0 is required; it names the branch release/X.Y.0" >&2; exit 2
fi

branch="release/${version#v}"
mark="refs/release-prepared/${version#v}"
repo="${GITHUB_REPOSITORY:-$(basename "$(git remote get-url origin 2>/dev/null || echo unknown)" .git)}"

included='[]'; ci='null'; conflict='[]'; tag='null'

write_result() { # status
  jq -n --arg repo "$repo" --arg release "$release" --arg version "$version" --arg branch "$branch" \
    --arg phase "$phase" --arg status "$1" --argjson included "$included" --argjson ci "$ci" \
    --argjson conflict "$conflict" --argjson tag "$tag" \
    '{repo:$repo, release:$release, version:(if $version == "" then null else $version end), branch:$branch,
      phase:$phase, status:$status, included:$included, ci:$ci, conflict:$conflict, tag:$tag}' > "$result"
}
refuse() { echo "release-phase: $*" >&2; exit 1; }

git config user.name >/dev/null 2>&1 || git config user.name "github-actions[bot]"
git config user.email >/dev/null 2>&1 || git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

git fetch -q --prune --tags origin '+refs/heads/*:refs/remotes/origin/*' || refuse "cannot fetch origin"
git rev-parse -q --verify refs/remotes/origin/main >/dev/null || refuse "origin has no main"
remote_sha() { git ls-remote origin "$1" | awk 'NR == 1 { print $1 }'; }
has_branch() { git rev-parse -q --verify "refs/remotes/origin/$branch" >/dev/null; }
pr_numbers() { # the PRs of the branch: (#N) of the subjects of the commits main does not have
  git log --format=%s "origin/main..$1" | grep -o '(#[0-9]*)' | tr -d '(#)' | sort -un | jq -R 'tonumber' | jq -s .
}

case "$phase" in
  open)
    [ -z "$(remote_sha "refs/tags/$version")" ] || refuse "$version is tagged already; open the next version"
    if has_branch; then
      echo "release-phase: $branch exists, nothing to do"
    else
      git push -q origin "refs/remotes/origin/main:refs/heads/$branch" || refuse "cannot make $branch"
    fi
    write_result green
    ;;

  prepare)
    has_branch || refuse "$branch does not exist; run open first"
    old="$(git rev-parse "refs/remotes/origin/$branch")"
    git push -q origin ":$mark" 2>/dev/null || true
    git checkout -q -B "$branch" "refs/remotes/origin/$branch" || refuse "cannot check out $branch"
    if ! git rebase -q origin/main >/dev/null 2>&1; then
      conflict="$(git diff --name-only --diff-filter=U | jq -R . | jq -s .)"
      git rebase --abort >/dev/null 2>&1
      write_result conflict
      echo "release-phase: rebase of $branch onto main conflicts in: $(jq -r 'join(", ")' <<< "$conflict")" >&2
      exit 1
    fi
    if [ "$(git rev-parse HEAD)" != "$old" ]; then
      git push -q --force-with-lease="refs/heads/$branch:$old" origin "HEAD:refs/heads/$branch" \
        || refuse "$branch moved while it was rebased; run prepare again"
    fi
    included="$(pr_numbers HEAD)"

    checks=()
    if [ -n "${RELEASE_CHECKS:-}" ]; then
      checks+=("checks=${RELEASE_CHECKS}")
    elif [ -f AGENTS.md ]; then
      while IFS= read -r line; do checks+=("$line"); done < <(sed -n 's/^# \(lint\|test\)  *\(.*\)$/\1=\2/p' AGENTS.md)
    fi
    failed=""
    for entry in "${checks[@]+"${checks[@]}"}"; do
      name="${entry%%=*}"; cmd="${entry#*=}"
      if ! bash -c "$cmd"; then failed="$name"; break; fi
    done
    if [ -n "$failed" ]; then
      ci="$(jq -n --arg f "$failed" '"red: " + $f')"
      write_result red
      echo "release-phase: the check '$failed' is red on $branch" >&2
      exit 1
    fi
    ci='"green"'
    git push -q origin "HEAD:$mark" || refuse "cannot mark the prepared head"
    write_result green
    ;;

  publish)
    if [ -n "$(remote_sha "refs/tags/$version")" ]; then
      # the version is tagged: published already when its commit is on main and its branch is gone
      git rev-parse -q --verify "refs/tags/$version^{commit}" >/dev/null || refuse "cannot read the tag $version"
      if git merge-base --is-ancestor "refs/tags/$version^{commit}" origin/main && ! has_branch; then
        gh release view "$version" >/dev/null 2>&1 \
          || gh release create "$version" --verify-tag --title "$version" --generate-notes || refuse "cannot make the release $version"
        tag="$(jq -n --arg t "$version" '$t')"; write_result published
        echo "release-phase: $version is published already, nothing to do"
        exit 0
      fi
      refuse "the tag $version exists and is not this release's; a tag is never moved"
    fi

    if has_branch; then
      head="$(git rev-parse "refs/remotes/origin/$branch")"
      [ "$(remote_sha "$mark")" = "$head" ] || refuse "no green prepare of the head of $branch ($head); run prepare"
      git merge-base --is-ancestor origin/main "$head" || refuse "main moved after prepare; run prepare again"
      included="$(pr_numbers "$head")"
      git checkout -q -B main origin/main && git merge -q --ff-only "$head" || refuse "main cannot be fast-forwarded to $branch"
      git push -q origin main || refuse "cannot push main"
    else
      # urgent: no branch, the work is on main already
      git checkout -q -B main origin/main
    fi

    git tag "$version" HEAD && git push -q origin "refs/tags/$version" || refuse "cannot tag $version"
    if has_branch; then
      git push -q origin ":refs/heads/$branch" ":$mark" || refuse "cannot delete $branch"
    fi
    gh release create "$version" --verify-tag --title "$version" --generate-notes || refuse "the tag $version is set, the release is not made; run publish again"
    tag="$(jq -n --arg t "$version" '$t')"
    write_result published
    ;;
esac
