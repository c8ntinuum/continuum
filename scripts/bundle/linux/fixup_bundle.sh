#!/bin/sh
set -eu

if [ "$#" -ne 3 ]; then
  echo "usage: $0 <binary> <libdir> <binary_rpath>" >&2
  exit 2
fi

BIN="$1"
LIBDIR="$2"
BIN_RPATH="$3"

PATCHELF="${LINUX_PATCHELF:-patchelf}"
REQUIRED="${LINUX_PATCHELF_REQUIRED:-false}"
LIB_RPATH="${LINUX_LIB_RPATH:-\$ORIGIN}"

[ -f "$BIN" ] || { echo "ERROR: binary not found: $BIN" >&2; exit 1; }
[ -d "$LIBDIR" ] || { echo "ERROR: libdir not found: $LIBDIR" >&2; exit 1; }

if ! command -v "$PATCHELF" >/dev/null 2>&1; then
  msg="patchelf not found; cannot set RPATH (may need LD_LIBRARY_PATH)"
  if [ "$REQUIRED" = "true" ]; then
    echo "ERROR: $msg" >&2
    exit 1
  fi
  echo "ℹ️  fixup_bundle: $msg" >&2
  exit 0
fi

echo "fixup_bundle: setting RPATH on binary -> $BIN_RPATH"
"$PATCHELF" --force-rpath --set-rpath "$BIN_RPATH" "$BIN"

for f in "$LIBDIR"/*.so*; do
  [ -e "$f" ] || continue
  echo "fixup_bundle: setting RPATH on $(basename "$f") -> $LIB_RPATH"
  "$PATCHELF" --force-rpath --set-rpath "$LIB_RPATH" "$f" || true
done
