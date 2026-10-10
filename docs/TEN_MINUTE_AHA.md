# Ten-minute aha — clone to useful finding

**Target:** useful finding on screen in **under 10 minutes**, not merely “software running.”

Stopwatch optional. Feature freeze: no new scanners required for this path.

## Canonical path

```text
docker compose up -d
        ↓
open browser → /onboard
        ↓
connect Gitea / Forgejo
        ↓
select repositories
        ↓
Run First Assessment
        ↓
open a finding → Finding Detail 2.0 brief
        ↓
dashboard Noise Reduction card shows what calibration removed
        ↓
optional: remediation plan / Gitea issue / PR (when enabled)
```

## Timed checklist

| Minute | Action |
|-------:|--------|
| 0–2 | `cp .env.example .env`, set `REPOSITORY_DETECTIVE_API_KEY` + Gitea URL/token/webhook secret |
| 2–4 | `docker compose pull && docker compose up -d` → `curl -s http://127.0.0.1:8081/health` |
| 4–6 | Open `/onboard` — Connect → Select → Protect → Verify |
| 6–8 | Push or scan the [vulnerable demo fixture](../examples/vulnerable-demo/README.md) (or any connected repo) |
| 8–10 | Open **Findings** → one finding → answer in 30s: what / why / what changes / can RD fix / how verify. Then check **Dashboard → Noise Reduction** |

## Deliberately vulnerable demo

See [`examples/vulnerable-demo/`](../examples/vulnerable-demo/) — pinned old `cryptography` + synthetic secret fixture for disposable forges only.

## Trust surface (while you wait for the scan)

- Pin image by tag **and** digest when possible ([VERIFY_RELEASE.md](VERIFY_RELEASE.md))
- SBOM + checksums for published digests
- Operator UI fonts are **self-hosted / system** — no Google Fonts CDN
- Canonical source: Gitea; GitHub is a public mirror ([RELEASE_MIRROR.md](RELEASE_MIRROR.md))

## Definition of done

An unfamiliar operator can answer within **30 seconds** on a finding page:

**What happened? Why does it matter? What changes? What might break? Can RD fix it? How will RD verify it?**

If not, Finding Detail is not finished — file against RD-PRODUCT-001.
