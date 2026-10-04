# Independent correctness audit

Historical independent acceptance of the 4 October implementation. On 5 October, test filenames and symbols were renamed by behavior, dependencies updated and formatter/autofixes applied; links below use the current names. The earlier source fingerprint does not certify the updated state. Current verification is recorded in ../validation.md.

Authority: the full `.cursor/tasks/task1.md`, inspected independently of `docs/acceptance.md`. This first audit covers dataset/codec, runner/lifecycle/budget, grader/pairwise/calibration, capture/restore, comparison/denominators, artifact publication, export/HTTP, observation worker and optimizer implementations. It does not certify external LLM robustness, remote filesystem durability, or production adapters.

## Confirmed defects, first pass

1. **P1 — hash-valid malformed evidence can produce an absence pass.** `ValidateEvidence` checks only top-level version/state/hash. A sealed record claiming coverage=true despite gaps/errors, unknown event version/kind, broken sequence, or a credential-bearing reference is accepted; `AbsenceGrader` returns scored/pass. Reproduction: `TestRestoredEvidenceRejectsFalseCompleteness` in `evidence_validation_test.go`, six independently failing variants. Violates EVL-004 completeness, privacy, unknown-required-version semantics and EVL-006 restored artifact validity.
2. **P1 — capture diagnostics are unbounded.** With MaxEvents=1 and MaxBytes=512, reporting 10,000 events retains 9,999 error entries. Reproduction: `TestCaptureDiagnosticsAreBounded`. Capture must bound retained diagnostics as well as successful event payloads.
3. **P2 — scenario final step hides deadline.** A cooperative Step waits for ctx.Done then returns done=true,nil; Drive reports completed,nil. Reproduction: `TestScenarioFinalStepCannotHideDeadline`. Post-Step deadline must be checked before accepting completion.
4. **P2 — dataset restore silently repairs corrupt case revision.** Replace a CaseRecord.Revision while leaving the sealed dataset revision unchanged: RestoreDataset ignores the stored case revision, reconstructs the original and returns success. Reproduction: `TestDatasetRestoreChecksCaseRevision`. Stored immutable revisions must be validated, not repaired.
5. **P1 — duplicate online grading bypasses budget.** Capacity=1, usage per call=1, repeated enqueue of the same immutable observation: two paid grader calls, MemoryBudget.Used=1. Reproduction: `observation/TestDuplicateObservationCannotBypassGraderBudget`. Reservation idempotency cannot authorize multiple external dispatches without fresh attempt identity or deduplication.
6. **P1 — duplicate controlled run bypasses budget.** Repeating Run with the same ID and budget executes target twice under the same spent reservation; total accounting remains one unit. Reproduction: `TestRepeatedRunIdentityCannotReuseSpentBudget` in `runner_validation_test.go`. An atomic dispatch claim is needed in addition to idempotent reservation.
7. **P1 — unknown dispatched usage can be released.** Reserve -> Reconcile(Usage{Known:false}) -> Release succeeds and removes all liability. Reproduction: `TestUnknownReconciliationCannotReleaseDispatchedLiability`. Reconciliation proves dispatch even when usage is unknown; only undispatched reservations may be released.
8. **P2 — optimizer starts a holdout evaluation after budget stop.** First candidate fits capacity=1; second candidate exhausts budget; Search still claims holdout and calls Evaluate once more. Reproduction: `optimizer/TestOptimizerDoesNotDispatchAfterBudgetStop`. Ranking already collected records is allowed; starting another evaluation after stopping is not.
9. **P2 — restored experiment bypasses typed semantic validation.** Valid recalculated hash with negative target usage, unknown cleanup state, empty target provenance or reversed lifecycle states is accepted as sealed. Reproduction: `TestRestoreRejectsMalformedTrialAndManifest` in `runner_validation_test.go`, four independently failing variants. Hash integrity is not validity of runtime/state contracts.

Initial executed commands (all demonstrated the listed regression failures):

- `GOCACHE=/private/tmp/evaly-go-cache go test . -run Test -count=1`
- `GOCACHE=/private/tmp/evaly-go-cache go test ./observation ./optimizer -run Test -count=1`
- `GOCACHE=/private/tmp/evaly-go-cache go test . -run TestRepeatedRun -count=1`
- `GOCACHE=/private/tmp/evaly-go-cache go test . -run TestUnknownReconciliation -count=1`
- `GOCACHE=/private/tmp/evaly-go-cache go test . -run TestRestoreRejectsMalformed -count=1`

