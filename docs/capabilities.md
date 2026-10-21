# Studio capabilities

Studio authors typed Datly 1.0 components backed by DQL and Go-linked types.
The browser uses the same native component and SDK contracts as authoring MCP
clients. The catalog stores editable drafts, source revisions, publication
records, resource files, folders and component skill roots.

| Capability | Boundary |
| --- | --- |
| Connectors | Named database configurations, server-held credentials, probing and activation |
| Namespaces | Ownership, visibility and dedicated MCP listener configuration |
| Component authoring | View graphs, SQL, typed parameters, predicates, relations and output columns |
| Validation | Checks an exact version/source revision before preview or publication |
| Publication | Activates an immutable generation of selected component versions |
| Reader, cube and composition | Published contracts executed by the Datly runtime |
| Resources and skills | Versioned component resources; publication makes admitted resources discoverable |
| Permissions | Component authoring ACLs and separate `viant/authz` resource policies |
| Embedding | Reusable `app/studioapi`, `app/datlycmd`, runtime host and UI entry points |

The standard executables link MySQL, BigQuery, Oracle (`go-ora/v2`) and SQLite
(`modernc.org/sqlite`), plus PostgreSQL. Oracle and SQLite do not require CGO.
Driver registration/build checks are separate from live-database acceptance;
Oracle and BigQuery require their own reachable services and credentials for
live queries. Embedded hosts choose which drivers to link.

Independent namespace-owned skills have their own immutable revisions and native
MCP management API, separate from component-version skill roots. Authoring works
with MCP disabled; publication requires a ready namespace listener. The default
UI editor is implemented. The combined UI acceptance case creates both ownership
types and verifies native MCP listing, retrieval, resource reads and component
invocation for allowed and denied callers (66 assertions passed). The final admin
discovery catalog acceptance remains outstanding. See the
[e2e plan](../e2e/plan.md) and [results](../e2e/results.md) for tested behavior and
remaining coverage. Passing the current suite does not mean every planned
capability or database dialect has live acceptance coverage.

## Database discovery scopes

The Forge Explorer loads only the connector list when opened. **Load
schemas** explicitly calls `studio.sdk.connectors.discover`, a custom Datly
handler returning the database product/version, current catalog/schema and
available metadata choices. **Stop waiting** dismisses a pending lookup and
ignores its late result. Discovery requests have a 30-second server deadline.

**Edit lookup lists** uses Forge `arrayFields` to add/remove projects/catalogs
and schemas/datasets. Lists are also editable in connector configuration and
persist under `options.discovery.catalogs` and `options.discovery.schemas`;
no DDL is required. Editing those lists preserves activation and probe status.
They are metadata hints, not restrictions or database-creation operations.
Use **Discover entries** to merge available metadata into the editable lists.
Some drivers expose only the current catalog; other accessible catalog names
can be entered manually. BigQuery does not expose a database server version.

The existing `studio.sdk.connectors.schemas`, `.tables` and `.table` tools
accept `input.catalog`; table and column discovery also accept `input.schema`.
The selected scope overrides discovery defaults without changing stored DSNs
or credentials. BigQuery preserves connection credentials, region and options
while selecting the requested project and dataset. SQLite attachment discovery
uses only attachment paths already configured on the connector. Source access
continues to be checked by the connector and database credentials.

Explorer uses a full-page Forge SQL workbench beside connector/schema/table discovery.
**New SQL** opens an editor tab for the selected connector. Inspecting a table
opens a separate discovery tab with its SQL and columns. Every **Run SQL** opens
a result tab that keeps the captured SQL, connector, timeout and status.
SQL is collapsed behind **Show SQL**; later runs do not replace the result tab. **Create component** uses the selected
tab's SQL and connector, then opens the component editor. When the tab strip
overflows, left/right scroll buttons appear and new or selected tabs are revealed.
The dedicated `e2e/management/explorer-tabs.yaml` covers execution snapshots, table
tabs, explicit timeouts, component creation, and tab overflow.

**Timeout (seconds)** is always visible: positive values set the execution
budget, and **0 disables the Studio deadline**. There is no product-specific
upper cap. The initial value is 60 seconds; SDK requests omitting the field also
default to 60. HTTP request cancellation and upstream deadlines still apply.
Timeouts produce a specific 504 message rather than a syntax/compilation error.
The BFF does not impose its normal write deadline on SQL/MCP execution.

Result tabs use server-side offset pagination with **Previous**, **Next**, and
**Rows per page** (25, 50, 100 or 200; initially 50). The backend fetches one extra
row to identify the last page without a count query. Previously visited pages are
retained in the result tab. Each new page executes the captured query again; use
`ORDER BY` for consistent ordering, and be aware that source changes can affect
later pages. Page size does not limit the database scan or aggregation workload. Use qualified table names for the selected project
and schema; connector discovery scope does not rewrite authored SQL.
