#!/bin/sh

set -eu

if [ "${DRAFTSPEC_BIN:-}" != "" ]; then
  if command -v "$DRAFTSPEC_BIN" >/dev/null 2>&1; then
    exec "$DRAFTSPEC_BIN" "$@"
  fi
  if [ -x "$DRAFTSPEC_BIN" ]; then
    exec "$DRAFTSPEC_BIN" "$@"
  fi
  echo "ERROR: DRAFTSPEC_BIN is set but could not be resolved: $DRAFTSPEC_BIN" >&2
  echo "Set DRAFTSPEC_BIN to an executable path or command name, or add draftspec to PATH." >&2
  exit 1
fi

if command -v draftspec >/dev/null 2>&1; then
  exec draftspec "$@"
fi

echo "ERROR: draftspec CLI not found." >&2
echo "Set DRAFTSPEC_BIN to an executable path or add draftspec to PATH." >&2
exit 1
