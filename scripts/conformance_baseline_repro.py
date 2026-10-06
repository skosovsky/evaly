#!/usr/bin/env python3
"""Reproduce F10 and D51 on the original revision without source worktree effects."""
import io
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile

root = Path(__file__).resolve().parents[1]
archive = subprocess.run(["git", "archive", "76c224a"], cwd=root, check=True, capture_output=True).stdout
with tempfile.TemporaryDirectory(prefix="evaly-conformance-baseline-") as directory:
    baseline = Path(directory)
    with tarfile.open(fileobj=io.BytesIO(archive)) as source:
        source.extractall(baseline, filter="data")
    (baseline / "conformance/remediation_baseline_test.go").write_bytes((root / "testdata/repro/conformance_test.go").read_bytes())
    subprocess.run(["go", "test", "./conformance", "-run", "^TestBaseline", "-count=1", "-v"], cwd=baseline, env=dict(os.environ, GOCACHE="/tmp/evaly-go-build"), check=True)
print("F10 leak and D51 no-op Claim acceptance reproduced on 76c224a")
