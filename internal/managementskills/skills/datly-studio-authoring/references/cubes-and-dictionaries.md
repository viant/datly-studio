# Fact-table cubes and outer metadata views

All identifiers below are synthetic. Discover the actual schema, connectors and
authorization boundary before adapting the pattern. No database engine or
business domain is assumed.

## Start with the fact grain

Identify the fact table, dimensions, additive measures, date/partition boundary
and entity scope. Aggregate facts once at the intended selected dimension grain.
Keep stable IDs in that result so outer metadata views can enrich them through
complete native relation keys. Do not join another connector's tables into the
fact SQL or multiply facts by expanding a many-row dictionary before aggregation.

Example input declarations and inner SQL fragment, not a complete component:

```sql
#define($_ = $Dates<[]string>(query/dates).Required().WithPredicate(0, 'in', 'f', 'event_date'))
#define($_ = $ProductIDs<[]int>(query/productIds).Optional().WithPredicate(0, 'in', 'f', 'product_id'))

SELECT f.event_date, f.product_id, f.country_code,
       SUM(f.units) AS units, SUM(f.net_amount) AS net_amount
FROM fact_measurements f
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
GROUP BY f.event_date, f.product_id, f.country_code
```

Use discovered source types; the example date representation is not a universal
database conversion rule. Caller filters are not authorization. Add any required
namespace authorization input and AND-connect its predicate group to the entire
filter expression. Main-query filtering belongs in typed input predicates and
`predicate.Builder`, rather than ad-hoc WHERE string construction. Request native
SQL metadata before running bounded data checks.

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
Cube exposure normally adds a POST `*.readCube` sibling to the reader's `*.read`
tool. Filter names use the derived lower-camel contract (for example `Dates`
becomes `filters.dates`). Still discover the exact published name and input
schema through `catalog.tools.list`; deployment prefixes and authored layouts
can differ. Use the inspected source contract to explain the meaning of filters.

Do not sum non-additive values such as distinct users, percentages or daily rates
across incompatible groups. Aggregate additive numerators/denominators and derive
the final ratio at the selected grain, or use a documented source-native rule.
Selecting fewer dimensions must not change required contributor authorization.

## Expand IDs after aggregation

A declared Datly outer metadata view can use its own connector and keys
from the grouped fact parent. For example, the parent
retains product_id and an optional related Product object resolves its label.
Use the connected builder's Add child / relation operation, select the metadata
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

## Scope fact warmup separately from scalar prerequisites

Component-level cache warmup is inherited by independent input-view reads.
A scalar prerequisite returning `range_start` and `range_end` must not inherit a
fact warmup projecting `tenant_id` and `amount`: native binding can fail because
those fact columns do not exist in the scalar view. Changing only the scalar's
cache name does not clear inherited warmup.

Use existing view-local bindings when the producer has a named warmup profile:

```sql
SELECT fact.*, use_cache(fact, 'result-cache'),
       cache_warmup(fact, 'fact-projection')
FROM (${embed:sql/fact.sql}) fact
```

The producer's existing `Caches` configuration must define `result-cache` and
the profile, for example `"fact-projection": {"Warmup": {"FieldNames":
["tenant_id", "amount"]}}`. Remove the component-level warmup only after its
complete projection, parameters, cases and limits have moved to that declared
profile. Keep the fact backend and authorization unchanged. With no global cache
default, unrelated scalar inputs remain uncached while the fact uses its explicit
backend and warmup. This is a fragment, not a complete component/configuration.

A source bundle does not install producer cache profiles. Declare that dependency
in deployment configuration and verify it in both validation and execution.
Test scalar binding, fact warmup and cached fact replay on invented local data;
a successful scalar query alone does not prove caching was preserved. Do not add
fake columns to scalar views or disable fact caching to conceal the mismatch.

## Verify

Compare fact totals before/after enrichment for representative grouped frames.
Test full-key and reduced-key selections, missing and duplicate dictionary rows,
cross-account denial, empty results and computed aliases. Use the advertised
cube input schema; do not infer filter property spellings from a label. Prefer
bounded read-only checks, then validate the exact saved revision and follow the
namespace readiness → Release → authenticated tools/list workflow.
