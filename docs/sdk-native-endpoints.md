# Native Datly SDK endpoints

Status: all 65 public SDK operations have generated native Datly HTTP/MCP
routes as of 2026-09-26. The script
`node scripts/check-native-sdk-coverage.mjs` measures OpenAPI paths (65/65);
`TestSelectedStudioStaticComponentsBootstrapTogether` boots the configured
host and checks one selected native route and one live MCP tool per path,
with no duplicate or undeclared `studio.sdk.*` tools.
That route-count gate does not prove every deployment configuration. Static,
authoring, and dynamic predicate catalogs now share the trusted
`STUDIO_PREDICATE_PACKAGES` package-path allowlist; native handlers discover
only linked `predicate.Handler` types in those packages. A deployment must
link the selected packages in its binary. Direct-browser ACL acceptance also
remains separate from route coverage. A configurable user-info facts adapter
can validate the
browser ID token and resolve account-scoped roles/features, but deployed
identity-provider and tenant configuration must be verified before it replaces
the dedicated ACL-token path.

## Contract

The Studio UI calls the Studio SDK. Each backend SDK operation is a public
Datly v1 component endpoint. The same transcribed component supplies:

- the stable SDK HTTP path and DTO-shaped request/response;
- generated OpenAPI documentation;
- an explicitly declared MCP tool;
- the same trusted identity binding, typed predicates, validation and error
  behavior regardless of whether HTTP or MCP invokes it.

One-to-one SDK operations use their actual reader or writer as that public
component. Do not add a Go custom handler that merely forwards to it. A public
custom-handler component is for genuine workflows or policy orchestration
(publication, validation, staged transactions, etc.); it may invoke internal
transcribed components. The internal `store_*` components are not public
HTTP/MCP tools.

## Migration history and remaining deployment gaps

The following inventory describes the migration path; it is not a current
list of missing native routes. `cmd/studio-api/main.go` still mounts
`sdk/httptransport.Gateway` at `/v1/studio/sdk/` for its compatibility
transport. In authenticated mode, exact `acl.delete`, `acl.list`, `acl.upsert`,
`authorization_predicates.create`, `authorization_predicates.delete`, `authorization_predicates.get`,
`authorization_predicates.list`, `authorization_predicates.types`, `authorization_predicates.update`,
`connectors.activate`, `connectors.create`, `connectors.delete`, `connectors.disable`, `connectors.get`, `connectors.list`, `connectors.schemas`, `connectors.table`, `connectors.tables`, `connectors.test`, `connectors.test_sql`, `connectors.update`, `namespaces.create`,
`namespaces.delete`, `namespaces.get`, `namespaces.list`, `namespaces.update`,
`preview.execute`, `publications.get`, `publications.events.list`, `components.create`, `components.get`, `components.list`, `components.update`,
`versions.apply`, `versions.builder`, `versions.create`, `versions.get`, `versions.list`, `versions.inspect`, `versions.load_dql`, `versions.load_archive`, `versions.export_dql`, `versions.validate`, `versions.test_view`, `versions.test_relation`, `versions.test_compose`,
`versions.descriptor`, `versions.download`, `versions.warmup`, `versions.warmup_get`,
`versions.warmup_list`, and
`resources.get`, `resources.upsert_file`, `resources.delete_file`,
`resources.upsert_folder`, `resources.delete_folder`, `resources.upsert_skill`,
and `resources.delete_skill` routes forward
to their static Datly components instead.
The SQL transport now calls many transcribed components, but that does not
make those SDK HTTP routes Datly components. Most UI calls in
`ui/src/studioApi.js` now use generated native clients. The Studio SDK declares
62 `sdk.Operation*` operations (including the separately declared
`versions.download`) plus the three `access.*` operations.
The release coverage check now measures 65 generated native paths out of 65
public SDK operations. Its earlier 58/65 and seven-route inventory was a
migration snapshot, not the current state.
Native `versions.validate` uses a private edit guard, an exact source-revision
check, the shared Datly runtime compiler and an internal optimistic status
writer. Signed HTTP/direct/BFF MCP tests cover valid and invalid DQL,
delegated-editor redaction, run-only denial, stale revision and the Docker
wide reader. Native `versions.test_view`, `.test_relation`, and `.test_compose`
share the verified `can_run` guard and configured DB capability with preview.
They execute Datly's actual transient view, relation and derived cube-compose
paths; SQLite signed HTTP/direct/BFF MCP tests cover root rows, attached-child
evidence and composed data. The Docker wide view also passed native HTTP and
both MCP paths.

