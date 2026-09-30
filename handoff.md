# Datly Studio handoff

Updated: 2026-09-30 (America/Los_Angeles)

## Current checkpoint — read this before the historical sections below

The rest of this file preserves a detailed development history through
2026-09-27. Its old ports, dependency hashes, pending-route counts, and
"immediate next goal" are historical evidence, **not current instructions**.
This section is the current source of truth for resuming the Forecasting Studio
work. No new branch was created; public Studio work is on `main`.

### What the user is building

- Keep public `viant/datly-studio` independent and reporting-free. It is the
  reusable Datly 1.0 SDK/UI authoring base; private `aistudio` will add Forge
  reporting separately.
- Author components, connectors, DQL/embedded SQL, predicates, skills, ACL,
  namespace configuration, and publication through UI/typed SDK flows. A
  namespace owns its components, resources, MCP exposure, and skills; private
  by default, visible to its owner and assigned roles, with explicit public
  visibility. Global connectors are allowed. Distinct namespaces may expose
  MCP on distinct ports.
- Forecasting is the only current business-data scope. Use fabricated
  `viant-e2e` BigQuery data and a local `ci_ads` MySQL fixture. Do not inspect
  production table rows. The UI graph must retain all seven Steward join views
  (Channel, Publisher, Forecastingsite, Carrier, Countryregion, Dma,
  Agegroup), 235 predicates, and the selected cube measures. Any publisher
  entity authorization requires an authoritative publisher-grant source;
  advertiser IDs must never be inferred to be publisher IDs.
- User explicitly requested no new branches for this project and no Fable
  review. UX review should use Codex GPT-6 Sol when needed.

### Verified work and current Git state

| Repository | Current checkpoint | Status |
| --- | --- | --- |
| `viant/datly-studio` | `4a4025f` on `main`, matching `origin/main` | Pushed graph layout: root above the first child row; all seven first-level children fit at a 1600px desktop width without sideways scrolling. Includes earlier `fd53857` owner visibility during optional user-info outage, `e89a83f` rotating JWKS support, and `7763500` reusable `app/studioapi.Run`. |
| `viant/sqlparser` | `4a52637` on `main`, matching `origin/main` | Pushed protected-region scanner fast path. Full `GOWORK=off go test ./...` passed. The uncommitted `source/scanner_benchmark_test.go` adds six ordinary/comment/quote/dollar/bracket scaling cases; an unrelated `mcp-go-debugger.log` is untracked. |
| `viant/authz` | `d5fd275` on `main`, matching `origin/main` when last checked | Pushed verified identity-only resolver so owner/public namespace access does not depend on optional user-info role service. Public OAuth user-info facts require `subject`, `roles`, `features`, and named `entityPermissions`. The working tree later acquired unrelated edits; inspect before touching it. |
| `viant-internal/auth` | `ENG-57520` existing branch | `4e5ad75` was pushed with public `GET /v1/api/user/info` authority fields and a five-minute successful-facts cache. The branch has subsequently moved to newer commits; do not reset it. Its push CI publishes `gcr.io/unison-cloud/datly-auth:latest` and a Helm chart. No claim is made that the live IDP deployment includes these fields. |
| `viant/datly` | existing `v1` branch | Datly 1.0 is pinned by Studio; do not substitute legacy APIs. The sibling checkout has unrelated active edits; do not reset or sweep it. |
| private `steward/studio-prereq` | untracked nested module under Steward | Links forecasting predicate handlers into custom Studio API/runtime binaries and carries fixture/acceptance scripts. It is **not** committed as part of public Studio. Its `go.mod` still pins older Studio/authz revisions, while the dedicated local build used an explicit temporary `go.work`; update only after reviewing private-module scope. |

The Studio `main` working tree currently has **uncommitted** changes for an
optional `versions.inspect` request field `discoverColumns`, regenerated Go
and browser OpenAPI clients, an SDK test, documentation, and a direct
`github.com/viant/sqlparser` version bump to the pushed scanner fix. Focused
`GOWORK=off go test ./cmd/datly ./studio/report_versions/inspect ./runtime/host`,
62 `studioApi.test.js` cases, and `npm run build` passed before the turn was
interrupted. Re-run relevant checks after any further edits. Preserve the
pre-existing untracked `datly-runtime.dedicated.yaml` and `ui/ux-fixtures/*`;
they were not part of these commits.

### Current local deployment and reproducible evidence

The dedicated loopback stack was restarted on 2026-09-30 and the UI root
returned HTTP 200. It is a temporary local process set, not a durable service
manager; check again after a host/app restart.

| Service | Address and state at checkpoint |
| --- | --- |
| Built Studio UI + Viant IDP OAuth BFF | `http://127.0.0.1:19173/`, served by `/tmp/datly-studio-dedicated-19173/start-idp-broker.py` and its private custom API binary. |
| Static native Studio HTTP / MCP | `127.0.0.1:19181` / `127.0.0.1:19190`, launched from `/tmp/datly-studio-dedicated-19173/datly-studio-static` with `static.yaml`. |
| Dynamic namespace admin | `127.0.0.1:19184`, custom `forecasting-studio-runtime` with `runtime.yaml`. |
| Namespace MCP listeners | Forecasting `18893`, Alpha `18891`, Beta `18892` in the isolated SQLite catalog. Simultaneous isolation was previously verified with synthetic fixture identity; real-IDP runtime authorization remains a separate gate. |
| Studio catalog | `/tmp/datly-studio-dedicated-19173/studio.db`, an isolated SQLite snapshot; backup before real-IDP fixture ownership alignment is `studio.before-idp-owner.db` beside it. |
| Dedicated `ci_ads` MySQL | Docker `datly_studio_forecast_mysql`, loopback host port **13317**, `--restart unless-stopped`, separate named volume `datly_studio_forecast_mysql_data`. `mysql_dev` on 3307 and `mysql_auth` on 3308 are other workers' containers; do not stop or change them. |

The local MySQL container was copied from `mysql_dev` because that shared
container had previously been stopped by other work. Only the six lookup
tables required by Forecasting were imported into the dedicated `ci_ads`:
`CI_CHANNEL_V2`, `CI_PUBLISHER`, `CI_SITE`, `CI_CARRIER_ALIAS`,
`CI_COUNTRY_REGION`, and `CI_DMA`. The dump/import was limited to those tables;
no row contents were printed or inspected. The dedicated connector uses a
server-held, mode-0600 DSN file
`/tmp/datly-studio-dedicated-19173/ciads-13317.dsn` and a MySQL principal
granted `SELECT, SHOW VIEW` on `ci_ads.*`. Do not put its contents in Git,
logs, UI fields, or handoff text.

The `ci_ads_local` connector was changed **through Studio UI** to that secret
reference. The latest SQLite catalog check shows `draft`,
`last_test_status=passed`, etag 8. The UI test action had completed before the
turn was interrupted, but **activation was not completed**. Reopen the
authenticated Studio UI, find `ci_ads_local` in Connectors, verify the passing
probe, then activate it through the UI. Connector changes deliberately reset
active status. `bq_metrics` remains active/passed. Until activation, a new
Forecasting validation or column-discovery request may fail its active-source
check even though the dedicated database is running.

The previous real-IDP browser run showed three governed namespaces, the
Forecasting component, seven child join views, 236 inputs/235 predicates, and
61 output columns when `ci_ads_local` was active and reachable. Draft v12
revision 9 passed `Validate revision` after the private predicate package was
linked; published v15 revision 7/generation 30 was validated against synthetic
fixture grants in `studio-prereq/ACCEPTANCE.md`. These do not prove real-user
publisher access. The live `https://idp.viantinc.com/v1/api/user/info` response
previously lacked `subject`, `roles`, `features`, and `entityPermissions`, so
strict role/entity decisions correctly fail closed until the updated auth
service is deployed. Owner-only namespace visibility uses independently
verified ID-token identity.

### UX and performance findings

- A blank signed-in page was traced to the temporary staged UI bundle loading
  two React copies, caused by an incomplete Vite alias/dedupe configuration.
  The staged build now uses the same Forge aliases and React dedupe as Studio;
  browser verification showed the app and graph rendering. Do not copy the
  one-line old staged `vite.config.js` back into the repository.
- A callback showing `login state is unavailable` was an expired/missing
  short-lived PKCE state cookie. Start a fresh flow at
  `http://127.0.0.1:19173/v1/studio/auth/login` in the same browser/profile;
  reloading a callback URL cannot complete it. OAuth login and static API
  remain distinct origins/ports. Never log ID/refresh tokens or client secret.
- The graph originally used a horizontally overflowing flex row that clipped
  the first children and risked browser back-navigation via trackpad scroll.
  The pushed layout moves Forecasting below the Input/Output summary row and
  uses a wrapping grid. At desktop width, all seven children are visible in
  the first row; at narrow width, measured graph `scrollWidth == clientWidth`.
  Full labels are available via hover titles where visual truncation remains.
- `versions.inspect` for Forecasting v12 returned ~428,070 JSON bytes and
  took about 4.4–4.7 seconds when source-column discovery succeeded. Approximate
  response breakdown: `structure` 310,937 bytes (`declarations` 196,646;
  `component` 110,854), `version` 78,158, top-level `dql` 38,827. Datly
  Reader Builder's DQL parse took ~54 ms, while `InspectContract` metadata
  discovery took ~1.5–1.8 seconds in a direct probe. Do not describe all
  4.7 seconds as BigQuery time: Forecasting also opens its local MySQL joins.
- SQL scanner CPU profile found `source.syntaxContext.advance` and
  `source.protectedAt` dominated the ~54 ms parser pass. The pushed scanner
  optimization made six 4/64 KiB SQL-shape benchmarks roughly 12–14% faster
  and the Forecasting repeated Reader Builder pass ~48–50 ms. This is useful
  generally, but cannot by itself remove the multi-second inspect delay.
- Explicit `discoverColumns:false` made the inspect request ~0.54 seconds but
  returned only **3 declared columns** for this `*`-heavy draft rather than
  the full **61**. The uncommitted API therefore defaults to full discovery
  and exposes the fast option only to callers that know the dynamic shape is
  already complete. Ordinary Studio UI keeps full discovery. Do not enable
  the fast option by default for this Forecasting draft.

### Important Datly runtime distinction and remaining design work

Dynamic MCP readers do **not** require prelinked generated reader types.
Datly 1.0 `transcribe.RuntimeContracts` materializes input/output `reflect.Type`
at runtime and `runtime/host/service.go` registers the resulting components
once per generation. Their HTTP/MCP calls reuse that registered contract;
they do not perform column discovery on every tool invocation. Application
predicate handlers/codecs such as private `PublisherScope` are separate and
must still be linked into the host binary.

Current generation **reload** in `runtime/host/service.go` calls
`transcribe.NewCompiler().RuntimeContracts` with
`ColumnRefiner: column.New(sources.connections)`, so it rediscovers source
columns when building a new generation. The user's desired direction is:
discover wildcards when SQL changes in authoring, then keep the resolved
dynamic Go shape in the registered generation as the execution authority.
No prelinked generated reader package is required. Avoid adding a parallel
manifest merely to cache what the dynamic registered shape already expresses.
The unresolved part is making a fresh generation reconstruct the same complete
shape without an unnecessary database metadata pass while preserving exact
version identity, authorization, and wildcard semantics. Do not simply remove
`ColumnRefiner` from runtime compilation: this current draft then drops from
61 to 3 columns. Design and test that lifecycle before changing publish/reload.

### Forge windows through MCP — added 2026-09-30

