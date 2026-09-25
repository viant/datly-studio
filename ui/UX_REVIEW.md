# Studio UX review gate

> Status: prior Astra-labelled findings are retained only as historical defect
> notes. They are not independent UX approval and must not be used to green-light
> any Studio surface. Record future approval only from the requested reviewer.

## GPT-5.6 light findings — Runtime and Skills, 2026-09-20

This was a critique, not approval. It identified stale MCP refresh behavior,
silent tool-catalog failure, unvalidated skill tool declarations, root-level
`SKILL.md` detection, editor label association, and unnamed action headers.
All six findings were addressed with rendered regression coverage; independent
reviewer approval remains pending.

Run this review before declaring a Studio Forge UI surface complete.

### Shared ACL report policy review — local check, 2026-09-24

- The generic Permissions workspace and its dedicated review frame now include
  report resources. `preview` and `execute` are separate declared actions; a
  custom resource kind requires its own action contract.
- At 390 px, the report policy frame was inspected after selecting `preview`.
  A labeled action selector replaces the clipped horizontal action rail. The
  role/exposure rule, required project scope, and Save state remain visible.
- The review frame uses in-memory data and issued no Studio API request. This
  is local layout evidence, not IdP/BFF authorization or production approval.
- The production editor now opens a pre-save review dialog for changed actions.
  At 390 px, its current/proposed rule, entity scope, exact policy revision,
  and Save/Back controls remained readable. Cancel preserved the draft; the
  conflict path still preserves edits for explicit reload. This is structural
  policy review, not a client-side effective-access calculation.

## Menu and workspace

- The current location is apparent and every visible menu item opens a useful SDK-backed surface.
- Navigation remains usable when collapsed, filtered, and on a narrow viewport.
- The workspace has a clear title, purpose, and primary action.
- Loading, empty, unavailable, unauthorized, and failed states explain what happened and the useful next action.

## Individual windows

- Form labels, defaults, validation messages, destructive actions, and async pending states are unambiguous.
- Keyboard focus order, button names, table headers, and status/error announcements are usable.
- Dense tables retain identity columns and preserve essential actions at narrow widths.
- Every read and mutation is traceable to one public Studio SDK operation; no UI state writes directly to SQL, DQL, or a generated Datly package.

## Evidence

- Run UI unit tests and a production Vite build.
- Inspect the completed menu, main layout, and each changed window at desktop and narrow widths.
- Record any accepted limitation next to the corresponding component or window metadata.

## Production-readiness review in progress — 2026-09-20

The following surfaces passed local functional and visual inspection. Fable 5.1
remained unavailable; the product owner subsequently authorized Codex Astra low
as the available substitute. Astra approval is recorded explicitly below and is
scoped to inspected scenarios rather than treated as blanket Fable approval.

### Codex Astra low substitute review — approved targeted changes, 2026-09-20

- The product owner authorized Codex Astra at low reasoning effort when Fable
  remained unavailable and supplied three explicit requirements to the reviewer:
  remove the noisy SQL metadata strip, place global identity/connector/cube/MCP
  controls in Edit component, and expose MCP Catalog plus Skills & Resources as
  visible destinations.
- Two isolated Astra assessments were run under the Impeccable critique method.
  The first pass rejected production approval for invalid tab semantics, an
  unnamed real SQL editor, nested graph controls, repeated column selection,
  and an unsafe release comparison. Every reported P1 was implemented and sent
  back for independent re-review.
- Final targeted verdict: **APPROVE** from the design reviewer and **APPROVE WITH
  CONDITIONS** from the technical reviewer, with zero Impeccable detector
  findings. Live evidence confirmed real tabs/tabpanel keyboard behavior, named
  CodeMirror, separate graph controls, direct column editing, cohesive Edit
  component, explicit MCP/skills catalogs, and serving-to-proposed release state.
- Follow-up fixes require exact nonempty publication snapshot hashes before
  claiming equivalence, stage advanced tags until Save, and provide draggable
  plus Minimize/Balanced/Maximize SQL pane controls. Direct rendered regression
  tests cover same-version hash mismatch, staged-tag Cancel/reopen, and SQL pane
  sizing.
- Approval is scoped to reviewed desktop changes. Authentication/denied flows,
  deployment configuration, exhaustive failure recovery, dark theme, contrast,
  zoom, and whole-product screen-reader evidence remain separate production
  conditions; no untested state is approved by inference.
- Follow-up visual polish removed the competing pastel fill from every sidebar
  destination and binds Blueprint's selected-tree semantics to the actual
  workspace section. One neutral hover and one high-contrast selected treatment
  now communicate location; a rendered navigation assertion protects it.
- Final 23-row Astra matrix review now records **22 APPROVE** and **1 APPROVE
  WITH CONDITIONS**, with zero unreviewed surfaces and no remaining P0/P1
  desktop design blocker. Follow-up live review approved schema root/subview
  dialogs, populated predicate Groups, enabled composition with result data, and
  three completed warmup runs. The remaining row condition is deployed
  identity-provider integration. A real Vendor Catalog v3/rev4 resource fixture
  closed the prior lifecycle condition: Astra inspected one file, published
  folder, declared skill, populated edit controls, and canceled the explicit
  dependency-block deletion confirmation. The independent technical reviewer separately
  retains shared contrast, zoom, theme, and whole-product screen-reader evidence
  as production conditions.
- A measured live desktop contrast scan found no failing visible text after
  CodeMirror string highlighting was changed semantically to `#b42318`; the only
  sub-4.5 result is the aria-hidden decorative view glyph. Generated CodeMirror
  class names are not targeted.
- A 640×720 viewport review, equivalent to a 1280px desktop at 200% layout
  scaling, retained every SQL command, proper tab/panel identity, and exact
  640px document/body scroll width with no horizontal overflow. The temporary
  override was reset after inspection.
- Studio currently supports and declares a light color scheme. Dark mode is not
  presented as supported or production-verified; this explicit boundary avoids
  an untested adaptive-theme claim while Forge token migration remains optional
  future scope.

### Governed namespace catalog and editor — locally verified

- Namespaces are first-class owner-scoped records backed by canonical schema,
  Datly reader/writer DQL, generated static components, Studio SDK operations,
  optimistic revisions, and reference-safe deletion.
