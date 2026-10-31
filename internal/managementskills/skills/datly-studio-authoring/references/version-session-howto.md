# How to change and verify a component version

This session pattern uses existing management tools. Read their live schemas;
IDs, revisions and publication generations come from responses, never examples.
For complete payloads and DQL, follow the [worked cube session](worked-example.md).

1. Discover the namespace with `studio.sdk.namespaces.get` and inspect its current
   runtime. Use the intended namespace owner. When both resource management and
   SDK management are used, establish ownership through the supported namespace
   workflow before creating components; do not repair a mismatch with broad grants.
2. Inspect the existing component/version. Create a draft only when needed using
   `studio.sdk.versions.create`. Preserve the returned component package identity.
3. Apply DQL with `studio.sdk.versions.apply`, command `set_dql`, and the exact
   `expectedSourceRevision`. On conflict, read and reconcile; do not blindly retry.
4. Inspect with `studio.sdk.versions.inspect` and `discoverColumns: true`. Verify
   aliases, nullable types, tags, dimensions/measures and source grain. Catalogue
   table metadata alone cannot prove the projection's shape.
5. Validate and release through the tools advertised by this server. Re-read the
   source revision before release; changing settings can invalidate validation.
6. Verify the active version/generation and the authenticated runtime endpoint.
   For an internal reader, verify through its authorized public consumer rather
   than enabling direct MCP exposure just to satisfy a discovery client.
7. Give the consumer the exact route, component/version, schema, mandatory scope,
   optional filters, sorting/paging and observed output. Keep an actual response
   separate from independently authored expected values.

## Example verification scenarios

- **Date column change:** typed date and raw datetime serialize differently as
  declared; null remains null; changing format does not change aggregation.
- **Reusable cube:** two dimension projections reuse one source; additive totals
  reconcile; distinct counts, rates and snapshots are not summed blindly.
- **Dictionary relation:** missing names remain nullable and do not remove facts;
  declared cardinality prevents multiplying totals.
- **Optional multi-filter:** selected members narrow data; omission removes only
  the optional predicate; required authorization remains active.
- **New revision:** an existing consumer stays pinned until explicitly updated;
  the new consumer reads the new generation without borrowing old receipts.

## Component resource-check option

Preserve `resourceCheckEnabled` during unrelated edits. Configure thresholds on
the namespace; enable the check only for the requested component draft using
`versions.apply`, command `set_resource_check`. This changes its source revision
and clears validation. Read [security and resource checks](security-and-resource-checks.md)
for complete payloads, sparse group updates, security package selection and the
boundary between dry-run estimates and actual returned rows.
