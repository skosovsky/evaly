"""Read-only Go overlay for the F03 baseline behavioral repro."""
import json
import os
import pathlib
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='evaly-budget-baseline-') as directory:
    base = pathlib.Path(directory)
    old = base / 'budget.go'
    old.write_bytes(subprocess.check_output(
        ['git', 'show', '76c224a:budget.go'], cwd=root))
    overlay = base / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': {str(root / 'budget.go'): str(old)}}))
    subprocess.run(['go', 'run', '-overlay=' + str(overlay), 'testdata/repro/budget.go'],
                   cwd=root, env=dict(os.environ, GOCACHE='/tmp/evaly-go-build'), check=True)