- The catalog has a distinct navigation destination, lifecycle state, edit and
  destructive actions. Component creation selects an active governed namespace
  instead of accepting arbitrary free text.
- Desktop inspection covered populated catalog and create dialog. Empty, error,
  conflict, delete-blocked, and narrow states still require final scenario
  evidence before external approval.

### Component command hierarchy and narrow layout — locally verified

- Component actions are grouped by product responsibility: Author, Govern, and
  Release. The component identity/revision header no longer competes with an
  eleven-button undifferentiated toolbar.
- Publish is disabled until the exact revision is valid and the principal has
  publish capability. Preview, editing, governance, and advanced DQL controls
  follow their dedicated capabilities.
- At a 720px viewport, document scroll width now equals viewport width (720px;
  previously 1,305px). Navigation uses an overlay drawer and the graph remains
  the first component workspace.

### Governed namespaces — locally verified, Fable pending, 2026-09-20

- The namespace catalog now delegates query, lifecycle status, limit, and
  offset to the owner-scoped SDK instead of loading an unbounded client list.
- Search, clear, active/archived filtering, 25-row pagination, filter-aware
  empty states, create/edit/archive, and reference-safe deletion are exposed as
  one catalog workflow. Live desktop review narrowed the fixture from two
  governed namespaces to `inventory.forecasting` through server-side search.
- Update/delete use the namespace etag. Stale changes remain unapplied and offer
  explicit catalog or dialog reload before review and retry.
- Rendered dialog coverage verifies that stale edits preserve the user's values,
  explicit Reload namespace replaces them with the authoritative revision, and
  ordinary create rejection keeps entered namespace/title values available for
  correction. Backend integration coverage continues to enforce reference-safe
  deletion while components belong to the namespace.

### Inputs and constants — locally verified

- One searchable/filterable workspace presents request parameters and trusted
  constants without creating a second browser-owned component model.
- The catalog is paginated for 200+ declarations and filters by source kind.
  Constants expose typed source identity and authored default; instance values
  retain Datly's server/deployment authority and request values remain bound.
- Add/update/remove default behavior is implemented by the generic Datly 1.0
  Reader Builder field operation and covered by focused Reader Builder tests.
- Request inputs and trusted constants are distinct tabs rather than one mixed
  list. The 239-input Forecasting fixture stays in a bounded searchable catalog,
  so its editor remains reachable without scrolling through every declaration.
- The structured binding contract now covers arbitrary validated Datly type
  expressions, codecs and ordered arguments, named resource URI, query selector,
  required state, response emission, and constant authored defaults. Datly
  Reader Builder owns source-preserving `.WithCodec`, `.WithURI`, and `.Output`
  mutations; the browser never writes DQL fragments.
- Absolute `.WithURI("/…")` values now appear as route activation while named
  values remain resources, matching Datly's one canonical URI option. Structured
  `.WithDescription` and `.WithExample` metadata flows through inspection and
  generated HTTP/MCP documentation without becoming runtime defaults.
- Constants clearly state instance-file precedence and prevent invocation-owned
  values. Zero, false, and empty-string override semantics match Datly 1.0.
- Rendered scale coverage exercises 120 request inputs across 50-row pages,
  source-name search, the separate constants tab, and persistence of an authored
  zero through the structured `updateField` operation.

### Predicate catalog — complete compiled Boolean controls locally verified, Fable pending, 2026-09-20

- The 234-occurrence Forecasting fixture renders as a bounded, paginated
  catalog with predicate-family, kind, target-view, and page-size filters.
- Each row now exposes every compiled expansion view for its Datly filter group,
  plus its canonical within-group operator. Search includes target views and a
  clear-search control; editing preserves multi-view expansion scope.
- A synchronized Groups view aggregates shared expansion scope, distinguishes
  dormant groups, keeps large groups collapsed, and edits within-group AND/OR
  through Datly Reader Builder's source-preserving `updatePredicateGroup`
  operation. It never derives Boolean order from numeric group IDs.
- Editing preserves `applyWhenAbsent`; authorization group controls remain
  locked. The Reader Builder rejects partial shared scope and atomically patches
  every explicit expansion site.
- The dialog uses the viewport-constrained shared dialog body, so the catalog
  and authoring controls remain reachable without growing the document.
- Datly inspection now returns each outer Builder chain by view and source-order
  occurrence, with CombineAnd/CombineOr terms, persistent And/Or connectors,
  exact group occurrence lists, WHERE/AND attachment, source span, and explicit
  editability. Studio exposes those structured controls above FilterGroup rows.
- Source-preserving updates retain every group occurrence and within-group
  operator, patch only the selected chain, recompile, and verify the result.
  Nested/raw expressions are shown read-only rather than guessed; authorization
  group 99 cannot be OR-connected or bypassed.
- Rendered scale coverage exercises 240 predicate occurrences, deterministic
  25-row pagination and search, then switches to Groups and persists a distinct
  Combine OR plus Build AND outer-composition operation. Within-group operator
  and absent-input controls now have explicit accessible label bindings.

### Column contracts — locally verified

- The view-detail field catalog is searchable and paginated for 100+ columns.
  Selection is keyboard-accessible through real buttons rather than pointer-only
  table rows.
- A cohesive column contract controls explicit cast, output visibility,
  component-cased public name, cube role, and registered metadata tags.
- Studio sends one typed `setColumnContract` operation. Datly Reader Builder
  performs source-preserving CAST/tag edits, returns normalized contract
  metadata, preserves unrelated tags, and recompiles the complete candidate.

### Validation diagnostics — locally verified, Fable pending, 2026-09-20

- Validation remains bound to the displayed version and source revision and
  materializes the publishable Datly runtime contract without running the
  business query or changing the active generation.
- Wrapped Datly compile errors now preserve every diagnostic's severity, code,
  hint, line, and column through the Studio SDK. The UI presents severity counts,
  exact locations, recovery hints, and honest checking/checked/prior-pass stage
  states instead of flattening everything into generic red callouts.
- Advanced source navigation appears only with `canUseDql`; ordinary SQL authors
  retain cohesive UI recovery without being pushed into structural DQL.
