# Warmup cache backend metadata: verified gap

Status: investigated against Datly `4ea9465ef23b`; the extensions below are
proposed, not implemented. Existing execution and historical operation records
are unchanged.

## Native information available

`CompiledReader.NewExecution` resolves per-view cache services from host settings
and supplied services. The compiled reader implements
`exec.ReaderWarmupTargeter.WarmupTargets()`, but each returned target contains only
`View` and `Settings` (`CacheWarmupSettings`). Neither includes the selected cache
name or resolved backend. `CacheViews()` returns view names, and
`ObservationTargets()` exposes view declarations, not resolved cache services.
The compiled plan and per-view service map are private. The SQLX cache interface
also has no backend-description method.

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
not recoverable from the currently exposed native targets.

## Studio result limitation

`sdk.WarmupRun` has one `Target sdk.WarmupTarget`; that descriptor has one view,
cache name and provider string. The operation store serializes that single target
into `target_json`, and the projection reads it back as that same singular type.
Different cache backends across targets cannot be represented faithfully there.
Forge displays this stored provider; changing Forge's interpretation would hide
the producer gap.

The current Dynamic.Warmup code reads the authored root cache provider and labels
an empty value as AFS. A named reference legitimately has no inline provider, so
this default misreports a host-resolved Aerospike service.

## Smallest changes for review

1. Datly: expose read-only, per-target cache metadata from native materialization
   and warmup work selection: view/policy identity, cache name and normalized
   backend kind. Do not expose provider URLs, credentials or storage addresses.
   Account for output work added by `IndexMeta`; policy targets alone do not
   necessarily enumerate those implicit physical cache targets. Reuse native
   work selection rather than duplicating it in Studio. For supplied services
   whose backend is unknown, retain unknown metadata rather than infer a kind
   from configured names or Go type names.
2. Datly Studio: consume that metadata for new operation records. Its existing
   singular descriptor can carry a single target. Before supporting multiple
   targets with distinct caches, review an additive per-target result/storage
   representation, its compatibility with the existing descriptor, and Forge's
   direct rendering of the new metadata. Do not put a root backend in a global
   field that implies it covers every target.

Regression acceptance: named Aerospike, explicit/native-default AFS, host profiles,
mixed root/child backends, IndexMeta work and unknown supplied services. New
records must contain the producer's facts; opening old records must not rewrite
them. Backend metadata must not change execution, authorization or cache contents.

No native API, SDK/storage contract, Forge code, historical row or backend label
has been changed as part of this investigation. Those extensions require review
before implementation.
