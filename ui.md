# Datly Studio UI — Reader Graph Builder

## 1. Product objective

Datly Studio is a visual authoring environment for **dynamic Datly 1.0 readers**.
It lets an authorized user compose a typed data graph, test every view, preview
the assembled result, and publish a versioned reader without learning the full
DQL grammar.

The first product is intentionally read-only:

- build Datly GET readers;
- add, update, relate, test, and remove views;
- add typed parameters and predicates;
- configure selectors, derived outputs, cubes, cube composition, caches, and
  warmup;
- preview and publish a complete reader generation;
- never generate PATCH/POST/PUT writers for a dynamic reader component;
- never let the browser execute SQL or connect directly to a database.

The UI uses Viant Forge for windows, data sources, forms, editors, tables,
dialogs, status workflows, permission boundaries, and responsive layout. Every
backend interaction goes through the JSX `StudioAPI`; components never call raw
operation strings, SQL stores, generated Datly packages, or legacy control APIs.

## 2. Canonical-authority rule

There must be one component definition, not a DQL model and a competing canvas
model.

```text
Versioned authored DQL + resources
              │
              ▼
Datly readerbuilder.inspect / Datly compiler
              │
              ▼
Canonical component + view/relation descriptor
              │
              ▼
Forge graph, forms, SQL editor, preview, diagnostics
```

The graph is a visual projection of Datly's compiled component. A graph edit is
a typed command applied by Datly's source-preserving reader builder. The command
returns candidate DQL, canonical structure, and source-linked diagnostics. A
failed edit leaves the prior version unchanged.

`report_views`, `report_fields`, `report_parameters`, `report_predicates`, and
`report_cube_configs` are queryable projections of a Datly component version. They do not
become a second executable model. `report_versions.authored_dql`, resources,
component snapshot, type manifest, source revision, and digest remain the
version authority.

## 3. Product vocabulary

| UI term | Datly meaning |
| --- | --- |
| Reader | One typed GET component with input, output, views, behavior, and exposure |
| Data graph | Root view plus relation, independent, and derived views |
| View | A typed Datly query result backed by SQL, table, resource, derived, or virtual source |
| Edge | A relation with parent, child, complete relation keys, relation type, and cardinality |
| Input | A typed parameter bound from query, path, header, constant, resource, or component |
| Predicate | A parameterized filter attached to an input and expanded in one or more views |
| Selector | Controlled projection/filter/order/page/limit/offset behavior for a named view |
| Cube | Component-derived grouped reader with declared dimensions, measures, and filters |
| Composition | A bounded query over several authorized cube frames |
| Cache | A named native SQLX read cache selected for a prepared view |
| Warmup | Bounded server-side population of declared cache cases |
| Test | Explicit evidence at SQL, relation, graph, cube, cache, or protocol level |

## 4. Information architecture

### 4.1 Application menu

```text
Datly Studio
├── Overview
├── Connectors
│   ├── Catalog
│   └── Health
├── Readers
│   ├── Drafts
│   ├── Published
│   └── Archived
├── Runtime
│   ├── Active generation
│   ├── Cache & warmup
│   └── MCP tools
└── Administration
    ├── Permissions
    └── Diagnostics
```

The Guardian pattern is appropriate for the shell: persistent top navbar,
collapsible searchable tree navigation, and Forge windows in the main workspace.
The Steward pattern is appropriate inside each window: metadata-owned data
sources, authorization, containers, responsive tables, toolbars, and dialogs.

### 4.2 Forge metadata layout

```text
ui/forge/
├── datasources/
│   ├── studio_connectors.yaml
│   ├── studio_connector_test.yaml
│   ├── studio_readers.yaml
│   ├── studio_reader_definition.yaml
│   ├── studio_reader_edit.yaml
│   ├── studio_view_test.yaml
│   ├── studio_reader_preview.yaml
│   ├── studio_reader_validate.yaml
│   ├── studio_reader_publish.yaml
│   └── studio_reader_warmup.yaml
└── windows/
    ├── connectorList/shared/{main,content}.yaml
    ├── readerList/shared/{main,content}.yaml
    ├── readerBuilder/shared/{main,content}.yaml
    └── runtimeStatus/shared/{main,content}.yaml
```

