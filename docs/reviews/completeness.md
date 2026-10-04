# Independent completeness acceptance

Historical independent acceptance of the 4 October implementation. On 5 October, test filenames and symbols were renamed by behavior, dependencies updated and formatter/autofixes applied; links below use the current names. The earlier source fingerprint does not certify the updated state. Current verification is recorded in ../validation.md.

Final result: **171/171 = 100%**. Authority: the full `.cursor/tasks/task1.md`, read independently; the initial matrix denominator remains 171. Partial requirements receive zero credit. No original row was removed or narrowed. The ten mandatory synthetic fixtures are scenarios over these requirements and receive no extra percentage credit. No unresolved completeness gap remains in the inspected implementation.

| Group | Complete / total | Completeness |
|---|---:|---:|
| BOOT | 19/19 | 100% |
| EVL-001 | 21/21 | 100% |
| EVL-002 | 22/22 | 100% |
| EVL-003 | 21/21 | 100% |
| EVL-004 | 25/25 | 100% |
| EVL-005 | 19/19 | 100% |
| EVL-006 | 24/24 | 100% |
| EVL-007 | 20/20 | 100% |

## Evidence inspected

- **Bootstrap:** module path and minimum Go1.26.1; stdlib-only core dependency graph; README/package docs/design/acceptance, 14 public wire schemas, separate contracttest module, Makefile and pinned CI. BYOT and host boundaries are implemented in typed ports. BOOT-04 chronology is process evidence from the implementation tool-call journal: design.md and the first envelope schema were saved before Go implementation mutation. Temporal order is not inferred from mutable mtimes.
- **EVL-001:** private canonical case/dataset snapshot bytes, codec and domain identities, versioned generation lineage, host validation, duplicate/malformed rejection, restored case-revision validation, two distinct typed fixtures, bounded versioned scenario driver, saved search trajectory/replay and draft derivation. Sources: dataset.go, codec.go, scenario.go; examples/calculation, crm and protocols.
- **EVL-002:** typed lifecycle handles, independent preparation/reset and cleanup, serial/shared execution, attempts/repeat identities, bounded dispatch/deadline/cancellation, atomic reservations plus dispatch claim, retained unknown usage and side effects. Sources: runner.go, budget.go; acceptance/core/protocol/conformance and independent budget audit tests.
- **EVL-003:** deterministic/outcome/trajectory/LLM reference paths; separate grader statuses, named scales/direction, required assertion policy, revisions, trusted rubric vs untrusted content, blind pair order/abstention/disagreement and calibration denominators. Sources: grader.go, fixtures/crm and protocol/acceptance/conformance tests. Scripted fixture does not certify real-model robustness.
- **EVL-004:** per-trial bounded capture, projected secrets before persistence/judge/export, semantic validation of restored completeness, replay availability, immutable permitted views, FileStore reopen/re-score without target/tools, observation lineage/sampling, bounded nonblocking worker and fake deadline/overload/flush/cancel. Sources: evidence.go, rescore.go, observation/worker.go; examples/observation and independently retained audit regressions.
- **EVL-005:** immutable planned/running/sealed/incomplete experiment history and provenance; compatible comparisons; explicit eligible/scored/excluded denominators and setup/target/grader/cleanup counters; independent cases with repeat averaging; named paired case bootstrap; separate repeat success metrics; versioned threshold reasons; rescore immutability. Sources: runner.go, comparison.go and denominator/paired/restore audit tests.
- **EVL-006:** staged/validated/committed local FileStore with checksum, immutable identity/dedup retry, reopen/fault injection; portable artifacts and declared codecs; reports contain case/trial/evidence and accurate replay guidance; subprocess pass/fail/inconclusive/infra exits; export delivery separated from verdict; boolean-loss and required-version rejection; real local HTTP JSON request/response with body/context bounds; compatible envelope extensions preserved. Sources: artifact.go, export.go, comparison.go, cmd/evaly, adapters/httpjson; contracttest independently validates nested wire schemas.
- **EVL-007:** typed immutable candidate enumeration and proposal reference port; disjoint splits, proposal excludes holdout, typed evaluation requests bind fresh search/phase/candidate identities, budgets account for proposal/target/graders, failed/incomplete history retained and excluded from ranking, stop prevents holdout dispatch, holdout ledger detects repeated selection, baseline/holdout comparisons and lineage retain no-improvement results. Sources: optimizer/search.go, its tests/audit regressions and examples/optimizer.

