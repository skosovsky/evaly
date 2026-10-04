# TASK01 independent completeness acceptance

Reviewer: `accept_execution_completeness`, independent of implementation. Authority:
the entire `.cursor/task/01-execution-and-assessments.md` and its README. Source
implementation was read only. Final independently computed fingerprint:
`0be19a748c94a25fa740af4310fb5ef80bf3a3f59f8882685500b1c869316126`
(83 files). Historical 171-row acceptance is not evidence here.

The fixed denominator is **55 conjunctive rows**: 35 numbered-requirement rows,
all nine acceptance rows, and 11 common README rows. Every subclause belongs to a
row; a row receives no credit if any of its clauses is incomplete. No denominator
reduction is permitted. Every row below was rechecked on the frozen tree.

| ID | Required result (all clauses required) | Current evidence | Status |
|---|---|---|---|
| N1a | Codec/port identities validated structurally | preflight.go, Snapshot.Validate, worker.Start, Search | complete |
| N1b | Policy/projection/provenance revisions checked | ValidateRunConfig, NewCapture, validateProvenance | complete |
| N1c | Plans, enum values, limits checked | validatePlan, RunScenario, worker.Start, Search | complete |
| N1d | Unique complete grader revisions | ValidateGraderRevisions; Run, Assess, Rescore, Start | complete |
| N1e | Config consistency before Prepare/Reset/target/judge/proposal | ValidateRunConfig, RunPaired, Rescore, Start, Search; StructuralValidator rejects nil built-in callbacks and typed nil ports before effects | complete |
| N1f | Callback errors retained as runtime failures | codecFailure, Assess, runTrial, worker.evaluate, RunScenario | complete |
| N1g | No paid validation probes | preflight reads metadata; tests count actual callbacks | complete |
| N2a | Point immutable case access | Dataset.CaseAt | complete |
| N2b | Fresh domain values per target attempt and grader | worker loop, runTrial factory | complete |
| N2c | No unrelated case decoding in per-trial loop | CaseAt, TestRunLinearCaseDecodingAndIsolation | complete |
| N2d | No shared mutable decoded cache | private serialized Dataset; CaseAt private record clone | complete |
| N2e | Full per-case enumeration and ignored decode errors removed | runner.go uses CaseAt and records codec failures | complete |
| N3a | One explicit typed isolated grading input contract | Assess factory; Snapshot codecs for pair inputs | complete |
| N3b | Applied to public Assess and both pair directions | grader.go and snapshot.go | complete |
| N3c | Unsafe old signatures removed, consumers updated | conformance, examples, test consumers compile | complete |
| N3d | Factory receives no target capability from evaly | factory signature, SavedView.View | complete |
| N3e | Sanitizing projection remains host boundary | runTrial calls Project; SavedView stores permitted values | complete |
| N4a | Planned graders declared | Assessment.Planned and manifest.Graders | complete |
| N4b | Obtained results retained | Rescore append, worker append before reconciliation | complete |
| N4c | Explicit skipped graders/reasons and assessment completeness | Skipped, State, StopReason; trial grading fields | complete |
| N4d | Results and valid usage survive subsequent budget error | TestPartialAssessmentRetainsPaidResults/budget | complete |
| N4e | Results and valid usage survive deadline/reconciliation | same test deadline/reconcile; cancelled-preflight regression | complete |
| N4f | Canonical revision/lineage; partial not successful measurement | SealAssessment, ValidateAssessment; trial partial validation | complete |
| N4g | Undispatched vs abstention vs unknown usage distinct | Grade.Dispatched, SkippedGrader, pair dispatch flags, Usage.Known | complete |
| N5a | Context checked before each pair dispatch; no second after cancel | CheckPair; TestPairSnapshotsAndCancellationPreserveUsage | complete |
| N5b | Valid usage preserved on error/invalid preference | CheckPair retains response usage while clearing invalid preference | complete |
| N5c | Dispatch liability never released by reconciliation | runTrial/worker/Search claim/reconcile; unknown-liability tests | complete |
| N6a | Setup/cleanup/grader/usage stop policy explicit | docs/execution-contract.md; trialInfrastructureFailure | complete |
| N6b | Quality, execution, cleanup remain separate | Grade, TrialRecord.Status, CleanupStatus, grading fields | complete |
| N6c | In-flight callback cooperative cancellation | WithCancelCause and propagated contexts; documented limits | complete |
| N6d | Undispatched slots retain explicit stop reason | full scheduled jobs retained, InfrastructureStop | complete |
| N6e | No automatic retry of effects; setup retry avoids contamination | only SetupError retry, cleanup-failure guard | complete |
| N7a | Typed seed/mode/step supplied to driver | ScenarioContext and RunScenario driver.Step | complete |
| N7b | Seed available without provider determinism promise | execution contract; context regression test | complete |
| N7c | Scenario identity and conformance updated | scenario v2 canonical hash, conformance.Scenario, examples | complete |
| A1 | Empty output codec and other preflight errors produce zero callbacks | TestRunPreflightBeforeEffects; Assess/Search preflight tests | complete |
| A2 | Post-callback failure preserves valid results/reasons | paid partial worker tests; scenario state-codec failure record | complete |
| A3 | 2 graders/budget 1 preserves first; deadline/reconcile equivalent | TestPartialAssessmentRetainsPaidResults; canonical validation | complete |
| A4 | Mutations cannot reach caller or next grader/pair direction | linear decode test, SavedView mutation test, pair snapshot test | complete |
| A5 | Cancelled pair 0 calls, after first 1; known usage retained | TestPairSnapshotsAndCancellationPreserveUsage | complete |
| A6 | Linear decode counting and multi-size allocation/time benchmark | N=1/8/32 test; BenchmarkDatasetCaseAt N=10/100/1000 | complete |
| A7 | Mutation/retry/cleanup/unknown usage/cancel preserve invariants; both isolation modes stop-policy tested | mutation/unknown liability/cancellation tests; stop matrix now includes setup/cleanup/grader/budget/invalid usage in both isolation modes | complete |
| A8 | Driver seed/step; replay restoration cannot execute effects | TestScenarioExecutionContextAndRestore; replay-lineage tests | complete |
| A9 | Schemas/restore/validation/conformance/worker/optimizer/examples synchronized | v2 schemas + schemagen, Restore*, independent contracttest | complete |
| C1 | Clear break; all affected consumers replaced; no aliases/shims/readers/dead fields | factory/snapshot API; v1 replaced by v2 affected formats | complete |
| C2 | Normative design and contracts cover input/result/state/error/ownership/identity/cancel/limits | design.md and execution-contract.md | complete |
| C3 | Breaking wire gets explicit new version; old semantics unsupported; fixtures updated | experiment/assessment/scenario/observation-result/search v2; restore rejects old | complete |
| C4 | BYOT/codecs; no mandatory agent/message/prompt/map model or reflection domain clone | generics, explicit codecs, typed factory; service envelopes only | complete |
| C5 | Core stdlib, no network/vendor/credentials/neighbor modules; optional packages outside core graph | go.mod and core imports; optional package dependency direction | complete |
| C6 | No runtime/dashboard/scheduler/distributed storage/deploy/prompt optimizer; host policies retained | inspected changed APIs and normative ownership boundary | complete |
| C7 | No false forced-stop/exactly-once/no-secrets/independence/judge-accuracy promises | README/design/execution contract explicit limitations | complete |
| C8 | AAA behavioral adversarial/failure regressions | new execution, assessment, partial tests use Arrange/Act/Assert | complete |
| C9 | New contracts have reference implementation and conformance; no TODO ports | built-in StructuralValidator references; conformance.Structural exercises valid/zero/typed-nil/missing-callback ports repeatedly without budget consumption; Grader/PairJudge/Scenario suites updated | complete |
| C10 | Current checks for all affected packages/contracttest, concrete commands; no historical PASS or weakened tests | independently executed make validate exit 0, benchmark PASS, all six examples exit 0; current logs only | complete |
| C11 | TASK01 boundaries respected; no TASK02 outcome/evidence or TASK03 objective redesign; no cache/ignore/style scope creep | inspected diff and task boundaries | complete |

