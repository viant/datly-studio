# Project instructions

## Capability reuse and explicit user approval

- Before adding any feature or mechanism, inspect the owning framework's existing APIs, configuration, documentation, implementation, tests, and callers at the exact version used by the application. Prove the relevant native behavior with an executable example before proposing an alternative.
- Trace the existing path across layers and identify its owner. Reuse that path; remove demonstrated duplication instead of adding parallel wrappers, adapters, fallback paths, caches, validation, permission checks, or configuration switches.
- Before implementing a new feature or framework extension, consult the user with the existing options, evidence of the remaining gap, the smallest proposed change and owning layer, and compatibility implications. Obtain explicit approval of that approach first. A request to fix a regression is not approval to invent a new mechanism. Agent or reviewer recommendations do not substitute for user approval.
- Keep validation, strict/lenient argument handling, authorization and contract enforcement in the framework or producer that already owns them. Do not silently replace native lenient behavior with stricter application, host, proxy or UI rejection.
- Passing tests does not justify a redundant feature. Preserve existing contracts and add regressions through the real native path. Do not introduce a new API, flag, gate, compatibility layer or workaround solely to make a test pass.
- Carry these requirements into delegated work. When the user has already approved the exact approach, continue within that authorization; ask again only if the proposed approach or scope changes.

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

## Release qualification

- During troubleshooting, use Endly MCP exact case selectors to run the failed case and every subsequent case. Preserve the session and reuse built artifacts when source is unchanged. Do not restart the complete suite after each fix; run it only after the entire remaining suffix passes.

- Before releasing, run the default `endly` command from `e2e/` with no task or case selectors. The native MySQL qualification and every discovered MCP, permission, and browser case must execute and pass.
- A subset, skipped case, zero-assertion operation, or partial "Passed" count is not release acceptance. Record the exact source revision, Endly revision, terminal status, executed case IDs, and failures/skips.
- Fix the application when it violates its contract; update fixtures or expectations only when the intended contract proves them obsolete. Never weaken authorization or skip failing cases to obtain a release.
- E2E services must use owned fixtures. Never stop unrelated containers/processes or reuse a production datastore; fail on port conflicts rather than taking over another instance.
