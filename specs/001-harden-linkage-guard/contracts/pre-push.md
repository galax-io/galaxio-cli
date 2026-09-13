# Contract: git pre-push hook

**File**: `.githooks/pre-push` (executable; byte-identical to upstream). **Enabled by**: `git config core.hooksPath .githooks`, once per clone. **Suite**: `.githooks/pre-push_test.sh`, 12 cases.

## Input

git runs the hook with `<remote name> <remote url>` as arguments and one line per ref on stdin: `<local ref> <local sha> <remote ref> <remote sha>`. The hook reads stdin only.

## Decision table

| stdin line | Effect |
|---|---|
| `refs/heads/x <sha> refs/heads/x <sha>` (any branch, any name) | pass, checker not called |
| `refs/tags/foo <sha> refs/tags/foo <sha>` (tag not named `v<digit>…`) | pass |
| `refs/tags/v1.2.3 <sha> refs/tags/v1.2.3 <sha>` | `check-linkage.sh --for-tag v1.2.3`; pass on 0, refuse otherwise |
| `(delete) 0000…0 refs/tags/v1.2.3 <sha>` | pass (deletion publishes nothing) |
| several tag lines (`git push --tags`) | each judged; one failure refuses the push |
| checker missing or not executable | refuse: `pre-push: <path> is missing, so v1.2.3 cannot be verified` |
| run outside a git work tree | exit 0 |

## Output

Exit 0 lets the push proceed. Exit 1 refuses it; stderr carries the checker's findings followed by `pre-push: refusing to publish <version> — its milestone is not ready`.

## Non-goals

Opt-in and skippable with `--no-verify`; it is the local fast failure, not the authority. In this repository CI creates release tags, so this hook fires only on the rare manual tag.
