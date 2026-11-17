# Datly v1 DQL essentials

This guide uses synthetic identifiers. Read dql.ebnf.txt for structural grammar and
validate with the connected Datly v1 compiler for exact supported semantics.
The bundled EBNF comes from the Datly version pinned by this Studio build; it is
not an executable validator or an exhaustive SQL dialect grammar.

## Complete reader shape

```sql
#package('example.com/analytics/orders/read')

#setting($_ = $connector('orders_db'))
#setting($_ = $route('/analytics/orders', 'GET'))
#setting($_ = $case_format('lc'))

#define($_ = $AccountID<int>(query/accountId).Required().WithPredicate(0, 'equal', 'o', 'account_id'))
#define($_ = $Rows<[]*OrderRow>(output/view))
#define($_ = $Meta<?>(output/summary) /*
SELECT COUNT(*) AS row_count FROM ($View.NonWindowSQL) AS filtered
*/)

SELECT orders.*, type(orders, 'OrderRow'), set_limit(orders, 50)
FROM (
    SELECT o.order_id, o.account_id, o.order_date, o.amount
    FROM orders o
    ${predicate.Builder().CombineOr($predicate.FilterGroup(0, "AND")).Build("WHERE")}
) orders
```

This example declares a required account filter, not trusted authorization.
Bind real authorization from a verified JWT or declared server-owned component
before using it as an access boundary; never trust a client-supplied account ID
as proof of access. Required scope must remain effective when optional groups
are omitted or combined with OR.

The outer query defines Datly views, row types, relations and controls. Inner SQL
belongs to its connector's SQL dialect. `orders` is a view identity; `o` is only a
SQL alias. The typed output/view holder binds the root rows; naming an output
type alone is not sufficient. Preserve the package/module and named row identity.

## Paging metadata from the native summary

Inside an output/summary or derived output, `$View.Limit`, `$View.Offset` and
`$View.Page` refer to the parent view's effective selector window. `$View.Limit`
includes the declared default when the caller omits the limit. Use these scoped
values instead of an optional raw request input when computing page metadata.
`$View.NonWindowSQL` retains the parent authorization and filters before paging.

For a paged MySQL reader with a positive effective limit:

```sql
#define($_ = $Meta<?>(output/summary) /*
SELECT COUNT(*) AS record_count,
       CEIL(COUNT(*) * 1.0 / $View.Limit) AS page_count,
       $View.Limit AS page_size
FROM ($View.NonWindowSQL) AS filtered
*/)
```

Preserve any existing summary fields when extending a reader. Verify omitted
limit, an explicit nondefault limit, adjacent pages, and an empty population.
An empty filtered population has zero records and zero pages. Use the connector's
supported numeric conversion if the public page-count field requires an integer.

## Deployment-owned summary caches

A native summary can select a named cache already supplied by the host. The
`use_cache` argument names the summary SQL alias, followed by the deployment-owned
cache name. This declaration caches the summary, not the parent rows:

```sql
#define($_ = $Meta<?>(output/summary) /*
SELECT COUNT(*) AS record_count,
       CEIL(COUNT(*) * 1.0 / $View.Limit) AS page_count,
       use_cache(filtered, 'countCache')
FROM ($View.NonWindowSQL) AS filtered
*/)
```

`countCache` is illustrative. Use a verified configured cache name; the directive
does not create a provider or choose its location/TTL. Preserve the authorized
filtered parent SQL and its bound values. Test page/projection reuse, changed
filters, distinct authorized scopes, expiry and invalidation through the actual
reader. Page size affects a cached page count and must remain part of its cache
identity. Root rows should remain uncached when only counts require reuse.

A duration or an unchanged result does not prove a cache hit. Native execution
metrics can report cache writes and hits when the reader exposes those metrics.
If the connected preview response supplies only duration/row evidence, report
cache-hit qualification as pending rather than inferring it or adding a wrapper.

## Required predicate-builder filtering

For authored readers, construct filtering WHERE clauses with `predicate.Builder`.
Inputs declare the predicate; SQL expands it. Never use `WHERE $criteria.In(0)`
as a group expansion or interpolate caller values into SQL text.

For example, add an optional name filter to the complete reader above:

```sql
#define($_ = $Name<*string>(query/name).Optional().WithPredicate(0, 'contains', 'o', 'name'))
```

`AccountID` is required, `Name` is optional, and both belong to group 0. The
existing `FilterGroup(0, "AND")` combines present predicates with AND. Omission
of Name removes only that filter. The predicate implementation owns binding and
wildcard construction; do not add `%` in a second layer.

When a search needs alternatives, put them in a separate group and combine
groups with AND so optional search cannot bypass required scope:

```sql
#define($_ = $Search<*string>(query/search).Optional().WithPredicate(1, 'contains', 'o', 'name').WithPredicate(1, 'contains', 'o', 'description'))
```

```sql
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND"), $predicate.FilterGroup(1, "OR")).Build("WHERE")}
```

