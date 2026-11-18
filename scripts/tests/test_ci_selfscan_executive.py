"""Offline contract checks for the commit-pinned self-audit report."""
import importlib.util
import pathlib
import tempfile
import unittest

MODULE = pathlib.Path(__file__).resolve().parents[1] / 'ci-selfscan-executive.py'
spec = importlib.util.spec_from_file_location('rd_selfaudit', MODULE)
audit = importlib.util.module_from_spec(spec)
spec.loader.exec_module(audit)
SHA = 'a' * 40


def identity(repo='team/Repository-Detective', forge='gitea', rid=42):
    return {'id': rid, 'full_name': repo, 'forge_type': forge}


def scan(repo_id=42):
    return {'id': 'scan-123456789', 'repository_id': repo_id, 'ref': SHA,
            'commit_sha': SHA, 'status': 'completed'}


class SelfscanReportTests(unittest.TestCase):
    def test_complete_scan_is_not_called_secure(self):
        report = audit.summarize(SHA, 'team', 'Repository-Detective',
            scan(),
            [{'scanner_name': 'gitleaks', 'status': 'clean', 'findings_count': 0},
             {'scanner_name': 'govulncheck', 'status': 'found', 'findings_count': 2}],
            {'version': 'test'}, ['gitleaks', 'govulncheck'], identity=identity())
        self.assertEqual('REVIEW_FINDINGS', report['decision'])
        self.assertEqual(2, report['scanner_finding_events'])
        self.assertIn('not deduplicated', audit.markdown(report))
        self.assertNotIn('SAFE', report['decision'])

    def test_sha_mismatch_fails_closed(self):
        report = audit.summarize(SHA, 'team', 'repo',
            {**scan(), 'commit_sha':'b'*40},
            [{'scanner_name':'gitleaks','status':'clean','findings_count':0}], {}, ['gitleaks'], identity=identity('team/repo'))
        self.assertEqual('EVIDENCE_INCOMPLETE', report['decision'])
        self.assertTrue(any('differs' in x for x in report['release_blockers']))

    def test_unexecuted_scanner_fails(self):
        report = audit.summarize(SHA, 'team','repo', scan(),
            [{'scanner_name':'gitleaks','status':'binary_missing','findings_count':0}], {}, ['gitleaks'], identity=identity('team/repo'))
        self.assertEqual('EVIDENCE_INCOMPLETE', report['decision'])

    def test_empty_and_unverified_results_fail(self):
        report = audit.summarize(SHA,'team','repo',scan(),[],{},[],identity=identity('team/repo'))
        self.assertEqual('EVIDENCE_INCOMPLETE',report['decision'])

    def test_empty_required_scanner_policy_never_passes(self):
        report = audit.summarize(SHA, 'team', 'Repository-Detective', scan(),
                                 [{'scanner_name': 'gitleaks', 'status': 'clean', 'findings_count': 0}],
                                 {}, [], identity=identity())
        self.assertEqual('EVIDENCE_INCOMPLETE', report['decision'])
        self.assertTrue(any('policy is empty' in issue for issue in report['release_blockers']))

    def test_report_artifacts_and_hash(self):
        report = audit.summarize(SHA,'team','repo',scan(),
                                 [{'scanner_name':'gitleaks','status':'clean','findings_count':0}],{},['gitleaks'], identity=identity('team/repo'))
        with tempfile.TemporaryDirectory() as d:
            p = pathlib.Path(d)
            audit.write_report(p,report)
            self.assertIn('expected_commit', (p/'executive-selfscan.json').read_text())
            self.assertIn(SHA, (p/'executive-selfscan.md').read_text())
            self.assertIn('executive-selfscan.json', (p/'SHA256SUMS').read_text())
            self.assertIn('executive-selfscan.md', (p/'SHA256SUMS').read_text())

    def test_wrong_repository_and_forge_fail_closed(self):
        result = [{'scanner_name': 'gitleaks', 'status': 'clean', 'findings_count': 0}]
        for mismatch in [identity('other/repository'), identity(forge='github'), identity(rid=99), None]:
            with self.subTest(mismatch=mismatch):
                report = audit.summarize(SHA, 'team', 'Repository-Detective', scan(), result,
                                         {}, ['gitleaks'], identity=mismatch, expected_scan_id='scan-123456789')
                self.assertEqual('EVIDENCE_INCOMPLETE', report['decision'])

    def test_new_scan_id_is_required(self):
        report = audit.summarize(SHA, 'team', 'Repository-Detective', scan(),
                                 [{'scanner_name': 'gitleaks', 'status': 'clean', 'findings_count': 0}],
                                 {}, ['gitleaks'], identity=identity(), expected_scan_id='new-scan-id')
        self.assertEqual('EVIDENCE_INCOMPLETE', report['decision'])
        self.assertTrue(any('Scan ID differs' in x for x in report['release_blockers']))

    def test_non_object_scanner_response_is_incomplete(self):
        report = audit.summarize(SHA, 'team', 'Repository-Detective', scan(), ['invalid'],
                                 {}, ['gitleaks'], identity=identity())
        self.assertEqual('EVIDENCE_INCOMPLETE', report['decision'])


if __name__ == '__main__': unittest.main()
