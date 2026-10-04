# Evidence references and export delivery contract

Reference admissibility is a syntactic and bounded credential check, distinct
from permission to retain a value. Capture defaults to dropping references and
payload. The host policy classifies allowed references and nested payload values;
the host projection classifies domain input, output and reference values.

Nonempty artifact IDs and URI references, including non-HTTP schemes and opaque
URIs, are supported. Invalid UTF-8, whitespace, control characters, malformed percent escapes,
userinfo and literal fragment delimiters are rejected. Fragments are prohibited
entirely, including an empty fragment, to avoid a second unchecked credential
channel. HTTP and HTTPS URIs require a host and cannot use opaque syntax.
Query parsing must succeed without discarded parameters. Decoded query names
containing token, secret, key, password, passwd, credential, authorization or
signature, and the names auth, pwd and sig, are rejected case-insensitively.
This bounded check does not discover arbitrary secrets in paths, opaque values,
unrecognized query names or domain payloads. Host retention authorization remains
necessary even when syntax passes.

The same check applies after policy projection, when validating restored evidence,
and when validating grade references. Rejections use contract errors and fixed
diagnostic codes; they do not include the rejected reference or sink error text.
These corrections enforce the existing evidence-v1 privacy invariant; they do
not add a new evidence layout or legacy reader. Outcome references have exactly
one retention path: policy-controlled evidence events. TargetResult contains
only the typed output and usage.

Export preserves the source artifact and separates delivery from quality verdict.
Delivery State is delivered or failed. Failed Reason is one of invalid, conflict,
unsupported, cancelled, delivery_failure; a delivered result has no Reason.
Reasons classify errors returned by sinks with errors.Is: invalid/corrupt/unsealed
contracts are invalid, identity conflicts are conflict, unsupported mappings are
unsupported, context cancellation/deadlines are cancelled, and all other failures
are delivery_failure. A cancelled context prevents dispatch. Sink capability and
artifact validation happen before delivery; typed-nil sinks are invalid. The sink
receives a detached artifact copy, so mutation cannot change the caller's artifact.
Deduplication semantics and stable observation ID are retained across attempts.
MemoryExport's configured failure represents delivery_failure, while an ID reused
with different content returns conflict. Evaly performs no retry or outbox work;
the host decides whether and when each class merits another attempt.

Delivery is an in-process service result, not a registered persisted artifact
kind. The fixed reason codes replace the previous lossy sink_unavailable code
without an alias. Evidence and grade reference changes reject previously accepted
malformed or credential-bearing values; backward compatibility is not provided.
