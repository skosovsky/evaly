# Task 03 — independent completeness acceptance

Status: **accepted — 53/53 rows, 100% completeness**.

Reviewer: `accept_measurement_completeness`, independent of all implementation authors. Source of scope: `.cursor/task/03-measurement-and-comparison.md` and `.cursor/task/README.md`. Denominator is fixed at **53 conjunctive rows: 35 normative, 7 acceptance, 11 common**. Each row is accepted only in full; no partial credit, exclusion or scope reduction. Final percentage is accepted rows / 53 × 100. All 53 rows are accepted against the same frozen source; no partial credit or excluded requirements.

## Fixed normative checklist

| ID | Full requirement | Result/evidence |
| --- | --- | --- |
| N01 | Objective protocol designed before implementation; no universal expression DSL | PASS — docs/design.md; docs/measurement-contract.md (spec prepared before implementation); measurement.go Objective, no DSL |
| N02 | Objective preserves identity and revision | PASS — ObjectiveIdentity ID/Revision; Compare identity frozen and checked after callbacks; TestObjectiveProvenanceAndUnsupportedAggregation |
| N03 | Objective preserves native unit | PASS — ObjectiveIdentity.Unit; NumericObjective.Measure compares metric unit, native means preserved |
| N04 | Objective preserves declared scale and direction | PASS — ObjectiveIdentity finite Minimum/Maximum/ScaleRevision/Direction; numeric higher/lower comparison and conformance tests |
| N05 | Objective preserves eligibility and missingness | PASS — Objective Eligible and Measurement Present/Reason; TestMeasurementEmptyEligibilityAndContractConflict |
| N06 | Objective preserves aggregation revision | PASS — AggregationRevision validated as supported repeat-mean-case-mean-v1; unsupported aggregation test |
| N07 | Host owns domain meaning; compatibility validation and provenance are retained | PASS — docs/measurement-contract.md host semantics; compatibility before aggregation; sealed Objective in Comparison |
| N08 | Working assertion-pass reference objective | PASS — AssertionObjective via same Objective port; calculation/CRM fixtures, calculation example |
| N09 | Working numeric reference objective with explicit scale | PASS — NumericObjective; TestNumericObjectiveNativeScaleAndMissingness; TestNumericComparisonDirectionAndMatchedMeans |
| N10 | Unknown/non-applicable measurements remain missing, never synthetic zero | PASS — Measurement.Present, exclusions, MeanAvailable/MatchedMeansAvailable; unavailable report rendering; empty eligibility test |
| N11 | Heterogeneous scales/latency/currency/quality are not implicitly normalized or mixed | PASS — NumericObjective matches exact units/scale/revision/direction; no normalization or mixed metric expression |
| N12 | Usage/latency require actual measurements and matching units; no inference from reason codes | PASS — Objective sees persisted actual TrialRecord only; no implicit usage/latency estimator; docs/measurement-contract.md |
| N13 | Policy declares minimum matched independent cases | PASS — GatePolicy.MinimumMatchedCases validated >=1; comparison threshold |
| N14 | Policy declares minimum matched coverage | PASS — GatePolicy.MinimumMatchedCoverage validated [0,1]; comparison threshold |
| N15 | Matched denominator is common eligible declared set; missing trials cannot shrink it | PASS — Eligibility from Manifest.Cases before trials; MatchedEligible retained; 100-case adversarial test |
| N16 | Insufficient policy thresholds yield inconclusive | PASS — GateInconclusive on matched count, side coverage or matched coverage failure; empty eligibility test |
| N17 | Incompatible comparison yields invalid | PASS — GateInvalid on incompatible manifests and measurement/eligibility contract errors; invalid comparison sealed with reason |
| N18 | Host chooses independence thresholds; repeats never increase independent N | PASS — Case-level mean across all repeats; independent N is len(Pairs), threshold host-owned; repeats regression test |
| N19 | Report retains both side coverages | PASS — BaselineAggregate/CandidateAggregate Coverage in Comparison and Report |
| N20 | Report retains matched count and matched coverage | PASS — MatchedEligible/MatchedCases/MatchedCoverage in Comparison and Report |
| N21 | Report retains excluded cases and reasons | PASS — Aggregate.Excluded case/reason, including host eligibility reasons; Report emits per-side exclusions |
| N22 | Report retains objective identity and analysis unit | PASS — Comparison.Objective and Unit=case; Report objective id/revision/unit/direction/scale |
| N23 | Report retains uncertainty limitations | PASS — UncertaintyReason; Interval.Limitation; docs measurement/paired contracts |
| N24 | Confidence interval, delta and regression gate use identical matched set | PASS — Single differences slice from Pairs drives Delta, bootstrap and direction-aware regression; direction/matched mean tests |
| N25 | Paired execution actually interleaves sides by case/repeat with seeded order | PASS — RunPaired paired.go per-slot first/second execution; TestPairedScheduleMatchesActualDispatchAndRepeats |
| N26 | Separate lifecycle handles and explicit schedule provenance | PASS — PairSchedule full slots/seed/revision/experiment IDs in both manifests; separate runTrial lifecycle and handle uniqueness test |
| N27 | Paired execution preserves bounded concurrency and budget; no simultaneity promise | PASS — Global pair worker limit; TestPairedBoundedConcurrencyAndWithinSlotOrder; shared-budget test; documented timing limits |
| N28 | Shared/non-isolated environments cannot be interpreted as independent controlled comparisons | PASS — pairedConfigurationCompatible rejects shared lifecycle before effects; compatible requires Isolated; preflight test |
| N29 | Binary calibration retains TP/TN/FP/FN | PASS — CalibrationCounts TP/TN/FP/FN; TestCalibrationConfusionMissingAndGroups |
| N30 | Calibration retains unreviewed/error/abstain counts and coverage | PASS — CalibrationCounts Unreviewed/Errors/Abstentions/MissingLabels/MissingRecords; explicit coverage |
| N31 | Calibration rates have explicit denominators; zero denominator unavailable | PASS — CalibrationRate Numerator/Denominator/nullable Value; unavailable tests; validator recomputes rates |
| N32 | Host slice/group labels supported without raw input disclosure | PASS — Host Groups on CalibrationLabel; aggregate only group reports sorted, no raw inputs; group tests |
| N33 | No human disagreement assessment without corresponding human labels | PASS — Single human binary label per case, duplicates rejected; docs expressly exclude human disagreement claims |
| N34 | Case IDs do not prove independence; host group/time/content split and correlation responsibility documented | PASS — docs measurement/paired/calibration contracts: IDs not independence; host group/time/content splits and correlated case responsibility |
| N35 | Cluster bootstrap/sequential tests/multiple-comparison correction absent; limitations explicit | PASS — Explicit no cluster/sequential/multiple-comparison methods in docs; bootstrap limitation retained |

