# Current acceptance — tasks 01–05

Authority: `.cursor/task/README.md` and its five task files. The matrix below maps
current contracts to executable evidence; it is not a percentage claim. Independent
completeness and correctness acceptance requires both reports on the same frozen
source fingerprint. Actual commands, toolchain, outcomes and limitations are in
[validation](validation.md). Reports for the original task and earlier source
fingerprints remain historical and cannot substitute for current checks.

| Current contract | Implementation / specification | Executable evidence |
|---|---|---|
| Preflight before paid dispatch, fresh typed views, partial assessments | `runner.go`, `grader.go`, [execution](execution-contract.md) | `runner_validation_test.go`, `assessment_execution_test.go`, `observation/partial_test.go` |
| Selected-case decoding and isolated pair snapshots | `dataset.go`, pair grading protocol | `execution_v2_test.go`, `assessment_execution_test.go` |
| Evidence completeness distinct from target success; no absence proof after delivery loss | [HTTP](http-protocol.md), [evidence](evidence-contract.md) | `adapters/httpjson/failure_test.go`, `workflow_integration_test.go` |
| Bounded reference syntax and separately classified export failure | `evidence.go`, export port | `evidence_reference_test.go`, `export_classification_test.go` |
| Runtime wire structure and independent generated schema agreement | [wire](wire-contract.md), `internal/wirecontract` | `contracttest/corpus_test.go`, `contracttest/schema_test.go` |
| Explicit objective, native scale/direction, unknown measurement and matched denominators | [measurement](measurement-contract.md) | `measurement_test.go`, `comparison_numerical_test.go` |
| Isolated seeded paired schedule and bounded dispatch | [paired execution](paired-contract.md) | `paired_execution_test.go`, `paired_schedule_internal_test.go` |
| Revisioned calibration counts and unavailable zero-denominator rates | [calibration](calibration-contract.md) | `calibration_test.go` |
| Bounded typed search rounds, service feedback, accounting and stop | [search](search-contract.md), `optimizer` | `optimizer/search_test.go`, `optimizer/adversarial_rounds_test.go`, `optimizer/feedback_behavior_test.go` |
| Best measured vs feasible winner, deterministic ranking and holdout isolation | `optimizer`, host split validation | `optimizer/search_test.go`, `optimizer/split_test.go`, `examples/optimizer` |
| Restored search rejects rehashed semantic contradictions and detaches candidates | `optimizer/restore.go` | `optimizer/restore_contract_test.go`, `optimizer/roundtrip_test.go` |
| Confident text without effect fails; permitted alternative tool path passes | [workflow](workflow-contract.md), [integration](integration-contract.md) | `TestWorkflowTransportSemantics`, `examples/integration` |
| Same host behavior in process and over HTTP; error after effect retains action and usage | `internal/fixtures/workflow.go` | `TestWorkflowTransportSemantics`, `TestWorkflowTargetFailureConformance` |
| Disconnected evidence cannot prove absence; isolation, budget stop and cancellation | workflow host ports | `TestWorkflowTransportDisconnectCannotProveAbsence`, `TestWorkflowBudgetCancellationAndIsolation` |
| Publication/reopen, offline re-score without execution capability, immutable parent lineage | saved view, experiment and assessment ports | `TestWorkflowArtifactAssessmentLineage` |
| Online partial assessment and explicit baseline/candidate comparison | observation/core assessment ports | `TestWorkflowOnlinePartialAssessment`, `TestWorkflowArtifactAssessmentLineage` |
| Generic failure-after-work conformance with host factories, reference negative cases | [failure conformance](conformance-contract.md), `conformance` | `conformance/TestFailureReferenceAdapters`, `TestWorkflowTargetFailureConformance` |
| Explicit versioned CLI gate and supported serializable objective; unsupported callbacks rejected | [CLI](cli-contract.md), comparison-policy schema | `cmd/evaly` subprocess tests, independent `contracttest` corpus |
| Four exit classes and genuine trial/case/revision/seed metadata without fabricated replay | `cmd/evaly`, `Report` | `cmd/evaly` subprocess tests |
| Six existing examples plus offline integration example | `examples/*` | commands recorded in [validation](validation.md) |
| Opt-in real host/judge procedure without SDK/secrets in core/CI | [live integration](live-integration.md) | **Not run:** no host credentials supplied; scripted checks are not LLM accuracy evidence |

