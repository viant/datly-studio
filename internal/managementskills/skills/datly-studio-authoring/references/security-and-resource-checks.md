# Namespace security and opt-in query resource checks

Discover the connected tool schemas first. These examples use the supported
Studio SDK options; they are not DQL directives or global database driver settings.
Use the namespace name, component ID, version and revisions returned by this server.

## Read and change namespace settings

Read `studio.sdk.namespaces.settings_get` for an authorized namespace administrator:

```json
{"name":"analytics"}
```

The response contains `settings` and the namespace's current `etag`. Only the verified namespace-admin role can add, change or remove these settings.
Owners and delegated content managers do not receive that permission implicitly.
There is no separate setting revision. Namespace settings persist independently of DQL and
component specs, so ordinary source and namespace metadata edits retain them.

`studio.sdk.namespaces.settings_update` accepts only the groups being changed.
For example, configure resource thresholds without changing security:

```json
{
  "name":"analytics",
  "input":{
    "etag":7,
    "settings":{
      "resourceLimits":{
        "maxBytesProcessed":10737418240,
        "maxReferencedTables":10,
        "maxReturnedRows":100000
      }
    }
  }
}
```

Replace `7` with the returned ETag. Omitted groups remain unchanged. A supplied
group replaces that group's value: when changing one threshold, carry forward
its other saved thresholds unless the user wants them cleared. An omitted
threshold means unlimited; zero is an actual zero threshold. Negative values
are invalid. On conflict, read again and reconcile rather than overwriting.

Configuring namespace thresholds does **not** enable checks on components.

## Opt one component version in or out

Read `studio.sdk.versions.get` or `versions.inspect` and preserve the existing
`resourceCheckEnabled` choice when making unrelated edits. New versions default
to false unless explicitly configured; native draft cloning retains the source
choice. Do not turn this on for every component just because a data connector
or namespace limits exist.

When the user wants resource checks on a particular editable draft, use
`studio.sdk.versions.apply` with this command:

```json
{
  "reportId":"component-id-from-response",
  "versionNo":3,
  "namespaceId":"canonical-namespace-id-from-response",
  "command":{
    "kind":"set_resource_check",
    "expectedSourceRevision":12,
    "payload":{"resourceCheckEnabled":true}
  }
}
```

Substitute the returned identities and source revision. Explicit false disables
this option. This is an SDK version command, not a Reader Builder operation.
Do not write a fabricated `#setting` or rewrite SQL to store the choice.
The command preserves source text, advances the existing source revision and
clears validation evidence. Validate the returned revision before publication.
Published versions require a new editable draft.

Only opted-in component versions attach resource admission. Their native reader
queries, relation batches, partitions and composed projections are checked on
those queries' actual source connectors. Disabled components use ordinary native
execution with no extra resource dry runs. The source engine handles physical partition pruning and nested SQL inside
each submitted statement.

Where supported, a dry run can report estimated bytes processed, estimate
accuracy, referenced tables and schema. Check the connected driver’s capabilities. It does not return SELECT result-row cardinality. `maxReturnedRows`
is enforced on actual results before return; do not claim it was predicted by dry
run. Keep checks enabled on a budget or capability failure and report the cause;
do not silently disable the user's option to obtain a successful execution.

Use only capabilities advertised by the connected build. Full native admission
and the driver's statistics API require their supporting Datly and driver builds.
Local implementation or a passing mock is not evidence that a remote server was
upgraded or that an actual source query was qualified.

## Namespace predicate packages and required security

The `security` setting contains `predicatePackages` and
`requireSecurityPredicate`:

```json
{
  "name":"analytics",
  "input":{
    "etag":8,
    "settings":{
      "security":{
        "predicatePackages":["example.com/app/security"],
        "requireSecurityPredicate":true
      }
    }
  }
}
```

Use actual host-approved linked package paths discovered from
`studio.sdk.authorization_predicates.types`; example paths are placeholders.
Missing, null and empty package lists select no external handlers; nothing is
enabled by default. The setting does not download packages or compile Go code.
Preserve the saved requirement and package selection when changing resource limits.

Before component publication or release bundle creation, the server checks the
component's compiled Datly metadata against its owning namespace's setting. A
required input must declare an explicitly marked authorization handler from a selected package;
explicit application when absent also qualifies. An import or a handler from a
different namespace's package does not satisfy the requirement. The server
rechecks even when the component was validated before the namespace flag changed.
Other namespaces are unaffected by this namespace's requirement.

Use `studio.sdk.authorization_predicates.types` with the actual canonical
`namespaceId` for the scoped handler choices. Unscoped discovery remains available
for startup/recovery. Namespace visibility and publisher permission are verified
server-side.

Before choosing a handler, read its advertised description and any linked usage
resource. Record the exact package/type, accepted bound input, authored arguments,
defaults and dependency requirements. Query the governed predicate links when
the host exposes them. Do not infer a handler contract from its type name or
assume that one namespace's configured package is permitted in another.

Apply both the namespace setting and the user's stated requirement. A user may
require authorization even when the namespace minimum does not; a false or
missing setting does not cancel that request. An explicit exemption belongs to
the application's namespace configuration, never to a business-name rule in
this public skill. If authorization is required and no usable scoped handler is
available, report the missing prerequisite without weakening the requirement.

After applying the draft, inspect its compiled parameter metadata and verify
that the authorization handler is selected from the allowed package, binds the
declared trusted source, and remains active for absent optional filters. Validate
and publish the same source revision. Runtime invocation must still demonstrate
the intended permitted and denied cases; discovery alone does not prove scope.

**Example: a reader validates in one namespace and reports `runtime_predicate_type_unavailable` in a fresh namespace.** Call `studio.sdk.authorization_predicates.types` first with `{}`, then with the target `namespaceId`, on the same host. If the handler appears only in the first result, read the target’s `settings_get`. A namespace administrator can add its package to the existing `security.predicatePackages` with the returned ETag and preserve the other packages and required-predicate policy. Repeat scoped discovery and validate the original draft. A fully qualified DQL type name still requires namespace selection. If unscoped discovery also lacks the handler, verify the host build and linked type registration first.

A trusted optional main-DQL validator can impose additional application rules and
return diagnostics. Agents cannot install executable callbacks through DQL or tool
arguments. Use the existing namespace setting and version operations rather than
inventing a source-level hook.

Native validation failures are diagnostics. Reinspect the exact revision, resolve
the reported policy or linked-package issue, then validate again. Report source
validation, dry-run resource evidence, actual query execution and publication as
separate outcomes.

## Explicit authorization occurrences

On hosts supporting `WithAuthPredicate`, use the same arguments as
`WithPredicate`, for example:

```sql
.WithAuthPredicate(0, 'handler', 'security.Scope', 'Column', 'tenant_id', 'Alias', 'f')
```

The handler and options above are illustrative: discover the actual host contract.
The fluent method sets Datly's per-occurrence `Auth` metadata; no extra auth tag
is needed. Other predicates on the same input remain ordinary filters.
An ordinary handler from an allowed package does not satisfy an enabled
authorization requirement merely because it is required or named like security.
The marker alone is insufficient: package admission, requiredness or explicit
absent-input activation, trusted binding and runtime authorization remain distinct.
When requirement enforcement is disabled, do not enable it as a side effect of
authoring. Existing enabled namespaces must have their saved definitions audited
and migrated to explicit markers before deploying stricter validation.
