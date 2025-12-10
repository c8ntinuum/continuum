#!/bin/sh
# Fix up Mach-O binary + bundled .dylib directory (rpaths + ids).
#
# Usage:
#   fixup_bundle.sh <binary> <libdir> <binary_rpath>
#
set -eu

if [ "$#" -ne 3 ]; then
  echo "usage: $0 <binary> <libdir> <binary_rpath>" >&2
  exit 2
fi

BIN="$1"
LIBDIR="$2"
BIN_RPATH="$3"

CODESIGN_MODE="${DARWIN_CODESIGN:-auto}"
DYLIB_RPATH="${DARWIN_DYLIB_RPATH:-@loader_path}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "fixup_bundle: missing tool '$1'; skipping fixup" >&2
    exit 0
  fi
}

need otool
need install_name_tool
need awk
need grep

[ -f "$BIN" ] || { echo "ERROR: binary not found: $BIN" >&2; exit 1; }
[ -d "$LIBDIR" ] || { echo "ERROR: libdir not found: $LIBDIR" >&2; exit 1; }

list_rpaths() {
  otool -l "$1" | awk '
    $1=="cmd" && $2=="LC_RPATH" {inr=1}
    inr && $1=="path" {print $2; inr=0}
  '
}

ensure_rpath() {
  file="$1"
  rpath="$2"
  if list_rpaths "$file" | grep -Fx "$rpath" >/dev/null 2>&1; then
    return 0
  fi
  install_name_tool -add_rpath "$rpath" "$file"
}

patch_deps_to_rpath() {
  file="$1"
  libdir="$2"

  otool -L "$file" | awk 'NR>1 {print $1}' | while IFS= read -r dep; do
    [ -n "$dep" ] || continue
    case "$dep" in
      /usr/lib/*|/System/Library/*) continue ;;
      @rpath/*|@loader_path/*|@executable_path/*) continue ;;
    esac
    depbase=$(basename "$dep")
    if [ -f "$libdir/$depbase" ]; then
      install_name_tool -change "$dep" "@rpath/$depbase" "$file"
    fi
  done
}

echo "fixup_bundle: BIN=$BIN" >&2
echo "fixup_bundle: LIBDIR=$LIBDIR" >&2
echo "fixup_bundle: BIN_RPATH=$BIN_RPATH DYLIB_RPATH=$DYLIB_RPATH" >&2

ensure_rpath "$BIN" "$BIN_RPATH"

# Normalize dylib IDs and give each dylib @loader_path rpath for its own deps
for f in "$LIBDIR"/*.dylib; do
  [ -f "$f" ] || continue
  base=$(basename "$f")
  install_name_tool -id "@rpath/$base" "$f"
  ensure_rpath "$f" "$DYLIB_RPATH"
done

# Patch dependencies inside dylibs
for f in "$LIBDIR"/*.dylib; do
  [ -f "$f" ] || continue
  patch_deps_to_rpath "$f" "$LIBDIR"
done

# Patch binary deps
patch_deps_to_rpath "$BIN" "$LIBDIR"

# Optional codesign
if [ "$CODESIGN_MODE" != "false" ] && command -v codesign >/dev/null 2>&1; then
  set -- "$LIBDIR"/*.dylib
  if [ -f "$1" ]; then
    echo "fixup_bundle: codesign dylibs" >&2
    codesign --force --sign - "$LIBDIR"/*.dylib || {
      [ "$CODESIGN_MODE" = "true" ] && exit 1 || true
    }
  fi
  echo "fixup_bundle: codesign binary" >&2
  codesign --force --sign - "$BIN" || {
    [ "$CODESIGN_MODE" = "true" ] && exit 1 || true
  }
fi

echo "fixup_bundle: done" >&2
exit 0