- Live Vendor Catalog validation rechecked v2 revision 1 successfully and
  updated its validated timestamp while leaving generation 12 unchanged.
- Rendered failure coverage verifies diagnostic counts, codes, locations,
  actionable hints, permission-gated source navigation, unchanged runtime
  messaging, and retry after the validation service itself fails.

### Permissions — locally verified, Fable pending, 2026-09-20

- The owner-only editor distinguishes implicit owner authority from delegated
  verified JWT subjects and offers Viewer, Operator, SQL author, Advanced author,
  and Publisher starting points with a visible effective-capability summary.
- Capabilities are grouped by data access, authoring, and release. UI and server
  both enforce that run/edit/publish/DQL require view and that structural DQL
  additionally requires edit; inconsistent direct SDK grants are rejected.
- Schema v8 adds ACL row etags to the dedicated reader/writer contracts.
  Conditional update/delete and a competing-writer test prevent lost updates;
  stale responses include expected/current revisions and the dialog reloads the
  current policy before retry. Live desktop review verified the empty owner-only
  state and least-privilege grant workflow for Vendor Catalog.

### Staged publication and recovery — locally verified

- The release window distinguishes serving version/generation from desired
  version/generation and explains the compensation boundary.
- Publishing and unpublishing retain the prior active database state during
  runtime reload. Runtime restart reconstructs persisted active state rather
  than an orphaned pending candidate.
- Focused failure injection proves that a successful runtime swap followed by
  failed activation persistence reloads the previous generation, restores the
  prior publication row, and marks the candidate generation failed.
- Schema v7 adds an owner-scoped, append-only publication event ledger. Success
  is recorded inside the activation transaction; failed transitions retain a
  bounded, secret-redacted failure code/message without changing compensation.
- The release window shows recent publication, rollback, and unpublish outcomes,
  isolates partial context-loading failures, and requires an audit note plus a
  second confirmation before rollback or unpublish. Ordinary publication remains
  a single exact-revision staged action.

### Reviewer access status

- Fable 5.1 remained unavailable: the Claude surface was signed out and no
  Claude CLI was installed. The product owner explicitly authorized Codex Astra
  low as the substitute reviewer; its scoped verdict and remaining scenario
  conditions are recorded above.
- Rechecked 2026-09-25: `claude` 2.1.281 is installed, but `claude auth status`
  reports `loggedIn: false` and `authMethod: none`. Fable 5.1 review is still
  pending; the installed executable does not supply reviewer evidence.
- Rechecked 2026-09-25: no deployed Studio tab or deployed sign-in URL was
  available in the browser inventory or repository configuration. Local Studio
  was reachable on the preseeded SQLite fixture, but an attempted actual Chrome
  zoom check was interrupted when the active browser session changed. The
  earlier 640px viewport simulation remains local layout evidence only;
  deployed login, restoration, expiry/re-login, denied-user operation,
  manual screen-reader announcements, and actual 200% browser zoom are still
  pending.

## Completed window reviews

### Reader permissions — passed, 2026-09-18

- The toolbar opens a focused permission window without interrupting the
  builder graph or source inspection.
- Ownership remains implicit and explicitly non-editable; the empty state says
  so rather than suggesting the owner must add themselves.
- The grant form requires an exact JWT-subject value and starts with the
  useful least-privilege pair: catalog view and reader execution. Authoring,
  publication, and raw DQL access require intentional opt-in.
- Report/package identity is derived strictly from verified JWT `sub`. The BFF
  and Datly authorization predicates reject tokens that contain only username,
  email, or other display claims.
- Rendered least-privilege coverage verifies that the Advanced author preset
  establishes view/run/edit/DQL dependencies, stale grant writes remain
  unapplied while preserving the entered subject, and Reload permissions
  replaces the catalog with the authoritative ETag revision.

### Stale reader revision — passed, 2026-09-18

- Browser SDK errors retain the server's typed `conflict` code, HTTP status,
  field, and violations instead of collapsing them into message text.
- A real stale parameter submission opened a focused recovery window above the
  preserved editor. “Review my draft” returned to the unchanged `Limit` form so
  values could be copied; “Reload latest” disclosed that unsaved form values
  would be discarded, closed the stale editor, and loaded revision 5.
- The window states that published runtime remains unchanged. Its warning
  hierarchy, two decisions, loading state, and narrow layout were live-reviewed.
  The temporary revision-only database change used to trigger the authentic 409
  was restored to revision 4 after verification.
- Automated coverage proves preservation of typed conflict metadata and renders
  the recovery dialog in JSDOM. It verifies the warning, discard/runtime-impact
  copy, Review-my-draft decision, and disabled/loading state while reloading.
  End-to-end rendered coverage now also submits a real stale SQL save through
  the parent Reader Builder, reloads the authoritative inspection, and restores
  focus to the component heading for a predictable screen-reader/keyboard
  continuation point.

### Authenticated shell — passed, 2026-09-18

- Authenticated Forge requests use `credentials: include`; browser code has no
  token provider, bearer header, client secret, or development identity.
- The backend exchanges a verified JWT for an opaque `studio_session` cookie
  (`HttpOnly`, `SameSite=Lax`, secure in authenticated mode), supports
  `auth/me` recovery and logout, and exposes the same exchange for OOB backend
  session pre-seeding.
- The BFF proxies dynamic HTTP and dedicated MCP traffic, strips the browser
  cookie, and injects the server-held JWT as an Authorization header. Browser
  preview therefore reuses the current authenticated session without exposing
  token material to Forge.
- The dynamic application manager is now a separate process: Datly HTTP binds
  to 8082 and streamable MCP binds to dedicated 8091. Both listeners reject
  missing/invalid bearer credentials before component dispatch; the BFF is the
  browser-facing JWT expansion boundary.
- Startup now restores `/auth/me` before mounting Studio, displays intentional
  loading and unavailable states, and presents a narrow-reviewed sign-in /
  session-expired window with deployment-owned login and retry actions. A 401
  from any SDK operation returns the shell to that boundary. The browser stores
  no token. Deployment still supplies the configured OAuth login callback;
  that external endpoint is not fabricated by the UI.
