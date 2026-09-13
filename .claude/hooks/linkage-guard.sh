#!/usr/bin/env bash
# PreToolUse(Bash) guard — gate ONLY release tagging. ~0 tokens. Normal push/merge untouched.
# Fires ONLY on a real git tag/push command, never on text that merely mentions one.
# Bypass a deliberate release: LINKAGE_OFF=1 <cmd>
#
# This is the source every Galaxio repository copies. Change it here and carry the
# change outward; linkage-guard_test.sh beside it is the contract those copies
# share. It reached this shape after two repositories hit the same failure
# independently — a gh pr create whose body documented the hook was blocked by
# it, reporting a version that came from prose.
#
# A release branch serves a whole minor line, so its name is not the version of
# every patch release. Publishing is what the tag does; a branch push is not a
# release.
#
# This is the fast in-session failure: it only ever sees commands an agent
# runs, so a push from a terminal or an IDE never reaches it. .githooks/pre-push
# catches those by reading the refs Git hands it. The linkage CI job validates
# the issue, PR and milestone before the normal automatic release path reaches
# main; these local guards protect exceptional manual tags.
set -uo pipefail
input=$(cat)
cmd=$(printf '%s' "$input" | jq -r '.tool_input.command // ""' 2>/dev/null || true)
[ -n "$cmd" ] || exit 0
[ "${LINKAGE_OFF:-}" = "1" ] && exit 0
case "$cmd" in *LINKAGE_OFF=1*) exit 0 ;; esac
case "$cmd" in *git*) ;; *) exit 0 ;; esac
root="${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"
checker="$root/scripts/check-linkage.sh"
block() { printf 'BLOCKED by linkage-guard:\n%s\n' "$1" >&2; exit 2; }

# Drop heredoc bodies before inspecting a command. A tag example in a body is
# documentation, not an operation the shell will run.
strip_heredocs() {
  local line trimmed term="" re='<<-?[[:space:]]*["'"'"']?([A-Za-z_][A-Za-z0-9_]*)["'"'"']?'
  while IFS= read -r line; do
    if [ -n "$term" ]; then
      trimmed="${line#"${line%%[![:space:]]*}"}"
      [ "$trimmed" = "$term" ] && term=""
      continue
    fi
    printf '%s\n' "$line"
    [[ $line =~ $re ]] && term="${BASH_REMATCH[1]}"
  done
}

# Split shell lists only outside quotes. This is deliberately a small lexer,
# not a shell evaluator: the guard must never execute the command it inspects.
split_segments() {
  local s="$1" segment="" state="" c next
  local i=0 len=${#s}
  while [ "$i" -lt "$len" ]; do
    c="${s:i:1}"
    case "$state" in
      single)
        segment="${segment}${c}"
        [ "$c" = "'" ] && state=""
        ;;
      double)
        segment="${segment}${c}"
        if [ "$c" = "\\" ]; then
          state=escape-double
        elif [ "$c" = '"' ]; then
          state=""
        fi
        ;;
      escape|escape-double)
        segment="${segment}${c}"
        state=""
        ;;
      *)
        case "$c" in
          "'") segment="${segment}${c}"; state=single ;;
          '"') segment="${segment}${c}"; state=double ;;
          "\\") segment="${segment}${c}"; state=escape ;;
          ';'|$'\n')
            printf '%s\n' "$segment"
            segment=""
            ;;
          '&'|'|')
            printf '%s\n' "$segment"
            segment=""
            next="${s:i+1:1}"
            [ "$next" = "$c" ] && i=$((i + 1))
            ;;
          *) segment="${segment}${c}" ;;
        esac
        ;;
    esac
    i=$((i + 1))
  done
  printf '%s\n' "$segment"
}

# Peel leading environment assignments and wrappers so rtk proxy git tag still
# counts as a Git command.
peel() {
  local s="$1" head
  s="${s#"${s%%[![:space:]]*}"}"
  while :; do
    head="${s%% *}"
    case "$head" in
      [A-Za-z_]*=*) s="${s#"$head"}" ;;
      rtk|proxy|sudo|env|nohup|time|command|builtin|exec|xargs) s="${s#"$head"}" ;;
      *) break ;;
    esac
    s="${s#"${s%%[![:space:]]*}"}"
  done
  printf '%s' "$s"
}

is_tag=0
hit=""
segments=$(split_segments "$(printf '%s\n' "$cmd" | strip_heredocs)")
while IFS= read -r seg; do
  seg=$(peel "$seg")
  case "$seg" in git|git\ *) ;; *) continue ;; esac
  # Non-release subcommands are out of scope. Scoped to THIS segment: a read
  # or commit before a tag must not disable the following release check.
  printf '%s' "$seg" | grep -qE '^git[[:space:]]+(commit|log|show)\b' && continue
  # Any git tag carrying a vX.Y.Z, in any flag order. Read-only or destructive
  # tag subcommands are not release creation and remain out of scope.
  if printf '%s' "$seg" | grep -qE '^git[[:space:]]+tag\b' \
     && printf '%s' "$seg" | grep -qE '\bv?[0-9]+\.[0-9]+\.[0-9]+' \
     && ! printf '%s' "$seg" | grep -qE '^git[[:space:]]+tag[[:space:]]+(-l\b|--list\b|-d\b|--delete\b|-v\b|--verify\b|-n)'; then
    is_tag=1; hit="$seg"; break
  fi
  if printf '%s' "$seg" | grep -qE '^git[[:space:]]+push\b' \
     && printf '%s' "$seg" | grep -qE '(--tags|refs/tags/|[[:space:]](origin[[:space:]]+)?v[0-9]+\.[0-9]+\.[0-9]+([[:space:]]|$))'; then
    is_tag=1; hit="$seg"; break
  fi
done <<<"$segments"
[ "$is_tag" = 1 ] || exit 0
[ -x "$checker" ] || block "checker missing ($checker) — release cannot be verified"
ver=$(printf '%s' "$hit" | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | head -1)
[ -n "$ver" ] || block "tag/release push without explicit vX.Y.Z — verify: scripts/check-linkage.sh --for-tag <version>"
out=$("$checker" --for-tag "$ver" 2>&1) || block "$out"
exit 0
