"""Reproduce F04/F05 against a disposable archive of the reviewed baseline."""
import io
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile

root = Path(__file__).resolve().parents[1]
archive = subprocess.run(["git", "archive", "76c224a"], cwd=root, check=True, capture_output=True).stdout
with tempfile.TemporaryDirectory(prefix="evaly-codec-baseline-") as directory:
    baseline = Path(directory)
    with tarfile.open(fileobj=io.BytesIO(archive)) as source:
        source.extractall(baseline, filter="data")
    fixture = baseline / "testdata/repro/codec.go"
    fixture.parent.mkdir(parents=True, exist_ok=True)
    fixture.write_bytes((root / "testdata/repro/codec.go").read_bytes())
    subprocess.run(["go", "run", "testdata/repro/codec.go"], cwd=baseline,
                   env=dict(os.environ, GOCACHE="/tmp/evaly-go-build"), check=True)
print("F04/F05 baseline behavior reproduced; source checkout unchanged")
