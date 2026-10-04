# Independent task 05 completeness acceptance

Reviewer: `/root/accept_integration_completeness` (no implementation edits).
Authority: full `.cursor/task/05-integration-and-acceptance.md` and `.cursor/task/README.md`.

Result: **59/59 (100%)**. Fixed denominator established before implementation: 40 concrete normative checks (8+6+6+5+4+3+4+4), all 8 acceptance requirements, all 11 common requirements. No partial row receives credit; no omitted row is removed from the denominator.

Frozen source SHA-256: `2e4897bfda0bc1778cdf6246fa45c5606e98fb6dd2e6172fc8cd0c28996b26e1` — 134 files, using `/private/tmp/evaly-review-state.py`. Includes task authorities 01–05; excludes acceptance reports and validation journal. Fingerprint checked before and after independent execution.

Independent final commands on that source, Go **go1.27.1 darwin/arm64**:

- `GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate` — exit 0, formatting/vet/race tests in root and contracttest; independent generated schema inventory/byte drift and 16-format structural corpus passed. contracttest race 94.379s.
- `GOCACHE=/private/tmp/evaly-final-1.27.1-cache go run ./examples/{calculation,crm,protocols,http,observation,optimizer,integration}` — seven separate commands, each exit 0.
- `git diff --check` — exit 0.

Logs: `/private/tmp/evaly-integration-completeness-validation.log`, `/private/tmp/evaly-integration-completeness-example-<name>.log`.

