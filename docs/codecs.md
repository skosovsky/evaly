# JSON codec representation and ownership

JSONCodec decodes numbers in interface-valued objects and arrays as `json.Number`,
including nested values. Integers above 2^53 retain their exact value. Typed int
and struct decoding still uses encoding/json conversion and rejects overflow.
Valid generic numeric tokens may exceed float64 magnitude; `1e400` is a finite
decimal JSON value retained as a token, not converted into an infinite float.
Go float NaN and infinity remain invalid for encoding.

Canonical identity sorts object keys and preserves numeric lexemes: `1`, `1.0`,
`1e0` and `1E+0` have distinct canonical representations. JSONCodec.Encode checks
Decode→re-encode canonical equality before returning success. A custom marshaler
whose output cannot be decoded and re-encoded through its declared type is rejected
with ErrInvalid before sealing. This requires custom callbacks to be stable; Encode
can invoke a marshaler twice and its unmarshal counterpart once. Dynamic Go types
such as int64 inside `any` are not retained across decode; their numeric value is.

## Cycle and UTF-8 validation decision (D23)

The standard JSON encoder owns field selection and actual cycle rejection. The
previous custom cycle detector is removed. A struct pointer and a pointer to its
first field, empty overlapping slices and shared references in separate branches
are valid. Real serialized self-pointer/map/slice cycles are rejected by the
encoder and classified as ErrInvalid.

Reflection remains only to reject source strings that encoding/json would silently
replace when they contain invalid UTF-8. Its visited identities include type and
slice length and only bound traversal; they do not classify a repeated node as a
JSON cycle. Fields with `json:"-"` are excluded, including cyclic caches and invalid
source strings. Custom MarshalJSON implementations own source-string validation;
their emitted JSON is still canonically validated and checked for reversibility.

The UTF-8 source check is deliberately conservative for other exported fields,
including fields shadowed by embedded-field dominance and omitted by custom
`omitzero` methods. Invalid strings there may be rejected even when the standard
encoder omits them. This preserves the strict source-string guarantee without a
second incomplete implementation of encoding/json field selection. Use an explicit
domain Codec or `json:"-"` for nonserialized invalid-byte data. Unexported fields
are not traversed, except anonymous structs whose exported children can be
serialized by encoding/json. String map keys remain the stock object-key contract.
No claim of identical acceptance for every encoding/json Go type is made.

## Byte and value ownership decision (D24)

Codec.Encode transfers caller-owned bytes stable after return. A host codec that
reuses and overwrites its returned buffer on the next Encode violates the contract.
Decode returns fresh owned mutable values without retaining or mutating input.
These promises apply when one codec implementation serves input/output/reference
roles; helpers may retain a returned slice while invoking another codec.
JSONCodec satisfies them, and snapshot/saved-record boundaries own canonical bytes.
Shared implementations must be safe for concurrent calls and stable in identity,
validation and behavior. No reflection clone or sandbox of host codecs is implied.

Migration: generic consumers should assert `json.Number` rather than float64 and
choose exact integer/decimal conversion for their domain. Numeric lexemes, schemas
and codec identity fields do not change for already supported reversible values.
Previously admitted nonreversible custom representations now fail early; provide
a matching custom unmarshal implementation or an explicit reversible domain codec.
No legacy decoding fallback or automatic identity rewrite is introduced.

Verification: F04/F05 baseline repro uses a read-only overlay. AAA regressions cover
large numbers, nested containers, decimal/exponent lexemes, typed overflow,
snapshot/dataset identities, aliases/cycles, ignored fields, UTF-8 and buffer/decode
ownership under concurrency. Canonical JSON roundtrip fuzz checks arbitrary valid
JSON tokens after canonical parsing, including number spelling.
