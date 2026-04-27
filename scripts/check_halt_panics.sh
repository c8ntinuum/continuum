#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REVIEW_FILE="$ROOT_DIR/audit/HALT_PANIC_REVIEW.md"

cd "$ROOT_DIR"

if ! command -v rg >/dev/null 2>&1; then
  echo "check_halt_panics: ripgrep (rg) is required" >&2
  exit 1
fi

if [[ ! -f "$REVIEW_FILE" ]]; then
  echo "check_halt_panics: missing reviewed panic list: $REVIEW_FILE" >&2
  exit 1
fi

mapfile -t scoped_files < <(
  rg --files ante mempool x precompiles \
    | rg '(^ante/.*\.go$)|(^mempool/.*\.go$)|(^x/[^/]+/keeper/.*\.go$)|(^precompiles/[^/]+/tx\.go$)' \
    | rg -v '(_test\.go$|/mocks/)'
)

violations=()
reviewed_files=()

for file in "${scoped_files[@]}"; do
  prev_line=""
  while IFS=$'\t' read -r line_no text; do
    if [[ $text =~ (^|[^[:alnum:]_])panic\( ]]; then
      if [[ $text == *"nolint:halt"* || $prev_line == *"nolint:halt"* ]]; then
        reviewed_files+=("$file")
      else
        violations+=("$file:$line_no:$text")
      fi
    fi
    prev_line="$text"
  done < <(nl -ba -w1 -s $'\t' "$file")
done

if ((${#violations[@]} > 0)); then
  echo "Unreviewed panic() calls found in halt-sensitive paths:" >&2
  printf '  %s\n' "${violations[@]}" >&2
  exit 1
fi

if ((${#reviewed_files[@]} > 0)); then
  mapfile -t unique_reviewed_files < <(printf '%s\n' "${reviewed_files[@]}" | sort -u)
  missing_reviews=()
  for file in "${unique_reviewed_files[@]}"; do
    if ! rg -F -q "$file" "$REVIEW_FILE"; then
      missing_reviews+=("$file")
    fi
  done

  if ((${#missing_reviews[@]} > 0)); then
    echo "Tagged halt panics missing from $REVIEW_FILE:" >&2
    printf '  %s\n' "${missing_reviews[@]}" >&2
    exit 1
  fi
fi

echo "halt panic policy check passed"