## Acceptance status

First pass: defects confirmed; correctness acceptance withheld. Awaiting implementation repairs and independent recheck. Successful existing tests are not evidence against these counterexamples. This report will retain defect history and add the final recheck results and limits.

Additional first-pass malformed-capability checks:

10. **P2 — nil grader panics in public Assess.** `TestAssessNilGraderReturnsTypedError` recovers a nil dereference from Assess([]Grader{nil}). Invalid grader capability must produce an explicit protocol error rather than panic; the design currently promises nil/invalid grader results as grader_error.
11. **P2 — unknown generation mode is sealed.** `TestGenerationRejectsUnknownMode`: Generator="g1", Mode="unknown", LabelValidated=true successfully seals. The declared generation enum is replay/search; unknown protocol modes must be rejected.

Executed `GOCACHE=/private/tmp/evaly-go-cache go test . -run 'TestAssessNil|TestGeneration' -count=1`: both counterexamples failed as expected.

12. **P2 — absolute online deadline is extended by timer registration race.** Worker computes remaining from Clock.Now and registers a relative Clock.After; advancing the injected clock past the absolute deadline between those calls still dispatches the grader and returns graded. `observation/TestWorkerCannotExtendAbsoluteDeadlineDuringTimerRegistration` deterministically simulates that boundary. It must recheck the absolute injected-clock expiry before dispatch and final outcome; ctx cancellation alone is insufficient.

Executed `GOCACHE=/private/tmp/evaly-go-cache go test ./observation -run TestWorkerCannotExtend -count=1`: result=graded and one grader call despite expired absolute deadline.

13. **P2 — equivalent assessment content has inconsistent identity.** Worker hashes json.Marshal field order while core Rescore hashes sorted canonical JSON. Identical assessment fields/grades/source/view/parent/mode produce different Revision values depending on authoring path. `observation/TestWorkerAndOfflineUseSameAssessmentDigest` reproduces the difference. Identity must use one canonical wire representation.

Executed `GOCACHE=/private/tmp/evaly-go-cache go test ./observation -run TestWorkerAndOffline -count=1`: worker digest `29387464ddacecdabad540b938832a2a79448aaf94109c977c785a21b5af73c5`, offline digest `b74aeb348309b89f6fa4b4df187bf82a1fce331668a6b501f6f2a8678db50964` for equivalent content.

## Independent recheck after repairs

All thirteen confirmed defects above are closed by implementation changes and the unchanged independent regression assertions. The optimizer wrapper was adapted only to the new typed `EvaluationRequest` signature. That request now supplies distinct search/phase/candidate experiment identity, preventing reuse of reservations across searches. Core and optional adapters were reread at the repaired budget/dispatch, restore, capture, deadline and digest boundaries.

Verified final commands:

- `GOCACHE=/private/tmp/evaly-go-cache go test ./... -count=1` — PASS, including independent audit regressions, core fixtures, HTTP adapter, conformance suite, observation/optimizer and CLI subprocess suite.
- `GOCACHE=/private/tmp/evaly-go-cache go test -race ./... -count=1` — PASS; no race diagnostics in exercised paths.
- `GOCACHE=/private/tmp/evaly-go-cache go vet ./...` — PASS.
- `GOPATH=/private/tmp/evaly-toolchains GOTOOLCHAIN=go1.26.1 GOCACHE=/private/tmp/evaly-go-cache go test ./... -count=1` — PASS.
- `rg --files -g '*.go' -0 | xargs -0 gofmt -l` — no unformatted Go files.
- `go version`, `GOTOOLCHAIN=local go version` — actual local compiler `go1.26.5 darwin/arm64`; explicitly selected compiler `go1.26.1 darwin/arm64` confirmed separately. No claim that this auditor ran Go 1.27.1.

Final conclusion: **в рамках проведённой проверки дефектов не выявлено**. There are no unresolved confirmed defects from this audit.

