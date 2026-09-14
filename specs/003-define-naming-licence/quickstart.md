# Quickstart: Verify Ecosystem Naming and Licensing

Run from the repository root on branch `003-define-naming-licence` after the implementation
PRs have been applied. GitHub checks require an authenticated `gh` session with access to
the private `galax-io/comet` repository.

## 1. Verify the public command namespace

```sh
go run ./cmd/galaxio --help
go run ./cmd/galaxio report
go run ./cmd/galaxio report --help
```

Expected:

- root help lists `report` without removing any existing root command;
- both direct report invocations exit 0, print help to stdout, and leave stderr empty;
- report help identifies the namespace as finished-run reporting;
- no report flags, input formats, calculations, or output schemas are advertised.

Use the command-level tests to verify stream placement and invalid-argument exit code:

```sh
go test -run 'Test(HelpPrintsMinimalUsage|ReportCommand)' ./cmd/galaxio
```

Expected: the suite covers `report`, `report --help`, and an unexpected argument; the last
case exits 2 with an error on stderr.

## 2. Verify canonical repository identities

```sh
gh repo view galax-io/parsec --json nameWithOwner,visibility,licenseInfo,url
gh repo view galax-io/comet --json nameWithOwner,visibility,licenseInfo,url
```

Expected:

- `galax-io/parsec` is `PUBLIC` and its licence key is `mit`;
- `galax-io/comet` is `PRIVATE` for an authorized maintainer;
- lack of private-repository access is a failed evidence check, not proof of absence;
- no licence conclusion is drawn for `comet`.

Search active decision and user-facing documents for the exact names:

```sh
rg -n 'parsec|comet|galaxio report' README.md specs/003-define-naming-licence
rg -n 'galaxio-results|galaxio-tail' README.md specs/003-define-naming-licence
```

The first search must show all three canonical identities. Any match from the second search
must be historical/rejected-alternative context, never an active alias. No unresolved
clarification marker may remain in a planning artifact.

## 3. Audit licence surfaces

```sh
rg -n 'GPL-2.0-only' README.md Dockerfile specs/003-define-naming-licence
sed -n '1,24p' LICENSE
gh repo view galax-io/galaxio-cli --json nameWithOwner,licenseInfo,url
gh repo view galax-io/parsec --json nameWithOwner,visibility,licenseInfo,url
```

Expected:

- README and the OCI label say `GPL-2.0-only` exactly;
- `LICENSE` keeps the GNU GPL Version 2 terms verbatim and includes or is paired with the
  approved v2-only project notice;
- GitHub recognizes the CLI as GNU GPLv2 (its API may use the legacy `gpl-2.0` key);
- `parsec` is public and MIT-licensed;
- the decision record links the FSF Expat/MIT and Apache-2.0 classifications;
- no statement claims a whole-module licence audit or declares the pre-existing Cobra risk
  resolved.

## 4. Run repository quality gates

```sh
test -z "$(gofmt -l .)"
go mod tidy
git diff --exit-code -- go.mod go.sum
go vet ./...
go test -race -coverprofile=coverage.out ./...
go build ./...
go test -tags=integration -race -count=1 ./...
```

Expected: all commands pass, `go mod tidy` changes nothing, and CI confirms total coverage
is at least 80%. The feature adds no dependency and no new integration fixture.

## 5. Verify issue and milestone closure

Before merge, each implementation PR must be assigned to milestone 1 and carry only its
own closing reference:

```sh
scripts/check-linkage.sh --pr <PR_NUMBER>
```

After both implementation PRs land on `main`:

```sh
gh issue view 47 --repo galax-io/galaxio-cli --json number,state,milestone,url
gh issue view 48 --repo galax-io/galaxio-cli --json number,state,milestone,url
gh api repos/galax-io/galaxio-cli/milestones/1
```

Expected: #47 and #48 are closed, both remain attached to
`v0.12.0 Naming and licence`, and the milestone reports zero open issues.

Release tagging is deliberately not part of this quickstart. Once the milestone is complete,
the maintainer follows the separate release procedure, beginning with
`scripts/check-linkage.sh --for-tag v0.12.0` against complete history and tags.
