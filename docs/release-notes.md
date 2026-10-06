# Current remediation status — 6 October 2026

Runtime baseline: `7a37ea3`. T01–T10 are implemented and independently accepted.
The final T10 commit completes the local remediation sequence.

- Exact conservative budget accounting and reversible generic JSON representation.
- Structural capability preflight, validated absence grader and fallible HTTP setup.
- Shared optimizer audit semantics, bound callback artifacts and concrete provenance.
- Lifecycle cleanup and budget Claim/Release conformance regressions.
- Direction-neutral QualityThreshold, canonical numeric descriptors, safe measurement
  localization, binary agreement API and explicit runtime/serialization ownership.
- Root-only isolated release checkout and exact atomic tag publication/recovery.

Go 1.27.1; root runtime has no third-party dependencies. contracttest is test-only.
Supported wire formats and API breaks: [authoritative migration](migration.md).
Final T10 `make validate` passed root/contracttest race, pinned golangci-lint
2.14.0 with **0 issues in both modules**, and **seven runnable examples**.
Independent schema drift, corpus/semantic negatives and local release/baseline
fixtures passed. Commands, independent verdicts and limits are recorded in
[remediation verification](remediation-verification.md).

Limits: cooperative cancellation, process-local reference budgets/ledger,
local filesystem fixtures on macOS, scripted judges and short targeted fuzz.
No real release/push, live provider quality, arbitrary callback safety or production
infrastructure certification. See [concurrency](concurrency.md) and [release](release.md).

## Historical initial release snapshot (superseded)

The original undated text below predates the 6 October remediation. Its six-example
and outstanding-lint counts are historical, not current guarantees.

Initial Go library release for evaluating typed targets.

- Immutable datasets, versioned codecs, bounded scenario runs and generation lineage.
- Isolated execution, cleanup, cancellation and atomic budget reservations.
- Deterministic and scripted judge grading, evidence capture, privacy projections and offline re-scoring.
- Compatible experiment comparisons, case-level uncertainty, local artifact persistence and export delivery.
- Optional observation workers, typed optimizer, HTTP JSON adapter, CLI and runnable examples.

Requires Go 1.27.1. The runtime core has no third-party dependencies.
The contracttest module is development-only and is not separately released.

Validation: formatting, vet and race tests in both modules, real local HTTP tests,
CLI subprocesses and six runnable examples. The initial independent acceptance
covered 171/171 requirements and closed 14 reproduced defects; subsequent updates
were checked with the documented test commands.

Known limits: callbacks must cooperate with contexts; filesystem guarantees are
local; reference budgets and holdout ledger are process-local; real LLM accuracy
and production integrations are unverified. The restored strict linter reports
219 outstanding findings (215 root, 4 contracttest); this release does not claim
strict-lint acceptance.
