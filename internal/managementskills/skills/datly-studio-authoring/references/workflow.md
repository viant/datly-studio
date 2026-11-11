# Management MCP workflow

## Discover and select

Use native tools/list and read input schemas. This server normally advertises
`studio.sdk.*` authoring tools; business tools live on namespace MCP endpoints.
Use the authenticated connection supplied by the host, with the caller's verified
identity. Do not assume protocol dates or port numbers: negotiate a supported MCP
version and retrieve the namespace connection/runtime metadata.

Search within this management connection and the needed SDK operation family.
Do not repeatedly search the entire tool catalog for generic terms such as
"predicate" or "read". `authorization_predicates.*` manages authorization links;
it is not the SQL filter builder. Input predicates and `predicate.Builder`
syntax are documented in this skill's DQL reference.

If host skill discovery reports a cursor/snapshot mismatch when no cursor was
sent, report that distinction. Do not invent a cursor or cycle through unrelated
searches. Use this same server's advertised native skill/resource operations to
read its authoring guide when available. Do not guess a skill URI, provider alias
or syntax when the guide could not be loaded.

List namespaces and choose the intended canonical `namespaceId`; include it in
requests where advertised. Names and canonical IDs are not interchangeable: `namespaces.get` takes `name`
(the slug such as `analytics`), while component/version/reader/runtime operations
use the returned canonical `namespaceId`.
Check ownership/manager permissions. Namespace execution defaults can grant access
to all underlying tools; explicit tool policies override those defaults. Do not
make a namespace public or grant global admin as a workaround. Audit viewing is a
separate permission from tool or namespace management.

Inspect active connectors using connectors.list with a fields projection.
Use connectors.get only when its full configuration is needed; discovery lookup
lists can make that response large. Probe with
connectors.test when connectivity is uncertain. connectors.discover supplies
product/version, projects/catalogs and schemas/datasets; connectors.schemas,
connectors.tables and connectors.table browse physical metadata when advertised.
Use their actual schemas, including catalog and schema inputs. BigQuery usually
maps project to catalog and dataset to schema. Configured lookup lists are editable
choices, not hard restrictions or proof that no other qualified SQL can work.

For result metadata, create/inspect the component with versions.inspect and
`discoverColumns: true`. Discovery uses the saved SQL and per-view connectors,
including expressions and aliases; table catalog availability is not its contract.
A discovery failure is not proof that a table is absent. Report driver/permission
limitations and retain known columns rather than inventing fields.

## Inspect and explain an existing component

Resolve the component in the selected namespace with components.list/get, then
select its actual version with versions.list/get. Use versions.inspect with
discoverColumns false for the saved graph and DQL; enable discovery only when
live result types are needed. Read its versioned SQL/resources with the advertised
resources operations. A component name, UI label or physical table alone is not
enough to explain a computed reader.

Describe the root SQL and connector, selected grain, dimensions, additive and
non-additive measures, input requiredness, predicates, relation keys/cardinality,
per-view connectors, and output shape from that exact saved version. Separate
verified authorization from caller-supplied filters. Explain which dictionary
lookups enrich facts and whether missing labels retain their parent rows.

Read publication/runtime evidence separately from source validation. Distinguish
the draft source revision, validated revision, published version and active
generation; do not describe a pending edit as deployed. Do not claim data was
queried or reconciled from metadata alone. Explanation does not require a save,
validation, query or release. If source is capability-restricted, state that limit
instead of inferring hidden SQL or broadening access.

## Namespace policies and optional resource checks

Read [security and resource checks](security-and-resource-checks.md) when changing
predicate-package policy, resource thresholds or a component's resource-check
option. Namespace thresholds alone do not opt components in. Preserve the stored
flag for ordinary DQL edits; use `versions.apply` / `set_resource_check` when the
user requests a change. Revalidate its returned source revision before release.

## Build and test

Create the component and its draft through components.create and versions.create.
Use versions.builder for typed graph changes, or versions.apply for authored DQL,
according to their advertised schemas. Preserve expectedSourceRevision on every
source change. Inspect the saved structure and SQL resources after applying it.
Do not invent builder operation payloads: inspect the operation schema or export
a small working component and adapt its declared contract.

For a source-only edit, inspect with `discoverColumns:false` first. This reads the
saved source without attempting database column discovery on possibly broken DQL.
Use the existing authored-source command shape:

```json
{
  "reportId": "example-reader",
  "versionNo": 1,
  "command": {
    "kind": "set_dql",
    "expectedSourceRevision": 3,
    "payload": {"authoredDql": "<complete corrected DQL source>"}
  }
}
```

