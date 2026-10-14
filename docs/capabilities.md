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
