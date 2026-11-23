# Recurring warmup schedules

Studio can run an interval schedule for an exact numbered version. The schedule
uses that version's saved effective warmup plans, including root and child plans.
It does not edit the reader, construct cases, create a version, or change cache
storage. Fixed case values remain fixed when the plan runs again.

## Host setup

Apply metadata migration `030-warmup-schedules` before running the updated host.
An embedding application enables recurring triggering with
`studiohost.Options.WarmupSchedulerCredentials`, using the existing
`sdk.SystemCredentialProvider` contract. A nil provider starts no scheduler.

The provider must renew and return a verified, unexpired signed JWT for
`sdk.SystemPrincipal().Subject` (`system:datly-studio`), with a complete `Bearer`
authorization header and standard JWT claims. Credential acquisition and renewal
remain the consuming host's responsibility. No browser token is retained, no
local token issuer is introduced, and the identity receives no automatic grants.
Grant it the existing component/namespace warmup publish authority, required
connector visibility, and reader-specific authorization facts. Every admission
rechecks current namespace and warmup permissions. Policy checks during native
reader execution remain unchanged.

The process host owns one polling loop and stops it before closing its storage.
Multiple hosts may run the loop: the native revision-guarded admission graph
allows only one to consume a due occurrence. Standalone embedders can use
`host.RunWarmupScheduler` with their own lifecycle and verified native codec
factory. It uses Studio's existing SQL SDK authorizer and warmup executor.

## SDK and native operations

Use `Versions().GetWarmupSchedule`, `SaveWarmupSchedule`, and
`DeleteWarmupSchedule`. Their HTTP routes and MCP tools are respectively
`versions.warmup_schedule_get`, `versions.warmup_schedule_save`, and
`versions.warmup_schedule_delete` under the existing Studio SDK prefix.
All three require existing publish-level authority for the target component
and a selected namespace; permission to inspect a reader alone is insufficient.

Example save body:

```json
{
  "reportId": "reader-id",
  "versionNo": 1,
  "input": {
    "expectedRevision": 0,
    "intervalSeconds": 3600,
    "firstRunAt": "2026-12-01T06:00:00Z",
    "enabled": true
  }
}
```

Revision zero creates the schedule; updates and deletion require its returned
positive revision. An absent schedule is returned as an empty result object.
Interval and first-run changes establish a new cadence. Enabling/disabling
preserves the current next due time; disabling invalidates an already inspected
occurrence but does not cancel accepted work. Intervals must be positive and fit
Go's native duration range. Calendar cron expressions are not supported.

## Admission, history and failures

Missed intervals coalesce into at most one admission, followed by the next future
interval on the original cadence. Acceptance starts the existing host-owned
warmup timeout; time missed while offline does not consume that execution budget.

One native transaction updates the schedule and creates an accepted run, or links
matching accepted/running work using Studio's existing version/source identity.
The schedule's `lastRunId` links the latest admitted or coalesced run. It is an
informational reference, not a foreign key or a complete occurrence history.
Execution status, diagnostics, evidence and deadlines remain in ordinary warmup
history. Deleting a schedule leaves run history intact; deleting its owning
version removes the schedule through the existing guarded deletion graph.

A rolled-back admission leaves the schedule due for a later poll. Errors are
reported through host diagnostics; later polls consult persisted state, including
when the caller could not determine whether a commit completed. A failed business warmup is not retried by the
scheduler; the next interval may create new work. Different source revisions
retain the existing warmup coalescing behavior rather than adding a global lock.

A crash after acceptance commits but before dispatch can leave the run accepted
without execution. Existing expiry recovery eventually marks it failed. This
version does not resume accepted work after a crash and makes no exactly-once
execution promise. Forge schedule management UI is a separate consuming change.
