# HTTP JSON protocol v2

`adapters/httpjson` provides a bounded, optional bridge. `go run ./examples/http`
runs the local reference server and client. Core has no HTTP dependency.

POST `application/json` requires `version=2`, codec identities, consumer-encoded
`input`, and `trial` containing `id`, `case_revision`, `seed`, `fixture`, `reset`.
Service fields are required and non-null. Raw `input` is required but may be any
JSON value, including null, when the host codec admits it. Seed zero is valid;
omission is invalid.
Unknown fields, duplicate keys, malformed JSON and unsupported versions reject
before invocation. Codec identities and nonempty trial identities are semantic
checks. The host owns fixture/reset compatibility and environment isolation.

The response requires `version=2`, `status` (`completed` or `target_error`),
`usage`, `events`, `capabilities`, and `evidence={complete,reason}`. `output` is
required for completed execution and absent for target_error. Events may be an
empty array or null; both mean zero delivered events. Domain output remains the
host codec's responsibility. Envelope structural validity is checked with the
shared runtime wire decoder; version/status/capability checks precede trust in
usage or events. Capabilities retain their independent version 1.

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
read, size, malformed envelope, unsupported version/status/capability and event
retention errors mark capture incomplete with fixed labels. These failures cannot
prove absence of an external action. A domain output decode failure after a valid
complete response does not erase already delivered evidence or its completeness.
Neither context cancellation nor bounded delivery forcibly stops a host callback.

Response Content-Type must be `application/json` (parameters are allowed) before
trusting its envelope. Request/response limits must be positive and below MaxInt64,
so the one-byte overflow probe cannot overflow. Limits are configured by each host. No streaming, retry, outbox,
authorization, secret detection, or universal agent/output model is provided.
Version 1 and the old tuple callback are removed without a compatibility reader.
