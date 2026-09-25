# Generic resource permissions

This package is shared by Datly Studio and embedding applications. Resources are
identified by tenant, kind, ID and version. The package is independent of any
embedding application's workflows. Missing policies deny access.

`Evaluate` handles protected/public actions, all/any rules, subjects, roles,
feature exposures and entity type/ID constraints. Mandatory tenant and entity
scope checks remain outside OR branches. Its returned scope must be bound by the
resource executor; policy management does not itself wire runtime enforcement.

## Management

`Service` separately authorizes `viewAccess` and `manageAccess`. The SQLite store
records immutable policy revisions and atomically advances a revision pointer.
`Provision` is a deployment-only operation for initial policies; clients cannot
call it. SDK operations are `access.get`, `access.replace`, and `access.context`.

Policy reads and writes execute the generated `studio/resource_policy` components.
They are hosted in-process behind these authorized SDK operations and deliberately
excluded from the default standalone `GoBootstrap.Packages` endpoint. The writer
owns head CAS and history insertion in one Datly-managed transaction.

The Studio Security workspace includes Permissions and Authorization predicates.
Permissions edits an explicitly selected resource policy using the shared
`ResourceAccessEditor`, exported from the UI embedding API. Current choices come
from verified identity claims. Configure `Service.Directory` for broader trusted
provider catalogs. The editor never supplies authoritative identity facts.
Its structural pre-save review shows changed actions and the current policy
revision, not an effective authorization decision. The server rechecks
`manageAccess` and the Datly policy-head CAS when saving.

Configure `studio-api` in authenticated mode with `-access-issuer`,
`-access-audience` and `-access-public-key` (RSA PEM). Its BFF credential must satisfy
both the configured Studio authentication and access-provider validation. Separate
issuer configuration does not automatically exchange tokens. Policy management is
not enabled through Studio's development-identity header.

The OAuth provider accepts signed access tokens with standard `iss`, `aud`, `sub`,
`exp`, optional `nbf`/`iat`, plus `tenant`, `roles`, `exposures`, and the canonical
grouped claim `"allowedEntities":{"project":[101,102],"organization":["north"]}`.
The public Go fact is `access.Facts.EntityGroups` (`access.EntityGroups`, a map
from type to `[]access.EntityID`); `IDsForType("project")` exposes its checked
typed IDs. JSON string IDs stay strings. Positive JSON integer IDs up to
`18446744073709551615` are converted exactly to decimal strings; zero,
negative, fractional, exponent and larger JSON numbers are rejected. An absent
type or empty list grants no entity access. Duplicate type keys, duplicate IDs,
null lists and malformed types deny the claim. Signed legacy flat arrays
`[{"type":"project","id":"101"}]` and flat Go `Facts.Entities` remain an
explicit compatibility path; serialization emits the grouped map, and any
populated flat and grouped views must agree or evaluation denies. Only asymmetric
algorithms explicitly allowed in configuration are accepted. Keys/JWKS
resolution is an injected deployment responsibility. Claims are refreshed on
each request.

## Current integration boundary

Legacy deployments retain their existing ownership/ACL checks. A runtime configured
with `Access` instead uses generic policies on HTTP/MCP component invocations and
resource reads. Component keys are `{kind: component, id: Studio ID, version: active
version, tenant: configured tenant}`. Policies are loaded on each invocation.

Example runtime configuration (in addition to its existing listener/database fields):

```yaml
Access:
  Tenant: example
  Issuer: https://identity.example
  Audience: studio-access
  PublicKeyFile: /configured/access-public.pem
  ResourceBindings:
    "skill://operations/":
      kind: skill
      id: operations
      version: "1"
      tenant: example
```

Resource bindings are deployment-owned and use the longest matching URI prefix.
Missing policies or bindings deny. Component entity scopes bind through a
required native `component` input for the server-owned access context, with a
`param` input deriving typed IDs for SQL predicates. No separate scope-binding
registry is required. See [access-context binding](../../runtime/host/AUTHENTICATION.md).
HTTP and MCP tests exercise real SQLite queries, identity isolation and rejected
override attempts. Bounded decisions on non-component resource reads still deny.
Publication dependency checks and separate discovery/description action hooks
remain pending. Initial
policy provisioning is required; enabling this configuration does not import or
silently inherit legacy ACL grants.
