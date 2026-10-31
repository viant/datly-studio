# Project instructions

## Go context convention

- When a function or method accepts `context.Context`, it must be the first parameter. This includes test helpers and internal APIs.
- Pass the caller’s context as the first argument. Context is required, never an optional or variadic trailing argument. Do not pass nil or create a replacement background context just to satisfy a signature.

## Application-neutral runtime

- Keep Data Studio independent and application-neutral. Application-specific binary extensions are limited to custom predicates and custom codecs; do not link per-report runners, report IDs, field mappings, business formulas or subscription workflows into the upstream runtime.
- Application-specific orchestration belongs in the consuming application, not the upstream runtime.
- Author application readers, writers, source branches and calculations as managed assets using reusable Datly capabilities. Shared mechanisms and their contracts must work without any particular report or namespace.
- Keep public APIs, examples, fixtures, documentation and commit messages domain-neutral. Do not include customer/business identities, private application terminology, production captures, private hosts or personal workstation paths.

## Native data graphs only

- Production reads and persistence must use Datly v1 components generated with `transcribe get|patch|post|put`. Declare cohesive data/object graphs, complete relation keys, prerequisite reads, predicates, and mutation ownership in DQL with adjacent SQL assets.
- Do not issue application-owned `database/sql` queries/writes or obtain `TransactionSQLProvider` to execute handwritten SQL. Wrapping those calls in a custom component does not satisfy this rule.
- Do not replace each SQL statement with a separate reader and assemble the result in Go. Extend an existing graph when the facts share a lifecycle, scope, and authorization boundary; use separate components only for genuinely independent contracts.
- Standalone readers already perform reads only. Auxiliary `(TABLE)` markers express nonmutating dependencies within writer graphs; preserve those roles when transcribing writers. Executable database SQL must not contain that authoring marker.
- Test fixtures may set up and inspect their isolated database, but assertions must exercise the real generated components, transaction behavior, and authorization boundaries. No new production bypass or allowlist is permitted for a migration.