The reusable conformance suite executes artifact, budget, codec, target, lifecycle, grader, export, capture policy/evidence, generator, scenario, judge/pair judge, proposal, holdout ledger and clock reference paths. Optional packages are isolated from core imports; the separate schema validator module is test-only. No vendor SDK capability is claimed.

## Ten required synthetic fixtures

| Fixture | Result | Executable evidence |
|---|---|---|
| 1 | complete | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds; TestPairedCaseRevisionsAndIndependentEnvironments; TestScenarioSearchTrajectoryAndReplayLineage; TestDatasetRestoreChecksCaseRevision. |
| 2 | complete | TestCRMTextDoesNotOverrideOutcome; TestEvidencePrivacyGapAndJudgeError. Failed outcome, grader timeout and insufficient trajectory remain distinct. |
| 3 | complete | TestRunResetFailureAndCleanup; TestPairIsolationAndAttempts; TestPairedCaseRevisionsAndIndependentEnvironments; TestSharedFixtureActuallySerial. |
| 4 | complete | TestSecretAbsentInJudgeExportAndVerdictUnchanged; TestEvidencePrivacyGapAndJudgeError; TestSavedViewReopenAndRescoreWithDeclaredCodecs; TestOfflineRescoreRevisionAndNoTarget. |
| 5 | complete | TestBudgetBlocksThirdRealDispatchAndCancellationRetainsEffects; TestBudgetUnknownUsageAndCancellation; conformance.Budget; retained independent repeated-dispatch/unknown-liability regressions. |
| 6 | complete | TestComparisonCoverageAndCaseDenominator: partial 2/4 is inconclusive; full 3/4 denominator 4; repeated observations preserve n=4 cases. |
| 7 | complete | TestBlindPairOrderAndCalibration; TestPairAbstentionIsVisible; conformance.PairJudge. Blind A/B, both orders, disagreement/abstention and unchanged trusted instructions asserted. |
| 8 | complete | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestSecretAbsentInJudgeExportAndVerdictUnchanged; TestHTTPRejectsLossAndUnknownVersion. |
| 9 | complete | TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade; independent absolute-deadline and identical-digest regressions. |
| 10 | complete | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination; TestOptimizerDoesNotDispatchAfterBudgetStop. |

## Independent executed final checks

After the final scenario semantic repair, this completeness auditor executed:

- `env GOCACHE=/private/tmp/evaly-go-cache GOPATH=/private/tmp/evaly-toolchains GOTOOLCHAIN=go1.26.1 go test ./... -count=1` — PASS, including all five unchanged scenario restore counterexamples and existing independent regressions.
- Same environment `make validate` — PASS: root and contracttest go vet, go test, go test -race and repository-wide gofmt check, including CLI subprocesses and local HTTP tests.

In the preceding repair recheck, this auditor also ran uncached contracttest normal tests (PASS) and the changed protocols/observation examples (both exit0). The parent's docs/validation.md records actual executions of all six examples and explicitly selected Go1.26.5/1.26.1/1.27.1 gates; this auditor does not claim to have personally executed 1.27.1. Go1.26.1 minimum compatibility is directly independently verified.

## Repair history

Initial audit: 150/171 (87.72%). G1–G10 identified missing scenario provenance/replay, wire schemas, protocol conformance, persisted offline re-score, explicit synthetic assertions, accurate report/replay, malformed evidence/dataset restoration, bounded capture diagnostics and final validation evidence. Each is closed by inspected implementation plus executable checks listed above.

Second audit: 168/171 (98.25%). G11 remained: checksummed contradictory scenario trajectories were accepted. Shared `validateScenario` now checks stop-vs-step/output cardinality, mode/generator identity, codec IDs/bounds, canonical payloads and digest. Both RestoreScenario and DraftFromScenario use it. Five unchanged independent variants (completed zero steps/missing output, early step_limit, generation mode/identity) pass after repair.

This percentage measures fulfilment of the requested specification, not proof of absence of all possible defects. Correctness has its own independent report. Callback bounds require cooperative contexts; privacy requires host classification; filesystem guarantees are local; budgets/ledger memory references are process-local; real LLM accuracy, remote storage and production deployment remain explicitly unverified/outside the requested scope. No commits, release or production actions are part of this acceptance.

## Final disposition of all original atomic requirements

Detailed implementation/test/example locators remain in docs/acceptance.md. Every requirement below was assessed against the source, including its full original card and invariants.