## Fixed acceptance checklist

| ID | Full requirement | Result/evidence |
| --- | --- | --- |
| A01 | 100 eligible/one matched pair yields inconclusive for minimum greater than one; true denominators reported | PASS — TestMatchedDenominatorAndIndependentRepeats: 100 eligible, 50 per side, one common, three repeats, inconclusive and 1/100 report |
| A02 | Repeats do not inflate independent N; missing grader/target/setup cannot improve coverage or become zero | PASS — Same test plus aggregate skips missing/failure/partial records before Measure; trial_missing/status/grading_incomplete exclusion paths; independent N case-only |
| A03 | Numeric higher/lower supported; incompatible units/scales/revisions/invalid values rejected; assertion fixture outcomes preserved | PASS — Numeric native-scale and higher/lower tests; unit/scale revision/direction/range/NaN rejection; assertion fixtures/examples preserved |
| A04 | Same matched set for delta/bootstrap; empty/degenerate interval reason; product thresholds not significance claims | PASS — One shared pair/differences loop; empty/one/constant reasons; docs thresholds are product policy, not significance |
| A05 | Fake target demonstrates actual interleaving, isolation, seed reproducibility under declared concurrency; cancellation retains completed trials | PASS — paired_execution_test.go actual dispatch, fresh handles, reproducibility, concurrency, cancellation/budget/infrastructure retained slots and paid usage |
| A06 | Deliberate FP/FN/missing-label calibration matrix and coverage verifiable; zero denominator not numeric | PASS — calibration_test.go exact TP/TN/FP/FN 1 each; reviewed 4/eligible8; missing labels/records; group denominators; unavailable rates |
| A07 | Measurement/calibration conformance, schemas, examples; old hardcoded comparison bypass removed | PASS — conformance/measurement.go and measurement_test.go reference Objective/Calibration checks; 15-kind generated corpus; examples/calculation/protocols/optimizer; Compare has mandatory Objective, no hardcoded branch |

