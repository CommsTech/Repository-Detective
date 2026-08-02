#!/usr/bin/env bash
# Warm-response timing for Repository Detective UI/API pages.
# Usage:
#   source .env && ./scripts/ui-responsiveness-bench.sh [label]
# Env:
#   RD_BASE_URL / REPOSITORY_DETECTIVE_PUBLIC_BASE_URL (default http://127.0.0.1:8081)
#   REPOSITORY_DETECTIVE_API_KEY
#   RD_BENCH_SAMPLES (default 5)
#   RD_BENCH_REPORT (default docs/dogfood-reports/ui-responsiveness-bench.md)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

LABEL="${1:-run}"
SAMPLES="${RD_BENCH_SAMPLES:-5}"
BASE="${RD_BASE_URL:-${REPOSITORY_DETECTIVE_PUBLIC_BASE_URL:-http://127.0.0.1:8081}}"
BASE="${BASE%/}"
API_KEY="${REPOSITORY_DETECTIVE_API_KEY:-}"
REPORT="${RD_BENCH_REPORT:-docs/dogfood-reports/ui-responsiveness-bench.md}"

ROUTES=(
  "/ui"
  "/ui/repos"
  "/ui/health"
  "/ui/reports"
  "/ui/findings"
  "/ui/scans"
  "/ui/learning"
  "/ui/configure"
  "/ui/projects"
  "/ui/preinstall"
  "/api/v1/dashboard/summary"
  "/api/v1/repos"
)

mkdir -p "$(dirname "$REPORT")"
tmp="$(mktemp)"
{
  echo "# UI responsiveness benchmark — ${LABEL}"
  echo
  echo "Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo "Base: \`${BASE}\` (warm, ${SAMPLES} samples, times in ms)"
  echo
  echo "| Route | p50 | p90 | mean | min | max |"
  echo "|-------|-----|-----|------|-----|-----|"
} >"$tmp"

curl_args=(-sS -o /dev/null -w '%{time_total}' -m 60)
if [ -n "$API_KEY" ]; then
  curl_args+=(-H "X-Repository-Detective-API-Key: ${API_KEY}" -H "Authorization: Bearer ${API_KEY}")
fi

for route in "${ROUTES[@]}"; do
  times_csv=""
  for ((i = 1; i <= SAMPLES; i++)); do
    t="$(curl "${curl_args[@]}" "${BASE}${route}" || echo "0")"
    ms="$(python3 -c "print(int(float('${t}') * 1000))")"
    if [ -n "$times_csv" ]; then
      times_csv+=","
    fi
    times_csv+="${ms}"
  done
  stats="$(python3 - <<PY
times=sorted([int(x) for x in "${times_csv}".split(",") if x != ""])
if not times:
  print("0|0|0|0|0")
else:
  p50=times[len(times)//2]
  p90=times[min(len(times)-1, int(round(0.9*(len(times)-1))))]
  mean=sum(times)/len(times)
  print(f"{p50}|{p90}|{mean:.0f}|{times[0]}|{times[-1]}")
PY
)"
  IFS='|' read -r p50 p90 mean mn mx <<<"$stats"
  echo "| \`${route}\` | ${p50} | ${p90} | ${mean} | ${mn} | ${mx} |" >>"$tmp"
  printf '%s %s -> p50=%sms mean=%sms\n' "$LABEL" "$route" "$p50" "$mean"
done

cat "$tmp" >"$REPORT"
rm -f "$tmp"
echo "Wrote ${REPORT}"
