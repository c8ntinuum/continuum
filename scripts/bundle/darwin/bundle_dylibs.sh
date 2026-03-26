#!/bin/sh
# Bundle allowlisted .dylib dependencies for a Mach-O binary.
#
# Usage:
#   bundle_dylibs.sh <macho-file> <dest-libdir>
#
# Env:
#   ROCKSDB_BUNDLE_LIBS="rocksdb snappy zstd lz4 bz2 z gflags numa ..."
#
set -eu

if [ "$#" -ne 2 ]; then
  echo "usage: $0 <macho-file> <dest-libdir>" >&2
  exit 2
fi

macho="$1"
dest="$2"

bundle_list="${ROCKSDB_BUNDLE_LIBS:-}"
if [ -z "$bundle_list" ]; then
  echo "bundle_dylibs: ROCKSDB_BUNDLE_LIBS is empty; nothing to bundle" >&2
  exit 0
fi

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "bundle_dylibs: missing tool '$1'; skipping bundling" >&2
    exit 0
  fi
}

need otool
need awk
need grep
need sed

if [ ! -f "$macho" ]; then
  echo "bundle_dylibs: macho file not found: $macho" >&2
  exit 1
fi

mkdir -p "$dest"

# Build whitelist regex alternation from ROCKSDB_BUNDLE_LIBS (escape regex chars).
alts=$(printf '%s\n' $bundle_list \
  | sed -e 's/[.[\*^$()+?{|]/\\&/g' \
  | awk 'BEGIN{ORS=""; first=1}{if(!first)printf "|"; printf "%s",$0; first=0}')

# Allow:
#   lib<name>.dylib
#   lib<name>.<digits...>.dylib  (versioned)
pattern="^lib(${alts})(\\.[0-9].*)?\\.dylib$"

search_dirs=""
add_dir() {
  d="$1"
  [ -n "$d" ] || return 0
  [ -d "$d" ] || return 0
  case " $search_dirs " in
    *" $d "*) ;; # already present
    *) search_dirs="$search_dirs $d" ;;
  esac
}

# Hints from environment (these come from your Makefile)
[ -n "${ROCKSDB_LIB_DIR:-}" ] && add_dir "$ROCKSDB_LIB_DIR"
[ -n "${ROCKSDB_PREFIX:-}" ] && { add_dir "$ROCKSDB_PREFIX/lib"; add_dir "$ROCKSDB_PREFIX/lib64"; }
[ -n "${SNAPPY_PREFIX:-}"  ] && add_dir "$SNAPPY_PREFIX/lib"
[ -n "${ZSTD_PREFIX:-}"    ] && add_dir "$ZSTD_PREFIX/lib"
[ -n "${LZ4_PREFIX:-}"     ] && add_dir "$LZ4_PREFIX/lib"
[ -n "${BZ2_PREFIX:-}"     ] && add_dir "$BZ2_PREFIX/lib"
[ -n "${ZLIB_PREFIX:-}"    ] && add_dir "$ZLIB_PREFIX/lib"
[ -n "${GFLAGS_PREFIX:-}"  ] && add_dir "$GFLAGS_PREFIX/lib"

# Common fallback locations
add_dir "/usr/local/lib"
add_dir "/opt/homebrew/lib"

resolve_dep() {
  dep="$1"
  if [ -f "$dep" ]; then
    printf '%s\n' "$dep"
    return 0
  fi
  base=$(basename "$dep")
  for d in $search_dirs; do
    if [ -f "$d/$base" ]; then
      printf '%s\n' "$d/$base"
      return 0
    fi
  done
  return 1
}

changed=0
copy_one() {
  src="$1"
  base=$(basename "$src")
  dst="$dest/$base"
  if [ ! -f "$dst" ]; then
    echo "  -> $base" >&2
    cp "$src" "$dst"
    changed=1
  fi
}

scan_macho() {
  file="$1"
  tmp=$(mktemp -t otooldeps.XXXXXX)
  # NR>1: skip the first line (the file itself)
  otool -L "$file" | awk 'NR>1 {print $1}' > "$tmp"

  while IFS= read -r dep; do
    [ -n "$dep" ] || continue
    case "$dep" in
      /usr/lib/*|/System/Library/*) continue ;;
    esac

    base=$(basename "$dep")
    printf '%s\n' "$base" | grep -E "$pattern" >/dev/null 2>&1 || continue

    src=$(resolve_dep "$dep" 2>/dev/null || true)
    if [ -n "$src" ]; then
      copy_one "$src"
    else
      echo "  !! cannot resolve $dep (searched:$search_dirs)" >&2
    fi
  done < "$tmp"

  rm -f "$tmp"
}

echo "bundle_dylibs: scanning $macho (allowlist: $bundle_list)" >&2
changed=0
scan_macho "$macho"

# Recurse into copied dylibs until no new libs appear
while [ "$changed" -eq 1 ]; do
  changed=0
  for f in "$dest"/*.dylib; do
    [ -f "$f" ] || continue
    scan_macho "$f"
  done
done

echo "bundle_dylibs: done" >&2
exit 0
