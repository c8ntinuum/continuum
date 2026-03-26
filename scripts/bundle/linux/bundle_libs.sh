#!/bin/sh
set -eu

if [ "$#" -ne 2 ]; then
  echo "usage: $0 <elf-file> <dest-libdir>" >&2
  exit 2
fi

elf="$1"
dest="$2"
bundle_list="${ROCKSDB_BUNDLE_LIBS:-}"

if [ -z "$bundle_list" ]; then
  echo "bundle_linux: ROCKSDB_BUNDLE_LIBS empty; nothing to do" >&2
  exit 0
fi

need() { command -v "$1" >/dev/null 2>&1 || { echo "ERROR: missing tool: $1" >&2; exit 1; }; }
need ldd
need awk
need sed
need grep
need cp
need mkdir

[ -f "$elf" ] || { echo "ERROR: ELF file not found: $elf" >&2; exit 1; }
mkdir -p "$dest"

alts=$(printf '%s\n' $bundle_list \
  | sed -e 's/[.[\*^$()+?{|]/\\&/g' \
  | awk 'BEGIN{ORS=""; first=1}{if(!first)printf "|"; printf "%s",$0; first=0}')

pattern="^lib(${alts})\\.so(\\.[0-9].*)?$"

copy_one() {
  name="$1"
  path="$2"
  out="$dest/$name"
  if [ -f "$out" ]; then return 0; fi
  if [ ! -f "$path" ]; then
    echo "  !! missing file for $name: $path" >&2
    return 0
  fi
  echo "  -> $name"
  cp -L "$path" "$out"
  changed=1
}

scan_elf() {
  file="$1"
  ldd "$file" 2>/dev/null | awk '
    /not a dynamic executable/ { exit 0 }
    /statically linked/       { exit 0 }
    $2=="=>" {
      name=$1
      if ($3=="not") { next }
      path=$3
      if (path ~ /^\//) { print name "\t" path }
    }
  ' | while IFS="$(printf '\t')" read -r name path; do
    [ -n "$name" ] || continue
    printf '%s\n' "$name" | grep -E "$pattern" >/dev/null 2>&1 || continue
    copy_one "$name" "$path"
  done

  ldd "$file" 2>/dev/null | awk '$2=="=>" && $3=="not" {print $1}' | while IFS= read -r name; do
    [ -n "$name" ] || continue
    printf '%s\n' "$name" | grep -E "$pattern" >/dev/null 2>&1 || continue
    echo "  !! $name => not found" >&2
  done
}

echo "bundle_linux: scanning $elf (allowlist: $bundle_list)"
changed=1
scan_elf "$elf"

while [ "$changed" -eq 1 ]; do
  changed=0
  for f in "$dest"/*.so*; do
    [ -e "$f" ] || continue
    scan_elf "$f"
  done
done
