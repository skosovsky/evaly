# Ownership and concurrency

Configuration is not a synchronization mechanism. Set it before use and keep it
stable. A copied slice/interface/value does not clone its underlying implementation.
Callbacks cooperate with contexts; no row below promises forcibly stopping host
code, rollback or exactly-once external effects.

| Value or port | Ownership and concurrency contract |
|---|---|
| Dataset, Snapshot, SavedView and sealed private experiment | Records are copied; decode supplies fresh mutable domain values. Public zero values are unsealed. Host codecs must honor isolation/stability. |
| Public Record/Result structs | Caller-owned mutable copies. Freeze while validating, hashing or publishing; do not mutate a shared record concurrently. A checksum is not a mutex. |
| JSONCodec | Stateless stock value; independent Encode/Decode results. Host MarshalJSON/UnmarshalJSON callbacks inside domain types still need stability and safety. |
| Custom Codec | Stable Identity and representation; Encode transfers caller-owned bytes stable after return, without next-call reuse; Decode returns independent mutable values. Shared concurrent calls require host safety. |
| Lifecycle/Target/Project | Per-trial handles isolate environments, not shared adapter state. Host protects shared state and external resources. SerialShared serializes trials; it does not grant arbitrary thread safety. |
| Grader/Judge/Objective/optimizer ports | Host-owned implementations; revisions and behavior remain stable after preflight. No hidden global mutex or reflective clone of closures. Use safe ports or configured concurrency 1. |
| FieldPolicy | Allowed map/nested slices are immutable after configuration. Struct copy and revision label do not snapshot them. New settings require a separate policy/capture. |
| Capture | Record/MarkIncomplete/Seal synchronize internal state; Seal is irreversible. Policy/configuration must stay stable. Incomplete coverage cannot prove absence. |
| MemoryBudget | Method calls are synchronized and accounting is exact/process-local. Configure capacity through construction; receipts/recovery and durable external accounting belong to host. |
| FileStore | Stable MaxBytes/Fault; local immutable no-replace publication. Concurrent settings mutation, hostile swaps, network/cloud-sync filesystems and multi-host durability are not certified. |
| MemoryExport | Method state is synchronized; Dedup/Fail configured before use. Process-local reference sink, not a durable outbox. |
| MemoryLedger | Synchronized process-local holdout claims; no durable distributed truth or automatic reconciliation. |
| HTTP NewHandler/Target | Shared codecs/invoke/client must support concurrent requests. Constructor snapshots identities; host must keep behavior stable. Auth/retries/external idempotency are host policy. |
| Observation worker | Bounded queue/concurrency, fresh grading values, shared grader/clock/budget implementations. Deadline starts at enqueue; OutcomeDelay is metadata. Stop producers before final Flush; Cancel closes admission. |

The SavedView benchmark and the retained per-grader observation validation
decision are recorded in [local decisions](local-remediation-decisions.md).
Fresh decoding is intentional: mutable decoded values are not cached/shared.
