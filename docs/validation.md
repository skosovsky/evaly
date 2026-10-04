# Validation record

Current toolchain/dependency update: see the 5 October entry below. Earlier
versions and acceptance results are preserved as historical evidence.

Task date: 4 October 2026 (Asia/Ho_Chi_Minh). No commits or releases.

## Toolchains and module graph

The minimum declared Go version is **1.26.1**, matching the neighboring flowy
module's minimum; evaly does not depend on it. The default compiler at final
inspection is **Go 1.26.5 darwin/arm64**. Go **1.26.1** and **1.27.1** were also
explicitly selected and their version outputs verified. The initial environment
reported 1.27.1; final validation uses explicit selections instead of assuming the
ambient binary has remained the same. CI pins supported versions.

`go list -m all` in the root returns only `github.com/skosovsky/evaly`.
Optional packages live in the core module but the core does not import them.
`contracttest` is a separate test-only module, with jsonschema/v6 v6.0.2 and its
pinned transitive dependency. It cannot add runtime dependencies to consumers.

## Executed checks

- `env GOCACHE=/private/tmp/evaly-go-cache GOPATH=/private/tmp/evaly-toolchains make validate` — PASS on Go 1.26.5. Includes root and contracttest `go vet`, normal tests and race tests, plus gofmt check.
- `env GOCACHE=/private/tmp/evaly-go126-cache GOPATH=/private/tmp/evaly-toolchains GOTOOLCHAIN=go1.26.1 make validate` — PASS. Same full gates, including the separate schema module and CLI subprocesses.
- `env GOCACHE=/private/tmp/evaly-go127-cache GOPATH=/private/tmp/evaly-toolchains GOTOOLCHAIN=go1.27.1 make validate` — PASS on both modules. A final scenario-restore regression was added afterward; final post-repair gates are recorded below.
- All six runnable examples were actually executed: calculation, CRM, protocols, HTTP, observation and optimizer — PASS (process exit 0). Calculation correctly reports a quality fail (3/4) without treating this as a process infrastructure failure; CRM records refund_exists=false despite “refund done”; protocols save a versioned search trajectory; HTTP performs four local real requests; observation persists/reopens its permitted view and grades; optimizer reports a complete winner/holdout comparison without changing settings.
- `contracttest/TestGeneratedArtifactsAgainstWireSchemas` validates generated envelope, dataset, experiment, evidence, comparison, saved view, assessment, scenario, candidate, observation/result, search and HTTP request/response using an independent Draft 2020-12 engine. Negative nested usage/cleanup/grade/plan/version fixtures reject. Every emitted schema compiles.
- Independent correctness audit added adversarial regression files, discovered fourteen defects (including final scenario semantic restore) and retained their assertions through repairs. Its final report records actual independent checks and scope limits.

No benchmark numbers or production/vendor integration claims are made. Local HTTP
servers require no external credentials. Filesystem tests use temporary local
storage. CLI tests build and execute a subprocess binary and check all four exit
codes. Other hosts, remote filesystems, real LLM robustness and distributed state
remain explicit unsupported or host-owned boundaries in design/README.

## Acceptance history

Initial independent completeness audit: 150/171 (87.72%). The denominator is
unchanged. Repairs add saved-view persistence, versioned scenario lineage, full
schemas, shared protocol conformance suites, report/replay provenance and explicit
missing synthetic-fixture assertions. Final independent audit status is recorded
in `reviews/completeness.md`; correctness in `reviews/correctness.md`.

## Final post-repair gates

After the shared scenario semantic validator and its unchanged independent
regressions, all three commands below completed with exit 0, using fresh caches:

- `env GOCACHE=/private/tmp/evaly-final-1.26.1-cache GOPATH=/private/tmp/evaly-toolchains GOTOOLCHAIN=go1.26.1 make validate` — PASS.
- `env GOCACHE=/private/tmp/evaly-final-1.26.5-cache GOPATH=/private/tmp/evaly-toolchains GOTOOLCHAIN=go1.26.5 make validate` — PASS.
- `env GOCACHE=/private/tmp/evaly-final-1.27.1-cache GOPATH=/private/tmp/evaly-toolchains GOTOOLCHAIN=go1.27.1 make validate` — PASS.

Each command ran root and contracttest vet, normal tests, race tests and the
repository-wide formatting gate. CLI subprocesses, local HTTP integration,
all ten synthetic fixtures and independent audit regressions are included.
No implementation changes followed these gates; only acceptance/validation
status was finalized.

Final independent completeness: **171/171 (100%)**, BOOT and every EVL-001–007
card 100%. Final independent correctness: all fourteen confirmed defects closed;
**в рамках проведённой проверки дефектов не выявлено**. Reports preserve the
initial 150/171 and second 168/171 results and the reproduced defects.

## 5 October 2026 — toolchain, dependencies and naming

Both modules now declare Go 1.27.1, the current stable release verified through
https://go.dev/dl/?mode=json. The installed compiler reports go1.27.1 darwin/arm64.
The root module still has no third-party dependencies. contracttest uses
jsonschema/v6 v6.0.3 and x/text v0.42.0. `go get -u ./...` and `go mod tidy`
completed; `go list -deps -test` confirms these are its only external packages.
Upstream dependency-test-only module graph entries are not imported by evaly.

The restored Makefile template is adapted to `. contracttest`, retains lint/fix,
race tests, benchmarks, coverage and fuzz targets, and adds formatter/vet/validate,
schemas and fixtures targets. Targets calling a nonexistent release script were
removed. Fuzz functions run individually. The linter template retains its strict
rules; exhaustruct migrated to exhaustruct_v5, import prefixes and local replace
settings describe evaly, and unused third-party exceptions were removed.
CI follows go.mod and installs golangci-lint v2.14.0 for formatting/config checks.
Checkout/setup-go/lint actions are pinned to v7.0.1/v7.0.0/v9.3.0.

No task numbers are present in Go code, schemas, source filenames or entity names.
Regression test files and helpers use behavior names instead of audit provenance.
Synthetic fixture IDs are descriptive names. Numbered requirements remain only
in specification/acceptance documentation. LICENSE names evaly contributors.
All workspace .DS_Store files were removed; the ignore rule remains.

Executed update verification:

- `golangci-lint config verify` — PASS.
- `make fmt` — PASS; configured goimports/golines/gofmt applied.
- `make validate` under Go 1.27.1 — PASS, including both modules, formatter check,
  vet, race tests, HTTP integration and CLI subprocesses.
- All six example commands — PASS (each process exited 0).
- `go mod verify` in contracttest — PASS.
- Strict root linter — 215 reported issues; contracttest — 4 reported issues.
  `make lint` exits nonzero. Rules were not weakened to hide existing complexity,
  complete-struct initialization, repeated literals and other style findings.
  CI preserves the existing format/vet/test gate; lint remains an explicit local
  target pending the user's choice about a full strict-style refactor.

The independent acceptance reports above describe the prior implementation state;
this update was verified through the commands here, not a new independent audit.
No commits or releases were created.
