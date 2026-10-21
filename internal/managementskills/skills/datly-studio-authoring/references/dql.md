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

#define($_ = $AccountID<int>(query/accountId).WithPredicate(0, 'equal', 'o', 'account_id'))
#define($_ = $Rows<[]*OrderRow>(output/view))

SELECT orders.*, type(orders, 'OrderRow'), set_limit(orders, 50)
FROM (
    SELECT o.order_id, o.account_id, o.order_date, o.amount
    FROM orders o
    WHERE ${predicate.Builder().CombineOr($predicate.FilterGroup(0, "AND")).Build("AND")}
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

## Bindings and contracts

Use #define/#set typed declarations with a source such as query, body, header,
path, component, or output/view. Keep requiredness explicit and preserve pointer
presence where omitted, NULL and zero differ. Declare imports for non-builtin
Go types. Use QuerySelector on the exact view for supported limit/offset/fields
bindings; enabling a UI control does not grant selector authority automatically.

Use outer `CAST(view.column AS GoType)` for an actual type override. SQL CAST
inside the subquery remains database SQL. Use `tag`/format policy for deliberate
JSON naming and visibility; do not replace all fields with redundant tags.
Keep computed aliases explicit and discover their names/types from SQL results.

Use `use_connector(alias, 'configured_name')` on related views, `type` for their
row/holder identity, and complete ON keys for relationships. A Datly outer join
can describe a batched related read on another connector; it is not permission
to place MySQL identifiers inside BigQuery SQL. Preserve to-one/to-many semantics.

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
