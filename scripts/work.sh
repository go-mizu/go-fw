#!/usr/bin/env bash
set -euo pipefail

modules=$(find . -name go.mod -type f -exec dirname {} \;)

go work init >/dev/null 2>&1 || true

# include root module for tools (cmd/extract)
go work use .

# include all example modules
go work use $modules

go work sync