These data sources require a Forge `studio_sdk` connector/provider that invokes
named methods on `StudioAPI`. Do not encode speculative raw HTTP details in every
YAML file. The provider owns API base URL, development/authenticated mode,
bearer handling, SDK error normalization, cancellation, and request IDs.

## 5. Reader Builder window

### 5.1 Main layout

The user-facing catalog name is **Components**. SDK and storage continue to use
their stable `reports`/`reportId` identifiers internally; a Component is the
versioned Studio object that owns one dynamic reader graph.

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Breadcrumb / reader name / Draft v7 / Valid                  Save  Preview  │
│ source revision 18                     Validate  Publish ▾                  │
├──────────────────┬──────────────────────────────────┬────────────────────────┤
│ Graph explorer   │ Data graph canvas                │ Inspector              │
│                  │                                  │                        │
│ ▾ Views          │  ┌──────────────┐                │ General                │
│   ● vendors      │  │ Vendors ROOT │──1:N────────┐  │ Source                 │
│   └─ contacts    │  └──────────────┘             │  │ Fields                 │
│   └─ spend       │                         ┌──────▼─┐│ Relation               │
│ ▾ Inputs         │                         │Contact ││ Predicates             │
│ ▾ Outputs        │                         └────────┘│ Selectors              │
│ ▾ Exposures      │                                  │ Cache / warmup         │
│                  │                                  │ Cube / composition     │
├──────────────────┴──────────────────────────────────┴────────────────────────┤
│ SQL | Parameters | Preview | Diagnostics | DQL diff | History              │
└──────────────────────────────────────────────────────────────────────────────┘
```

Use Forge primitives where they fit:

- `TreeEditor` for the view/output outline;
- a dedicated graph canvas component for nodes and relation edges;
- `DraftForm` for inspector sections;
- `Editor` for resource SQL and advanced DQL;
- `ResponsiveDataGrid` for test and preview rows;
- `DataStateBoundary` for loading/empty/error states;
- `StatusWorkflow` for draft → validated → published;
- `HistoryDiff` for version/source changes;
- `PermissionBoundary` for DQL, edit, test, warmup, and publish controls;
- `QueryToolbar` for graph search, zoom, validation filters, and test controls.

### 5.2 Header and command bar

Always show:

- reader title and immutable report ID;
- draft/version number;
- source revision and unsaved/saving/saved state;
- compile state: pending, valid, invalid, or error;
- active connector and last connection-test status;
- current runtime state when published;
- undo/redo for accepted command history;
- Validate, Preview, and Publish actions according to permissions.

Publish is unavailable until validation succeeds at the current source revision.
An older successful validation must not authorize publishing a newer draft.

## 6. Graph model and interactions

### 6.1 View node types

| Node | Visual treatment | Required information |
| --- | --- | --- |
| Root | Strong primary border | Name, row type, source, connector, output holder |
| Relation | Normal card attached by edge | Parent, relation name, keys, cardinality, match policy |
| Independent | Detached card with output binding | Name, output path, source, type |
| Derived | Aggregate/badge treatment | Output path, aggregate scope, source dependency |

Cache, warmup, cube, MCP, selector, and authorization states are node badges or
inspector sections. They are policies on a view/component, not fake data nodes.

The graph is database-agnostic. Table identifiers may follow BigQuery,
MySQL, PostgreSQL, or SQLite conventions, but every view binding is an exact
active Studio connector name compiled into Datly metadata. Studio never keeps a
legacy per-view connector registry and never aliases an unavailable connector
to the report default.

### 6.2 View node content

Collapsed cards show:

- view name and role;
- source kind and connector override;
- output type and cardinality;
- field count;
- predicate/selector count;
- SQL test state and timestamp;
- cache, cube, MCP, and diagnostic badges.

Expanded cards show a compact field list with data type, nullable, filterable,
orderable, groupable, and measurable roles. Internal relation keys stay visible
to the author but are distinguished from public response fields.

### 6.3 Add root view

The create-reader wizard asks for:

1. title, slug, immutable owner-derived package identity;
2. active connector;
3. source mode: table, SQL editor, or SQL resource;
4. root view name, row type name, and output collection name;
5. schema discovery/test;
6. initial route and optional MCP exposure.

The server creates a report and first draft version, generates a complete reader
contract with required `#package`, input/output types, case format, route,
connector, typed output holder, and root `type(...)`, then returns the compiled
descriptor. The browser does not concatenate DQL.

