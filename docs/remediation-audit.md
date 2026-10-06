# Final remediation audit

Source: `.cursor/tasks/task2-evaly-review-remediation.md`, baseline
`76c224a97e5f03a44e87ab7ea308216a7e10d139`. Runtime HEAD `7a37ea3` plus T10 diff.
Final commands, independent acceptance and limits: [verification](remediation-verification.md).

## F01–F10 behavioral proof and AAA regressions

| Item | Baseline behavioral command | Current regression evidence |
|---|---|---|
| F01/F02 | `python3 scripts/release_test.py --baseline` | `scripts/release_test.py`: 10 local fixtures, unrelated files/tags, exact refs, rejected push, cleanup/retry and unknown outcomes |
| F03 | `python3 scripts/budget_baseline_repro.py` | `budget_exact_test.go`: tiny admission, large settlement, mixed orders, extremes, claim/concurrency |
| F04/F05 | `python3 scripts/codec_baseline_repro.py` | `codec_roundtrip_test.go`: exact numbers/lexemes/overflow, dataset restore, aliases/cycles, UTF-8/ownership |
| F06 | `python3 scripts/preflight_baseline_repro.py` | `restore_preflight_test.go`: invalid capabilities before codec/store effects, valid identity roundtrip |
| F07 | Same preflight script | `absence_preflight_test.go`: invalid setup before all effects, valid coverage |
| F08 | `python3 scripts/optimizer_baseline_repro.py` | `optimizer/remediation_repro_test.go`: five independently rehashed semantic mutations plus restore/terminal negatives |
| F09 | Same optimizer script | foreign calibration/holdout with/without callback error; bound partial records remain restorable |
| F10 | `python3 scripts/conformance_baseline_repro.py` | `conformance/lifecycle_cleanup_test.go`: partial Prepare, Reset error, Goexit, cleanup failures |

Every baseline command was rerun during T10 and passed behavioral assertions.
The original codec-only overlay failed compilation after a private helper change;
that is not defect evidence. The repaired full disposable baseline checkout passed.

## D01–D60 decision inventory

Each linked record states the final decision, reason and implementation/test evidence.
Retained semantics are deliberate host boundaries, not outstanding implementation.

