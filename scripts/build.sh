#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
(cd web && npm ci && npm run build)
mkdir -p dist
go build -trimpath -o dist/devhub ./cmd/devhub
