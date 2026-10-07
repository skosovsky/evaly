#!/usr/bin/env python3
"""Root module artifact and checksum evidence from exact tracked bytes."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile
import shutil


def command(args,cwd,env=None):
    result=subprocess.run(args,cwd=cwd,env=env,text=True,capture_output=True)
    if result.returncode:
        raise RuntimeError(f'{args}: {result.stdout}{result.stderr}')
    return result.stdout


def build(root,version,proxy):
    root=Path(root); proxy=Path(proxy)
    module=json.loads(command(['go','mod','edit','-json'],root))['Module']['Path']
    names=command(['git','ls-files','-z'],root).split('\0')
    destination=proxy/module/'@v'; destination.mkdir(parents=True)
    (destination/(version+'.mod')).write_bytes((root/'go.mod').read_bytes())
    (destination/(version+'.info')).write_text(json.dumps({'Version':version,'Time':'2000-01-01T00:00:00Z'}))
    (destination/'list').write_text(version+'\n')
    with tempfile.TemporaryDirectory(prefix='evaly-zip-') as temp:
        tools=Path(temp); source=tools/'source'; source.mkdir()
        for name in filter(None,names):
            path=root/name
            if path.is_file() and not path.is_symlink():
                target=source/name; target.parent.mkdir(parents=True,exist_ok=True); shutil.copy2(path,target)
        pins=json.loads((root/'checks/registry.json').read_text())
        (tools/'go.mod').write_text('module example.invalid/modulezip\n\ngo '+pins['go']+'\n\nrequire '+pins['module_zip_tool']+'\n')
        (tools/'main.go').write_bytes((root/'checks/modulezip.go.txt').read_bytes())
        env=dict(os.environ,GOFLAGS='',GOENV='off',GOWORK='off',GOTOOLCHAIN='go'+pins['go'],GOPATH='/tmp/evaly-gopath',GOMODCACHE='/tmp/evaly-gopath/pkg/mod',GOCACHE='/tmp/evaly-go-build')
        command(['go','run','-mod=mod','.',module,version,str(source),str(destination/(version+'.zip'))],tools,env)
    with tempfile.TemporaryDirectory(prefix='evaly-artifact-') as temp:
        env=dict(os.environ,GOFLAGS='',GOENV='off',GOWORK='off',GOTOOLCHAIN='go'+json.loads((root/'checks/registry.json').read_text())['go'],GOPROXY=proxy.as_uri(),
                 GOSUMDB='off',GONOPROXY='none',GOMODCACHE=temp+'/modcache')
        evidence=json.loads(command(['go','mod','download','-json',module+'@'+version],root,env))
        command(['go','mod','verify'],root,env)
    manifest={k:evidence[k] for k in ('Path','Version','Sum','GoModSum')}
    (proxy/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
    return manifest


def main():
    parser=argparse.ArgumentParser(); parser.add_argument('version'); parser.add_argument('proxy')
    args=parser.parse_args(); print(json.dumps(build(Path.cwd(),args.version,Path(args.proxy).resolve()),indent=2))


if __name__=='__main__': main()
