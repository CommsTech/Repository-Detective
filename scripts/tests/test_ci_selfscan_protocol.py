"""Simulated API contract test. No production server or forge connection."""
import http.server
import json
import os
from pathlib import Path
import subprocess
import tempfile
import threading
import unittest

SCRIPT = Path(__file__).resolve().parents[1] / 'ci-selfscan-executive.py'
SHA = 'a' * 40

class ScanAPI(http.server.BaseHTTPRequestHandler):
    def log_message(self, *args): pass
    def do_GET(self):
        if self.path == '/api/v1/about': body = {'version':'simulated','commit':'not-scanned-engine'}
        elif self.path == '/api/v1/scans/scan-123456789': body = {'id':'scan-123456789','ref':SHA,'commit_sha':SHA,'status':'completed'}
        elif self.path == '/api/v1/scans/scan-123456789/scanner-results': body = {'scanner_results':[{'scanner_name':'gitleaks','status':'found','findings_count':1}]}
        else: self.send_error(404);return
        self.send_response(200);self.send_header('Content-Type','application/json');self.end_headers();self.wfile.write(json.dumps(body).encode())
    def do_POST(self):
        if self.path != '/api/v1/analyze':self.send_error(404);return
        size = int(self.headers.get('Content-Length','0'))
        body = json.loads(self.rfile.read(size))
        assert body['ref'] == SHA and body['report_only_dry_run'] is True
        self.send_response(202);self.send_header('Content-Type','application/json');self.end_headers();self.wfile.write(json.dumps({'scan_id':'scan-123456789'}).encode())

class ProtocolTests(unittest.TestCase):
    def test_request_scan_and_uploadable_artifact(self):
        srv = http.server.ThreadingHTTPServer(('127.0.0.1',0),ScanAPI)
        thread=threading.Thread(target=srv.serve_forever,daemon=True);thread.start()
        try:
            with tempfile.TemporaryDirectory() as d:
                env={**os.environ,'RD_SELFSCAN_URL':f'http://127.0.0.1:{srv.server_port}','RD_SELFSCAN_API_KEY':'fake-test-key'}
                completed=subprocess.run(['python3',str(SCRIPT),'--sha',SHA,'--owner','team','--repo','Repository-Detective','--required','gitleaks','--output',d],env=env,capture_output=True,text=True,timeout=15)
                self.assertEqual(0,completed.returncode,completed.stderr)
                report=json.loads((Path(d)/'executive-selfscan.json').read_text())
                self.assertEqual('REVIEW_FINDINGS',report['decision'])
                self.assertEqual(SHA,report['reported_commit'])
                self.assertEqual(1,report['scanner_finding_events'])
        finally:
            srv.shutdown();srv.server_close()

    def test_configuration_failure_produces_failure_report(self):
        with tempfile.TemporaryDirectory() as d:
            env={**os.environ,'RD_SELFSCAN_URL':'','RD_SELFSCAN_API_KEY':''}
            completed=subprocess.run(['python3',str(SCRIPT),'--sha',SHA,'--owner','team','--output',d],env=env,capture_output=True,text=True,timeout=15)
            self.assertNotEqual(0,completed.returncode)
            self.assertEqual('EVIDENCE_INCOMPLETE',json.loads((Path(d)/'executive-selfscan.json').read_text())['decision'])

if __name__ == '__main__': unittest.main()
