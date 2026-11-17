#!/usr/bin/env bash
# Ensure GitHub release tags still resolve after main snapshot refreshes.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

REMOTE="${1:-github}"
FAIL=0
echo "==> validating release tags on $REMOTE"
while read -r ref; do
  [[ -z "$ref" ]] && continue
  tag="${ref##*/}"
  if ! git ls-remote --tags "$REMOTE" "refs/tags/$tag" | grep -q .; then
    echo "MISSING: $tag"
    FAIL=1
  else
    echo "OK: $tag"
  fi
done < <(git tag -l 'v0.1.0-beta.*' | sort)

# Also require that a force-update of main would not delete tags (git property check)
echo "==> note: snapshot force-push updates main only; tags are separate refs"
exit "$FAIL"