- Rendered authentication-boundary coverage verifies intentional session
  loading, a real deployment-login link, absence of browser token controls,
  announced identity-provider failure with retry, successful recovery, and an
  SDK 401 returning the shell to the expired-session sign-in state. The review
  exposed and fixed a non-link Blueprint rendering and missing alert semantics.

### Connector catalog — passed, 2026-09-18

- Re-review on 2026-09-20 replaced the historical MySQL fixture assumption
  with the requested preseeded SQLite environment. Both active connectors show
  `sqlite`, a passed probe, and guarded Test/Edit/Disable/Delete actions.
- Server-side free-text search covers connector name, description, driver, and
  owner; lifecycle filtering and 25-row pagination share the same catalog
  query. A live unmatched search rendered a specific “No matching connectors”
  state with recovery guidance. A SQL transport regression test covers
  description/status search, driver pagination, and owner search. Rendered
  React tests cover search submission, lifecycle filtering, recovery copy, and
  clearing back to the populated catalog.

- Desktop and narrow live review: responsive compact table retains connector
  name, lifecycle/probe state, and accessible Test/Activate/Disable actions;
  nonessential driver/owner columns collapse rather than clipping actions.
- Empty, error, loading, and populated states were reviewed. The populated
  state uses the real Endly-hydrated MySQL reporting schema; the connector
  probe passed against that source.
- The New connector dialog was reviewed at narrow viewport: basic fields and
  footer actions remain visible, advanced settings are disclosed deliberately,
  and ownership is server-derived rather than editable in the browser.
- Browser actions use named JSX `StudioAPI` methods only. Backend tests cover
  owner derivation, blocked activation before a successful probe, and SQLite
  driver selection; Node SDK contract tests and the production build passed.
- Stale lifecycle action review: a real ETag conflict on `ci_ads_mysql`
  rendered one focused “Connector changed elsewhere” warning with Refresh
  catalog, did not disable the connector, and did not duplicate the low-level
  error. The temporary ETag-only fixture change was restored after review.
- Rendered catalog coverage now reproduces an optimistic lifecycle conflict,
  verifies that no change is reported as applied, and exercises the explicit
  Refresh catalog recovery action.
- Connector edit — passed, 2026-09-18: the edit form presents immutable name,
  current optimistic revision, lifecycle status, and an explicit warning that
  provider/DSN/secret/options changes require a fresh probe and activation.
  Stored DSN material is never returned to the browser: the blank “New DSN
  template” field says whether a server-held value exists without revealing it.
  A live description-only edit preserved `active`/`passed`, then restored the
  original description. At narrow width, the dialog body scrolls while Save and
  Cancel remain reachable; advanced settings are collapsed unless configured.
- Connector rotation — passed, 2026-09-18: a disposable connector was created,
  probed, and activated exclusively through SDK operations. Replacing its DSN in
  Forge moved it from `active / passed` to `draft`, cleared the probe badge and
  activation action, and retained only Test/Edit. The disposable connector was
  then deleted through the SDK and verified absent from the active catalog.
- Connector deletion — passed, 2026-09-18: active connector delete actions are
  disabled with “Disable connector before deleting.” Draft/disabled connectors
  open an explicit destructive confirmation naming the connector and explaining
  that report references block deletion. The confirmation was live-reviewed and
  cancelled; the disposable review connector was deleted separately through the
  optimistic SDK operation and verified absent.

### Report catalog — passed, 2026-09-18

- Component-create re-review on 2026-09-20 verified the current product
  language and governance flow: title and slug are grouped first, namespace is
  selected from the governed catalog, and only tested active SQLite connectors
  can be chosen. Identity and ownership remain server-derived. The dialog has a
  single primary action and keeps explanatory copy adjacent to the affected
  field instead of exposing Datly directives.

- Desktop and narrow live review: the catalog shows real Vendor reader drafts,
  compactly retaining title and lifecycle state while nonessential columns
  collapse at narrow widths.
- The New reader dialog exposes only title, slug, description, and an active
  authorized connector. It does not expose owner, report ID, component scope,
  or component name; those identities are derived by the SDK host.
- The active-connector selector, required-field affordances, cancel/create
  actions, and server-side validation feedback have a visible and keyboard
  accessible form surface.
- Backend tests cover generated report identity/owner and rejection of inactive
  default connectors. Node SDK contract tests and the production build passed.
  Rendered tests verify that an empty governed namespace catalog blocks create,
  and that a server rejection preserves title, derived slug, description, and
  the still-open dialog for correction.

### Reader Builder — implemented windows passed

- Live inspection displays the real versioned Vendor DQL through the Datly
  reader-builder API: root view, related `PRODUCT` view, parameters, predicates,
  source revision, and diagnostics.
- Structured Add Parameter and Add Predicate commands were exercised against
  the real Vendor version. Both updated source revision and compiled DQL without
  diagnostics. Add Subview has a reviewed command form backed by Datly's
  `addView` operation but was not submitted
  because the current reporting fixture has no additional authorized table.
- Full Preview now executes the dynamic Datly component built from the stored
  version DQL and SQLX-refined runtime input/output types. Live output included
  real nested Vendor/Product data from the authorized MySQL connector.
- Isolated view testing is now SDK-backed and live-reviewed: Datly's inspected
  canonical view names (`vendor` and `products`) create transient runtime
  components from the stored DQL, rather than running a browser SQL shortcut.
  Each returned real MySQL rows and a duration in the Reader Builder. The
  buttons deliberately use the inspection's `structure.views` identities,
  avoiding a mismatch with presentation graph labels such as `catalog`.
- Graph selection and contextual toolbar — superseded and rejected by desktop-web
  screenshot review, 2026-09-20: the top-down
  Datly view graph has keyboard-selectable view and join blocks with persistent
  pressed-state styling. Selecting a view activates only Add child, Modify SQL,
  Test, and Remove; root removal is visibly disabled. Selecting a join exposes
  the compiled key condition and relation kind, then activates Modify SQL,
  Test, and Remove for the joined child. The join-to-child action was
  prior behavior opened the CodeMirror SQL editor for `products`, not the root
  view. That view-tab behavior is now rejected: selecting any graph view must
  instead activate the inline right-side settings inspector. A clear-selection
  action returns the graph to its neutral guidance state.