`authorization_predicates.types` invokes a selected internal global-publisher
guard with the verified JWT subject before returning the linked predicate
catalog. Local signed HTTP, direct MCP, authenticated BFF MCP, denied caller,
OpenAPI, and private-child checks pass.
Native `authorization_predicates.get` and `.list` reuse that guard and a
selected internal Datly store reader, with the same DTO projection as the
generic SDK. The list retains default/capped paging and exact query/status
filtering. Local signed HTTP/direct/BFF MCP tests cover linked and unlinked
rows, denied callers, missing names, pagination, and child-route isolation.
Native `authorization_predicates.create` uses the same publisher guard, exact
linked-type validation, and an internal Datly insert writer. It binds owner,
status, etag and timestamps server-side; signed HTTP, direct/BFF MCP, duplicate,
unlinked-type and denied-caller tests pass.
Native `authorization_predicates.update` and `.delete` finish that SDK family.
Both use the selected internal optimistic PATCH writer. Update projects the
exact SDK DTO after ETag advancement; delete soft-deletes with a 204 response.
Signed HTTP/direct/BFF MCP tests cover edits, soft-delete persistence, stale
conflicts, invalid values, denial and private writer isolation.
Native `connectors.schemas`, `.tables`, and `.table` load connection material
only after verified-principal read authorization and an independently scoped
private catalog lookup. The static host now links SQL metadata adapters.
SQLite HTTP/direct/BFF MCP checks pass, and an opt-in Docker run found all
20 hierarchy views and 60 wide-table columns over native HTTP and MCP.
Connection material is absent from the responses. Secret-reference resolution
uses the shared Scy default when a server-held reference is configured;
signed HTTP/direct/BFF MCP checks with a local secret resource confirm that
neither the resolved DSN nor secret path reaches a response. Custom
deployment-specific resolver injection remains separate from this default.
Native `connectors.test` uses an edit-scoped guard, probes draft or active
connectors with the host-linked SQL driver, and persists the result through
the internal optimistic status writer without advancing the configuration
ETag. Driver and secret errors are returned as bounded failure evidence; the
generic SDK fallback now applies the same redaction. SQLite and Docker MySQL
signed HTTP/direct/BFF MCP checks pass, including denied editors and failed
probe responses.
Native `connectors.test_sql` finishes the connector SDK family. It requires
verified edit access and an active connector, then compiles browser SQL as a
bounded transient Datly reader. Its source-free runtime materialization uses
an explicit module authority, so a deployed binary does not need `go.mod`.
SQLite and Docker MySQL signed HTTP/direct/BFF MCP tests pass, including
invalid SQL and denied/draft connector cases. Driver details and DSNs are not
returned on execution failure. Its sibling Datly `RuntimeContractsInModule`
dependency is included in the current immutable Datly pin.
Native `preview.execute` authorizes the exact report with a private `can_run`
guard, borrows the configured Studio DB capability, and invokes the shared
Datly preview engine for the requested version under row/byte/time budgets.
SQLite HTTP/direct/BFF MCP tests cover owner, delegated runner, denial,
missing version and private-child isolation. An opt-in Docker test returns
two complete 60-column rows through HTTP and MCP. Secret-reference-only
connectors are now accepted by the preview connector reader instead of
blocking unrelated reader previews.
These handlers currently project `linked` against Studio's default static
predicate catalog. An embedding application that adds
`host.Config.PredicatePackages` still needs that catalog injected into its
static handler runtime for extension-type parity; the route count alone does
not prove that separate deployment case.
The generic `versions.validate` and `versions.builder` paths now redact their
returned version and source-bearing diagnostic text for a delegated editor
without `can_use_dql`; native `versions.get` and `versions.list` make the same
SQL-side diagnostic redaction. This closes response leaks while those two
Reader Builder still awaits a native route.
The DQL tree has public readers with `$mcp` declarations, but they do not
provide complete SDK-operation parity, and a declared MCP directive alone
does not prove a route is mounted or authorized.
The static-host inventory test now compares every DQL MCP tool declaration
with its selected package and linked route metadata. The older
authorization-predicate and publication-event readers are explicit unselected
exceptions; their declarations are not treated as serving tools.
An eager combined-host test now boots the exact `datly.yaml` package set,
checks every generated native SDK OpenAPI path and matching MCP declaration,
executes authenticated namespace create/list on that host, and confirms an
internal child route is not public. This is local RSA-fixture evidence, not a
deployed identity-provider acceptance test.
The static host now has a dedicated loopback MCP listener. A network test
checks the full native SDK tool catalog, signed `tools/call`, denial without a
bearer, and the same call through the authenticated BFF's separate
`/v1/studio/sdk-mcp/` proxy. Its protected-resource metadata URL is a local
default and must be set to the public URL in deployment.

