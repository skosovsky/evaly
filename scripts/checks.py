#!/usr/bin/env python3
"""Versioned local/CI gate. No runtime library dependencies."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
REGISTRY = ROOT / 'checks/registry.json'


class Blocked(RuntimeError):
    pass


def registry():
    return json.loads(REGISTRY.read_text())


def environment():
    return dict(os.environ, GOFLAGS='', GOENV='off', GOPROXY='https://proxy.golang.org',
                GOSUMDB='sum.golang.org', GONOSUMDB='none', GONOPROXY='none', GOPRIVATE='',
                GOWORK='off', GOTOOLCHAIN='go' + registry()['go'],
                GOPATH='/tmp/evaly-gopath', PYTHONDONTWRITEBYTECODE='1',
                GOCACHE=os.environ.get('GOCACHE', '/tmp/evaly-go-build'),
                GOMODCACHE='/tmp/evaly-gopath/pkg/mod',
                GOLANGCI_LINT_CACHE='/tmp/evaly-golangci-cache')


def run(command, cwd=ROOT, env=None):
    result = subprocess.run(command, cwd=cwd, env=env or environment(), text=True,
                            stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    if result.returncode:
        raise RuntimeError(f'{command!r} exited {result.returncode}:\n{result.stdout}')
    return result.stdout


def fingerprint(root=ROOT):
    # All tracked bytes, mode, removals and symlink targets. Untracked files are not release inputs.
    names = run(['git', 'ls-files', '-z'], root).split('\0')
    digest = hashlib.sha256()
    for name in sorted(filter(None, names)):
        path = root / name
        digest.update(name.encode() + b'\0')
        if path.is_symlink():
            digest.update(b'link:' + os.readlink(path).encode())
        elif path.is_file():
            digest.update(str(path.stat().st_mode & 0o777).encode() + b':' + path.read_bytes())
        else:
            digest.update(b'missing')
    return digest.hexdigest()


def prerequisites(root=ROOT):
    spec = registry()
    version = run(['go', 'version'], root).strip()
    if not version.startswith('go version go' + spec['go'] + ' '):
        raise Blocked('pinned Go unavailable: ' + version)
    declared = set(spec['modules'])
    actual = {str(p.parent.relative_to(root)) for p in root.rglob('go.mod')
              if not any(x in {'.git', 'vendor', '.cache'} for x in p.relative_to(root).parts)}
    if actual != declared:
        raise RuntimeError(f'module inventory drift: declared={declared}, actual={actual}')
    if spec['release_modules'] != ['.']:
        raise RuntimeError('release inventory must remain root only')
    if (root / '.golangci.yml').read_bytes() != (root / 'checks/linter-baseline.yml').read_bytes():
        raise RuntimeError('shared linter baseline drift')
    for lane in spec['lanes']:
        for arg in lane['command']:
            if 'golangci-lint@' in arg and not arg.endswith('@'+spec['linter']):
                raise RuntimeError('lane linter pin drift: '+lane['id'])
    if (root/'checks/Dockerfile').read_text().splitlines()[0] != 'FROM '+spec['linux_image']:
        raise RuntimeError('Linux image pin drift')
    if (root/'integrations/recipes/.golangci.yml').read_bytes() != (root/'checks/linter-baseline.yml').read_bytes():
        raise RuntimeError('recipes shared linter baseline drift')
    for directory in declared:
        text = (root / directory / 'go.mod').read_text()
        if '\ngo ' + spec['go'] + '\n' not in text:
            raise RuntimeError('module Go pin drift: ' + directory)
    return version


def execute(profile, *, linux_native=False, report=None, spec=None, root=ROOT):
    spec = spec or registry()
    selected = [x for x in spec['lanes'] if profile in x['profiles']]
    rows = []
    initial = fingerprint(root)
    metadata = {'source_sha': run(['git', 'rev-parse', 'HEAD'], root).strip(),
                'fingerprint': initial, 'go': spec['go'], 'linter': spec['linter'],
                'peers': spec['peers'], 'modules': spec['modules'], 'profile': profile,
                'platform': sys.platform, 'linux_native': linux_native}
    try:
        metadata['toolchain'] = prerequisites(root)
        prerequisite = 'PASS'
        detail = ''
    except (OSError, RuntimeError) as error:
        prerequisite = 'BLOCKED' if isinstance(error, (Blocked, FileNotFoundError)) else 'FAIL'
        detail = str(error)
    rows.append(dict(id='prerequisites',status=prerequisite,duration=0,command='inventory + pins + baseline',detail=detail))
    report = report or os.environ.get('EVALY_CHECK_REPORT')
    out = Path(report) if report else Path(tempfile.mkdtemp(prefix='evaly-gate-')) / 'summary.json'
    out.parent.mkdir(parents=True, exist_ok=True)
    for lane in selected:
        start = time.monotonic()
        command = lane['command']
        status, detail = 'PASS', ''
        if lane['id'] == 'linux' and linux_native:
            # This is the implementation of the Linux lane, not a skipped prerequisite.
            detail = 'Linux environment: child lanes executed in this run'
        elif prerequisite != 'PASS' or any(next((r['status'] for r in rows if r['id']==dep), 'BLOCKED') != 'PASS' for dep in lane.get('depends', [])):
            status, detail = 'BLOCKED', 'prerequisite/dependency failed'
        else:
            try:
                env=environment(); env['EVALY_GATE_PROFILE']=profile
                detail = run(command, root / lane.get('cwd', '.'),env=env)
            except (OSError, RuntimeError) as error:
                detail = str(error)
                status = 'BLOCKED' if isinstance(error, (Blocked, FileNotFoundError)) or 'BLOCKED:' in detail else 'FAIL'
        duration = round(time.monotonic() - start, 3)
        log = out.parent / (out.stem + '-' + lane['id'] + '.log')
        log.write_text(detail)
        rows.append(dict(id=lane['id'],status=status,duration=duration,command=command,
                         cwd=lane.get('cwd','.'),log=str(log),detail=detail[-3000:]))
        print(f'{lane["id"]:28} {status:8} {duration:8.3f}s {command}', flush=True)
    if fingerprint(root) != initial:
        rows.append(dict(id='immutable-input',status='FAIL',duration=0,command='tracked fingerprint',detail='inputs changed during gate'))
    if profile == 'fast':
        for lane in spec['lanes']:
            if profile not in lane['profiles']:
                rows.append(dict(id=lane['id'],status='SKIP',duration=0,command=lane['command'],detail='explicitly outside reduced fast profile'))
    success = all(row['status']=='PASS' or (profile=='fast' and row['status']=='SKIP') for row in rows)
    result = dict(metadata=metadata,results=rows,success=success,full=profile=='check')
    out.write_text(json.dumps(result,indent=2)+'\n')
    print(f'Report: {out}; {"PASS" if success else "FAIL/BLOCKED"}; profile={profile}',flush=True)
    return 0 if success else 1


def peers(kind, destination):
    resolved = {}
    for name, sha in registry()['peers'][kind].items():
        path = destination / name
        path.mkdir()
        run(['git','init','-q',str(path)])
        run(['git','-C',str(path),'fetch','-q','--depth=1',f'https://github.com/skosovsky/{name}.git',sha])
        run(['git','-C',str(path),'checkout','-q','--detach','FETCH_HEAD'])
        resolved[name] = run(['git','-C',str(path),'rev-parse','HEAD']).strip()
        if resolved[name] != sha:
            raise RuntimeError('peer SHA mismatch')
    print(json.dumps({'resolved_peer_shas':resolved}),flush=True)


def operation(name):
    with tempfile.TemporaryDirectory(prefix='evaly-lane-') as temp:
        work = Path(temp)
        if name == 'generation':
            (work/'schemas').mkdir()
            run(['go','run','./internal/schemagen',str(work / 'schemas')])
            actual = {p.name:p.read_bytes() for p in (work/'schemas').iterdir()}
            expected = {p.name:p.read_bytes() for p in (ROOT/'schemas').iterdir()}
            if actual != expected:
                raise RuntimeError('generated schema inventory/bytes differ')
        elif name.startswith('consumer-'):
            mode = name.removeprefix('consumer-')
            if mode == 'published':
                print(run(['bash','scripts/consumer_checks.sh','published']))
                return
            peers('unsupported' if mode=='unsupported' else 'supported',work)
            if mode == 'candidate':
                import artifacts
                version = os.environ.get('EVALY_CANDIDATE_VERSION','v0.0.0-ci')
                proxy = work/'proxy'
                expected = artifacts.build(ROOT,version,proxy)
                env = environment()
                env.update(GOPROXY=proxy.as_uri()+',https://proxy.golang.org',
                           GONOSUMDB='github.com/skosovsky/evaly',GONOPROXY='none',
                           CONSUMER_VERSION=version,CONSUMER_EXPECTED_SUM=expected['Sum'],
                           CONSUMER_MODCACHE=str(work/'modcache'))
                print(run(['bash','scripts/consumer_checks.sh','candidate',str(work)],env=env))
            elif mode == 'unsupported':
                result = subprocess.run(['bash','scripts/consumer_checks.sh','source',str(work)],cwd=ROOT,
                                        env=environment(),text=True,capture_output=True)
                combined=result.stdout+result.stderr
                print(combined)
                if result.returncode != 2 or registry()['unsupported_diagnostic'] not in combined:
                    raise RuntimeError('unsupported lane failed for unexpected reason: '+combined)
            else:
                print(run(['bash','scripts/consumer_checks.sh','source',str(work)]))
        elif name == 'linux':
            if shutil.which('docker') is None:
                raise Blocked('Docker unavailable')
            try:
                run(['docker','info'])
            except RuntimeError as error:
                raise Blocked('Docker daemon unavailable: '+str(error)) from error
            print(run(['docker','build','-t','evaly-check:go'+registry()['go'],'-f','checks/Dockerfile','.']))
            # Copy tracked input bytes, not neighboring trees or user go.work; mounted read-only.
            source=work/'source'; source.mkdir()
            for name in filter(None,run(['git','ls-files','-z']).split('\0')):
                path=ROOT/name
                if path.is_file() or path.is_symlink():
                    target=source/name; target.parent.mkdir(parents=True,exist_ok=True); shutil.copy2(path,target,follow_symlinks=False)
            # Include staged new gate files during development; git ls-files already covers staged input.
            sha=run(['git','rev-parse','HEAD']).strip()
            run(['git','init','-q',str(source)])
            run(['git','-C',str(source),'add','.'])
            run(['git','-C',str(source),'-c','user.name=Gate','-c','user.email=gate@example.invalid','-c','commit.gpgsign=false','commit','-qm','input '+sha])
            print(run(['docker','run','--rm','-v',str(source)+':/input:ro','-v',str(work)+':/evidence',
                       '-e','EVALY_CANDIDATE_VERSION='+os.environ.get('EVALY_CANDIDATE_VERSION','v0.0.0-ci'),
                       'evaly-check:go'+registry()['go'],'bash','-c',
                       'cp -a /input /work && cd /work && python3 scripts/checks.py '+os.environ.get('EVALY_GATE_PROFILE','check')+' --linux-native --report /evidence/linux.json']))
            evidence=json.loads((work/'linux.json').read_text())
            if not evidence['success'] or evidence['metadata']['platform']!='linux' or evidence['metadata']['fingerprint']!=fingerprint():
                raise RuntimeError('Linux child gate did not succeed')
            print(json.dumps(evidence),flush=True)
        else:
            raise RuntimeError('unknown operation '+name)


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('profile',choices=['fast','test','lint','check','plan','operation','fingerprint'])
    parser.add_argument('name',nargs='?')
    parser.add_argument('--linux-native',action='store_true')
    parser.add_argument('--report')
    args=parser.parse_args()
    if args.profile=='plan':
        print(json.dumps(registry(),indent=2)); return 0
    if args.profile=='fingerprint':
        print(fingerprint()); return 0
    if args.profile=='operation':
        operation(args.name); return 0
    if args.linux_native and sys.platform != 'linux':
        raise RuntimeError('--linux-native requires Linux')
    return execute(args.profile,linux_native=args.linux_native,report=args.report)


if __name__=='__main__':
    try:
        sys.exit(main())
    except (OSError,RuntimeError) as error:
        print(('BLOCKED: ' if isinstance(error,(Blocked,FileNotFoundError)) else 'FAIL: ')+str(error),file=sys.stderr)
        sys.exit(1)
