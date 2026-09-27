# Datly Studio UI production plan

Status: active implementation plan
Audience: Datly, Studio, Forge, and UX reviewers
Product mode: operate — production SQL authors building Datly reader components

Product boundary: Studio authors MCP tools and Skills from Datly components.
The `components` table and `components.*` SDK operations are component-record
contracts, not a presentation-report feature. Presentation reports are
outside this project. A future UI definition may be a typed, versioned view over the
compiled component descriptor; it must not duplicate Datly execution logic.

## Product direction

Datly Studio opens a component on its compiled view graph. The graph is the
navigation and dependency surface, not a second executable model. Selecting a
graph view activates its inline, right-side settings inspector in the component
workspace; it does not open a view tab. Every view owns its exact SQL/resource
source, connector and dialect context, relation role, field contract, predicates,
selectors, cache/warmup policy, test evidence, and diagnostics.

Studio never duplicates Datly authoring semantics. All edits are typed Datly 1.0
Reader Builder operations. If a source-preserving operation is missing, it is
implemented and tested generically in Datly before Studio exposes the control.

Raw DQL is reserved for users with `can_use_dql` and remains an advanced mode.
Every ordinary option has a cohesive, scope-aware UI instead of a generic DQL or
setting text field.

## Authoring hierarchy

1. **Component graph** — view topology, relation identity, readiness, and
   HTTP/MCP/publication status.
2. **Selected-view inspector** — an inline right-side inspector for the selected
   graph view: dialect, connector, relation context, fields, predicates,
   selectors, cache/warmup, tests, and diagnostics. Its full lower area is the
   searchable, paginated column catalog; there are no separate explanatory
   “View behavior” or “Column authoring” cards.
3. **Column contract** — cast/type, tag, public name/casing, SQL mapping,
   internal/private output policy, nullability, selector permissions, and
   dimension/measure roles.
4. **Component contract** — package identity, route, typed inputs and constants,
   cube/composition, resources/skills, MCP exposure, permissions, validation,
   versions, and publication.

The UI distinguishes DQL ownership:

- `#setting`: component-wide runtime, exposure, report/cube, resource, and
  component policy.
- `#define`: typed input/output bindings, constants, parameters, predicates,
  codecs, selectors, and activation.
- executable `#set`: template state only; never presented as an ordinary
  binding or settings editor.
- view controls: selected-view SQL/relation/field/selector/cache metadata,
  expressed through Reader Builder operations and the compiled descriptor.

## Scale targets

- At least 200 predicates and parameters without a single oversized form.
- At least 100 columns per view with fast search, filters, stable selection,
  pagination or virtualization, and bulk contract actions.
- Large graphs retain search, hierarchy navigation, collapse, diagnostics
  filtering, fit/center actions, and an accessible tree representation.
- No horizontal document overflow at supported narrow widths.

## Current verified deficiencies

- The independent GPT-6 Sol UX review resumed after the UI unit suite passed.
  Its scoped Skills findings were fixed and re-reviewed without remaining P1/P2
  issues. The later, substantial Output catalog change received a separate
  post-test GPT-6 Sol review with no remaining scoped P1/P2 issue. A 2026-09-26
  whole-product local Chrome pass found no confirmed P1/P2 issue in the mocked
  auth, graph/SQL and wide-column flows. Deployed IdP, denied-user, spoken
  screen-reader and 200% zoom conditions still prevent production acceptance.
  A later direct-auth Sign out change received its own post-test GPT-6 Sol
  review at desktop and 390 px; no scoped P1/P2 issue remained.
- The historical Codex Astra low review approved targeted graph/SQL/component/MCP changes
  after all reported P1 findings were fixed; untested authentication,
  deployment, failure-recovery, theme/contrast/zoom, and full screen-reader
  scenarios remain conditions rather than inferred approval.
- The production build keeps the SQL editor in a lazy 673 kB chunk. It is not
  initial-load JavaScript, but Vite still reports the chunk-size warning and a
  real deployment profile should decide whether further editor splitting is
  worthwhile.
- Rendered interaction coverage now protects the highest-risk catalog,
  graph/SQL/column, schema-composition, and conflict flows, but does not yet
  simulate every command dialog end to end.
- Deployment-specific identity settings and the default MCP publication policy
  remain product/deployment decisions, not browser defaults Studio can safely
  invent.

## Delivery sequence

### Phase 0 — dependable review baseline

- Keep `npm test` and `npm run build` green.
- Add rendered-component tests for navigation, builder tabs, graph selection,
  SQL workspace, column contracts, permission boundaries, and stale conflicts.
- Add deterministic representative fixtures for large graphs, 200 predicates,
  and 100 columns.