| Item | Original topic | Authoritative decision/reason record |
|---|---|---|
| D01 | StopOnInfrastructure, paired.go:103 | [local-remediation-decisions.md](local-remediation-decisions.md) — D01 |
| D02 | Claim→cancel/projection failure | [budget.md](budget.md) — D02 |
| D03 | Ошибка Claim и автоматический refund | [budget.md](budget.md) — D03 |
| D04 | Cleanup partial Prepare, detached bounded context | [conformance-contract.md](conformance-contract.md) — D04 |
| D05 | NewCapture fallback, ignored errors; runner.go:369,469 | [local-remediation-decisions.md](local-remediation-decisions.md) — D05 |
| D06 | Privacy-safe generic failure categories | [local-remediation-decisions.md](local-remediation-decisions.md) — D06 |
| D07 | Фиксированные max slots/repeats/workers | [local-remediation-decisions.md](local-remediation-decisions.md) — D07 |
| D08 | No-op reflect switch + misplaced GoDoc; preflight.go:23 | [preflight.md](preflight.md) — D08 |
| D09 | JSON tags на callback-bearing configs | [local-remediation-decisions.md](local-remediation-decisions.md) — D09 |
| D10 | RunScenario timeout начинается после Encode | [local-remediation-decisions.md](local-remediation-decisions.md) — D10 |
| D11 | Standalone Run и paired scheduling | [local-remediation-decisions.md](local-remediation-decisions.md) — D11 |
| D12 | MinimumQuality на lower-is-better | [local-remediation-decisions.md](local-remediation-decisions.md) — D12 |
| D13 | NumericObjective Descriptor vs GraderID/MetricName | [local-remediation-decisions.md](local-remediation-decisions.md) — D13 |
| D14 | CleanupFailures при quality GatePass | [local-remediation-decisions.md](local-remediation-decisions.md) — D14 |
| D15 | measurement_contract_invalid без локализации | [local-remediation-decisions.md](local-remediation-decisions.md) — D15 |
| D16 | Опечатка source grader → missing/inconclusive | [local-remediation-decisions.md](local-remediation-decisions.md) — D16 |
| D17 | Case bootstrap, point thresholds | [local-remediation-decisions.md](local-remediation-decisions.md) — D17 |
| D18 | Calibrate | [local-remediation-decisions.md](local-remediation-decisions.md) — D18 |
| D19 | Serialized policy vs runtime Objective | [local-remediation-decisions.md](local-remediation-decisions.md) — D19 |
| D20 | Fresh View factory — host promise | [preflight.md](preflight.md) — D20 |
| D21 | graded/complete ≠ pass | [preflight.md](preflight.md) — D21 |
| D22 | Observation reconciliation timeout | [local-remediation-decisions.md](local-remediation-decisions.md) — D22 |
| D23 | Stock JSON validation видит json:"-" fields | [codecs.md](codecs.md) — D23 |
| D24 | Encode buffer lifetime / Decode isolation | [codecs.md](codecs.md) — D24 |
| D25 | FieldPolicy map/slices сохраняются по ссылке | [local-remediation-decisions.md](local-remediation-decisions.md) — D25 |
| D26 | MarkIncomplete(reason) игнорирует reason | [local-remediation-decisions.md](local-remediation-decisions.md) — D26 |
| D27 | Coverage channel completeness vs event presence | [preflight.md](preflight.md) — D27 |
| D28 | Save/Load store helpers без ValidatePort | [preflight.md](preflight.md) — D28 |
| D29 | Reference credential filter по substring | [local-remediation-decisions.md](local-remediation-decisions.md) — D29 |
| D30 | MemoryExport/FileStore mutable public settings | [local-remediation-decisions.md](local-remediation-decisions.md) — D30 |
| D31 | FileStore local durability | [local-remediation-decisions.md](local-remediation-decisions.md) — D31 |
| D32 | Worker status/reason raw strings | [local-remediation-decisions.md](local-remediation-decisions.md) — D32 |
| D33 | Повторная SavedView validation на каждого grader | [local-remediation-decisions.md](local-remediation-decisions.md) — D33 |
| D34 | Flush vs shutdown | [local-remediation-decisions.md](local-remediation-decisions.md) — D34 |
| D35 | Shared grader/clock/budget при concurrency>1 | [local-remediation-decisions.md](local-remediation-decisions.md) — D35 |
| D36 | Sampling/OutcomeDelay и deadline | [local-remediation-decisions.md](local-remediation-decisions.md) — D36 |
| D37 | Размер Reasons/Metrics/Pair reason | [local-remediation-decisions.md](local-remediation-decisions.md) — D37 |
| D38 | Judge retries/fallback | [local-remediation-decisions.md](local-remediation-decisions.md) — D38 |
| D39 | Winner означает calibration winner | [optimizer-remediation.md](optimizer-remediation.md) — D39 |
| D40 | Candidate/round limit → stopped | [optimizer-remediation.md](optimizer-remediation.md) — D40 |
| D41 | Candidate Parent | [optimizer-remediation.md](optimizer-remediation.md) — D41 |
| D42 | Empty proposal и Exhausted=false | [optimizer-remediation.md](optimizer-remediation.md) — D42 |
| D43 | StaticProposer игнорирует Request.Maximum | [optimizer-remediation.md](optimizer-remediation.md) — D43 |
| D44 | RoundHistory/History/Ranking/States/usage дублируются | [optimizer-remediation.md](optimizer-remediation.md) — D44 |
| D45 | cloneEvaluation/Record helpers игнорируют JSON errors | [optimizer-remediation.md](optimizer-remediation.md) — D45 |
| D46 | EvaluationUnits — overhead liability | [budget.md](budget.md) — D46 |
| D47 | Holdout Ledger.Claim до budget authorize | [optimizer-remediation.md](optimizer-remediation.md) — D47 |
| D48 | Split.Revision — host label | [optimizer-remediation.md](optimizer-remediation.md) — D48 |
| D49 | terminal Reason overwritten при нескольких failures | [optimizer-remediation.md](optimizer-remediation.md) — D49 |
| D50 | Metadata oversized proposals до rejection | [optimizer-remediation.md](optimizer-remediation.md) — D50 |
| D51 | Budget conformance не вызывает Claim/Release | [conformance-contract.md](conformance-contract.md) — D51 |
| D52 | HTTP Handler вызывает callback при уже cancelled ctx | [http-protocol.md](http-protocol.md) — D52 |
| D53 | HTTP interoperability bits RichStatus/MetricScales | [http-protocol.md](http-protocol.md) — D53 |
| D54 | Один MaxBytes на request/response | [http-protocol.md](http-protocol.md) — D54 |
| D55 | Handler invalid setup →400 на каждый request | [http-protocol.md](http-protocol.md) — D55 |
| D56 | Общий FlagSet CLI | [cli-contract.md](cli-contract.md) — D56 |
| D57 | Conformance hardcoded IDs/timeouts | [conformance-contract.md](conformance-contract.md) — D57 |
| D58 | Wrapper cancellation conformance | [conformance-contract.md](conformance-contract.md) — D58 |
| D59 | Schema corpus numeric mutation через float64 | [conformance-contract.md](conformance-contract.md) — D59 |
| D60 | Lint-driven exhaustive zero initialization | [local-remediation-decisions.md](local-remediation-decisions.md) — D60 |

