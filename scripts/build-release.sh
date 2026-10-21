#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

cd "$ROOT"
npm ci
npm run build

cd "$ROOT/api"
CGO_ENABLED=0 GOOS=linux GOARCH="${GOARCH:-amd64}" \
  go build -trimpath -ldflags="-s -w" -o library-api ./cmd/server
