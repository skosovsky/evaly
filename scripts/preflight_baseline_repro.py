"""Read-only Go overlays reproduce F06/F07 against the reviewed baseline."""
import json
import os
import pathlib
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='evaly-preflight-baseline-') as directory:
    base = pathlib.Path(directory)
    replacements = {}
    for name in ('dataset.go', 'scenario.go', 'grader.go'):
        old = base / name
        old.write_bytes(subprocess.check_output(['git', 'show', '76c224a:' + name], cwd=root))
        replacements[str(root / name)] = str(old)
    overlay = base / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': replacements}))
    subprocess.run(['go', 'run', '-overlay=' + str(overlay), 'testdata/repro/preflight.go'],
                   cwd=root, env=dict(os.environ, GOCACHE='/tmp/evaly-go-build'), check=True)
