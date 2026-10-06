# CLI comparison contract

`evaly compare --store DIR --baseline ID --candidate ID --policy FILE` requires
an explicit JSON `ComparisonPolicy` version 1. The policy contains `version`,
`objective_kind` (`assertion` or `numeric`), a complete task-03 `objective`
identity, and `gate`. No policy is inferred from target names or fixtures.

Assertion descriptions must exactly match the built-in assertion objective
identity, including `all`/`any` policy and fixed binary scale. Numeric descriptions
require grader and metric identities, the native unit, bounds, direction and
scale revision. Both supported objectives use `all-declared-v1` eligibility,
`all-repeats-required-v1` missingness and `repeat-mean-case-mean-v1` aggregation.
Other kinds or callback-specific eligibility/missingness revisions are rejected
as unsupported; Go callbacks remain available through the library API.

Structural JSON decoding rejects unknown or duplicate fields, missing required
fields, null scalars and old versions. Semantic validation precedes artifact
loading. A comparison uses its declared objective and gate, publishes the sealed
comparison and exits 0 (pass), 1 (fail), 2 (inconclusive), or 3 (invalid arguments,
unsupported policy, invalid comparison or infrastructure failure).

Output reports each trial's case and revision, seed, target revision, fixture,
reset and evidence revision. These are host replay coordinates, not runnable
commands. The fixture subcommand remains an explicit calculation demonstration;
compare never invents fixture replay commands for arbitrary targets.

An explicit policy for the offline calculation fixture (save as `policy.json`):

```json
{
  "version": 1,
  "objective_kind": "assertion",
  "objective": {
    "id": "assertion-pass",
    "revision": "1",
    "assertion_policy": "all",
    "unit": "pass_fraction",
    "scale_revision": "binary-v1",
    "minimum": 0,
    "maximum": 1,
    "direction": "higher",
    "eligibility_revision": "all-declared-v1",
    "missingness_revision": "all-repeats-required-v1",
    "aggregation_revision": "repeat-mean-case-mean-v1"
  },
  "gate": {
    "revision": "fixture-v1",
    "minimum_matched_cases": 1,
    "minimum_matched_coverage": 1,
    "minimum_coverage": 1,
    "minimum_quality": 0.8,
    "maximum_regression": 0,
    "bootstrap_samples": 1000,
    "seed": 42
  }
}
```

The numeric variant sets `objective_kind` to `numeric`, removes
`assertion_policy`, and adds `source_grader` and `source_metric` to `objective`.
Its native unit/bounds/direction/scale must match the chosen persisted metric;
`minimum_quality` is interpreted in that unit, including the `lower` direction.


D56: each subcommand registers only its own flags. Both accept `--store`;
fixture accepts `--id`, `--behavior`, `--seed`, `--case`; compare accepts
`--baseline`, `--candidate`, `--policy`. Unknown commands, irrelevant flags,
positionals, missing required values and invalid IDs/fixture behavior reject with
exit 3 before opening/creating a store. Compare policy decoding/resolution also
precedes opening the store. Existing comparison exit classes and replay output
are unchanged; flags previously accepted but ignored are now errors.
