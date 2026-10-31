# Native release persistence

`Store` accepts a Datly `ComponentInvoker`, an immutable verified `value.Scope`,
and a request actor resolver. The invoker must expose the internal generated
components under `studio/artifact_release`. Hosts register those components with
their existing native bootstrap and inject the store into `ControllerConfig.Store`.
There is no SQL connection, transaction callback, or fallback persistence path
in this adapter.

DQL under `dql/studio/artifact_release` is the source of truth. Regenerate each
package with the pinned stock `datly transcribe get|post|patch` command. Generated
Go and embedded SQL must never be edited. `rules.go` contains separate business
lifecycle checks; the generated runtime owns Current reads, validation, buffered
writes, CAS execution, commit, and rollback.

The Stage graph reserves exactly one channel revision and pending operation,
inserts its intent, and creates an immutable binding when absent in one native
transaction. Its revision and pending-operation predicates remain mandatory.
Binding reuse omits the writable child and verifies the generated Current binding
against the expected fingerprint inside the transaction. Supplying an existing
binding for mutation is rejected. A concurrent creator may conflict safely.

Complete validates the pending intent, applies its channel metadata, and marks
that same operation applied atomically. Replay succeeds only after a native read
proves that the exact operation was applied at the requested target revision.
Fail validates the same ownership and pending facts, retains the active release,
and records a bounded failure code atomically. Neither path claims completion
from a queued write.

Snapshot, bundle, exact intent, scoped channel list, and history readers use
native predicates and typed views. Pending recovery validates referenced intents;
pending or failed attempts never become active selections. Scope, policy,
authorization, and artifact projection remain the controller and trusted host's
responsibility. Internal component fields are not browser authority.

The synthetic tests in `studio/artifact_release/stage` invoke the generated
components through `standalone.Server`, including zero-revision CAS, foreign
scope, pending reservations, immutable binding reuse, applied replay, failed
recovery, and database-trigger failures proving transaction rollback. SQL in
those tests is fixture setup/assertion only.
