# Complete cube authoring example

All database identifiers below are synthetic. Substitute the user's discovered
sources and requested dimensions/measures; never execute these example tables.
Argument objects below are the arguments to the named management tool, not a
JSON-RPC envelope. Include the chosen `namespaceId` on version/component calls.
Read live schemas; use values returned by the server for every placeholder.

## 1. Resolve scope and create

`studio.sdk.namespaces.get` takes `{"name":"analytics"}`: the namespace **name**,
not its canonical ID. Save the returned `namespaceId`. Prefer
`connectors.list` with `fields:["name","driver","status","ownerId"]` to avoid
large discovered lookup lists; `connectors.get` returns the full configuration.
Use the active BigQuery and MySQL connectors the user authorized.

`studio.sdk.components.create`:

```json
{"namespaceId":"<namespaceId>","namespace":"analytics","slug":"sales-by-product","title":"Sales by product","defaultConnectorName":"warehouse","description":"BigQuery facts with MySQL product and account labels"}
```

The slug is required, max 200 characters, lowercase letters/digits/dashes only,
with no leading/trailing dash. Use `sales-by-product`, not `sales_by_product`.
The server generates `id`, `ownerId`, `componentScope`, and `componentName`.
Do not copy these from another component. The package for this component is
`<returned componentScope>/<returned componentName>`; preserve it in DQL.
Use an owner-prefixed MCP tool identity from `ownerPackage` plus the slug with
dashes replaced by underscores, e.g. `owner.sales_by_product.read`. Check for
collisions before saving; do not take the existing Test component's identity.

Create a draft with `studio.sdk.versions.create`:

```json
{"namespaceId":"<namespaceId>","reportId":"<id>","input":{"authoringMode":"dql","notes":"Sales cube with two dictionary lookups"}}
```

Save `versionNo` and `sourceRevision` from the response. A source revision changes
on every DQL, builder, resource, or skill edit. Never hardcode revision 1 later.

## 2. Apply a complete inline DQL graph

Start inline for this small example so SQL resources need not exist before the
first compilation. In `studio.sdk.versions.apply`, send:

```json
{"namespaceId":"<namespaceId>","reportId":"<id>","versionNo":1,"command":{"kind":"set_dql","expectedSourceRevision":1,"payload":{"authoredDql":"<the complete DQL below>"}}}
```

Replace the example version/revision with the returned current values. `kind` is
`set_dql`, not `dql` or `replace_dql`; payload is an object with `authoredDql`.
`set_sql` takes `{"authoredSql":"..."}` and `set_spec`/`replace_spec` take a
component spec. Do not use those modes for this DQL graph.

```sql
#package('example.com/app/dynamic/owner/generated_component/reader')

#setting($_ = $connector('warehouse'))
#setting($_ = $route('/analytics/sales-by-product', 'GET'))
#setting($_ = $case_format('lc'))
#setting($_ = $mcp('owner.sales_by_product.read', 'Daily sales with optional dictionary names'))
#setting($_ = $cube('', '', '', '', '', '', '', true))

#define($_ = $Dates<[]string>(query/dates).Required().WithPredicate(0, 'in', 's', 'event_date'))
#define($_ = $AccountIDs<[]int>(query/accountIds).Required().WithPredicate(0, 'in', 's', 'account_id'))
#define($_ = $ProductIDs<[]int>(query/productIds).Optional().WithPredicate(0, 'in', 's', 'product_id'))
#define($_ = $Rows<[]*SalesRow>(output/view))

SELECT fact.*, type(fact, 'SalesRow'),
       tag(fact.units, 'groupable:"false"'),
       tag(fact.net_amount, 'groupable:"false"'),
       product.*, type(product, 'Product'),
       account.*, type(account, 'Account'),
       use_connector(fact, 'warehouse'),
       use_connector(product, 'dictionary'),
       use_connector(account, 'dictionary'),
       cardinality(product, 'One'), cardinality(account, 'One'),
       allow_nulls(product), allow_nulls(account),
       grouping_enabled(fact), set_limit(fact, 50),
       selector_fields(fact, true), selector_limit(fact, true),
       selector_offset(fact, true), selector_order_by(fact, true)
FROM (
    SELECT s.event_date, s.account_id, s.product_id,
           SUM(s.units) AS units, SUM(s.net_amount) AS net_amount
    FROM `example_project.analytics.daily_sales` s
    ${predicate.Builder().CombineOr($predicate.FilterGroup(0, "AND")).Build("WHERE")}
    GROUP BY s.event_date, s.account_id, s.product_id
) fact
LEFT JOIN (
    SELECT product_id, product_name FROM product_dictionary
) product ON product.product_id = fact.product_id
LEFT JOIN (
    SELECT account_id, account_name FROM account_dictionary
) account ON account.account_id = fact.account_id
```

