# Studio database migration runbook

This runbook upgrades the **Studio metadata database** configured by `Store` in
`studio.yaml` or `datastudio.yaml`. It does not migrate application databases
referenced by reader connectors.

The current canonical layout is **29**. An empty store is initialized by the host.
An existing store is verified, not upgraded, at startup. Do not apply the fresh
creation DDL to an existing database. See the [schema contract](../schema/CONTRACT.md)
for ownership and retention details.

## Before upgrading

1. Use the migration scripts from the application revision being deployed. Run
   the commands below from the repository root.
2. Stop new writes and let accepted warmup/publication work finish. Stop every
   Studio instance, worker, recovery host, scheduled task, and other writer using
   this metadata database. Closing browser tabs does not stop accepted jobs.
   Coordinate other users of any shared Authz tables in the same database.
3. Take a database backup to a new path. Verify that backup and rehearse restoration
   on a disposable database before changing the deployment database.
4. Read the current layout and select only the next applicable script. Do not
   run all files using a wildcard or continue after a failed command.

Use database-client authentication appropriate to the deployment. The MySQL
examples assume an already configured `--login-path=studio` with backup and DDL
permissions; they do not put passwords on the command line.

## Upgrade order

Each basename has both `.mysql.sql` and `.sqlite.sql` variants in
[`schema/migrations`](../schema/migrations).

| Current layout | Next layout | Script basename | Main change |
| ---: | ---: | --- | --- |
| 23 | 24 | `024-audit` | Audit switches, audit records, and audit settings |
| 24 | 25 | `025-artifact-releases` | Release bundles, channels, and pending operations |
| 25 | 26 | `026-artifact-release-bindings` | Release dependency metadata and exact bindings |
| 26 | 27 | `027-namespace-settings` | Namespace settings and reader resource-check opt-in |
| 27 | 28 | `028-permanent-removal` | Deletion receipts, version allocation counter, retained publication history |
| 28 | 29 | `029-warmup-deadline` | Accepted warmup deadlines |
| 29 | 30 | `030-warmup-schedules` | Durable interval schedules; existing run history unchanged |

For example, layout 26 needs scripts 027, 028, 029, and 030 in that order. Layout 30
needs none. Missing, conflicting, older-than-23, or newer-than-30 layout markers
need a separately reviewed upgrade path; do not guess or alter the marker.

The marker is an operator guide, not proof of the physical schema. A previous
partial migration can leave new objects behind while the marker still reports
the old version. Startup checks actual canonical tables, columns, indexes, and
removal constraints; changing `schema_version` does not repair those objects.

## SQLite

Set paths for the configured metadata file and a new backup file. Use filesystem
paths, not the complete `file:...?...` DSN from configuration:

```sh
STUDIO_DB=/srv/studio/studio.db
STUDIO_BACKUP=/srv/backups/studio-before-upgrade.db

sqlite3 -readonly -bail "$STUDIO_DB" 'SELECT version FROM schema_version;'
sqlite3 -readonly -bail "$STUDIO_DB" ".backup '$STUDIO_BACKUP'"
sqlite3 -readonly -bail "$STUDIO_BACKUP" 'PRAGMA integrity_check;'
```

Create the backup directory beforehand. Do not overwrite an earlier backup. The
backup integrity result must be `ok`. SQLite's `.backup` captures a consistent
database, including committed WAL data; copying only the main database file can
omit that data.

The following guarded example applies **only layout 28→29**:

```sh
if [ "$(sqlite3 -readonly -bail "$STUDIO_DB" 'SELECT version FROM schema_version;')" = 28 ]; then
  sqlite3 -bail -cmd 'PRAGMA foreign_keys=ON;' "$STUDIO_DB" \
    < schema/migrations/029-warmup-deadline.sqlite.sql
else
  printf '%s\n' 'Expected layout 28; stop and inspect the database.' >&2
  false
fi
```

For earlier steps, change both the expected starting version and the script
basename according to the table. Apply one script per process, check its exit
status, then confirm its resulting version before continuing. The scripts own
their transactions; do not wrap them in another `BEGIN`.

After the final step:

```sh
sqlite3 -readonly -bail "$STUDIO_DB" 'SELECT version FROM schema_version;'
sqlite3 -readonly -bail "$STUDIO_DB" 'PRAGMA integrity_check; PRAGMA foreign_key_check;'
sqlite3 -readonly -bail "$STUDIO_DB" "PRAGMA table_info('component_warmup_runs');"
```

Expect layout `29`, integrity `ok`, no foreign-key violation rows, and a nullable
`deadline_at` column. These checks supplement the application's startup verifier.

## MySQL

Set the configured metadata database name and a new backup file:

```sh
STUDIO_DATABASE=studio
STUDIO_BACKUP=/srv/backups/studio-before-upgrade.sql

mysql --login-path=studio --database="$STUDIO_DATABASE" --batch --skip-column-names \
  -e 'SELECT version FROM schema_version;'
if mysqldump --login-path=studio --single-transaction --routines --triggers --events \
  --no-tablespaces "$STUDIO_DATABASE" > "$STUDIO_BACKUP"; then
  test -s "$STUDIO_BACKUP"
else
  printf '%s\n' 'Backup failed; do not migrate or use the partial dump.' >&2
  false
fi
```

