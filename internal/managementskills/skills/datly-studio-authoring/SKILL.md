---
name: datly-studio-authoring
description: "Create and edit Data Studio readers, MCP tools and cubes: discover source schemas, declare typed input predicates, build WHERE clauses with predicate.Builder, validate a specific draft revision and verify publication."
group: data-authoring
---

# Datly Studio authoring

Use the connected management MCP to take an authorized source query through a
validated component and, when requested, a live namespace tool. Discover actual
tool schemas before calling them; this skill does not grant permission to publish,
change namespace configuration, access new data, or execute expensive queries.

For a **single cube**, use `studio.sdk.preview.execute` with `input.cube:true`
and the cube's dimensions, measures, filters and limit in `input.input`.
`versions.test_compose` is for separately enabled multi-cube composition, not a
single-cube test. Do not enable composition just to test a cube.

After publication, discover and invoke through `studio.sdk.catalog.tools.list`
and `studio.sdk.catalog.tools.call` with the namespace ID. Host-wide tool search
may not include a newly published namespace. Saved cube-configuration listings
are not the live tool catalog and an empty configuration list does not establish
that a derived cube is absent.

## Required authoring sequence

For a reader request described in natural language rather than supplied DQL, read
[Create a reader from requirements](references/reader-from-requirements.md). Derive the implementation
from discovered contracts and verified schema relationships; do not require the
caller to write source or infer authorization from names.

