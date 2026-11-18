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


class SelfscanReportTests(unittest.TestCase):
    def test_complete_scan_is_not_called_secure(self):
        report = audit.summarize(SHA, 'team', 'Repository-Detective',
            {'id': 'scan-123456789', 'ref': SHA, 'commit_sha': SHA, 'status': 'completed'},
            [{'scanner_name': 'gitleaks', 'status': 'clean', 'findings_count': 0},
             {'scanner_name': 'govulncheck', 'status': 'found', 'findings_count': 2}],
            {'version': 'test'}, ['gitleaks', 'govulncheck'])
        self.assertEqual('REVIEW_FINDINGS', report['decision'])
        self.assertEqual(2, report['scanner_finding_events'])
        self.assertIn('not deduplicated', audit.markdown(report))
        self.assertNotIn('SAFE', report['decision'])

    def test_sha_mismatch_fails_closed(self):
        report = audit.summarize(SHA, 'team', 'repo',
            {'id':'scan-123456789','ref':SHA,'commit_sha':'b'*40,'status':'completed'},
            [{'scanner_name':'gitleaks','status':'clean','findings_count':0}], {}, ['gitleaks'])
        self.assertEqual('EVIDENCE_INCOMPLETE', report['decision'])
        self.assertTrue(any('differs' in x for x in report['release_blockers']))

    def test_unexecuted_scanner_fails(self):
        report = audit.summarize(SHA, 'team','repo', {'id':'scan-123456789','ref':SHA,'commit_sha':SHA,'status':'completed'},
            [{'scanner_name':'gitleaks','status':'binary_missing','findings_count':0}], {}, ['gitleaks'])
        self.assertEqual('EVIDENCE_INCOMPLETE', report['decision'])

    def test_empty_and_unverified_results_fail(self):
        report = audit.summarize(SHA,'team','repo',{'status':'completed','ref':SHA,'commit_sha':SHA},[],{},[])
        self.assertEqual('EVIDENCE_INCOMPLETE',report['decision'])

    def test_report_artifacts_and_hash(self):
        report = audit.summarize(SHA,'team','repo',{'status':'completed','ref':SHA,'commit_sha':SHA,'id':'test-scan'},
                                 [{'scanner_name':'gitleaks','status':'clean','findings_count':0}],{},['gitleaks'])
        with tempfile.TemporaryDirectory() as d:
            p = pathlib.Path(d)
            audit.write_report(p,report)
            self.assertIn('expected_commit', (p/'executive-selfscan.json').read_text())
            self.assertIn(SHA, (p/'executive-selfscan.md').read_text())
            self.assertIn('executive-selfscan.json', (p/'SHA256SUMS').read_text())


if __name__ == '__main__': unittest.main()