| ID | Requirement | Final disposition |
|---|---|---|
| BOOT-01 | Go module github.com/skosovsky/evaly | complete |
| BOOT-02 | Pinned verified toolchain compatible with local libraries and CI | complete |
| BOOT-03 | README and package docs | complete |
| BOOT-04 | Design precedes public API implementation | complete |
| BOOT-05 | Public wire schemas and separate domain schema identities | complete |
| BOOT-06 | Acceptance matrix for every card and invariant | complete |
| BOOT-07 | Runnable examples and testdata | complete |
| BOOT-08 | Shared conformance suites for implemented ports | complete |
| BOOT-09 | Core runs without ai-libs/harness/network/API keys/telemetry | complete |
| BOOT-10 | Optional adapters isolated from core dependency graph | complete |
| BOOT-11 | BYOT input/output/reference/environment and no mandatory map domain API | complete |
| BOOT-12 | Host owns auth/credentials/storage/deployment/scheduler/production policy | complete |
| BOOT-13 | I/O propagates context; worker counts and queues bounded | complete |
| BOOT-14 | Unknown and unsupported are explicit errors/outcomes | complete |
| BOOT-15 | No exactly-once external effect or unverified durable guarantees | complete |
| BOOT-16 | AAA behavioral/adversarial/fault tests | complete |
| BOOT-17 | No legacy shims or hidden scope deferral | complete |
| BOOT-18 | Formatting, vet, normal tests and race checks | complete |
| BOOT-19 | Explicit limitations and real validation report | complete |
| EVL-001-01 | Stable case IDs and typed inputs | complete |
| EVL-001-02 | Optional references and explicit grader reference requirement | complete |
| EVL-001-03 | Metadata/evidence requirements are identity-bearing | complete |
| EVL-001-04 | Dataset fixes ordered case revisions and selection policy | complete |
| EVL-001-05 | Codec/schema identity is part of dataset and case digest | complete |
| EVL-001-06 | Draft validation before sealed runnable snapshot | complete |
| EVL-001-07 | Draft mutations cannot alter sealed snapshots | complete |
| EVL-001-08 | Snapshot access returns independently decoded copies | complete |
| EVL-001-09 | Duplicate IDs reject including same payload | complete |
| EVL-001-10 | Noncanonical/nonserializable payload rejects | complete |
| EVL-001-11 | Input/reference/metadata/composition changes create revision | complete |
| EVL-001-12 | Labels/aliases cannot replace immutable revision | complete |
| EVL-001-13 | Domain migration creates new revision with parent | complete |
| EVL-001-14 | External fixture version or nonreproducible marker | complete |
| EVL-001-15 | Typed target receives case input and trial context | complete |
| EVL-001-16 | Target output/status/usage/outcome evidence are distinct | complete |
| EVL-001-17 | Two materially different typed targets run without harness | complete |
| EVL-001-18 | Host scenario driver has bounded steps/turns/time and versioned behavior | complete |
| EVL-001-19 | Generator creates unvalidated drafts with parent/generator/model/seed | complete |
| EVL-001-20 | Self-assigned generated labels cannot bypass host validation | complete |
| EVL-001-21 | Adversarial search and scenario replay modes/trajectory lineage distinct | complete |
| EVL-002-01 | Run plan fixes repeats/concurrency/deadline/stop/limits | complete |
| EVL-002-02 | Lifecycle prepares namespaces and typed environment handles | complete |
| EVL-002-03 | Readiness/reset precedes dispatch | complete |
| EVL-002-04 | Failed reset never invokes target | complete |
| EVL-002-05 | Queued/preparing/running/collecting/terminal history | complete |
| EVL-002-06 | Completed/target error/setup error/cancelled/budget exhausted statuses | complete |
| EVL-002-07 | Cleanup bounded with independent status/error | complete |
| EVL-002-08 | Cleanup handles partially prepared environment | complete |
| EVL-002-09 | Every repeat has stable unique identity | complete |
| EVL-002-10 | Infrastructure retry has distinct attempt; prior failures retained | complete |
| EVL-002-11 | Unisolated fixtures force serial/shared and cannot claim independence | complete |
| EVL-002-12 | Cancellation stops new target dispatch and forwards context | complete |
| EVL-002-13 | Cancellation cannot imply effect rollback | complete |
| EVL-002-14 | Hard cap uses atomic host reservation before dispatch | complete |
| EVL-002-15 | Capacity 2, cost 1 blocks third dispatch | complete |
| EVL-002-16 | Unknown actual usage retains reservation | complete |
| EVL-002-17 | Known usage is separate observation with reconciliation errors | complete |
| EVL-002-18 | Seed fixed without provider determinism claim | complete |
| EVL-002-19 | Paired sides get same cases/fixture/reset and separate environments | complete |
| EVL-002-20 | Paired order/randomization and timestamps recorded | complete |
| EVL-002-21 | Incompatible shared baseline state invalidates independent pair | complete |
| EVL-002-22 | Reset/retry changes alter experiment identity | complete |
| EVL-003-01 | Common typed grader port receives only permitted view | complete |
| EVL-003-02 | Deterministic grader reference path | complete |
| EVL-003-03 | Outcome grader reads independent environment outcome | complete |
| EVL-003-04 | Trajectory absence checks require complete evidence | complete |
| EVL-003-05 | LLM port has scripted reference adapter | complete |
| EVL-003-06 | Scored/not applicable/insufficient evidence/grader error distinct | complete |
| EVL-003-07 | Judge timeout is grader error without zero metric | complete |
| EVL-003-08 | Named scores retain scale and direction | complete |
| EVL-003-09 | Named assertions/reason codes/bounded evidence refs | complete |
| EVL-003-10 | Grader revision includes implementation/rubric/model/prompt/config | complete |
| EVL-003-11 | LLM usage/provenance separate from target | complete |
| EVL-003-12 | Human calibration labels measure disagreement and coverage | complete |
| EVL-003-13 | Calibration labels do not imply certified accuracy | complete |
| EVL-003-14 | Explicit conflict policy preserves all grader results | complete |
| EVL-003-15 | Judge instructions separate from untrusted candidate content | complete |
| EVL-003-16 | Adversarial candidate text cannot alter scripted trusted rubric | complete |
| EVL-003-17 | Pairwise blind labels A/B | complete |
| EVL-003-18 | Both pair orders persist as separate checks | complete |
| EVL-003-19 | Order disagreement and abstention/reviewed counts visible | complete |
| EVL-003-20 | Real judge quality/security not claimed from scripted tests | complete |
| EVL-003-21 | Rubric/scale revision incompatibility requires explicit mapping | complete |
| EVL-004-01 | Separate evidence sink/profile per trial | complete |
| EVL-004-02 | Ordered versioned events and tool correlation IDs | complete |
| EVL-004-03 | Permitted payload projections and outcome snapshots | complete |
| EVL-004-04 | Manifest records per-kind coverage/redactions/truncation/gaps/digest | complete |
| EVL-004-05 | Capture open to sealed/incomplete irreversible lifecycle | complete |
| EVL-004-06 | Recording failure cannot assert completeness | complete |
| EVL-004-07 | Critical evidence loss follows explicit delivery policy | complete |
| EVL-004-08 | Sampled telemetry cannot prove full transcript or absence | complete |
| EVL-004-09 | Secret redaction before artifact/judge/export | complete |
| EVL-004-10 | Default raw payload/reference capture disabled | complete |
| EVL-004-11 | Separate identity-bearing grading projection sanitizes domain views | complete |
| EVL-004-12 | Credentials rejected in references | complete |
| EVL-004-13 | Host access/retention ownership documented | complete |
| EVL-004-14 | Outcome supplied by environment adapter not target prose | complete |
| EVL-004-15 | Controlled trials/rescore/observational modes distinct | complete |
| EVL-004-16 | Offline rescore cannot call target/tools | complete |
| EVL-004-17 | Late grading creates new assessment with lineage | complete |
| EVL-004-18 | Immutable observation ID/revision and projected snapshot | complete |
| EVL-004-19 | Sampling rule/reason/probability/population/window/outcome delay recorded | complete |
| EVL-004-20 | Missing/delayed evidence remains insufficient | complete |
| EVL-004-21 | Worker fixed concurrency/bounded queue/deadline/budget | complete |
| EVL-004-22 | Overload/dropped/rejected explicit; production enqueue nonblocking | complete |
| EVL-004-23 | Flush/cancel bounded with fake-clock fixtures | complete |
| EVL-004-24 | Observational sample cannot silently become controlled A/B | complete |
| EVL-004-25 | Unknown mandatory event type/version blocks relevant grading | complete |
| EVL-005-01 | Experiment manifest freezes dataset/selection/target versions | complete |
| EVL-005-02 | Model/prompt/tools/policy/provider known and unknown provenance | complete |
| EVL-005-03 | Fixture/reset/grader/capture/projection/repeat/retry identities | complete |
| EVL-005-04 | Planned/running/sealed/incomplete lifecycle | complete |
| EVL-005-05 | Baseline resolves to sealed immutable experiment | complete |
| EVL-005-06 | Comparison checks compatible manifests and case revisions | complete |
| EVL-005-07 | Coverage denominator includes all eligible cases | complete |
| EVL-005-08 | Setup/target/grader/cleanup failures visible separately | complete |
| EVL-005-09 | Missing scoring cannot produce quality pass | complete |
| EVL-005-10 | Threshold product policy separate from significance | complete |
| EVL-005-11 | Unit of analysis is cases; repeats average within case | complete |
| EVL-005-12 | Named seeded uncertainty method/assumptions/edge cases | complete |
| EVL-005-13 | Paired differences aggregated by matching case identity | complete |
| EVL-005-14 | Repeats cannot inflate independent sample count | complete |
| EVL-005-15 | At-least-once and all-repeats success distinct denominators | complete |
| EVL-005-16 | Gate pass/fail/inconclusive/invalid comparison plus reasons/policy | complete |
| EVL-005-17 | Aggregate/gate changes produce new comparison identity | complete |
| EVL-005-18 | Sealed experiment never mutates on rescore | complete |
| EVL-005-19 | No automatic production promotion or universal numeric score | complete |
| EVL-006-01 | Portable versioned dataset/experiment/evidence/comparison artifacts | complete |
| EVL-006-02 | Human report IDs/denominators/evidence references/replay guidance | complete |
| EVL-006-03 | CLI runs fixtures, persists and reads sealed baseline, compares | complete |
| EVL-006-04 | Documented CLI pass/fail/inconclusive/infra exit codes | complete |
| EVL-006-05 | Exit codes verified by subprocess tests | complete |
| EVL-006-06 | Artifact staged/validated/committed lifecycle | complete |
| EVL-006-07 | Checksum verifies on read/reopen | complete |
| EVL-006-08 | Partial/corrupt manifest never baseline | complete |
| EVL-006-09 | Publication atomic immutable identity with idempotent retry | complete |
| EVL-006-10 | Local filesystem capabilities and unsupported durability explicit | complete |
| EVL-006-11 | Consumer codecs used; no implicit reflection store payload decoding | complete |
| EVL-006-12 | Export is optional sink port with runnable reference | complete |
| EVL-006-13 | Source of truth artifact; export outage does not change verdict | complete |
| EVL-006-14 | Stable observation ID survives retry | complete |
| EVL-006-15 | Declared dedup avoids duplicates; undeclared backend may repeat | complete |
| EVL-006-16 | Delivery status and semantics separate from verdict | complete |
| EVL-006-17 | Reports/exports share permitted privacy projections | complete |
| EVL-006-18 | Retention/deletion may make replay unavailable explicitly | complete |
| EVL-006-19 | Import/export capabilities list schema/status/scale/loss mappings | complete |
| EVL-006-20 | Unknown required version is unsupported | complete |
| EVL-006-21 | Boolean-only status mapping returns loss instead of fabricated boolean | complete |
| EVL-006-22 | HTTP JSON protocol has codecs/context/body bounds and runnable adapter | complete |
| EVL-006-23 | No vendor SDK readiness claim | complete |
| EVL-006-24 | Compatible envelope extensions preserved by round trip | complete |
| EVL-007-01 | Optional optimizer ships with core capabilities | complete |
| EVL-007-02 | Bounded enumeration over host supplied typed candidates | complete |
| EVL-007-03 | Immutable candidate descriptions/versioned codec/revisions | complete |
| EVL-007-04 | Disjoint training/calibration/holdout split validation | complete |
| EVL-007-05 | Proposal port sees training/calibration only, never holdout labels | complete |
| EVL-007-06 | Proposal scripted reference path | complete |
| EVL-007-07 | Ordinary experiments used for candidate evaluation | complete |
| EVL-007-08 | History preserves invalid candidates and failed/incomplete trials | complete |
| EVL-007-09 | Budget includes proposals/targets/graders | complete |
| EVL-007-10 | Candidate/deadline/stop policy bounds and identity recorded | complete |
| EVL-007-11 | Proposing/evaluating/selecting/completed/stopped state history | complete |
| EVL-007-12 | Proposal/invalid/exhaustion/contamination separate from quality | complete |
| EVL-007-13 | Incomplete experiment cannot win ranking | complete |
| EVL-007-14 | Ranking/tradeoff report with baseline comparison and lineage | complete |
| EVL-007-15 | Reproducible comparison even without improvement | complete |
| EVL-007-16 | Host validation restricts parameters; production settings unchanged | complete |
| EVL-007-17 | Holdout selected winner only; repeated selection marks contamination | complete |
| EVL-007-18 | Human review remains host decision | complete |
| EVL-007-19 | No global optimum/significance promise | complete |
| EVL-007-20 | Algorithm/split/stop changes versioned | complete |
