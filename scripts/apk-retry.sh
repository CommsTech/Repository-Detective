#!/bin/sh
# Retry apk add when Alpine CDN mirrors return transient errors.
# Safe to source: defines apk_retry(). When executed as a script, runs apk_retry "$@".
set -eu

apk_retry() {
	tries="${APK_RETRY_COUNT:-5}"
	delay="${APK_RETRY_DELAY_SEC:-8}"

	if [ "$#" -eq 0 ]; then
		echo "apk_retry: no packages specified" >&2
		return 1
	fi

	while [ "$tries" -gt 0 ]; do
		if apk add --no-cache "$@"; then
			return 0
		fi
		tries=$((tries - 1))
		if [ "$tries" -le 0 ]; then
			break
		fi
		echo "apk add failed; retrying in ${delay}s ($tries left)..." >&2
		sleep "$delay"
	done

	echo "apk add failed after retries: $*" >&2
	return 1
}

# When executed directly (not sourced), install the given packages.
case "${0##*/}" in
apk-retry.sh)
	apk_retry "$@"
	;;
esac
