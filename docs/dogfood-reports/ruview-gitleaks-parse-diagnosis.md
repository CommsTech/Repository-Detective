# RuView gitleaks `parse_failed` diagnosis

**Date:** 2026-06-04 (UTC)  
**Context:** RuView pre-install audit (`dae05e0c-4c24-441e-9c05-c8ce5db4cbe0`) recorded gitleaks as `parse_failed`.

---

## Command used by Repository Detective (before fix)

```bash
gitleaks dir <workspace> \
  --report-format json \
  --report-path - \
  --no-banner \
  --redact
```

(`scanners/gitleaks.go` — pre-fix)

---

## Environment

| Item | Value |
|------|--------|
| gitleaks version | **8.21.2** |
| Runtime | `repository-detective` container |
| Test repo | https://github.com/ruvnet/RuView (shallow clone) |

---

## Observed behavior (gitleaks 8.21.2)

| Check | Result |
|-------|--------|
| `--report-path -` (stdout) | **Empty stdout** (0 bytes) |
| Exit code with leaks | **1** (default when leaks found) |
| stderr | Colored log lines only (`INF scan completed`, `WRN leaks found: 10`) |
| `--report-path /tmp/report.json` | **Valid JSON array** (~6 KB, 10 findings) |
| Secrets in report | **Redacted** (`--redact` active; no raw values recorded here) |

### Safe stderr sample (first/last lines)

```text
INF scan completed in 11.5s
WRN leaks found: 10
```

(ANSI color codes stripped in logs above.)

### JSON shape (file report)

- Format: **JSON array** `[{ ... }, ...]`
- Count: **10** findings
- Example rule class: `generic-api-key` (rule IDs only; values redacted)

---

## Root cause

1. **Wrong output sink:** In gitleaks 8.x, `--report-path -` does **not** emit the JSON report on stdout. The report is written only when `--report-path` points to a real file.
2. **Parser input:** Repository Detective merged stdout+stderr and called `extractJSONArray`. With empty stdout, stderr contained ANSI sequences like `\x1b[90m` — the `[` in color codes could confuse bracket-based extraction. Parser returned **“no JSON array in output”** → `parse_failed`.
3. **Exit code 1 with leaks:** Nonzero exit is **expected** when leaks exist; parser must still read the report file.

---

## Fix (implemented)

- Write report to a **temp file** (`gitleaks-report-*.json`).
- Add `--no-color` and `--log-level error` to reduce stderr noise.
- Parse **report file first**; fall back to combined command output only if the file is empty (unit tests).
- Treat **nonzero exit + parseable report** as `found`.

---

## Verification commands (operator)

```bash
gitleaks version
gitleaks dir ./repo --report-format json --report-path /tmp/gitleaks.json \
  --no-banner --no-color --redact --log-level error
# exit 1 if leaks; inspect /tmp/gitleaks.json (redacted JSON array)
```

Do not paste report contents into tickets if they may contain operator-local data.

---

## Related

- Code: `scanners/gitleaks.go`, `scanners/gitleaks_test.go`
- Follow-up: RuView audit rerun (`ruview-preinstall-audit-rerun.md`)