Current format inventory: envelope, dataset, evidence, observation, view,
calibration and comparison-policy v1; candidate, scenario and comparison
v2; HTTP request/response, experiment, assessment and observation-result v3; search v5. Readers reject
unsupported prior major formats. There is no migration or compatibility path.
Domain codecs remain independently versioned and owned by the host.

---

Everything after this separator is retained historical evidence for the original
`.cursor/tasks/task1.md` tree. Historical commands, replay descriptions, schemas and
API statements are not instructions for the current tree. Use the contracts and
executable tests above for current behavior.

# Historical acceptance matrix — original task only

Historical authority: the entire `.cursor/tasks/task1.md`. Every section below, including boundaries and repair evidence index, describes the original accepted tree, not current APIs. Historical status: final independent acceptance 171/171 (100%), all seven EVL cards and all ten mandatory synthetic fixtures complete; all fourteen confirmed correctness defects closed. Initial audit was 150/171; the denominator is unchanged. A partially implemented row is not complete. Auditors must inspect source requirements and may identify omissions; no row may be removed to increase completeness.

## BOOT

| ID | Requirement | Implementation/docs | Behavioral evidence to inspect | Runnable example |
|---|---|---|---|---|
| BOOT-01 | Go module github.com/skosovsky/evaly | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-02 | Pinned verified toolchain compatible with local libraries and CI | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-03 | README and package docs | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-04 | Design precedes public API implementation | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-05 | Public wire schemas and separate domain schema identities | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-06 | Acceptance matrix for every card and invariant | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-07 | Runnable examples and testdata | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-08 | Shared conformance suites for implemented ports | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-09 | Core runs without ai-libs/harness/network/API keys/telemetry | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-10 | Optional adapters isolated from core dependency graph | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-11 | BYOT input/output/reference/environment and no mandatory map domain API | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-12 | Host owns auth/credentials/storage/deployment/scheduler/production policy | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-13 | I/O propagates context; worker counts and queues bounded | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-14 | Unknown and unsupported are explicit errors/outcomes | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-15 | No exactly-once external effect or unverified durable guarantees | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-16 | AAA behavioral/adversarial/fault tests | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-17 | No legacy shims or hidden scope deferral | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-18 | Formatting, vet, normal tests and race checks | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
| BOOT-19 | Explicit limitations and real validation report | doc.go; docs/design.md; schemas/; Makefile; .github/workflows/go.yml; docs/design.md | conformance/; all test packages | README.md |
## EVL-001

| ID | Requirement | Implementation/docs | Behavioral evidence to inspect | Runnable example |
|---|---|---|---|---|
| EVL-001-01 | Stable case IDs and typed inputs | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-02 | Optional references and explicit grader reference requirement | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-03 | Metadata/evidence requirements are identity-bearing | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-04 | Dataset fixes ordered case revisions and selection policy | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-05 | Codec/schema identity is part of dataset and case digest | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-06 | Draft validation before sealed runnable snapshot | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-07 | Draft mutations cannot alter sealed snapshots | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-08 | Snapshot access returns independently decoded copies | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-09 | Duplicate IDs reject including same payload | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-10 | Noncanonical/nonserializable payload rejects | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-11 | Input/reference/metadata/composition changes create revision | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-12 | Labels/aliases cannot replace immutable revision | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-13 | Domain migration creates new revision with parent | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-14 | External fixture version or nonreproducible marker | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-15 | Typed target receives case input and trial context | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-16 | Target output/status/usage/outcome evidence are distinct | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-17 | Two materially different typed targets run without harness | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-18 | Host scenario driver has bounded steps/turns/time and versioned behavior | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-19 | Generator creates unvalidated drafts with parent/generator/model/seed | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-20 | Self-assigned generated labels cannot bypass host validation | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
| EVL-001-21 | Adversarial search and scenario replay modes/trajectory lineage distinct | dataset.go; codec.go; docs/design.md | TestDatasetSnapshotAndCanonicalIdentity; TestGeneratedDraftAndScenarioBounds | examples/calculation; examples/protocols |
## EVL-002

