#!/usr/bin/env bash
#
# release-phase.sh — the phases `prepare` and `publish` of .github/workflows/release-build.yml
# (BCKLG-1038, trunk-based). All work is on `main`. A minor version vX.Y.0 cuts its branch release/X.Y.0
# from `main`; the patches vX.Y.Z of that version are tagged on the same branch, after the fix has landed
# on `main` and been cherry-picked onto it. A release branch is never merged back into `main`.
#
#   prepare  vX.Y.0: release/X.Y.0 is cut from origin/main when it does not exist; vX.Y.Z (Z > 0): the
#            branch must exist. An existing branch is never moved. The checks of the repository run on
#            its head; a green head is marked in refs/release-prepared/X.Y.Z. No tag, main is not touched.
#            A version that is tagged already is refused.
#   publish  only when that mark is the head of the branch: the version is tagged on that head and a
#            GitHub Release is made; the mark is deleted, the branch stays, main is not touched. A version
#            that is tagged on the branch already changes nothing.
#
# Usage: release-phase.sh <prepare|publish> --release <id> --version vX.Y.Z [--result <file>]
#
# --release  ^[a-z0-9.]+$, the id of the backlog's release (for the result)
# --version  ^v[0-9]+\.[0-9]+\.[0-9]+$, required by every phase: it names the branch release/X.Y.0
# --result   where the JSON result is written (default release-result.json)
# RELEASE_CHECKS  the checks to run on the branch; default: the `# lint` and `# test` lines of the
#                 Commands block of AGENTS.md.
#
# The input is checked before anything is written. Only the maintainer starts the workflow that runs it.
# Exit: 0 done | 1 the phase did not complete (red checks, refused) | 2 wrong usage.

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

case "$phase" in prepare|publish) ;; *) echo "release-phase: the phase is prepare or publish" >&2; exit 2 ;; esac
[[ "$release" =~ ^[a-z0-9.]+$ ]] || { echo "release-phase: --release must match ^[a-z0-9.]+\$" >&2; exit 2; }
if [[ ! "$version" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
  echo "release-phase: --version vX.Y.Z is required; it names the branch release/X.Y.0" >&2; exit 2
fi
patch="${BASH_REMATCH[3]}"
branch="release/${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.0"
mark="refs/release-prepared/${version#v}"
repo="${GITHUB_REPOSITORY:-$(basename "$(git remote get-url origin 2>/dev/null || echo unknown)" .git)}"

included='[]'; ci='null'; tag='null'

write_result() { # status
  jq -n --arg repo "$repo" --arg release "$release" --arg version "$version" --arg branch "$branch" \
    --arg phase "$phase" --arg status "$1" --argjson included "$included" --argjson ci "$ci" --argjson tag "$tag" \
    '{repo:$repo, release:$release, version:$version, branch:$branch,
      phase:$phase, status:$status, included:$included, ci:$ci, tag:$tag}' > "$result"
}
refuse() { echo "release-phase: $*" >&2; exit 1; }

git config user.name >/dev/null 2>&1 || git config user.name "github-actions[bot]"
git config user.email >/dev/null 2>&1 || git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

git fetch -q --prune --tags origin '+refs/heads/*:refs/remotes/origin/*' || refuse "cannot fetch origin"
git rev-parse -q --verify refs/remotes/origin/main >/dev/null || refuse "origin has no main"
remote_sha() { git ls-remote origin "$1" | awk 'NR == 1 { print $1 }'; }
has_branch() { git rev-parse -q --verify "refs/remotes/origin/$branch" >/dev/null; }
pr_numbers() { # the PRs of a head: (#N) of the subjects of its commits after the last tag before it
  local since range="$1"
  since="$(git describe --tags --abbrev=0 --match 'v[0-9]*.[0-9]*.[0-9]*' "$1" 2>/dev/null)" && range="$since..$1"
  git log --format=%s "$range" | grep -o '(#[0-9]*)' | tr -d '(#)' | sort -un | jq -R 'tonumber' | jq -s .
}

case "$phase" in
  prepare)
    [ -z "$(remote_sha "refs/tags/$version")" ] || refuse "$version is tagged already; prepare the next version"
    if ! has_branch; then
      [ "$patch" = 0 ] || refuse "$branch does not exist: a patch is tagged on the branch of its minor version"
      git push -q origin "refs/remotes/origin/main:refs/heads/$branch" || refuse "cannot cut $branch from main"
      git fetch -q origin "+refs/heads/$branch:refs/remotes/origin/$branch" || refuse "cannot fetch $branch"
    fi
    git push -q origin ":$mark" 2>/dev/null || true
    git checkout -q -B "$branch" "refs/remotes/origin/$branch" || refuse "cannot check out $branch"
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
      # the version is tagged: published already when its commit is on the branch
      git rev-parse -q --verify "refs/tags/$version^{commit}" >/dev/null || refuse "cannot read the tag $version"
      if has_branch && git merge-base --is-ancestor "refs/tags/$version^{commit}" "refs/remotes/origin/$branch"; then
        gh release view "$version" >/dev/null 2>&1 \
          || gh release create "$version" --verify-tag --title "$version" --generate-notes || refuse "cannot make the release $version"
        tag="$(jq -n --arg t "$version" '$t')"; write_result published
        echo "release-phase: $version is published already, nothing to do"
        exit 0
      fi
      refuse "the tag $version exists and is not on $branch; a tag is never moved"
    fi

    has_branch || refuse "$branch does not exist; run prepare"
    head="$(git rev-parse "refs/remotes/origin/$branch")"
    [ "$(remote_sha "$mark")" = "$head" ] || refuse "no green prepare of the head of $branch ($head); run prepare"
    included="$(pr_numbers "$head")"
    git tag "$version" "$head" && git push -q origin "refs/tags/$version" || refuse "cannot tag $version"
    git push -q origin ":$mark" || refuse "the tag $version is set, the mark $mark is not deleted"
    gh release create "$version" --verify-tag --title "$version" --generate-notes || refuse "the tag $version is set, the release is not made; run publish again"
    tag="$(jq -n --arg t "$version" '$t')"
    write_result published
    ;;
esac
