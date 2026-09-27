#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

gate_failed=0
node scripts/check-native-sdk-coverage.mjs || gate_failed=1
node --test scripts/verify-live-reader.test.mjs || gate_failed=1

datly_replace="$(GOWORK=off go list -m -f '{{if .Replace}}{{.Replace.Path}}{{end}}' github.com/viant/datly)"
if [[ -n "$datly_replace" ]]; then
  printf 'release build blocked: github.com/viant/datly uses replace %s\n' "$datly_replace" >&2
  printf 'Publish the required Datly changes, pin that immutable version, and remove the replace directive.\n' >&2
  gate_failed=1
fi
if [[ "$gate_failed" -ne 0 ]]; then exit 1; fi

GOWORK=off go mod verify
GOWORK=off go test ./...
build_dir="$(mktemp -d "${TMPDIR:-/tmp}/datly-studio-build.XXXXXX")"
GOWORK=off go build -trimpath -buildvcs=false -o "$build_dir/datly" ./cmd/datly
GOWORK=off go build -trimpath -buildvcs=false -o "$build_dir/studio-api" ./cmd/studio-api
GOWORK=off go build -trimpath -buildvcs=false -o "$build_dir/studio-runtime" ./cmd/studio-runtime

cd "$repo_root/ui"
npm ci
npm test
npm run build
printf 'Verified Go binaries: %s\nVerified UI bundle: %s\n' "$build_dir" "$repo_root/ui/dist"