- Typed view creation — passed, 2026-09-18: Add view explicitly distinguishes
  Regular join from Derived view. Regular joins expose parent, child SQL, and
  complete relation keys. Derived views lock attachment to the root,
  remove join/key controls, and explain their independent output semantics.
  Graph labels and the selected-node toolbar read `Relation.Kind` and relation
  links from Datly's compiled metadata; Studio does not classify SQL. Join-kind
  choice is intentionally absent because outer joins are decomposed graph
  syntax and do not control runtime child attachment.
  Both variants use the same typed Datly 1.0 `addView` operation through the
  Studio SDK.
- Real `ci_ads` composition — passed, 2026-09-18: the local MySQL 8.4 Docker
  database was loaded from `viant-internal/platform/skeema/ci_ads` (460
  supported base tables plus supported routines). A probed, active
  `ci_ads_mysql` connector backed a new CI Ads Vendor Catalog. Studio created
  the root `CI_VENDOR` reader and added `CI_VENDOR_DOMAIN` with Datly's typed
  `addView` command. The compiled graph rendered `ROOT VIEW` → `JOINED SUBVIEW`
  with `ID = VENDOR_ID`; full preview returned three seeded vendors with 2/1/2
  nested domains in 2.6ms and no diagnostics.
- Nested result preview — passed, 2026-09-18: Datly result envelopes render as
  a responsive root table with scalar columns and nested-view count badges.
  Rows with subviews expose keyboard-accessible expand controls; expanding the
  first CI Ads vendor rendered its two `Domains` rows as an indented child
  table. Each level owns horizontal scrolling at constrained width. A Table /
  JSON switch changes only presentation and displays the same response as
  formatted JSON without rerunning the component.
- Schema Browser SQL and composition actions — passed, 2026-09-18: default
  schema now means the connector's current database, so `ci_ads_mysql` opens
  its tables immediately. Test SQL compiles the editor contents into a bounded
  transient Datly reader through `connectors.test_sql`; live `CI_VENDOR`
  execution returned three typed rows in 1.9ms. Add as root view invokes
  Datly's typed `createReader` operation, with server-derived package and
  connector identity; the generated reader compiled and previewed in 1.7ms.
  Add as subview lists compatible readers, inspects the selected graph, derives
  `domains.VENDOR_ID=ci_vendor.ID` from SQLX FK metadata, applies `addView`, and
  opens the updated graph. The full nested preview returned 2/1/2 child rows in
  3.1ms. No Schema Browser button writes SQL, DQL, or storage directly.
- Schema Browser re-review on 2026-09-20 used the requested preseeded SQLite
  fixture: connector and schema selectors, searchable table tree, starter SQL,
  Test SQL, Add as root, and Add as subview remained visible in one workspace;
  selecting `VENDOR` exposed real columns and constraints. Rendered tests cover
  connector discovery failure, table discovery, starter SQL, bounded SQL test,
  root/subview action activation, tab/result/dialog reset, and blocking an
  incomplete relation before any Datly command is sent.
- Schema tab close — passed, 2026-09-18: an active table tab now has a visible
  × action with an exact accessible label (for example “Close
  CI_AD_ORDER_FLIGHT table”). Closing clears its editor, transient SQL result,
  columns, and dependent add-view dialogs without changing the connector or
  any reader definition; the schema explorer remains available for the next
  table.
- Related-view removal has a live-reviewed safe confirmation surface. It shows
  the selected source/projection impact, blocks a view with compiled dependent
  subviews, does not offer the root as a target, and requires typing the exact
  Datly view identifier before its destructive SDK command can be enabled.
- Rendered relation coverage verifies complete compiled key submission through
  one typed update operation, preserves rejected edits for correction, and
  blocks removal while the selected child still owns dependent subviews.
- Cube readiness has an SDK-backed dialog. On the live joined Vendor graph,
  Datly rejected activation without changing source and returned the specific
  grouped-root diagnostic; the UI presents this as a recoverable authoring
  constraint rather than an unsupported or silent toggle.
- Successful cube/compose evidence also exists: the real MySQL Product Status
  reader enabled `cube`, then persisted bounded composition (8 cubes, 100 rows,
  30 seconds) and selected MCP exposure through the dialog. Its full dynamic
  Datly preview and isolated `summary` view test both returned the real grouped
  PRODUCT result.
- Cache configuration is also live-reviewed: the same reader has a native
  Datly AFS cache directive, full dynamic preview succeeded, and Datly created
  its local cache entry. The Forge dialog clearly exposes name/TTL/location
  and states that saving a cache never starts background work.
- Cache & Warmup dialog — passed, 2026-09-18: the dialog exposes only
  cache identity, TTL, and approved AFS location; no credentials reach the
  browser. A live server-owned Datly warmup completed one native cache entry
  in 4.5ms and rendered that evidence in an accessible status region.
- Relation assembly test — passed, 2026-09-18: selecting a compiled relation
  exposes separate Test relation and Test child actions. Test relation executes
  the complete Datly graph and labels the evidence with the selected edge. The
  live `ci_vendor → ci_vendor_domain` run completed in 2.4ms and returned three
  parents with 2/1/2 nested children; Test child retains isolated SQL evidence.
- Cube composition playground — passed, 2026-09-18: the Cube dialog derives a
  typed default frame from compiled dimension/measure metadata and provides
  CodeMirror JSON and SQL editors. Execution compiles the persisted runtime
  contract, derives Datly's real cube and compose components, expands
  `$CubeSQL1…N`, and enforces authored cube-count, row, and timeout budgets. The
  live Product Status composition returned its column contract and grouped row
  in 2.2ms; Table/JSON result modes remained usable in the scrolled narrow
  dialog. Browser code never substitutes frame SQL or opens a database.
- Parameter lifecycle window — passed, 2026-09-18: authored inputs are listed
  separately from output holders and expose Add, Edit, New, and confirmed
  Remove states. A live `Limit` update changed `query/limit` to
  `query/maxLimit` while preserving Optional and `less_or_equal` predicate
  options; diagnostics remained empty. A temporary boolean input was added and
  removed after exact-name confirmation, with the graph recompiling at each
  revision. Reference-aware rename was subsequently live-reviewed by changing
  predicate-bearing `Limit` to `MaxLimit` and back: the declaration changed,
  `less_or_equal` remained attached, diagnostics stayed empty, and the query
  binding remained `maxLimit`. The UI explains executable-token versus quoted-
  literal behavior and performs rename with the same atomic update command.
