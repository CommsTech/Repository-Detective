#!/usr/bin/env python3
"""Fail closed before publishing a release without a commit-specific self-scan.

This validates *evidence integrity and scan coverage*, not the security of the
code. REVIEW_FINDINGS remains a review warning, never a clean certification.
"""
import argparse
import hashlib
import json
import pathlib
import re
import sys

SHA = re.compile(r'^[a-f0-9]{40}$')
SCANNER = re.compile(r'^[a-z][a-z0-9_.+\-]{0,63}$')
FILES = ('executive-selfscan.json', 'executive-selfscan.md')
SUCCESS = frozenset({'found', 'clean', 'ok', 'passed'})
DECISIONS = frozenset({'REVIEW_FINDINGS', 'REVIEW_COVERAGE'})
MAX_SIZE = 2 * 1024 * 1024


class EvidenceError(Exception):
    """Missing, inconsistent or unverifiable evidence."""


def check_evidence(directory: pathlib.Path, sha: str, forge: str, repository: str) -> dict:
    if not SHA.fullmatch(sha) or not repository or '/' not in repository:
        raise EvidenceError('Expected full commit SHA and owner/repository are required')
    if forge not in ('gitea', 'github'):
        raise EvidenceError('Expected forge must be gitea or github')
    if not directory.is_dir():
        raise EvidenceError('Self-scan artifact directory does not exist')

    sums = directory / 'SHA256SUMS'
    if not sums.is_file() or sums.is_symlink() or sums.stat().st_size > 1024:
        raise EvidenceError('Missing or invalid SHA256SUMS')
    entries = {}
    for line in sums.read_text(encoding='ascii').splitlines():
        match = re.fullmatch(r'([a-f0-9]{64})  (executive-selfscan\.(?:md|json))', line)
        if not match or match.group(2) in entries:
            raise EvidenceError('Checksum manifest contains invalid, duplicate, or unsafe entries')
        entries[match.group(2)] = match.group(1)
    if set(entries) != set(FILES):
        raise EvidenceError('Checksum manifest does not cover exactly both release evidence files')
    for name in FILES:
        path = directory / name
        if not path.is_file() or path.is_symlink() or path.stat().st_size > MAX_SIZE:
            raise EvidenceError('Missing or oversized evidence file: ' + name)
        if hashlib.sha256(path.read_bytes()).hexdigest() != entries[name]:
            raise EvidenceError('Evidence checksum does not match: ' + name)

    try:
        report = json.loads((directory / FILES[0]).read_text(encoding='utf-8'))
    except (ValueError, UnicodeError) as exc:
        raise EvidenceError('Executive JSON is malformed') from exc
    if not isinstance(report, dict) or report.get('schema_version') != 1:
        raise EvidenceError('Unsupported report schema')
    if report.get('report_type') != 'repository-detective-executive-self-scan':
        raise EvidenceError('Report is not a Repository Detective self-scan')
    decision = report.get('decision')
    if not isinstance(decision, str) or decision not in DECISIONS or report.get('release_blockers') != []:
        raise EvidenceError('Self-scan evidence is incomplete or has unresolved collection blockers')
    if report.get('expected_commit') != sha or report.get('reported_commit') != sha:
        raise EvidenceError('Scan does not attest the exact publishing commit')
    if report.get('scan_status') != 'completed' or not isinstance(report.get('scan_id'), str) or not re.fullmatch(r'[A-Za-z0-9_-]{8,128}', report['scan_id']):
        raise EvidenceError('Scan is not a completed, identified execution')
    for key in ('repository', 'reported_repository'):
        value = report.get(key)
        if not isinstance(value, str) or value.casefold() != repository.casefold():
            raise EvidenceError('Scan repository identity does not match publishing repository')
    if report.get('reported_forge') != forge:
        raise EvidenceError('Scan forge identity does not match publishing forge')
    required, results = report.get('required_scanners'), report.get('scanner_results')
    if not isinstance(required, list) or not required or not isinstance(results, list) or not results:
        raise EvidenceError('Required scanner policy or scanner evidence is empty')
    observed, counts = {}, 0
    for item in results:
        if not isinstance(item, dict):
            raise EvidenceError('Malformed scanner outcome')
        name, status, count = item.get('scanner'), item.get('status'), item.get('findings_count')
        if (not isinstance(name, str) or not SCANNER.fullmatch(name) or name in observed
                or not isinstance(status, str) or status not in SUCCESS | {'disabled', 'binary_missing'}
                or type(count) is not int or count < 0):
            raise EvidenceError('Invalid, duplicate, or failed scanner outcome')
        observed[name] = status
        counts += count
    if any(not isinstance(x, str) or not SCANNER.fullmatch(x) for x in required):
        raise EvidenceError('Malformed required scanner policy')
    if len(required) != len(set(required)) or any(observed.get(x) not in SUCCESS for x in required):
        raise EvidenceError('A required scanner was not successfully observed')
    if type(report.get('scanner_finding_events')) is not int or report['scanner_finding_events'] != counts:
        raise EvidenceError('Reported finding events disagree with scanner evidence')
    if report['decision'] != ('REVIEW_FINDINGS' if counts else 'REVIEW_COVERAGE'):
        raise EvidenceError('Executive decision disagrees with scanner evidence')
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--dir', default='dist/selfscan')
    parser.add_argument('--sha', required=True)
    parser.add_argument('--forge', choices=('gitea', 'github'), required=True)
    parser.add_argument('--repository', required=True, help='owner/repository being published')
    args = parser.parse_args()
    try:
        report = check_evidence(pathlib.Path(args.dir), args.sha.lower(), args.forge, args.repository)
    except (EvidenceError, OSError) as exc:
        print('RELEASE BLOCKED: ' + str(exc), file=sys.stderr)
        return 1
    print('Evidence gate PASS: ' + report['expected_commit'] + ' / ' + report['scan_id'])
    if report['decision'] == 'REVIEW_FINDINGS':
        print('ACTION REQUIRED: scanner findings exist. The evidence gate does NOT approve or certify the release.')
    else:
        print('Coverage reviewed; clean scanner outcomes do NOT prove this release is safe.')
    return 0


if __name__ == '__main__':
    sys.exit(main())
