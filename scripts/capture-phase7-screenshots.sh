#!/usr/bin/env bash
# Capture Phase 7 (RD-020) documentation screenshots from a disposable RD UI.
# Default target: e2e stack on :18081 — never production :8081 unless overridden intentionally.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${SCREENSHOT_DIR:-$ROOT/docs/assets/screenshots}"
BASE="${RD_SCREENSHOT_BASE:-http://127.0.0.1:18081}"
API_KEY="${REPOSITORY_DETECTIVE_API_KEY:-e2e-acceptance-api-key-not-a-secret}"
GITEA_BASE="${RD_E2E_GITEA_URL:-http://127.0.0.1:13000}"
WIDTH="${RD_SHOT_WIDTH:-1440}"
HEIGHT="${RD_SHOT_HEIGHT:-900}"

mkdir -p "$OUT"

if [[ "$BASE" == *"8081"* && "${RD_ALLOW_PROD_SCREENSHOTS:-}" != "1" ]]; then
  echo "Refusing BASE=$BASE (looks like production). Set RD_SCREENSHOT_BASE to disposable :18081." >&2
  echo "Or set RD_ALLOW_PROD_SCREENSHOTS=1 only if you are certain." >&2
  exit 2
fi

hdr=(-H "X-Repository-Detective-API-Key: ${API_KEY}")

# Discover synthetic ids when possible
FINDING_ID="${RD_SHOT_FINDING_ID:-}"
REPO_ID="${RD_SHOT_REPO_ID:-1}"
if [[ -z "$FINDING_ID" ]]; then
  FINDING_ID="$(curl -fsS "${hdr[@]}" "$BASE/api/v1/findings?limit=1" 2>/dev/null | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d["findings"][0]["id"] if d.get("findings") else "")' 2>/dev/null || true)"
fi
FINDING_ID="${FINDING_ID:-1}"

capture_playwright() {
  local name="$1" url="$2"
  docker run --rm --network host \
    -v "$OUT:/out" \
    mcr.microsoft.com/playwright:v1.55.0-jammy \
    bash -lc "cd /tmp && npm init -y >/dev/null 2>&1 && npm i playwright-core@1.55.0 >/dev/null 2>&1 && node - <<'NODE'
const { chromium } = require('playwright-core');
(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: ${WIDTH}, height: ${HEIGHT} } });
  await page.setExtraHTTPHeaders({ 'X-Repository-Detective-API-Key': process.env.API_KEY || '' });
  await page.goto(process.env.SHOT_URL, { waitUntil: 'networkidle', timeout: 120000 });
  await page.waitForTimeout(1200);
  await page.screenshot({ path: '/out/' + process.env.SHOT_NAME, fullPage: false });
  await browser.close();
})().catch(e => { console.error(e); process.exit(1); });
NODE" \
    -e "SHOT_URL=$url" -e "SHOT_NAME=$name" -e "API_KEY=$API_KEY"
}

# Prefer host chromium if present; else Playwright container (needs working docker create).
BROWSER=""
for c in chromium-browser chromium google-chrome; do
  command -v "$c" >/dev/null 2>&1 && BROWSER="$c" && break
done

shot() {
  local file="$1" path="$2"
  local url
  if [[ "$path" == http* ]]; then
    url="$path"
  else
    url="${BASE}${path}"
  fi
  echo "==> $file <- $url"
  if [[ -n "$BROWSER" ]]; then
    "$BROWSER" --headless --disable-gpu --window-size="${WIDTH},${HEIGHT}" \
      --screenshot="$OUT/$file" "$url" 2>/dev/null || echo "WARN: failed $file"
  else
    if ! capture_playwright "$file" "$url"; then
      echo "WARN: playwright capture failed for $file (docker create may be wedged)" >&2
    fi
  fi
}

shot "01-onboarding-connect.png" "/onboard/"
# Protect stage is client-side; same URL — operator advances manually if needed.
shot "02-onboarding-protect.png" "/onboard/"
shot "03-doctor.png" "/ui/doctor"
shot "04-dashboard.png" "/ui/"
shot "05-finding-evidence.png" "/ui/findings/${FINDING_ID}"
shot "06-policy-evaluation.png" "/ui/repos/${REPO_ID}/settings"
shot "07-pr-compact-summary.png" "${GITEA_BASE}/"
shot "08-privacy-local-only.png" "/ui/health"
shot "09-remediation-plan-preview.png" "/ui/findings/${FINDING_ID}"

echo "Wrote shots under $OUT"
echo "Privacy review: strings *.png | grep -iE 'commsnet|token|192\\.168|password' || true"
