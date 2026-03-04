#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

verbose=0
if [[ "${1:-}" == "--verbose" ]]; then
  verbose=1
fi

echo "[strict-test] race + shuffle + rerun + atomic coverage"

cmd=(
  go test
  -race
  -count=1
  -shuffle=on
  -covermode=atomic
  -coverpkg=./...
  -timeout=15m
)

if [[ $verbose -eq 1 ]]; then
  cmd+=(-v)
fi

cmd+=(./...)
"${cmd[@]}"

echo "[strict-test] OK"
