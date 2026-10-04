# Validation record

## 5 October 2026 — measurement and paired execution

Authority: task 03 and common task README, measurement, paired and calibration
contracts. Source fingerprint:
`554d9261c1d27994f427afc2b8485260dd5f3f8da475da86187922f0282a0bdc`
(108 files; reports and this journal excluded).

- `env GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate` — exit 0 on
  Go 1.27.1 darwin/arm64, both modules' formatting/vet/race. Log:
  `/private/tmp/evaly-measurement-validation.log`; contracttest race 98.456 s.
- All six runnable examples — exit 0 on the same source; logs
  `/private/tmp/evaly-measurement-example-*.log`.
- Coverage regression uses 100 eligible cases and one matched case; repeats do
  not inflate independent N. Native numeric directions, metadata incompatibility,
  unavailable means, fixed aggregation and source provenance are covered.
- Numerical regressions preserve identical subnormal measurements across repeats,
  matched means and bootstrap; opposite finite extremes retain a finite mean.
- Real paired callbacks verify seeded within-slot interleaving, isolated lifecycle
  handles, bounded global concurrency and retained slots after cancellation,
  shared-budget exhaustion or infrastructure stop. Forged schedules reject restore.
- Calibration verifies TP/TN/FP/FN, missing/error/abstain counts, overlapping host
  groups and unavailable zero-denominator rates. New conformance helpers exercise
  assertion, numeric higher/lower and calibration reference implementations.
- Independent schema corpus covers 15 wire kinds, including actual numeric grades,
  paired schedule and calibration. Disposable generation checks byte/inventory drift.
- `git diff --check` — PASS. Current formats: experiment/assessment/
  observation-result/search v3, comparison v2, calibration v1.

Independent final acceptance reports: `docs/reviews/measurement-completeness.md`
and `measurement-correctness.md`. Repairs invalidate the fingerprint and require
both reviewers to recheck; this journal records executed checks separately.

## 5 October 2026 — evidence and HTTP revision 2

Authority: `.cursor/task/02-evidence-and-wire-contracts.md`, common README,
`docs/evidence-contract.md`, `docs/wire-contract.md` and `docs/http-protocol.md`.

- `env GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate` — PASS on
  Go 1.27.1 darwin/arm64. Formatter, vet and race tests in both modules; output
  `/private/tmp/evaly-evidence-validation.log` (contracttest race: 62.454 s).
- All six runnable examples — exit 0, logs
  `/private/tmp/evaly-evidence-example-{calculation,crm,protocols,http,observation,optimizer}.log`.
  Expected demonstration quality failures remain unchanged.
- Independent JSON Schema engine and runtime decoder agree on all 14 produced
  service envelope types and 4124 structural field mutations. Duplicate-key byte
  tests cover each root separately; seven public restore paths and two semantic
  checksum/revision failures distinguish structure from semantic validity.
- Schema generation runs in a temporary directory and compares the exact
  inventory and bytes; it does not rewrite the checked-out schemas.
- `git diff --check` — PASS. Root source fingerprint after checks:
  `aabce87e5dec89ddeaeaae3592c44f0319d23260e150ec63242f9c63819c27b0`
  (93 files; review reports and this journal excluded).

Final independent acceptance is recorded in `docs/reviews/evidence-completeness.md`
and `evidence-correctness.md`. A source repair requires a new fingerprint and
both reviewers' rechecks; this entry records actual checks, not an unconditional
claim of acceptance.

## 5 October 2026 — execution revision 2

Current task authority: `.cursor/task/01-execution-and-assessments.md`, common
README and `docs/execution-contract.md`. Earlier records below are historical.

- `env GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate` — PASS on
  Go 1.27.1 darwin/arm64. Root and contracttest formatter, vet and race tests.
  Full output: `/private/tmp/evaly-execution-validation.log`.
- All six `go run ./examples/{calculation,crm,protocols,http,observation,optimizer}`
  commands — exit 0 on the same source state. Calculation intentionally reports a
  regression; CRM intentionally reports missing refund. These quality failures
  are expected example results, not infrastructure failures.
- `go test -run '^$' -bench BenchmarkDatasetCaseAt -benchtime=1x -benchmem .` —
  PASS. N=10/100/1000: 320/3394/34987 allocations and
  18000/184144/1881744 B/op; observed 46/314/2917 microseconds. These single-sample
  observations are not performance guarantees. Counting-codec test separately
  asserts linear decode count without timing thresholds.
- `git diff --check` — PASS. Updated schemas generated from current types; no
  legacy revision-1 schemas retained for changed formats.

Independent completeness and correctness acceptance are recorded separately in
`docs/reviews/execution-completeness.md` and `execution-correctness.md`. Their
fingerprints cover implementation, tests, schemas, applicable contracts and task
authority, excluding generated review reports and this validation journal. Any
implementation repair requires new verification and both reviewers' rechecks.

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

## Sequential task 04 — bounded optimizer

Final source fingerprint: `08eba985354d909f1ac16e7bc9012618d604ee897f41e00c46f3524dfb0a802b`
(119 files, including task authorities; acceptance reports and this journal excluded).

`GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate` completed with exit 0:
format, vet and race tests in both modules; contracttest race completed in 93.815s.
All six runnable examples completed with exit 0. `git diff --check` passed.
The final formatter fix split assertions without removing any condition.
Independent completeness and correctness reports are in `reviews/search-completeness.md`
and `reviews/search-correctness.md` and refer to the same source fingerprint.
