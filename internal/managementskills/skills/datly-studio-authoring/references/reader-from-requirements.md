# Create a reader from requirements

The caller may describe a result rather than provide SQL or DQL. Use the connected
authoring tools and skills to implement that behavior. The caller's description
is the acceptance contract, not evidence that particular tables, predicates or
fields exist. Keep each unresolved fact explicit.

## Establish the contract before creating a component

Record the result grain, requested fields and meanings, relationships and
cardinalities, filters and presence semantics, date bounds, aggregation rules,
projection, sorting, paging, authorization, cache behavior and expected failures.
Resolve the target namespace and discover its allowed connectors and predicate
types. Read the advertised predicate descriptions and trusted input requirements.
Do not infer an authorization predicate, relationship, connector or cache policy
from a namespace, table or column name.

Inspect only the relevant source schemas. Use declared keys and relationship
metadata or an authoritative source contract to establish joins. A similarly
named column is not a proven relationship. If necessary metadata is unavailable,
report the exact missing fact; do not scan every schema or invent a join.

Propose the actual source tables, connector boundaries, grain, relationship keys,
field meanings and security binding using the discovered evidence. Use a native
typed graph for related data, retaining complete keys and cardinalities. Keep
independent grains separate when joining would multiply measurements.

## Example: an equipment lookup

Request: “Provide one row per equipment item, with its identifier, label and
location label. Support optional label search, location selection, selectable
fields, sorting and pages of 20. Only return equipment the caller may read.”

Discover the equipment and location sources and their actual key relationship.
Confirm whether equipment with no location remains in the result. A nullable
foreign key does not establish whether unmatched rows must be retained or
excluded. Preserve the
authoritative membership contract; if it is unspecified, identify that gap before
choosing a relationship that changes the result population. Likewise, a newly
available authorization handler may impose additional eligibility conditions;
compare those conditions rather than treating all authorization handlers as
interchangeable.

Discover the installed authorization contract and bind its trusted context independently of
the caller's optional location selection. Document the actual native search and
null semantics. Do not assume that a parameter called tenantId proves authority.
Return a proposal with discovered names before creating the managed reader.

The agent authors source using the connected Datly contract. Every filtering
WHERE clause uses predicate.Builder and declared typed predicates. Required
authorization stays outside optional OR alternatives. Table aliases and columns
come from explicit authored mappings supported by the discovered handler;
never accept SQL identifiers or authorization context from public caller fields.

## Caches and cubes

Discover existing cache configuration and handler capabilities before preserving
or changing cache behavior. Distinguish verified authority snapshots, query/result
caches, indexed cache identity and trusted population/warmup invocations. An
identity-rendering context is not permission to return unfiltered rows. Do not
add a cache bypass input or remove a security predicate to obtain a cache hit.

For a cube, state dimensions, measures, units, aggregation and fact grain. Retain
all dimensions required by indexed cached projections and grouping. Preserve
scope and date semantics when joining dictionaries across connectors. Do not
convert a detailed lookup into a cube merely because it is exposed through MCP.

## Resolve dimension names through native outer views

Customer-facing results should display a verified dimension name or label when
available, rather than only its technical ID. When the fact source contains IDs
only, discover the corresponding dictionary source and add it as a native outer
data-graph view. The dictionary may use a different connector from the fact.
For example, a warehouse fact with `product_id` can resolve `product_name` from
a separate catalog connector. This is a declared cross-connector graph relation,
not a SQL join sent across unrelated database engines.

Default to resolving names for the dimensions exposed by the requested result.
Inspect only the relevant dictionaries. Verify the complete relation key,
uniqueness, cardinality, source grain and applicable authorization; matching
column names alone do not establish a relationship. Do not guess dictionary
tables, load every dimension table, or add an unverified lookup. If a required
key or dictionary is unavailable, identify the specific gap.

Keep the fact as the root. Declare the dictionary connector with `use_connector`
and use complete native relation keys, the `AND 1=1` to-one marker where
uniqueness is proven, and nullable outer
relation semantics so absent metadata does not drop or multiply fact rows. Use
the framework's batched key lookups; do not issue a lookup per result row or join
and filter dictionary rows in application code. Fetch only metadata for the
selected authorized fact keys, preserving the dictionary's own access rules.

This ordinary reader/cube enrichment requires no StructQL. Return the name as
preferred display metadata and keep its stable ID available
for filtering, joins and traceability. Consumers use that ID when the name is
missing; do not invent labels or present an unknown name as an empty successful
lookup. Include relevant verified descriptive attributes when requested.

Enrichment must follow cube grain. When a rollup omits a dimension's full key,
do not force that key into grouping to obtain a name, and do not attach an
arbitrary label to the aggregate. Omit that dimension's enrichment at that grain.
Distinguish nullable name metadata plus a stable ID from an explicitly computed
display field. If the caller requires the latter, verify a supported output
contract that runs after relation attachment and respects selected dimension
keys. A dictionary-side fallback cannot handle an absent dictionary row. Do not
assume the StructQL codec implements arbitrary database SQL functions or CASE:
discover its supported projection contract and validate it before authoring.
Use an isolated synthetic fixture for missing/null/empty metadata; never alter
shared source data to manufacture that case. Report an unavailable native
capability precisely rather than claiming the fallback is complete.

Test matched and missing dictionary rows, full-key and reduced-grain queries,
unchanged fact counts/totals, and bounded batched query counts before publication.
See the native fact/dictionary graph in [the worked example](worked-example.md).

## Create and prove the result

Use management MCP to create the component and edit its exact draft revision.
Read back source and metadata, validate the returned revision, and publish only
within the requested environment. Discover the namespace runtime endpoint and
invoke the published tool; management discovery alone is not runtime proof.

Test the advertised fields and descriptions, projection, sorting, paging,
presence/null semantics, empty results, invalid inputs and unauthorized scope.
For cached readers, test cold and warm results, repeated calls, distinct caller
scopes and expiry/invalidation through the owning mechanism. Use authorized
fixtures for negative cases. Record query counts and end-to-end timing when
performance is part of the contract.

Retain the request, skill revisions, discovered contracts, proposal, exact
component/draft/source revision and safe verification results. Separate saved,
validated, published, active and invoked states. Refine reusable skills from
observed failures with neutral examples, not rules keyed to business names.
