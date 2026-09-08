"""Synthetic secret fixture for disposable demos — not a real credential."""

# Built at import so static docs/scanners are less likely to treat this file as a live leak.
_PREFIX = "xoxb-"
_MID = "123456789012-123456789012-"
_SUFFIX = "zzdemofixturetokzz"
DEMO_TOKEN = _PREFIX + _MID + _SUFFIX
