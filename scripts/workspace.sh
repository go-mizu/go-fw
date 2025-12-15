#!/usr/bin/env bash
set -e

modules=$(find . -name go.mod -type f -exec dirname {} \;)
go work init >/dev/null 2>&1 || true
go work use $modules