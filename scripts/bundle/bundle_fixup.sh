#!/bin/sh
set -eu

if [ "$#" -ne 4 ]; then
  echo "usage: $0 <binary> <libdir> <darwin_rpath> <linux_rpath>" >&2
  exit 2
fi

BIN="$1"
LIBDIR="$2"
DARWIN_RPATH="$3"
LINUX_RPATH="$4"

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)

[ -f "$BIN" ] || { echo "ERROR: binary not found: $BIN" >&2; exit 1; }
mkdir -p "$LIBDIR"

OS="$(uname -s)"

echo "Bundling for '$OS'" >&2
echo "ROOT_DIR:   '$ROOT_DIR'" >&2
echo "SCRIPT_DIR: '$SCRIPT_DIR'" >&2

if [ "$OS" = "Darwin" ]; then
  echo "Darwin: bundling + fixup" >&2

  if [ "${ROCKSDB_ENABLED:-false}" = "true" ]; then
    echo "  -> bundle_dylibs.sh" >&2
    "$SCRIPT_DIR/darwin/bundle_dylibs.sh" "$BIN" "$LIBDIR"
  fi

  echo "  -> fixup_bundle.sh" >&2
  "$SCRIPT_DIR/darwin/fixup_bundle.sh" "$BIN" "$LIBDIR" "$DARWIN_RPATH"

elif [ "$OS" = "Linux" ]; then
  echo "Linux: bundling + fixup" >&2

  if [ "${ROCKSDB_ENABLED:-false}" = "true" ]; then
    echo "  -> bundle_libs.sh" >&2
    "$SCRIPT_DIR/linux/bundle_libs.sh" "$BIN" "$LIBDIR"
  fi

  echo "  -> fixup_bundle.sh" >&2
  "$SCRIPT_DIR/linux/fixup_bundle.sh" "$BIN" "$LIBDIR" "$LINUX_RPATH"

else
  echo "bundle_fixup: unsupported OS '$OS' (skipping)" >&2
fi