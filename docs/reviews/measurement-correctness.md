# TASK03 independent correctness acceptance

Reviewer: `accept_measurement_correctness`. This reviewer did not implement task03 and did not change implementation, specifications, schemas or tests. Review authority is the complete task03 and common task README.

Verdict: **PASS — no identified unclosed defects** on frozen source fingerprint `554d9261c1d27994f427afc2b8485260dd5f3f8da475da86187922f0282a0bdc` (108 files). The fingerprint was independently checked before and after validation with `/private/tmp/evaly-review-state.py`; reports and the validation journal are excluded. This is a scope-specific review result, not a proof that every possible defect is absent.

## Review findings and closure

Pre-freeze inspection identified risks in summing finite measurements, unrestricted aggregation provenance, concrete-type assertion diagnostics and returning an unclassified partial report on measurement incompatibility. These were communicated to the parent and addressed by implementation authors. The frozen contract supports only the declared arithmetic aggregation revision, rejects unrepresentable scale spans, uses stable online means, removes the old assertion-only aggregate diagnostics and seals invalid comparisons for measurement/eligibility contract failures. Assertion policy and numeric source selection are retained in objective identity.

The parent independently found identical subnormal values becoming zero through weighted averaging. The reviewer also identified the corresponding bootstrap underflow risk. Repairs use the same stable mean across repeats, matched cases and bootstrap, with wider stdlib arithmetic for opposite-sign updates whose subtraction overflows. Independent external reproduction on the frozen tree verifies all of the following:

- Four cases with two repeats and a present value of `1e308` retain the finite mean `1e308` and delta zero.
- A measurement callback returning `ErrConflict` yields a sealed `invalid_comparison` with `measurement_contract_invalid`, rather than a misleading pass or an empty verdict.
- Host exclusion of every case yields inconclusive, eligible count zero, unavailable means and `no_matched_cases`.
- Baseline zero and candidate `math.SmallestNonzeroFloat64` retain candidate mean, delta and both bootstrap endpoints exactly `5e-324` across two repeats and four cases.
- Opposite paired deltas of `-1e308` and `1e308` retain finite matched means, delta zero and bootstrap endpoints `[-1e308, 1e308]`.
- A pointer to the reference assertion objective works through the same comparison path.

Reproduction source: `/private/tmp/evaly-measurement-adversarial.go`. The reviewer did not edit repository tests to obtain these results. No review finding remains open.

## Independent validation

- `env GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate` completed with exit 0 on the frozen tree. Log: `/private/tmp/evaly-measurement-correctness-validate.log`. Formatting, vet and race checks passed in the root and separate contracttest modules.
- All six examples (`calculation`, `crm`, `protocols`, `http`, `observation`, `optimizer`) independently completed with exit 0. Logs: `/private/tmp/evaly-measurement-correctness-example-<name>.log`.
- The standalone adversarial reproduction above completed with exit 0 on the frozen tree.
- `git diff --check` passed and the source fingerprint remained unchanged.
- Reviewed measurement/comparison, paired execution and calibration implementations and contracts, their regressions and conformance, all-kind wire corpus, version changes and minimal optimizer integration.

## Correctness assessment

The 100-eligible/one-matched regression reports both side coverages, matched denominator 100 and matched coverage .01, and is inconclusive under a minimum of two cases. Three repeats do not increase independent N. Final attempts with missing trials, failed targets, partial grading or absent measurements stay excluded with reasons. Higher/lower directions use native scales, and unit/scale/revision/direction incompatibility never receives an invented score. Reported matched means, delta, regression and bootstrap share exactly the same matched case set. Zero means have availability flags; empty, one-case and constant-sample uncertainty have explicit reasons.

Paired execution schedules case/repeat slots rather than whole experiments. Both configurations are validated before effects, each side prepares/resets/cleans up its own handle, and the same sealed schedule is present in both artifacts. Tests compare actual target dispatch against scheduled side order, reproduce seed -17 with concurrency one, bound total active callbacks with concurrency three and retain every scheduled side after cancellation, shared-budget exhaustion and infrastructure stop. Forged algorithm/seed/limit/order/side/slot/isolation/pair provenance is rejected. The contract correctly limits reproducibility across concurrent workers and leaves the truth of host isolation to the host.

Calibration deliberately exercises every confusion cell plus missing labels, missing records, errors, abstentions and overlapping groups. Coverage and accuracy retain their actual denominators; unavailable rates remain null. Numeric-only grades do not become binary predictions. Duplicate/malformed labels and records, mismatched grader revisions and report mutations reject. Group reports contain opaque group names and aggregate counts/rates, without raw inputs or explanations.

Comparison v2 and experiment/assessment/observation-result/search v3 are synchronized through runtime and generated wire schemas; replaced versions are rejected. Calibration is a new version-1 wire kind. Independent contracttest mutations and disposable schema generation checks pass. Assertion and numeric objectives share one public contract and have reference conformance coverage. Optimizer receives an explicit objective and validates lower-direction selection in its existing limited integration; feedback and broader optimizer redesign remain task04.

## Limits

The tests use offline scripted targets, local reference adapters and independent structural validation. They do not demonstrate provider determinism, causal effects, statistical independence, representative data or universal judge reliability. Product thresholds are not significance tests. Host-owned group/time/content splits, correlated cases and bounded float64 precision remain explicit; no cluster bootstrap, sequential test or multiple-comparison correction is implied.