Dynamic package identity is server-owned. The owner segment is derived only from
the verified JWT `sub` (not `username`, email, or another display claim): take
the part before `@`, lowercase it, and remove dots, underscores, and other
non-alphanumeric characters. A report package is
`github.com/viant/datly-studio/dynamic/<owner>/<report-id>/reader`. Existing
sources move only through Datly Reader Builder's source-preserving `setPackage`
operation and a source-revision precondition.

Business namespace is separate catalog metadata, not a Go package or connector
alias. It is a validated dot-separated lowercase value such as `inventory` or
`delivery.forecasting`, defaults to `general`, and supports simple report
filtering/grouping without changing component identity.

Permissions grant a report capability to one exact verified JWT `sub`; there is
no role selector until a shared, verified role-claim mapping exists in both the
Studio SDK and dynamic runtime.

### 6.4 Add joined or derived view

View creation starts by selecting its Datly relation type from compiled
metadata: `subview` for an ordinary joined child or `derived` for an independent
query-derived output such as totals or bounds.

Joined-subview creation is a graph-first wizard:

1. select parent node;
2. choose relation name and child view name;
3. select SQL/table/resource source and connector inheritance/override;
4. test child SQL;
5. map all parent/child key pairs;
6. select one/many cardinality and match policy;
7. validate relation with duplicate parent IDs and NULL/no-match samples;
8. apply one `addView` command.

Datly's reader builder supports typed `addView` with `kind`, `name`, `sql`, and
`parent`. A `subview` also requires complete `on` links; its outer SQL uses a
canonical `JOIN` only as graph syntax. `JOIN` versus `LEFT JOIN` does not define
runtime attachment semantics because Datly decomposes and executes the views
separately. A `derived` view rejects join/ON metadata and is attached to the
root output with generated upper-camel row type
identity. Datly compiles the result and verifies the resulting
`spec.Relation.Kind` and requested parent. The UI reads `Relation.Kind`,
cardinality, match strategy, links, and child metadata from the
compiled component; it never infers relation type by parsing SQL or constructs
an edge without a successful compiled relation.

### 6.5 Update or remove a view

Updating SQL uses `updateView`; changing a relation should be represented by an
explicit relation update extension rather than remove/re-add when identity and
history matter.

Removal opens an impact panel:

- descendant views;
- output bindings;
- predicates/selectors targeting the view;
- cube/cache/warmup dependencies;
- MCP schema impact;
- published consumers where known.

Removing a parent with children is blocked until children are moved or removed.
Datly's current builder already rolls back an invalid parent removal and
preserves sibling joins and outer clauses.

### 6.6 Graph ergonomics

- drag changes layout only, never semantic parentage;
- re-parent uses a dedicated command and relation-key review;
- auto-layout by hierarchy, with stable persisted viewport/layout metadata;
- zoom-to-fit, center selected, collapse descendants, and focus diagnostics;
- minimap for larger graphs;
- keyboard node navigation and accessible non-canvas tree equivalent;
- selection is reflected in URL/window state so refresh restores context.