## Fixed common checklist

| ID | Full requirement | Result/evidence |
| --- | --- | --- |
| C01 | Clear break updates all internal consumers/examples/tests | PASS — Compare mandatory Objective; calibration API replaced; all consumers, optimizer, CLI fixtures, examples/test/schema consumers updated |
| C02 | Replaced APIs/fields/aliases/dual paths/legacy readers/shims removed | PASS — MeanPassRate/PairOrder/old calibration fields removed; old wire schemas removed; no aliases or compatibility readers |
| C03 | Spec-first design covers affected inputs/results/states/errors/ownership/identity/cancellation/limits | PASS — docs/design.md links normative measurement/paired/calibration contracts with identity/ownership/errors/missingness/schedule/cancellation/budget limits |
| C04 | Incompatible wire semantics versioned; current schemas/fixtures/runtime synchronized; old versions unsupported | PASS — comparison2, experiment/assessment/observation-result/search3, calibration1; schema registry/corpus/runtime versions synchronized; previous versions rejected |
| C05 | BYOT preserved without mandatory agent/message/prompt models or arbitrary domain reflection copy | PASS — Generic domains remain host-owned; Objective operates permitted service TrialRecord; no agent/message/prompt model or domain copy contract introduced |
| C06 | Core stdlib only, no network/vendor/credentials/optional-package imports | PASS — Root imports stdlib; optional packages consume root; go.mod no runtime external dependencies |
| C07 | Host/domain/environment/durable budget/retention/sampling/independence boundaries respected | PASS — docs contracts preserve domain/unit/group/sampling independence, budget/retention/environment host responsibility |
| C08 | No forced callback termination/exactly-once/universal URL secrecy/ID independence/scripted judge reliability promises | PASS — docs contracts retain cooperative cancellation and provider effects limits; schedule/IDs explicitly not independence proof; scripted fixtures not real judge guarantees |
| C09 | AAA observable/adversarial tests; new ports have working reference implementation and conformance | PASS — AAA measurement/calibration/paired observable/adversarial tests; ObjectiveFuncs/AssertionObjective/NumericObjective and public reusable conformance helpers |
| C10 | Updated docs/schema/implementation/regressions/verification; independent make validate and all six examples on final frozen source | PASS — Independent `make validate` exit 0, both modules/race; all six examples exit 0; source fingerprint unchanged |
| C11 | No weakened checks/style detour/foreign cache deletion; actual results retained; ignored task files not force-added | PASS — git diff --check PASS; reviewed no weakened checks/foreign deletions/force-added ignored task files; no scope style refactor |

## Final validation

Frozen source fingerprint: `554d9261c1d27994f427afc2b8485260dd5f3f8da475da86187922f0282a0bdc`, **108 files**, computed with `/private/tmp/evaly-review-state.py`. It includes `.cursor/task/README.md` and tasks 01–03; review reports and validation journal are excluded. Fingerprint checked before and after verification and remained identical.

Independent checks performed by this reviewer:

- `env GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate`: **exit 0**. Formatting, vet, verbose race tests in root and isolated `contracttest` all passed. Log: `/private/tmp/evaly-measurement-completeness-validation.log`. Independent contract module completed in 102.782 seconds; generated-schema drift and shared structural corpus passed.
- Each `go run ./examples/{calculation,crm,protocols,http,observation,optimizer}`: **exit 0**. Logs: `/private/tmp/evaly-measurement-completeness-example-<name>.log`.
- `git diff --check`: **exit 0**.
- Core import inspection confirms only stdlib and its own stdlib-only internal wire contract. An initial inspection without the configured temporary GOCACHE encountered sandbox cache access denial; rerun with the validation GOCACHE passed. This did not affect actual validation.

Earlier review gaps (metadata conflicts lacked invalid verdict, reference identity omitted assertion policy, constant intervals lacked explicit degeneration reason) are closed in this frozen source and protected by measurement tests. No unfulfilled requirement remains. This percentage addresses completeness against this task, not a claim that all possible implementation errors are absent; the separate correctness reviewer provides that acceptance.

Implementation files were not edited by this reviewer; only this acceptance report was created/updated.
