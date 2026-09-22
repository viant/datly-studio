# Datly Studio DQL

The static Datly aggregates cover Connector, Report, ReportVersion, views,
parameters, cube configuration, MCP exposures, resources, skills, runtime
generations, publications, and ACL. A Datly component owns the DAO
and service behavior for its capability; there is no separate forwarding DAO or
service layer.

```text
dql/
  studio/
    connectors/
      reader/
        connector.dql
        sql/read.sql
      writer/
        connector.dql
        sql/patch.sql
    reports/
      reader/report.dql
      writer/report.dql
    report_versions/
      reader/version.dql
      writer/version.dql
    report_views/reader/view.dql
    report_parameters/
      reader/parameter.dql
      writer/parameter.dql
    report_cube_configs/
      reader/config.dql
      writer/config.dql
    report_mcp_exposures/
      reader/exposure.dql
      writer/exposure.dql
    report_resource_files/
      reader/file.dql
      writer/file.dql
    report_resource_folders/
      reader/folder.dql
      writer/folder.dql
    report_skill_roots/
      reader/skill.dql
      writer/skill.dql
    runtime_generations/
      reader/generation.dql
      writer/generation.dql
    report_publications/
      reader/publication.dql
      writer/publication.dql
    report_acl/
      reader/acl.dql
      writer/acl.dql
```

Every source declares its required `studio/.../reader` or `studio/.../writer`
package explicitly.

- `reader/connector.dql` serves the regular collection route. Its `Name` path parameter uses
  `WithURI('/{name}')`, so Datly materializes the `ByName` route from the same
  reader component.
- `writer/connector.dql` is the one mutation component. Generated PATCH behavior supplies
  sparse Has semantics for inserts and updates. `delete_marker` requires an
  explicit `shouldDelete: true` with a complete existing identity; omission and
  false never delete. `etag` is the expected concurrency token.

`report_versions` uses the composite `(report_id, version_no)` identity.
Its reader scopes every request to `reportId`, adds a `WithURI` version route,
and supports state/mode/compile-status/author predicates plus selectors. Its
writer performs full insert validation, sparse PATCH validation, and optimistic
concurrency through `source_revision`. Version lifecycle uses state transitions;
there is deliberately no physical delete marker.

All DQL filenames are lowercase snake case. SQLite unit behavior is required
for each aggregate; MySQL rule/E2E coverage remains a separate Endly stage.

Database SQL lives under each component's `sql/` folder and is included with
Datly 1.0 `${embed:sql/...}` resources. DQL retains package, route, bindings,
view graph, type, selector, mutation, and exposure metadata.

## Minimum unit contract

Every Datly aggregate must generate and execute its reader/writer packages
against SQLite before it is accepted. Use `internal/datatest` for ordered
JSON hydration, schema validation, query verification, isolated generated modules, and
generated-package test execution.

A reader suite must cover:

- generated input, output, and UpperCamel view shapes;
- collection and `WithURI` identity routes;
- every predicate independently, grouped OR/AND behavior, combined groups,
  and omitted versus explicitly supplied empty/zero predicate inputs through
  generated `Has` markers;
- field projection, every enabled selector, pagination, allowed ordering, and
  rejection of disallowed ordering;
- empty results and exclusion rules such as soft-deleted rows.

A PATCH writer suite must cover:

- generated input, output, entity, `Has`, and SQLX physical-column mappings;
- insert and update classification;
- sparse updates preserving every omitted field;
- explicit false/zero/null where supported by the model;
- valid and stale concurrency tokens with rollback verification;
- false/omitted delete markers preserving rows and explicit true deleting;
- database state and returned transformed output after each operation.

`datly transcribe get|patch` must emit inspectable Go shapes (`input.go`,
`output.go`, view/entity files, and writer support) in the component's declared
`#package`; the generated import path is `<module>/<#package>` and the declared
Go package name is the terminal `#package` segment. A successful DQL parse
without these package-matched artifacts is insufficient.
