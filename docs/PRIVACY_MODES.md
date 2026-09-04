# Privacy operating modes (RD-007)

## Modes

| Mode | Env / config | Meaning |
|------|--------------|---------|
| `local_only` | `PRIVACY_MODE=local_only` | Fail closed: no intentional **EXTERNAL AI** or **EXTERNAL notification webhook** egress |
| `hybrid` | **default** | Operator-approved external integrations may receive documented subsets of context (disclosed in Operator Status) |
| `external_ai_enabled` | | External AI is intentionally configured and disclosed |

Environment: `REPOSITORY_DETECTIVE_PRIVACY_MODE` / `privacy_mode` (default **`hybrid`** so existing installs are not silently flipped).

## What LOCAL_ONLY guarantees

**Guaranteed (enforced today — CODE_PRESENT + WIRED + UNIT_TESTED):**

- External AI provider configs (`openai`, remote OpenAI-compatible URL classified EXTERNAL) are rejected when LLM auditors need AI.
- OpenClaw / advisory AI that would invoke EXTERNAL endpoints is blocked (startup fail when enabled/invokable).
- EXTERNAL Slack/Discord/generic webhooks and Telegram are disabled under `local_only`.
- Operator Status shows privacy mode, AI endpoint classification, and egress posture.

**Not guaranteed by LOCAL_ONLY alone:**

- **Forge traffic** (Gitea/GitHub): issue bodies, PR summaries, and webhook callbacks still go to the configured forge URL. If that forge is a public cloud host, finding/snippet content leaves the RD host by design of integration.
- **DNS rebinding**: hostname classification is checked when evaluating config / connection setup; residual TOCTOU risk is PARTIAL (see SECURITY_MODEL.md).

**LOCAL_ONLY is not “air-gapped.”** It is “no external AI / notification content egress by default.”

## Outbound data-flow inventory (code paths)

| Path | Content types | LOCAL_ONLY impact |
|------|---------------|-------------------|
| Gitea/Forge issues + PR summaries | findings, snippets, metadata | Not blocked (forge is the product sink) |
| AI providers / Ollama / OpenAI-compat | prompts, snippets, findings | **Blocked** if endpoint EXTERNAL |
| OpenClaw / advisory AI | findings ± snippets | **Blocked** if EXTERNAL and invokable |
| Notifications (Slack/Discord/webhook/Telegram) | finding summaries | **Channels disabled** if EXTERNAL |
| Runner job payloads | clone URL + metadata | Internal control plane |
| Telemetry / crash reporters | none shipped by default | N/A |
| Logs | may include paths/metadata; secrets redacted | See RD-009 |

## Endpoint locality

Classification uses resolved IPs (loopback, RFC1918, link-local, ULA, CGNAT) — **not** provider name alone.
`ollama` pointing at a public IP is EXTERNAL.
`openai` / `anthropic` / `openrouter` cloud providers are always EXTERNAL under LOCAL_ONLY.
Hostnames require DNS resolution; failure → UNKNOWN → blocked under LOCAL_ONLY.

## UI disclosure

Operator Status / health surface:

- Privacy mode
- Code-content AI egress: blocked / limited / allowed
- AI provider + endpoint + LOCAL/EXTERNAL classification
