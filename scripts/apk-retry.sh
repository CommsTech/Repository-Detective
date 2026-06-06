#!/bin/sh
# Retry apk add when Alpine CDN mirrors return transient errors.
set -eu

tries="${APK_RETRY_COUNT:-5}"
delay="${APK_RETRY_DELAY_SEC:-8}"

while [ "$tries" -gt 0 ]; do
	if apk add --no-cache "$@"; then
		exit 0
	fi
	tries=$((tries - 1))
	if [ "$tries" -le 0 ]; then
		break
	fi
	echo "apk add failed; retrying in ${delay}s ($tries left)..." >&2
	sleep "$delay"
done

echo "apk add failed after retries: $*" >&2
exit 1