| ID | Requirement | Implementation/docs | Behavioral evidence to inspect | Runnable example |
|---|---|---|---|---|
| EVL-002-01 | Run plan fixes repeats/concurrency/deadline/stop/limits | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-02 | Lifecycle prepares namespaces and typed environment handles | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-03 | Readiness/reset precedes dispatch | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-04 | Failed reset never invokes target | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-05 | Queued/preparing/running/collecting/terminal history | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-06 | Completed/target error/setup error/cancelled/budget exhausted statuses | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-07 | Cleanup bounded with independent status/error | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-08 | Cleanup handles partially prepared environment | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-09 | Every repeat has stable unique identity | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-10 | Infrastructure retry has distinct attempt; prior failures retained | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-11 | Unisolated fixtures force serial/shared and cannot claim independence | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-12 | Cancellation stops new target dispatch and forwards context | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-13 | Cancellation cannot imply effect rollback | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-14 | Hard cap uses atomic host reservation before dispatch | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-15 | Capacity 2, cost 1 blocks third dispatch | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-16 | Unknown actual usage retains reservation | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-17 | Known usage is separate observation with reconciliation errors | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-18 | Seed fixed without provider determinism claim | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-19 | Paired sides get same cases/fixture/reset and separate environments | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-20 | Paired order/randomization and timestamps recorded | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-21 | Incompatible shared baseline state invalidates independent pair | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
| EVL-002-22 | Reset/retry changes alter experiment identity | runner.go; budget.go; docs/design.md | TestRunResetFailureAndCleanup; TestBudgetUnknownUsageAndCancellation; TestPairIsolationAndAttempts; conformance.Budget | examples/calculation; examples/crm |
## EVL-003

| ID | Requirement | Implementation/docs | Behavioral evidence to inspect | Runnable example |
|---|---|---|---|---|
| EVL-003-01 | Common typed grader port receives only permitted view | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-02 | Deterministic grader reference path | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-03 | Outcome grader reads independent environment outcome | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-04 | Trajectory absence checks require complete evidence | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-05 | LLM port has scripted reference adapter | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-06 | Scored/not applicable/insufficient evidence/grader error distinct | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-07 | Judge timeout is grader error without zero metric | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-08 | Named scores retain scale and direction | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-09 | Named assertions/reason codes/bounded evidence refs | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-10 | Grader revision includes implementation/rubric/model/prompt/config | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-11 | LLM usage/provenance separate from target | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-12 | Human calibration labels measure disagreement and coverage | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-13 | Calibration labels do not imply certified accuracy | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-14 | Explicit conflict policy preserves all grader results | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-15 | Judge instructions separate from untrusted candidate content | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-16 | Adversarial candidate text cannot alter scripted trusted rubric | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-17 | Pairwise blind labels A/B | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-18 | Both pair orders persist as separate checks | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-19 | Order disagreement and abstention/reviewed counts visible | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-20 | Real judge quality/security not claimed from scripted tests | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
| EVL-003-21 | Rubric/scale revision incompatibility requires explicit mapping | grader.go; rescore.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestBlindPairOrderAndCalibration; TestCRMTextDoesNotOverrideOutcome | examples/crm; examples/protocols |
## EVL-004

| ID | Requirement | Implementation/docs | Behavioral evidence to inspect | Runnable example |
|---|---|---|---|---|
| EVL-004-01 | Separate evidence sink/profile per trial | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-02 | Ordered versioned events and tool correlation IDs | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-03 | Permitted payload projections and outcome snapshots | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-04 | Manifest records per-kind coverage/redactions/truncation/gaps/digest | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-05 | Capture open to sealed/incomplete irreversible lifecycle | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-06 | Recording failure cannot assert completeness | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-07 | Critical evidence loss follows explicit delivery policy | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-08 | Sampled telemetry cannot prove full transcript or absence | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-09 | Secret redaction before artifact/judge/export | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-10 | Default raw payload/reference capture disabled | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-11 | Separate identity-bearing grading projection sanitizes domain views | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-12 | Credentials rejected in references | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-13 | Host access/retention ownership documented | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-14 | Outcome supplied by environment adapter not target prose | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-15 | Controlled trials/rescore/observational modes distinct | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-16 | Offline rescore cannot call target/tools | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-17 | Late grading creates new assessment with lineage | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-18 | Immutable observation ID/revision and projected snapshot | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-19 | Sampling rule/reason/probability/population/window/outcome delay recorded | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-20 | Missing/delayed evidence remains insufficient | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-21 | Worker fixed concurrency/bounded queue/deadline/budget | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-22 | Overload/dropped/rejected explicit; production enqueue nonblocking | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-23 | Flush/cancel bounded with fake-clock fixtures | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-24 | Observational sample cannot silently become controlled A/B | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
| EVL-004-25 | Unknown mandatory event type/version blocks relevant grading | evidence.go; rescore.go; observation/worker.go; docs/design.md | TestEvidencePrivacyGapAndJudgeError; TestOfflineRescoreRevisionAndNoTarget; TestBoundedOverloadFakeDeadlineAndFlush; TestDelayedObservationLineageAndOfflineRegrade | examples/observation; examples/crm |
## EVL-005