- Relation update/re-parent window — passed, 2026-09-18: selected ordinary
  relation edges expose a dedicated Modify relation action separate from child
  SQL. The dialog preserves child identity and SQL, edits only parent and full
  key expression, warns when nesting changes, and delegates AST-span mutation
  and parent verification to Datly. Live `domain_aliases` re-parenting from
  `ci_vendor` to `ci_vendor_domain` compiled without diagnostics; a three-level
  preview returned nested children in 4.8ms. Sibling branches now render
  horizontally on desktop and carry explicit parent→child labels when stacked
  at narrow widths. The shared flex workspace no longer causes page overflow.
- Field role window — passed, 2026-09-18: selecting a view exposes its compiled
  fields, physical source names, types, and Dimension/Measure role. Datly's
  typed `setColumnRole` operation uses Tagly and preserves unrelated tags. Live
  Product Status toggling emitted explicit `groupable:"false"`, retained the
  field in inspection, then restored `groupable:"true"`; the Cube playground
  immediately regenerated `status` under dimensions and `productCount` under
  measures. The window also presents an intentional empty state when wildcard
  SQL lacks schema-refined field metadata.
- Large predicate catalog — passed, 2026-09-18 against the imported Steward
  Forecasting reader: 234 predicate occurrences render as a searchable catalog,
  not one giant form. Summary counters showed 135 handler predicates, 113
  Include predicates, and 114 Exclude predicates. SQL/handler and
  Include/Exclude/Core filters compose with search; page size is selectable at
  25/50/100. Pagination remains above the scrolling result region. Live review
  advanced from rows 1–25 to 26–50 and page 2 of 10, while the Add form retained
  the canonical root target `forecasting` followed by its seven child views.
  Keyboard labels distinguish the previous/next controls.
- Forecasting import and connector integrity — passed for static authoring,
  deliberately blocked for execution until a real BigQuery connector exists.
  The legacy source was normalized to required `#package`, `#define`, and
  `#setting` directives. Its six lookup views now reference the exact active
  Studio connector `ci_ads_mysql`; the root and `ageGroup` still require the
  source-authored `bq_metrics`. Builder inspection is complete with no static
  diagnostics and exposes exactly the active authorized Studio connector
  catalog (`ci_ads_mysql`, `reporting_mysql`). Preview fails early and clearly
  with `Datly reader requires active Studio connector "bq_metrics"`; Studio did
  not create a fake connector or substitute the report default. BigQuery and
  PostgreSQL database drivers are linked for use after real connectors and
  credentials are configured.
- HTTP/MCP exposure window — passed, 2026-09-18: the dialog presents the one
  compiled HTTP route, draft compile state, shared typed/auth/connector contract,
  and the dedicated dynamic MCP listener boundary. Base-reader exposure uses
  Datly's typed `setSetting(mcp)` operation and accepts ordinary punctuation in
  descriptions through safe DQL argument serialization. Tool names must use the
  server-derived JWT-sub owner prefix and are checked across other readers before
  a source revision is persisted. Live Forecasting authoring saved and reloaded
  `awitas.forecasting.read`; an invalid spaced name rendered an accessible inline
  error without creating a revision. Cube and compose tool states are shown from
  compiled report metadata rather than a Studio-side registry.
- Owner package and business namespace — passed, 2026-09-18: dynamic package
  identity is server-owned and derives its owner segment from verified JWT
  `sub` (`Adrian.Witas_test@…` becomes `adrianwitastest`). Existing local readers
  were migrated through Datly's typed, source-preserving `setPackage` operation;
  Forecasting now compiles under
  `github.com/viant/datly-studio/dynamic/awitas/<report-id>/reader`.
  Business namespace is independent metadata used only for catalog grouping and
  filtering. The New reader window exposes a validated dot-separated namespace,
  the catalog has an All namespaces filter, and the builder header displays the
  selected namespace. Schema version 2 upgrades existing rows to `general` from
  the single canonical schema definition.
- Runtime validation window — passed, 2026-09-18: validation materializes the
  exact Datly runtime contract, opens every required active named connector,
  refines columns, builds typed input/output/resources, checks routes and MCP
  identity, and initializes the executable reader without running the business
  query or changing the active generation. The same window was live-reviewed in
  both states. Forecasting became `invalid` with the actionable missing
  `bq_metrics` diagnostic; Vendor Catalog became `valid` at exact revision 9.
  Status updates propagate to the builder header, diagnostics remain in an
  accessible region, and the current runtime is explicitly described as
  unchanged on failure.
- Publication window — passed, 2026-09-18: the window shows the exact validated
  draft revision, explains that the browser never selects a connector, port, or
  generation, and permits an optional release note. A live Vendor Catalog
  publication staged generation 4, invoked the dedicated dynamic runtime reload
  endpoint, and displayed “Runtime generation active” only after the reload
  succeeded. Database evidence confirms generation 4 active and generations
  1–3 retired. A publication cannot begin until runtime-contract validation is
  valid for the current source revision.
- Resources & Skills window — passed, 2026-09-18: a single focused authoring
  flow persists text files, explicit MCP folders, and declared skill roots only
  through the Studio SDK. Vendor Catalog saved the owner-scoped resource
  `awitas.docs:guide/SKILL.md`, published `guide` at
  `skill://awitas-guide/`, and declared `./SKILL.md` as a skill root. Each save
  incremented the reader source revision and reset validation. Runtime validation
  compiled Datly's folder/skill publication plan successfully; live publication
  activated generation 5. Namespaces, paths, URI prefixes, SKILL.md existence,
  owner scope, digests, and skill compilation are server-side invariants.
- Resource lifecycle follow-up — locally verified, 2026-09-20: stored files,
  published folders, and skill roots are selectable for editing and expose
  confirmed deletion. Folder deletion remains blocked while a skill depends on
  it. All six mutation paths compare the inspected source revision and perform
  the resource change, revision increment, and validation reset in one database
  transaction. A synchronized two-writer test proves exactly one commit and one
  conflict; the dialog offers explicit reload/review recovery.