- Establish desktop and narrow screenshot routes.
- The separate GPT-6 Sol UX review resumed after the UI unit suite passed.
  Record model, date, viewport, scenario, findings, fixes, and final verdict
  in `UX_REVIEW.md`. Historical local reviews do not supply this verdict.

### Phase 1 — shell and governed catalogs

- Information architecture for Overview, Connectors, Namespaces, Components,
  Runtime, MCP, and Administration.
- Dedicated governed namespace catalog and component namespace selection.
- Catalog search/filter/status and useful operational summaries.
- Responsive navigation becomes a drawer rather than consuming permanent
  workspace width on narrow screens.

### Phase 2 — graph-first component workspace

- Separate identity/readiness header from a grouped command bar.
- Graph explorer/tree plus graph canvas and an inline right-side selected-view
  settings inspector. Selecting a graph view must not create or focus a tab.
- Rich node/edge evidence and diagnostic/readiness badges.
- Graph search, collapse, fit, center, keyboard navigation, and persisted
  selection without allowing drag to change semantics.
- Publish remains unavailable until the exact source revision is valid.

### Phase 3 — production view and SQL workspace

- SQL remains a dedicated editable-resource tab, opened only by an explicit SQL
  action; graph selection itself stays in the inline inspector.
- Per-view connector/dialect/resource identity, test state, schema diff, and
  source-linked diagnostics.
- SQL test fixtures for typed parameters and selectors, timeout/row/byte budgets,
  bound arguments with redaction, inferred columns, plan, rows, nested output,
  raw response, and truncation.
- Stored compiled field metadata remains usable when the connector is offline;
  live metadata is clearly labeled as refreshable evidence.
- Large column catalog with search, role/visibility/type filters, virtualization
  or pagination, keyboard selection, and bulk-safe operations.

### Phase 4 — cohesive Datly contract controls

- Inputs: parameters, constants, codecs, defaults/examples, activation, output
  emission, and selector targets.
- Predicates: searchable catalog plus synchronized per-view boolean group
  composer; authorization groups are visibly locked.
- Columns: cast/type, tag, mapping, public name, internal/private policy,
  selector eligibility, nullability, and cube roles.
- Component settings: route, cube/composition, MCP, resources, and skills.
- View policy: relations, selectors, cache, and warmup at their actual Datly
  scope.
- Instance constants remain trusted deployment configuration; request values
  remain bound parameters and never become identifier interpolation.

### Phase 5 — seamless MCP and publication

- One release workspace joins validation evidence, route and MCP schemas,
  resource/skill readiness, permissions, version diff, desired generation,
  active generation, failure recovery, rollback, and unpublish.
- Preview and published HTTP/MCP use the same compiled typed component and
  authorization path.
- Globally unique owner-scoped MCP identities and explicit cube/compose opt-in.
- Publication progress is observable and failed activation visibly preserves
  the prior generation.

### Phase 6 — production hardening

- WCAG AA keyboard, focus, semantics, contrast, and reduced-motion verification.
- Forge theme tokens replace Studio hard-coded color drift; light and dark
  themes are verified.
- Request cancellation, request IDs, retry boundaries, offline/expired-session
  recovery, and large-data performance.
- Code splitting and initial-load budget.
- Endly/MySQL authoring-to-publication proof and HTTP/MCP execution evidence.

## Window and view approval matrix

Every row requires functional SDK evidence, loading/empty/error/denied/conflict
states as applicable, desktop and narrow review, keyboard review, rendered tests,
production build, and a recorded independent GPT-6 Sol reviewer verdict.
Historical Astra output does not close the new review gate. The entries below
retain prior local verdicts; current scoped findings are in `UX_REVIEW.md`.

