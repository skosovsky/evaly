# Optional host recipes

This is a separate consumer Go module; core does not import its SDK dependencies.
The shared release protocol includes it under `integrations/recipes/vX.Y.Z` for
future releases. Historical releases through v0.4.0 contain reference sources
in the root archive but have no separately published recipes tag.
The host owns configuration, transport and persistent delivery. The adapter API
is described in [the normative contract](../../docs/composition-contract.md).

From the root checkout:

```sh
make test
make test-integration
make test-e2e
```

Development replaces evaly with the current core. The integration consumer matrix
copies recipes into temporary modules and checks published SDK pins, clean audited
SDK source SHAs and explicitly unsupported old SDK snapshots. The published lane
removes all replacements. Source lanes fetch exact Git objects, without sibling
working trees. All semantic fixtures use deterministic local sources and in-memory
SDK exporters; credentials and live model quality are outside this proof.
See [verification](../../docs/verification.md) for exact source provenance.

Run the streaming example or complete disk/export roundtrip directly:

```sh
cd integrations/recipes
GOWORK=off go test -run ExampleStreamingTarget -v
GOWORK=off go test -race -count=1 -tags=e2e -run '^TestE2ERoundtripAndRealEvaluationDelivery' -v
```

## Consumer changes

Before: return a lazy handle from Target.Run, cast token counters to Units, infer
capture completeness from sampled telemetry, or publish an experiment as one
boolean evaluation.

After: construct `StreamingTarget[I,O,E]`. Bind the supplied context in `Open`,
synchronously record permitted domain effects in `Observe`, supply `Project` for
the terminal output, and explicitly declare source completeness in `Complete`.
`CloseTransport`, when needed, releases the host transport after source unwind;
its failure is joined with source/cancellation errors without erasing receipts.
The SDK stream is always closed even when this callback fails. Do not return
remote/lazy handles as evaluated output. Refusal and unresolved tool calls require
an explicit host domain target; this reference accepts completed output only.

Use the same immutable conversion for target, judge and reservation:

```go
policy := recipes.Conversion{
    Identity: "billing-total", SourceUnit: "token", Unit: "credit",
    Mode: "total", TotalRate: 1, Bound: 10,
}
upper := prompty.Usage{TotalTokens: 6, Known: []prompty.UsageCounter{prompty.UsageTotal}}
units, err := policy.ReservationUnits("token", upper)
// Handle err before configuring RunPlan.DispatchUnits; reserve before dispatch.
// RunConfig.Provenance.Provider records adapter and policy identities.
```

Rates use their exact binary-rational float representation; results round upward
to whole budget units and then upward to a representable float receipt. Choose
rates/bounds accordingly. Total mode requires known total; component mode requires
known input and output and never fabricates them. Do not sum cache/reasoning
breakdowns or cumulative stream snapshots again. Unknown keeps the reservation;
a known zero may settle to zero. Error does not prove no billable dispatch.

`JudgeAdapter` receives trusted instructions and typed projected data as separate
callback arguments. Its callback must finish execution and return final usage
with errors. Return `InsufficientEvidence` for an unavailable measurement; a
negative assertion is `Scored` with `Pass:false`. Structured decode failure is an
error, with usage retained. This separation is not a model injection guarantee.

`CaptureBridge.Record` preserves original sequence and trial correlation. Call
`Finish` with the actual source CaptureReport; a missing/failed/incomplete report
or sampled source marks evidence incomplete. Mark omitted or truncated events
incomplete even if your local numbering is contiguous. Classify nested payloads
and references before retention, and require the relevant capture kinds in the
host gate. Grade real domain outcomes separately from generated text.

Export uses `EvaluationSink` with a required privacy projection and atomic
`DeliverOne` callback. `ProjectEvaluations` omits raw grade reasons, usage,
trajectory and payloads by design. Each assertion/metric/status has its own
stable ID. Identity and measurement fields may not be rewritten by projection;
association/provenance/artifact references may be removed, never replaced with
foreign identities. Remove the whole association tuple together. Reasoning can be
redacted. Metric unit/scale/range/direction
remain in the host EvaluationRecord sidecar because the telemetry DTO has no
fields for them. Retain the artifact/sidecar as semantic authority.

`LocalRegistry{Recorder: tracker.EvaluationRecorder()}` is the process-local dedup
reference. Production uses a host durable registry; telemetry is not an outbox.
Retry partial or ambiguous delivery with the same policy and IDs. Reusing an ID
with changed projected content is a conflict. Delivery success means SDK
acceptance, not a durable collector acknowledgment. Preserve artifact verdicts
regardless of delivery. Artifact privacy is independent of sink privacy.

## Unsupported source sets

The audited SDK source SHAs are pinned in Go integration fixtures, rather than
moving default branches. The unsupported lane checks that the legacy prompty
snapshot lacks Stream, records the missing capabilities, and does not run semantic
fixtures. Missing network/Git/Go prerequisites fail the selected profile. The
negative result proves the unsupported boundary and does not count as a successful
stream/evaluation integration. Update source pins only after rerunning the full
semantic matrix against the intended new source set.
