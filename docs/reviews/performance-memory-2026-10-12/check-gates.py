"""Recompute task gates from raw Go benchmark samples, using medians."""
import json
import re
import statistics
from pathlib import Path

root = Path(__file__).resolve().parent
pattern = re.compile(r'^(Benchmark\S+)\s+\d+\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op$')

def parse(name):
    samples = {}
    for line in (root / name).read_text().splitlines():
        match = pattern.match(line)
        if match:
            samples.setdefault(match[1], []).append(tuple(map(float, match.groups()[1:])))
    return samples

baseline, candidate = parse('baseline.log'), parse('candidate.log')
assert baseline.keys() == candidate.keys(), 'benchmark matrix differs'
assert all(len(samples) >= 5 for samples in [*baseline.values(), *candidate.values()]), 'matrix has fewer than five samples'
checks = [
    ('BenchmarkAuditCapture/16384/FreshRecordAndSeal-8', {0: 1.1, 1: .8}),
    ('BenchmarkAuditCapture/16384/SaturatedRecord-8', {0: 1.1, 1: .8}),
    ('BenchmarkSavedViewValidation/bytes_16384/fresh_view-8', {1: .8}),
    ('BenchmarkSavedViewValidation/bytes_16384/rescore_8-8', {0: 1.1, 1: 1.1, 2: 1.1}),
]
results = {}
for name, limits in checks:
    assert len(baseline[name]) >= 5 and len(candidate[name]) >= 5, name
    before = [statistics.median(s[i] for s in baseline[name]) for i in range(3)]
    after = [statistics.median(s[i] for s in candidate[name]) for i in range(3)]
    ratio = [a / b for a, b in zip(after, before)]
    passed = all(ratio[i] <= limit for i, limit in limits.items())
    results[name] = dict(baseline=before, candidate=after, ratio=ratio, PASS=passed)
    assert passed, (name, results[name])
for name, samples in candidate.items():
    if name.startswith('BenchmarkArtifactIDCorpus/') or name == 'BenchmarkAuditArtifactID/Current-8':
        assert len(samples) >= 5 and all(s[1:] == (0, 0) for s in samples), name
print(json.dumps(results, indent=2))
