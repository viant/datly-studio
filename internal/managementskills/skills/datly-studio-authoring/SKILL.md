---
name: datly-studio-authoring
description: Inspect and explain existing Datly components, discover database schemas, create Datly v1 readers and cubes with BigQuery facts and MySQL dictionaries, and validate, release, and verify them through Studio management MCP.
---

# Datly Studio authoring

Use the connected management MCP to take an authorized source query through a
validated component and, when requested, a live namespace tool. Discover actual
tool schemas before calling them; this skill does not grant permission to publish,
change namespace configuration, access new data, or execute expensive queries.

Read the relevant supporting resources with native `resources/read`, resolving
paths relative to this skill URI:

- [MCP workflow](references/workflow.md): inspecting saved components, connector and schema discovery, editing,
  revision checks, validation, release, runtime readiness, authentication, and recovery.
- [Complete worked example](references/worked-example.md): exact create/apply/validate/release payloads and a BigQuery cube with two MySQL lookups.
- [Builder operation schema](references/builder-operations.schema.json): JSON payload fields and supported operation types, checked against the linked builder.
- [DQL](references/dql.md): complete reader shape, bindings, SQL column discovery,
  predicates, resources, and generation boundaries.
- [Formal grammar](references/dql.ebnf.txt): structural Datly v1 grammar; SQL dialects,
  registered types/codecs, and semantic validity still require the connected compiler.
- [Cubes and dictionaries](references/cubes-and-dictionaries.md): fact grain,
  dimension/measure roles, BigQuery aggregation, and MySQL lookup relations.
- [Report-consumer handoff](references/report-consumers.md): verified reader
  contracts for dashboard agents, including source branches, date semantics,
  required context, nullable calculations and execution evidence.

## Keep the contracts distinct

A component **version** is its draft/release identity. A **source revision** changes
when that draft is edited. A runtime **generation** identifies deployed content.
Saving Cube or MCP settings can reset validation to `pending`: it means **needs
validation**, not that a background publication is running. Validate the latest
source revision, then release that same draft version; a failed activation alone
does not require creating v2. Published source changes require an editable draft.

Component MCP exposure, cube MCP exposure, and namespace MCP availability are
separate controls. A valid draft with MCP enabled is not yet a callable tool.
Verify the namespace runtime before release, then verify active publication and
an authenticated tools/list on its returned MCP URL. Management tools/list is not
proof of a business component's activation.

## Authoring boundaries

Use Datly v1 components, typed inputs/output holders and native SQL discovery.
Request `studio.sdk.versions.inspect` with `discoverColumns: true` for view/result
columns. Datly's zero-row SQL discovery obtains aliases, expressions and driver
column types; do not replace it with table catalog metadata or browser inference.

Use existing authorized connectors. Keep verified subject/tenant scope outside
optional user predicate groups. Do not place database credentials or OAuth secrets
in DQL, resource files, skill content, tool arguments, or reports. Use expected
source revisions and etags; reload conflicts instead of overwriting someone else's
work. Leave the database schema unchanged unless DDL is explicitly requested.

Prefer SDK authoring operations over editing Studio persistence directly. For
filesystem development, use the selected Datly v1 reader/writer/custom-component
skills and generation tools; do not substitute legacy repository/view APIs or
hand-edit generated Go. Inspect the installed command/tool capabilities rather
than inventing names or flags.

Finish by reporting the component/version/source revision, checks actually run,
namespace/runtime state, and whether a release and authenticated tool verification
succeeded. Do not equate initialization, static validation, or a preview with a
successful release.
