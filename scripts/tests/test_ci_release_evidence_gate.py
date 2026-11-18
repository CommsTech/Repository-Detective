"""Offline tests for release evidence fail-closed behavior."""
import hashlib
import importlib.util
import json
import pathlib
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).resolve().parents[1] / 'ci-release-evidence-gate.py'
spec = importlib.util.spec_from_file_location('rd_release_gate', SCRIPT)
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)
REPORT_SCRIPT = pathlib.Path(__file__).resolve().parents[1] / 'ci-selfscan-executive.py'
spec2 = importlib.util.spec_from_file_location('rd_selfscan', REPORT_SCRIPT)
selfscan = importlib.util.module_from_spec(spec2)
spec2.loader.exec_module(selfscan)
SHA = 'c' * 40
REPO = 'team/Repository-Detective'


def report():
    return selfscan.summarize(SHA, 'team', 'Repository-Detective',
        {'id': 'scan-example-123456', 'repository_id': 9, 'ref': SHA, 'commit_sha': SHA,
         'status': 'completed'},
        [{'scanner_name': 'gitleaks', 'status': 'clean', 'findings_count': 0},
         {'scanner_name': 'govulncheck', 'status': 'found', 'findings_count': 2}],
        {'version': 'test'}, ['gitleaks', 'govulncheck'],
        identity={'id': 9, 'full_name': REPO, 'forge_type': 'gitea'},
        expected_scan_id='scan-example-123456')


class ReleaseEvidenceTests(unittest.TestCase):
    def check_bad(self, obj):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            selfscan.write_report(root, report())
            # Mutate a well-formed fixture with its SHA-256 manifest updated.
            # This tests gate validation, not the report generator's renderer.
            (root / 'executive-selfscan.json').write_text(json.dumps(obj))
            entries = [(hashlib.sha256((root / name).read_bytes()).hexdigest(), name)
                       for name in gate.FILES]
            (root / 'SHA256SUMS').write_text(''.join(digest + '  ' + name + '\n' for digest, name in entries))
            with self.assertRaises(gate.EvidenceError):
                gate.check_evidence(root, SHA, 'gitea', REPO)

    def test_complete_coverage_with_findings_passes_with_warning_status(self):
        with tempfile.TemporaryDirectory() as tmp:
            selfscan.write_report(pathlib.Path(tmp), report())
            result = gate.check_evidence(pathlib.Path(tmp), SHA, 'gitea', REPO)
            self.assertEqual('REVIEW_FINDINGS', result['decision'])

    def test_zero_findings_only_valid_as_review_coverage(self):
        obj = report()
        obj['scanner_results'][1]['findings_count'] = 0
        obj['scanner_results'][1]['status'] = 'clean'
        obj['scanner_finding_events'] = 0
        obj['decision'] = 'REVIEW_COVERAGE'
        with tempfile.TemporaryDirectory() as tmp:
            selfscan.write_report(pathlib.Path(tmp), obj)
            self.assertEqual('REVIEW_COVERAGE', gate.check_evidence(pathlib.Path(tmp), SHA, 'gitea', REPO)['decision'])

    def test_sha_repository_forge_and_incomplete_are_rejected(self):
        for field, value in [('expected_commit', 'a'*40), ('reported_commit', 'b'*40),
                             ('reported_repository', 'other/repo'), ('reported_forge', 'github'),
                             ('decision', 'EVIDENCE_INCOMPLETE'), ('scan_status', 'failed'),
                             ('release_blockers', ['missing evidence'])]:
            with self.subTest(field=field):
                obj = report(); obj[field] = value
                self.check_bad(obj)

    def test_missing_duplicate_failed_scanner_and_false_count_rejected(self):
        variations = [
            ('required_scanners', []), ('required_scanners', ['gitleaks', 'secret-scanner']),
            ('required_scanners', ['gitleaks', 'gitleaks']),
            ('scanner_results', []),
            ('scanner_results', [{'scanner':'gitleaks','status':'binary_missing','findings_count':0}]),
            ('scanner_finding_events', 0),
            ('required_scanners', [{'unexpected': 'object'}]),
            ('decision', ['REVIEW_FINDINGS']),
            ('scan_id', ['invalid']),
            ('decision', 'REVIEW_COVERAGE'),
        ]
        for field, value in variations:
            with self.subTest(field=field, value=value):
                obj = report(); obj[field] = value
                self.check_bad(obj)

    def test_tampering_and_missing_files_fail_even_if_json_is_valid(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            selfscan.write_report(root, report())
            (root/'executive-selfscan.md').write_text('modified')
            with self.assertRaises(gate.EvidenceError):
                gate.check_evidence(root, SHA, 'gitea', REPO)
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            selfscan.write_report(root, report())
            (root/'executive-selfscan.json').unlink()
            with self.assertRaises(gate.EvidenceError):
                gate.check_evidence(root, SHA, 'gitea', REPO)

    def test_manifest_additional_entry_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            selfscan.write_report(root, report())
            with (root/'SHA256SUMS').open('a') as manifest:
                manifest.write('0'*64 + '  ../malicious\n')
            with self.assertRaises(gate.EvidenceError):
                gate.check_evidence(root, SHA, 'gitea', REPO)

    def test_no_evidence_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaises(gate.EvidenceError):
                gate.check_evidence(pathlib.Path(tmp), SHA, 'gitea', REPO)


if __name__ == '__main__':
    unittest.main()

class WorkflowReleaseGateTests(unittest.TestCase):
    def test_all_publication_paths_are_gated_against_exact_commit(self):
        root = SCRIPT.parents[1]
        for forge in ('.gitea', '.github'):
            with self.subTest(forge=forge):
                data = (root / forge / 'workflows' / 'docker-publish.yml').read_text()
                verify, publish = data.split('  publish:' if forge == '.gitea' else '  mirror:', 1)
                self.assertIn('scripts/ci-selfscan-executive.py', verify)
                self.assertIn('scripts/ci-release-evidence-gate.py', verify)
                self.assertIn('--sha "${GITHUB_SHA}"', verify)
                self.assertIn('--repository "${GITHUB_REPOSITORY}"', verify)
                self.assertIn('--forge ' + ('gitea' if forge == '.gitea' else 'github'), verify)
                self.assertIn('if: always()', verify)
                self.assertIn('needs: verify', publish)
        release = (root / '.gitea/workflows/release.yml').read_text()
        self.assertLess(release.index('scripts/ci-release-evidence-gate.py'),
                        release.index('Build multi-platform binaries'))
        self.assertLess(release.index('scripts/ci-release-evidence-gate.py'),
                        release.index('Push image to Gitea Package Registry'))
        for artifact in ('executive-selfscan.json', 'executive-selfscan.md', 'executive-selfscan.SHA256SUMS'):
            self.assertIn('dist/' + artifact, release)
        self.assertIn('curl -fsS -o /dev/null -X POST', release)
        self.assertNotIn('skipping remote release upload', release)