The sibling public Forge checkout now has a focused MCP bridge extension in
`backend/mcp`:

- `forgeWindowList({clientId?})` lists **active** windows in one connected UI
  client's current MCP namespace. Its response includes `windowId`, key,
  title, tab/modal/minimized flags, and selected state. It intentionally omits
  parameters, forms, datasource collections, and other window content.
- `forgeWindowGet({clientId?,windowId})` returns the existing semantic snapshot
  for one exact active window ID in that same UI client and namespace. It
  rejects missing/whitespace-padded IDs, missing clients/windows, and snapshots
  with duplicate window IDs; it does not search other clients or namespaces.
- These are typed `github.com/viant/mcp-protocol/server` tools registered by
  Forge's existing MCP handler. `GOWORK=off go test ./backend/mcp/...` passed,
  including service isolation, stale-client refusal, ambiguous-ID refusal,
  and tool-registration tests. The implementation is committed locally as
  Forge `a448c2c` on existing `main`. It is **not pushed** at this checkpoint:
  Forge already had an unrelated outgoing `main` commit (`6b9635b`), so a
  routine push would publish both. Preserve unrelated Forge untracked files.

This is an **active-window** list, not a registry/catalog of every window
definition that could be opened. A frontend must opt into Forge's UI bridge
(`startUIBridge` or `startUIBridgeHTTP`) and publish a current semantic
snapshot; without a connected frontend the list reports `connected:false` and
get fails. Datly Studio presently uses Forge theme/components but does not by
itself establish this UI-bridge connection. The Forge MCP tools also do not
automatically appear on Datly Studio's namespace MCP listener: private AI
Studio must compose or proxy them through its generic MCP adapter and enforce
namespace, role, exposure, and allowed-entity policy **server-side before tool
discovery and retrieval**. `forgeWindowGet` can contain form/parameter/row
state; never expose it merely because a caller knows a window ID. No connected
Forge frontend, authenticated host adapter, or end-to-end window retrieval has
yet been verified for AI Studio.

### Exact next actions

1. Confirm the three local processes still listen and the dedicated
   `datly_studio_forecast_mysql` container is healthy. Resume the existing
   authenticated UI session or start a fresh IDP login; no collision precheck
   is needed when bootstrapping dedicated ports.
2. In the UI, finish `ci_ads_local` **activation** after its passing test on
   port 13317. Confirm both connectors are active/passed. Reopen Forecasting
   v12 and verify all seven joins and 61 columns. Do not touch the shared
   `mysql_dev`/`mysql_auth` containers.
3. Review the uncommitted optional `discoverColumns` API and generated clients.
   Test default full inspection and explicit `false` against a stable fixture;
   ensure a missing metadata source yields an actionable diagnostic and never
   silently claims a complete graph. Commit/push on existing `main` only after
   validation; preserve the unrelated untracked fixture files.
4. Decide and implement the dynamic shape/reload boundary described above.
   Re-run publication, reload, HTTP/MCP tool discovery/calls, cube dimensions
   and measures, namespace-port isolation, and denial cases. Do not infer
   publisher permissions from advertiser grants.
5. Check real IDP rollout for the extended user-info contract. Until it is
   actually live, protected runtime preview and user-specific entity scope
   remain blocked. Keep the policy fail-closed; do not turn on a broad public
   mode merely to make the preview work.
6. Review the private Studio SDK dependency version in `studio-prereq/go.mod`
   once public commits are finalized. Do not accidentally commit that untracked
   nested module inside Steward. Continue AI Studio reporting only after the
   Datly Studio prerequisite flow is accepted.
7. Review the local Forge MCP window-list/get commit `a448c2c` and its
   pre-existing outgoing commit before pushing either. Connect a Forge
   frontend to the bridge, verify real MCP `tools/list`, list,
   exact-ID get, disconnected-client behavior, and denied cross-client access.
   Wire into private AI Studio only through its authorized namespace MCP
   adapter; do not add reporting behavior to public Datly Studio.

### Verification commands and limits

```bash
cd /Users/awitas/go/src/github.com/viant/datly-studio
GOWORK=off go test ./cmd/datly ./studio/report_versions/inspect ./runtime/host
cd ui && node --test src/studioApi.test.js && npm run build

cd /Users/awitas/go/src/github.com/viant/sqlparser
GOWORK=off go test ./...
```

The focused Studio Go packages, 62 browser SDK Node cases, Vite build, and
SQL parser suite passed at the last code-edit checkpoint. The six-shape
benchmark file remains uncommitted; keep or discard it based on its review.
The full Studio suite passed before the current uncommitted inspect-parameter
change; run it again before claiming a new production-ready commit. Neither
the synthetic acceptance suite nor a UI screenshot proves real IDP publisher
authorization or production BigQuery data correctness.

## Executive summary

Datly Studio is being rebuilt as a database-agnostic visual authoring product
for **Datly 1.0 reader components**. Its business purpose is to let an
authenticated user connect to an authorized data source, inspect schema
metadata, compose a typed Datly view graph, test it, version it, and publish it
as a dynamic Datly reader and optional MCP capability.

The original Studio was incomplete and coupled to legacy Datly concepts. The
current direction is deliberately different:

- Datly owns query compilation, typed reader/writer semantics, predicates,
  generated shapes, runtime execution, cache/cube behavior, and MCP exposure.
- Studio owns tenancy, connector governance, versioning, ACL, resources,
  publication, lifecycle and the browser-facing SDK.
- Forge/Blueprint UI talks **only** to the Studio SDK. It neither runs SQL
  directly nor writes DQL/storage itself.
- Static Studio data-model components use generated Datly readers and writers.
  User-authored dynamic capabilities are **readers only**. Do not remove the
  static writers because of this distinction.

The main backend and a usable Forge UI are running locally. The latest work
completed the view workspace: no raw DQL by default, no compiler-wrapped SQL,
connector-backed columns, scoped column authoring, and a clean component graph.

## Product objectives

1. Connector-first workflow

   A connector has a stable name and is the reference used in DQL/component
   configuration. It can be created, safely edited, probed, activated,
   disabled, inspected for schemas/tables/columns, and deleted when unused.
   Secret connection material stays server-side.

2. Dynamic Datly reader composition

   A user creates a Component (the UI name; legacy internal DTO/table names
   still say `report`), defines a root view and subviews, relations, parameters,
   predicates, output controls, cube/cache/MCP/resource options, tests each
   part, validates a runtime contract, and publishes a version.

3. Safe, visual authoring

   The default authoring experience is a graph and metadata form, not DQL.
   A component can be understood as:

   ```text
   Component
     ├── parameters and predicates
     ├── view graph: Vendor → Products
     ├── per-view embedded SQL
     ├── per-column tags / casts / functions
     └── component-wide cache, cube, MCP, resources and lifecycle
   ```

4. Tenant-safe publication

   User ownership comes from a verified JWT `sub`. Dynamic component package
   identity is server-owned and derives from a sanitized subject segment plus
   component ID, preventing collisions. The dynamically published reader host
   has its own HTTP and MCP ports.

## Non-negotiable design decisions

- Use Datly **1.0 only**. Do not reintroduce legacy repository/view/service or
  DAO forwarding layers.
- Each dynamic component requires `#package`; Studio determines the package
  identity. The browser must not choose it.
- DQL lives under `dql/` with lower-case underscore paths. Static Studio DQL is
  decomposed by component and operation; generated Go stays under `studio/`.
- All UI actions are public Studio SDK operations. The UI must not open a raw
  DB session, construct a direct SQL transport, or mutate version DQL itself.
- Reader Builder semantic commands are the editing API: `addView`,
  `updateView`, `removeView`, parameter/predicate operations, relation updates,
  settings, `setColumnRole`, and generic function operations.
- Typed authorization is explicit. A valid JWT is required; `sub` is the only
  accepted owner identity. Do not restore role/email/username fallbacks.
- Static component discovery is reflection-based from listed packages; DQL
  transcription may use AST/source inspection. `internal/dependencylink` exists
  only for blank/default imports, not hand-maintained registration contracts.
- Generated static components expose their embedded resources through public
  `EmbedFS()` component methods. Do not add private hook files, stage manifests,
  `.datly-gen.json`, global registries, or init-time registration machinery.

## Repository map

| Path | Responsibility |
| --- | --- |
| `datly.yaml` | Static Studio Datly component configuration. Includes readers, writers, and authorization. |
| `dql/studio/` | Authored static Studio reader/writer DQL. |
| `studio/` | Generated Go shapes/handlers for static components. |
| `studio/authorization/` | JWT-aware owner, ACL, report and runtime predicates. |
| `sdk/` | Public Studio Go SDK DTOs and operations. |
| `sdk/transport/sql/` | Studio persistence/authoring transport; bridges Studio versions to Datly Reader Builder. |
| `store/sql/` and `schema/` | Single-source Studio SQLite/MySQL schema and migrations. |
| `runtime/preview/` | Bounded dynamic Datly execution for SQL/view/reader/cube testing and validation. |
| `runtime/host/` | Published dynamic reader host, generation reload, MCP service and resource loading. |
| `cmd/studio-api` | Local BFF/SDK host on port 8080. |
| `cmd/studio-runtime` | Dynamic reader HTTP/MCP host. |
| `ui/` | Forge/Blueprint application, browser SDK client, UI tests and UX review. |
| `e2e/` | Endly/MySQL Docker-oriented e2e structure; SQLite remains the unit-test store. |
| `studio.md` | Deep architecture, prior findings and acceptance design. |
| `ui.md` | UI product/interaction design. |
| `ui/UX_REVIEW.md` | Window-by-window UX acceptance evidence. |

Run Studio Go commands with `GOWORK=off` unless the workspace is explicitly
configured; otherwise Go rejects the module due to the parent `go.work`.
Datly v1 is now pinned to immutable Git revision `72521de9c511` as
`v1.0.1-0.20260926140723-72521de9c511` in `go.mod`, with no local Datly
replacement. Module checksum verification and both full Go suites pass.
Run `scripts/verify-production-build.sh` in a clean checkout before declaring
release readiness. The script refuses any Datly module replacement,
requires every public SDK operation to have a generated native route/MCP tool,
then verifies module checksums, tests and builds the Go binaries, and runs the
lockfile-installed UI tests and production build.

The sibling Datly checkout currently has three unreleased change groups used
by this work: Reader Builder `SourceProjectionAll` inspection for safe
`SELECT *` metadata fallback; duplicate-key HTTP 409 mapping; and declared
`JSONWireType` for generated OpenAPI response schemas. Their tests and
the Datly module suite passed locally, but those edits are uncommitted in the
sibling repository and cannot be supplied by the pinned remote version yet.

