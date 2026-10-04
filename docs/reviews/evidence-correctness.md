# TASK02 independent correctness acceptance

Reviewer: `accept_evidence_correctness`. This reviewer did not implement task02 and did not edit implementation, contracts, schemas, or tests. Authority is the complete task02 and common task README.

Verdict: **PASS — no identified unclosed defects** on frozen source fingerprint `aabce87e5dec89ddeaeaae3592c44f0319d23260e150ec63242f9c63819c27b0` (93 files). The fingerprint was independently checked before and after validation using `/private/tmp/evaly-review-state.py`; review reports and the validation journal are excluded. This verdict concerns the stated task scope and tested behavior, not a claim that every possible defect is absent.

## Reproduced findings and closure

1. Preliminary implementation accepted malformed percent escapes in opaque URI references (`urn:artifact%ZZ`, `urn:artifact%`, `urn:artifact%0G`). A retaining capture sealed these values because url.Parse does not check opaque escapes. After repair, independently rerunning `/private/tmp/evaly-ref-review.go` rejects all three before retention with a fixed contract error; malformed HTTP and relative references also reject. Shared reference checking covers capture projection, restored evidence and grade references. The positive corpus preserves permitted opaque/artifact references and default dropping.
2. Preliminary schema/runtime number disagreement: comparison aggregate coverage `1e400` passed the independent schema but failed typed decoding. Frozen generated float bounds and runtime checks now reject it in both paths. `/private/tmp/evaly-float-review.go` reproduced the rejection using the independent jsonschema engine with json.Number parsing. A small HTTP request containing an enormous numeric exponent rejects promptly; no practical allocation blowup was identified in that check.
3. Preliminary timestamp disagreement: Go parsing accepted `+24:00` and `+00:60` offsets rejected by asserted JSON Schema format, while leap-second and lowercase separator forms passed the schema and failed Go parsing. Frozen shared timestamp pattern plus format assertion rejects all four consistently. `/private/tmp/evaly-date-review.go` independently checked these cases and an invalid date string. Independent corpus format assertions are enabled.

All three findings were communicated to the implementer, repaired by others, and rerun successfully on the accepted fingerprint. None remains open.

## Independent checks

- `env GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate` completed with exit 0. Log: `/private/tmp/evaly-evidence-correctness-validation.log`. Formatting, vet, root and independent contracttest race checks passed.
- All six examples (`calculation`, `crm`, `protocols`, `http`, `observation`, `optimizer`) completed independently with exit 0. Logs: `/private/tmp/evaly-evidence-correctness-example-<name>.log`.
- `git diff --check` passed, and the frozen fingerprint remained unchanged after the commands.
- Read the full task, common rules, evidence/wire/design/HTTP contracts, implementation changes, reference/export regressions, HTTP failure tests and all-kind structural corpus.

## Correctness assessment

HTTP validates structural shape and supported protocol before trusting usage/events. Missing/null/duplicate required seed does not invoke the server target; explicit zero does. Target errors preserve permitted delivered events and known usage, and remain distinct from unsupported protocol. Transport/read/size/malformed/unsupported responses mark delivery incomplete and do not turn empty records into proof of absence. Explicit complete delivery remains independent of target success. A domain output decoding failure after proven event delivery retains that evidence completeness; this matches the contract rather than silently discarding the delivered effects. Retention failure preserves the accepted event prefix and usage while making coverage incomplete.

The registered fourteen wire roots share stdlib structural definitions with generated schemas. Independent positive/negative mutations cover required/null/zero/unknown fields, numeric representability, enums, versions and duplicate byte keys. Schema generation checks filenames and bytes in a disposable directory. RawMessage/domain codecs remain host-owned; reflection does not copy arbitrary host types. Structural decoding is followed by semantic Restore/Validate checks where the API provides them; checksum corruption is explicitly separate. Removed OutcomeRefs has no remaining source/legacy channel.

Export keeps invalid/conflict/unsupported/cancelled/delivery_failure distinct, avoids dispatch on pre-cancellation, and gives sinks a detached artifact. Source verdict/data remains immutable across failed, retried or conflicting deliveries. It introduces neither core transport nor retry/outbox behavior.

## Limits

No external infrastructure was contacted for acceptance. HTTP uses the reference handler and local httptest transport; export uses reference/adversarial sinks. Secret detection is deliberately bounded syntax/credential-name checking with host-owned retention classification, not DLP. Typed Restore cannot recover JSON presence lost by a caller that bypassed DecodeWire; this boundary is explicit and library byte-loading paths use DecodeWire. Byte limits remain transport/store owned. Reviews identify defects through code inspection and adversarial checks, not exhaustive proof.
