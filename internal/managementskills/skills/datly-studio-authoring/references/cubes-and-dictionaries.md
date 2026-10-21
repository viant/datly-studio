# BigQuery fact cubes and MySQL dictionary expansion

The design below reflects patterns reviewed in Steward forecasting DQL, with no
copied queries, table names, custom handlers, credentials, or business rules.
All identifiers in the examples are synthetic. Discover the actual schema,
connectors and authorization boundary before adapting the pattern.

## Start with the fact grain

Identify the real event/fact table, additive measures, date/partition boundary,
and authorized entity scope. Keep database-native IDs in the fact output.
Aggregate the fact once at the intended selected dimension grain. Do not add a
MySQL metadata join inside BigQuery SQL or pre-expand a many-row dictionary into
the fact; either can multiply contributions or change totals.

Example inner fact SQL, not a complete DQL component:

```sql
SELECT event_date, product_id, country_code,
       SUM(units) AS units, SUM(net_amount) AS net_amount
FROM `example_project.analytics.daily_sales`
WHERE event_date >= @from_date AND event_date < @to_date
GROUP BY event_date, product_id, country_code
```

Replace @from_date/@to_date with the target driver's declared Datly parameter
bindings; they are explanatory placeholders, not portable DQL syntax. Add the
real verified scope predicate. Do not run an unbounded aggregation to discover
its output columns: request native SQL metadata first.

## Enable the cube contract

Use Component settings / the advertised versions.builder settings operation to
enable Cube. Mark grouping keys as Dimensions (`groupable:true`) and additive
outputs as Measures (`groupable:false`). Configure every field deliberately;
unknown roles must not silently become dimensions. Keep grouping_enabled on the
root view. Inspect the resulting canonical DQL/settings rather than guessing the
argument order of a cube directive or injecting an unregistered report facade.

Enable cube MCP exposure when an agent must invoke the cube, and component MCP
exposure when the source reader should also be callable. Composition is optional;
enable it only for multiple typed query frames, with explicit limits and timeout.
The derived cube contract preserves required source authorization and filters.

Do not sum non-additive values such as distinct users, percentages or daily rates
across incompatible groups. Aggregate additive numerators/denominators and derive
the final ratio at the selected grain, or use a documented source-native rule.
Selecting fewer dimensions must not change required contributor authorization.

## Expand IDs after aggregation

A declared Datly outer relation can read a MySQL dictionary through its own
connector, using keys from the grouped BigQuery parent. For example, the parent
retains product_id and an optional related Product object resolves its label.
Use the connected builder's Add child / relation operation, select the MySQL
connector, and declare the complete key. Native batching owns the lookup reads.

Illustrative outer-graph fragment (not a complete component):

```sql
SELECT fact.*, product.*,
       use_connector(fact, 'warehouse'),
       use_connector(product, 'dictionary'),
       grouping_enabled(fact),
       cardinality(product, 'One')
FROM (${embed:sql/fact.sql}) fact
LEFT JOIN (
    SELECT product_id, product_name FROM product_dictionary
) product ON product.product_id = fact.product_id
```

Combine this fragment with the package, route, typed output holder and explicit
row types from dql.md before compiling. Use the actual installed grammar for
holder names and annotations. Missing dictionary rows should preserve the fact
unless the requested semantics explicitly require exclusion. A to-one lookup
must be unique by its full key; duplicate matches are an error to resolve, not
something to hide by taking the first row.

For a dictionary keyed by country_code AND region_code, retain and relate both
keys. Enrich only when the selected cube grouping contains the full key. Grouping
by country alone must not force region into GROUP BY, split the aggregate, or
attach an arbitrary region. Prefer an appropriate country-only lookup or omit
the region relation for that selection. Scope tenant/account keys where the
real schema requires them; IDs from another account are not interchangeable.

Keep dictionary entity labels separate from source-native analytical dimensions.
Do not claim a metadata table is authoritative for ownership, contributor lineage
or a missing business metric. Preserve genuine dictionary absence as unknown.

## Verify

Compare fact totals before/after enrichment for representative grouped frames.
Test full-key and reduced-key selections, missing and duplicate dictionary rows,
cross-account denial, empty results and computed aliases. Use the advertised
cube input schema; do not infer filter property spellings from a label. Prefer
bounded read-only checks, then validate the exact saved revision and follow the
namespace readiness → Release → authenticated tools/list workflow.