The existing `studio/reports/store_catalog` is a concrete reason not to
expose server-only components directly: its input includes caller-bindable
`subject` and `scoped` parameters; the SDK wrapper currently supplies
these from the verified principal. A public replacement must bind principal
scope from the authenticated context and fail closed. Merely adding `$mcp`
to this internal DQL would permit a caller to request an unscoped read.
The public `studio/reports/reader` binds its required
`Auth` component output into the typed `ReportCatalogRead` predicate,
with no caller-settable `subject` or `scoped` field. Its SQL retains only
the deleted-row condition and declared filter groups. SQLite tests cover
owner scope, ACL grants and revocation, and its inline MCP declaration
compiles to `studio.sdk.components.list`. The native POST body and page output
match the SDK's filters, default and capped limits, ordering, ignored legacy
field/order selectors, derived owner package, and authorization scope.

`components.create` is a native linked-handler workflow because creation checks
an authorized active connector, requires the verified owner's active namespace
(creating `general` when absent), and inserts a draft report with server-owned
identity scope, state, etag, and timestamps. The namespace lookup and report
insert children are selected as internal-only Datly components. The handler
projects the inserted row within the mutation transaction, avoiding a
read-after-write visibility race. Combined-host tests cover persistence,
forged owner/identity fields, inactive connectors, missing namespaces,
duplicate conflict, unauthenticated denial, direct HTTP/MCP, authenticated
BFF MCP, and private child-route isolation. OpenAPI and generated Go/browser
clients include its exact SDK path.

`components.update` is a native linked-handler workflow. A private typed
`ReportEdit` guard precedes the metadata read and optimistic config writer;
the handler rejects caller-controlled component identity, validates an active
owned namespace, and checks an active connector plus a private version head
before a default-connector change. Once any version exists, that connector
change returns 409 and must go through Reader Builder. The handler projects
the writer's result in the transaction, avoiding a post-write visibility
race. Combined-host tests cover HTTP, direct/BFF MCP, owner and edit denial,
forged identity, stale etag, invalid namespace, unversioned connector change,
versioned-connector conflict, persistence, and private child isolation.

## Migration rule

For each SDK operation:

1. Identify its current SDK DTO, authorizer, transcribed backing and
   transaction semantics. Record whether it is one-to-one or a workflow.
2. Transcribe a public component at the stable SDK path with SDK-compatible
   input/output. Bind identity from trusted header/context, not body/query.
   Keep resource and entity authorization in native inputs and typed
   predicates; never copy a complex SDK WHERE clause into authored SQL.
3. Declare the MCP tool on that component, with a stable name and description.
   Include the component in the actual Datly runtime's public exposure set;
   do not create a parallel MCP adapter calling `sdk.Transport.Invoke`.
4. Prove SDK HTTP, OpenAPI and MCP all resolve to the same component. Test
   owner/grant/denial/revocation, invalid input, paging, exact revision and
   conflict behavior on both protocol paths. Assert internal components are
   absent from public route, OpenAPI and MCP catalogs.
5. Route the existing UI SDK operation to the component and remove the
   generic gateway case only after equivalent behavior is verified. Keep
   transport compatibility during migration, but do not call the migration
   complete while any UI-used operation still runs through the dispatcher.

