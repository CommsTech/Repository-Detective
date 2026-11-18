#!/usr/bin/env python3
"""Generate an evidence-backed executive report from a commit-pinned RD self-scan.

Uses the existing /api/v1/analyze + scan evidence APIs, not an invented second
scanner. No forge issues or PRs are created. Requires an authenticated staging
RD instance with the project repository connected and scanner tools installed.
"""
import argparse
import datetime as dt
import hashlib
import json
import os
import pathlib
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

SUCCESS = {'found', 'clean', 'ok', 'passed'}
STOP = {'completed', 'failed', 'partial', 'cancelled', 'canceled', 'ingestion_failed'}
SHA = re.compile(r'^[a-fA-F0-9]{40}$')


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise RuntimeError('API redirected; refusing to forward credentials')


def api(base, key, method, path, payload=None):
    data = json.dumps(payload).encode() if payload is not None else None
    req = urllib.request.Request(base + path, data=data, method=method,
        headers={'X-Repository-Detective-API-Key': key, 'Content-Type': 'application/json', 'Accept': 'application/json'})
    try:
        with urllib.request.build_opener(NoRedirect()).open(req, timeout=45) as r:
            return json.load(r)
    except urllib.error.HTTPError as e:
        # Never write response bodies that might contain secrets to CI logs.
        raise RuntimeError(f'API {method} {path.split("?")[0]} returned HTTP {e.code}') from None


def summarize(expected, owner, repo, scan, outcomes, engine, required):
    problems = []
    commit = str(scan.get('commit_sha') or '').lower()
    if commit != expected.lower():
        problems.append('Scan commit differs from the pushed commit or is missing')
    if str(scan.get('ref') or '').lower() != expected.lower() and not scan.get('commit_pinned'):
        problems.append('Scan ref is not commit-specific and commit_pinned is not true')
    if scan.get('status') != 'completed':
        problems.append('Scan was not completed successfully')
    if not outcomes:
        problems.append('No per-scanner evidence was returned')
    details = []
    seen = set()
    for item in outcomes:
        name = str(item.get('scanner_name') or '').lower().strip()
        status = str(item.get('status') or '').lower().strip()
        count = item.get('findings_count', 0)
        if not name or name in seen or not isinstance(count, int) or count < 0:
            problems.append('Malformed or duplicate scanner result')
            continue
        seen.add(name)
        details.append({'scanner': name, 'status': status, 'findings_count': count})
        if status not in SUCCESS or item.get('error'):
            problems.append('Scanner did not produce usable evidence: ' + name)
    for name in required:
        if name not in seen or next((d['status'] for d in details if d['scanner'] == name), '') not in SUCCESS:
            problems.append('Required scanner missing or incomplete: ' + name)
    details.sort(key=lambda x: x['scanner'])
    observed = sum(d['findings_count'] for d in details if d['status'] in SUCCESS)
    decision = 'EVIDENCE_INCOMPLETE' if problems else ('REVIEW_FINDINGS' if observed else 'REVIEW_COVERAGE')
    return {'schema_version': 1, 'report_type': 'repository-detective-executive-self-scan',
        'generated_utc': dt.datetime.now(dt.timezone.utc).isoformat(), 'repository': owner + '/' + repo,
        'expected_commit': expected.lower(), 'reported_commit': commit,
        'scan_id': str(scan.get('id') or ''), 'scan_status': str(scan.get('status') or ''),
        'workspace_mode': str(scan.get('workspace_mode_used') or ''),
        'engine_version': str(engine.get('version') or 'unknown'), 'engine_commit': str(engine.get('commit') or 'unknown'),
        'scanner_results': details, 'required_scanners': required,
        'scanner_finding_events': observed, 'decision': decision, 'limitations': [
            'Scanner counts are per-tool events, not deduplicated open findings or verified CVEs.',
            'A completed scan does not prove absence of vulnerabilities; untested files and attack paths remain possible.',
            'Self-scanning does not replace independent security review or tests on released binaries.',
        ], 'release_blockers': sorted(set(problems))}


def markdown(report):
    out = ['# Repository Detective — executive self-scan report', '',
        '**Decision:** ' + report['decision'], '',
        '| Evidence | Value |', '|---|---|',
        '| Repository | `' + report['repository'] + '` |',
        '| Expected pushed commit | `' + report['expected_commit'] + '` |',
        '| Scan commit | `' + report['reported_commit'] + '` |',
        '| Scan ID | `' + report['scan_id'] + '` |',
        '| Scan status | ' + report['scan_status'] + ' |',
        '| Scanner engine | `' + report['engine_version'] + '` (`' + report['engine_commit'] + '`) |',
        '| Generated (UTC) | ' + report['generated_utc'] + ' |',
        '| Scanner finding events (not unique) | ' + str(report['scanner_finding_events']) + ' |', '',
        '## Scanner execution evidence', '', '| Scanner | Outcome | Finding events |', '|---|---|---:|']
    for s in report['scanner_results']:
        out.append('| `' + s['scanner'] + '` | ' + s['status'] + ' | ' + str(s['findings_count']) + ' |')
    if not report['scanner_results']:
        out.append('| — | no evidence | — |')
    out += ['', '## Release blockers / missing proof', '']
    out += ['- ' + p for p in report['release_blockers']] or ['- No automatic evidence-collection blockers; review findings and coverage before release.']
    out += ['', '## Limitations', ''] + ['- ' + x for x in report['limitations']]
    out += ['', '> This is a point-in-time, commit-specific automated report, not a safety certification or a promise that all defects were found.', '']
    return '\n'.join(out)


