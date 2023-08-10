#!/usr/bin/env bash

set -euo pipefail

echo "Building SPIRE agent supply..."
ENABLE_CGO=0 GOARCH=amd64 GOOS=linux go build -o bin/supply ./src/spire/supply/cli
echo "Done."
