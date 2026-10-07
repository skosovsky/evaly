"""AAA negative fixtures for gate aggregation, missing infrastructure and parity."""
import contextlib
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import checks


class GateTests(unittest.TestCase):
    def test_independent_failures_and_blocked_dependency(self):
        # Arrange: local-only registry fixture executes actual failing subprocesses.
        spec=checks.registry()
        spec['lanes']=[dict(id=name,profiles=['test'],command=['python3','-c','raise SystemExit(19)'])
                       for name in ('ordinary-test','linter','script-test','consumer')]
        spec['lanes'].append(dict(id='dependent',profiles=['test'],command=['python3','-c','raise SystemExit(0)'],depends=['consumer']))
        with tempfile.TemporaryDirectory() as temp, mock.patch.object(checks,'prerequisites',return_value='fixture toolchain'):
            report=Path(temp)/'report.json'
            # Act
            with contextlib.redirect_stdout(io.StringIO()):
                status=checks.execute('test',spec=spec,report=report)
            # Assert
            result=json.loads(report.read_text())
            self.assertNotEqual(status,0)
            self.assertFalse(result['success'])
            self.assertEqual([r['status'] for r in result['results'][1:]],['FAIL']*4+['BLOCKED'])

    def test_missing_prerequisite_cannot_pass(self):
        for missing in ('Docker','peer','toolchain'):
            # Arrange
            with self.subTest(missing=missing), tempfile.TemporaryDirectory() as temp:
                report=Path(temp)/'summary.json'
                spec=checks.registry(); spec['lanes']=[dict(id='required',profiles=['check'],command=['python3','-c','raise SystemExit(0)'])]
                with mock.patch.object(checks,'prerequisites',side_effect=checks.Blocked(missing)), contextlib.redirect_stdout(io.StringIO()):
                    # Act
                    status=checks.execute('check',spec=spec,report=report)
                # Assert
                self.assertNotEqual(status,0)
                self.assertEqual([r['status'] for r in json.loads(report.read_text())['results']],['BLOCKED','BLOCKED'])

    def test_real_lane_failures_aggregate(self):
        # Arrange: actual Go test, pinned linter, Python test and consumer entrypoint.
        with tempfile.TemporaryDirectory(prefix='evaly-negative-') as temp:
            work=Path(temp)
            (work/'go.mod').write_text('module example.invalid/negative\n\ngo 1.27.1\n')
            (work/'negative_test.go').write_text('package negative\nimport "testing"\nfunc TestMustRun(t *testing.T) { t.Fatal("ordinary failure") }\n')
            (work/'bad.yml').write_text('version: invalid\n')
            (work/'test_failure.py').write_text('import unittest\nclass Failure(unittest.TestCase):\n def test_failure(self): self.fail("script failure")\n')
            spec=checks.registry()
            spec['lanes']=[
                dict(id='ordinary-test',profiles=['test'],command=['go','-C',temp,'test','-race','-count=1','./...']),
                dict(id='linter',profiles=['test'],command=['go','run','github.com/golangci/golangci-lint/v2/cmd/golangci-lint@'+spec['linter'],'config','verify','--config',temp+'/bad.yml']),
                dict(id='script-test',profiles=['test'],command=['python3','-m','unittest','discover','-s',temp,'-v']),
                dict(id='consumer',profiles=['test'],command=['bash','scripts/consumer_checks.sh','source',temp+'/missing-peer']),
            ]
            # Act: inherited filter would hide the failing Go test if not sanitized.
            with mock.patch.dict(os.environ,GOFLAGS='-run=__no_tests__'), contextlib.redirect_stdout(io.StringIO()):
                status=checks.execute('test',spec=spec,report=work/'summary.json')
            # Assert
            result=json.loads((work/'summary.json').read_text())
            self.assertNotEqual(status,0)
            self.assertEqual(result['results'][0]['status'],'PASS')
            self.assertEqual([r['status'] for r in result['results'][1:]],['FAIL']*4)
            self.assertIn('ordinary failure',result['results'][1]['detail'])

    def test_production_missing_docker_and_wrong_toolchain(self):
        # Arrange
        with mock.patch.object(checks.shutil,'which',return_value=None):
            # Act / Assert
            with self.assertRaises(checks.Blocked): checks.operation('linux')
        with mock.patch.object(checks,'run',return_value='go version go1.26.0 linux/amd64'):
            with self.assertRaises(checks.Blocked): checks.prerequisites()

    def test_unsupported_infrastructure_failure_cannot_pass(self):
        # Arrange: production negative lane must reject unrelated failures and exit2.
        for code,diagnostic in [(1,'network unavailable'),(2,'UNSUPPORTED dependency API: wrong symbol')]:
            response=mock.Mock(returncode=code,stdout='',stderr=diagnostic)
            with mock.patch.object(checks,'peers'), mock.patch.object(checks.subprocess,'run',return_value=response), contextlib.redirect_stdout(io.StringIO()):
                # Act / Assert
                with self.assertRaisesRegex(RuntimeError,'unexpected reason'): checks.operation('consumer-unsupported')

    def test_user_flags_cannot_skip_tests(self):
        # Arrange
        with mock.patch.dict(os.environ,GOFLAGS='-run=__no_tests__',GOENV='/untrusted/go-env',GOWORK='/untrusted/go.work'):
            # Act
            env=checks.environment()
        # Assert
        self.assertEqual(env['GOFLAGS'],'')
        self.assertEqual(env['GOENV'],'off')
        self.assertEqual(env['GOWORK'],'off')
        self.assertEqual(env['GOSUMDB'],'sum.golang.org')

    def test_linux_and_ci_same_registry(self):
        # Arrange
        spec=checks.registry()
        # Act
        ids=[lane['id'] for lane in spec['lanes'] if 'check' in lane['profiles']]
        workflow=(checks.ROOT/'.github/workflows/go.yml').read_text()
        # Assert
        self.assertEqual(len(ids),len(set(ids)))
        self.assertIn('python3 scripts/checks.py check --linux-native',workflow)
        self.assertIn('cache-dependency-path: "**/go.sum"',workflow)
        self.assertEqual(set(spec['modules']),{'.','contracttest','integrations/recipes'})
        self.assertEqual(spec['release_modules'],['.'])


if __name__=='__main__': unittest.main()
