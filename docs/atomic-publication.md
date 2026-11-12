# Atomic component publication

`publications.publish` accepts a primary validated component version as before.
To publish related components together, add `input.additionalVersions`:

```json
{
  "namespaceId": "<selected namespace ID>",
  "reportId": "orders",
  "versionNo": 2,
  "input": {
    "expectedSourceRevision": 3,
    "reason": "Publish related readers together",
    "additionalVersions": [
      {
        "reportId": "order-lines",
        "versionNo": 4,
        "expectedSourceRevision": 1
      }
    ]
  }
}
```

The HTTP operation is `POST /v1/studio/sdk/publications.publish`; the native MCP
tool is `studio.sdk.publications.publish`. Go clients pass the same fields through
`sdk.PublishInput`. No additional endpoint, database schema, or deployment setting
is needed. Existing single-version requests and responses remain compatible.

All members must be distinct components in the same namespace and owner scope.
Each member must pass its existing publication authorization, namespace security
policy, validated-version check, and expected source revision. A denied or stale
member prevents staging the whole batch. Additional versions are not accepted by
the rollback operation.

The existing native persistence writers stage all members in one transaction and
one generation. The runtime builds that complete candidate once. Activation then
commits every member, version state, generation state, and publication event in
one transaction. Other active components are repointed once, preserving the
existing namespace selection. Runtime failure restores every staged member;
persistence failure after reload also requests restoration of the previous runtime
generation. Compensation failures preserve the original error and remain errors.

The response is the primary component's normal publication receipt. Its active
generation is shared by every batch member. Read `publications.get` for each
member to verify the requested versions and generation after publication. A
transport timeout does not prove failure: read back the publication state before
retrying. The existing staged-generation lease and recovery behavior still apply.

This operation does not modify report SQL, bypass reader authorization, enable MCP
exposure, or replace named artifact-release policies. A disabled namespace still
cannot activate a runtime generation.