The native SDK migration now includes `reports.create` as a linked Datly
workflow, bringing the generated HTTP/OpenAPI/MCP contract to 20 SDK routes.
It checks the verified owner, authorized active connector, owned active
namespace (with `general` creation), then invokes an internal report insert
writer. Returning the writer's row avoids an uncommitted read-after-write
race. Combined-host tests cover HTTP, direct and BFF MCP, forged owner and
component identity, inactive connector, missing namespace, duplicate 409,
unauthenticated denial, persistence, and private child-route isolation. The
browser uses the generated client. Full Studio Go tests, all 62 Node and 78
rendered UI tests, and the Vite production build passed on 2026-09-26. This
was an API-client change, not a substantive UI change, so the conditional UX
review trigger was not met.
`namespaces.update` is the next native SDK operation (21 total). Its linked
handler binds verified ownership, reads the current namespace with a private
Datly component, validates the nested patch, and uses an internal writer for
optimistic etag changes. Direct HTTP/MCP and authenticated BFF MCP tests cover
success, stale revision, non-owner denial, invalid title, and private-child
isolation. The browser calls the generated client; no substantive UI changed.
`namespaces.delete` is the twenty-second native route. The linked workflow
checks owner, etag, and a private report-usage count before an internal
soft-delete writer. HTTP returns 204; generated OpenAPI declares only 204 and
the browser preserves a void result. Direct/BFF MCP and HTTP tests cover
archival, referenced/stale conflicts, non-owner denial, and private children.
`connectors.disable` is the twenty-third native route. A private typed edit
guard, redacted metadata reader, and internal optimistic status writer keep
secrets out of HTTP/MCP while preserving them in storage. Direct/BFF MCP and
HTTP tests cover owner and delegated view+edit success, non-owner denial, stale revision, secret
redaction, and persistence. The browser uses the generated route; the
conditional substantive-UI review trigger was not met.
`connectors.activate` is the twenty-fourth native route. It shares the
verified edit guard and redacted status workflow with disable, but enforces a
persisted passing connectivity test in both preflight and the writer lifecycle.
HTTP, direct MCP and authenticated BFF MCP tests cover success, untested 400,
non-owner 403, stale 409, persistence, and secret redaction. The generated
browser route is selected without a substantive UI change.
`connectors.delete` is the twenty-fifth native route. The verified edit guard,
private usage count, and internal status writer archive only an unreferenced
connector at the expected etag. HTTP and generated OpenAPI return 204;
direct/BFF MCP tests, denial/conflict cases, persisted archival, and private
child isolation pass. The browser preserves a void result without a
substantive UI change.
`connectors.update` is the twenty-sixth native route. Its private catalog
reader supplies the server-held configuration only after a typed edit guard;
the linked writer applies optimistic etag changes. Connection changes reset
the connector to draft and clear probe evidence; description-only edits leave
active/test state intact. HTTP, direct/BFF MCP, denial, conflict, persistence,
secret redaction, and internal-child isolation have combined-host coverage.
The generated browser client is selected without a substantive UI change.
`reports.update` is the twenty-seventh native route. Its linked workflow uses
private typed edit, active-namespace, version-head, and optimistic config
children. The tested contract denies forged component identity, stale etags,
invalid namespace, non-editors, and default-connector changes after the first
version; it permits a pre-version connector change to another active source.
HTTP, direct/BFF MCP, persistence, and internal-child isolation pass. The
browser uses the generated client; no substantive UI changed.
`versions.create` is the twenty-eighth native route. It links private typed
edit, capability, version-head and insert components, binds the actor to the
verified principal, and preserves initial draft metadata and spec hashing.
Owner HTTP/direct MCP/BFF MCP creation passes; a delegated editor can create
without receiving DQL fields back, while the source remains stored. Forged
actor, invalid mode, persistence and private-child tests pass. The browser
uses the generated route without a substantive UI change.
`acl.delete` is the twenty-ninth native route. The owner-only ACL reader and
internal optimistic writer enforce the exact grant and etag; HTTP/OpenAPI
return 204, while stale conflicts retain safe expected/current etag fields.
Direct/BFF MCP, non-owner denial, missing row, persistence and private-route
tests pass. Combined-host exercise found and fixed a sibling Datly standalone
capability-binding gap for reader predicates; that fix has a Datly regression
test. The full Studio and sibling Datly `GOWORK=off go test ./...` suites, 67
Node tests, 78 rendered UI tests, and the Vite production build passed on
2026-09-26. The ACL writer's pointer-key regeneration gap was closed by adding
explicit casts to canonical DQL; `go generate` now recreates its private
router and typed mutation fields. The unpublished sibling Datly dependency
remains a separate release-readiness item.

`acl.upsert` is the thirtieth native route. Its linked workflow validates
capability implications and owner scope, then uses the now-regenerable private
PATCH writer for both insert and optimistic update. Combined-host HTTP,
direct/BFF MCP, denial, stale-etag metadata and persisted grant tests pass;
the generated browser client is selected without a substantive UI change.
The full Studio Go suite, 68 Node tests, 78 rendered UI tests, and production
UI build passed after this route was added.

`versions.load_dql` is the thirty-first native route. It uses a managed
transaction to insert the version and `main.dql` resource and advance the
draft pointer atomically. A forced pointer-write failure rolled all three
back; HTTP, direct/BFF MCP, denied/malformed input and monotonic imports pass.
Generic and native responses now redact DQL for delegated editors lacking a
DQL grant, while persisting the authorized import. The browser uses the
generated route. This did not change substantive UI behavior.

The full Studio Go suite, 69 Node tests, 78 rendered UI tests, and the Vite
production build passed after the DQL import and generic redaction changes.

`versions.load_archive` is the thirty-second native route. It shares DQL
import authorization, redaction, bundle hashing and managed atomic writes,
but uses the bounded archive parser and explicit root selection. HTTP,
direct/BFF MCP and a delegated editor check passed with a multi-root,
three-file ZIP; generated browser requests preserve base64 bytes. No
substantive UI layout or interaction changed.
The full Studio Go suite, 70 Node tests, 78 rendered UI tests, and the Vite
production build passed after the archive route was selected.
`versions.inspect` is the thirty-third native route. It uses the exact
authorized version, scoped active connector catalog and Datly Reader Builder.
Inspection projection is shared with the generic SDK. A new regression found
that DQL-disabled viewers previously received SQL inside the structure JSON;
they now receive safe status/view metadata and diagnostic codes only. Owner
HTTP/direct/BFF MCP, delegated HTTP/MCP redaction, OpenAPI and private-reader
isolation pass. The browser uses the generated route without a substantive UI
change.
The full Studio Go suite, 70 Node tests, 78 rendered UI tests, and the Vite
production build passed after the inspection redaction and native route.
`versions.apply` is the thirty-fourth native route. It validates all three
SDK edit kinds and the expected source revision, then uses the internal
optimistic version writer. Owner HTTP/direct/BFF MCP, delegated redaction,
non-editor denial, stale revision, persistence and private writer tests pass.
The generic SDK edit response now redacts DQL for delegated editors too.
The generated browser adapter exposes the operation without changing UI layout.
`authorization_predicates.types` is the thirty-fifth native route. A private
Datly global-publisher guard checks the verified subject before the type
catalog is projected. Signed HTTP, direct MCP, authenticated BFF MCP,
non-publisher and unauthenticated denial, generated clients and private-child
isolation pass. The browser adapter changed, but no UI layout or interaction
changed; the full Studio Go suite, 72 Node tests, 78 rendered tests, and the
Vite build passed.
`authorization_predicates.get` and `.list` are the thirty-sixth and
thirty-seventh native routes. They invoke the private global-publisher guard
and internal predicate store reader; shared row projection preserves the SDK
DTO, linked flag, SQL scope metadata, filtering and bounded pages. Combined
host signed HTTP/direct/BFF MCP tests cover owner, denial, missing identity,
link state, filters, paging and private-child isolation. Generated Go/browser
clients and exact BFF mounts include both routes. The full Studio Go suite,
73 Node tests, 78 rendered UI tests and Vite build pass after this slice.
The new native predicate handlers use the default static linked catalog;
embedding hosts with extra `host.Config.PredicatePackages` still need an
explicit runtime catalog binding and extension-parity test.
`authorization_predicates.create` is the thirty-eighth native route. It uses
the same private publisher guard and an internal Datly POST writer, with
server-owned identity/state and linked-type validation. Combined-host tests
pass for signed HTTP, direct/BFF MCP, persistence, duplicate conflict,
unlinked-type rejection, denied caller and private writer isolation. The
browser uses the generated native client without a substantive UI change.
The full Studio Go suite, 74 Node tests, 78 rendered UI tests and Vite build
pass after this route.
`authorization_predicates.update` and `.delete` are the thirty-ninth and
fortieth native routes, completing that six-operation SDK family. Their
private optimistic Datly writer advances ETags and performs durable soft
deletion; combined-host HTTP/direct/BFF MCP tests cover stale conflicts,
invalid updates, denied callers, persistence and 204 deletion. Generated
Go/browser clients and the exact BFF mounts include both routes. The full
Studio Go suite, 75 Node tests, 78 rendered UI tests and Vite build pass.
`connectors.schemas`, `.tables`, and `.table` are native routes forty-one
through forty-three. They authorize through the public connector reader,
then load scoped private connection material for SQL metadata discovery.
The static executable now links SQLite/MySQL/Postgres/BigQuery metadata
adapters. Combined-host SQLite HTTP/direct/BFF MCP checks pass; with the
Docker MySQL fixture enabled, native HTTP and both MCP paths found the 20
views and 60-column table. Responses exclude DSN material. A deployment
using the shared Scy resolver can use server-held secret references: a local
secret resource passed static HTTP/direct/BFF MCP schema and probe checks
without exposing the DSN or secret path. Custom resolver injection remains
an embedding integration choice, not a missing default. The full
Studio Go suite passed again after the secret-reference check; 75 Node tests,
78 rendered UI tests and the Vite build passed after the native routes.
`connectors.test` is the forty-fourth native route. An edit-scoped guard
permits a draft connector probe, and the internal optimistic status writer
persists bounded pass/fail evidence without changing the configuration ETag.
SQLite and Docker MySQL signed HTTP/direct/BFF MCP probes pass; rejected
editor access and redacted failed-probe responses have regression checks.
The generic SDK probe response also now suppresses raw driver/secret errors.
The full Studio Go suite, 75 Node tests, 78 rendered UI tests and Vite build
pass after this route.
`connectors.test_sql` is the forty-fifth native route and completes the
connector SDK family. It compiles and executes transient SQL through Datly
under verified connector edit access, a 15-second deadline, and the existing
row/byte budgets. Signed HTTP/direct/BFF MCP tests pass on SQLite and the
Docker wide table; draft, denied, and empty SQL requests fail. The sibling
Datly compiler now has `RuntimeContractsInModule`, allowing this runtime
contract to materialize without a source checkout or `go.mod`. The local
Datly replacement remains a release gate until those changes are published.
Both complete `GOWORK=off go test ./...` suites passed, along with 75 Node
tests, 78 rendered UI tests and the Vite build.
`preview.execute` is the forty-sixth native route. A private exact-report
`can_run` guard precedes borrowing the configured Studio DB capability and
running the shared Datly preview engine for the requested version. SQLite
HTTP/direct/BFF MCP tests cover owner/delegated access, denial, missing
version and private child isolation. Docker HTTP and both MCP paths returned
the two complete 60-column wide rows. The preview connector reader now
accepts secret-reference-only active connectors without blocking previews.
The full Studio Go suite, 75 Node tests, 78 rendered UI tests and Vite build
pass after the native preview route; the prior full sibling Datly suite passed
after its source-free runtime-contract addition.
`versions.validate` is the forty-seventh native route. It enforces edit
authorization and exact source revision, compiles the real Datly runtime
contract, persists valid/invalid diagnostics through a private optimistic
writer, and redacts source-bearing output for an editor without `can_use_dql`.
SQLite and Docker signed HTTP/direct/BFF MCP validation checks pass, including
stale conflict, run-only denial and private writer isolation. The full Studio
Go suite, 75 Node tests, 78 rendered UI tests and Vite build pass.
`versions.test_view`, `.test_relation`, and `.test_compose` are native routes
forty-eight through fifty. They reuse the private `can_run` guard and trusted
Studio DB, then invoke Datly's actual transient view, relation-evidence and
derived cube-compose services. Signed HTTP/direct/BFF MCP checks pass for
each on SQLite; the 60-column Docker view passed all three paths. The browser
adapter uses the generated clients without a substantive UI change. The full
Studio Go suite, 75 Node tests, 78 rendered UI tests and Vite build pass.