The first security-sensitive reader migrated is `components.list`. Its generic
SDK wrapper still invokes the transcribed `store_catalog` reader with a verified
principal and scoped predicate. The public component keeps that behavior
without exposing `subject` or `scoped` to the caller. The private
`store_catalog` remains available for server-owned operations.

`connectors.list` now uses the public connector reader and typed `ConnectorRead`
predicate over the compiled view. Its embedded query contains no owner/ACL
SQL and never selects DSN or secret values. The native POST body and page
match the SDK's filters, bounds and ordering; SQL-derived configuration flags
and JSON options retain the SDK wire shape. SQLite tests cover 137-row paging,
owner/delegated scope, revocation, deleted-report grants, HTTP, OpenAPI and
MCP redaction.

`connectors.get` has its own native reader at the SDK POST path. It returns
one direct connector DTO or 404, derives only `dsnConfigured` and
`secretConfigured` from server-held material, and uses the same typed ACL
predicate. Tests cover owner and delegated access, missing/denied/revoked
identities, HTTP/MCP/OpenAPI wire parity, and secret redaction.

`connectors.create` is a native Datly mutation writer with the SDK's flat
request body. Its input initializer binds owner to the verified JWT and sets
draft status, options default, etag and timestamps server-side. The public
response uses a typed wire projection with `dsnConfigured` and
`secretConfigured` flags, never returning DSN or secret-reference material.
SQLite tests cover persistence, forged-owner and invalid-input denial,
duplicate conflict, HTTP/MCP redaction and exact OpenAPI input/output shapes.

`connectors.disable` is a native linked-handler workflow. A private typed
`ConnectorAccess` predicate checks the verified subject's edit scope before
the public metadata reader and internal optimistic status writer run. Its
response preserves configuration flags and test metadata, never DSN or secret
references. Combined-host tests cover owner and delegated view+edit success, non-owner denial, stale
etag conflict, persisted status/etag with unchanged server-held secrets,
direct HTTP/MCP, authenticated BFF MCP, and private child-route isolation.
The browser uses the generated client at the exact SDK path.
`connectors.activate` uses that same guard, redacted DTO, and internal status
writer, with a server-owned `activate` operation. It requires a persisted
passing connectivity test both before the write and in the writer lifecycle;
untested connectors return 400. Combined-host tests cover owner activation,
non-owner denial, stale 409, direct and BFF MCP, persisted active status,
unchanged server-held secrets, and exact OpenAPI/client routing.
`connectors.delete` is a native linked-handler soft-delete workflow. It checks
the verified editor with the private typed guard, counts references through
an internal reader, then invokes the internal optimistic status writer.
Referenced connectors return 409; a stale write preserves the generic SDK's
404 behavior. HTTP returns 204 with no body, reflected in OpenAPI and the
generated void browser call. Combined-host tests cover direct/BFF MCP,
unauthorized denial, persisted archival, and private child-route isolation.

`connectors.update` is a native linked-handler workflow. Its private typed
edit guard precedes an internal catalog read that supplies the full current
configuration; the handler merges the nested SDK patch and invokes the
internal optimistic config writer. Rotating driver, DSN, secret reference, or
options resets status to draft and clears probe evidence, while a
description-only edit preserves them. The direct response includes only
configuration flags and metadata, never connection material. HTTP, direct
and BFF MCP tests cover both update classes, non-editor denial, stale etag,
persistence, redaction, and private child-route isolation.

`namespaces.list` now uses the generated Datly reader with a typed
`NamespaceRead` predicate. Its embedded SQL contains no caller-controlled
owner or ACL scope. The SDK-shaped POST response applies the same search,
status, page bounds and default ordering; SQLite tests cover owner/delegated
access, deleted-report grants, revocation, HTTP, OpenAPI and MCP.

`namespaces.get` uses a dedicated direct-response reader at the SDK POST path.
It returns a single authorized namespace or 404, with the same typed scope.
HTTP, MCP, OpenAPI, missing/denied identity and revocation tests cover it.

`namespaces.create` is the first native SDK mutation writer. Its generated
Datly writer accepts the flat SDK body, then `Init` binds owner to the verified
JWT and overrides caller-supplied status, etag, and timestamps. HTTP and MCP
share that contract; tests cover first namespace creation, forged owner,
invalid name, duplicate conflict, persistence, and the direct OpenAPI response.
Datly now maps structured SQLX duplicate-key errors to HTTP 409 without
publishing database details. A typed custom-JSON wire declaration keeps the
writer's flattened SDK response and generated OpenAPI in agreement.

