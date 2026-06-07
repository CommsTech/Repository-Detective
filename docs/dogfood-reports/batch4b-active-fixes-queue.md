# Batch 4b active fixes queue

Generated: 2026-06-02  
Scan: `db2d7061eaac8eb0`

| Issue | Fingerprint | Scanner | Rule | Sev | Conf | Path | Line | Planned fix | Test |
|------:|-------------|---------|------|-----|------|------|-----:|-------------|------|
| #345 | bugbot-88f255b120b097e1 | trivy | TRIVY-MIS-DS011 | critical | — | Dockerfile | 83 | Split multi-source COPY into one dest per COPY | docker-build-verify |
| #53 | bugbot-9a6eaeee6abc7aa7 | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | Makefile | 175 | Use 127.0.0.1 in pprof help echo | static analyzer unit |
| #66 | bugbot-b67022bb59a95c9b | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | deploy.ps1 | 52 | Use 127.0.0.1 in health help | static analyzer unit |
| #143 | bugbot-bc82928a9cc02518 | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | preinstall/url.go | 16 | FP: blocked-host suffix catalog | url_test |
| #144 | bugbot-c073c4b84ab723fb | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | preinstall/url.go | 31 | FP: blocked-host map | url_test |
| #145 | bugbot-10d09a1086712459 | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | preinstall/url.go | 75 | FP: loopback rejection message | url_test |
| #280 | bugbot-794b38b4d1a0a43a | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | patcher/git.go | 137 | Use noreply.invalid git identity email | patcher tests |
| #296 | bugbot-54c4fba028826514 | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | deploy/nginx-bugbot.conf.example | 6 | Skip *.example in static analysis | static skip test |
| #321 | bugbot-c277e8b2a3ae0431 | gosec | G201 | medium | — | store/findings_batch_sqlite.go | 21 | Build IN clause without fmt.Sprintf | store tests |
| #324 | bugbot-f0e175991662ee4e | gosec | G203 | medium | — | ui/ui_helpers.go | 50 | json.Valid gate before template.JS | ui tests |
| #332 | bugbot-3e397af3dbef8964 | gosec | G304 | medium | — | scanners/archive_extract.go | 92 | pathWithinRoot guard before OpenFile | workspace_test |
