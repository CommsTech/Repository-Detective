#!/bin/sh
# Deterministic Alpine runtime user + package setup for Repository Detective images.
# Fails hard if apk or user/group creation does not complete (avoids later chown on missing user).
set -eu

RD_UID="${RD_RUNTIME_UID:-1001}"
RD_GID="${RD_RUNTIME_GID:-1001}"
RD_USER="${RD_RUNTIME_USER:-repositorydetective}"
RD_GROUP="${RD_RUNTIME_GROUP:-repositorydetective}"

if [ "$#" -gt 0 ]; then
	if [ ! -f /usr/local/lib/rd/apk-retry.sh ]; then
		echo "missing /usr/local/lib/rd/apk-retry.sh" >&2
		exit 1
	fi
	. /usr/local/lib/rd/apk-retry.sh
	apk_retry "$@"
fi

if ! id -u "$RD_USER" >/dev/null 2>&1; then
	addgroup -g "$RD_GID" -S "$RD_GROUP"
	adduser -u "$RD_UID" -S "$RD_USER" -G "$RD_GROUP"
fi

if ! id -u "$RD_USER" >/dev/null 2>&1; then
	echo "runtime user $RD_USER was not created" >&2
	exit 1
fi
actual_gid="$(id -g "$RD_USER")"
if [ "$actual_gid" != "$RD_GID" ]; then
	echo "runtime user $RD_USER has gid $actual_gid, expected $RD_GID" >&2
	exit 1
fi