Final result: **55/55 (100%)**, no incomplete requirement rows. Preliminary gaps
N1e, A7 and C9 were closed through structural preflight, reusable conformance and
the two-isolation usage-failure matrix. C10 was closed by independent checks on
the frozen state. Percentage describes completeness against this entire task,
not absolute absence of software defects; correctness has a separate reviewer.

Independent checks on the final fingerprint:

- `python3 /private/tmp/evaly-review-state.py` — hash above, 83 files. Helper scope
  inspected: tracked and ordinary untracked implementation/tests/schema/docs plus
  task README and TASK01, excluding review outputs and validation journal.
- `GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate` — exit 0; formatter,
  vet and race tests for both modules. Independent full log:
  `/private/tmp/evaly-completeness-validate.log`.
- Targeted race check of preflight, Assess, pair snapshots, stop matrix and both
  conformance suites — exit 0.
- All six `go run ./examples/{calculation,crm,protocols,http,observation,optimizer}`
  commands — exit 0. Deliberate quality-failure examples remain failures in their
  reports, as intended.

Independent diagnostic benchmark executed:

`GOCACHE=/private/tmp/evaly-final-1.27.1-cache go test -run '^$' -bench '^BenchmarkDatasetCaseAt$' -benchmem -benchtime=1x .`

PASS on the final fingerprint. N=10/100/1000: 67,583/352,292/2,701,083 ns/op;
18,000/184,144/1,881,744 B/op; 320/3,394/34,987 allocs/op. No timing threshold is
asserted. One-shot timings are diagnostic only; the deterministic decode-count
test proves the linear domain-decode requirement.

Completeness acceptance applies only to this frozen implementation fingerprint.
It must match the independent correctness report before task acceptance/commit.
Any implementation repair invalidates the final acceptance until independently
rechecked. TASK02 evidence/OutcomeRefs and TASK03 objectives are excluded only
because TASK01 explicitly assigns them to later tasks, not to reduce scope.
