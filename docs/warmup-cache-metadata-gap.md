# Warmup cache backend metadata: native evidence and historical gap

Historical investigation against Datly `4ea9465ef23b`. The pinned Datly
`b4da7876a2e8` now supplies resolved metadata, and Studio projects it into optional
`target.scheduledTargets` evidence. Execution and historical records are unchanged.

## Native information available at the investigated revision

At that revision, `CompiledReader.NewExecution` resolved per-view cache services
from host settings and supplied services. The compiled reader implemented
`exec.ReaderWarmupTargeter.WarmupTargets()`, but each target contained only `View`
and `Settings` (`CacheWarmupSettings`). Neither included the selected cache name
or resolved backend. `CacheViews()` returned view names, and `ObservationTargets()`
exposed view declarations, not resolved cache services. The compiled plan and
per-view service map were private. The SQLX cache interface also had no
backend-description method.

An executable metadata-only proof initialized equivalent root/child readers in
three configurations:

| Root backend | Child backend | Warmup target metadata |
|---|---|---|
| AFS | AFS | Identical |
| AFS | Aerospike | Identical |
| Aerospike | Aerospike | Identical |

The proof compared serialized native targets byte-for-byte after initialization.
It used disposable AFS locations and a local Aerospike client pool; it executed
no SQL, warmup, cache writes or invalidation. Backend information is therefore
not recoverable from the native targets exposed at that revision.

## Previous Studio result limitation

The previous `sdk.WarmupRun.Target` descriptor contained one view, cache name and
provider string. The operation store serialized that target into `target_json`,
and the projection read it back as the same singular type. Different backends
across targets could not be represented faithfully. Forge displayed the stored
provider; changing Forge's interpretation alone would have hidden the producer
gap.

The previous Dynamic.Warmup code read the authored root cache provider and labeled
an empty value as AFS. A named reference legitimately has no inline provider, so
this default misreported a host-resolved Aerospike service.

## Implemented integration

Studio consumes the initialized native reader's `WarmupTargets()` once, reusing
that snapshot for connector acquisition and evidence. The optional SDK field
`WarmupTarget.ScheduledTargets` preserves each scheduled policy and its descriptive
`Work`. Each descriptor contains only view/policy identity, logical cache name,
native backend kind, connector name and index metadata. Native settings, cases,
provider URLs, credentials and storage addresses are not serialized.

Unknown native backends remain empty/omitted even when a logical cache name is
known. Native `Work` entries, including `IndexMeta` expansion, never add scheduled
invocations. No backend is inferred from authored configuration or cache names.

The existing `target_json` column persists the hierarchy without a database schema
change. Legacy cache, connector and index fields are populated only for a single
scheduled target with a single work entry, using native metadata; otherwise those
fields are empty. The legacy `view` remains the component's root view. Consumers
must use the hierarchy for expanded or multiple targets rather than interpret a
single root backend as global evidence.

Old rows without `scheduledTargets` remain valid and retain their original fields;
reads neither synthesize metadata nor rewrite their JSON. OpenAPI and generated
clients expose the optional addition. Forge's per-target presentation is a
separate consuming change.

Regression acceptance: named Aerospike, explicit/native-default AFS, host profiles,
mixed root/child backends, IndexMeta work and unknown supplied services. New
records must contain the producer's facts; opening old records must not rewrite
them. Backend metadata must not change execution, authorization or cache contents.

No scheduling, execution, authorization, Forge code or historical rows change.
