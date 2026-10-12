# Frozen acceptance requirements

Baseline: ca98aafa445eb6ab82f1825941aa4b2cb82ee63d. Denominator: **25**. A requirement counts only as PASS in full. No deferred gates or partial credit. Reviewers inspect the same final candidate commit and tracked evidence tree. No release or consumer/dependency changes are authorized.

| # | Mandatory requirement |
|---|---|
| 1 | Scope evaly only; public signatures, wire schemas and dependency pins unchanged |
| 2 | Artifact ID language equivalent on boundary corpus and executed fuzz |
| 3 | Artifact ID valid and representative invalid cases 0 B/op and 0 allocs/op |
| 4 | Permanent reproducible complete benchmark matrix in ordinary library tests |
| 5 | Baseline/candidate >=5 samples, identical toolchain/CPU/GOMAXPROCS/fixtures, serialized Go commands |
| 6 | Raw baseline/candidate logs, benchstat and candidate SHA recorded |
| 7 | FreshRecordAndSeal16KiB median B/op >=20% reduction and ns/op <=110% baseline |
| 8 | SaturatedRecord16KiB median B/op >=20% reduction and ns/op <=110% baseline |
| 9 | SavedView.View16KiB median B/op >=20% reduction |
| 10 | Rescore_8 median B/op, allocs/op, ns/op each <=110% baseline |
| 11 | Profiles captured and examined for selected JSON optimizations |
| 12 | Original validation/normalization maintained for specialized copies; no arbitrary BYOT copies |
| 13 | Canonical bytes, number lexemes, duplicate-key/UTF-8/trailing/number checks preserved |
| 14 | Revision/checksum semantics unchanged and external restore fully validated |
| 15 | Policy callbacks order/count and error precedence characterized before optimization and preserved |
| 16 | Caller event inputs isolated from mutating policies, including raw bytes and slices |
| 17 | Policy outputs and returned records isolated from private retained state |
| 18 | Nested maps/slices/raw bytes and nil/empty distinctions covered |
| 19 | Privacy projection before retention; snapshots contain no forbidden raw data |
| 20 | MaxEvents/MaxBytes and diagnostic caps preserved |
| 21 | Fresh mutable domain views for each grader; host codec checks/callback volume preserved |
| 22 | Scheduler concurrency, SerialShared, cleanup/cancellation/budget/retained slots unchanged and tested |
| 23 | New tests AAA; no expectations retaining an old defect or shifted invocation costs |
| 24 | Root go test ./... and go test -race ./... PASS; contracttest/nested modules and repository CI checks PASS |
| 25 | A 100% APPROVE and independent B APPROVE with zero unfixed findings on same final candidate SHA |

Item 25 is a closing condition established by the two verdicts; A reports 24/24 substantive requirements plus its review and B's final verdict to close 25/25. Any substantive failure prevents approval.
