# Task 2 execution plan

Source: `.cursor/tasks/task2-evaly-review-remediation.md`, reviewed baseline
`76c224a97e5f03a44e87ab7ea308216a7e10d139`. This plan covers the entire source
task, including its documentation checklist and Definition of Done.

## Execution and acceptance

Execute the following tasks in order. Only one implementation task is active.
Before advancing, run relevant checks and commission two independent agents
that did not implement the task, both reviewing the same frozen diff: completeness
(fulfilled mandatory criteria / total mandatory criteria × 100) and correctness
(contract, regression tests, edge cases, findings and verification limits).
Acceptance requires 100% completeness and no unresolved confirmed errors.
After any correction, both agents review the new final diff again. Record their
reports and commands in `docs/remediation-verification.md`, then commit only the
accepted task with a short English conventional message. Do not push or release.

Every numbered acceptance criterion below is mandatory. Source AAA scenarios
are incorporated by reference, not replaced by these condensed criteria.
For every F item preserve a behavioral reproduction against the baseline and
an AAA regression for the new behavior; compilation failure is not a repro.
For every D item record the final decision, reason and implementation/doc evidence.
The mapping below assigns each D exactly once; related tasks may depend on it.

## Sequential checklist

- [x] T01 — Spec-First and execution plan. F03–F09 contracts; D23/D24/D28
  are specified here and implemented in their assigned tasks.
  1. All F01–F10 and D01–D60 have explicit task ownership; all source DoD and
     documentation requirements are assigned.
  2. Numeric, JSON/ownership, preflight and optimizer state contracts are
     explicit in `docs/remediation-contracts.md`, with rationale and boundaries.
  3. Acceptance/rework/commit protocol and verification record exist. This is
     a proposed implementation contract; existing code is not claimed compliant.
- [x] T02 — Release safety: F01/F02 (together), release portability docs.
  1. Disposable release checkout, exact generated-file allowlist and exact
     root-only tag refspec; original HEAD/index/worktree/local refs unchanged.
  2. Local fixtures cover happy path, unrelated untracked files/tags, rejecting
     push, preparation/tag failures, retry and explicit detached-start policy.
  3. Publication ambiguity and recovery are explicit; no blind deletion of
     remote refs; atomic publication policy and Linux/macOS editing documented.
- [x] T03 — Budget: F03; D02/D03/D46.
  1. Exact liability and conservative Used reporting satisfy both baseline
     repros and mixed-scale settlement/release permutations without epsilon.
  2. Idempotency, claim-once, unknown liability and concurrent accounting pass.
  3. Public accounting/host recovery semantics and migration limits documented.
- [x] T04 — Codec: F04/F05; D23/D24.
  1. Generic numbers, typed overflow and lexical identity pass all source cases
     and supported dataset/snapshot roundtrips.
  2. Aliases and ignored fields follow declared semantics; genuine cycles fail;
     UTF-8/custom marshaler limits and byte ownership are tested/documented.
  3. Targeted canonicalization edge/race/fuzz checks and codec migration recorded.
- [x] T05 — Preflight/grading/storage: F06/F07; D08/D20/D21/D27/D28.
  1. Restore codecs and public store helpers reject nil/typed nil/invalid ports
     before dispatch; unsupported nonnil codec identity remains distinct.
  2. Invalid AbsenceGrader predicate/kind fails ValidatePort, Assess, Run and
     observation preflight without effects, regardless of evidence contents.
  3. Valid roundtrips, absence/coverage behavior and BYOT view ownership remain;
     constructor migration and correctly placed GoDoc are recorded.
- [x] T06 — Optimizer artifacts: F08/F09; D39–D45/D47–D50.
  1. Live/restore share semantic validation of all declared state relationships;
     five independently rehashed F08 mutations fail both validators.
  2. Calibration and holdout wrong-ID outputs with/without callback error stop
     dispatch and produce restorable diagnostics; valid bound partials survive.
  3. Unchanged, cancelled, budget-stopped, duplicate and encoding-rejected
     histories pass; derived fields, lineage, static proposal limits, provenance,
     reason precedence and diagnostic bounds have explicit decisions/evidence.
  4. Relevant semantic mutation/race/fuzz checks, schemas/revisions and migration
     are synchronized; no holdout-based reselection or deploy permission.
- [x] T07 — Conformance: F10; D04/D51/D57–D59.
  1. Subprocess lifecycle fixtures prove exactly-once bounded cleanup after
     successful/partial Prepare, failed Reset and Goexit, including cleanup error.
  2. Budget suite checks claim-once/concurrent claim, reserve idempotency/conflict,
     release before/after claim and unknown usage, and rejects a no-op Claim host.
  3. Numeric schema boundary mutations use exact tokens; independent semantic
     negatives remain; fixture namespace/timeouts/cancellation limits explicit.
- [ ] T08 — HTTP and CLI: D52–D56.
  1. Pre-cancelled handler avoids callback effects; static invalid handler setup
     fails construction; request errors retain appropriate runtime classification.
  2. Capability flags and MaxBytes guarantees have explicit decisions/docs;
     irrelevant CLI flags and unknown commands fail before creating a store.
  3. Adapter regression tests, example/API/schema migration stay synchronized.
- [ ] T09 — Local cleanup and measurement contracts: D01/D05–D07/D09–D19/D22/D25/D26/D29–D38/D60.
  1. Each assigned decision has reason/evidence; recommended changes are either
     implemented with regression coverage or explicitly rejected with justification.
  2. Naming/identity, policy ownership, safe diagnostics and callback/config
     serialization are aligned; behavior, gate separation and host boundaries stay
     explicit. Helpers simplify locally without a workflow DSL or lint relaxation.
  3. D33 is measured before optimizing; record commands/data and decision. No
     mutable decoded values are cached. Other speculative optimizations require
     demonstrated need rather than expanding scope into infrastructure.
- [ ] T10 — Documentation and integrated acceptance: all source documentation
  requirements and full Definition of Done; cross-check all F/D decisions.
  1. Release notes/current status (including `.cursor/task/README.md` and sibling
     `ai-libs/feature-specs/evaly.md`), public GoDoc, runnable end-to-end quickstart,
     concurrency matrix and authoritative migration guide match the final code.
     Historical reports remain dated evidence, not current guarantees.
  2. `make validate` with pinned lint passes both modules and seven examples;
     independent schema regeneration/diff, corpus and semantic negatives pass;
     addressed edge/race/fuzz and release fixtures have recorded evidence.
  3. Complete requirement-by-requirement DoD audit, F repro/regression inventory,
     D01–D60 decision inventory and durable command/outcome/SHA/limits records.
     BYOT independence and stated verification limits remain intact.
  4. Two independent final agents accept 100% completeness and no unresolved
     confirmed errors; all task changes committed. Goal completes only then.

An external file permission issue does not justify silently dropping a document:
prepare the exact edit and obtain required filesystem authorization when needed.