Replace the example identifiers and revision with readback values. The field is
`authoredDql`, not `dql` or `source`. After saving, verify the returned source and
revision, then pass that revision to `versions.validate`. A successful save is not
a validation result or publication. If column discovery was requested and fails,
preserve its error and use source-only inspection to repair the source; do not
label the failed discovery as passed.

Maintain one checkpoint: namespaceId, reportId, versionNo, sourceRevision and
state. An edit to draft v1 revision 3 produces draft v1 revision 4, not published
v4. Update that checkpoint from every successful mutation response. A metadata
ETag does not replace expectedSourceRevision. Clone or create a draft when the
selected version is published; do not mutate published source. On a timeout,
read the saved version before retrying because the mutation outcome is unknown.

Declare bindings, required verified scope, optional predicates and output roles.
Use resources.upsert_file when embedded SQL belongs to the component resource tree;
keep resource paths relative to the version. Re-inspect computed output columns.

For `WithEnum`, supply complete JSON scalar literals: `.WithEnum(3,1,2)` for
integer choices or `.WithEnum("low","high")` for strings. SQL-style single-quoted
strings are not this declaration's grammar. Preserve the source scalar type and
declaration order; numeric strings are not numeric IDs. Existing `choice` metadata
describes allowed values, not a value-to-label transformation or an expanded
`{value,label}` output. For an output-shape change, inspect native codecs/transfer
mapping or a source projection before introducing another enum/format mechanism.

Use versions.test_view / preview.execute and the cube-specific test facilities
advertised by this build. connectors.test_sql is an exploratory SQL execution,
not a column-discovery substitute. Its timeout is explicit: 0 disables Studio's
deadline, omitted defaults to 60 seconds. Results support limit/offset paging;
LIMIT or a page-size control does not bound BigQuery scan cost. Bound fact dates
and authorized entity scope before testing. Start with metadata discovery, then
run only the queries the user authorized.

## Validate, release, verify

1. Read the latest draft version and source revision. `pending` compile status
   means validation is needed. Invoke versions.validate for that exact revision;
   inspect diagnostics, including skipped checks, and resolve failures.
2. Read the selected namespace configuration and runtime/connection metadata.
   Namespace `mcpEnabled` must be true and the listener ready. Enabling component
   or cube MCP exposure alone does not start it. If configuration changes are in
   scope, update with the current etag; otherwise describe the required change.
   UI equivalent: Namespaces → select namespace → Edit → Enable MCP → Save.
3. Confirm the component tool name and desired cube exposures. Invoke
   publications.publish (UI: Release) on the validated version with its current
   expectedSourceRevision. Use a reason/requester when required by the schema.
4. Inspect publications.get, publication events and runtime.status until the
   requested generation becomes active or a concrete failure is reported. Do not
   poll a failed release indefinitely or keep republishing without a state change.
5. Prefer catalog.tools.list and catalog.tools.call on this management endpoint
   for current namespace tools, their schemas and authenticated execution after
   release. Follow cursors, pass the canonical namespaceId and inspect the nested
   result.isError. A client tool-search cache may predate publication; repeated
   searches of that cache are not runtime discovery. Alternatively obtain the
   namespace MCP connection using runtime.status host.mcpUrl and connection
   metadata as advertised. Connect with the intended authenticated identity, list
   tools, find the exact reader/cube tool, and make an authorized bounded call if
   requested. An initialize handshake alone proves transport, not tool access.

If activation reports namespace runtime unavailable, enable/save namespace MCP or
check the already-enabled runtime's status, then retry the same release version.
Do not insert publication rows or mark a generation active manually. On a stale
source revision, re-read and revalidate; do not blindly retry. If the runtime is
ready but reload fails, inspect the publication diagnostics: schema, missing
resources, credentials, tool collisions or port conflicts require their own fix.

Keep authenticated call failures distinct: 401 is credential verification, 403
is authorization, an unavailable namespace is runtime readiness, and an empty
filtered inventory can reflect visibility. Never use management API listing as
proof that a namespace business tool is live.

If validation/inspection returns a generic 500, preserve the failed operation,
component/version/source revision and any request ID. It is not evidence of an
invalid SQL clause, lack of authorization, or successful validation. Inspect the
saved version's diagnostics through the existing version-read operation; if no
cause was stored, report the missing diagnostic and stop publication. Do not
silently create another version or claim the correction was applied unless an
apply response and source readback confirm it. Give one concise progress update
per meaningful phase rather than repeating the plan after every discovery call.
