# HTTP JSON protocol v1

Implementation: `adapters/httpjson`. Runnable local endpoint and client:
`go run ./examples/http`. Contract tests use real local HTTP sockets and the same
`conformance.Target` suite as the in-process calculation target. No external network
or credentials are used.

POST application/json request fields: `version=1`, `input_codec={id,version}`,
`output_codec={id,version}`, `input` (consumer-encoded JSON), and `trial` with
`id`, `case_revision`, `seed`, `fixture`, `reset`. Trial ID is the external namespace;
the server owns separate environment setup and must check fixture/reset compatibility.
Requests/responses are bounded by MaxBytes. Context cancellation closes client work.

Response fields: `version=1`, `status` (`completed` or `target_error`), `output`
(consumer-encoded JSON), `usage={known,units}`, `events` (versioned Event records),
`capabilities` (Version, Outcome, ResetIdentity, Evidence, RichStatus, MetricScales).
Event payloads pass through the caller's capture policy before any retention.
Server-side domain errors return a target_error response; client reports a target
error and retains observed usage. Unknown versions/statuses, omitted guarantee
capabilities, oversized data and malformed/duplicate-key JSON are explicit errors.

Loss mapping: missing outcome/reset/evidence/status/scale capability is rejected
using `CheckMapping`; boolean-only backends do not receive fabricated scores.
No wire domain conversion is implicit; mismatched codec identities reject requests.
Envelope extensions apply to portable store artifacts, not this strict HTTP protocol.
