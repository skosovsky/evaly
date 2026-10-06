# HTTP JSON protocol v3

`adapters/httpjson` provides a bounded, optional bridge. `go run ./examples/http`
runs the local reference server and client. Core has no HTTP dependency.

POST `application/json` requires `version=3`, codec identities, consumer-encoded
`input`, and `trial` containing `id`, `case_revision`, `seed`, `fixture`, `reset`.
Service fields are required and non-null. Raw `input` is required but may be any
JSON value, including null, when the host codec admits it. Seed zero is valid;
omission is invalid.
Unknown fields, duplicate keys, malformed JSON and unsupported versions reject
before invocation. Codec identities and nonempty trial identities are semantic
checks. The host owns fixture/reset compatibility and environment isolation.

The response requires `version=3`, `status` (`completed` or `target_error`),
`usage`, `events`, and `evidence={complete,reason}`. `output` is
required for completed execution and absent for target_error. Events may be an
empty array or null; both mean zero delivered events. Domain output remains the
host codec's responsibility. Envelope structural validity is checked with the
shared runtime wire decoder; version/status checks precede trust in
usage or events.

The handler callback returns `Invocation[O]` and an error. Invocation retains
host-owned output, usage and events plus explicit `EvidenceDelivery`. The host
must declare whether all relevant events were delivered: `complete=true` requires
an empty reason; `complete=false` requires a nonempty reason. This declaration
is independent of target success. A target_error with complete evidence preserves
usage/events and returns `evaly.ErrTarget`; it does not imply an incomplete trace.
Incomplete reasons are classification labels and must not contain secrets.

After validating the protocol, the client retains usage and passes available
events through capture policy, then marks declared incomplete evidence, then
returns target failure or decodes successful output. Transport, HTTP status,
read, size, malformed envelope, unsupported version/status and event
retention errors mark capture incomplete with fixed labels. These failures cannot
prove absence of an external action. A domain output decode failure after a valid
complete response does not erase already delivered evidence or its completeness.
Neither context cancellation nor bounded delivery forcibly stops a host callback.

Response Content-Type must be `application/json` (parameters are allowed) before
trusting its envelope. Request/response limits must be positive and below MaxInt64,
so the one-byte overflow probe cannot overflow. Limits are configured by each host. No streaming, retry, outbox,
authorization, secret detection, or universal agent/output model is provided.
Versions 1/2 and the old Handler constructor are removed without a compatibility reader.


D55: construct with `handler, err := httpjson.NewHandler(inputCodec, outputCodec,
maxBytes, invoke)` and handle the error before registering the handler or starting
a server. Nil/typed-nil codecs, StructuralValidator errors, invalid codec identities,
nil Invoke and invalid MaxBytes fail construction without decoding, encoding or
invoking the host. Structural validation covers both peers before any Identity
call. The handler captures codec identities; codecs must keep validity, identity,
reversibility and concurrency behavior stable for its lifetime.

D52: already-cancelled requests return 408 before reading the body or running
codecs/Invoke. Guards after reading and decoding avoid dispatch when cancellation
became known during those stages. A client also checks cancellation before input
encoding and again before HTTP dispatch. These checks do not atomically synchronize
cancellation with a host effect or forcibly terminate a callback already running.
Runtime request errors remain separate: malformed wire/read failure or invalid
content type/method/body is 400, codec/trial binding mismatch is 422, oversized
request is 413, and invalid/oversized callback response is 500. Invoke errors with
a valid partial Invocation remain protocol `target_error` responses, preserving
available usage and evidence.

D53: HTTP v3 removes Response.Capabilities. Outcome, rich target status and evidence
shape are guaranteed by this wire version; fixture/reset identities belong to the
request and Invoke's host compatibility checks. The target response carries no
framework grader metrics, so MetricScales could not certify such a mapping.
Domain output remains host-codec owned. Generic evaly.InteropCapabilities and
CheckMapping remain available for independent mappings; HTTP does not claim them.

D54: one MaxBytes per endpoint still governs both directions. Reads inspect at most
MaxBytes+1 wire bytes; input/output encoding and callbacks can allocate larger values
before the encoded-size check. MaxBytes is a wire acceptance limit, not a total RAM
limit for buffers, domain objects or host effects. Hosts supply their own byte/
resource limits where necessary; no transport-policy framework is introduced.

Migration: use fallible NewHandler; both request/response schemas are v3 and older
versions reject. Remove capabilities from response producers; a legacy field on a
v3 record is an unknown-field error. The reference HTTP example handles construction
failure and the same client protocol is used by workflow integration.