The release coverage audit currently finds 50 native routes out of 65 public
SDK operations. `scripts/check-native-sdk-coverage.mjs` lists the 15 missing
operations and fails the production gate until each has a selected Datly
HTTP/OpenAPI/MCP contract. The gate also reports the still-local `../datly`
replacement in the same run. Upstream `v1` at `7ee70134` is an ancestor of
the local Datly branch and does not yet include the required local commits and
uncommitted changes; no upstream publication was performed.

## Data model and static components

Studio persists its own metadata in `schema/schema.ddl`; it is the single source
of truth used by unit fixtures and runtime migration. The core entities are:

- `connectors`: named data-source configuration, driver, server-held DSN/secret,
  provider options, ownership, probe state and lifecycle state.
- `reports`: user-facing Components; title, business namespace, owner,
  connector reference, deterministic dynamic package scope/name, status/etag.
- `report_versions`: immutable authored/generated DQL and validation state.
- `report_views`, `report_parameters`, `report_cube_configs`,
  `report_mcp_exposures`: component authoring metadata.
- `report_resource_files`, `report_resource_folders`, `report_skill_roots`:
  versioned resource and skill publishing records.
- `report_publications`, `runtime_generations`: atomic deployment lifecycle.
- ACL tables: owner and delegated `can_view`, `can_edit`, `can_publish` access.

Every schema table has a corresponding static Studio Datly component set where
needed. These are generated readers plus universal generated writers. Writers
have lifecycle/invariant authorization, sparse update/presence behavior, and
delete support. The static readers use typed authorization predicates; writers
authorize via input/lifecycle hooks.

Connector security details already implemented:

- Reader output never returns `dsn_template`.
- UI sees only `dsnConfigured`, never existing DSN material.
- Changing provider, DSN, secret or provider options clears prior probe evidence
  and returns the connector to draft. Description-only edits preserve state.
- Supported connector validation scope: MySQL, PostgreSQL, SQLite, BigQuery and
  Aerospike. MySQL is the e2e target; SQLite is only the unit-test store.

## Dynamic reader lifecycle

1. Component/version metadata is persisted through Studio SDK operations.
2. Studio calls Datly Reader Builder for inspect/edit, preserving source and
   applying only typed semantic operations.
3. Preview/test dynamically compiles a bounded reader through Datly; it does
   not use a browser-side SQL execution path.
4. Runtime validation opens the named active connector(s), refines schema/types,
   initializes the executable contract and verifies exposure/resources without
   executing business data.
5. Publication stages a generation, requests the dedicated runtime reload, then
   marks the generation active only after successful reload.
6. The runtime host rebuilds the active generation from Studio DB on startup,
   so it recovers after restart. Runtime database handles are bound to the
   Datly generation shutdown lifecycle and retired safely.

Dynamic readers are intentionally the only user-authored runtime capability at
this stage. Dynamic writer authoring is out of scope; it does not imply that
the static Studio writers should be removed.

## Authentication and authorization

- In development mode, the BFF accepts an explicit local development subject.
- In authenticated mode, the BFF holds an opaque HttpOnly session cookie,
  verifies JWT through configured JWKS/cert validation, and forwards the bearer
  token only to exact native SDK routes on the static Datly host and the
  dynamic Datly HTTP/MCP proxy.
- Authenticated deployments require an exact JWT issuer and audience plus a
  base64-encoded 32-byte `STUDIO_SESSION_KEY`. Sessions are shared through the
  Studio database; cookie identifiers are hashed and bearer/claims payloads
  are AES-GCM encrypted at rest, allowing authentication and revocation across
  Studio API instances that share that database.
- Studio uses verified JWT claims. `sub` is authoritative for ownership and
  package derivation.
- Static component DQL imports the typed authorization package and applies
  handler predicates. Authorization packages are included in `datly.yaml` and
  the dependency-link blank-import package.
- `internal/dependencylink/link_test.go` enforces component auth coverage and
  reflection availability for linked predicate types.

## Current Forge UI

The UI is in `ui/src/` and is intentionally a Forge/Blueprint composition over
`StudioAPI`.

Completed areas:

- navigation and session shell;
- Components catalog (renamed from Reports in all user-facing language);
- connector CRUD/probe/activation/delete dialog;
- schema browser with closeable table tabs and transient Datly SQL test;
- graph-first component builder with Input, Views, and Output catalogs below the graph;
- predicate builder, relation editor, root/subview flows;
- cube/cache/MCP/resources/skills/ACL/validation/publication/history windows;
- nested preview table and JSON mode, including proper expandable embedded
  subtables rather than `[object Object]` scalar cells;
- conflict/reload handling and runtime status.

Latest view-workspace design (verified live):

- Input and Output are compact selectable graph blocks. Their lower workspace
  contains searchable, paginated catalogs; selecting a parameter or predicate
  opens its settings inline rather than in a modal. Small catalogs omit
  redundant count and range labels.
- Components with more than 12 views use a compact Views block and a paginated
  lineage catalog, so 50-view multilevel branches do not expand into an
  unmanageable graph. Search covers view path and source table.
- First tab is `Component <title>` with a component icon; view and SQL tabs use
  distinct icons and have individual close buttons.
- The graph uses simple business names such as `Vendor → Products`; technical
  labels (`ROOT VIEW`, `reader`, `JOINED SUBVIEW`) are removed.
- Clicking a view opens a **view overview**, not SQL. It shows lineage,
  inherited connector, view-wide functions/settings, searchable paginated
  fields and column authoring.
- Column metadata is taken from compiled Datly columns when available, falling
  back to the authorized Studio SDK schema catalog for older `SELECT *` views.
- A column opens a scoped dialog for cube role, Go-style tag, Go type cast and
  an advanced generic outer-projection function. Those controls submit Reader
  Builder operation DTOs; they do not parse/mutate DQL in the UI.
- `Open SQL` creates a dedicated CodeMirror SQL tab. It shows only the embedded
  view query. Raw structural DQL remains under the Advanced cog and is opt-in.
- Navigation destinations have distinct warm-pastel icon treatments.

## Datly 1.0 extensions made during this work

The local Datly v1 checkout is `/Users/awitas/go/src/github.com/viant/datly`.
The following recent changes are relevant and currently uncommitted unless a
subsequent maintainer has committed them:

1. `application.Build.Shutdown`

   A shutdown callback lets dynamic runtime builds safely close generation-owned
   connector DB handles after in-flight leases drain. Studio binds its dynamic
   host generation resources to this callback.

2. Package-aware type resolution

   The type catalog resolves local package types before imported same-name types
   in both package and transcription authority modes. This avoids an imported
   `Input` masking the component-local `Input` type.

3. Embedded view SQL inspection

   `readerbuilder.ViewOccurrence` now carries `SQL`, the exact inner query from
   the named view source span. Studio consumes it in preference to compiled
   `View.Source.SQL`; this prevents the UI from showing a wrapper such as:

   ```sql
   SELECT * FROM (SELECT * FROM VENDOR t ...) vendor
   ```

   The verified SQL tab now shows only:

   ```sql
   SELECT * FROM VENDOR t
   ${predicate.Builder()...}
   ```

Do not replace this with browser-side SQL unwrapping. The canonical Datly
Reader Builder inspection response is the correct ownership boundary.

## Local services and runbook

Last verified local endpoints:

| Service | Address | Start command |
| --- | --- | --- |
| Forge UI | `http://127.0.0.1:5173` | `cd ui && npm run dev -- --host 127.0.0.1` |
| Studio SDK/BFF | `http://127.0.0.1:8080` | `GOWORK=off go run ./cmd/studio-api -address 127.0.0.1:8080 -dsn 'file:.data/studio.db?cache=shared' -subject awitas` |
| Static Studio Datly control plane | `http://127.0.0.1:8081` | `GOWORK=off go run ./cmd/datly start -conf datly.yaml` |
| Static Studio SDK MCP host | `http://127.0.0.1:8090` | started by the static Datly command; authenticated BFF path `/v1/studio/sdk-mcp/mcp` |
| Dynamic reader HTTP host | `http://127.0.0.1:8082` | `GOWORK=off go run ./cmd/studio-runtime -conf datly-runtime.yaml` |
| Dynamic MCP host | `http://127.0.0.1:8091` | started by `studio-runtime` |

Run the first two commands in persistent terminal sessions. A background Vite
process spawned from a short-lived shell may be reaped; use the session runner
or terminal directly.

Useful checks:

```bash
# Studio Go suite (local Datly replacement)
cd /Users/awitas/go/src/github.com/viant/datly-studio
GOWORK=off go test ./...

# UI
cd ui
npm test -- --run
npm run build

# Focused Datly Reader Builder contract
cd /Users/awitas/go/src/github.com/viant/datly
GOWORK=off go test ./authoring/readerbuilder

# Static component transcription uses Studio's wrapper and source mapping
cd /Users/awitas/go/src/github.com/viant/datly-studio
GOWORK=off go run ./cmd/datly transcribe get ...
```

For static schema writers, transcription additionally requires schema discovery
arguments, for example `-schema -connector studio -driver sqlite -dsn
'file:.data/studio.db?cache=shared'`.

## Test evidence

Verified during the current production-readiness work:

- `GOWORK=off go test ./authoring/readerbuilder` in Datly: passed.
- `GOWORK=off go test ./sdk/transport/sql ./cmd/datly` in Studio: passed.
- `GOWORK=off go test ./...` in Studio: passed.
- Focused Datly bootstrap, type catalog, Reader Builder, HTTP, runtime, MCP and
  MCP invocation suites: passed.
- Full `go test ./...` in Datly: passed with the default package timeout;
  the post-feature rerun completed `transcribe` in 424.798s inside the
  repository-wide run. A fresh uncached default profile completed in 390.54s,
  leaving roughly 3m30s margin.
- `npm test` in `ui`: passed, 44 Node contract/unit tests plus 41 rendered React
  connector-catalog, component-create, Schema Browser, subview-validation,
  graph/SQL/column-contract, component-settings, and stale-conflict interaction
  tests, including 120-input and 240-predicate scale scenarios.
- `npm run build` in `ui`: passed.
- `npm audit --audit-level=moderate` in `ui`: passed with zero known
  vulnerabilities after upgrading the rendered-test runner to Vitest 4.1.11.
- The Impeccable mechanical UI detector reports no findings.
- Datly 1.0 authoring/generation changes are committed on `v1` as `bc590691`
  and included by remote merge `fa412c1f`; Studio now pins
  `v0.39.2-0.20260921002714-fa412c1fdab6` rather than relying only on its local
  workspace replacement.
- SQL and preview share an adjustable vertical workspace: the editor is
  draggable and provides Minimize/Balanced/Maximize presets without rerunning or
  discarding preview evidence.
- MCP catalog projection now lists base reader, Cube, and CubeCompose as
  distinct generated components. Datly `report.ProjectedComponents` is the
  identity authority; a focused publication/runtime integration test proves all
  three names and routes appear after activation.
- Vendor Catalog draft v3/rev4 is the deterministic populated resource fixture:
  `awitas.docs:guide/SKILL.md`, folder `guide` at
  `skill://awitas-vendor-guide/`, and skill root `.`. It was created only through
  Studio SDK operations and validates successfully; Astra approved the populated
  edit and dependency-block surfaces without deleting data.