- Rendered stale-revision coverage verifies that a rejected file write preserves
  unsaved content, reports that no resource change was applied, and reloads the
  authoritative revision plus stored resource catalog on demand.
- Unpublish lifecycle — passed, 2026-09-18: an active publication shows its
  generation and a deliberate Unpublish action in the lifecycle window. Live
  Vendor Catalog removal created a new runtime generation and reloaded the host
  without the reader, its MCP tools, resources, or skill. The UI reports
  “Reader removed from active generation,” not a fake active generation. The
  reader was then republished successfully in generation 7, proving the full
  withdrawal-and-restore path.
- Runtime status — passed, 2026-09-18: the lifecycle window reads the protected
  Studio SDK runtime status and presents the real state before any action. Live
  review showed `active`, generation 9, and 3 readers. Runtime state is a compact
  neutral status strip separate from the selected reader's publication status,
  making an unhealthy global host visible without suggesting that a report-local
  row is enough evidence of deployment.
- Version history and rollback — passed, 2026-09-18: the lifecycle window lists
  historical versions with source revisions and validation state, enables
  rollback only for a validated selection, and explains that rollback stages the
  same runtime activation rather than rewriting source. Live Vendor Catalog v2
  was rolled back to validated v1, activating generation 10. Publication marks
  the formerly active version `superseded` and the selected target `published`.
- Release activity — locally verified, 2026-09-20: the same lifecycle window
  loads owner-scoped append-only events through `publications.events.list`,
  renders an honest empty state for readers with no post-migration events, and
  prevents unpublish confirmation until a release-note reason is present.
- Rendered publication-failure coverage verifies that a rejected runtime reload
  reports “Publication did not activate,” preserves the serving version and
  generation evidence, and never emits a false active-generation success state.
- The Reader Builder's implemented windows now pass their individual UX gates.

- Graph-first component workbench — reworked and locally verified after
  desktop-web screenshot review, 2026-09-20: the first tab is
  `Component <title>`, with the graph central and the selected-view settings in
  a stable right-side inspector. Technical labels
  such as `ROOT VIEW`, `reader`, and `JOINED SUBVIEW` were removed from graph
  cards; the live Vendor graph reads `Vendor → Products`. The prior
  click-to-open/refocus CodeMirror view tab was removed. Selecting a graph view
  now activates the inline
  right-side settings inspector; only an explicit SQL action may open the
  dedicated editable-resource SQL tab. The ellipsis action menu opens
  independently of the card click. Raw DQL is hidden by default and is exposed
  only through the Advanced cog’s explicit `Show raw DQL` setting; structural
  DQL otherwise presents SQL-resource chips.
- Nested table rendering — passed, 2026-09-18: structured values are excluded
  from scalar columns defensively. Live Vendor preview no longer renders
  `[object Object]` or a “Nested views” column; expanding a vendor embeds the
  typed Products subtable directly below its parent row.
### View inspector and column catalog — implemented and locally verified, Fable pending, 2026-09-20

- Screenshot feedback rejects the prior separate explanatory “View behavior”
  and “Column authoring” cards. They must be removed rather than restyled.
- Selecting `Vendor` or another graph view must activate its inline right-side
  settings inspector and must not open a view tab. The full lower inspector
  area is reserved for the searchable, paginated metadata-backed column catalog
  and its contextual contract actions.
- SQL may remain a dedicated editable-resource tab, but it opens only through
  an explicit SQL action. It is not the selection destination.
- The SQL tab now keeps only the selected view/path, saved state, Test, Save,
  and relation keys when relevant. The former five-cell connector/source/graph/
  contract/revision strip was removed because it duplicated the component
  header, graph inspector, and column catalog.
- Unsaved SQL is protected when switching component/SQL/DQL tabs, closing the
  SQL tab, returning to the component catalog, or unloading the page. Live
  review verified that navigation presents Keep editing versus Discard changes
  and that discarding leaves the saved Datly view untouched.
- The desktop-web rework is implemented. Live review at 1600×1000 verified that
  selecting `Vendor` leaves the component tab active, selects the graph card,
  fills the right-side Source/Connector/Contract inspector, and renders seven
  seeded SQLite columns in the full lower catalog. Selecting `Products` updates
  the same surfaces in place and merges its compiled contract with connector
  metadata instead of reducing the catalog to one internal key.
- The selected Vendor view test completed through the Studio SDK in 605µs and
  returned all three deterministic fixture rows. UI tests, production build,
  the full Studio Go suite, and the Impeccable detector passed. Fable 5.1
  external review remains pending; no Fable approval has been recorded.
- Rendered Reader Builder coverage now proves the graph-first interaction: a
  view click activates the inline inspector and merged columns without opening
  a tab; Open SQL is the explicit transition to the per-view SQL resource; and
  dirty SQL blocks component-tab navigation with Keep editing / Discard changes.
- Rendered column-contract coverage verifies the single normalized
  `setColumnContract` payload for cast, visibility, cube role, and public name,
  plus local rejection of invalid advanced tag identifiers before Reader
  Builder is invoked.
- Accessibility hardening gives workspace tabs real tab/tab-panel semantics and
  arrow/Home/End navigation, labels the actual CodeMirror surface, and separates
  graph selection from collapse/action buttons. Selecting a column now opens
  its contract directly; the large picker is optional and Save/Cancel stay in
  the dialog footer.
- A rendered least-privilege projection verifies that read-only users receive
  disabled Edit/Input/Predicate/Compose/Publish and selected-view mutation
  controls, while Advanced DQL is absent rather than merely visually hidden.
- The SQL/preview workspace now exposes a vertically draggable editor plus
  Minimize, Balanced, and Maximize presets. Sizing changes are presentation-only:
  the existing preview remains visible and is not rerun, allowing focused result
  inspection or focused SQL authoring without losing evidence.

### Edit component — implemented and locally verified, Astra re-review pending, 2026-09-20

- Catalog title/description, active connector, Cube, Cube composition,
  composition budgets/MCP exposure, base MCP enablement, and MCP tool name now
  live in one Edit component dialog beside the component header.
