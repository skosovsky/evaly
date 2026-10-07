"""Local-only AAA release fixtures. python3 scripts/release_test.py [--baseline]."""
import os
import pathlib
import subprocess
import sys
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
BASELINE = '--baseline' in sys.argv
if BASELINE:
    sys.argv.remove('--baseline')


def run(args, cwd, *, data=None, env=None, check=True):
    result = subprocess.run(args, cwd=cwd, input=data, text=True,
                            capture_output=True, env=env)
    if check and result.returncode:
        raise AssertionError(f'{args}: {result.stdout}{result.stderr}')
    return result


class ReleaseFixture(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='evaly-release-test-')
        self.addCleanup(self.tmp.cleanup)
        self.base = pathlib.Path(self.tmp.name)
        self.repo = self.base / 'source repo'
        self.repo.mkdir()
        self.remote = self.base / 'remote.git'
        self.env = dict(os.environ, GIT_CONFIG_NOSYSTEM='1',
                        GIT_CONFIG_GLOBAL=os.devnull, TMPDIR=str(self.base))
        self.git('init', '-b', 'main')
        self.git('config', 'user.name', 'Fixture')
        self.git('config', 'user.email', 'fixture@example.invalid')
        self.git('config', 'commit.gpgsign', 'false')
        self.write('go.mod', 'module example.invalid/fixture\n\ngo 1.27.1\n')
        self.write('contracttest/go.mod', 'module example.invalid/test\n\ngo 1.27.1\n')
        if BASELINE:
            source = run(['git', 'show', '76c224a:scripts/release.sh'], ROOT).stdout
        else:
            source = (ROOT / 'scripts/release.sh').read_text()
        self.write('release.sh', source)
        if not BASELINE:
            for name in ('checks.py','artifacts.py'):
                self.write('scripts/'+name,(ROOT/'scripts'/name).read_text())
            self.write('checks/modulezip.go.txt',(ROOT/'checks/modulezip.go.txt').read_text())
            self.write('checks/registry.json',(ROOT/'checks/registry.json').read_text())
            # The fixture supplies a synthetic gate, never a production bypass.
            self.write('Makefile', 'check:\n\t@python3 fixture_gate.py\n')
            self.write('fixture_gate.py', 'import os,pathlib\n'
                       'if os.environ.get("FIXTURE_GATE_FAIL"): raise SystemExit(17)\n'
                       'if os.environ.get("FIXTURE_SOURCE_MUTATE"): pathlib.Path(os.environ["FIXTURE_SOURCE_MUTATE"]).write_text("module changed.invalid/source\\n\\ngo 1.27.1\\n")\n'
                       'if os.environ.get("FIXTURE_IGNORED"): pathlib.Path("ignored.go").write_text("package fixture\\n")\n'
                       'if os.environ.get("FIXTURE_MUTATE"): pathlib.Path("go.mod").write_text("module changed.invalid/fixture\\n\\ngo 1.27.1\\n")\n')
            self.write('scripts/verify_release.py','import os\nraise SystemExit(18 if os.environ.get("FIXTURE_PUBLIC_FAIL") else 0)\n')
        self.git('add', '.')
        self.git('commit', '-m', 'fixture')
        run(['git', 'init', '--bare', str(self.remote)], self.base, env=self.env)
        # Exercise relative remote resolution from disposable checkout.
        self.git('remote', 'add', 'origin', '../remote.git')
        self.git('push', '-u', 'origin', 'main')

    def git(self, *args, check=True):
        return run(['git', *args], self.repo, env=self.env, check=check)

    def write(self, name, text):
        path = self.repo / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text)
        return path

    def state(self):
        return tuple(self.git(*args).stdout for args in [
            ('rev-parse', 'HEAD'), ('symbolic-ref', '-q', 'HEAD'),
            ('status', '--porcelain=v1', '--untracked-files=all'),
            ('show-ref',), ('ls-files', '--stage')])

    def release(self, kind='patch', modules='.'):
        return run(['bash', 'release.sh', kind, modules], self.repo,
                   data='y\n', env=self.env, check=False)

    def tags(self):
        return run(['git', '--git-dir', str(self.remote), 'tag', '-l'],
                   self.base, env=self.env).stdout.splitlines()

    def clean_temps(self):
        self.assertEqual(list(self.base.glob('evaly-release.????????')), [])

    def require_current(self):
        if BASELINE:
            self.skipTest('new contract only; separate tests reproduce baseline defects')

    def test_unrelated_files_and_tags(self):
        # Arrange
        self.git('tag', 'scratch-local')
        self.git('tag', 'v9.0.0')  # unpublished version must not affect next version
        self.write('private-untracked.txt', 'harmless synthetic marker\n')
        before = self.state()
        # Act
        result = self.release()
        # Assert
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        if BASELINE:
            self.assertIn('scratch-local', self.tags())
            exposed = run(['git', '--git-dir', str(self.remote), 'show',
                           'v9.0.1:private-untracked.txt'], self.base, env=self.env)
            self.assertIn('synthetic marker', exposed.stdout)
            return
        self.assertEqual(self.tags(), ['v0.0.1'])
        self.assertEqual(self.state(), before)
        exposed = run(['git', '--git-dir', str(self.remote), 'show',
                       'v0.0.1:private-untracked.txt'], self.base, env=self.env, check=False)
        self.assertNotEqual(exposed.returncode, 0)
        self.clean_temps()

    def test_rejected_push_and_retry(self):
        # Arrange
        hook = self.remote / 'hooks/pre-receive'
        hook.write_text('#!/bin/sh\nexit 1\n')
        hook.chmod(0o755)
        before = self.state()
        # Act
        failed = self.release()
        # Assert
        self.assertNotEqual(failed.returncode, 0)
        self.assertEqual(self.tags(), [])
        if BASELINE:
            self.assertEqual(self.git('branch', '--show-current').stdout.strip(), '')
            self.assertEqual(self.git('tag', '-l').stdout.strip(), 'v0.0.1')
            return
        self.assertEqual(self.state(), before)
        self.assertIn('Remote ref is absent', failed.stderr)
        self.clean_temps()
        hook.unlink()
        retried = self.release()
        self.assertEqual(retried.returncode, 0, retried.stderr)
        self.assertEqual(self.tags(), ['v0.0.1'])
        self.assertEqual(self.state(), before)
        self.clean_temps()

    def test_clean_happy_path_and_break(self):
        self.require_current()
        # Arrange
        before = self.state()
        # Act
        patch = self.release()
        breaking = self.release('break')
        # Assert
        self.assertEqual(patch.returncode, 0, patch.stderr)
        self.assertEqual(breaking.returncode, 0, breaking.stderr)
        self.assertEqual(self.tags(), ['v0.0.1', 'v0.1.0'])
        self.assertEqual(self.state(), before)
        self.clean_temps()

    def test_detached_start_and_submodule_rejected(self):
        self.require_current()
        # Arrange
        self.git('checkout', '--detach', 'HEAD')
        head = self.git('rev-parse', 'HEAD').stdout
        # Act
        detached = self.release()
        submodule = self.release(modules='. contracttest')
        # Assert
        self.assertNotEqual(detached.returncode, 0)
        self.assertIn('detached source HEAD is unsupported', detached.stderr)
        self.assertNotEqual(submodule.returncode, 0)
        self.assertEqual(self.git('rev-parse', 'HEAD').stdout, head)
        self.assertEqual(self.tags(), [])
        self.clean_temps()

    def test_tracked_changes_rejected(self):
        self.require_current()
        # Arrange
        self.write('go.mod', 'module example.invalid/changed\n\ngo 1.27.1\n')
        self.git('add', 'go.mod')
        before = self.state()
        # Act
        result = self.release()
        # Assert
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.state(), before)
        self.assertEqual(self.tags(), [])
        self.clean_temps()

    def test_preparation_and_tag_failure_cleanup(self):
        self.require_current()
        for phase in ('go', 'tag'):
            with self.subTest(phase=phase):
                # Arrange: wrapper fails only the operation in the private checkout.
                wrappers = self.base / ('bin-' + phase)
                wrappers.mkdir()
                real_git = run(['which', 'git'], self.base).stdout.strip()
                target = wrappers / ('go' if phase == 'go' else 'git')
                if phase == 'go':
                    target.write_text('#!/bin/sh\nexit 42\n')
                else:
                    target.write_text('#!/bin/sh\nif [ "$1" = tag ]; then exit 42; fi\n'
                                      f'exec "{real_git}" "$@"\n')
                target.chmod(0o755)
                before = self.state()
                old_path = self.env['PATH']
                self.env['PATH'] = str(wrappers) + os.pathsep + old_path
                # Act
                result = self.release()
                self.env['PATH'] = old_path
                # Assert
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.state(), before)
                self.assertEqual(self.tags(), [])
                self.clean_temps()

    def test_unknown_push_status_retains_recovery(self):
        self.require_current()
        # Arrange: emulate unavailable remote only after push starts.
        wrappers = self.base / 'bin-unknown'
        wrappers.mkdir()
        real_git = run(['which', 'git'], self.base).stdout.strip()
        marker = self.base / 'push-started'
        wrapper = wrappers / 'git'
        wrapper.write_text('#!/bin/sh\n'
                           f'if [ "$1" = push ]; then touch "{marker}"; exit 43; fi\n'
                           f'if [ "$1" = ls-remote ] && [ -f "{marker}" ]; then exit 44; fi\n'
                           f'exec "{real_git}" "$@"\n')
        wrapper.chmod(0o755)
        before = self.state()
        self.env['PATH'] = str(wrappers) + os.pathsep + self.env['PATH']
        # Act
        result = self.release()
        # Assert
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Remote status unknown', result.stderr)
        retained = list(self.base.glob('evaly-release.????????'))
        self.assertEqual(len(retained), 1)
        self.assertEqual(run(['git', 'tag', '-l'], retained[0], env=self.env).stdout.strip(), 'v0.0.1')
        self.assertEqual(self.state(), before)
        self.assertEqual(self.tags(), [])

    def test_generated_commit_allowlist_and_exact_atomic_push(self):
        self.require_current()
        # Arrange: formatting must produce a private commit; source stays intact.
        self.write('go.mod', 'module   example.invalid/fixture\n\ngo 1.27.1\n')
        self.git('add', 'go.mod')
        self.git('commit', '-m', 'unformatted fixture')
        self.write('private.txt', 'synthetic untracked marker')
        before = self.state()
        wrappers = self.base / 'bin-spy'
        wrappers.mkdir()
        real_git = run(['which', 'git'], self.base).stdout.strip()
        capture = self.base / 'push-arguments'
        wrapper = wrappers / 'git'
        wrapper.write_text('#!/bin/sh\n'
                           f'if [ "$1" = push ]; then printf "%s\\n" "$@" > "{capture}"; fi\n'
                           f'exec "{real_git}" "$@"\n')
        wrapper.chmod(0o755)
        self.env['PATH'] = str(wrappers) + os.pathsep + self.env['PATH']
        # Act
        result = self.release()
        # Assert
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.state(), before)
        args = capture.read_text().splitlines()
        self.assertEqual(args[:2], ['push', '--atomic'])
        self.assertEqual(args[-1], 'refs/tags/v0.0.1:refs/tags/v0.0.1')
        self.assertEqual(len(args), 4)
        changed = run(['git', '--git-dir', str(self.remote), 'diff-tree',
                       '--no-commit-id', '--name-only', '-r', 'v0.0.1'],
                      self.base, env=self.env).stdout.splitlines()
        self.assertEqual(changed, ['go.mod'])
        self.clean_temps()

    def test_push_failure_after_remote_acceptance(self):
        self.require_current()
        # Arrange: real local push succeeds but wrapper reports transport failure.
        wrappers = self.base / 'bin-accepted'
        wrappers.mkdir()
        real_git = run(['which', 'git'], self.base).stdout.strip()
        wrapper = wrappers / 'git'
        wrapper.write_text('#!/bin/sh\n'
                           f'if [ "$1" = push ]; then "{real_git}" "$@" || exit; exit 43; fi\n'
                           f'exec "{real_git}" "$@"\n')
        wrapper.chmod(0o755)
        before = self.state()
        self.env['PATH'] = str(wrappers) + os.pathsep + self.env['PATH']
        # Act
        result = self.release()
        # Assert
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Observed remote ref:', result.stderr)
        self.assertEqual(self.tags(), ['v0.0.1'])
        self.assertEqual(self.state(), before)
        self.assertEqual(len(list(self.base.glob('evaly-release.????????'))), 1)

    def test_required_gate_failure_forbids_ref_mutations(self):
        self.require_current()
        # Arrange
        self.env['FIXTURE_GATE_FAIL']='1'
        before=self.state()
        # Act
        result=self.release()
        # Assert
        self.assertNotEqual(result.returncode,0)
        self.assertIn('required candidate gate failed',result.stderr)
        self.assertEqual(self.tags(),[])
        self.assertEqual(self.state(),before)
        self.clean_temps()

    def test_mutation_after_gate_forbids_publication(self):
        self.require_current()
        # Arrange
        self.env['FIXTURE_MUTATE']='1'
        before=self.state()
        # Act
        result=self.release()
        # Assert
        self.assertNotEqual(result.returncode,0)
        self.assertIn('candidate changed after check',result.stderr)
        self.assertEqual(self.tags(),[])
        self.assertEqual(self.state(),before)
        self.clean_temps()

    def test_source_mutation_during_gate_forbids_publication(self):
        self.require_current()
        # Arrange: another writer changes caller source during the isolated gate.
        self.env['FIXTURE_SOURCE_MUTATE']=str(self.repo/'go.mod')
        before_head=self.git('rev-parse','HEAD').stdout
        # Act
        result=self.release()
        # Assert: the writer's edit is preserved, but no release refs may be published.
        self.assertNotEqual(result.returncode,0)
        self.assertIn('committed source changed',result.stderr)
        self.assertEqual(self.tags(),[])
        self.assertEqual(self.git('rev-parse','HEAD').stdout,before_head)
        self.assertIn('changed.invalid/source',(self.repo/'go.mod').read_text())

    def test_ignored_input_after_gate_forbids_publication(self):
        self.require_current()
        # Arrange
        self.write('.gitignore','ignored.go\n')
        self.git('add','.gitignore'); self.git('commit','-m','ignored fixture')
        self.env['FIXTURE_IGNORED']='1'
        before=self.state()
        # Act
        result=self.release()
        # Assert
        self.assertNotEqual(result.returncode,0)
        self.assertIn('candidate changed after check',result.stderr)
        self.assertEqual(self.tags(),[])
        self.assertEqual(self.state(),before)

    def test_postpublication_failure_is_not_success(self):
        self.require_current()
        # Arrange
        self.env['FIXTURE_PUBLIC_FAIL']='1'
        before=self.state()
        # Act
        result=self.release()
        # Assert
        self.assertNotEqual(result.returncode,0)
        self.assertIn('published but verification failed',result.stderr)
        self.assertEqual(self.tags(),['v0.0.1'])
        self.assertEqual(self.state(),before)
        self.assertEqual(len(list(self.base.glob('evaly-release.????????'))),1)

    def test_global_tag_signing_cannot_change_ref_type(self):
        self.require_current()
        # Arrange: global automatic signing must not change the lightweight policy.
        global_config = self.base / 'global-config'
        global_config.write_text('[tag]\n\tgpgSign = true\n')
        self.env['GIT_CONFIG_GLOBAL'] = str(global_config)
        before = self.state()
        expected = self.git('rev-parse', 'HEAD').stdout.strip()
        # Act
        result = self.release()
        # Assert
        self.assertEqual(result.returncode, 0, result.stderr)
        actual = run(['git', '--git-dir', str(self.remote), 'rev-parse',
                      'refs/tags/v0.0.1'], self.base, env=self.env).stdout.strip()
        kind = run(['git', '--git-dir', str(self.remote), 'cat-file', '-t', actual],
                   self.base, env=self.env).stdout.strip()
        self.assertEqual(kind, 'commit')
        self.assertEqual(actual, expected)
        self.assertIn(f'Published refs/tags/v0.0.1 at {actual}', result.stdout)
        self.assertEqual(self.state(), before)
        self.clean_temps()


if __name__ == '__main__':
    unittest.main(verbosity=2)
