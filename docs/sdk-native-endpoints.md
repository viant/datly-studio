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
remaining operations. In authenticated mode, exact `acl.list` and
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

The `acl.list` candidate is in progress in the working tree. Its generated
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
`acl.list` and `reports.list`; expanding it to every public SDK route remains migration work.
The generator scopes its input to native SDK routes because broad static
control-plane OpenAPI includes unrelated routes with unresolved dynamic
status schema fields.

AI Studio's private report SDK follows the same endpoint rule within AI
Studio. This does not move private report code or product names into
Datly Studio.
