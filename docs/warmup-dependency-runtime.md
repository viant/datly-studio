# Dynamic warmup dependencies and credentials

Dynamic warmup is implemented in `runtime/preview/dynamic.go` and the native
`studio.sdk.versions.warmup` handler.

## Native execution logging

Shared host configuration forwards Datly's existing logging profile to full-reader
preview, selected-view testing, and warmup execution:

```yaml
Logging:
  EnableAudit: true
  EnableTracing: true
  IncludeSQL: false
```

The host owns one native observability owner for these dynamic runtimes; each
runtime borrows it through `WithManagedObservability`. Dependency runtime options
and cleanup remain in effect. Omitting `Logging` retains native defaults.
Embedding hosts can supply an existing native owner through
`studio/host.Config.Observability` and remain responsible for its lifetime.

Native logging writes to stdout. Cache-read diagnostics distinguish warmup hits,
lazy hits, and misses using `found_warmup`, `found_lazy`, and `source`. With
`IncludeSQL: false`, SQL text is excluded by Datly's native policy.

The direct native warmup operation emits per-view completion diagnostics while
filling the cache. It does not emit the gateway endpoint's warmup start/completion
messages. Studio's durable job acceptance and terminal persistence diagnostics
are separate; this wiring adds no job lifecycle events or gateway callbacks.

## Host-owned operation deadline

Shared deployment configuration (`datastudio.yaml`) accepts:

```yaml
WarmupOperationTimeout: 15m
```

The shared host forwards this setting to native authoring and the browser
runtime. Omission preserves a trusted embedding override when supplied;
otherwise the native default is five minutes. Invalid durations fail loading.

The existing trusted `studio/host.Config` accepts an optional
`WarmupOperationTimeout` duration string. For example, an embedding host can use:

```go
timeout := "10m"
config := host.Config{WarmupOperationTimeout: &timeout}
```

Supply that configuration through `host.AuthoringConfig.Config` for the native
host, or `app/studioapi.Options.HostConfig` for the SQL API host. The native
authoring snapshot captures the resolved duration once; the SQL API host also
validates and captures it at startup. Omission retains five minutes. Explicit
empty, malformed, overflowing, zero, and negative durations fail startup.
This is not a browser argument, authored warmup case, or Datly gateway timeout.

Layout 29 adds an internal nullable `component_warmup_runs.deadline_at` column.
Acceptance captures one UTC `requested_at` and stores its deadline in the same
native writer operation. The worker uses that absolute deadline, including
commit visibility, compilation, dependency initialization, and execution. The
existing independent five-second commit-visibility bound is unchanged.

Both recovery paths use the accepted deadline, not their current host timeout.
Old rows with NULL deadlines retain their historical `requested_at + 5m` budget;
no historical evidence or timestamps are rewritten. Configuration changes affect
new jobs only. Apply the explicit `029-warmup-deadline` database upgrade before
starting updated hosts, and upgrade all recovery hosts before enabling longer
timeouts. Startup does not migrate existing stores automatically.

The native list/recovery operation declares Datly's `imperative` component-call
policy so each recovery write is visible to its next expiry read inside the
same root transaction. Buffered child writes would leave that loop reading the
same expired batch until root completion. This uses Datly's existing transaction
policy; it does not create independent child transactions.

Accepted jobs remain independent of browser cancellation. Existing execution
guards, completed/partial/failed outcomes, cleanup, and terminal-state persistence
remain in place. The deadline is internal and adds no public response field.

Dynamic warmup reuses `prepareContexts` after definition and predicate-policy
validation, registering the resolved components and runtime options alongside the
reader. Existing SDK access checks remain unchanged. Runtime shutdown precedes
dependency cleanup; dependency cleanup also runs on initialization and preparation
failures.

The accepted-job transport already preserves SDK context values through
`context.WithoutCancel`. Warmup now bridges the server-owned verified bearer into
the native execution contract:

1. Ask `Warmup.CredentialHeaders()` for canonical JWT header requirements.
2. Supply the SDK verified bearer only for `Authorization` (case-insensitive).
   No arbitrary browser headers, caller inputs, or case fields are copied.
3. Call native `Warmup.Prepare(ctx, providers...)` and run the returned operation.

An empty credential requirement remains valid and requires no bearer. Other JWT
header requirements produce an unavailable error explaining that Studio cannot
supply that credential type; they are not classified as invalid Datly contracts.
Native binding, JWT verification, authored override rejection, query preparation,
and warmup authorization semantics remain owned by Datly. This adds no API,
authentication mechanism, application-specific registration, or permission policy.

Regressions exercise native warmup against disposable SQLite and cache storage:

- Credential-free execution and declared dependency resolution.
- Dependency runtime options and authorization predicate execution.
- Real native RSA JWT verification: valid, missing, malformed, and expired JWTs.
- Identity rejection after successful JWT verification.
- Unsupported credential headers, authored credential cases and index overrides.
- Cache files created on success and absent after rejected preparation.
- Dependency cleanup after success, binding/preparation failure, and runtime
  initialization failure; unresolved dependencies remain errors.

All identities and keys are synthetic. No external-data warmup is part of these
regressions. They do not constitute the full MySQL/browser release qualification.

## Native MCP worker handoff

The native `studio.sdk.versions.warmup` handler now captures its execution values
before launching the worker. The namespace comes from the publish-guard result,
not unchecked operation arguments. Principal/claims come from verified native
binding, the bearer is read from the same trusted binder, and the host codec
factory is retained through its existing context API. No full request, handler
session, or unrelated context values are retained.

The five-minute job timeout and existing execution-guard capture are unchanged.
An actual MCP-over-HTTP regression waits for the accepted operation, disconnects
the client while dependency initialization is paused, then polls the durable
result through MCP and verifies native JWT binding and cache creation. Rejection
cases cover invalid transport JWTs, wrong workspace selection, and a dependency
that rejects the JWT; rejected execution writes no cache files and releases its
dependency resources.

Server lifetime is a separate gap: this worker has no server-lifetime context or
shutdown registration. HTTP server shutdown waits for handlers, not detached
workers. This change does not promise job draining or graceful cancellation on
server shutdown. The existing operation timeout, guards, and expiry recovery
remain in effect; no new shutdown mechanism is introduced.

## Native warmup connector requirements

The pinned native reader implements `exec.ReaderWarmupTargeter.WarmupTargets()`.
Call it after `CompiledReader.NewExecution`: native initialization has resolved
named host cache profiles, view policies, plural definitions and shared cases.
A retained metadata regression proves root/child targets and host policies without
executing SQL or warming a cache.

Dynamic warmup now consumes those targets directly. Nonempty policy connector
names are acquired by the same source owner that opens query connectors, using
its existing secrets/options path, registration and cleanup. Already-open names
are reused. Empty connector overrides remain empty; native execution selects the
view connector already loaded for queries. No policy precedence or defaulting is
reimplemented in Studio.

Ordinary preview does not acquire warmup-only sources. The actual MCP regression
uses distinct query/warmup databases: two indexed entries prove warmup used the
dedicated source, whereas the query database contains one authorized row. After
the dedicated connector is disabled, ordinary preview still returns the query
source's row. An explicitly unavailable warmup connector fails without cache
writes. No Datly producer API extension is needed.
