#!/usr/bin/env bash
set -euo pipefail

BOOK="BOOK.md"
ROOT_README="README.md"

: > "$BOOK"

if [[ -f "$ROOT_README" ]]; then
  cat "$ROOT_README" >> "$BOOK"
  echo "" >> "$BOOK"
fi

shift_headings_one_level() {
  awk '
    BEGIN { in_fence = 0 }

    # Toggle on lines that start a fenced code block: ``` or ```lang
    /^[[:space:]]*```/ { in_fence = !in_fence; print; next }

    {
      if (!in_fence && $0 ~ /^#{1,6}[[:space:]]/) {
        # If already ######, keep it at ######
        if ($0 ~ /^######[[:space:]]/) { print; next }
        sub(/^#/, "##", $0)
      }
      print
    }
  '
}

for dir in $(ls -d [0-9][0-9]-* 2>/dev/null | sort); do
  sub="$dir/README.md"
  [[ -f "$sub" ]] || continue

  echo "" >> "$BOOK"
  echo "" >> "$BOOK"

  echo "" >> "$BOOK"

  shift_headings_one_level < "$sub" >> "$BOOK"
done

echo "BOOK.md generated: $BOOK"
