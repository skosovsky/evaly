"""Read-only Go overlay for F04/F05 baseline behavioral repros."""
import json
import os
import pathlib
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='evaly-codec-baseline-') as directory:
    base = pathlib.Path(directory)
    old = base / 'codec.go'
    old.write_bytes(subprocess.check_output(
        ['git', 'show', '76c224a:codec.go'], cwd=root))
    overlay = base / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': {str(root / 'codec.go'): str(old)}}))
    subprocess.run(['go', 'run', '-overlay=' + str(overlay), 'testdata/repro/codec.go'],
                   cwd=root, env=dict(os.environ, GOCACHE='/tmp/evaly-go-build'), check=True)
