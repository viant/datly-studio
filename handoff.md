# Datly Studio handoff

Updated: 2026-09-20

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

The Studio module has a deliberate local development replacement:

```text
github.com/viant/datly => ../datly
```

Run Studio Go commands with `GOWORK=off` unless the workspace is explicitly
configured; otherwise Go rejects the module due to the parent `go.work`.

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
  token only to the dynamic Datly route/MCP proxy.
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

As of 2026-09-25, `claude` 2.1.281 is installed, but `claude auth status`
reports `loggedIn: false` and `authMethod: none`, so a Fable 5.1 review still
cannot be run from this workspace. Do not substitute Astra approval.
The product owner has asked to defer Claude/Fable review for now; keep its
acceptance row pending rather than treating the deferral as approval.
The remaining reviewer evidence therefore requires that reviewer to be made
available or run in its authorized environment.

Complete the one remaining conditional row against a real deployed identity
provider: login callback, authenticated restoration, expiry/re-login, and a
denied-user operation. Perform bounded manual screen-reader announcement and
actual 200% browser-zoom checks in that environment. Earlier Astra review
labels are superseded and must not be treated as UX approval; independent
reviewer evidence remains pending. Local desktop P0/P1 findings are resolved.
Keep preseeded SQLite as the explicit
local test target; do not silently replace it with Docker/MySQL.

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
  `/Users/awitas/go/src/github.com/viant/datly-studio`; do not absorb
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
- Preserve the deterministic preseeded SQLite authoring/runtime proof until the
  user explicitly requests a different connector environment.
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
- Claude Fable 5.1 was the preferred UX reviewer, but no Claude CLI or
  authenticated Claude surface was available. The product owner explicitly
  authorized Codex Astra low as the substitute. Record Astra verdicts as Astra
  rather than Fable approval and preserve scenario limits in `ui/UX_REVIEW.md`.
- Reviewer availability was re-audited on 2026-09-20 after the local completion
  pass: neither `claude` nor `fable` is installed, no Claude/Fable application is
  available, and no authenticated Claude browser tab is open. The production
  matrix contains 23 surface rows; all 23 have local evidence and all 23 retain
  an explicit pending Fable 5.1 verdict rather than a substituted approval.
- On 2026-09-25 the Claude CLI appeared on the host but remained signed out;
  this changes availability, not the pending Fable verdict. The native
  `acl.list` SDK route is under contract review in the working tree. The
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
  Version get/list SDK responses now redact structural DQL unless the subject
  has `canUseDql`; the source-bearing static version reader requires that
  permission until a native metadata-plus-redaction contract is authored.
  No deployed Studio browser tab, deployed URL, or test-identity configuration
  was available during the 2026-09-25 local audit. Actual deployed IdP,
  screen-reader, and 200% Chrome zoom acceptance evidence remains pending;
  local viewport scaling is not a substitute.
  The remaining SDK operations still use the generic dispatcher.
  See `docs/sdk-native-endpoints.md` for the remaining broader SDK migration.
  Focused
  Datly embedded-view SQL and rendered UI wrapped-query regressions pass.
  The local dynamic host was restarted on 2026-09-25 with the preseeded SQLite
  configuration; its HTTP and MCP listeners answered on 8082 and 8091, its
  status endpoint returned ready, and MCP tools/list returned published tools.
  Publication reloads this existing host; it does not launch either listener.
  This local proof does not satisfy the pending deployed IdP or 200% zoom rows.

## Primary references

- [Architecture and Datly findings](studio.md)
- [UI product design](ui.md)
- [UX verification log](ui/UX_REVIEW.md)
- [Static component configuration](datly.yaml)
- [Dynamic runtime configuration](datly-runtime.yaml)
- [Datly 1.0 Reader Builder](../datly/authoring/readerbuilder)
