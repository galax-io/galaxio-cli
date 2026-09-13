# Contract: CI jobs added to `.github/workflows/ci.yml`

Job ids and display names are a contract: branch protection and the `needs` edges reference them. Existing jobs keep their ids, names and steps.

## `shell-suites` (name: `shell suites`)

| Field | Value |
|---|---|
| Triggers | every `pull_request` and every `push` to `main` (the workflow's own `on:`; no `if`) |
| Permissions | inherits `contents: read` |
| Steps | `actions/checkout@v4`; one `run` step under `set -euo pipefail` |
| Discovery | `suites=(scripts/*_test.sh .claude/hooks/*_test.sh .githooks/*_test.sh)`; fails if fewer than 3 match |
| Execution | `bash "$t"` for each, printing `--- <path>` before it; first failing suite fails the job with that suite's own case output |
| Depended on by | `docker-image-pr`, `docker-image-main` (so a red suite stops the image and, on `main`, the release) |
| Runtime tools | `bash`, `git`, `jq` (all on `ubuntu-latest`) |

Suites expected at landing: `scripts/install_test.sh`, `.claude/hooks/linkage-guard_test.sh`, `.githooks/pre-push_test.sh`.

## `linkage` (name: `linkage`)

| Field | Value |
|---|---|
| Triggers | `pull_request` only: `if: github.event_name == 'pull_request'` |
| Permissions | job-level `contents: read`, `pull-requests: read`, `issues: read` |
| Env | `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}`, `REPO: ${{ github.repository }}` |
| Steps | `actions/checkout@v4`; `scripts/check-linkage.sh --pr ${{ github.event.pull_request.number }}` |
| Passes when | the PR has a milestone, closes ≥ 1 issue, and every closed issue is in that milestone |
| Fails when | any rule above fails (exit 1), or the PR cannot be read (exit 2); both are red, never silently green |
| Depended on by | nothing; merge protection is expected to require it |

## Unchanged

`test`, `integration`, `docker-image-pr`, `docker-image-main`, `release`, `publish-image` keep every step. The only edits outside the two new jobs are the two added `needs: shell-suites` lines.
