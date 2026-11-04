# Parameterized grouping and source calculations

Validate the query that the native reader actually emits, including parameter
identities and types. Replacing parameters with literals can prove a different
query. Keep source validation, parameterized database validation and returned-row
reconciliation as separate evidence.

## Group the selected expression once

For an already declared typed input `Dimension`, an inner BigQuery query can
select a grouping expression once:

```sql
SELECT CASE WHEN $Dimension = 'region'
            THEN r.region ELSE r.category END AS bucket,
       SUM(r.quantity) AS quantity
FROM observations r
-- Retain the reader's existing trusted scope and optional predicates here.
GROUP BY 1
```

This is a SQL fragment, not a complete reader or an authorization declaration.
Use the discovered connector/table and retain its verified scope predicates.
An explicit selected alias is another option where the connected dialect admits
it. Use only syntax that passes native Datly discovery and the actual database.

Repeating the CASE in GROUP BY can produce separate bound parameter identities
for the two occurrences. A database may then reject the projection even if both
parameters happen to have the same value. Grouping by the selected ordinal or
alias avoids repeating that expression without adding a wrapper, extra reader,
string interpolation or client-side grouping. If the SELECT list changes,
recheck every ordinal's meaning.

Verify the actual emitted SQL and typed bindings through the selected native
component. For BigQuery, a dry run validates the query without reading result
rows; it does not prove values or permissions. A SQLite fixture is useful for
independent expectations but does not prove BigQuery grouping rules. Keep real
fact extracts out of reproducible examples; use distinguishing invented rows.

## Preserve aggregation order

`SUM(CASE WHEN adjustment < 0 THEN adjustment END)` preserves negative
contributions inside a group containing both signs. Testing the sign of
`SUM(adjustment)` afterward does not express the same operation. Conversely,
clamping an aggregate difference after grouping must not become a sum of
per-row clamps.

Keep duplicate rows when the source deliberately sums them. Lack of a physical
unique key alone is not a reason to deduplicate or block a faithful aggregate.
Test duplicates, mixed signs, all-NULL and mixed-NULL groups, zero denominators,
and every parameter-selected grouping branch. Preserve source ranks and
already-computed averages/ratios at their admitted grain; do not expose them as
freely summable measures to obtain a scalar result.
