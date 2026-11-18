#!/usr/bin/env python3
"""Read-only HTTP response timing; prints no credentials or response contents.

Examples:
  python3 scripts/bench-http-latency.py --base http://127.0.0.1:8081 --path /health/ready
  RD_BENCH_API_KEY=<token> python3 scripts/bench-http-latency.py --base http://127.0.0.1:8081 --path /ui/findings --iterations 30
"""
import argparse
import os
import statistics
import time
import urllib.error
import urllib.parse
import urllib.request


def percentile(values, percent):
    ordered = sorted(values)
    return ordered[max(0, min(len(ordered) - 1, (len(ordered) * percent + 99) // 100 - 1))]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base', default='http://127.0.0.1:8081')
    parser.add_argument('--path', action='append', default=[])
    parser.add_argument('--iterations', type=int, default=20)
    args = parser.parse_args()
    if not 1 <= args.iterations <= 300:
        parser.error('iterations must be between 1 and 300')
    base = args.base.rstrip('/')
    parsed = urllib.parse.urlparse(base)
    if parsed.scheme not in ('http', 'https') or not parsed.netloc or parsed.username or parsed.password:
        parser.error('base must be an absolute HTTP(S) URL without embedded credentials')
    paths = args.path or ['/health/ready']
    token = os.getenv('RD_BENCH_API_KEY', '')
    for path in paths:
        if not path.startswith('/') or path.startswith('//'):
            parser.error('paths must start with one /')
        url = base + path
        durations, errors = [], {}
        for _ in range(args.iterations):
            request = urllib.request.Request(url)
            if token:
                request.add_header('X-Repository-Detective-API-Key', token)
            start = time.perf_counter()
            try:
                with urllib.request.urlopen(request, timeout=20) as response:
                    response.read()
                    status = response.status
            except urllib.error.HTTPError as exc:
                status = exc.code
            except (urllib.error.URLError, TimeoutError):
                status = 'network-error'
            elapsed = (time.perf_counter() - start) * 1000
            if status == 200:
                durations.append(elapsed)
            else:
                errors[status] = errors.get(status, 0) + 1
        if durations:
            print(f'{path}: ok={len(durations)}/{args.iterations} p50={statistics.median(durations):.1f}ms p95={percentile(durations,95):.1f}ms max={max(durations):.1f}ms')
        else:
            print(f'{path}: no successful responses')
        if errors:
            print(f'  non-200 responses: {errors}')


if __name__ == '__main__':
    main()
