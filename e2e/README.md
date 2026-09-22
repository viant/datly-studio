# Datly Studio end-to-end automation

The local E2E environment uses [Endly](https://github.com/viant/endly), matching
the orchestration structure used by Datly.

## Prerequisites

- `endly` installed and available on `PATH`
- Docker running locally
- an Endly credential named `mysql-e2e` with user `root` and password `dev`
- Go 1.25.x

## Run

From `e2e/local`:

```bash
endly
```

Run selected phases:

```bash
endly -t=init
endly -t=build,test
endly -t=destroy
```

The workflow starts a MySQL 8 container on host port `43306`, recreates two
databases with DSUnit, loads deterministic fixtures, builds/tests Studio, and
then runs the regression workflow.

Datastores:

- `datly_studio_e2e`: Studio connectors, reports, versions, and publications
- `reporting_e2e`: source data used by report components

Unit tests derive SQLite DDL in memory from `schema/schema.ddl`; Docker
MySQL remains the E2E dialect.

The catalog schema represents the Datly 1.0 redesign. Store and SDK integration
tests will be added as those layers are implemented.

The initial catalog also models named MCP route exposures, cube and cube-compose
settings, versioned resource files, resource folders, and explicitly declared
skill roots. Skill resources are stored as version artifacts so a published
generation can rebuild the same Datly MCP resource snapshot after restart.