- Browser review on `Vendor Catalog`:
  - graph-first component workspace with inline selected-view inspector;
  - graph selection does not create a view tab; explicit SQL is the only
    per-view editable-resource tab;
  - `Vendor` inspector with seven merged connector/compiled columns;
  - column authoring dialog;
  - dedicated `Vendor SQL` showing unwrapped embedded SQL;
  - nested Vendor/Product preview behavior recorded in `ui/UX_REVIEW.md`.
- `scripts/seed-development-sqlite.sh` recreates the deterministic local
  reporting fixture and temporarily points every live connector at it when
  Docker is unavailable. The current Vendor view test returned all three
  seeded rows through `versions.test_view`.

Additional completed production-readiness slices:

- governed owner-scoped namespaces with schema migration v4, Datly DQL
  reader/writer components, generated packages, SDK operations and catalog UI;
- Datly Reader Builder-owned constants and atomic column contract operations;
- normalized column metadata, scalable 200+ input/predicate and 100+ column UX;
- atomic Datly Reader Builder batches and one cohesive Component settings UI
  for Cube, Cube Compose, composition MCP exposure, and the named reader MCP
  tool;
- an authorization-scoped Runtime workspace for the recorded active generation,
  deployed component versions, revisions, connectors, MCP inventory, and an
  authenticated dynamic-host readiness probe;
- explicit SQLite connector attachments for development fixtures, allowing
  published SQL such as `ci_ads.CI_VENDOR` to execute without rewriting its
  versioned DQL; both seeded dynamic HTTP routes are live locally;
- capability-gated raw DQL and edit/run/publish controls;
- report `can_run` enforcement for dynamic HTTP and MCP component targets;
- BFF sessions capped by verified JWT expiration;
- staged publication preserves active state, startup ignores orphaned desired
  state, unpublish preserves the active generation until swap, and activation
  persistence failure compensates the runtime to the previous generation.
- report connectors can no longer be repointed after a reader version exists;
  connector changes require a new Reader Builder-authored validated contract;
- MCP uniqueness checks cover both the latest draft and the active published
  version, and expired `building` generations are recovered instead of
  permanently blocking publication;
- Overview is the operational landing window with independently loaded runtime,
  component, connector, namespace, and attention sections.
- schema v5 adds durable report warmup runs plus an authorization-scoped Datly
  reader component. Studio executes the canonical Datly authored-case expansion
  under a detached bounded lifetime and persists accepted/running/terminal
  evidence; the cache dialog renders its latest run and history after reopen.
- preview and isolated view tests execute the exact draft version in every auth
  mode with 30-second, 200-row, and 2 MiB budgets plus version/connector/row/
  byte/truncation evidence. A distinct `versions.test_relation` operation uses
  the same compiled Datly reader and typed assembled output to report parent,
  matched, unmatched, and attached-child evidence for an exact selected edge;
  live `Vendor → Products` returned 3/2/1/3 for v2 revision 1.
- the predicate workspace now supports a 234-occurrence target-aware catalog
  and a synchronized compiled-group view. Datly Reader Builder owns the new
  `updatePredicateGroup` operation, which preserves complete shared expansion
  scope and patches within-group AND/OR atomically; Studio preserves
  `applyWhenAbsent` and never derives Boolean scope from numeric group order.
- Datly Reader Builder now canonically inspects and source-preservingly edits
  outer predicate Builder chains: CombineAnd/CombineOr, persistent And/Or
  connectors, exact repeated group occurrences, and WHERE/AND attachment.
  Studio exposes these controls in the Groups workspace; unsupported nested/raw
  chains remain explicitly read-only and group 99 cannot be OR-bypassed.
- Inputs now separate 239 request bindings from trusted deployment constants in
  bounded catalogs. Datly Reader Builder owns structured type/source/default,
  selector, codec, resource URI, and output-emission edits; instance constants
  retain server-owned override precedence and cannot come from invocation input.
- input contracts now also expose canonical absolute-URI route activation,
  named resource URIs, and source-preserving description/example metadata.
  Datly parses `.WithDescription`/`.WithExample`; Studio sends structured field
  mutations and never authors declaration text in the browser.
- schema v7 adds owner-scoped append-only `report_publication_events` plus a
  dedicated Datly reader and `publications.events.list` SDK operation. Successful
  publish/rollback/unpublish events share the activation transaction; failed
  transitions record bounded redacted evidence after recovery. The release UI
  shows this history and requires a reason plus confirmation for destructive
  rollback/unpublish actions.
- resource files, folders, and skill roots now have complete edit/delete UI and
  atomic optimistic concurrency. Every mutation compares `source_revision`,
  changes the resource, increments the revision, and invalidates validation in
  one transaction; stale responses carry expected/current revisions and the UI
  reloads for review. Competing-writer tests prove one winner and one conflict.
- the governed namespace catalog now uses SDK-side query/status/limit/offset,
  bounded pagination, filter-aware empty states, and explicit stale-etag reload
  for edit/delete. Live search narrowed the local catalog to
  `inventory.forecasting` without loading an unbounded namespace list.
- validation now preserves structured Datly compile diagnostics (severity,
  code, hint, line, column) through wrapped runtime-contract errors. The window
  shows exact revision/time, honest stage state, severity counts and hints;
  source navigation remains gated by `canUseDql`. Live Vendor Catalog v2 rev 1
  revalidated successfully without changing the active runtime generation.
- schema v8 adds report ACL row etags across canonical schema, migration, SDK,
  and generated Datly reader/writer metadata. Owner-only update/delete are
  conditional, stale conflicts expose expected/current etags, and competing
  writes yield one winner. The UI groups least-privilege data/author/release
  capabilities and enforces view/edit dependencies on both client and server.
- per-view SQL tabs now show compiled connector/driver, physical source,
  relation scope and keys, contract counts, exact revision and validation state.
  Tab switching, closing, catalog return and browser unload protect unsaved SQL;
  live review confirmed the discard guard without changing persisted source.
- authenticated BFF startup now requires an exact browser origin and non-default
  runtime token, rejects invalid modes, denies proxied `/_studio/*` controls,
  allowlists forwarded headers, requires Origin for cookie-authenticated unsafe
  methods, emits no-store responses, and propagates request IDs.
- The authenticated MCP proxy now preserves `Mcp-Name` for `tools/call` while
  replacing browser-supplied authorization and stripping cookie, admin, and
  development headers. A cookie-authenticated protocol test covers the call
  and unauthenticated denial. A real scoped Datly runtime test also invokes
  its published MCP reader through that BFF proxy and verifies that only the
  authorized project's rows return. Another test exercises every current exact
  native SDK BFF mount and verifies unmigrated operations still fall through
  to the compatibility gateway.
- `datly.yaml` now starts a separate loopback SDK MCP listener on 8090 with a
  protected-resource policy. The authenticated BFF exposes its own
  `/v1/studio/sdk-mcp/` session proxy, distinct from the dynamic reader MCP
  proxy. The combined static-host test verifies every native SDK tool appears
  in network `tools/list`, signed and cookie-proxied `namespaces.list`
  `tools/call` both work, and a call without a bearer is denied. Browser CORS
  preflight includes `Mcp-Name`. Deployments must set the protected-resource
  metadata URL to their public SDK MCP address.

The broad Datly baseline is now green. The repair covered CLI ownership and
filename-controlled writer artifacts, outer/composite SQL aliases, authored
hook and invariant preservation, OpenAPI wire parity, package/resource authority,
query-only schema provenance, to-one mutation writers, initialized builds and
developer MCP workflows. Redundant nested race/build repetitions were removed
from permutation matrices while canonical race coverage and every semantic case
remain; the default transcribe package now has roughly 3m30s timeout margin.

The UI production build emits a current large-chunk warning. It is not a test
failure; code splitting is a performance follow-up.

## Pending work and next goal

### Immediate next goal

Make Datly Studio's reader workflow production ready: finish native SDK reader
contracts and protocol parity, verify authenticated publication/runtime
behavior and close production UX acceptance. The Studio module now uses its
pinned XDatly and SQLX versions: the sibling `main` checkouts do not contain
the v1 contracts required by Datly. The local Datly `v1` checkout remains the
development replacement. `GOWORK=off go test ./...` passed after the correction.
After updating the sibling Forge checkout to its existing `origin/main`,
`npm ci`, all 62 Node and 78 rendered UI tests, and the Vite production build
passed. The build still reports an editor chunk size warning.

The independent GPT-6 Sol UX review resumed after the UI unit suite passed.
Its scoped Skills findings were fixed and re-reviewed with no remaining P1/P2
issue in that flow; evidence is in `ui/UX_REVIEW.md`. Earlier Astra findings
remain historical local evidence. Keep whole-product production acceptance
pending until broader review and deployed checks are resolved.

The native `connectors.create`, `versions.get`, `versions.list`, `versions.export_dql`,
`versions.descriptor`, `versions.download`, `resources.get`, and
`namespaces.create`, `versions.warmup_get`, and `versions.warmup_list` components
are selected by the static Datly host. `versions.get` returns the SDK DTO
directly, including DQL redaction for
view-only subjects. Focused SQLite tests cover owner, delegated view/DQL
access, denial, missing version, revocation, HTTP, MCP, and OpenAPI. The
authenticated BFF proxies its exact route, and generated Go/browser clients
include it. Development mode still uses the generic fallback. The remaining
SDK operations still need native migration.

Complete the one remaining conditional row against a real deployed identity
provider: login callback, authenticated restoration, expiry/re-login, and a
denied-user operation. Perform bounded manual screen-reader announcement and
actual 200% browser-zoom checks in that environment. The separate GPT-6 Sol
review and these deployed checks are distinct acceptance evidence. Local
desktop P0/P1 findings are resolved.
Keep preseeded SQLite for deterministic SDK tests. The Docker/MySQL scale
fixture is an additional, explicitly requested end-to-end exercise.

### Engineering follow-ups

- Runtime MCP contract discovery now uses dedicated streamable HTTP bridge
  methods `tools/list` and `skills/list`. Reader, Cube, and Cube Compose expose
  input and typed structured output schemas. Skills declare every cross-component
  tool they use in portable `SKILL.md` frontmatter (`allowed-tools`); browser and publish-time
  validation reject stale, duplicate, or unknown tool names.
- Datly also exposes `skills/list` and `skills/get` as compatibility tools.
  Agently negotiates `io.modelcontextprotocol/skills` and prefers native
  methods, falling back to those tools only when the extension is absent.
- Runtime consolidates Components, MCP tools, and Resources into searchable,
  paginated tabs. The separate MCP sidebar destination was removed. Skills list
  the live frontmatter contract and open their versioned markup editor.

- Datly 1.0 repair work is committed as `bc590691` and included by remote v1
  merge `fa412c1f`; Studio pins the resulting pseudo-version.
- Commit the Studio UI/SDK integration only after reviewing its uncommitted
  working tree. The intended Git root is now confirmed as
  `/Users/adrianwitas/go/src/github.com/viant/datly-studio`; do not absorb
  concurrent ACL route work into an unrelated commit.
- Focused `ViewOccurrence.SQL` and Studio `viewSourceSQL` regressions, including
  a wrapped legacy query fixture, are implemented and passed on 2026-09-25.
- Expand rendered UI coverage for the remaining complex command dialogs as new
  regressions are found. Catalog search/empty/conflict, component-create failure
  preservation, Schema Browser actions/errors, stale-conflict recovery, inline
  graph selection, SQL dirty navigation, column metadata merge, and semantic
  column contracts now have rendered coverage in addition to the Node SDK suite.
