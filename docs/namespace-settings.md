# Namespace security and query resource settings

Schema layout 27 stores typed policies in `namespace_settings`, keyed by
`(owner_id, namespace, setting_key)`. The namespace foreign key has no cascading
delete. `component_versions.resource_check_enabled` is the per-version opt-in.
Existing component versions default to false. Apply the explicit MySQL or SQLite
`027-namespace-settings` migration after layout 26; startup does not migrate stores.

The `security` value contains `predicatePackages` and
`requireSecurityPredicate`. A null package list inherits the trusted host catalog;
an empty list selects no external handlers. Paths must already be approved and
linked into the executable. Namespace selection removes unselected predicate types
while retaining codecs, dependency shapes and Studio's own authorization types.
It never downloads packages or compiles browser-provided Go code.
`authorization_predicates.types` accepts optional `namespaceId` and filters the
handler picker to that namespace's packages after verifying namespace access.
Omitting scope preserves global linked-type discovery for startup/recovery.

The `resourceLimits` value contains optional nonnegative `maxBytesProcessed`,
`maxReferencedTables` and `maxReturnedRows`. Omitted limits are unlimited. Bytes
and table counts apply to each bound query; returned-row checks apply to the main
reader result before it is returned. A dry run does not estimate SELECT row count.
Unknown or lower-bound byte estimates cannot satisfy an upper byte threshold.

## Native persistence

`namespaces.settings_get` and `namespaces.settings_update` are authenticated native
Datly components. Both require namespace owner or current manager permission.
The update accepts the current namespace ETag and only the supplied setting groups.
A generated PATCH graph updates the namespace token and settings together.
Omitted groups survive ordinary namespace saves and DQL regeneration. Invalid
settings, stale tokens and failed child writes leave the persisted graph unchanged.
A supplied group replaces that group's value, so callers changing one member
carry forward the other saved members. There is no independent setting revision.

`versions.apply` accepts `set_resource_check` with an explicit
`resourceCheckEnabled` boolean. The generated PATCH writer checks the source
revision, preserves source text, clears validation evidence and advances the
existing revision. Published versions cannot be edited. Draft creation and native
cloning retain the configured flag. `versions.get` and inspection expose it.

The authoritative DQL is under `dql/studio/namespace_settings` and
`dql/studio/report_versions/store_resource_check`. Generate readers with
`datly transcribe get` and writers with `datly transcribe patch` (POST for the
existing version-insert graph). The generated universal writer owns DML and
transaction completion; application hooks contain validation only.

## Main DQL validation hook

Trusted hosts may set `studio/host.AuthoringConfig.SourceValidator`; runtime hosts
set the matching `runtime/host.Config.SourceValidator`. A combined Studio host
passes the authoring callback into browser preview/validation too. The hook receives
component identity, exact source revision, package, authored main DQL and effective
main DQL. Effective generated source takes precedence where present. Nil preserves
ordinary behavior; an error rejects validation and execution/staging, including a
compiled-artifact cache hit.

When `requireSecurityPredicate` is true, Datly's compiled parameter metadata must
contain a handler from the owning namespace's selected packages on a required
input, or one explicitly applied when absent. An import alone does not satisfy
this requirement. Component publication rechecks the current namespace policy
before staging a generation. Exact component binding compilation applies the
same check before release bundle/channel creation; cached compiled contracts are
checked too. Other namespaces retain their own independent requirement flag.

The source hook remains optional and can additionally inspect actual application
in effective SQL. Its input includes the namespace requirement and package
selection; a null selection means the deployment's approved packages. Studio
uses native compiled metadata for declaration checks, not a second SQL parser.
Use `sourcevalidation.Rejection` for deliberate public diagnostics. Other errors
retain their cause for trusted callers and expose a generic policy rejection.

## Query execution

The native Datly reader's detached `WithQueryGuard` admits fully bound SQL at its
shared row-read boundary. Root queries, relation batches, partitions, derived
outputs and warmup retain native planning, selector, cache and transaction behavior.
Only component versions with `resource_check_enabled=true` attach this guard.
Disabled versions follow the ordinary native reader path and perform no resource
dry runs. For an opted-in component, Studio's guard performs the driver's metadata-only BigQuery dry run on the actual
source connection before permitting that statement. BigQuery handles physical
partition pruning and nested SQL within each submitted statement.

The driver extension exposes `DryRun` directly and through `database/sql.Conn.Raw`,
returning the API Job statistics without polling or fetching result rows. It reuses
the configured project, location, dataset, credentials and parameter conversion.

The new Datly and BigQuery capabilities are local source changes until released
and pinned by Studio. With older Datly, the compatibility path checks ordinary
single-root queries and rejects shapes it cannot fully admit. A driver lacking
dry-run statistics rejects enabled resource checks; it never silently executes.
Local integration tests use the three modules together and mocked BigQuery HTTP,
not a live cloud account. Namespace policy changes are read when a runtime build is
prepared; reload the namespace runtime to apply them to existing generations.

## UI

Namespace management offers **Security and resource limits** independently of the
ordinary namespace editor. Components offer **Check query resources against
namespace limits** and **Save resource check** for the loaded draft. Conflicts keep
unsaved form values visible. The shared Forge window sources serve native and
remote hosts; there is no duplicate settings screen.
