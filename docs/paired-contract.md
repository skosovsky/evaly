# Paired execution contract, revision 3

Experiment wire revision 3 replaces experiment-wide `pair_order` with a sealed
`pair_schedule`. Previous experiment revisions are unsupported. `RunConfig` does
not let callers attach a synthetic pairing claim to an ordinary run.

`RunPaired` validates both configurations before any lifecycle, target or judge
effect, then requires identical dataset revisions, repeats, seed and concurrency,
distinct experiment IDs, a valid pair ID, and isolated lifecycle identities on
both sides. The entire run plan, fixture/reset identities, output codec, grader
revisions, projection and evidence capture contract must also match before
dispatch. Host promises that each Prepare returns an independent environment;
evaly allocates different trial IDs and invokes Prepare/Reset/Cleanup separately.
Shared lifecycle environments are rejected for paired controlled comparisons.

The schedule identifies the algorithm (`case-repeat-v1`), seed, global worker
limit, both experiment IDs, and every case/repeat slot in dataset order. A fixed
SplitMix64 computation of seed and slot ordinal chooses each slot's first side.
Every worker executes the first side, including its allowed setup retries and
cleanup, then the second side of that same slot before accepting another slot.
There are at most Concurrency active slots and target callbacks in total. The
records in both manifests contain the same complete schedule, including cancelled
or budget-blocked slots. Record order remains case/repeat/attempt order per side.

With concurrency one, the actual target dispatch order is reproducible whenever
the same callbacks reach dispatch. With larger concurrency, assignment order and
within-slot side order are reproducible, while timing between independent workers
is not: neither simultaneity nor a deterministic global completion order is
promised. Seeds do not make providers deterministic. A shared Budget port may be
provided to enforce one budget across both sides; separate ports enforce separate
limits. Evaly does not invent a durable budget or transactional provider effects.

Cancellation and StopOnInfrastructure apply to the whole paired run. Already
obtained usage, grades and trials remain retained; every undispatched scheduled
side has its own cancelled/infrastructure-stop record. One side failing quality
does not cancel its partner. Retry, evidence and cleanup semantics otherwise use
the execution revision-2 contract. A schedule is provenance of dispatch policy,
not evidence that cases are statistically independent; the host owns content,
group and time splits and must account for correlated cases.
