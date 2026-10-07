#!/usr/bin/env python3
"""Exact public proxy/checksum/consumer/CI proof, with bounded eventual-consistency retries."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

from checks import ROOT, environment, registry


def invoke(command,env):
    result=subprocess.run(command,cwd=ROOT,env=env,text=True,capture_output=True,timeout=600 if command[0]=="bash" else 90)
    if result.returncode:
        raise RuntimeError(f'{command}: {result.stdout}{result.stderr}')
    return result.stdout


def validate_download(record,manifest,sha):
    for key in ('Path','Version','Sum','GoModSum'):
        if record.get(key)!=manifest[key]:
            raise RuntimeError('public artifact differs: '+key)
    origin=record.get('Origin',{})
    if origin.get('Hash') and origin['Hash']!=sha:
        raise RuntimeError('public artifact origin differs')

def verify(version,sha,manifest_path,output):
    manifest=json.loads(Path(manifest_path).read_text())
    if manifest['Version']!=version:
        raise RuntimeError('manifest version mismatch')
    with tempfile.TemporaryDirectory(prefix='evaly-public-') as temp:
        env=environment()
        env.update(GOPROXY='https://proxy.golang.org',GOSUMDB='sum.golang.org',GONOSUMDB='none',
                   GONOPROXY='none',GOPRIVATE='',GOMODCACHE=temp+'/modcache',CONSUMER_MODCACHE=temp+'/modcache',
                   CONSUMER_VERSION=version,CONSUMER_EXPECTED_SUM=manifest['Sum'])
        error=None
        for attempt in range(12):
            try:
                record=json.loads(invoke(['go','mod','download','-json',manifest['Path']+'@'+version],env))
                if record.get('Error'):
                    raise RuntimeError(record['Error'])
                validate_download(record,manifest,sha)
                error=None; break
            except (RuntimeError, json.JSONDecodeError, subprocess.TimeoutExpired) as caught:
                error=caught
                if attempt<11: time.sleep(10)
        if error: raise error
        print(json.dumps(record,indent=2),flush=True)
        smoke=invoke(['bash','scripts/consumer_checks.sh','public'],env)
        print(smoke,flush=True)
        # Only the root v* tag workflow for this SHA satisfies release verification.
        for attempt in range(180):
            runs=json.loads(invoke(['gh','api',f'repos/skosovsky/evaly/actions/runs?head_sha={sha}&event=push&per_page=100'],env))['workflow_runs']
            relevant=[r for r in runs if r['path']=='.github/workflows/go.yml' and r['head_branch']==version and r['head_sha']==sha]
            if relevant:
                newest=max(relevant,key=lambda r:r['id'])
                if newest['status']=='completed':
                    if newest['conclusion']!='success':
                        raise RuntimeError('release CI failed: '+newest['html_url'])
                    jobs=json.loads(invoke(['gh','api',f'repos/skosovsky/evaly/actions/runs/{newest["id"]}/jobs'],env))['jobs']
                    if not jobs or any(j['conclusion']!='success' for j in jobs):
                        raise RuntimeError('required release jobs incomplete/failed')
                    evidence=dict(version=version,sha=sha,module=record,ci=newest['html_url'],jobs=jobs,consumer='PASS',pins=registry())
                    Path(output).write_text(json.dumps(evidence,indent=2)+'\n')
                    print('Verified public release: '+newest['html_url'],flush=True)
                    return
            if attempt<179: time.sleep(10)
        raise RuntimeError('BLOCKED: exact-tag CI did not complete within 30 minutes')


def main():
    parser=argparse.ArgumentParser(); parser.add_argument('version'); parser.add_argument('sha'); parser.add_argument('manifest'); parser.add_argument('output')
    args=parser.parse_args(); verify(args.version,args.sha,args.manifest,args.output)


if __name__=='__main__': main()