`namespaces.update` is a native linked-handler workflow for nested SDK patch
input. It reads only the verified owner's namespace through a private Datly
reader, applies title/description/status validation, and invokes an internal
optimistic-etag writer. The handler projects the updated row without a
read-after-write transaction visibility race. Combined-host tests cover HTTP,
direct and authenticated BFF MCP, stale-etag 409, non-owner/missing 403,
invalid title 400, and private child-route isolation. Its exact BFF path and
generated Go/browser clients are selected.

`namespaces.delete` is a native linked-handler soft-delete workflow. It checks
the verified owner and exact etag, invokes a private usage-count reader before
the internal writer, and archives only an unreferenced namespace. HTTP returns
204 with no body; the generated OpenAPI response is corrected to 204 and the
browser adapter retains the SDK's void return. Combined-host tests cover
referenced and stale 409s, non-owner 403, persisted archival, direct HTTP,
direct and authenticated BFF MCP, and private child-route isolation. Public
error messages use explicit safe payloads rather than JSON null.

The generic SDK redacts authored/generated DQL for subjects without
`canUseDql` on both `versions.get` and `versions.list`. The older static version
reader carries full source, so its typed `ReportVersionRead` predicate requires
`can_use_dql`. Routing that source reader to an SDK path would widen access.
Its existing list and direct-version tests cover viewer denial and revocation
of `can_use_dql`.

`versions.create` is a native linked-handler workflow. A private typed report
edit guard, capability reader, version-head reader and insert writer preserve
actor binding, draft numbering, authoring-mode validation, spec hashing,
source revision 1 and initial compile state. It projects the inserted row
within the mutation transaction, and redacts authored/generated DQL from a
delegated editor without `can_use_dql` while keeping stored source intact.
Combined-host tests cover HTTP, direct/BFF MCP, forged actor, invalid mode,
delegated edit/redaction, persisted versions, and private child isolation.

`versions.load_dql` is a native linked-handler import workflow. It authorizes
editing before parsing bounded UTF-8 DQL, then joins the imported version,
`main.dql` resource and report draft-pointer writers under one managed
transaction. The generic and native paths share bundle spec-hash and resource
namespace derivation. A forced draft-pointer failure proves that version,
resource and pointer changes all roll back. HTTP, direct/BFF MCP, malformed
DQL, non-editor denial, monotonic version numbers, persistence and private
child isolation pass combined-host tests. Both native and generic SDK responses
now redact authored/generated DQL for delegated editors without `can_use_dql`;
the source remains stored. The generated browser client uses the exact path.

`versions.load_archive` uses the same authorized managed transaction after
the bounded ZIP/TAR/TAR.GZ parser has rejected unsafe paths, links, oversize
content, and ambiguous root selection. An explicit `entryDql` selects one
root while all archive files remain version resources. Combined-host HTTP,
direct/BFF MCP, delegated-editor redaction, three-file persistence, version
numbering and draft-pointer tests pass. The browser's generated client retains
its base64 archive request and selected entry fields.

`versions.inspect` is a native linked-handler workflow over the exact
authorized version, active connector catalog and Datly Reader Builder. The
generic and native paths share an inspection projector. A view grant without
`can_use_dql` now receives a metadata-only structure (status and view names)
and diagnostic codes without source-bearing messages; previously the
Reader Builder structure leaked SQL through component and view fields despite
an empty top-level DQL. Owner HTTP/direct MCP, delegated HTTP/MCP redaction,
authenticated BFF MCP, OpenAPI, and private source-reader isolation are
covered by tests. The generated browser client uses the exact SDK route.

`versions.apply` is a native linked-handler edit workflow. The private typed
report edit guard precedes full version/catalog and capability reads. The
handler validates `set_spec`, `set_dql` and `set_sql` commands, compares the
exact source revision, and invokes the internal optimistic version writer.
The response projects the writer's advanced revision without a post-write
read; delegated editors without `can_use_dql` receive no authored/generated
DQL. The generic SDK edit response now applies the same redaction. HTTP,
direct/BFF MCP, denial, invalid kind, stale revision, all three edit kinds,
persistence and private-child isolation are tested.