## 7. SQL authoring and per-view tests

Full-reader and isolated-view results default to a tabular Datly envelope. The
first output collection becomes the root table; scalar fields become columns,
while array-valued subviews are shown as count badges and expandable nested
tables. Derived object outputs appear as separate structured detail sections.
Every result also offers a Table / JSON presentation switch. JSON is pretty
printed from the same response in memory and never triggers a second query.

### 7.1 SQL workspace

Selecting a view opens its SQL/resource in the bottom dock. The editor provides:

- SQL dialect and connector context;
- schema/table/column completion from authorized discovery;
- DQL template and parameter completion;
- format and source-linked diagnostics;
- column/result metadata diff after test;
- explicit aliases required by cube composition;
- read-only generated outer DQL preview.

Simple mode edits the selected inner-view SQL. Advanced mode shows the complete
authored DQL and requires `can_use_dql`. Switching modes never silently loses a
structured command; the server returns exact export limitations.

### 7.2 “Test this view” contract

Every view card and SQL editor has **Test view**. This is a server-side SDK
operation, never browser SQL execution. Required request:

```text
report ID + version + expected source revision
view stable identity
typed parameter fixture
selector fixture
row limit + timeout
test mode: SQL | relation | graph
```

The server:

1. resolves the authorized connector and secret;
2. compiles the current candidate with the same type/resource catalog;
3. binds verified identity and typed parameters;
4. enforces read-only execution, timeout, and row/byte budgets;
5. executes the selected inner view through Datly/SQLX ownership;
6. returns SQL, ordered arguments with sensitive values redacted, fields,
   typed rows, duration, truncation, and diagnostics.

Result tabs:

- Rows;
- Columns and inferred types;
- Bound parameters;
- Query plan when the connector supports an authorized explain operation;
- Diagnostics;
- Raw response in an advanced inspector.

Testing one SQL source proves only that source. It does not prove relation
attachment, output encoding, authorization, cache replay, cube grouping, or
protocol exposure. Evidence levels are shown separately:

| Test | Proves |
| --- | --- |
| SQL | Selected source parses, binds, executes, and decodes |
| Relation | Parent/child keys and cardinality assemble correctly |
| Graph preview | Complete reader input → typed output behavior |
| Cube | Dimension/measure/filter grouping behavior |
| Compose | Frame inheritance, aliases, joins, and budgets |
| Cache | Miss/fill/hit/expiry and projection compatibility |
| HTTP/MCP | Actual public binding, auth, schema, and response |

### 7.3 Parameter fixtures

The test drawer generates a form from canonical inputs. Omitted and explicitly
supplied zero/false/empty/null are distinct. Internal Has markers are never shown
as fields. Required authorization inputs are server/host owned and cannot be
overridden with arbitrary browser claims.

Named test cases may be saved with the report version. Secret values and bearer
tokens are never persisted in the fixture.

## 8. Parameters, predicates, and selectors

### 8.1 Parameter editor

Fields:

- canonical name and type expression;
- source kind and source name/location;
- required/optional;
- codec;
- route activation / `WithURI`;
- output emission;
- target query-selector view;
- example/default for test fixtures where permitted.

`addField` is currently additive. Rename, type change, reorder, and remove need
explicit source-preserving reader-builder operations before the UI exposes them
as ordinary structured edits.

### 8.2 Predicate editor

The predicate UI has two synchronized representations:

- predicates listed under each parameter;
- a boolean group composer under each target view.

Each predicate shows target view, group number, predicate kind, arguments,
apply-when-absent, and expansion views. Each group shows AND/OR within the group
and how it combines with other groups. The UI must require an explicit combine
operator when an existing composition would otherwise be ambiguous.

Authorization is visually locked in a dedicated required group (Studio uses
group 99). Optional user filters cannot broaden or replace it. Removing or
editing the authorization predicate requires a separate privileged workflow.

