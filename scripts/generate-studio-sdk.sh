#!/usr/bin/env bash
set -euo pipefail

studio_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$studio_root"

GOWORK=off go run ./cmd/studio-openapi
GOWORK=off go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0 \
  -generate types,client -package studioopenapi \
  -o sdk/openapi/client.gen.go sdk/openapi/studio.json

cd ui
npm run generate:studio-client
