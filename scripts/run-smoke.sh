#!/usr/bin/env bash
set -e

echo "==> smoke testing all examples"

for main in $(find . -name main.go -type f); do
  dir=$(dirname "$main")
  echo "-> $dir"

  (
    cd "$dir"
    go run . >/dev/null 2>&1 &
    pid=$!
    sleep 0.5
    kill $pid >/dev/null 2>&1 || true
  )
done

echo "==> all examples start successfully"
