# Native Datly SDK endpoints

Status: migration requirement, not yet implemented end to end (2026-09-25).

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

## Current gap

`cmd/studio-api/main.go` mounts `sdk/httptransport.Gateway` at
`/v1/studio/sdk/`; the gateway dispatches to `sdk.Transport.Invoke` for the
remaining operations. In authenticated mode, exact `acl.list`,
`connectors.get`, `connectors.list`, `namespaces.get`, `namespaces.list`,
`publications.get`, `publications.events.list`, `reports.get`, and
`reports.list` mounts forward to their static Datly components instead.
The SQL transport now calls many transcribed components, but that does not
make those SDK HTTP routes Datly components. Most UI calls in
`ui/src/studioApi.js` still target the generic dispatcher. The Studio SDK declares
62 `sdk.Operation*` operations (including the separately declared
`versions.download`) plus the three `access.*` operations.
The DQL tree has public readers with `$mcp` declarations, but they do not
provide complete SDK-operation parity, and a declared MCP directive alone
does not prove a route is mounted or authorized.

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
compiles to `studio.sdk.reports.list`. The native POST body and page output
match the SDK's filters, default and capped limits, ordering, ignored legacy
field/order selectors, derived owner package, and authorization scope.

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

The first security-sensitive reader migrated is `reports.list`. Its generic
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

`namespaces.list` now uses the generated Datly reader with a typed
`NamespaceRead` predicate. Its embedded SQL contains no caller-controlled
owner or ACL scope. The SDK-shaped POST response applies the same search,
status, page bounds and default ordering; SQLite tests cover owner/delegated
access, deleted-report grants, revocation, HTTP, OpenAPI and MCP.

`namespaces.get` uses a dedicated direct-response reader at the SDK POST path.
It returns a single authorized namespace or 404, with the same typed scope.
HTTP, MCP, OpenAPI, missing/denied identity and revocation tests cover it.

Before migrating `versions.get` and `versions.list`, the generic SDK now
redacts authored/generated DQL for subjects without `canUseDql`. The older
static version reader carries full source, so its typed `ReportVersionRead`
predicate requires `can_use_dql`. Native version DTOs still need a reader
contract that can return metadata to viewers while withholding structural
DQL; exposing the current static reader at SDK paths would widen access.
The static reader's list and direct-version routes now have focused tests for
viewer denial and revocation of `can_use_dql`. A native SDK replacement must
preserve both that source boundary and the generic SDK's redacted metadata
response for `canView`-only users. Routing the existing source reader to the
SDK path is therefore not a valid migration shortcut.

`reports.get` has a dedicated native reader at the SDK POST path. It requires
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
`scripts/generate-studio-sdk.sh` reproducibly exports the Datly route
contract and generates Go and browser clients. The document currently covers
`acl.list`, `connectors.get`, `connectors.list`, `namespaces.get`,
`namespaces.list`, `publications.get`, `publications.events.list`, `reports.get`,
and `reports.list`;
expanding it to every public SDK route remains migration work.
The generator scopes its input to native SDK routes because broad static
control-plane OpenAPI includes unrelated routes with unresolved dynamic
status schema fields.

AI Studio's private report SDK follows the same endpoint rule within AI
Studio. This does not move private report code or product names into
Datly Studio.
