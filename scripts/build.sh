#!/usr/bin/env bash
set -euo pipefail

for mod in $(find . -name go.mod -type f); do
  dir=$(dirname "$mod")
  (cd "$dir" && go build ./...)
done