- Decide whether view-wide settings should gain dedicated semantic UI controls
  beyond the current read-only function summary and existing component cache/
  cube dialogs. Do not invent unsupported per-view connector editing.
- Preserve the deterministic preseeded SQLite authoring/runtime proof alongside
  the requested Docker/MySQL scale fixture.
- Address UI bundle splitting if initial-load performance becomes material.

### Product decisions still open

- Which dynamic reader capabilities are exposed by default as MCP, and which
  need explicit publication/approval.
- Whether a business namespace is merely a catalog filter (current behavior) or
  gets stronger governance/ACL semantics.
- Final authenticated deployment configuration: JWKS/certificate source,
  cookie `Secure` policy and dynamic-host authorization mode.
- The initial curated set of Datly projection/view functions exposed as friendly
  controls before relying on the advanced generic function form.

## Handoff cautions

- “Component” is the product/UI term. Preserve internal `reports` names until a
  deliberate storage/API migration; do not rename them mechanically.
- `WithURI` is a reader directive, not a general writer option.
- `#package` is required for every component, including dynamic ones.
- Connected schema discovery is evidence-based. Do not infer PK/FK or types in
  the browser when SQLX metadata is available; use tags only for deliberate
  authoring overrides.
- Output behavior belongs in DQL/type metadata. `internal:"true"`, `json`,
  `sqlx`, case formatting and `output_exclude` have distinct purposes.
- Sparse writes and validation remain generated Datly writer responsibilities;
  use govalidator-compatible lifecycle/invariant patterns, not custom browser
  logic.
- Keep resource state in generated/component embedded resources and DB-backed
  resource storage. Never reintroduce generated stage manifests or
  `.datly-gen.json` files.
- The production matrix contains 23 surface rows with local evidence. The
  independent GPT-6 Sol reviewer must assess that evidence and inspect the
  current UI before its own verdict is recorded. Preserve scenario limits in
  `ui/UX_REVIEW.md`; historical Astra verdicts do not close this review.
- As of 2026-09-25, the native `acl.list` SDK route is implemented. The
  authenticated BFF now forwards that exact SDK path to the static Datly
  host with its session bearer; development mode retains the ACL gate.
  Owner-only policy and HTTP/OpenAPI/MCP behavior have focused SQLite tests.
  A route-scoped Datly OpenAPI export now generates Go and browser clients;
  the browser's ACL-list and Component-list calls use the generated client.
  The native `reports.list` reader now supplies the SDK page shape and keeps
  the verified-auth predicate; its HTTP, OpenAPI, MCP, and scoped SQLite tests
  pass. A dedicated native `reports.get` reader now returns a single report
  DTO or 404 under the same typed authorization and generated-client path.
  `connectors.list` now uses a native reader with server-derived configuration
  flags and no secret material in HTTP/MCP/OpenAPI responses. `connectors.get`
  now uses the same native contract for a single authorized connector or 404.
  The namespace catalog now uses a typed Datly authorization predicate and a
  native `namespaces.list` SDK route with page-shaped HTTP/MCP/OpenAPI output.
  `namespaces.get` now uses the same typed scope for one direct DTO or 404.
  `publications.get` now has a native direct-response reader with typed view
  scope, HTTP/MCP/OpenAPI parity, BFF forwarding and generated Go/JS clients;
  SQLite tests cover owner, delegated viewer, denial and revocation.
  `publications.events.list` now uses a native Datly route with nested SDK
  input, bounded page output, HTTP/MCP/OpenAPI parity, BFF forwarding, and
  generated Go/JS clients. Its typed predicate and the generic fallback both
  enforce current ownership; transfer, delegated publisher denial, filters,
  paging and invalid input have SQLite coverage. The older control-plane
  event reader remains outside the static host exposure set.
  The shared global publish predicate now includes live report ownership and
  rejects grants attached only to deleted reports, matching the SDK authorizer.
  Native authorization-predicate catalog routes remain pending because their
  SDK `linked` flag comes from configured predicate packages, not the SQL row;
  the public DTO must also decode SQL alias/columns metadata safely.
  Version get/list SDK responses redact structural DQL unless the subject
  has `canUseDql`. The source-bearing older static reader still requires that
  permission; the new native `versions.get` uses metadata scope and redacts
  DQL in the authorized view. Native `versions.list` uses the same metadata
  scope and redaction, with nested filters and bounded pagination; its
  HTTP/OpenAPI/MCP and grant/revocation tests pass.
  Native `versions.export_dql` requires the typed DQL grant and preserves the
  SDK's source fallback/empty-export behavior. Its HTTP/OpenAPI/MCP and
  grant-revocation tests pass, with an exact authenticated BFF route and
  generated Go/browser clients.
  Native `versions.descriptor` returns the SDK component/types/resources JSON
  envelope under typed version metadata view scope. Delegated view access,
  denial, missing version, revocation, HTTP/OpenAPI/MCP, exact BFF routing,
  and generated-client tests pass.
  Native `versions.download` is a linked Datly workflow: it authorizes source
  through the existing DQL-scoped version reader, then invokes a newly
  internal-only resource-file reader and assembles the same importable ZIP as
  the generic fallback through shared code. HTTP/MCP/OpenAPI, denied/revoked
  DQL access, internal-route isolation, named factory linking, and the browser
  SDK route have focused tests. A private aggregate preflight now rejects
  downloads over 2,000 resource files or 32 MiB of resource content before
  reading blobs. The reader caps rows at 2,001; assembly enforces 4 MiB DQL
  source and 16 MiB ZIP limits. Oversized native and generic SDK downloads
  return invalid-input errors in focused tests.
  Native `resources.get` reads the authorized version before invoking three
  internal-only snapshot readers for files, folders, and skills. The native
  and generic paths share DTO projection. A 137-file SQLite contract checks
  ordering, delegated view access, DQL redaction, denied/revoked access,
  HTTP/MCP/OpenAPI and private child-route isolation. The browser resource
  workspace now calls the generated client through the exact BFF mount.
  Native `versions.warmup` is now a linked Datly workflow with a verified
  publish guard, exact version lookup, active-key deduplication and a durable
  accepted-run write. The asynchronous worker waits for that commit, then
  uses the existing generated Datly store and bounded runtime warmup executor.
  Combined-host tests cover signed HTTP, direct MCP, authenticated BFF MCP,
  denied callers, deduplication, and accepted-to-terminal progression. The
  browser uses the generated client and exact BFF mount.
  Native `versions.warmup_get` returns one durable warmup run under the typed
  publish-capability predicate, including audit fields, duration, target and
  diagnostics. SQLite HTTP/MCP/OpenAPI tests cover owner/delegated publisher,
  view-only denial, missing report/run and revoked grants. The browser uses
  the generated client and exact BFF mount. Native `versions.warmup_list`
  authorizes the requested report with a private typed publish guard before
  global expired-run recovery through internal CAS components. Its 106-run
  batch test verifies viewer denial before recovery, system audit attribution,
  paging, revocation, HTTP/MCP/OpenAPI and private child-route isolation.
  Native `namespaces.create` is a generated mutation writer with a flat SDK
  body. Its input initialization binds owner to verified JWT identity and
  overrides status, etag, and timestamps; focused SQLite HTTP/MCP/OpenAPI
  tests cover persistence, forged-owner denial, invalid names, and duplicate
  conflict. Datly's duplicate-key response mapping now returns 409, and its
  explicit custom-JSON wire type preserves a precise OpenAPI response schema.
  Native `connectors.create` is also a generated mutation writer: it binds
  owner and draft/revision metadata server-side, while its direct SDK response
  exposes only configuration flags, not DSN or secret-reference values.
  Focused SQLite HTTP/MCP/OpenAPI tests cover persistence, forged owner,
  duplicate conflict, secret redaction and static writer selection.
  No deployed Studio browser tab, deployed URL, or test-identity configuration
  was available during the 2026-09-25 local audit. Actual deployed IdP,
  screen-reader, and 200% Chrome zoom acceptance evidence remains pending;
  local viewport scaling is not a substitute. A later actual local Chrome
  200% zoom pass covered Overview and the Vendor graph/Input/Output flow and
  fixed a clipped graph count; it remains local evidence only.
  The dynamic host now rejects mutation settings and mutating HTTP methods
  before registering a published component. A failed PATCH-route reload left
  the prior GET reader and serving generation intact in a SQLite host test.
  The Input catalog's nested tabs now have roving keyboard focus and a
  labelled tab panel; local browser accessibility-tree/Right Arrow checks
  passed. Spoken screen-reader announcements remain unverified.
  A static-host contract test now verifies that each selected DQL MCP tool
  matches the linked component metadata. The two older, unselected catalog
  readers remain explicit exceptions rather than being counted as live tools.
  The exact `datly.yaml` package set also eagerly bootstraps as one generation
  against a fresh SQLite catalog and local RSA verifier. It selects every
  generated native SDK OpenAPI path exactly once with the matching MCP name;
  a bearer-authenticated namespace create/list round trip succeeds in that
  combined host, and an internal snapshot child route is not public.
  The remaining SDK operations still use the generic dispatcher.
  See `docs/sdk-native-endpoints.md` for the remaining broader SDK migration.
  Focused
  Datly embedded-view SQL and rendered UI wrapped-query regressions pass.
  The local dynamic host was restarted on 2026-09-25 with the preseeded SQLite
  configuration; its HTTP and MCP listeners answered on 8082 and 8091, its
  status endpoint returned ready, and MCP tools/list returned published tools.
  Publication reloads this existing host; it does not launch either listener.
  This local proof does not satisfy the pending deployed IdP or 200% zoom rows.

### Docker scale and MCP acceptance — 2026-09-25

- The original Datly `origin/master` e2e MySQL schema, one-to-many DQL and tree
  case informed `e2e/testdata/scale_reader_mysql.sql`. Docker MySQL 8.4 was
  loaded with 20 views across five hierarchy levels and a 60-column table;
  the fixture and runbook are in `e2e/scale/README.md`. The Studio SDK authored,
  validated, previewed and published hierarchy and wide readers. View 20
  filtering returned its two fixture rows; the nested preview remained intact.
- Live browser inspection found the compact 20-view navigator, searchable
  lineage catalog, 100-column Output catalog, and 60-column wide-view catalog.
  A primary-key-first inspector fix and projection-aware Output fallback were
  added. Full UI tests and production build pass. After that gate, independent
  GPT-6 Sol review found no remaining P1/P2 issue in this scoped change.
- The live runtime exposed and invoked all five declared tools: hierarchy
  reader, wide reader, cube reader, Cube, and CubeCompose. The repeatable MCP
  probe is `e2e/testdata/verify_scale_mcp.mjs`; the authenticated BFF wrapper
  is `e2e/scale/verify_bff_mcp.go`. Both paths invoked all five published
  tools, and the wrapper denied a cookie-less request. The full seed and verifier
  passed again against a fresh catalog. Every selected native SDK MCP
  declaration is checked against linked component metadata, and each of the
  then-nineteen native SDK routes had focused MCP invocation parity coverage.
  `reports.create` became the twentieth route on 2026-09-26. These
  local checks do not substitute for deployed static-host/IdP acceptance.
- A fresh Docker-backed `scale-filter-studio.db` catalog on 2026-09-26 now
  exercises actual reader filtering as well as UI catalog filtering. The seed
  adds optional root/row ID predicates through Reader Builder before
  validation and publication. Direct MCP and authenticated BFF MCP both list
  the typed `RootID`/`RowID` arguments and return exactly one selected
  hierarchy root (with its level-five descendant) and one complete 60-column
  row; the other three Cube tools still pass. MCP uses the authored DQL field
  names, whereas HTTP uses the lower-camel query names.