| Surface | Functional | Responsive/a11y | Historical local review |
| --- | --- | --- | --- |
| Session and application shell | hardened BFF plus encrypted DB-backed shared sessions and issuer/audience binding implemented | security review, full Go suite, and rendered loading/sign-in/error/retry/expiry tests passed | Codex Astra low: **APPROVE WITH CONDITIONS** — deployed IdP return/expiry |
| Overview | implemented with partial-failure isolation | desktop web locally verified | Codex Astra low: **APPROVE** |
| Connector catalog and editor | server-side search/status pagination, probe-gated lifecycle actions, secret-safe editing, and optimistic conflicts implemented | desktop web verified against the preseeded SQLite catalog with rendered search/status/empty/conflict-refresh tests | Codex Astra low: **APPROVE** |
| Namespace catalog and editor | governed CRUD plus server-side search/status pagination, stale-etag recovery, and reference-safe deletion implemented | desktop web plus rendered stale reload/create preservation and backend delete-block tests verified | Codex Astra low: **APPROVE** |
| Component catalog and create flow | governed namespace and tested-active-connector creation plus server-derived identity implemented | desktop web verified with rendered empty-namespace and server-rejection draft-preservation tests | Codex Astra low: **APPROVE** |
| Schema browser and root/subview creation | connector/schema/table discovery, starter SQL, bounded test, and typed root/subview actions implemented | desktop web verified against preseeded SQLite plus rendered discovery/error/SQL/root/subview/reset/relation-validation tests | Codex Astra low: **APPROVE** |
| Component graph | implemented; node selection stays in component workspace | desktop web plus rendered graph-selection test verified | Codex Astra low: **APPROVE** |
| Selected-view inline inspector | implemented | desktop web plus rendered inline-inspector/metadata-merge test verified | Codex Astra low: **APPROVE** |
| Per-view SQL editable-resource tab | exact embedded SQL plus compiled connector/driver, source, graph relation, contract, revision metadata and unsaved-navigation protection implemented | desktop web plus rendered explicit-open/dirty-navigation test verified | Codex Astra low: **APPROVE** |
| Column catalog and contract actions | implemented; compiled + connector metadata merge | desktop web plus rendered cast/visibility/cube-role/public-name/tag-validation tests verified | Codex Astra low: **APPROVE** |
| Parameter and constants workspace | separated request/constant workspaces plus typed source, selector, default, codec, resource/route activation, description/example, and output controls implemented | desktop web plus rendered 120-input pagination/search and constant-zero preservation verified | Codex Astra low: **APPROVE** |
| Predicate catalog and group composer | scalable catalog plus canonical FilterGroup, CombineAnd/CombineOr, persistent connector, and WHERE/AND controls implemented | desktop web plus rendered 240-predicate search/pagination and outer-composition mutation verified | Codex Astra low: **APPROVE** |
| Relation editor and impact review | implemented; selected edge stays in graph context with keys, classification, and distinct child/graph/relation actions | desktop web plus rendered key/update/rejection/dependent-removal tests verified | Codex Astra low: **APPROVE** |
| Preview and view/relation test evidence | exact-version bounded preview/view evidence plus typed Datly relation attachment metrics implemented | desktop web locally verified | Codex Astra low: **APPROVE** |
| Cube and composition workspace | cohesive Edit component settings + ordered typed-frame composition lab implemented | desktop web plus rendered atomic-settings, accessible-budget, frame/inheritance/payload/error tests verified | Codex Astra low: **APPROVE** |
| Cache and warmup workspace | durable server-owned runs + bounded Cartesian case planner implemented | desktop web plus rendered history/invalid-plan/failure-preservation tests verified | Codex Astra low: **APPROVE** |
| HTTP/MCP exposure | atomic Edit component settings plus distinct reader, Cube, and CubeCompose generated MCP components implemented | desktop web plus rendered enable/name/owner-prefix/description-resource and runtime-catalog tests verified; Astra targeted approval | Codex Astra low: **APPROVE** |
| Resources and skills | complete file/folder/skill CRUD with dependency-safe deletion and atomic revision conflicts implemented | desktop web plus rendered stale-write preservation/reload tests and live populated v3 lifecycle/dependency review verified | Codex Astra low: **APPROVE** |
| Permissions | owner-only least-privilege grants with capability invariants, row etags, stale recovery, and generated Datly ACL contracts implemented | desktop web plus rendered preset/dependency/stale-reload tests verified | Codex Astra low: **APPROVE** |
| Validation diagnostics | exact-revision runtime validation with structured severity/code/hint/location and permission-gated source recovery implemented | desktop web plus rendered invalid/request-failure/retry tests verified | Codex Astra low: **APPROVE** |
| Version history, publication, rollback | exact-version staged lifecycle, guarded destructive actions, append-only owner-scoped release events, and serving-to-proposed contract comparison implemented | desktop web plus rendered snapshot-mismatch and activation-failure preservation tests verified | Codex Astra low: **APPROVE** |
| Runtime generations and MCP catalog | authorization-scoped generation projection, explicit tool/resource catalogs, Datly-derived Cube/CubeCompose identities, published skills, and authenticated host readiness implemented | desktop web and rendered catalog tests verified; Astra targeted approval | Codex Astra low: **APPROVE** |
| Stale revision recovery | typed conflict metadata plus review-draft/reload-latest recovery implemented | desktop and narrow web verified with an authentic stale revision plus rendered decision/loading tests | Codex Astra low: **APPROVE** |

Independent UX approval is pending for every surface. Measured desktop contrast and 200%-equivalent
no-overflow evidence are recorded in `UX_REVIEW.md`; Studio explicitly supports
the light color scheme only. Deployment IdP integration and whole-product
assistive-technology validation remain external production conditions; no
test-only inference waives them.