For customer-facing dimensions, prefer verified names over bare IDs. If a fact
contains only IDs, resolve names through native outer dictionary views, including
across different connectors. Preserve IDs, missing-metadata facts, authorization
and cube grain; do not add keys to rollups solely to obtain labels. Follow the
[dimension enrichment contract](references/reader-from-requirements.md#resolve-dimension-names-through-native-outer-views).

Use this Data Studio connection's `studio.sdk.*` tools for components, readers
and cubes. Resource-catalog tools are a different authoring surface; do not
substitute them for component version operations.
Resolve the requested namespace once; retain the returned namespace ID,
component ID, draft version number and source revision throughout the task.
Before authoring, read `studio.sdk.namespaces.settings_get` for the resolved
namespace. If `security.requireSecurityPredicate` is true, check the configured
`security.predicatePackages`, then discover allowed types with
`studio.sdk.authorization_predicates.types(namespaceId=...)`. Bind a real
installed authorization predicate from a selected package to its required
trusted input; keep it effective when optional filters are absent. An import,
a client-provided ID filter, or a type listed only by unscoped discovery does
not satisfy the requirement. If no selected package/type is usable, report the
exact prerequisite and preserve the setting. Do not disable it to publish.
Validate the exact draft and let the server enforce the compiled policy again
at publication. Follow [Security and resource checks](references/security-and-resource-checks.md).

Read `references/dql.md` before writing source and the relevant portion of
`references/workflow.md` before saving or publishing. Load only the references
needed for the requested change; do not enumerate unrelated tools or tables.

For an existing DQL draft, discover `versions.inspect`, `versions.apply` and
`versions.validate` together using the management provider and name filters.
Read source with `versions.inspect(discoverColumns:false)` before editing; column
discovery is a separate operation and can fail on the source you are repairing.
For `versions.apply`, use `command.kind: "set_dql"`, put the complete source in
`command.payload.authoredDql`, and put the readback revision in
`command.expectedSourceRevision`. A `dql` field is not this payload contract.
Read back after saving, then validate the returned revision. Do not publish a
requested draft edit unless publication is also requested.

**Construct authored filtering WHERE clauses with `predicate.Builder`.** Declare
each input's type, transport binding, presence and predicate in `#define` first:

```sql
#define($_ = $RecordID<int>(query/recordId).Required().WithPredicate(0, 'equal', 'r', 'id'))
#define($_ = $Name<*string>(query/name).Optional().WithPredicate(0, 'contains', 'r', 'name'))
```

Inside the named view's SQL, after `FROM records r`, use:

```sql
${predicate.Builder().CombineOr($predicate.FilterGroup(0, "AND")).Build("WHERE")}
```

The group number links the input declarations to `FilterGroup`; alias `r` must
match the inner SQL. `Build("WHERE")` supplies the keyword. Do not write
`WHERE $criteria.In(0)`, concatenate input values into SQL, or prepend another
WHERE. Keep required trusted authorization outside optional OR alternatives.
Required caller filters alone are not authorization. The complete typed reader
and multiple-group composition are in `references/dql.md`.

Prefer registered typed predicates over custom SQL filter expressions. Use an
expression predicate only when the native catalog cannot express the required
source semantics, keep values bound, and validate the emitted query through
the actual connector. Do not assemble WHERE text from caller input.

**Edit a specific draft, not an unspecified latest resource.** Metadata updates
use the component ETag; source edits use `versionNo` plus
`expectedSourceRevision`. Each source edit increments the draft's source revision
and invalidates prior validation. Read the new revision, validate it, then publish
that exact version/revision only when requested. Published source requires a new
editable draft or clone. Report saved, validated, published and runtime-active
outcomes separately.

Read the relevant supporting resources with native `resources/read`, resolving
paths relative to this skill URI:

- [Grammar by task](references/grammar-map.md): small entry points for reader,
  column, cube, relation and version scenarios.
- [Column output how-to](references/column-output-howto.md): native date tags,
  formatting versus truncation, and serialization-boundary gotchas.
- [Security and resource checks](references/security-and-resource-checks.md):
  namespace predicate packages and thresholds, explicit component opt-in, and
  source-validation versus dry-run and returned-row evidence.
- [Version-session how-to](references/version-session-howto.md): edit/release
  checkpoints and practical verification scenarios.

- [MCP workflow](references/workflow.md): inspecting saved components, connector and schema discovery, editing,
  revision checks, validation, release, runtime readiness, authentication, and recovery.
- [Complete worked example](references/worked-example.md): exact create/apply/validate/release payloads and a fact-table cube with two metadata views.
- [Builder operation schema](references/builder-operations.schema.json): JSON payload fields and supported operation types, checked against the linked builder.
- [DQL](references/dql.md): complete reader shape, bindings, SQL column discovery,
  predicates, resources, and generation boundaries.
- [Formal grammar](references/dql.ebnf.txt): structural Datly v1 grammar; SQL dialects,
  registered types/codecs, and semantic validity still require the connected compiler.
- [Cubes and dictionaries](references/cubes-and-dictionaries.md): fact grain,
  dimension/measure roles, fact aggregation, and metadata relations.
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

## Find the operation before declaring it unavailable

Use the provider's advertised catalog and preserve the returned callable tool
name. Search one operation at a time when a broad query returns unrelated tools;
follow that query's cursor when needed. A search miss is not a capability check.
Do not invent pluralizations, suffixes, or punctuation variants of tool names.

The SDK operation identities are:

- `studio.sdk.connectors.tables`: bounded table inventory.
- `studio.sdk.connectors.table`: columns and keys for one exact table (singular
  `table`, not `columns`).
- `studio.sdk.versions.inspect`: saved source and compiled metadata.
- `studio.sdk.versions.create`, `studio.sdk.versions.apply`,
  `studio.sdk.versions.validate`, `studio.sdk.versions.preview`: draft lifecycle.

These are catalog search identities, not permission to assume a host exposes
all of them. Resolve their exact callable names and input schemas. For release
and invocation, inspect the advertised namespace/version publication operations
and runtime endpoint; do not conclude they are absent from a page of draft tools.
For a host-specific trusted binding, discover that host's authorization or secure
reader skill and its usage reference. Generic predicate type discovery alone
may not contain the complete binding recipe. Retrieve referenced business
knowledge through the resource-management catalog when the task names it.
