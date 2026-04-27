#!/usr/bin/env bash
set -euo pipefail
shopt -s nocasematch

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REVIEW_FILE="$ROOT_DIR/audit/B5_RISK_MARKER_REVIEW.md"

cd "$ROOT_DIR"

if ! command -v rg >/dev/null 2>&1; then
  echo "check_b5_risk_markers: ripgrep (rg) is required" >&2
  exit 1
fi

if [[ ! -f "$REVIEW_FILE" ]]; then
  echo "check_b5_risk_markers: missing reviewed site list: $REVIEW_FILE" >&2
  exit 1
fi

mapfile -t scoped_files < <(
  rg --files \
    | rg '\.go$' \
    | rg -v '(^audit/|^tests/|^testutil/|_test\.go$|\.pb\.go$|\.pulsar\.go$|/mocks/|/api/|^precompiles/frost/bytemare-stable/)'
)

violations=()
review_refs=()

func_re='^[[:space:]]*func[[:space:]]*(\([^)]*\)[[:space:]]*)?([A-Za-z_][A-Za-z0-9_]*)\b'
name_re='(repair|fixup|salvage|rescue|reconcile|sanitize)|Unsafe[A-Z][A-Za-z0-9_]*'
comment_re='//.*\b(unsafe|best-effort|should never|cannot happen|workaround|HACK|XXX|FIXME)\b'

for file in "${scoped_files[@]}"; do
  prev_line_1=""
  prev_line_2=""
  prev_line_3=""
  prev_line_4=""
  while IFS=$'\t' read -r line_no text; do
    symbol=""
    if [[ $text =~ $func_re ]]; then
      symbol="${BASH_REMATCH[2]}"
      if [[ $symbol =~ $name_re ]]; then
        if [[ $text == *"audit:B5"* || $prev_line_1 == *"audit:B5"* || $prev_line_2 == *"audit:B5"* || $prev_line_3 == *"audit:B5"* || $prev_line_4 == *"audit:B5"* ]]; then
          review_refs+=("$file:$symbol")
        else
          violations+=("$file:$line_no: function $symbol is missing audit:B5 review tag")
        fi
      fi
    fi

    if [[ $text =~ $comment_re ]]; then
      if [[ $text == *"audit:B5"* || $prev_line_1 == *"audit:B5"* || $prev_line_2 == *"audit:B5"* || $prev_line_3 == *"audit:B5"* || $prev_line_4 == *"audit:B5"* ]]; then
        review_refs+=("$file")
      else
        violations+=("$file:$line_no: comment marker is missing audit:B5 review tag")
      fi
    fi

    prev_line_4="$prev_line_3"
    prev_line_3="$prev_line_2"
    prev_line_2="$prev_line_1"
    prev_line_1="$text"
  done < <(nl -ba -w1 -s $'\t' "$file")
done

if ((${#violations[@]} > 0)); then
  echo "Unreviewed B5 risk markers found:" >&2
  printf '  %s\n' "${violations[@]}" >&2
  exit 1
fi

if ((${#review_refs[@]} > 0)); then
  mapfile -t unique_review_refs < <(printf '%s\n' "${review_refs[@]}" | sort -u)
  missing_reviews=()
  for ref in "${unique_review_refs[@]}"; do
    if ! rg -F -q "$ref" "$REVIEW_FILE"; then
      missing_reviews+=("$ref")
    fi
  done

  if ((${#missing_reviews[@]} > 0)); then
    echo "Tagged B5 risk markers missing from $REVIEW_FILE:" >&2
    printf '  %s\n' "${missing_reviews[@]}" >&2
    exit 1
  fi
fi

echo "B5 risk marker policy check passed"