Limits: finite source review and executable adversarial cases do not prove absolute correctness. Callback bounds require cooperative context handling; external effects cannot be killed or undone by timeout. Memory budgets/holdout ledger are process-local. Filesystem publication was exercised on the local supported environment, not network filesystems or multi-host storage. Scripted judges prove protocol behavior, not real model accuracy or injection immunity. HTTP integration uses local test servers, not vendor SDKs or credentialed production systems. Bootstrap inference retains its documented independent-case and multiple-comparison assumptions. Privacy correctness additionally depends on host classification of allowlisted projections. Later changes outside the inspected source state require a new audit.

## Final expanded-state audit — additional counterexample

The final review additionally inspected the new portable SavedView publication/restore path, encoded scenario per-step records/lineage, comparison trial replay references, explicit shared adapter conformance suites, independent JSON Schema validator module and the additional acceptance fixtures. The previous clean statement applies to its earlier source state; acceptance is temporarily withheld for this expanded state pending the defect below.

14. **P2 — scenario restore and lineage admit contradictory trajectory semantics.** Both RestoreScenario and DraftFromScenario accept valid recomputed hashes for completed-with-zero-steps, completed-with-missing-output, step_limit-before-max_steps, generation-mode mismatch and missing generator identity. `TestScenarioRestoreRejectsContradictoryTrajectory` in `scenario_evidence_validation_test.go` reproduces all five variants. Output count and terminal stop must agree with performed steps; generation identity/mode must obey the same contract as RunScenario. Restore and lineage must share semantic validation in addition to integrity checks.

Executed `GOCACHE=/private/tmp/evaly-go-cache go test . -run TestScenarioRestore -count=1`: all five variants failed the rejection assertion.

## Final expanded-state recheck — accepted

Defect #14 is closed. The shared `validateScenario` is applied by both restore and draft-lineage paths, validates terminal steps/output cardinality, generation identity/mode, codec bounds, raw canonical JSON and digest. All five independent counterexamples now pass their rejection assertions without changes. Restore also copies generation metadata.

The inspected expanded state retains all thirteen earlier audit regressions. Portable SavedView codecs/publication/reopen and zero-view rejection were reviewed; comparison trial evidence/replay references, exact CLI fixture behavior/seed replay, added acceptance fixtures, generated schemas, independent validator and shared protocol conformance suites were inspected. The runner's codec failure now produces an explicit setup record; context is rechecked before target dispatch. Worker finite-unit validation and post-Claim absolute clock check were inspected.

Final independent commands after the scenario repair:

- Root: `GOCACHE=/private/tmp/evaly-go-cache go test ./... -count=1` — PASS.
- Root: `GOCACHE=/private/tmp/evaly-go-cache go test -race ./... -count=1` — PASS.
- Root: `GOCACHE=/private/tmp/evaly-go-cache go vet ./...` — PASS.
- Root: `GOPATH=/private/tmp/evaly-toolchains GOTOOLCHAIN=go1.26.1 GOCACHE=/private/tmp/evaly-go-cache go test ./... -count=1` — PASS.
- `contracttest/`: `GOCACHE=/private/tmp/evaly-go-cache go vet ./...` — PASS.
- `contracttest/`: `GOCACHE=/private/tmp/evaly-go-cache go test ./... -count=1` — PASS.
- `contracttest/`: `GOCACHE=/private/tmp/evaly-go-cache go test -race ./... -count=1` — PASS after repair.
- `contracttest/`: `GOPATH=/private/tmp/evaly-toolchains GOTOOLCHAIN=go1.26.1 GOCACHE=/private/tmp/evaly-go-cache go test ./... -count=1` — PASS.
- All root and nested module Go files: `rg --files -g '*.go' -0 | xargs -0 gofmt -l` — empty output.

Inspected-source fingerprint: `af6166bfa9d94c84db85b4b330ce8e12d1ef7e3533b576a601b5748e78206260`, covering 68 files: sorted relative paths and contents of all Go source/tests, wire schemas, fixture catalog, root/contracttest module files, design and task source. Each path and file content is prefixed with its eight-byte big-endian length before SHA-256. Audit report text itself is excluded.

Current conclusion: **в рамках проведённой проверки дефектов не выявлено**. All fourteen confirmed audit defects are closed. The explicit limits in the earlier recheck remain applicable; this auditor independently ran local Go 1.26.5 and selected Go 1.26.1, while any Go 1.27.1 evidence is reported separately by the implementing agent.