The package, route, tool identity and connectors MUST be replaced with the new
component's server-owned identity and available connector names before applying.
Use discovered actual date/ID types and the driver's native conversions; the
example date strings are not a claim that every driver/date predicate is identical.
Use `.Required()` explicitly for Dates and AccountIDs; ProductIDs is optional.
Omitting `.Optional()` does not prove a collection or pointer is required. Verify
the saved parameter requiredness returned by versions.inspect before release.
Required client inputs bound scan scope, not verified row-level authorization.
Keep genuine tenant/subject authorization separate when the source requires it.

`Build("WHERE")` supplies the WHERE keyword and emits no clause for an empty
predicate group. If SQL already has a real WHERE condition, append
`Build("AND")` instead. Never write `WHERE ${...Build("AND")}`.

`$cube` arguments are: linked input type, dimensions holder, measures holder,
filters holder, orderBy holder, limit holder, offset holder, MCP tool flag. Seven
empty strings select generated holders; the eighth `true` enables the cube tool.
`$cube()` enables the cube with defaults; it does not explicitly opt into its MCP
exposure. `$mcp(name, description, descriptionPath)` exposes the source reader.
Use `groupable:"false"` for additive outputs and root `grouping_enabled`.
Cube composition is separate and unnecessary for this single query.

A to-one dictionary must have a unique full key. Preserve the fact ID and use a
left relation so missing names do not remove facts. Verify cube selections retain
the full dictionary key before enabling enrichment. Do not force an omitted key
into grouping or return an arbitrary label for a rolled-up group.

For larger SQL resources: create the draft first, upload `sql/fact.sql` via
`resources.upsert_file` with the live tool's input shape, re-read the revision,
then apply the DQL referencing `${embed:sql/fact.sql}`. Do not attempt native
column discovery or validation until every referenced SQL file exists.

## 3. Inspect, test, validate, release

`studio.sdk.versions.inspect`:

```json
{"namespaceId":"<namespaceId>","reportId":"<id>","versionNo":1,"discoverColumns":true}
```

Confirm root aliases/types, both per-view dictionary connectors, complete relation
keys, dimensions/measures, required predicates and generated cube filters. Use
`versions.builder` for typed edits; its `command.operation` schema is published in
[builder-operations.schema.json](builder-operations.schema.json). Operation `type`
is a case-sensitive enum. Match its payload field in `x-operation-contracts`;
`setSetting` uses `setting:{name,args}`, and `batch` uses `operations:[...]`.
Setting/function args are DQL expressions: JSON `"'warehouse'"` represents a DQL
quoted string; JSON `"true"` represents a boolean expression. Do not guess fields.

Bound preview using the discovered input names. `preview.execute` supports
`input:{cube:true,input:{dimensions:{...},measures:{...},filters:{...},limit:10},limit:10}`.
Use live cube filter names and required source inputs. A successful plain reader
preview alone does not certify the derived cube. Compare full-key versus reduced
key grouping and facts before/after dictionary attachment.

Validate the latest revision with `studio.sdk.versions.validate`:

```json
{"namespaceId":"<namespaceId>","reportId":"<id>","versionNo":1,"expectedSourceRevision":3}
```

Require `valid:true` and no error diagnostics. If source changes afterward,
revalidate. Obtain namespace listener readiness and URL with `studio.sdk.runtime.status`
and the selected `namespaceId`; `host.mcpUrl` is runtime evidence, whereas
`namespaces.get.mcpPort:0` merely means the configured shared listener is used.

`studio.sdk.publications.publish`:

```json
{"namespaceId":"<namespaceId>","reportId":"<id>","versionNo":1,"input":{"requestedBy":"<verified subject>","reason":"Verified sales cube","expectedSourceRevision":3}}
```

Read the live schema for the publish wrapper; substitute the latest validated
revision. Poll `publications.get` and `runtime.status` to confirm an active
publication/generation. Releasing failed/pending validation is not completion.
Finally use `studio.sdk.catalog.tools.list` with `namespaceId` and optional
`query` (a tool-name substring) to retrieve the current tools and full input
schemas. Follow `nextCursor` pages. This reflects the live namespace even when
the client connected before publication. Invoke the exact returned name through
`studio.sdk.catalog.tools.call` with `namespaceId`, `name`, `arguments`, and an
explicit `timeoutSeconds` (default 60; 0 disables the Studio deadline). Check
`result.isError` and the nested MCP content/structuredContent. Both operations
forward only the verified caller credential; they do not grant execute permission.
No namespace-client reconfiguration or reconnect is needed for this path.

A separately configured business MCP client can also connect to the returned
namespace endpoint with the existing
verified credential, list tools and find the exact reader/cube names. Execute one
bounded cube call with explicit required date/account filters, limit 10, and the
requested dimensions/measures. Report actual rows, missing labels and errors.
Never claim an initialize handshake or a management listing proves business MCP
execution. Inspecting/explaining the original component must leave it unchanged.