Create the backup directory beforehand. Proceed only if the dump command
succeeded and restoration has been checked. `--single-transaction` does not make
concurrent DDL safe; keep writers and schema changes stopped.

The following guarded example applies **only layout 28→29**:

```sh
if [ "$(mysql --login-path=studio --database="$STUDIO_DATABASE" --batch --skip-column-names -e 'SELECT version FROM schema_version;')" = 28 ]; then
  mysql --login-path=studio --database="$STUDIO_DATABASE" \
    < schema/migrations/029-warmup-deadline.mysql.sql
else
  printf '%s\n' 'Expected layout 28; stop and inspect the database.' >&2
  false
fi
```

For earlier steps, change both the expected starting version and the script
basename according to the table. Apply one file at a time and verify its resulting
version. Do not use `mysql --force`: batch execution must stop on SQL errors.

**MySQL DDL commits implicitly.** A later failure can leave earlier statements
applied even though the script's final version update did not execute. Adding
`BEGIN` around a migration does not make its DDL atomic.

After the final step:

```sh
mysql --login-path=studio --database="$STUDIO_DATABASE" \
  -e "SELECT version FROM schema_version; SHOW COLUMNS FROM component_warmup_runs LIKE 'deadline_at';"
```

Expect layout `29` and a nullable `deadline_at` column. Application startup must
also pass its canonical schema verification.

## Failure, reruns, and restoration

None of these scripts is a general-purpose idempotent repair script. Successfully
applied scripts must not be rerun: they contain unconditional `ALTER TABLE`,
`CREATE TABLE`, or index creation statements.

- **SQLite:** with `-bail`, the client exits on an error and an open migration
  transaction rolls back when the connection closes. Earlier successfully
  committed migration steps remain applied. Inspect the actual layout before
  deciding whether to retry a failed step.
- **MySQL:** assume a failed script may be partially applied. Keep services
  stopped, preserve the error, and inspect actual objects. Either restore the
  backup or prepare a reviewed completion procedure for the observed state.
  Do not rerun the whole script or advance the marker manually.
- There are no reverse/downgrade scripts in this directory. A binary rollback
  is not a database rollback.

Restore into a **new, empty destination**, not over the partially migrated
database. For SQLite:

```sh
STUDIO_RESTORE_DB=/srv/studio/restored-studio.db
sqlite3 -readonly -bail "$STUDIO_BACKUP" ".backup '$STUDIO_RESTORE_DB'"
sqlite3 -readonly -bail "$STUDIO_RESTORE_DB" 'PRAGMA integrity_check; SELECT version FROM schema_version;'
```

Here `STUDIO_BACKUP` must name the SQLite backup, not a MySQL dump. Choose a fresh
destination path, preserving the failed database and its sidecar files for
diagnosis. Update `Store` to the restored file before starting compatible binaries.

For MySQL, create a new empty database using the deployment's normal database
provisioning procedure, then load the complete dump:

```sh
STUDIO_RESTORE_DATABASE=studio_restore
mysql --login-path=studio --database="$STUDIO_RESTORE_DATABASE" < "$STUDIO_BACKUP"
mysql --login-path=studio --database="$STUDIO_RESTORE_DATABASE" \
  -e 'SELECT version FROM schema_version;'
```

Confirm database grants and update `Store` or its secret reference to the restored
database. Loading an old dump over a partially migrated database is insufficient:
tables introduced by the failed migration might remain. Coordinate restoration
of shared Authz data with its other consumers.

## Restart order

1. Finish every required migration and verification with all writers stopped.
2. Deploy compatible binaries to every authoring/API, worker, and recovery host.
3. Start the upgraded hosts and confirm schema verification succeeds before
   restoring traffic and scheduled writes. Startup verification does not perform
   migration or repair.
4. After **all** warmup recovery hosts are upgraded, enable longer
   `WarmupOperationTimeout` values if desired. Old binaries ignore `deadline_at`
   and could expire a longer job prematurely.

Migration 029 leaves historical deadlines `NULL`; those runs retain their
`requested_at + 5m` budget. Newly accepted jobs store their configured deadline.
See [warmup deadline configuration](warmup-dependency-runtime.md).

## Recurring warmup schedules (29→30)

Apply `schema/migrations/030-warmup-schedules.mysql.sql` or its `.sqlite.sql`
counterpart after stopping writers and taking the backup described above.
The migration creates one table and its due-time index, then records layout 30;
it does not change existing run statuses, deadlines, diagnostics, or cached data.

Before enabling the host scheduler, configure a renewing verified service
credential provider and grant that identity ordinary warmup publish authority
and required connector visibility. See [warmup scheduling](warmup-scheduling.md).
The browser does not supply credentials, deadlines, or background execution inputs.
