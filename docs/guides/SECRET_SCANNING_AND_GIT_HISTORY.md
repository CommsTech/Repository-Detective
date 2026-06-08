# Secret scanning and Git history

Repository Detective uses **Gitleaks** for credential detection in two labeled modes:

| Mode | Scanner name | Speed | What it checks |
|------|--------------|-------|----------------|
| Current tree | `gitleaks` | Fast | Files in the scan workspace snapshot |
| Git history | `gitleaks-history` | Slower | Commits in the cloned repository history |

## Configuration

```yaml
enable_gitleaks: true
secret_scan_git_history_enabled: true
secret_scan_history_max_commits: 0          # 0 = full history when cloning
secret_scan_recent_commits_max: 50          # PR/push and pre-install window
secret_scan_history_timeout_seconds: 600
secret_scan_history_report_only_for_preinstall: true
secret_scan_redact: true
```

See **Configure → Secret scanning** in the operator UI.

## When each mode runs

| Scan type | Tree scan | History scan |
|-----------|-----------|--------------|
| Onboarding / scheduled deep (depth ≥ 2) | Yes | Full history (if enabled) |
| PR / push quick (changed files) | Yes | Recent commits only |
| Pre-install audit (deep) | Yes | Recent commits (report-only) |

## Finding labels

History findings include:

- **Commit hash** (when available)
- **File path** and rule ID
- **Redacted** match text (raw secrets are never stored or printed)
- **In current tree** vs **not in current tree**
- **Remediation:** rotate/revoke even if the secret was deleted from HEAD

Reports distinguish:

- `current-tree secret` — active in the scanned snapshot
- `historical secret` — found in Git history (may be deleted on HEAD)
- Test fixtures and placeholders may be downgraded by learning/calibration

## Screenshot

![Scanner status and secret history settings](../assets/screenshots/configure-secret-scan.png)

*Capture: `/ui/configure` secret scanning section.*