- Connector changes use Datly Reader Builder's source-preserving connector
  setting and synchronize catalog identity in the same database transaction.
  The Author toolbar now contains only view/input execution concerns.
- MCP activation is explicit for three distinct Datly components: the base
  reader tool, the derived Cube tool, and the derived CubeCompose tool. Datly's
  `report.ProjectedComponents` owns the generated names and `/cube` plus
  `/cube/compose` routes; Studio's published MCP catalog renders each generated
  component separately with enabled/disabled state.
- Saving submits one Datly Reader Builder `batch` operation. Datly applies and
  compiles every source-preserving setting change atomically, so the browser
  cannot leave a half-configured component after a stale-revision conflict.
- The former Cube & composition dialog is now a Composition lab that exercises
  the saved contract; it no longer duplicates settings.
- Rendered Composition lab coverage builds ordered typed frames, selects explicit
  filter inheritance, sends the exact cubes/SQL request, renders returned data,
  and rejects non-object filter JSON before invoking Datly. Frame controls now
  have stable accessible labels.
- Desktop browser review verified progressive disclosure for composition
  budgets and MCP identity. Datly Reader Builder tests and the production build
  pass; Codex Astra approved the targeted component/composition changes.
- Rendered tests exercise the complete cohesive settings window and assert one
  atomic Reader Builder batch for cube, cube composition budgets, composition
  MCP exposure, reader MCP enablement/name, description, and description
  resource. Owner-prefix validation blocks invalid MCP names before mutation.
- The rendered accessibility pass found that the three composition-budget
  labels were visually present but not programmatically associated with their
  inputs. Stable IDs and `labelFor` bindings now make Maximum cubes, Maximum
  result rows, and Timeout directly addressable by label.

### Runtime workspace — implemented projection, Fable pending, 2026-09-20

- The new Runtime window shows the recorded active generation, every deployed
  component/version visible to the operator, connector and runtime revision,
  and active MCP exposure inventory without client-side DQL parsing.
- Runtime reader rows are authorization-scoped by ownership or `can_publish`;
  the public SDK never exposes connector credentials or raw build manifests.
- The generation strip separates the control-plane activation record from an
  authenticated dynamic-host readiness probe. It renders ready, unavailable,
  and unknown states without exposing the administration token.
- Live desktop review verified generation 12, three deployed components, and
  dynamic host readiness revision 1. Both seeded HTTP routes executed their
  nested results. External Fable 5.1 review remains pending.

### Overview — implemented and locally verified, Fable pending, 2026-09-20

- Overview is now the default desktop landing window. It answers three
  operator questions without decorative metrics: whether the dynamic runtime
  is ready, where component work should resume, and which real conditions need
  attention.
- Runtime, component, connector, and namespace requests load concurrently with
  `Promise.allSettled`; one unavailable catalog leaves successful sections
  visible and is never rendered as a misleading zero or green state.
- Live review verified ready generation 12, five recent components, two tested
  connectors, two governed namespaces, and the two actual draft components in
  the attention queue. The initial `null` loading-state crash found in browser
  review is regression-tested. Fable 5.1 approval remains pending.

### Cache and warmup — durable evidence implemented, Fable pending, 2026-09-20

- Cache configuration and warmup execution are visibly separated. Saving a
  cache or plan never implies that a run occurred.
- The case-product planner rejects malformed, empty, and duplicate dimensions
  and displays the exact Cartesian expression, planned count, MaxCases
  capacity, and row limit before save.
- Warmup now executes Datly’s canonical authored-case runtime path under a
  detached five-minute server lifetime. Duplicate starts attach to the active
  immutable run rather than launching another operation.
- Schema v5 persists accepted/running/completed/partial/failed evidence with
  report/version/source revision, plan fingerprint, safe target identity,
  planned/completed cases, prepared entries, duration, actor, timestamps, and
  sanitized diagnostics. No DSN or credential is stored or returned.
- Live desktop review verified two completed `vendor-spend` runs. Closing and
  reopening the dialog retains the latest evidence and history; the latest run
  prepared one authored case/entry in 1.2ms. Fable 5.1 approval remains pending.
- Rendered operational coverage restores durable history, blocks malformed or
  duplicate Cartesian dimensions before mutation, and keeps the last successful
  run visible when “Run again” fails. The test exposed and fixed premature
  clearing of prior evidence before server acceptance.

### Exact-version preview and relation evidence — implemented, Fable pending, 2026-09-20

- Preview now always calls the authorization-scoped `preview.execute` SDK
  operation for the exact report/version being edited. Authenticated mode no
  longer substitutes the active published route for a draft preview.
- Server evidence includes report/version/source revision, connector, enforced
  row limit, returned rows, encoded bytes, and truncation. Execution has a
  30-second deadline, 200-row ceiling, and 2 MiB encoded response budget.
- Live desktop review verified draft `Vendor Spend Analysis` v1 revision 9 on
  `reporting_mysql`, returning 1/50 rows and 45 bytes within budget.
- Relation selection stays in the graph workspace. Its right rail distinguishes
  child SQL, isolated child testing, full graph preview, relation editing, and a
  dedicated exact-version relation test; the full-width lower pane remains the
  child column catalog.
- The relation test executes the compiled Datly reader and inspects its typed
  assembled output. Live `Vendor → Products` evidence reported 3 parent rows,
  2 matched, 1 unmatched, and 3 children attached for v2 revision 1. It does
  not run parallel hand-authored SQL. Fable 5.1 approval remains pending.

### Authenticated BFF boundary — hardened locally, 2026-09-20

- Authenticated startup rejects missing origins, missing runtime-admin tokens,
  the known local token, and unsupported mode strings. Development retains an
  explicit loopback-only local default.
- Exact-origin CORS rejects cross-origin requests; cookie-authenticated unsafe
  methods require Origin. Request IDs are generated, forwarded, exposed to the
  allowed browser, and appended to SDK errors.
- Browser proxies now deny dynamic `/_studio/*` administration paths and
  allowlist forwarded headers, stripping runtime/development control headers.
  Responses use `Cache-Control: no-store`; runtime token checks are constant-time.
