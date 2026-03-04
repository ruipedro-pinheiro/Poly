#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "[1/4] build"
go build ./...

echo "[2/4] vet"
go vet ./...

echo "[3/4] lint (same command family as CI)"
if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run ./...
else
  go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...
fi

echo "[4/4] test -race"
go test -race -count=1 ./...

echo "OK: local strict CI passed"
