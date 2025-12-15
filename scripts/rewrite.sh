#!/usr/bin/env bash
set -e

for file in [0-9][0-9]-*.md; do
  # skip if no matching files
  [ -e "$file" ] || continue

  base="${file%.md}"
  dir="$base"
  target="$dir/README.md"

  # create folder if not exists
  mkdir -p "$dir"

  # prevent overwrite
  if [ -f "$target" ]; then
    echo "SKIP: $target already exists"
    continue
  fi

  mv "$file" "$target"
  echo "MOVED: $file -> $target"
done

echo "Done."