Current Datly operations map directly:

- `addFieldPredicate`;
- `updateFieldPredicate` with occurrence and expected target;
- `removeFieldPredicate`;
- automatic insertion of a missing predicate-builder expansion;
- rejection of accidental cross-view group sharing.

Large readers use the same model with a catalog presentation. For hundreds of
predicates, the window provides search, SQL/handler filtering,
Include/Exclude/Core family filtering, and 25/50/100-row pagination. Summary
counts remain visible, pagination stays outside the scrolling table, and Add or
Edit is a focused form for one predicate. The target list is derived from the
compiled root graph, with the root first; it is not inferred from SQL aliases
in browser code. The boolean composer is a separate synchronized view of the
same compiled declarations and is grouped by target view and predicate group.

### 8.3 MCP names

The dynamic MCP listener is shared across users, so every explicit base-reader
tool name is globally unique. Studio requires the server-derived owner prefix,
for example `awitas.forecasting.read`, and rejects a name already used by another
reader. The browser may suggest the route-derived suffix but cannot choose or
override the owner segment. Report-derived cube/compose identities inherit the
owner-scoped component package. Tool metadata is draft state until validation
and publication; HTTP and MCP execute the same typed component and authorization.

Validation is a dedicated pre-publication window. It opens all required active
Studio connectors, compiles `RuntimeContracts`, resolves linked types and
resources, builds the Datly artifact, verifies route/MCP identities, and
initializes reader execution. It does not issue the business query and does not
mutate the active runtime generation. The exact source revision becomes `valid`
or `invalid` with persisted diagnostics; static Reader Builder inspection alone
is not publication authority.

Publication is a staged runtime operation. Studio writes a building generation
and a pending report publication, asks the dedicated dynamic host to reload that
candidate generation, then promotes the generation and participating report
publications to `active` only after the host confirms an atomic swap. A reload
failure records diagnostics and restores the previous publication state. The
publish window accepts only an optional release note; publisher identity comes
from the verified principal and runtime addresses, connector handles, and
generation numbers never enter browser DTOs.

## 8.4 Versioned resources and skills

Reader resources are immutable rows attached to a report version. A text file
has an owner-scoped namespace, a relative resource path, server-derived digest,
and content size. Saving a file, MCP folder, or skill root increments the source
revision and resets validation. Folders explicitly select the resource namespace,
published root, and safe URI prefix; they are not inferred from a file tree.

Skills are explicit folder-root declarations. The selected root must contain
`SKILL.md`; Datly compiles its frontmatter and sealed subtree during validation
and again while staging the runtime generation. A resource namespace begins with
the server-derived owner package segment and is reserved to one reader, avoiding
cross-user resource or skill collisions. Browser code handles only text DTOs;
it never creates an embedded filesystem, computes trust metadata, or registers
MCP resources directly.

Unpublish uses the same staged reload protocol. Studio marks the target
publication as `unpublishing`, stages a generation whose host source excludes
that report, and activates it only after the dynamic host swap succeeds. A
reload failure restores the previous publication state. The lifecycle window
reports removal explicitly and allows a validated version to be republished.

The same window reads the global runtime status through the Studio SDK. It
reports active generation, reader count, and activation state separately from
the selected report publication, so authors can distinguish an active report row
from an operational dynamic host.

Version history remains immutable. The lifecycle window lists version number,
source revision, and validation state; rollback is enabled only for a validated
historical version. It publishes that selected version through the same staged
runtime protocol and marks the previously published version `superseded`. It
does not copy DQL into the browser or overwrite a newer draft.

### 8.5 Selector policy

Configure each view independently:

- projection;
- criteria/filtering;
- ordering and allowed columns;
- limit/default/max/no-limit policy;
- offset/page;
- SQL methods where permitted.