The new `studio/report_versions/get` component uses metadata-level view scope
and conditionally redacts DQL in its embedded SQL. Owner, delegated viewer,
delegated DQL author, denial, missing version, HTTP, MCP, OpenAPI, and grant
revocation have focused SQLite coverage. The authenticated BFF forwards the
exact native route, and generated Go and browser clients include it;
development mode retains the generic SDK fallback. The browser SDK exposes
`getVersion` through the generated client, though no current workspace calls
it. Broader deployed identity-provider acceptance remains separate.

`versions.list` now has a separate native metadata reader with the SDK's nested
filters and bounded page response. It scopes through `ReportVersionMetadataRead`
and redacts authored/generated DQL in the SQL projection unless the verified
subject has `canUseDql`. Focused SQLite tests exercise owner, delegated view
and DQL grants, denial, revocation, filtering, HTTP, generated OpenAPI, and MCP
invocation. The authenticated BFF forwards its exact route, and generated Go
and browser clients include it. Development mode retains the generic fallback.

`versions.export_dql` is now a native direct-response reader. Its typed
`ReportVersionRead` predicate requires the verified subject's DQL capability;
no source field can be supplied by the caller. The SQL projection preserves
the SDK precedence of generated DQL, authored DQL, then authored SQL, and
reports `complete: false` when all are empty. SQLite tests cover precedence,
viewer denial, missing version, grant revocation, HTTP, MCP, and OpenAPI. The
authenticated BFF and generated Go/browser clients include the exact route.

`versions.descriptor` is a separate native reader with the SDK's direct
component/types/resources JSON envelope. It uses the typed version metadata
view predicate, preserving delegated view access without granting DQL export.
Focused SQLite tests cover authorized viewer output, denied/missing identity,
grant revocation, HTTP, MCP, and OpenAPI. The authenticated BFF and generated
clients include its exact route.

`versions.download`, which the UI uses for component transfer, is now a
native Datly linked-handler workflow. It invokes the source-bearing version
reader under typed `ReportVersionRead` before reading resource files. The
resource child component is selected as `internal=true`, with no public HTTP,
OpenAPI, or MCP exposure. Both the native handler and generic SDK fallback use
one archive assembler for path validation, SQL delegation and deterministic
ZIP contents. A private aggregate preflight rejects more than 2,000 resource
files or 32 MiB of stored resource content before the content reader runs;
the reader has a 2,001-row safety cap, and assembly also enforces a 4 MiB DQL
source limit and 16 MiB ZIP limit. Focused SQLite tests exercise an importable archive, view-only
denial, missing/revoked DQL grants, unauthenticated denial, HTTP, MCP, and
OpenAPI. Generated clients and the authenticated BFF use the exact route.

`resources.get`, used by the authoring UI, is a native linked-handler workflow.
It first invokes the metadata-scoped `versions.get` reader and only then invokes
three internal-only snapshot readers for files, folders, and skills. The native
and generic SDK paths share row-to-DTO projection code; the public DTO carries
the same capability-redacted version. SQLite tests cover 137 ordered files,
folder/skill ordering, view-only access, denial, revocation, HTTP, MCP,
OpenAPI, and internal-route isolation. Its exact BFF route and generated
browser/Go clients are selected.

The six resource mutations use one native endpoint invocation and the existing
generic Datly version, file, folder, skill, and namespace writer components.
Those child readers and writers share the endpoint's transaction; the writer
buffer flushes inside that transaction before a later reader runs. The linked
store routes are internal and have no MCP tools. The combined-host test covers
file, folder, and skill upsert/delete through HTTP, direct MCP, and authenticated
BFF MCP, plus revision advancement and rollback after an invalid file mutation.
OpenAPI and generated Go/browser clients expose the
same six public DTO contracts.

`versions.builder` is a native linked-handler workflow using the existing
Reader Builder service and generic version/report PATCH writers. It verifies
the edit guard, applies the expected source revision, and keeps both writes in
the endpoint transaction. The combined host checks inspect, package mutation,
version/report revision changes, stale conflict metadata, direct MCP, and
authenticated BFF MCP. Its OpenAPI and generated clients use the same command
DTO as the former generic SDK route.

