# Prepare a reader for a report consumer

Discover the connected management tools and inspect the actual component/version
before handing a reader to a dashboard agent. Use `versions.inspect` with
`discoverColumns: true` for native result metadata; table schemas alone do not
prove the output aliases, computed types, grouping roles or required parameters.
Follow the workflow reference for exact tool names, current source revisions,
validation, release and namespace readiness. Keep the handoff readable through
MCP; a local repository path is not a contract an external agent can discover.

## Define the analytical contract

Describe each admitted result's purpose and grain, actual output fields/types,
nullable values, dimensions and additive measures. Identify source branch or
context selections that must be fixed before aggregation. A physical source can
contain overlapping summary, daily, ranking or comparison rows; combining those
branches is not a valid total. Expose explicit typed selectors or named producer
results instead of expecting the consumer to guess filters from column labels.
Do not turn hidden source context into a public authorization grant.

Specify date semantics per result: row-date bounds versus a source interval,
inclusive/exclusive endpoints, timezone and default anchor. Ranked interval rows
can legitimately have no row date. If the default is relative to the latest
source date, derive that anchor within the same authorized entity and source
branch; a global maximum from another scope is not an equivalent default. Keep
caller-supplied bounds distinct from producer defaults.

Separate verified identity/entity bindings, required source context and optional
user filters. Required scope predicates must remain effective when optional
filters are omitted or grouped with OR. Preserve native parameter types and
omitted versus empty selection semantics. Reuse linked custom predicates and
their advertised attributes; a type name alone does not prove it is installed.
Use the authoritative producer for business authorization; the dashboard must
not supply a substitute account or contributor lineage.

For each calculated metric, state its additive dependencies and NULL/zero policy.
`SUM(current-prior)` and `SUM(current)-SUM(prior)` differ with partially NULL
pairs. Preserve paired source expressions when required, and calculate ratios
after the intended aggregation. Do not replace missing data with zero unless
that metric explicitly defines zero substitution. Non-additive measures need
their source-native aggregation rule or an authoritative typed finalizer.

## Verify and hand off

Record the component/version, source revision, active generation and actual
producer endpoint/tool/result name. Internal readers consumed by an authorized
runner remain internal; do not enable direct MCP exposure merely to test them.
Verify them through their intended public producer or an authorized management
validation path. For public tools, confirm authenticated runtime discovery and
execution; a successful management save is not runtime evidence.

Supply the consumer with actual argument names/types, approved defaults and
choice domains, response path/shape, row grain, pagination/completeness limits
and verified calculation semantics. Include a bounded executed example and its
independent expected values, plus empty and unauthorized cases. Distinguish
synthetic fixtures from production reconciliation. A successful empty response
does not prove a populated report, and a paged prefix does not prove a complete
aggregate or export.
