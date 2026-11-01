# Vulnerable demo fixture (disposable)

**Use only on disposable Gitea/Forgejo.** Do not push to production forges.

This tiny tree is designed so Repository Detective’s first assessment produces an
actionable dependency finding (and optionally a synthetic secret) within the
[ten-minute aha](../../docs/TEN_MINUTE_AHA.md) path.

## Contents

| File | Purpose |
|------|---------|
| `requirements.txt` | Pins `cryptography==41.0.3` (known vulnerable range for demos) |
| `app.py` | Trivial import so the dependency is “in use” |
| `demo_leak_fixture.py` | Builds a **synthetic** Slack-bot-shaped string at runtime (not a real credential) |

## Use

```bash
# Create empty repo on disposable Gitea, then:
cp -r examples/vulnerable-demo /tmp/rd-demo && cd /tmp/rd-demo
git init && git add . && git commit -m "demo: vulnerable fixture"
# add remote, push, connect in Repository Detective onboard, scan
```

Or copy `requirements.txt` into an existing demo repo and push.

## After the scan

Open the cryptography finding. Finding Detail 2.0 should show:

- Installed vs minimum fixed version
- Why you should care (grouped advisories)
- Recommended action line
- What Repository Detective can / cannot do
- Evidence collapsed under **Evidence & scanner details**