`versions.warmup` is a native linked-handler workflow. It verifies the current
publisher and exact version through private Datly components, performs
server-owned active-key deduplication and durable accepted-run insertion, then
starts the bounded Datly-backed executor after the transaction commits. Its
HTTP, direct MCP, authenticated BFF MCP, denied-caller, deduplication, and
accepted-to-terminal tests run on the combined static host, including a
cache-enabled run that reaches `completed` with its planned cases. The browser uses
the generated client and exact BFF mount.

`versions.warmup_get` is a native direct-response reader for durable warmup
evidence. Its typed `WarmupRead` predicate requires the current owner or a
publish grant. The response projects audit fields, duration, target, and
diagnostics into the SDK DTO; tests cover owner/delegated publisher access,
view-only denial, missing run/report, revocation, HTTP/MCP/OpenAPI parity,
and direct JSON without an internal row wrapper.

`versions.warmup_list` is a native linked-handler workflow, not a read-only
substitute. A private typed `ReportPublish` guard authorizes the requested
report before the handler invokes internal expired-run reader and CAS writer
components globally, preserving the generic SDK's 5-minute recovery and
100-row batching. It then reads the requested page through an internal reader;
the generic and native paths share SDK row projection. Tests cover 106 expired
runs across batches, viewer denial before any recovery, owner/publisher scope,
recent-run preservation, paging, invalid input, revocation, HTTP/MCP/OpenAPI,
and internal-route isolation.

The global publish predicate used by authorization-predicate and runtime
control readers now matches the SDK's live-report rule: current report owners
and subjects with a publish grant on a non-deleted report qualify. A stale
grant on a deleted report does not. SQLite tests cover all three cases.
`authorization_predicates.get/list` still need a native output contract for
`linked`, which the SDK derives from its configured predicate package catalog,
and for the decoded SQL alias/columns metadata. Exposing the older reader's
raw `sql_scope_json` field would not match the SDK DTO; do not route it to
those SDK paths without that server-owned projection.

`components.get` has a dedicated native reader at the SDK POST path. It requires
body `id`, binds the same trusted auth context and typed catalog predicate,
returns the report DTO directly, derives `ownerPackage` server-side, and
returns 404 for missing or inaccessible reports. SQLite contract tests cover
HTTP, OpenAPI, MCP, owner, delegated viewer, denial and revocation.

`publications.get` uses one direct-response Datly reader at the SDK POST path.
Its output matches the SDK publication DTO; required body `reportId` is scoped
through the typed `PublicationRead` predicate. Owner and delegated viewer
access, denial, revocation, missing rows, HTTP, MCP and OpenAPI are covered by
a preseeded SQLite contract test. The authenticated BFF proxies the exact
route, and browser `getPublication` uses the generated OpenAPI client.

`publications.events.list` now has a native Datly reader at the SDK POST path.
The nested SDK `input` object is a linked, typed body value; `Init` applies the
same validation and page bounds before its SQL executes. The output is an SDK
page with stable newest-first ordering. Its typed predicate requires the
report's current owner, so neither a delegated publisher nor a former owner
can read the trail. SQLite tests cover transfer, defaults, filters, paging,
invalid input, HTTP, MCP and OpenAPI. The BFF forwards the exact authenticated
route, and the browser uses the generated OpenAPI client. The older
control-plane event reader remains out of the static host package list.

The `acl.list` reader is implemented. Its generated
Datly component now uses `POST /v1/studio/sdk/acl.list`, includes the ACL
`etag`, and declares `studio.sdk.acl.list` as an MCP tool. A focused SQLite
contract test exercises the Datly HTTP route, generated OpenAPI path, MCP
invocation, owner/editor visibility, denied subjects, and ownership changes.
In authenticated mode, the BFF now proxies this exact SDK path to the static
Datly host with the bearer stored in its HttpOnly session. Development mode
continues to use the generic SDK gateway and keeps ACL management disabled.
The generated input tags `reportId`, so the native MCP argument and HTTP JSON
body use the same name. The product owner chose owner-only ACL listing. The
SDK list path checks report ownership, and the native `ACLRead` predicate
authorizes through the generated grant reader before returning rows. Owner,
delegated editor, and viewer cases are covered on both paths. The browser's
`listACL` call now uses the generated client from this native OpenAPI document.

