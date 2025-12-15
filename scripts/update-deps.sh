#!/usr/bin/env bash
set -euo pipefail

# Run from repo root.
# Updates deps for every nested go.mod under NN-*/<fw>/.

FRAMEWORK_TARGETS=(
  "github.com/go-chi/chi/v5@latest"
  "github.com/gin-gonic/gin@latest"
  "github.com/gofiber/fiber/v2@latest"
  "github.com/labstack/echo/v4@latest"
  "github.com/go-mizu/mizu@latest"
)

mods() {
  find . -name go.mod \
    -not -path "./.git/*" \
    -not -path "./scripts/*" \
    -print0 | xargs -0 -n 1 dirname | sort
}

has_go_files() {
  local dir="$1"
  # skip modules that don't contain any .go files (avoids noisy warnings)
  find "$dir" -maxdepth 1 -name '*.go' -print -quit | grep -q .
}

update_one() {
  local dir="$1"
  echo "==> $dir"
  pushd "$dir" >/dev/null

  # Prefer framework-only upgrades (stable, predictable).
  for t in "${FRAMEWORK_TARGETS[@]}"; do
    go get -d "$t" >/dev/null 2>&1 || true
  done

  go mod tidy >/dev/null 2>&1 || true
  popd >/dev/null
}

main() {
  while IFS= read -r dir; do
    if has_go_files "$dir"; then
      update_one "$dir"
    else
      # If you want to force-update even empty modules, remove this skip.
      echo "==> $dir (skip: no .go files)"
    fi
  done < <(mods)

  if [[ -f go.work ]]; then
    go work sync >/dev/null 2>&1 || true
  fi

  echo "Done: updated dependencies."
}

main
