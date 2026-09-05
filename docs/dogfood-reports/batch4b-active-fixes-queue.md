# Batch 4b active fixes queue

Generated: 2026-06-02  
Scan: `db2d7061eaac8eb0`

| Issue | Fingerprint | Scanner | Rule | Sev | Conf | Path | Line | Planned fix | Test |
|------:|-------------|---------|------|-----|------|------|-----:|-------------|------|
| #345 | rd-88f255b120b097e1 | trivy | TRIVY-MIS-DS011 | critical | — | Dockerfile | 83 | Split multi-source COPY into one dest per COPY | docker-build-verify |
| #53 | rd-9a6eaeee6abc7aa7 | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | Makefile | 175 | Use 127.0.0.1 in pprof help echo | static analyzer unit |
| #66 | rd-b67022bb59a95c9b | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | deploy.ps1 | 52 | Use 127.0.0.1 in health help | static analyzer unit |
| #143 | rd-bc82928a9cc02518 | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | preinstall/url.go | 16 | FP: blocked-host suffix catalog | url_test |
| #144 | rd-c073c4b84ab723fb | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | preinstall/url.go | 31 | FP: blocked-host map | url_test |
| #145 | rd-10d09a1086712459 | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | preinstall/url.go | 75 | FP: loopback rejection message | url_test |
| #280 | rd-794b38b4d1a0a43a | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | patcher/git.go | 137 | Use noreply.invalid git identity email | patcher tests |
| #296 | rd-54c4fba028826514 | static | REL-INTERNAL-INFRA-REF | medium | 0.75 | deploy/nginx-repository-detective.conf.example | 6 | Skip *.example in static analysis | static skip test |
| #321 | rd-c277e8b2a3ae0431 | gosec | G201 | medium | — | store/findings_batch_sqlite.go | 21 | Build IN clause without fmt.Sprintf | store tests |
| #324 | rd-f0e175991662ee4e | gosec | G203 | medium | — | ui/ui_helpers.go | 50 | json.Valid gate before template.JS | ui tests |
| #332 | rd-3e397af3dbef8964 | gosec | G304 | medium | — | scanners/archive_extract.go | 92 | pathWithinRoot guard before OpenFile | workspace_test |
