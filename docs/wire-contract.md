# Structural wire contract

Service envelopes use `DecodeWire[T]` before semantic Restore/Validate checks. The supported roots are the sixteen published schema kinds; arbitrary host types are rejected. Domain codecs and opaque `json.RawMessage` values remain host-owned. Reflection describes explicit service envelope fields only.

JSON tags define presence: fields without `omitempty` are required even at zero. Scalar values and structs forbid null; pointers, slices and maps permit null except declared non-null collections (`CalibrationReport.Groups` and `PairSchedule.Slots`). RawMessage accepts any JSON unless an explicit field rule narrows it. Unknown struct properties, duplicate keys, trailing JSON, invalid scalar types, integer overflow, unsupported versions, declared enums and collection limits are rejected. Signed/unsigned integers have their Go storage range; integral decimal/exponent numbers are accepted and normalized for decoding. Zero is valid unless a field rule specifies a positive minimum or constant. Maps permit arbitrary keys with values constrained by their element type; extension and domain JSON stay opaque.

`internal/wirecontract` produces both runtime structural validation and generated schemas from these same definitions. JSON Schema validates shape, not checksums, identity relationships, evidence trust, assessment plans, or state transitions. Typed Restore/Validate functions perform these additional semantic checks. A typed struct passed directly to Restore has no original JSON presence information; callers reading bytes must first use DecodeWire. Size limits remain explicit at transport/store boundaries rather than a hidden global decoder limit.

HTTP request and response major versions are 2. Complete evidence delivery requires an empty reason; incomplete delivery requires a reason. These relationships, codec compatibility and target-error/output relationships are protocol semantics checked after structural decoding. Earlier HTTP versions are unsupported. No runtime validator dependency is introduced.

Current revision breaks are explicit: comparison is version 2; experiment, assessment and observation-result are version 3; search is version 4. Calibration is a version-1 kind. Previous schemas for replaced formats are removed and their major versions are unsupported. Other wire kinds retain their current versions, including scenario and HTTP version 2.

Numeric grade metrics require `unit` and `scale_revision` alongside finite bounds and direction. Comparison records an objective identity, common eligible denominator, matched case count/coverage, exclusions and availability flags for observed means. Gate quality/regression thresholds use the declared native scale; only coverage remains in [0, 1]. Paired experiment records contain the complete case/repeat schedule rather than experiment-wide order. Calibration stores nonnegative confusion counts and rates with nullable values; a rate's zero-denominator relationship, schedule integrity and compatible measurement metadata are semantic validations after structural decoding.

Search v4 records bounded rounds, received proposal lineage, measurements,
feasibility and the distinct best measured candidate and selected winner. Runtime
restore additionally validates candidate/experiment/comparison links, deterministic
ranking, bounds, stopped-state ordering and nonnegative budget/usage fields. Rejected
duplicate proposals remain audit entries; they cannot become ranking entries.

Comparison policy is a new version-1 kind. It carries an explicit assertion or numeric
objective descriptor and gate; builtin descriptor semantics are validated by
`ComparisonPolicy.Resolve` after structural decoding. Arbitrary callbacks and custom
eligibility/missingness behavior are unsupported in JSON policies; they remain
library ports. See [CLI comparison contract](cli-contract.md).
