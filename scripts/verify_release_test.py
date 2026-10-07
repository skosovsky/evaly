"""AAA postpublication verifier negatives; no public refs are mutated."""
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import verify_release


class PublicEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.manifest={'Path':'github.com/skosovsky/evaly','Version':'v0.0.1','Sum':'h1:artifact','GoModSum':'h1:mod'}
        self.record=dict(self.manifest,Origin={'Hash':'a'*40})

    def test_mismatched_public_bytes_or_version_fail(self):
        for key in self.manifest:
            # Arrange
            record=dict(self.record); record[key]='wrong'
            # Act / Assert
            with self.subTest(key=key), self.assertRaisesRegex(RuntimeError,'public artifact differs'):
                verify_release.validate_download(record,self.manifest,'a'*40)

    def test_wrong_public_origin_fails(self):
        # Arrange
        record=dict(self.record,Origin={'Hash':'b'*40})
        # Act / Assert
        with self.assertRaisesRegex(RuntimeError,'origin differs'):
            verify_release.validate_download(record,self.manifest,'a'*40)

    def verify_fixture(self,steps):
        with tempfile.TemporaryDirectory() as temp:
            manifest=Path(temp)/'manifest.json'; manifest.write_text(json.dumps(self.manifest))
            with mock.patch.object(verify_release,'invoke',side_effect=steps), mock.patch.object(verify_release.time,'sleep'):
                verify_release.verify('v0.0.1','a'*40,manifest,Path(temp)/'public.json')

    def test_consumer_failure_cannot_be_release_success(self):
        # Arrange
        steps=[json.dumps(self.record),RuntimeError('consumer semantic failure')]
        # Act / Assert
        with self.assertRaisesRegex(RuntimeError,'consumer semantic failure'):
            self.verify_fixture(steps)

    def test_missing_or_failed_exact_ci_cannot_pass(self):
        cases=[([], 'did not complete'),
               ([dict(id=1,path='.github/workflows/go.yml',head_branch='v0.0.1',head_sha='a'*40,status='completed',conclusion='failure',html_url='https://example.invalid/run')],'CI failed'),
               ([dict(id=2,path='.github/workflows/go.yml',head_branch='old-tag',head_sha='a'*40,status='completed',conclusion='success',html_url='https://example.invalid/old')],'did not complete')]
        for runs,diagnostic in cases:
            # Arrange
            steps=[json.dumps(self.record),'smoke PASS']+[json.dumps({'workflow_runs':runs})]*180
            # Act / Assert
            with self.subTest(diagnostic=diagnostic), self.assertRaisesRegex(RuntimeError,diagnostic):
                self.verify_fixture(steps)

    def test_network_timeout_is_explicit_failure(self):
        # Arrange
        with mock.patch.object(verify_release.subprocess,'run',side_effect=subprocess.TimeoutExpired('go',90)):
            # Act / Assert
            with self.assertRaises(subprocess.TimeoutExpired):
                verify_release.invoke(['go','mod','download'],{})


if __name__=='__main__': unittest.main()