`acl.delete` is a native linked-handler workflow. It invokes the owner-only
`ACLRead` reader for the exact grant and expected etag, then the internal
optimistic ACL writer. HTTP and OpenAPI use 204 with no body; stale requests
return safe `expectedEtag`/`currentEtag` metadata. Combined-host tests cover
owner HTTP, direct/BFF MCP, non-owner denial, stale and missing entries,
persistence, and private writer isolation. This test exposed a Datly standalone
gap: reader predicates requiring a connector capability did not receive one
on the combined host. The sibling Datly host now binds that capability for
reader components, with a regression test.

The ACL writer's canonical DQL now explicitly casts its pointer-valued key
and capability fields, so the current transcriber regenerates the internal
writer package without a hand-maintained router tag.

`acl.upsert` is a native linked-handler workflow using that same private
PATCH writer's insert-or-update semantics. The handler validates the user
subject and capability implications, invokes owner-scoped `acl.list` for the
exact grant, checks its etag for updates, and returns a direct SDK ACL DTO.
HTTP, direct/BFF MCP, non-owner denial, invalid capabilities, create/update,
stale conflict metadata and persistence are tested on the combined host. The
generated browser client preserves expected/current etag details.

`scripts/generate-studio-sdk.sh` reproducibly exports the Datly route
contract and generates Go and browser clients. `runtime.status` is a
publisher-scoped native route and MCP tool. Its handler uses the same
stored-generation projection and trusted runtime-admin probe as the generic
SDK. The static host reads `STUDIO_RUNTIME_ADMIN_TOKEN` and optional
`STUDIO_DYNAMIC_HTTP_URL` from its own environment; callers cannot supply
either value. If the probe is unconfigured or fails, an active generation is
reported with an unavailable host rather than as live-ready. `access.context`,
`access.get`, and `access.replace` are native routes and MCP tools. The static
host requires `STUDIO_ACCESS_ISSUER`, `STUDIO_ACCESS_AUDIENCE`, and
`STUDIO_ACCESS_PUBLIC_KEY_FILE` to independently verify the ACL bearer; the
policy reader and writer are private linked children of the same Datly
transaction. The forwarded session bearer must satisfy both Studio's JWT
validator and the ACL issuer/audience/key check; the native route never
accepts identity claims from its JSON body. `publications.publish`,
`publications.rollback`, and `publications.unpublish` are native Datly routes
and MCP tools that call the existing lifecycle service. The static host must
provide `STUDIO_RUNTIME_ADMIN_TOKEN` and optional `STUDIO_DYNAMIC_HTTP_URL`.
Their lifecycle owns multiple committed database phases around the dynamic
reload; it does not join the Datly endpoint's one-commit database unit.

The document currently covers
`access.context`, `access.get`, `access.replace`, `acl.delete`, `acl.list`, `acl.upsert`, `connectors.activate`, `connectors.create`, `connectors.delete`, `connectors.disable`, `connectors.get`, `connectors.list`, `connectors.update`,
`namespaces.create`, `namespaces.delete`, `namespaces.get`, `namespaces.update`,
`namespaces.list`, `publications.get`, `publications.events.list`, `publications.publish`, `publications.rollback`, `publications.unpublish`, `runtime.status`, `components.get`,
`components.create`, `components.list`, `components.update`, `versions.apply`, `versions.builder`, `versions.create`, `versions.get`, `versions.list`, `versions.inspect`, `versions.load_dql`, `versions.load_archive`, `versions.export_dql`,
`versions.descriptor`, `versions.download`, `versions.warmup`, `versions.warmup_get`,
`versions.warmup_list`, and
`resources.get`, `resources.upsert_file`, `resources.delete_file`,
`resources.upsert_folder`, `resources.delete_folder`,
`resources.upsert_skill`, and `resources.delete_skill`.
The generator scopes its input to native SDK routes because broad static
control-plane OpenAPI includes unrelated routes with unresolved dynamic
status schema fields.

AI Studio's private report SDK follows the same endpoint rule within AI
Studio. This does not move private report code or product names into
Datly Studio.
