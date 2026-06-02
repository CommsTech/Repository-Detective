# Homelab: AI and Qdrant unreachable at container startup

**Priority:** P2  
**Type:** operations / infrastructure  
**Component:** deployment, networking

## Summary

When `BUGBOT_QDRANT_ENABLED=true` or LLM auditors are enabled, Repository Detective probes OpenClaw (`BUGBOT_AI_BASE_URL`) and Qdrant (`BUGBOT_QDRANT_URL`) during startup. If those hosts are down, firewalled, or slow, startup logs show connection timeouts and semantic dedup is disabled.

Deterministic scans (`standard_deterministic`) continue without AI.

## Expected

- OpenClaw and Qdrant reachable from the Docker host network within `startup_check_timeout` (default 10s), **or**
- Set `BUGBOT_QDRANT_ENABLED=false` when semantic dedup is not required, **or**
- Keep `BUGBOT_SKIP_STARTUP_CHECKS=true` (warnings are logged at debug when skipped).

## Verification

```bash
docker exec repository-detective wget -q -O- --timeout=5 http://192.168.255.11:6333/collections 2>&1 | head
curl -sk -m 5 "${BUGBOT_AI_BASE_URL}/models" -H "Authorization: Bearer ${BUGBOT_AI_API_KEY}"
```

## Not a product defect

This tracks homelab connectivity and operator configuration, not application logic.
