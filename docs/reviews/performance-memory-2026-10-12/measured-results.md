# Final measured results

Go 1.27.1, Apple M1 Max, GOMAXPROCS=8. Complete matrix: 25 variants, seven 1-second samples for each side. Median metrics in raw logs; check-gates.py uses unchanged thresholds.

| Operation (16KiB) | ns/op baseline → candidate | B/op baseline → candidate | allocs/op baseline → candidate | B/op reduction | Gate |
|---|---:|---:|---:|---:|---|
| FreshRecordAndSeal | 470,826 → 328,005 | 703,244 → 525,173 | 331 → 246 | 25.3% | PASS |
| SaturatedRecord | 397,484 → 224,723 | 503,042 → 381,137 | 179 → 107 | 24.2% | PASS |
| SavedView.View | 328,178 → 98,952 | 453,356 → 201,999 | 80 → 35 | 55.4% | PASS |
| Rescore_8 | 3,908,018 → 2,201,152 | 5,839,502 → 3,598,030 | 2,202 → 1,800 | 38.4% | PASS |

Artifact ID: valid and representative invalid IDs across six corpus cases each 0 B/op and 0 allocs/op, seven samples. Corpus/fuzz decisions match the original compiled regexp. Rescore_8 improves median ns/op by 43.7%, B/op by 38.4%, and allocs/op by 18.3%; no permitted regression budget is consumed.

PASS: root tests, race tests in all three modules, lint in all three modules, integration and e2e CI targets, 30-second fuzz runs for artifact ID and Event-copy equivalence, depth boundary regression tests on baseline and candidate. Final ID fuzz: 3,280,456 executions; final capture-copy fuzz: 694,576 executions (prior cached corpus also replayed).

The failed short pair remains in short-*.log/short-benchstat.txt; the final complete pair uses longer identical measurement parameters, without changing gates. The fixed-depth failure evidence is retained in depth-*-probe.log. Independent A/B acceptance is recorded separately against the committed candidate SHA.