The UI distinguishes “field exists” from “field may be publicly selected.”
Required relation keys may remain internally loaded even when absent from the
public payload.

## 9. Cube and cube composition

### 9.1 Enable cube

Cube activation is a guided readiness check, not a single unchecked toggle.

The Cube tab classifies fields as:

- dimensions;
- measures;
- filters;
- orderable outputs;
- required inherited inputs.

Initial current-builder support validates one simple grouped root SELECT: no
joins, CTEs, HAVING, windows, DISTINCT, or set operations; at least one grouped
dimension and one aliased aggregate measure are required. The UI explains that
as the current supported authoring envelope rather than silently rewriting SQL.

Settings are applied via the reader builder's `setSetting` for `cube`/`report`.
Source authorization and non-query inputs remain part of every derived request.

### 9.2 Enable composition

Composition requires cube/report enabled first. Configure:

- enabled;
- MCP tool exposure independently;
- maximum cubes (default 8);
- maximum result limit (default 100);
- timeout (default 30,000 ms);
- allowed output aliases and field mappings.

The Compose playground creates ordered frames, supports `inheritFrom`, edits
explicit filters, and authors SQL only over `$CubeSQL1...N` and validated output
columns. Omitted frame filters cannot inherit unrelated outer request values.
Budget failures are previewed before execution where possible and enforced again
server-side.

### 9.3 Cube test matrix

Test at minimum:

- each dimension alone and representative combinations;
- aggregate-only request if allowed;
- required filters and authorization denial;
- nullable joins in the source reader;
- explicit aliases;
- inherited composition frames;
- maximum cube/limit/timeout violations;
- cache reuse retaining every warmed dimension.

## 10. Cache and pre-warmup

### 10.1 Cache editor

Caching is configured per prepared view. The UI requires an explicit backend:

- AFS: name, positive TTL, location;
- Aerospike: approved provider, namespace/set location, positive whole-second
  TTL, and host-owned connection configuration.

There is no automatic AFS/Aerospike fallback. A cache name must resolve to an
enabled cache service. The browser never supplies Aerospike credentials.

### 10.2 Warmup planner

The warmup editor supports:

- dedicated equivalent connector;
- index column and canonical index parameter;
- named parameter case values;
- full case-product count;
- `MaxCases`, limit, timeout, and field names;
- selected related/derived outputs;
- startup/manual schedule policy where supported.

Before applying, show an estimate:

```text
3 periods × 2 granularities × 4 tenants = 24 cases
MaxCases = 30                              Ready
```

Warn when indexed warmup conflicts with fixed-ID SQL, when a case omits required
authorization, or when cube reuse would drop a warmed dimension.

### 10.3 Warmup operations and evidence

Manual warmup is a privileged SDK action backed by Datly's warmup owner and a
server-owned lifetime. Show accepted, running, completed, partial, and failed
states. Client disconnection is not completion.

Provide evidence for:

- cold miss and DB fill;
- subsequent hit;
- expiry;
- warm hit while source DB is unavailable in a controlled test;
- query/view/cache/index identity;
- selected connector;
- returned rows versus unknown physical cache-record count.

## 11. Preview, validation, versions, and publication

### 11.1 Command lifecycle

Every structured mutation includes `expectedSourceRevision`. The server applies
the command to the authoritative DQL, compiles the candidate, persists a new
revision atomically, and returns:

- accepted/rejected;
- new source revision;
- canonical component descriptor;
- graph projection;
- diagnostics;
- DQL diff;
- affected view/output/exposure identities.

Revision conflict opens a three-way comparison: current server source, the
user's base revision, and intended typed command. Never blindly replay a textual
patch against changed DQL.

### 11.2 Validation

Validation is staged and visibly reports what was and was not proven:

1. source/DQL parse;
2. reader-builder invariant checks;
3. connector/schema refinement;
4. type/resource resolution;
5. generated destination/layout validation where persisted shapes are used;
6. Go/runtime registration where applicable;
7. saved test cases;
8. HTTP/MCP/cube/cache checks selected for publication policy.