def write_report(out_dir, report):
    out_dir.mkdir(parents=True, exist_ok=True)
    raw = json.dumps(report, indent=2, sort_keys=True) + '\n'
    (out_dir / 'executive-selfscan.json').write_text(raw, encoding='utf-8')
    (out_dir / 'executive-selfscan.md').write_text(markdown(report), encoding='utf-8')
    digest = hashlib.sha256(raw.encode()).hexdigest()
    (out_dir / 'SHA256SUMS').write_text(digest + '  executive-selfscan.json\n', encoding='utf-8')


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--output', default='dist/selfscan')
    p.add_argument('--sha', default=os.getenv('GITHUB_SHA', ''))
    p.add_argument('--owner', default=os.getenv('RD_SELFSCAN_OWNER', ''))
    p.add_argument('--repo', default=os.getenv('RD_SELFSCAN_REPO', 'Repository-Detective'))
    p.add_argument('--forge', choices=('gitea', 'github'), default='gitea')
    p.add_argument('--timeout', type=int, default=1800)
    p.add_argument('--required', default='gitleaks,govulncheck,staticcheck')
    args = p.parse_args()
    sha, base, key = args.sha.strip().lower(), os.getenv('RD_SELFSCAN_URL', '').rstrip('/'), os.getenv('RD_SELFSCAN_API_KEY', '')
    if not SHA.fullmatch(sha):
        p.error('require the full 40-character pushed commit SHA')
    if not base or not key or not args.owner:
        report = summarize(sha, args.owner or '(not configured)', args.repo, {}, [], {}, [])
        report['release_blockers'].append('Self-scan URL, API key, and repository owner must be configured for CI')
        write_report(pathlib.Path(args.output), report)
        print('EVIDENCE_INCOMPLETE: self-scan configuration missing; failure artifact written', file=sys.stderr)
        return 1
    if not re.fullmatch(r'[A-Za-z0-9_.-]+', args.owner) or not re.fullmatch(r'[A-Za-z0-9_.-]+', args.repo):
        p.error('require explicit owner and repository (alphanumerics, dot, dash, underscore)')
    u = urllib.parse.urlsplit(base)
    if u.scheme != 'https' and not (u.scheme == 'http' and u.hostname in ('localhost', '127.0.0.1', '::1')):
        p.error('self-scan API must use HTTPS (except localhost)')
    if u.username or u.password or u.query or u.fragment or not u.hostname:
        p.error('self-scan API URL contains unsupported components')
    required = sorted(set(s.strip().lower() for s in args.required.split(',') if s.strip()))
    scan, evidence, engine = {}, [], {}
    failure = ''
    try:
        engine = api(base, key, 'GET', '/api/v1/about')
        started = api(base, key, 'POST', '/api/v1/analyze',
            {'forge_type': args.forge, 'owner': args.owner, 'repository': args.repo,
             'ref': sha, 'report_only_dry_run': True, 'scan_profile': 'standard_deterministic'})
        scan_id = str(started.get('scan_id') or '')
        if not re.fullmatch(r'[A-Za-z0-9_-]{8,128}', scan_id):
            raise RuntimeError('analyze endpoint did not return a valid scan ID')
        deadline = time.monotonic() + args.timeout
        while time.monotonic() < deadline:
            scan = api(base, key, 'GET', '/api/v1/scans/' + scan_id)
            if str(scan.get('status')) in STOP:
                break
            time.sleep(10)
        else:
            raise RuntimeError('scan exceeded timeout')
        evidence = api(base, key, 'GET', '/api/v1/scans/' + scan_id + '/scanner-results').get('scanner_results', [])
    except Exception as exc:
        failure = str(exc)
    report = summarize(sha, args.owner, args.repo, scan, evidence, engine, required)
    if failure:
        report['decision'] = 'EVIDENCE_INCOMPLETE'
        report['release_blockers'].append('Self-scan request or evidence collection failed: ' + failure)
    write_report(pathlib.Path(args.output), report)
    print('Self-scan report:', report['decision'], 'commit:', sha, 'scan:', report['scan_id'])
    print('Artifact:', pathlib.Path(args.output) / 'executive-selfscan.md')
    return 0 if report['decision'] != 'EVIDENCE_INCOMPLETE' else 1


if __name__ == '__main__':
    sys.exit(main())
