#!/usr/bin/env python3
"""Generate an evidence-backed executive report from a commit-pinned RD self-scan.

Uses the existing /api/v1/analyze + scan evidence APIs, not an invented second
scanner. No forge issues or PRs are created. Requires an authenticated staging
RD instance with the project repository connected and scanner tools installed.
"""
import argparse
import datetime as dt
import hashlib
import ipaddress
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
            result = json.load(r)
            if not isinstance(result, dict):
                raise RuntimeError('API response is not a JSON object')
            return result
    except urllib.error.HTTPError as e:
        # Never write response bodies that might contain secrets to CI logs.
        raise RuntimeError(f'API {method} {path.split("?")[0]} returned HTTP {e.code}') from None


def summarize(expected, owner, repo, scan, outcomes, engine, required,
              identity=None, expected_forge='gitea', expected_scan_id=''):
    """Fail closed unless the *scan*, repository record and forge identify one revision.

    A caller-supplied repository name is not proof that the scan result belongs
    to that repository: the scan's persisted repository_id must match a second,
    authenticated repository lookup.
    """
    problems = []
    commit = str(scan.get('commit_sha') or '').lower()
    if commit != expected.lower():
        problems.append('Scan commit differs from the pushed commit or is missing')
    if str(scan.get('ref') or '').lower() != expected.lower() and not scan.get('commit_pinned'):
        problems.append('Scan ref is not commit-specific and commit_pinned is not true')
    if scan.get('status') != 'completed':
        problems.append('Scan was not completed successfully')
    if expected_scan_id and str(scan.get('id') or '') != expected_scan_id:
        problems.append('Scan ID differs from the newly requested scan')
    identity = identity if isinstance(identity, dict) else {}
    reported_repo = str(identity.get('full_name') or '')
    reported_forge = str(identity.get('forge_type') or '').lower()
    repo_id = scan.get('repository_id')
    if not isinstance(repo_id, int) or repo_id <= 0 or identity.get('id') != repo_id:
        problems.append('Scan repository ID cannot be authenticated against repository record')
    if reported_repo.casefold() != (owner + '/' + repo).casefold():
        problems.append('Scan repository differs from requested repository or is missing')
    if reported_forge != expected_forge.lower():
        problems.append('Scan forge differs from requested forge or is missing')
    if not outcomes:
        problems.append('No per-scanner evidence was returned')
    if not required:
        problems.append('Required scanner policy is empty; cannot assert scan coverage')
    details = []
    seen = set()
    if not isinstance(outcomes, list):
        problems.append('Scanner results response is malformed')
        outcomes = []
    for item in outcomes:
        if not isinstance(item, dict):
            problems.append('Scanner result is not an object')
            continue
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
        'reported_repository': reported_repo, 'reported_forge': reported_forge,
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
        '| Verified repository record | `' + report['reported_repository'] + '` |',
        '| Verified forge | `' + report['reported_forge'] + '` |',
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
    md_digest = hashlib.sha256((out_dir / 'executive-selfscan.md').read_bytes()).hexdigest()
    (out_dir / 'SHA256SUMS').write_text(
        digest + '  executive-selfscan.json\n' + md_digest + '  executive-selfscan.md\n',
        encoding='utf-8')


def _private_host(host):
    """True for loopback or RFC1918 literals reachable only on the local network.

    In-LAN CI runners reach the scan service over the host's private bridge, so
    plain HTTP there never leaves local infrastructure. Public addresses and
    hostnames still require HTTPS.
    """
    if not host:
        return False
    if host in ('localhost', '::1'):
        return True
    try:
        addr = ipaddress.ip_address(host)
    except ValueError:
        return False
    return addr.is_loopback or addr.is_private


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
    required = sorted(set(s.strip().lower() for s in args.required.split(',') if s.strip()))
    configuration_errors = []
    if not required:
        configuration_errors.append('At least one required scanner must be configured')
    if len(required) > 32 or any(not re.fullmatch(r'[a-z][a-z0-9_.+\-]{0,63}', name) for name in required):
        configuration_errors.append('Required scanner names must be short, valid identifiers (maximum 32)')
    if not base or not key or not args.owner:
        configuration_errors.append('Self-scan URL, API key, and repository owner must be configured for CI')
    if not re.fullmatch(r'[A-Za-z0-9_.-]+', args.owner) or not re.fullmatch(r'[A-Za-z0-9_.-]+', args.repo):
        configuration_errors.append('Repository owner and name are not valid')
    u = urllib.parse.urlsplit(base)
    if u.scheme != 'https' and not (u.scheme == 'http' and _private_host(u.hostname)):
        configuration_errors.append(
            'Self-scan URL must use HTTPS (plain HTTP is accepted only for loopback '
            'or RFC1918 addresses inside the trusted local network)')
    if u.username or u.password or u.query or u.fragment or not u.hostname:
        configuration_errors.append('Self-scan API URL has unsupported components')
    if configuration_errors:
        report = summarize(sha, args.owner or '(not configured)', args.repo, {}, [], {}, required,
                           expected_forge=args.forge)
        report['release_blockers'] = sorted(set(report['release_blockers'] + configuration_errors))
        write_report(pathlib.Path(args.output), report)
        print('EVIDENCE_INCOMPLETE: self-scan configuration invalid; report artifact written', file=sys.stderr)
        return 1
    scan, evidence, engine, identity = {}, [], {}, {}
    failure = ''
    try:
        engine = api(base, key, 'GET', '/api/v1/about')
        started = api(base, key, 'POST', '/api/v1/analyze',
            {'forge_type': args.forge, 'owner': args.owner, 'repository': args.repo,
             'ref': sha, 'report_only_dry_run': True, 'scan_profile': 'standard_deterministic'})
        scan_id = str(started.get('scan_id') or '')
        if started.get('report_only_dry_run') is not True:
            raise RuntimeError('Analyze API did not acknowledge report-only dry-run; refusing to trust this run')
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
        repository_id = scan.get('repository_id')
        if isinstance(repository_id, int) and repository_id > 0:
            identity = api(base, key, 'GET', '/api/v1/repos/' + str(repository_id))
        evidence = api(base, key, 'GET', '/api/v1/scans/' + scan_id + '/scanner-results').get('scanner_results', [])
    except Exception as exc:
        failure = str(exc)
    report = summarize(sha, args.owner, args.repo, scan, evidence, engine, required,
                       identity=identity, expected_forge=args.forge,
                       expected_scan_id=locals().get('scan_id', ''))
    if failure:
        report['decision'] = 'EVIDENCE_INCOMPLETE'
        report['release_blockers'].append('Self-scan request or evidence collection failed: ' + failure)
    write_report(pathlib.Path(args.output), report)
    print('Self-scan report:', report['decision'], 'commit:', sha, 'scan:', report['scan_id'])
    print('Artifact:', pathlib.Path(args.output) / 'executive-selfscan.md')
    return 0 if report['decision'] != 'EVIDENCE_INCOMPLETE' else 1


if __name__ == '__main__':
    sys.exit(main())