### 11.3 Preview

Full preview invokes the public reader with typed input and renders its real
output. It offers table/tree/raw views and shows duration, row limits,
diagnostics, loaded fields, and generation identity. Preview must use the same
authorization and connector resolution as publication.

### 11.4 Version and publish UX

- autosave accepted commands into the current draft revision;
- checkpoints create immutable report versions;
- compare graph, DQL, schema, exposure, and test evidence;
- publish only a validated immutable version;
- display pending generation, active generation, failure, rollback, and runtime
  revision;
- failed publication leaves the prior active generation serving requests.

## 12. MCP, resources, and skills

The Exposure tab controls HTTP route metadata and named MCP tool/resource
exposure. It shows base and `WithURI` alternative routes independently.

Cube and compose MCP tools are separate opt-ins. An HTTP API-key-only policy
must not accidentally expose an MCP tool without an MCP-compatible policy.

The Resources tab manages versioned SQL/documentation files and explicit
resource folders. Skill roots require a valid `SKILL.md`, normalized paths,
unique URI prefixes, root-escape protection, and runtime resource authorization.
Developer authoring MCP tools remain distinct from published business tools.

## 13. JSX Studio SDK contract

UI components call named methods only. Operation IDs and HTTP routing remain
inside `StudioAPI`.

Proposed reader authoring surface:

```text
api.readers.list(input)
api.readers.get(reportId)
api.readers.create(input)
api.connectors.testSQL(connectorName, sql, limit)
api.versions.get(reportId, versionNo)
api.versions.inspect(reportId, versionNo)
api.versions.apply(reportId, versionNo, expectedRevision, command)
api.versions.validate(reportId, versionNo)
api.views.test(reportId, versionNo, viewId, fixture)
api.relations.test(reportId, versionNo, relationId, fixture)
api.preview.execute(reportId, versionNo, fixture)
api.cubes.test(reportId, versionNo, request)
api.compose.test(reportId, versionNo, request)
api.cache.test(reportId, versionNo, viewId, fixture)
api.warmup.plan(reportId, versionNo)
api.warmup.execute(reportId, versionNo, request)
api.publications.publish(reportId, versionNo, expectedRevision)
```

Errors use the SDK error envelope with code, message, field, and violations.
Cancellation uses `AbortSignal`. SDK DTOs are lower camel case. Secrets, raw DB
handles, connector DSNs after resolution, internal Has fields, and private causes
never cross the browser boundary.

## 14. Current Datly reader-builder mapping

| Studio action | Current Datly 1.0 operation | Status |
| --- | --- | --- |
| Inspect graph | `inspect` | Available |
| Create initial root graph | typed `createReader` | Available |
| Add input | `addField` | Available |
| Add predicate | `addFieldPredicate` | Available |
| Update predicate | `updateFieldPredicate` | Available |
| Remove predicate | `removeFieldPredicate` | Available |
| Add projection function | `addFunction` | Available |
| Update projection function | `updateFunction` | Available |
| Remove projection function | `removeFunction` | Available |
| Connector/cache/cube/compose/MCP settings | `setSetting` | Available within current validation envelope |
| Execute cube composition frames | Derived Datly cube-compose component via `versions.test_compose` | Available |
| Add joined or derived view | typed `addView` (`kind=subview|derived`) | Available |
| Update view SQL | `updateView` | Available |
| Remove related view | `removeView` | Available with dependency validation |
| Update/remove input | source-preserving `updateField` / `removeField` | Available |
| Rename input | Atomic token-aware `updateField` rename | Available |
| Update/re-parent relation metadata | AST-span `updateRelation` with compiled-parent verification | Available |
| Edit view field role metadata | Tagly-preserving `setColumnRole` | Available |
| Test isolated inner-view SQL | Studio/Datly execution facade | Available |
| Test Schema Browser SQL | bounded transient Datly reader via `connectors.test_sql` | Available |
| Return full graph descriptor | Compiled `readerbuilder.Structure`/`spec.Component` graph | Available |
| Schema/table/column discovery | Authorized connector SQLX catalog API | Available |
| Test relation assembly | Selected-edge full Datly execution with nested evidence | Available |
| Preview runtime reader | Dynamic Datly runtime contract through SDK/BFF | Available |
| Cache/warmup planning and evidence | Server-owned Datly warmup operation | Available |

