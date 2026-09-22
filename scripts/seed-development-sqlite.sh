#!/usr/bin/env bash

# Rebuild the deterministic local reporting database and point every live
# Studio connector at it. This is intentionally a development-only fixture;
# it must not be used against a shared or production Studio database.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
data_dir="$repo_root/.data"
studio_db="$data_dir/studio.db"
reporting_db="$data_dir/reporting-unit.db"
reporting_dsn="file:$reporting_db"

command -v sqlite3 >/dev/null 2>&1 || {
  echo "sqlite3 is required to seed the development fixture" >&2
  exit 1
}

mkdir -p "$data_dir"

# Recreate this explicitly named fixture so removed rows cannot survive a
# rerun. The source schema and seed rows remain the single reproducible input.
rm -f "$reporting_db"
sqlite3 "$reporting_db" <<SQL
.read $repo_root/schema/sqlite/reporting.sql
.read $repo_root/schema/sqlite/reporting_seed.sql
.read $repo_root/schema/sqlite/development_seed.sql
SQL

if [[ -f "$studio_db" ]]; then
  sqlite3 "$studio_db" <<SQL
PRAGMA foreign_keys = ON;
UPDATE connectors
SET driver = 'sqlite',
    dsn_template = '$reporting_dsn',
    secret_ref = NULL,
    status = 'active',
    options_json = '{"sqliteAttachments":{"ci_ads":"$reporting_db"}}',
    last_test_status = NULL,
    last_test_error_code = NULL,
    last_tested_at = NULL,
    etag = etag + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE deleted_at IS NULL;
SQL
else
  echo "Studio database not found: $studio_db" >&2
  exit 1
fi

echo "seeded $reporting_db"
echo "pointed non-deleted Studio connectors at $reporting_dsn"
