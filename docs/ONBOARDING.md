# Onboarding Web UI (RD-013)

Mental model: **Connect → Select → Protect → Verify → Ready**

| URL | Auth |
|-----|------|
| `http://your-host:8081/onboard/` | Public UI; API calls need operator API key |

## Stages

1. **Connect** — forge URL/token, public URL, webhook secret, privacy mode; live forge test + permission matrix  
2. **Select** — list repos; recommend profile (override allowed; RD-012A requirements unchanged)  
3. **Protect** — profile, Observe/Warn/Enforce (default Observe), privacy egress preview, optional AI, webhook register  
4. **Verify** — shared doctor engine (`POST /api/v1/onboard/verify`)  
5. **Ready** — READY / READY_WITH_LIMITATIONS / NOT_READY + env export + next actions  

Never uses SAFE / SECURE / SECURITY PASSED.

Manual first scan = `FIRST_SCAN_PROVEN` ≠ webhook E2E (RD-017).

## API (`/api/v1/onboard/*`)

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/defaults` | Defaults + stage metadata |
| POST | `/test-gitea` | Forge connectivity |
| POST | `/permissions` | Permission matrix + profile recommend |
| POST | `/repos` | List repositories |
| POST | `/recommend-profile` | Profile hint |
| POST | `/privacy-preview` | LOCAL/EXTERNAL egress disclosure |
| POST | `/test-ai` | Optional AI connectivity |
| POST | `/webhooks` | Register hooks |
| POST | `/verify` | Doctor report for wizard choices |

See also: [DOCTOR.md](DOCTOR.md), [PRIVACY_MODES.md](PRIVACY_MODES.md).