## Documentation checklist

| Source requirement | Evidence |
|---|---|
| Current release notes: seven examples, zero lint | `release-notes.md`; dated historical snapshot preserved |
| Local backlog and sibling feature spec current status | `.cursor/task/README.md`, `../ai-libs/feature-specs/evaly.md`; hashes/scope in verification, no forced Git inclusion |
| Public GoDoc, correctly placed codec comment | `runner.go`, `scenario.go`, `dataset.go`, `codec.go`, `optimizer/{doc,search,protocol,split}.go` |
| Runnable end-to-end quickstart and explicit gate | `quickstart.md`, `quickstart_test.go` ExampleRun; seven existing examples preserved |
| Ownership/concurrency matrix | `concurrency.md`, private records separated from public config and host ports |
| Portable release contract | `release.md`, portable Go editing rather than BSD sed; Linux execution not claimed |
| Authoritative clear-break migration/revisions | `migration.md`, current 16 schemas; no legacy readers, aliases or version relabeling |
| Durable verification | `remediation-verification.md`: commands, outcomes, SHAs, limits and independent verdicts |

## Definition of Done

| Source criterion | Evidence |
|---|---|
| 1. All F repaired, old behavioral proof and new AAA tests | F inventory above, baseline commands rerun; compilation error never counted as behavioral proof |
| 2. All D decisions with reasons | Explicit 60-item inventory above and authoritative linked records |
| 3. Supported sealed dataset/search/scenario restores | Codec roundtrip; generated-artifact schema tests; optimizer bound partial and foreign-output negatives; scenario state-codec-failure roundtrip |
| 4. Exact liabilities, no-op Claim rejection, cleanup failures | Budget exact/race/fuzz and conformance adversarial/subprocess tests |
| 5. Pinned validate, both modules, seven examples and schema semantics | T10 make validate PASS, independent 16-schema byte/inventory comparison; corpus and independently rehashed semantic negatives |
| 6. Addressed edge/race/fuzz/adversarial mutations | Final race suite; accepted T03/T04/T06 short fuzz outcomes retained in verification; documentation changes do not justify broad expensive fuzz repeats |
| 7. Local release fixtures | T10 10/10 PASS plus two baseline repros; exact root-only refs and failure recovery; no real publication |
| 8. Documentation synchronized with verification limits | Documentation checklist above; no claim of live provider quality from scripted judges |
| 9. BYOT independence and host deployment decision | Root go.mod has no external dependencies; root go list -deps has only stdlib, evaly/internal wirecontract; optional packages remain optional |

## Sequential accepted commits

T01 `e7ba3c2`; T02 `4b0bdb8`; T03 `5872faa`; T04 `bd8ba96`;
T05 `dd27a6f`; T06 `e56e0d8`; T07 `2e81d4d`; T08 `7823a2e`;
T09 `7a37ea3`. Each received two independent acceptances before commit.
T10 final acceptance and commit are recorded in verification when completed.
