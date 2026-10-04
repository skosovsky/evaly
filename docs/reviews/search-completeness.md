# Task 04 — independent completeness acceptance

Status: **accepted — 51/51 rows, 100% completeness**.

Reviewer: `accept_search_completeness`, independent of implementation authors. Authoritative scope: `.cursor/task/04-extensible-optimizer.md` and `.cursor/task/README.md`. Fixed denominator **51 conjunctive rows: 32 normative + 8 acceptance + 11 common**. All requirements must pass in full; no partial credit, exclusions, or scope shrink. Checklist was fixed before final implementation review.

## Normative requirements

| ID | Full requirement | Evidence |
| --- | --- | --- |
| N01 | declared task03 Objective/measurement used | PASS — optimizer/search.go Compare and persisted Objective; declared higher/lower ranking |
| N02 | eligibility distinct from quality | PASS — evaluated/incomplete state and comparison coverage/missingness distinct from Quality |
| N03 | revisioned host feasibility constraints | PASS — Constraints[T], ConstraintsFunc[T], frozen ConstraintsRevision; fresh typed value and copied summary |
| N04 | best measured separate from feasible eligible winner | PASS — BestMeasured includes infeasible; Winner requires complete eligible feasible passing gate; TestLowerObjectiveSeparatesMeasuredBestFromFeasibleWinner |
| N05 | deterministic tie revision | PASS — TieRevision candidate-revision-lexical-v1; search/restore sort full revisions |
| N06 | bounded rounds with typed proposal request | PASS — Config.MaximumRounds and ProposalRequest.Round/DispatchID; RoundHistory ordered |
| N07 | request only training/calibration plus permitted feedback | PASS — ProposalRequest exposes only sealed Training/Calibration and copied Feedback; no Holdout member |
| N08 | feedback candidate lineage | PASS — CandidateLineage ID/Revision/Parent/Algorithm/Codec; example/test verifies repaired parent |
| N09 | feedback measurements and failure/missingness reasons | PASS — Feedback Quality/Aggregate State/Reason; missingness Excluded reasons preserved |
| N10 | explicit host projection required for permitted evidence references | PASS — FeedbackProjector revision required; absent by default, ValidateGrade reference checks; conformance FeedbackProjection |
| N11 | no raw outputs/secrets or target/environment access in feedback | PASS — Feedback lacks raw descriptions/trials/outputs/target/environment; lineage only; callback inputs copied |
| N12 | host owns modification algorithm, no built-in optimizer/vendor/templates | PASS — host ProposalFunc owns repair; optional package no templates/vendor/GEPA/DSPy |
| N13 | working deterministic reference proposer | PASS — StaticProposer deterministic sealed inputs; dynamic deterministic host reference in examples/optimizer |
| N14 | static enumeration same protocol | PASS — StaticProposer implements same Proposer; only one Search accounting/ranking/stop path |
| N15 | candidate limit | PASS — all attempts including invalid/duplicate consume MaximumCandidates; capacity checked before dispatch |
| N16 | round limit | PASS — round limit stop; TestSearchLimitsForbidFurtherDispatch |
| N17 | elapsed limit | PASS — Timeout context; boundary checks before/after authorize and callbacks; paid cancellation tests |
| N18 | proposal/target/grader budget bounded | PASS — proposal Reserve/Claim/Reconcile and evaluation overhead; Evaluate receives same Budget for target/grader; conservative unknown liability |
| N19 | unique round/evaluation dispatch IDs | PASS — search/phase/round/position claims; full search/phase/candidate hash experiment IDs; full-length ID regression |
| N20 | repeated ID cannot pay for fresh dispatch | PASS — MemoryBudget duplicate claim conflict even units0; TestRepeatedSearchIDCannotRedispatchZeroUnitProposal |
| N21 | partial history and paid usage survive errors/stops | PASS — Round.Usage/Received and History experiment snapshots survive failure; paid proposal tests retain settlement |
| N22 | stop prevents proposal/evaluation/holdout dispatch | PASS — state stop guards loops/selection/holdout; limit/budget/cancel tests; structural protocol errors stop |
| N23 | selection calibration only | PASS — all candidate evaluations Phase=calibration; Compare CalibrationBaseline; winner fixed before holdout |
| N24 | holdout after fixed winner only | PASS — holdout block guarded Winner and not stopped; selection before ledger/evaluation |
| N25 | holdout labels/measurements/traces never in proposal inputs | PASS — typed request excludes holdout; preflight structural holdout restore only; multi-round spy test checks visible labels/feedback |
| N26 | holdout cannot restart selection | PASS — single holdout final block; failed quality holdout test asserts exactly2 proposals2calibrations1holdout |
| N27 | host ledger reuse marks contaminated nonindependent | PASS — MemoryLedger host port; Contaminated state preserved; contamination test; docs no statistical proof |
| N28 | revisioned host group/content or typed split validation | PASS — SplitValidator[I,R] revision, KeySplitValidator GroupKey/ContentKey and reference conformance |
| N29 | ID disjointness necessary, not sufficient independence | PASS — mandatory validateSplit ID overlap; IndependenceValidated only explicit successful revisioned host validator |
| N30 | different substantive cases in example | PASS — example input/reference 1+2,3+2,5+2 with content-key validation |
| N31 | Candidate[T]/caller codecs remain | PASS — Candidate[T], codec Seal/Value/RestoreCandidate; two typed domains; no universal recipe/prompt |
| N32 | identity contains objective/constraints/proposal/split/stop revisions and ordered rounds; restore preserves new generic results | PASS — Result identities plus ordered RoundHistory sealed revision; ValidateResult and RestoreResult verify/copy generic candidate results; adversarial rehashed false claims |