Only use predicate names/arguments advertised by the connected native catalog.
For a correction, change the saved draft using its current source revision,
validate the new revision, and surface diagnostics before attempting publication.

## Bindings and contracts

Use #define/#set typed declarations with a source such as query, body, header,
path, component, or output/view. Keep requiredness explicit and preserve pointer
presence where omitted, NULL and zero differ. Declare imports for non-builtin
Go types. Use QuerySelector on the exact view for supported limit/offset/fields
bindings; enabling a UI control does not grant selector authority automatically.

### Public MCP paging and field selection

Query selectors are explicit opt-in: no caller-controlled fields, limit, offset,
page, ordering or criteria should be assumed enabled by default. Author only the
selector capabilities requested for the specific prepared view. Do not add all
selectors to every reader. An absent selector is not a framework defect.

Reader execution defaults and public selector bindings are separate. A saved
`set_limit` or `selector_fields` annotation alone does not prove that the MCP
schema exposes a caller argument. Declare typed selector inputs and inspect the
actual published tool schema. A server-side row limit does not opt the caller
into changing that limit. The following fragment opts in to four capabilities
for a registered prepared view named `Records`; omit any not requested:

```sql
#define($_ = $Fields<[]string>(query/fields).Optional().QuerySelector('Records', 'fields'))
#define($_ = $Limit<int>(query/limit).Optional().QuerySelector('Records', 'limit'))
#define($_ = $Offset<int>(query/offset).Optional().QuerySelector('Records', 'offset'))
#define($_ = $OrderBy<string>(query/orderBy).Optional().QuerySelector('Records', 'orderBy'))
```

This is a declaration fragment: replace `Records` with the prepared view identity
returned by inspection. An outer SQL alias is not automatically a selector name.
Retain the view's allowed fields/order mappings and default limit. Discover the
wire property names from the generated MCP schema; do not assume a silently
ignored argument was applied. Verify a smaller page, the next offset, a narrow
field selection and the unchanged authorized total through real tool calls.

Declare date input types from the discovered source contract and the connector's
advertised binding behavior. Use native typed input predicates for the date
bounds and the normal predicate builder to emit them. Output date formatting
is a separate concern; see [Column output how-to](column-output-howto.md).
Do not assume a database-specific CAST or placeholder convention is portable.

For a linked variadic custom predicate, builder operations use `name: handler`,
the handler type identity in `args`, and literal configuration in `handlerArgs`.
The latter preserves string, boolean, integer and NULL values; it does not accept
expressions, nested lists or objects. Use only configuration pairs supported by
that installed handler (for example an advertised SQL alias attribute), with
the actual source alias. On update, omitting `handlerArgs` preserves the existing
configuration; `[]` clears it. Built-in predicate `args` remain strings, and
legacy non-variadic handlers do not gain argument support automatically.

Use outer `CAST(view.column AS GoType)` for an actual type override. SQL CAST
inside the subquery remains database SQL. Use `tag`/format policy for deliberate
JSON naming and visibility; do not replace all fields with redundant tags.
Keep computed aliases explicit and discover their names/types from SQL results.
For parameter-selected grouping expressions and distinguishing native query
validation from literal-only checks, read the
[parameterized grouping how-to](parameterized-grouping-howto.md).

Use `use_connector(alias, 'configured_name')` on related views, `type` for their
row/holder identity, and complete ON keys for relationships. A Datly outer join
can describe a batched related read on another connector; it is not permission
to place one connector’s identifiers inside another connector’s SQL. Preserve to-one/to-many semantics.

For external SQL use `${embed:sql/file.sql}` and save that file in the component's
versioned resources. Keep the resource tree, imports, declared types and connector
bindings available together. Do not inline or rewrite large existing SQL simply
to make the editor accept it.

## Compile and persist

Use Studio SDK authoring to create and apply the draft, then request
versions.inspect(discoverColumns=true). The native column refiner evaluates the
SQL discovery copy, adds a false predicate and reads driver ColumnTypes. It does
not infer a result contract from the physical table catalog. Inspect source
revisions before and after every edit. Run validation against the selected build;
static parse success alone does not prove database, type or resource resolution.

For Go-backed readers, generate using Datly v1 operation-based transcribe get.
Use separate writer DQL and transcribe post/patch/put for persistence mutations;
keep business checks in lifecycle hooks. Custom components own genuine external
orchestration. Do not create raw database persistence wrappers as a substitute.

## Return filtered totals with paged rows

In the complete reader above, Rows holds the current page and Meta holds the
summary over the filtered order rows.

With two matching records and a page limit of one, expect one record in Rows
and a count of two in Meta. Check a filter matching no records as well. The
parent SQL supplies its predicates, including authorization. An explicit LIMIT
inside the authored root SQL still bounds what the summary can count.

A calculation that determines which facts may enter the root query has a
different execution order. Declare that calculation as a private input view
and bind its value in the fact predicate. For example, an eligibility rule
over all authorized entities must run before the selected-entity filter.