- Validation and Reader Builder responses on the generic SDK path no longer
  return authored/generated DQL or source-bearing diagnostic text to a
  delegated editor without `can_use_dql`. The native `versions.get` and
  `versions.list` SQL readers also suppress stored compile diagnostics for
  that caller. Focused SQLite tests cover the grant boundary; native
  `versions.builder` is still missing a release route.
  `GOWORK=off go test ./...` passed for the full Studio module after this fix.
- A fresh file-backed Studio database exposed missing SQLX sequence-ledger
  initialization; migration now creates or repairs it even when schema version
  is current. First-namespace authorization was also fixed for the actual
  create operation. Both have regression tests. The full Studio and sibling
  Datly `GOWORK=off go test ./...` suites passed after these changes.

### Download resource budget — 2026-09-26

The native and generic SDK `versions.download` paths now use the same private
aggregate budget reader before fetching resource blobs. Focused tests cover
oversized native HTTP and generic SDK downloads, archive hard limits, and
private-route isolation. The resource reader has a 2,001-row cap as a race
safety check. `GOWORK=off go test ./...` passed for the full Studio module after
these changes; no UI files changed in this slice, so the UX-review trigger was
not met. The local Datly module replacement and deployed IdP/whole-product
acceptance remain release gates.
The combined static-host network test also verifies that the authenticated
SDK MCP proxy lists every generated native SDK tool and rejects a cookie-less
catalog request. The retained Docker 20-view/60-column fixture was rechecked
on 2026-09-26: direct dynamic MCP and authenticated BFF MCP both invoked all
five published reader/Cube/Cube Compose tools successfully. This is local
fixture evidence, not deployed identity-provider acceptance.
`bash -n scripts/verify-production-build.sh` passed, and running the release
gate correctly stops at the current `../datly` replacement. The gate must not
be reported green until it runs against a clean, immutable Datly dependency.

### Native warmup and MCP recheck — 2026-09-26

Native SDK coverage is now 51/65: `versions.warmup` is selected for HTTP,
direct SDK MCP, authenticated BFF SDK MCP, OpenAPI, and the generated browser
client. The combined static-host test verifies durable acceptance, active-key
deduplication, unauthorized denial, and accepted-to-terminal progression;
the cache-enabled fixture also reaches `completed` with all planned cases;
its Docker branch again passed the 20-view/60-column connector, preview,
validation, and static MCP checks. UI unit/render tests and production build
passed. The retained Docker dynamic fixture again passed all five direct and
authenticated BFF MCP tools, including both filtered reads. Fourteen native
SDK operations remain, as does the deployed identity-provider acceptance gate.
The immutable Datly dependency gate was cleared after this dated slice.
No substantive UI layout or interaction
changed in this slice, so no additional UX review was requested.

### Datly v1 selector and response-format contract — 2026-09-26

Datly v1 now auto-enables each explicitly declared `QuerySelector(view)`
property, including `Limit` bound from a POST body. Studio's 14 affected DQL
readers no longer repeat matching `selector_*` permissions; their base limits
and ordering allowlists remain. Datly commit `aba4f6d1` includes this contract
and the earlier reader fixes. The custom JSON wire declaration was renamed to
`JSONWireType()` in Datly commit `0005e9bb`; Studio implements the new method
but this larger Studio worktree is not committed.

Datly v1 also accepts one explicitly declared `FormatSelector()` string input
from `header/Accept` or a query key such as `_format`, with JSON as the
unchanged default. Header selection honors media quality values and OpenAPI
lists representable JSON/CSV/XML/XLSX responses. Legacy `_format` continues
when no selector is declared. JSON-only custom outputs reject alternate
formats before mutation execution; Studio's connector-create regression checks
that the rejection does not persist a row or reveal DSN/secret fields. Focused
Datly HTTP/OpenAPI/output/transcription tests, the full Studio Go suite, and
the Docker-backed combined static-host test passed. Both full Go suites passed
on the final source, and the format selector was committed to Datly v1 as
`72521de9`. The Datly worktree is clean; Studio's broader worktree remains
uncommitted.

### Immutable Datly dependency — 2026-09-26

The pushed Datly `origin/v1` head is `72521de9c511b8be583d4d7b500c648309471401`.
Studio now requires its Go pseudo-version
`v1.0.1-0.20260926140723-72521de9c511` and has no Datly `replace` directive.
`GOWORK=off go mod verify`, the full Studio Go suite, the UI unit/render suite,
and the Vite production build pass with that immutable module. The native SDK
coverage gate remains 51/65, so the full production-build script still fails
before the Go binary-build stage. Deployed IdP and whole-product UX acceptance
also remain open; this dependency update does not clear those gates.

### Explicit link sync and shared native mutation transaction — 2026-09-26

Datly `v1` commit `6ef09ac0` adds explicit `datly link sync`. It scans selected
Go source with AST for tagged component holders and predicate/codec interface
implementations, adds only missing blank imports to the project-owned
`internal/datlylink/link.go`, and supplies package-local type reachability and
`init` when needed. Package selection does not depend on a `DatlyLinkedType`
marker or an explicit type registry. The full Datly suite passed and this
commit is on `origin/v1`.

Datly `v1` commit `49abd3d0d7994639b8acaf3cff25ad54f19832bb` makes an
imperative child writer flush its buffer into the root-owned transaction before
returning. Child SQL readers resolve that same transaction by exact database
identity, including during lazy connector resolution; cache, partition, and
warmup guards use the resolved transaction. The root endpoint alone commits or
rolls back. Datly's full `GOWORK=off go test ./...` suite and module verification
passed. This commit is local only: HTTPS push from this host failed because no
GitHub credential is available. Studio's `go.mod` still pins `72521de9`; do not
claim an immutable production dependency until `49abd3d0` is pushed and pinned.

Studio's six `resources.upsert_*`/`resources.delete_*` routes now use their
existing generic Datly readers and writers through the parent component
invocation. The private store routes are selected as internal-only; public
HTTP, SDK MCP, OpenAPI, and generated Go/browser clients expose the six SDK
operations. The combined static-host test exercised file, folder, and skill
upsert/delete through HTTP, direct MCP, and authenticated BFF MCP. A failed
mutation after the version touch left the stored source revision unchanged,
proving root rollback. The full Studio Go suite passed against the local Datly
checkout using a temporary Go workspace; UI unit/render tests and Vite build
passed. Native SDK coverage is 57/65. The remaining eight operations are
`access.context`, `access.get`, `access.replace`, `publications.publish`,
`publications.rollback`, `publications.unpublish`, `runtime.status`, and
`versions.builder`. The production coverage gate and deployed IdP acceptance
remain open. No substantive UI interaction changed in this slice, so no new
UX review was requested.

`versions.builder` now uses the same parent-owned transaction path for its
existing generic version and report config writers. The combined host passed
inspect, a package mutation touching both rows, stale revision rejection with
the `sourceRevision` field, direct SDK MCP, and authenticated BFF SDK MCP.
OpenAPI and generated clients include the route, and the browser adapter calls
the generated client. Coverage is now 58/65; seven operations remain:
`access.context`, `access.get`, `access.replace`, `publications.publish`,
`publications.rollback`, `publications.unpublish`, and `runtime.status`.

Datly transcriber commit `2719985b` now emits `reflect.TypeFor` reachability
for the actual component and lifecycle types without generating any
`DatlyLinkedType` variable. The local `v1` branch merged the independent remote
selector/SQLx updates at `5a4fb775662965039eed42dd68cbe4481b354241`.
The complete Datly `GOWORK=off go test ./...` suite and module verification
passed on that merge; the complete Studio Go suite passed against it through
the temporary local Go workspace. The remote `v1` branch is now confirmed at
`5a4fb775`, but Studio still pins the older `72521de9` revision. The seven
remaining native operations require a
decision on moving the ACL verifier and dynamic runtime admin client into the
static Datly host; that deployment-boundary question has been sent to the user.

`runtime.status` now has a selected Datly HTTP/OpenAPI/MCP contract and an
authenticated BFF mount. It reuses the existing SDK runtime projection and
publisher authorization; a shared server-side runtime-admin client supplies
the same live host probe used by the BFF. Configure the static host with
`STUDIO_RUNTIME_ADMIN_TOKEN` and, if the dynamic host is not at the local
default, `STUDIO_DYNAMIC_HTTP_URL`. A missing token yields an unavailable
host check, never a false ready state. Combined-host tests cover signed HTTP,
publisher denial, direct/BFF MCP, idle state, and live host readiness. Coverage
is 59/65; the three `access.*` operations and three publication mutations
remain.

The publication lifecycle now has native `publications.publish`,
`publications.rollback`, and `publications.unpublish` HTTP/OpenAPI/MCP routes.
Their verified JWT principal is passed to the existing generic publication
service, which remains the sole owner of stage/commit, dynamic reload,
activation, and compensation. This multi-commit lifecycle is intentionally
isolated from Datly's single-commit endpoint transaction; carrying that root
transaction into the existing service produced a deterministic conflict.
Combined-host tests cover each operation through signed HTTP and direct or
authenticated-BFF MCP, non-publisher denial, stale source revision, failed
runtime reload, and persisted active-generation transitions. Coverage is now
62/65; only `access.context`, `access.get`, and `access.replace` remain.
Studio now pins remote Datly `v1` commit `5a4fb775` as
`v1.0.1-0.20260926192031-5a4fb7756629`, with no local replacement.

`access.context`, `access.get`, and `access.replace` now complete the 65/65
native SDK route inventory. Their Datly handlers verify the same bearer under
both the static host's JWT policy and the dedicated ACL issuer/audience/key,
then invoke the existing generic ACL service. The resource-policy reader and
writer are linked as private child components; the writer participates in the
Datly endpoint's root transaction. Configure the static host with
`STUDIO_ACCESS_ISSUER`, `STUDIO_ACCESS_AUDIENCE`, and
`STUDIO_ACCESS_PUBLIC_KEY_FILE` (the BFF flags default to those same values).
Combined-host tests cover signed HTTP, direct/BFF MCP, owner access, context,
optimistic revision advancement, stale conflict, non-owner denial, wrong ACL
audience, and private policy-route isolation. All 65 public SDK operations now
have generated native HTTP/OpenAPI/MCP contracts. The clean-dependency
`scripts/verify-production-build.sh` gate passed locally: module verification,
all Go tests and three binaries, lockfile-installed 76 Node and 78 rendered UI
tests, and the Vite production build. Generated Go/browser clients reproduce
from `scripts/generate-studio-sdk.sh`. Deployed identity-provider integration,
whole-product UX acceptance, and the separate legacy `reports` rename remain
open; this local build is not a deployment approval.

Datly `v1` commit `f3c88b304de8` is now on the remote and is pinned by
Studio as `v1.0.1-0.20260926223102-f3c88b304de8`. It changes the default
project link package to `internal/dependencylink`; `datly init`, `datly build`,
and explicit `datly link sync` accept `-link-package internal/<name>` for a
different package. Studio already uses the new default and keeps its link file
limited to blank imports. The older `internal/datlylink` references above are
historical snapshots, not the current default.

Studio's own `cmd/datly` executable now exposes the explicit `link sync`
command against that default path (with the same optional `-link-package`
override). Ordinary run and transcribe commands do not scan or edit links.

