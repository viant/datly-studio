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
Version 26 adds immutable producer-validated component bindings for each
bundle so the Studio API and native runtime use the same exact component
revisions and fingerprints; no browser alias is an authority.

Authz schema setup is for fresh schema creation. It does not rename previous
policy tables, rewrite keys, backfill history/audit fields, or apply a policy
version ledger. The SQLX allocation ledger and canonical layout marker are included in the main
creation DDL. `schema.Initialize` creates an empty store and verifies an existing
store without ALTER statements, history conversion, backfill, or repair. The
explicit additive `025-artifact-releases` and
`026-artifact-release-bindings` scripts upgrade a version-24 Studio store in
order after an operator backup; startup never applies them implicitly.

Focused validation:

```sh
GOWORK=off go test ./schema ./app/studiohost
```

The root-owned disposable MySQL check can apply `schema.MySQLDDL()` to an empty
local test schema and inspect the full policy keys and namespace association
constraints. This package's unit tests inspect the composed MySQL DDL without
opening a database.

Version 27 adds namespace settings and the component-version resource-check opt-in.
Apply `027-namespace-settings` explicitly after version 26; startup remains read-only
for existing stores. Settings use `(owner_id, namespace, setting_key)` and have no
cascading namespace delete. Namespace ETags protect atomic settings updates.
