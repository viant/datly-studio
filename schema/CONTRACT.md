# Studio and Authz schema ownership

`schema.ddl` owns Studio catalog tables. It does not duplicate Authz policy
head or revision DDL. Fresh SQLite initialization appends
`authz/component/schema.PolicyDDL("sqlite")`; `MySQLDDL` composes the same
Studio snapshot with `PolicyDDL("mysql")`. Both paths create the shared
`resource_policies` and `resource_policy_revisions` tables from the Authz
component's canonical schema.

Studio owns `resource_policy_namespace_bindings`, which associates an exact
tenant, resource kind, resource ID, resource version, and policy revision with
a Studio namespace. The shared Authz tables do not contain `namespace_id`.
Studio teardown drops its own catalog and namespace association tables while
preserving shared policy heads and immutable revision history.

Studio also owns scoped named-release bundles, channel pointers, and a pending
promotion outbox. Bundle rows are immutable. A channel's active bundle and
revision change only after publisher projection completes; pending operations
can be replayed after a restart without reporting premature activation. These
tables do not replace Authz policy heads or component ACLs.

Authz schema setup is for fresh schema creation. It does not rename previous
policy tables, rewrite keys, backfill history/audit fields, or apply a policy
version ledger. The SQLX allocation ledger and canonical layout marker are included in the main
creation DDL. `schema.Initialize` creates an empty store and verifies an existing
store without ALTER statements, history conversion, backfill, or repair. The
explicit additive `025-artifact-releases` scripts upgrade an existing
version-24 Studio store after an operator backup; startup never applies them
implicitly.

Focused validation:

```sh
GOWORK=off go test ./schema ./app/studiohost
```

The root-owned disposable MySQL check can apply `schema.MySQLDDL()` to an empty
local test schema and inspect the full policy keys and namespace association
constraints. This package's unit tests inspect the composed MySQL DDL without
opening a database.