The breaking component-catalog rename removes public `reports.*` SDK routes
and Go/browser SDK names in favor of `components.*` and `Components()`. Native
Datly HTTP, OpenAPI and MCP contracts are regenerated under the new names;
the authenticated BFF no longer mounts a generic SDK fallback for old paths.
SQLite canonical schema version 15 names the catalog table `components` and
renames an existing `reports` table in place before later upgrade steps, so
rows and child foreign keys survive. A version-14 fixture verifies that path.
The development-only generic SDK gateway and internal `report_*` relation and
package names remain separate cleanup work; they must not be mistaken for a
completed whole-schema rename or a production route.

Studio now has an opt-in direct identity-token mode. `studio-api -login-id-token`
keeps authorization-code/S256 PKCE, the provider client
secret, and refresh token on the backend. Its encrypted Datly-backed
`bff_sessions` row holds the ID and refresh tokens under a hashed opaque
cookie ID; `POST /v1/studio/auth/token` returns only the verified ID token and
subject. The browser keeps that ID token only in memory, sends it as Bearer to
static Datly SDK and dynamic MCP, and omits cookies on those calls. In this
mode the Studio host does not mount its legacy SDK/MCP/runtime proxies or
session exchange. UI and auth broker must be the same origin; static and
dynamic Datly may be separate origins with appropriate CORS. The scoped UX
review identified the split-origin sign-in/cookie issue, now prevented by the
same-origin check. Full Go tests, 80 Node tests, 80 rendered UI tests, and
the Vite build pass locally. A live identity-provider deployment test, Datly-side
issuer/audience enforcement for the ID token, and production CORS deployment
configuration remain required before release.

The auth-only login, callback, token and logout routes now dispatch through
typed Datly custom HTTP handlers. Their contracts bind the code/state, cookies
and Origin, and return buffered status/headers so Datly writes the redirect and
Set-Cookie values. They have no MCP exposure. The legacy BFF flow remains
available only when explicitly selected; the identity-token mode never mounts
those BFF routes. The auth store still uses private Datly reader/writer
components and an encrypted database payload.

Datly `v1` has a tested `JWTClaims` issuer/audience/subject policy for the
declared `JwtClaim` codec. Studio now pins the immutable Datly revision
`26d49975` and the sample static host enables that policy for the Studio ID
token issuer and audience. The dynamic host also checks default identity
issuer/audience for HTTP and MCP. The three resource-policy SDK routes can
resolve account-scoped facts from the optional user-info adapter; the legacy
tenant-bearing ACL-token path remains available only if its issuer/audience
also satisfies the static host's outer JWT policy. No external identity-
provider code is committed as part of Studio.

The static and dynamic sample hosts now declare exact, noncredentialed CORS
for the Studio UI origin; the dynamic host also binds its default JWT issuer
and audience. The current production-build gate and Docker-backed 20-view,
60-column native SDK integration test pass. An independent GPT-6 Sol local
browser pass reviewed direct-auth states, graph/SQL navigation and 60-/100-
column Output catalogs at desktop and 390 px. Its initial split-origin
concerns were withdrawn after checking the enforced same-origin UI/auth
contract; no local P1/P2 issue remains in those reviewed states. Deployed IdP,
denied-user, spoken screen-reader and actual deployed 200% zoom acceptance
remain unverified. The ACL-token model and publishing the local Datly revision
remain open.

The dynamic MCP transport's streamable OPTIONS handler overwrote the exact
configured CORS origin with `*`. Studio's dynamic host now owns that browser
preflight, returning the configured origin and rejecting unapproved origins,
methods and headers; signed MCP requests still use the normal transport.
Static Datly and dynamic MCP preflight have live handler tests. The
production-build gate passed again after this CORS fix.

A newer published-reader integration test now signs Studio-audience ID tokens
against a local JWKS and exercises the dynamic HTTP reader plus its actual MCP
tool call. The matching issuer/audience/subject succeeds; wrong issuer,
audience, or missing subject returns 401 on both HTTP and MCP. The test also
checks exact browser origin on successful and denied responses. It revealed
that ordinary MCP responses and pre-auth HTTP denials could still emit `*` or
omit CORS despite exact configuration; the dynamic host now enforces its
configured CORS at the outer response boundary for both protocols. The
reader's own run ACL remains authoritative after the token is accepted.
The direct MCP catalog and tool-call responses were also checked for
`Set-Cookie`; neither emits one for the bearer-token path.

The auth-only broker now coordinates refresh tokens across Studio
instances. Schema version 16 adds an owner/expiry lease to `bff_sessions`; a
private Datly custom writer uses one atomic managed SQL update to acquire it,
fences completion by owner, commits the encrypted replacement payload, and
releases it. A follower waits for that replacement rather than calling the
provider with the same refresh token. A two-instance SQLite test exercised
separate database handles and Datly runtimes, observed exactly one provider
refresh, and passed 60 repeated runs; the auth/migration race-detector suites
passed. The version-15 upgrade test retains the existing ciphertext and
initializes an empty lease. A transaction-ownership conflict was found when
the lease component carried both an eager DataSource and transaction-scoped
SQL; leaving its root source neutral resolved the conflict without a second
transaction owner. Production multi-pod/real-IdP acceptance is still pending.

Direct identity mode now exposes a visible Sign out action. It revokes the
opaque cookie session through the auth-only Datly logout handler before
clearing the page-memory ID token; failure keeps the signed-in UI and offers
an explicit retry. A generation check prevents an in-flight refresh from
repopulating memory after logout. The auth broker accepts ID tokens with up
to 60 minutes remaining lifetime, as explicitly requested. A copied stateless
token can remain valid at Datly after the refresh session is revoked. The
one-hour default (`ID_TOKEN_TTL_SECONDS=3600`) is now accepted;
the generic provider code remains unchanged. An independent GPT-6 Sol review
of Sign out at desktop/390 px found no local P1/P2 issue after 82 Node and
82 rendered UI tests passed. Deployed logout and spoken screen-reader behavior
remain unverified.

After the 60-minute ID-token policy was committed, race-enabled tests for the
dynamic host and linked static SDK host passed (`go test -race ./runtime/host
./cmd/datly`). The local linker emitted a macOS `LC_DYSYMTAB` warning for the
static test binary, but the command exited successfully. This is additional
concurrency evidence, not a replacement for deployed publication or IdP tests.
Race-enabled tests for publication and SQL transport also passed (`go test
-race ./studio/report_publications/... ./sdk/transport/sql/...`), with the same
non-fatal macOS linker warning on one test binary. The independent GPT-6 Sol
reviewer could not establish actual Chrome 200% zoom with the permitted browser
controls; manual or deployed verification remains open.

The external user-info contract must keep role grants distinct from feature
exposures; Studio must not treat a feature as a role. The user confirmed
`allowedEntities` is not needed now, so no arbitrary project/entity access is
inferred. Direct ID-token ACL access is opt-in through the trusted user-info
adapter; live provider and exact account/tenant deployment acceptance remain
open. Native authorization-predicate handlers now share the same trusted
package-path allowlist as embedding hosts and discover linked handler types
without per-type registration. A combined-host test covers an extension type
over HTTP and MCP, creation while linked, and `linked:false` after removal.
The dynamic published-reader test now adds the execution proof: without the
allowlist its handler predicate fails during generation load, while the same
linked package executes a filtered reader through HTTP and MCP when allowed.
The read-only `scripts/verify-live-reader.mjs` acceptance command is ready for
a deployed Studio URL and short-lived ID token. It checks protected SDK
access, exact bearer CORS/no cookies, static MCP discovery and invocation,
and one known published reader value over HTTP and MCP without printing the
token. Seven mocked transport tests run in the production-build gate. It has
not yet been run against a deployment and does not replace live
callback/logout or manual assistive-technology evidence.
The refreshed `datly link sync` pass added missing blank imports and
package-local init/type anchors; a second pass added nothing. A regression
test now requires `internal/dependencylink/link.go` to contain only one block
of blank imports. The full production-build gate passes on the immutable
Datly `26d49975` pin (65/65 native OpenAPI paths, all Go tests/binaries,
83 Node tests, 82 rendered UI tests, and Vite build). The opt-in Docker MySQL
static HTTP/direct MCP/BFF MCP test again passed with 20 SQL views and the
60-column table. Race-enabled static host, predicate catalog and host-config
tests pass; the macOS linker emitted its known non-fatal `LC_DYSYMTAB` warning.

The 65/65 standalone coverage script now labels its measurement as generated
OpenAPI routes, not MCP execution. The combined-host test additionally rejects
duplicate or unexpected live `studio.sdk.*` tools while retaining its
one-route/one-tool checks for every generated SDK path. Its focused test passes.

The broker's cross-instance refresh follower now detects a newly committed ID
token, not only a changed refresh token. Two-instance tests pass for rotated,
unchanged, and omitted refresh-token responses, with exactly one provider
refresh; the race-enabled auth suite also passes. A Datly session-store test
confirms that an in-flight refresh lease cannot recreate a session after
logout. The read-only live verifier now invokes the native SDK MCP tool,
requires exact browser-safe 401s on SDK and published-reader HTTP, and selects
the matching JSON-RPC response after stream notifications without echoing
malformed payloads. Its seven mocked tests and the full production gate pass.
The current-schema Docker fixture publishes the 20-view hierarchy, 60-column
reader, and Cube/Compose probe; all five tools passed direct and authenticated
BFF MCP execution on 2026-09-27. None of this closes deployed IdP, refresh,
logout, denied-user, screen-reader, or actual 200% zoom acceptance.

An optional generic user-info ACL provider now verifies the browser ID token
and obtains current roles/features from a deployment-owned endpoint. It
requires matching signed/returned user and account IDs, maps features to
exposures and the signed account ID to the tenant, bounds remaining token life
to 60 minutes, rejects redirects and malformed authority data, and grants no
entity IDs. The dedicated ACL-token provider remains the default. The static
native `access.*` HTTP/MCP route and dynamic host have signed opt-in tests;
the native bound access context now uses one facts/decision snapshot rather
than resolving the provider twice. A deployed identity-provider, exact tenant
configuration, and denied-user acceptance flow remain unverified.
The selected static-host test also signs wrong-issuer and wrong-audience ID
tokens and verifies native HTTP and MCP ACL calls deny them without contacting
the user-info endpoint.
A published dynamic reader is now exercised with the same opt-in account-scoped
user-info facts on real HTTP and MCP listeners. Its exposure policy permits
both transports while the feature is present; removing the feature immediately
denies HTTP and MCP without releasing rows, and a returned account mismatch
denies HTTP. The complete runtime-host suite and this test under the race
detector pass. This is local signed fixture evidence, not deployed provider
acceptance.

Direct Datly JWT verification does not make a per-request identity-provider
session-status call; the user explicitly accepted a 60-minute stateless JWT
window instead of immediate revocation. The accepted lifetime bounds the
window after logout but does not invalidate copied tokens immediately.
Deployed identity-provider, refresh, and logout behavior remains unverified.

## Primary references

Product scope correction: Datly Studio does not expose presentation reports.
The `reports`/`report_versions` names in storage and SDK code are historical
component-record identifiers. Presentation reports are outside this project.
Datly Studio authors MCP tools and Skills from Datly components; an optional UI
definition is future work as a typed, versioned artifact over a validated
component descriptor, not a new report engine. User-visible copy and future
endpoint planning should respect this boundary.

- [Architecture and Datly findings](studio.md)
- [UI product design](ui.md)
- [UX verification log](ui/UX_REVIEW.md)
- [Static component configuration](datly.yaml)
- [Dynamic runtime configuration](datly-runtime.yaml)
- [Datly 1.0 Reader Builder](../datly/authoring/readerbuilder)