| ID | Complete requirement | Current executable/source evidence |
|---|---|---|
| N1a | Reference consumer target has at least two tools | PASS — `internal/fixtures/workflow.go`; `TestWorkflowTransportSemantics`; actual refund/credit audit; controlled lookup and post-effect errors |
| N1b | Typed state/input/output/reference/environment | PASS — `internal/fixtures/workflow.go`; `TestWorkflowTransportSemantics`; actual refund/credit audit; controlled lookup and post-effect errors |
| N1c | Verifiable external state | PASS — `internal/fixtures/workflow.go`; `TestWorkflowTransportSemantics`; actual refund/credit audit; controlled lookup and post-effect errors |
| N1d | Controlled tool error | PASS — `internal/fixtures/workflow.go`; `TestWorkflowTransportSemantics`; actual refund/credit audit; controlled lookup and post-effect errors |
| N1e | Error after committed effect | PASS — `internal/fixtures/workflow.go`; `TestWorkflowTransportSemantics`; actual refund/credit audit; controlled lookup and post-effect errors |
| N1f | Text distinct from actual outcome | PASS — `internal/fixtures/workflow.go`; `TestWorkflowTransportSemantics`; actual refund/credit audit; controlled lookup and post-effect errors |
| N1g | Trajectory distinct from outcome | PASS — `internal/fixtures/workflow.go`; `TestWorkflowTransportSemantics`; actual refund/credit audit; controlled lookup and post-effect errors |
| N1h | Evidence completeness distinct from target success | PASS — `internal/fixtures/workflow.go`; `TestWorkflowTransportSemantics`; actual refund/credit audit; controlled lookup and post-effect errors |
| N2a | Same host behavioral contract in-process and HTTP | PASS — `workflow_integration_test.go`: transport semantics, shared TargetFailures, physical HTTP disconnect, budget/cancellation/isolation |
| N2b | Target-error evidence and usage retained | PASS — `workflow_integration_test.go`: transport semantics, shared TargetFailures, physical HTTP disconnect, budget/cancellation/isolation |
| N2c | Transport loss cannot prove forbidden-action absence | PASS — `workflow_integration_test.go`: transport semantics, shared TargetFailures, physical HTTP disconnect, budget/cancellation/isolation |
| N2d | Lifecycle isolation both adapters | PASS — `workflow_integration_test.go`: transport semantics, shared TargetFailures, physical HTTP disconnect, budget/cancellation/isolation |
| N2e | Budget stop both adapters | PASS — `workflow_integration_test.go`: transport semantics, shared TargetFailures, physical HTTP disconnect, budget/cancellation/isolation |
| N2f | Cancellation both adapters; semantic comparison not digests | PASS — `workflow_integration_test.go`: transport semantics, shared TargetFailures, physical HTTP disconnect, budget/cancellation/isolation |
| N3a | Artifact publication and actual reopen | PASS — `TestWorkflowArtifactAssessmentLineage`; `TestWorkflowOnlinePartialAssessment`; `examples/integration` |
| N3b | Offline rescore lacks target capability | PASS — `TestWorkflowArtifactAssessmentLineage`; `TestWorkflowOnlinePartialAssessment`; `examples/integration` |
| N3c | Online partial assessment | PASS — `TestWorkflowArtifactAssessmentLineage`; `TestWorkflowOnlinePartialAssessment`; `examples/integration` |
| N3d | Baseline candidate comparison | PASS — `TestWorkflowArtifactAssessmentLineage`; `TestWorkflowOnlinePartialAssessment`; `examples/integration` |
| N3e | Grader revision creates new assessment lineage | PASS — `TestWorkflowArtifactAssessmentLineage`; `TestWorkflowOnlinePartialAssessment`; `examples/integration` |
| N3f | Original experiment/verdict immutable | PASS — `TestWorkflowArtifactAssessmentLineage`; `TestWorkflowOnlinePartialAssessment`; `examples/integration` |
| N4a | Target host failure-after-effect conformance | PASS — `conformance/failures.go`, `conformance/failures_test.go`; shared `TestWorkflowTargetFailureConformance` |
| N4b | Grader and PairJudge partial failure conformance | PASS — `conformance/failures.go`, `conformance/failures_test.go`; shared `TestWorkflowTargetFailureConformance` |
| N4c | Evidence partial failure conformance | PASS — `conformance/failures.go`, `conformance/failures_test.go`; shared `TestWorkflowTargetFailureConformance` |
| N4d | Export partial failure conformance | PASS — `conformance/failures.go`, `conformance/failures_test.go`; shared `TestWorkflowTargetFailureConformance` |
| N4e | Proposal partial failure conformance; no domain hard coding | PASS — `conformance/failures.go`, `conformance/failures_test.go`; shared `TestWorkflowTargetFailureConformance` |
| N5a | Explicit versioned CLI policy | PASS — `comparison_policy.go`; `schemas/comparison-policy-v1.json`; CLI subprocesses; independent schema/runtime corpus |
| N5b | Serializable assertion and numeric measurement | PASS — `comparison_policy.go`; `schemas/comparison-policy-v1.json`; CLI subprocesses; independent schema/runtime corpus |
| N5c | Unsupported objective rejected | PASS — `comparison_policy.go`; `schemas/comparison-policy-v1.json`; CLI subprocesses; independent schema/runtime corpus |
| N5d | No callbacks/code execution through JSON policy or hidden fixture gate | PASS — `comparison_policy.go`; `schemas/comparison-policy-v1.json`; CLI subprocesses; independent schema/runtime corpus |
| N6a | No fabricated foreign-target replay | PASS — `cmd/evaly/main.go`, `comparison.go`; `TestCLIExitCodesAndSealedBaseline`, `TestCLINumericPolicyForHostTarget` |
| N6b | Trial/case/revision/seed and host metadata emitted | PASS — `cmd/evaly/main.go`, `comparison.go`; `TestCLIExitCodesAndSealedBaseline`, `TestCLINumericPolicyForHostTarget` |
| N6c | Documented pass fail inconclusive invalid/infrastructure exit codes | PASS — `cmd/evaly/main.go`, `comparison.go`; `TestCLIExitCodesAndSealedBaseline`, `TestCLINumericPolicyForHostTarget` |
| N7a | README design and package docs current | PASS — README, doc.go, docs/design.md, docs/acceptance.md, docs/wire-contract.md; validation journal records final checks |
| N7b | Acceptance and validation current executable evidence | PASS — README, doc.go, docs/design.md, docs/acceptance.md, docs/wire-contract.md; validation journal records final checks |
| N7c | Clear break unsupported old versions no migration layer | PASS — README, doc.go, docs/design.md, docs/acceptance.md, docs/wire-contract.md; validation journal records final checks |
| N7d | Prior reports explicit historical | PASS — README, doc.go, docs/design.md, docs/acceptance.md, docs/wire-contract.md; validation journal records final checks |
| N8a | Opt-in live real host/judge through same ports | PASS — `docs/live-integration.md`; explicit not-run status; scripted example output; stdlib root go.mod |
| N8b | Inputs versions usage results and integration claim scope documented | PASS — `docs/live-integration.md`; explicit not-run status; scripted example output; stdlib root go.mod |
| N8c | No vendor SDK or credentials in core/CI; no automatic paid calls | PASS — `docs/live-integration.md`; explicit not-run status; scripted example output; stdlib root go.mod |
| N8d | Live not run without credentials; scripted not LLM quality evidence | PASS — `docs/live-integration.md`; explicit not-run status; scripted example output; stdlib root go.mod |
| A1 | Text without effect fails; alternate path passes | PASS — `TestWorkflowTransportSemantics` (text-only/refund/alternative) |
| A2 | Error after effect retained and loss absence unknown | PASS — `TestWorkflowTargetFailureConformance`, `TestWorkflowTransportDisconnectCannotProveAbsence` |
| A3 | Artifacts reopen and offline rescore immutable lineage | PASS — `TestWorkflowArtifactAssessmentLineage`: actual FileStore reopen, no target capability to rescore, immutable source |
| A4 | Common contract both adapters and negative reference ports | PASS — Shared TargetFailures both transports; `TestFailureReferenceAdapters` all five failure port groups |
| A5 | CLI subprocess policy four exits unsupported foreign replay | PASS — CLI subprocess tests cover caller thresholds, four exits, unsupported descriptor, customer target metadata |
| A6 | Six prior and new seventh examples offline no legacy | PASS — All seven `go run ./examples/...` commands independently exited 0 |
| A7 | Root contracttest schema drift make validate seven examples real commands/toolchain | PASS — Independent final `make validate` exit 0; both modules/race/schema drift; Go 1.27.1 darwin/arm64 |
| A8 | Live execution explicitly declared and limited synthetic claims | PASS — `docs/live-integration.md`; live not run; no model-quality/immunity/durability certification |
| C1 | Clear break fully updated consumers and removed replaced API | PASS — All in-tree consumers updated; explicit mandatory policy; no legacy reader or compatibility path |
| C2 | Spec/Contract-first designed inputs results states identity ownership cancellation limits | PASS — `docs/integration-contract.md`, `docs/workflow-contract.md`, `docs/cli-contract.md`, `docs/conformance-contract.md`, design |
| C3 | Changed wire semantics explicit versions schemas runtime corpus coherent | PASS — New comparison-policy v1; existing artifact semantics unchanged; runtime/schema inventory and independent corpus synchronized |
| C4 | BYOT caller types/codecs no mandatory Agent/Message or reflection-copy | PASS — Generic ports and host fault factories; typed WorkflowInput/Output/State/Environment, caller-owned codecs |
| C5 | Stdlib core no optional imports runtime/vendor/platform scope growth | PASS — Stdlib root go.mod; optional packages separate; no vendor SDK/runtime/dashboard/scheduler added |
| C6 | Host owns domain policy secrets durable budget retention and independence | PASS — Design/live/conformance contracts assign domain policy, secrets, durable accounting, retention and data independence to host |
| C7 | No forcible callback stop/exactly-once/universal secret detection/ID independence/judge quality promise | PASS — doc.go, live integration and design limits explicitly narrow guarantees |
| C8 | AAA behavioral adversarial tests working reference conformance | PASS — Arrange/Act/Assert behavior tests and working negative reference adapters; race suite passes |
| C9 | Updated design/schema/docs regression commands | PASS — Current design/contracts/schema/tests/docs and explicit verification commands |
| C10 | All actual checks no weakened gate/foreign style refactor | PASS — Unchanged formatter/vet/race gate passes; independent schema drift; strict lint not claimed |
| C11 | Historical blocked checks not PASS; no foreign cache deletion/.cursor force add | PASS — Prior failures/report fingerprints labelled historical; no old PASS used; ignored task authorities remain local |

## Scope of acceptance

This is completeness acceptance for the frozen current contract, separate from the independent correctness review. The workflow independently checks actual external state rather than target text; HTTP and in-process paths share host invocation and failure conformance. Real transport disconnect leaves usage unknown and absence unproved. Reopened saved views permit rescore with no execution capability; changed rubric lineage leaves the source untouched.

Live host agent/judge validation was **not run**: no credentials supplied and no paid call authorized or automatically made. Synthetic tests demonstrate infrastructure behavior, not LLM accuracy, injection immunity or production durability. Callbacks require cooperative context handling; durable budget/ledger, secrets, retention and deployment remain host responsibilities. Strict lint acceptance is not claimed. Historical matrix/report outcomes do not certify this source.