| ID | Requirement | Implementation/docs | Behavioral evidence to inspect | Runnable example |
|---|---|---|---|---|
| EVL-005-01 | Experiment manifest freezes dataset/selection/target versions | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-02 | Model/prompt/tools/policy/provider known and unknown provenance | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-03 | Fixture/reset/grader/capture/projection/repeat/retry identities | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-04 | Planned/running/sealed/incomplete lifecycle | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-05 | Baseline resolves to sealed immutable experiment | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-06 | Comparison checks compatible manifests and case revisions | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-07 | Coverage denominator includes all eligible cases | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-08 | Setup/target/grader/cleanup failures visible separately | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-09 | Missing scoring cannot produce quality pass | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-10 | Threshold product policy separate from significance | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-11 | Unit of analysis is cases; repeats average within case | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-12 | Named seeded uncertainty method/assumptions/edge cases | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-13 | Paired differences aggregated by matching case identity | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-14 | Repeats cannot inflate independent sample count | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-15 | At-least-once and all-repeats success distinct denominators | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-16 | Gate pass/fail/inconclusive/invalid comparison plus reasons/policy | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-17 | Aggregate/gate changes produce new comparison identity | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-18 | Sealed experiment never mutates on rescore | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
| EVL-005-19 | No automatic production promotion or universal numeric score | runner.go; comparison.go; docs/design.md | TestComparisonCoverageAndCaseDenominator; TestPairIsolationAndAttempts | examples/calculation; examples/optimizer |
## EVL-006

| ID | Requirement | Implementation/docs | Behavioral evidence to inspect | Runnable example |
|---|---|---|---|---|
| EVL-006-01 | Portable versioned dataset/experiment/evidence/comparison artifacts | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-02 | Human report IDs/denominators/evidence references/replay guidance | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-03 | CLI runs fixtures, persists and reads sealed baseline, compares | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-04 | Documented CLI pass/fail/inconclusive/infra exit codes | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-05 | Exit codes verified by subprocess tests | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-06 | Artifact staged/validated/committed lifecycle | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-07 | Checksum verifies on read/reopen | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-08 | Partial/corrupt manifest never baseline | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-09 | Publication atomic immutable identity with idempotent retry | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-10 | Local filesystem capabilities and unsupported durability explicit | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-11 | Consumer codecs used; no implicit reflection store payload decoding | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-12 | Export is optional sink port with runnable reference | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-13 | Source of truth artifact; export outage does not change verdict | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-14 | Stable observation ID survives retry | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-15 | Declared dedup avoids duplicates; undeclared backend may repeat | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-16 | Delivery status and semantics separate from verdict | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-17 | Reports/exports share permitted privacy projections | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-18 | Retention/deletion may make replay unavailable explicitly | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-19 | Import/export capabilities list schema/status/scale/loss mappings | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-20 | Unknown required version is unsupported | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-21 | Boolean-only status mapping returns loss instead of fabricated boolean | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-22 | HTTP JSON protocol has codecs/context/body bounds and runnable adapter | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-23 | No vendor SDK readiness claim | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
| EVL-006-24 | Compatible envelope extensions preserved by round trip | artifact.go; export.go; cmd/evaly; adapters/httpjson; docs/design.md | TestArtifactReopenConflictCorruptionAndPartialPublication; TestExportDeliveryIsSeparateAndMappingLoss; TestCLIExitCodesAndSealedBaseline; TestHTTPReferenceConformance; TestHTTPRejectsLossAndUnknownVersion | examples/http; README.md CLI commands |
## EVL-007