## Acceptance requirements

| ID | Full requirement | Evidence |
| --- | --- | --- |
| A01 | static/proposer unified accounting/ranking/stops | PASS — StaticProposer/ProposalFunc one Search path; protocol conformance and search tests |
| A02 | two-round deterministic failure feedback improvement ordinary experiment and lineage | PASS — TestTwoRoundProposalImprovesMeasuredCandidateFromFeedback; ordinary sealed experiments, 0->1 score, repaired parent; deterministic example |
| A03 | two distinct typed candidate domains examples | PASS — examples/optimizer Recipe and ResourcePlan, generic host run[T] |
| A04 | lower-is-better constraints incomplete/infeasible cannot win explicit absent winner | PASS — lower objective infeasible measured best and feasible runner-up; all-infeasible explicit no_selectable_candidate; incomplete test |
| A05 | spy no holdout labels/results; holdout fail no hidden candidate selection | PASS — TestFeedbackRoundRepairsFailureWithoutHoldoutLeakageOrReselection; typed request no holdout/labels/results, failed holdout no extra selection |
| A06 | round/candidate/time/budget stops preserve partial/usage and repeated IDs | PASS — limits forbid extra dispatch; budget stop; claim cancellation; paid proposal cancellation retains usage/lineage; repeated ID rejects redispatch |
| A07 | host groups detect different-ID overlap; absence validator no independence claim | PASS — TestHostSplitKeysRejectRelatedCasesWithDifferentIDs; TestNoHostValidatorDoesNotClaimIndependentSplit |
| A08 | conformance/examples/schemas/README/design updated; old one-shot removed no shim | PASS — conformance Proposal/Constraints/FeedbackProjection/SplitValidation; examples/optimizer, search.v4 schema, README/design; Candidates/Validate removed |

## Common requirements

| ID | Full requirement | Evidence |
| --- | --- | --- |
| C01 | clear break all consumers | PASS — all callers/tests/examples/schema fixtures updated for bounded protocol |
| C02 | replaced APIs/fields/aliases/dual paths/readers/shims removed | PASS — Config.Candidates/Validate one-shot removed; no wrapper/alias/legacy search.v3 reader |
| C03 | spec-first inputs/results/states/errors/ownership/identity/cancel/limits | PASS — docs/search-contract.md and docs/design.md designed before author changes; states, ownership, ids, cancellation and bounds |
| C04 | incompatible wire versioned schemas/runtime/fixtures synchronized old unsupported | PASS — search version4 runtime/schemagen/wirecontract/contracttest corpus; former schema removed; version mismatch rejected |
| C05 | BYOT/no mandatory models/no domain reflection copy | PASS — host candidate/input/output/reference types and codecs; typed constraints; service feedback only |
| C06 | stdlib core/no optional imports/vendor/network credentials | PASS — core remains stdlib/internal stdlib wirecontract; optional optimizer imports root, no root optimizer dependency |
| C07 | host boundaries | PASS — host owns algorithm/domain meaning/constraints/keys/ledger/budget/retention; no production promotion |
| C08 | no unsupported forced-stop/exactly-once/secrecy/independence/judge claims | PASS — docs no forced termination/exactly-once/statistical proof/semantic duplicate inference; inherited core boundaries retained |
| C09 | AAA observable adversarial tests; working refs/conformance for ports | PASS — AAA observable failure tests; working StaticProposer/ConstraintsFunc/ProjectionFunc/KeySplitValidator with reusable conformance |
| C10 | docs/schema/regressions + independent final makevalidate all6examples | PASS — independent final make validate and all six examples PASS; source hash unchanged |
| C11 | no weakened checks/style detour/foreign cache deletion/force-add ignored task files | PASS — no weakened assertions; formatter repair preserved predicates; git diff --check; ignored tasks remain local, no foreign cache deletion |

## Frozen-source verification

Source fingerprint: `08eba985354d909f1ac16e7bc9012618d604ee897f41e00c46f3524dfb0a802b`, **119 files**, computed by `/private/tmp/evaly-review-state.py`. Includes task README and tasks01–04; excludes review reports and validation journal. Checked before and after verification with identical result.

- Independent `env GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate`: **exit 0**, formatting, vet, root and isolated contracttest race tests (contracttest 91.354 seconds). Log `/private/tmp/evaly-search-completeness-validation.log`. Shared corpus and schema generation drift passed.
- Each `go run ./examples/{calculation,crm,protocols,http,observation,optimizer}`: **exit 0**. Logs `/private/tmp/evaly-search-completeness-example-<name>.log`.
- `git diff --check`: **exit 0**.

Earlier frozen b936… failed the independent format check on two long test assertions; authors repaired formatting while retaining all checks, then issued the new fingerprint above. The failed run is not counted as a pass. Early feedback-description, protocol-stop, ranking/restore and observable-test gaps were resolved before final acceptance.

Implementation source was not edited by this reviewer. This percentage establishes completeness against the full task; the second independent reviewer assesses correctness.
