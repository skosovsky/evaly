# Repository verification

Infrastructure reference: [ragy at 91e3ff2](https://github.com/skosovsky/ragy/tree/91e3ff2cdc87c49b5ad9d1cf0afa23d35832521e).
Make and the Bash release script use that exact revision. Modules are discovered
from all go.mod files, excluding hidden directories and vendor; commands use
GOWORK=off. There is no registry, project.mk, aggregate runner or separate
publication list.

| Command | Scope |
|---|---|
| `make modules` | Automatic development and release inventory. |
| `make lint` | Verify config, formatting diff and strict lint in every module. |
| `make test` | Fresh ordinary tests with race; schemas, conformance, Make/release fixtures and recipe unit checks. |
| `make test-integration` | Integration tag and TestIntegration prefix; workflow/HTTP, streaming/judge/capture/export, consumer matrices and prepared module artifacts. |
| `make test-e2e` | E2E tag and TestE2E prefix; CLI, seven runnable examples, dataset/stream/grade/disk/export roundtrip. |
| `make test-live` | Live tag and TestLive prefix; no live-provider tests currently exist. |
| `make fix` | Go fix, formatting and lint fixes; modifies source. |
| `make cover` / `make bench` | Per-module coverage / allocation benchmarks. |
| `make fuzz` | Separate 30-second campaign for every discovered fuzz target. |

Tags and prefixes are both required: profile targets must not execute ordinary
unit tests. Keep ordinary arithmetic and port validation in unit tests; use the
integration profile for deterministic SDK/protocol composition, e2e for complete
executable or persisted workflows. Examples compile in unit tests and execute in
the e2e profile. Paid providers and long performance/fuzz campaigns are separate.
No credentials or live model calls are used by the mandatory profiles.

## Prerequisites and modules

Use Go 1.27.2, golangci-lint 2.14.0, Bash, Git, Make and a C compiler for race.
Go manifests require 1.27.1. Make uses tools from PATH; CI pins their versions.
Module dependencies and the pinned source-matrix commits require network access
unless already available through normal caches. Missing required tools, SDK APIs,
resolution or network prerequisites fail; tests do not skip them. Evaly has no PDF,
Python, database or Docker runtime dependency. Docker is only used to reproduce
Linux verification from macOS.

The inventory is core, contracttest and integrations/recipes. Core remains a
standard-library-only Go module. Contracttest owns schema, infrastructure and
artifact tests and adds x/mod for standard module ZIP construction. Recipes own
prompty/metry dependencies. Both nested modules use development replaces for the
current core. Future releases publish every discovered module; see
[release](release.md) for source/candidate semantics and historical root-only tags.

```sh
make modules | while IFS= read -r module; do
  (cd "$module" && go mod download all) || exit "$?"
done
make lint
make test
make test-integration
make test-e2e
```

Schema regeneration is `go run ./internal/schemagen`; tests generate to a temporary
directory and compare all schema bytes without modifying repository files.
CLI fixtures use `go run ./cmd/evaly fixture --store /tmp/evaly-fixtures --id baseline`.

## Consumer and artifact evidence

The Go integration matrix replaces the old consumer script/workflow:

| Lane | Evaly | prompty | metry |
|---|---|---|---|
| Published | v0.3.0, no replacements | v0.15.0 | v0.9.0 |
| Audited source | current checkout | 5607832ee869f63bc3c6768a02ed60a019cd691e | 7ef9f89638f29408f50996236d46d6c105dc8183 |
| Unsupported | current checkout | 860146dd6e74154abbaad415d9a44e112a05c16d | bd61e299a084880437fc79639c2ea14ad75b54a5 |

Source lanes fetch exact Git objects into temporary directories and log resolved
module identities. Supported lanes run unit, integration and e2e recipe fixtures
with race. The unsupported lane requires a missing Stream capability before any
semantic fixture execution; unrelated resolution errors fail that lane. This is
an explicit negative boundary, not a semantic-composition success.

Artifact tests export current source into a temporary candidate, prepare internal
manifests and create standard x/mod ZIPs for every discovered module. An external
consumer resolves exact versions without replacements, compiles all module
packages and runs the calculation example. Candidate artifacts install into a
fresh module cache; checksum-verified external archives may be reused from the
normal Go download cache, with the public proxy as fallback.

Release fixtures use temporary bare Git repositories and test source-to-main,
prepared-candidate-to-tags, each failed gate, atomic server refusal, cancellation,
identity/collision checks, lost push responses and inspect/resume/finish. They never
use the production origin.

## Adaptations from the reference

- CI removes ragy's PDF/Python runtime; evaly has no corresponding dependency.
- Lint uses the evaly import prefix, removes ragy-specific type/path and vendor-SDK
  exceptions, and retains shared rules. Recipes use the root strict configuration.
  The single nilnil suppression describes a deliberately invalid host fixture.
- Existing contracttest hosts infrastructure tests; no tooling module or runner
  was added. Existing workflows and Bash/Python release/consumer entrypoints were
  replaced by standard Make profiles and Go fixtures.
- Recipes explicitly replace the current core during development and join future
  module publication. Historical v0.4.0 and earlier remain root-only releases.
- Historical acceptance/validation documents retain their dated commands and
  evidence; they are not the current verification contract.

## Execution record — 9 October 2026

| Check | macOS arm64 | Linux amd64 |
|---|---|---|
| actionlint 1.7.12 | PASS | PASS |
| golangci-lint 2.14.0 config verify | PASS | PASS |
| Dependency download for all modules | PASS | PASS |
| make lint | PASS, zero issues in all three modules | PASS, zero issues in all three modules |
| make test | PASS with race | PASS with race |
| make test-integration | PASS with race | PASS with race |
| make test-e2e | PASS with race | PASS with race |
| bash -n scripts/release.sh | PASS | PASS |

Both platforms used Go 1.27.2. Linux ran Debian Bookworm from
`golang:1.27.2-bookworm` (digest
`sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61`)
under amd64 emulation on the macOS arm64 host. It used a temporary source snapshot
whose contents matched the working tree; no other library checkout was changed.
The macOS linter binary was built with Go 1.27.1; the Linux binary with Go 1.27.2.

Profile inventory contains 18 top-level integration tests and four top-level e2e
tests. Both platforms passed the published, audited-source and unsupported
consumer lanes, prepared-artifact checks and seven example executions. Ordinary
tests passed all 18 release fixtures, including every failed gate and recovery
with the original candidate. Tagged integration/e2e code also passed strict lint
on macOS. Module-path/directory correspondence and hidden/vendor discovery were
checked by Go fixtures. Git diff whitespace checks passed.

Makefile and scripts/release.sh were compared byte-for-byte with the fixed ragy
reference and match. Required prerequisites were not skipped. An initial
unsupported-consumer run timed out while tidying unrelated old SDK test
dependencies; the final negative lane checks its missing Stream API before tidy,
and both complete platform runs passed after that correction.

Live-provider calls, benchmark/fuzz campaigns and hosted GitHub CI were not run.
No production release, push or workflow dispatch was performed.
