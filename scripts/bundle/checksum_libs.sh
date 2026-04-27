#!/bin/sh
# Record SHA256 checksums of all files in a library directory.
# Usage: checksum_libs.sh <libdir> [<output-file>]
# If output-file is omitted, prints to stdout.
set -eu

LIBDIR="$1"
OUTFILE="${2:-}"

[ -d "$LIBDIR" ] || { echo "ERROR: not a directory: $LIBDIR" >&2; exit 1; }

if command -v sha256sum >/dev/null 2>&1; then
  HASHER="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  HASHER="shasum -a 256"
else
  echo "ERROR: neither sha256sum nor shasum found" >&2
  exit 1
fi

cd "$LIBDIR"
files=$(find . -maxdepth 1 -type f | LC_ALL=C sort)
if [ -z "$files" ]; then
  echo "WARNING: no files found in $LIBDIR" >&2
  exit 0
fi

if [ -n "$OUTFILE" ]; then
  echo "$files" | xargs $HASHER > "$OUTFILE"
  echo "Checksums written to $OUTFILE" >&2
else
  echo "$files" | xargs $HASHER
fi