Do not implement missing actions by directly mutating `spec.Component` in the
browser or by fragile text replacement in Studio. Extend the generic Datly
reader builder when the operation preserves DQL invariants and benefits other
authoring clients; keep Studio workflow/version/publication policy in Studio.

## 15. UX states and visual quality

Every window must have intentional:

- loading skeleton;
- useful empty state with primary action;
- partial/invalid source state that still exposes recoverable structure;
- field-level validation;
- permission denied state;
- connector unavailable state;
- stale-revision conflict;
- truncated test result;
- publication/build progress;
- recoverable failure with request ID and Retry.

Reader authoring treats SDK `conflict` as a dedicated recovery workflow. Studio
keeps the stale form mounted for review, offers an explicit discard-and-reload
action, and never retries a mutation against a newer revision automatically.

The graph must remain understandable without color. Use consistent node roles,
edge labels, icons, status text, contrast, focus rings, and keyboard navigation.
At narrow widths, switch from three columns to graph plus inspector drawer; the
tree remains an accessible semantic alternative to canvas-only interaction.

## 16. UX review gates

No menu, main layout, or individual authoring window is complete until it passes:

1. visual hierarchy and attractive, consistent Forge styling;
2. real SDK-backed happy, empty, loading, error, denied, and stale states;
3. keyboard and screen-reader basics;
4. desktop and narrow viewport review;
5. safe destructive confirmations and impact display;
6. source-linked diagnostics and focus navigation;
7. no backend call outside the JSX SDK;
8. UI tests and production Vite build;
9. live visual inspection with representative graph and test data.

## 17. Delivery slices

### Slice 0 — foundation

- connector catalog/test workflow;
- report catalog;
- Guardian-style shell and navigation;
- Forge metadata/data-source adapter over JSX SDK;
- development and authenticated-user modes.

### Slice 1 — first useful reader

- create reader wizard;
- root-view SQL editor;
- connector schema discovery;
- Test view;
- parameter/predicate editing;
- full preview;
- version validation.

### Slice 2 — data graph

- graph/tree synchronization;
- add/update/remove subviews;
- relation key/cardinality editor;
- relation test and impact analysis;
- derived and independent outputs.

### Slice 3 — analytical behavior

- selector policy;
- cube readiness and configuration;
- cube playground;
- compose frame/SQL playground;
- MCP exposure schema preview.

### Slice 4 — performance and operations

- cache configuration;
- warmup case planner and execution;
- cache evidence/observability;
- runtime generations, publication, rollback, and diagnostics.

## 18. First product definition: Vendor reader

The first complete authored product is a dynamic, reader-only Vendor graph.
Exact source table/fields remain connector/schema discoveries, but the proof
must include:

- one root Vendor view;
- at least one related subview;
- query/path parameters with present-zero behavior;
- required authorization predicate isolated from optional filters;
- projection/order/pagination selectors;
- isolated SQL tests for root and child;
- relation test with same IDs under different owners/tenants where applicable;
- full preview;
- cube-ready grouped variant with explicit aliases;
- optional bounded composition;
- cache and bounded warmup proof;
- HTTP and selected MCP exposure;
- immutable version validation and publication.

This one product should prove that the graph builder is a Datly reader authoring
surface—not merely a diagram editor or SQL scratchpad.
