# Task 01 — independent correctness acceptance

Date: 2026-10-05. Reviewer: independent correctness subagent; did not implement or repair runtime, contracts, tests, or schemas.

Verdict: **PASS within the audited scope and performed checks. No identified unclosed defects.** This does not assert that all possible bugs are absent. Completeness is assessed by the separate reviewer.

Final frozen state: SHA-256 `0be19a748c94a25fa740af4310fb5ef80bf3a3f59f8882685500b1c869316126`, 83 files, independently reproduced with `python3 /private/tmp/evaly-review-state.py`. The fingerprint covers source, tests, schemas, normative documentation, configuration and task 01/README; it excludes review reports and the validation ledger to avoid self-reference.

## Scope and evidence

Read the full `.cursor/task/README.md`, `01-execution-and-assessments.md`, `docs/execution-contract.md`, and the affected implementation: `preflight.go`, `dataset.go`, `grader.go`, `snapshot.go`, `rescore.go`, `runner.go`, `scenario.go`, `budget.go`, `observation/worker.go`, `optimizer/search.go`, updated conformance and independent schema tests. Checked interactions with comparison coverage and the generated revision-2 wire shapes.

- Preflight runs before lifecycle/target dispatch, checks configured identities/plans/policies and invokes the optional side-effect-free `StructuralValidator`. Built-in callback adapters reject absent functions/ports; typed nil capabilities and uninitialized local budgets are rejected. Opaque host ports remain responsible for their internal configuration.
- `CaseAt` copies and decodes only one case. Per-attempt and per-grader decoding isolate input, reference, metadata and provenance without storing decoded domain values in a shared cache.
- `Assess` requires a fresh typed view factory; saved projections and sealed pair snapshots supply concrete implementations. Each pair direction decodes private bytes and checks cancellation before dispatch. Invalid judgments and callback errors retain valid known usage separately from abstention/undispatched outcomes.
- Assessment revision 2 validates planned identities, obtained dispatched grades and explicit skips. Worker finalization preserves earlier grades across budget/deadline/reconciliation failure and seals a canonical partial result. Rescore preserves lineage and distinguishes undispatched projection failure from a paid grader error.
- Runner stop handling records all scheduled slots, keeps setup retries separate from target effects, forbids retry after cleanup failure, and distinguishes execution/cleanup/grading/accounting states. Both lifecycle isolation modes exercise setup, cleanup, grader, budget and invalid usage failures.
- Scenario driver receives seed/mode/zero-based step. Canonical partial records retain the last successfully encoded state and `StateStep`; output/state codec failure does not claim an old state represents all executed steps. Restore rejects revision 1 and contradictory state-step records.

## Finding history

**[P2] Library-owned malformed grading ports passed preflight — CLOSED.**

Before the repair, `ValidateRunConfig` inspected grader revisions but not the library-owned `LLMGrader` configuration. A valid identity and instructions with `Port == nil` passed preflight. The standalone reproduction `/private/tmp/evaly-preflight-review.go` created a one-case public `RunConfig`, counted target calls, and supplied this invalid grader:

`GOCACHE=/private/tmp/evaly-final-1.27.1-cache go run /private/tmp/evaly-preflight-review.go`

Before: `preflight=<nil>`, `targetCalls=1`, `err=<nil>`; a sealed trial contained a dispatched `grader_error`. This contradicted task 01's rejection of inspectable structural configuration before effects. The same configuration category included missing built-in lifecycle callbacks, nil function targets, and typed nil infrastructure ports.

The implementing agent added the optional `StructuralValidator`, `ValidatePort`, built-in adapter implementations, and shared entry-point checks. On the final frozen state, the same standalone reproduction yields `preflight=evaly: invalid contract`, `targetCalls=0`, `err=evaly: invalid contract`, `trials=[]`.

Regression evidence: `TestBuiltInPortsRejectMissingCallbacksBeforeEffects`, `TestGradingPreflightsInvalidBuiltInPorts`, `TestTypedNilCodecPreflightDoesNotCallIdentity`, `TestWorkerRejectsTypedNilInfrastructure`, `TestStartRejectsInvalidBuiltInGradingPorts`, optimizer preflight tests, and `conformance.Structural`. Source inspection confirms nested built-in judge/proposer validation and optional budget/clock/ledger checks are invoked before dispatch. The reviewer did not participate in the repair.

## Executed checks

All commands below completed with exit code 0 on the final fingerprint unless expressly identified as an earlier supplemental run:

| Command | Result |
| --- | --- |
| `python3 /private/tmp/evaly-review-state.py` | Frozen fingerprint and 83-file count matched |
| `GOCACHE=/private/tmp/evaly-final-1.27.1-cache go test -race ./... -count=1` from repository root | All root-module packages passed |
| Same race command from `contracttest/` | Independent JSON Schema module passed |
| Standalone nil-LLM-port reproduction above | Configuration rejected; zero target calls |
| `GOCACHE=/private/tmp/evaly-final-1.27.1-cache go test -run '^$' -bench '^BenchmarkDatasetCaseAt$' -benchtime=10x -benchmem .` | Sizes 10/100/1000 passed; approximately 42.1µs/278µs/2.98ms, 18,004/184,020/1,883,931 B, 320/3,393/34,978 allocations for a full traversal |
| `git diff --check` | No whitespace errors |

Before the final freeze, the reviewer also ran task-01-focused race regressions three times covering preflight, isolation/decode counts, stop policy, pair cancellation, factory failures, partial assessments, scenario restoration and optimizer accounting; all passed. These are supplemental, not substituted for the final full-module checks.

The main agent's `/private/tmp/evaly-execution-validation.log` additionally records terminal `make validate` success for formatting, vet and race tests in both modules. Final root-module and contract-module race results were independently rerun by this reviewer.

## Limits

No real vendor calls were made. Scripted ports establish harness behavior, not judge reliability or provider determinism. Cancellation remains cooperative; arbitrary closures are not sandboxed and host codecs/factories must honor their declared copying/identity contracts. No durable ledger, exactly-once external effects, environment rollback or distributed scheduling guarantee was inferred. Task-02 reference filtering/HTTP issues and task-03 statistics are outside this acceptance, except where task-01 changed code directly interacts with them. Benchmark timings are observations, not flaky acceptance thresholds.

No unresolved finding remains in the task-01 implementation under the reviewed contracts and tests.
