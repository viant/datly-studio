# Authorization and identity

Studio separates three decisions:

1. Namespace visibility determines which workspace a verified principal can see.
2. Component authoring ACLs govern viewing, editing, running, publishing and
   managing a component. A public namespace does not grant those capabilities.
3. `viant/authz` evaluates the published resource's policy for actions such as
   `discover`, `describe`, `execute` and `retrieve`.

The Permissions UI edits server-owned policy documents. A policy can use a role,
feature exposure, subject or allowed entity, combined with all/any rules.
`entityType` additionally requires bounded execution. Identity choices originate
from the verified provider; entering role names in an operation does not grant
those roles to the caller. Existing policy replacement checks `manageAccess`
and the current policy revision to reject stale writes.

Initial policy creation is opt-in through the host-owned
`STUDIO_ACCESS_POLICY_EDITOR_ROLES` setting, a comma-separated role list. Empty
configuration grants no initial creation authority. The UI's **Create access
policy** path uses the shared `authz.sdk.policies.create` contract. The server
resolves an omitted policy tenant from verified identity facts and rechecks
creation authority. Subsequent edits use the resource's management policy.

## Provider integration

Generic deployments can verify dedicated JWTs. An embedding host can register
`runtime/accessprovider.RegisterEnvironmentFactory` before native components
start and pass the same provider to `runtime/host.ResourceAccessConfig.Provider`.
The provider supplies verified subject, issuer, tenant, roles, exposures,
entity permissions and expiry. Provider outages and invalid identity must deny.
Public Studio does not depend on a private IdP module.

The e2e mock follows the Auth-style `GET /v1/api/user/info` envelope:

```json
{"status":"ok","info":{"subject":"example-user","userId":10,"accountId":42,"roles":["XYZ"],"features":["EXPOSE_SALES"],"entityPermissions":[{"type":"customer","id":"10","permissions":["read"]}]}}
```

The fixture adapter verifies the signed credential and matches subject/user/account
before accepting this response. Host-specific mapping of source permissions to
allowed entities must be explicit; entity permission names are not global roles.

## Bounded data access

Hiding an MCP tool is insufficient: direct invocation must repeat authorization.
An entity-bounded component binds Studio's server-owned access-context component
and uses its typed IDs in a SQL predicate. A bounded policy on a reader lacking
that binding is denied. Client query or MCP arguments cannot replace the native
access context. See `runtime/accesscontext` and the scope-enforcement tests.

The requested UI-driven role/exposure/entity matrix is under construction; its
planned scenarios are not included in the earlier 16/16 acceptance result.

### Namespace administrators

Set `STUDIO_NAMESPACE_ADMIN_ROLE` to an exact IdP role name to permit namespace
administration across owners. The role is optional and disabled when unset. Its
value must match a role returned by the configured, verified authorization
provider; request bodies and browser fields cannot grant it. Expired identity,
missing roles, differently cased names and provider failures do not grant the
administrator capability. Component execution remains governed by its own policy.

### Namespace manager delegation

A namespace owner can assign `managerRoles` in Namespace settings. Unlike
`STUDIO_NAMESPACE_ADMIN_ROLE`, each assignment applies only to that namespace.
Verified members can change namespace settings and viewer/manager grants, manage
namespace skills, edit and publish its components, and change component read
permissions. Component ownership remains unchanged. Public visibility and viewer
roles never imply management. Removing a manager role takes effect on subsequent
requests; stale namespace updates are rejected by the existing ETag check.

Manager grants use the existing namespace roles JSON column; the schema is
unchanged. Published execution remains governed by resource policies.


### Namespace creation and delegation

Embedding hosts may set `STUDIO_NAMESPACE_CREATE_ROLES` to a comma-separated
list of exact verified roles permitted to create namespaces. If unset, the
configured `STUDIO_NAMESPACE_ADMIN_ROLE` is used. With neither setting, native
public authoring remains available to authenticated users. Wildcards do not grant
creation. The native HTTP/MCP writer and SDK SQL transport enforce this before
inserting a row; namespace listing returns `canCreate` for the shared editor.
This restriction does not revoke existing owner or delegated-manager rights.
Configured namespace administrators may also manage namespace-default policies;
reader predicates retain their own authorization boundary.
