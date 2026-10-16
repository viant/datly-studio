# Go-linked custom predicates

Studio scans already-linked Go types in explicitly configured packages. It does
not compile arbitrary source packages or load Go source supplied by a browser.
A discovered type must implement `github.com/viant/xdatly/predicate.Handler`.

A host can supply package paths through `host.Config.PredicatePackages`, the
`STUDIO_PREDICATE_PACKAGES` comma-separated allowlist, or the trusted startup
`host.RegisterPredicatePackage` hook. `Types: nil` requests linked-type scanning;
an explicit type list retains the existing registration behavior. Optional
`STUDIO_AUTH_PREDICATE_TYPES=package/path#Type` entries restrict the governed
predicate catalog while preserving linked runtime types for DQL compilation.

In **Security → Authorization predicates → Discovered predicates**, a dedicated
window provides independent search and refresh. Its **Linked Go predicates** table
shows each available Go type, its full package path and alias, including types
without a saved reusable predicate entry. Saved entries and the handler picker
also display the full identity. MCP clients obtain the same `packagePath` and
`typeName` fields from `studio.sdk.authorization_predicates.types`; saved entries
are listed through `studio.sdk.authorization_predicates.list`.

The typed Datly v1 custom handler `studio/authorization_predicates/types` serves
`POST /v1/studio/sdk/authorization_predicates.types`, returning an `items` array
with `alias`, `packagePath` and `typeName`. It requires publisher authorization.
The list includes configured linked types without saved catalog entries and
respects `STUDIO_AUTH_PREDICATE_TYPES` restrictions.

## Compiled example

The standard `cmd/datly`, `cmd/studio-api` and `cmd/studio-runtime` entry points
blank-import `examples/predicates`. Its `OrderPresence` handler registers its own
package path and supports one combined correlated condition across two tables:

- `presence=eligible`: a paid order exists **and** no active exclusion exists.
  Both clauses are returned by the same predicate handler and applied together.
- `presence=exists`: customers with at least one paid order.
- `presence=not_exists`: customers with no paid order, including customers having
  only unpaid orders and customers having no orders.

The example uses `predicate_presence_customers` as the parent table with alias
`c`, `predicate_presence_orders(customer_id, status)` for the positive check,
and `predicate_presence_exclusions(customer_id, active)` for the negative check.
The combined mode emits `EXISTS (...) AND NOT EXISTS (...)`; paid status and active
flag are bound values. Individual existence modes remain available. Invalid modes are rejected. This is a query-filter example, not an
identity provider or a substitute for resource authorization.

```sql
#package('example.com/readers/orderpresence')
#import('examplepredicates','github.com/viant/datly-studio/examples/predicates')
#setting($_ = $connector('consumer'))
#setting($_ = $route('/order-presence','GET'))
#setting($_ = $mcp('owner.orderpresence.read','Customers by paid-order presence'))
#define($_ = $Presence<string>(query/presence).Required().WithPredicate(0,'handler','examplepredicates.OrderPresence'))
#define($_ = $Rows<[]*Customer>(output/view))
SELECT customers.*, type(customers,'Customer')
FROM (SELECT c.id,c.name FROM predicate_presence_customers c
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}) customers
```

See [the handler](../examples/predicates/order_presence.go) and its SQLite test
for duplicate orders, the two-table combined condition, individual branches,
repeated execution and invalid inputs.
Endly case `012_ui_custom_predicate` creates the namespace, connector, reader,
input, linked predicate and access policy through UI, then publishes through
Studio MCP. Its real namespace MCP calls verify the combined condition, individual
modes, repeated execution, rejected input and denied identity (54 assertions in
the focused run recorded in `e2e/results.md`).

## Embedding without the example

The reusable application packages do not import `examples/predicates`. An
external executable can use the application without linking the example:

```go
package main

import "github.com/viant/datly-studio/app/studioapi"

func main() { studioapi.Run() }
```

The equivalent native-command entry point is `app/datlycmd.Run`, and dynamic
runtime applications can use `runtime/host`. Import the example package only
when wanted. For your own predicates, link your package and provide its package
path through the configuration or startup hook. Keep SQL identifiers fixed or
validated and bind user values as placeholders.

## Optional handler descriptions

A handler may implement the structural interface `predicatecatalog.Describer`:

```go
func (*OrderPresence) Description() string {
    return "Filters customers by paid-order presence and active exclusions."
}
```

`Description() string` must work on a fresh zero-value receiver without injected
services or side effects. Pointer and value receivers are supported. Discovery
trims surrounding whitespace and adds optional `description` metadata to the
v1 types response; empty descriptions are omitted. The discovered predicates
window displays and searches this text. Existing handlers need no changes.
