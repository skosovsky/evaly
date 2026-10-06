#!/usr/bin/env python3
"""Reproduce F08/F09 behavior in a disposable checkout of the original commit."""
import io
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile

root = Path(__file__).resolve().parents[1]
archive = subprocess.run(["git", "archive", "76c224a"], cwd=root, check=True, capture_output=True).stdout
with tempfile.TemporaryDirectory(prefix="evaly-optimizer-baseline-") as directory:
    baseline = Path(directory)
    with tarfile.open(fileobj=io.BytesIO(archive)) as source:
        source.extractall(baseline, filter="data")
    (baseline / "optimizer/remediation_repro_test.go").write_bytes((root / "optimizer/remediation_repro_test.go").read_bytes())
    env = dict(os.environ, EVALY_BASELINE_REPRO="1", GOCACHE="/tmp/evaly-go-build")
    subprocess.run(["go", "test", "./optimizer", "-run", "^TestF0[89]", "-count=1", "-v"], cwd=baseline, env=env, check=True)
print("F08/F09 original behavior reproduced; source checkout unchanged")