| ID | Requirement | Implementation/docs | Behavioral evidence to inspect | Runnable example |
|---|---|---|---|---|
| EVL-007-01 | Optional optimizer ships with core capabilities | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-02 | Bounded enumeration over host supplied typed candidates | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-03 | Immutable candidate descriptions/versioned codec/revisions | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-04 | Disjoint training/calibration/holdout split validation | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-05 | Proposal port sees training/calibration only, never holdout labels | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-06 | Proposal scripted reference path | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-07 | Ordinary experiments used for candidate evaluation | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-08 | History preserves invalid candidates and failed/incomplete trials | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-09 | Budget includes proposals/targets/graders | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-10 | Candidate/deadline/stop policy bounds and identity recorded | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-11 | Proposing/evaluating/selecting/completed/stopped state history | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-12 | Proposal/invalid/exhaustion/contamination separate from quality | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-13 | Incomplete experiment cannot win ranking | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-14 | Ranking/tradeoff report with baseline comparison and lineage | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-15 | Reproducible comparison even without improvement | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-16 | Host validation restricts parameters; production settings unchanged | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-17 | Holdout selected winner only; repeated selection marks contamination | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-18 | Human review remains host decision | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-19 | No global optimum/significance promise | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |
| EVL-007-20 | Algorithm/split/stop changes versioned | optimizer/search.go; docs/design.md | TestBoundedSearchIncompleteCannotWin; TestProposalIsolationInvalidCandidatesAndContamination | examples/optimizer |

Frozen initial matrix denominator: **171 atomic rows**. Ten required synthetic fixtures below are scenarios over these rows, not extra percentage credit.

| Fixture | Evidence |
|---|---|
| 1 | TestDatasetSnapshotAndCanonicalIdentity + TestGeneratedDraftAndScenarioBounds |
| 2 | TestCRMTextDoesNotOverrideOutcome + TestEvidencePrivacyGapAndJudgeError |
| 3 | TestRunResetFailureAndCleanup + TestPairIsolationAndAttempts |
| 4 | TestEvidencePrivacyGapAndJudgeError + TestOfflineRescoreRevisionAndNoTarget |
| 5 | TestBudgetUnknownUsageAndCancellation + conformance.Budget |
| 6 | TestComparisonCoverageAndCaseDenominator |
| 7 | TestBlindPairOrderAndCalibration |
| 8 | TestArtifactReopenConflictCorruptionAndPartialPublication + TestExportDeliveryIsSeparateAndMappingLoss |
| 9 | TestBoundedOverloadFakeDeadlineAndFlush + TestDelayedObservationLineageAndOfflineRegrade |
| 10 | TestBoundedSearchIncompleteCannotWin + TestProposalIsolationInvalidCandidatesAndContamination |

## Boundaries and unknown properties

External vendor adapters, real model judge quality, production deployment and distributed scheduling are outside the requested scope. The HTTP bridge implements only its declared protocol. All callbacks require cooperative context handling; physical termination of uncooperative I/O is not promised. FileStore is local-only; cloud-sync/network filesystem durability and Windows are unverified. Memory adapters are process-local. Repeats are dependent observations within cases; bootstrap assumptions and multiple-comparison limitations appear in reports. Field allowlisting depends on host classification. Numeric custom scale aggregation is host policy; stock gates measure assertion pass rate. Holdout ledger must be shared/durable across host processes to track contamination there.

## Validation log

Results will be recorded in `docs/validation.md` after full checks and reviews.

## Repair evidence index (same 171-row denominator)

- Scenario revision/search/replay: `scenario.go`, `TestScenarioSearchTrajectoryAndReplayLineage`, `examples/protocols`.
- Portable saved view: `SavedViewRecord`, `SaveSavedView`, `LoadSavedView`; `TestSavedViewReopenAndRescoreWithDeclaredCodecs`; `examples/observation` performs a real reopen.
- All declared wire schemas include nested types: `internal/schemagen`, `schemas/*-v1.json`, `contracttest/TestGeneratedArtifactsAgainstWireSchemas`; isolated test module checked by Makefile/CI.
- Missing port conformance: `conformance/protocols.go` and `TestProtocolReferenceConformance` cover policy/sink/generator/scenario/judge/pair/proposal/ledger/clock.
- Synthetic fixture additions: `TestPairedCaseRevisionsAndIndependentEnvironments`, `TestSharedFixtureActuallySerial`, `TestBudgetBlocksThirdRealDispatchAndCancellationRetainsEffects`, `TestSecretAbsentInJudgeExportAndVerdictUnchanged`, `TestPairAbstentionIsVisible`.
- Report/replay: `Comparison.Trials`, `Report`, CLI seed/behavior replay; `TestCLIExitCodesAndSealedBaseline` asserts bad trial identity, evidence reference and accurate bad replay command.
- Independent adversarial regressions: `evidence_validation_test.go`, `runner_validation_test.go`, and audit tests in observation/optimizer. Repaired semantic restore, bounded diagnostics, deadline precedence, atomic dispatch Claim, stopped-search dispatch and canonical assessment identity.

Optimizer candidate records use schema v2 (`ParentRevision`). See [optimizer migration](optimizer-remediation.md).
