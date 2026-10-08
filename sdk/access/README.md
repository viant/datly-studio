# Studio access integration

This package is Studio's host adapter for the shared `github.com/viant/authz`
provider and evaluator. It supplies the namespace-aware resource catalog and
transport compatibility layer. Policy evaluation, policy storage, and the
generated policy reader/writer components belong to the shared Authz modules;
`sdk/access` does not define a second evaluator or policy store.

## Canonical shared operations

Studio exposes the shared policy operations as POST routes:

- `/v1/authz/sdk/policies.get`
- `/v1/authz/sdk/policies.context`
- `/v1/authz/sdk/policies.replace`
- `/v1/authz/sdk/authorization.check`

The matching MCP tools are `authz.sdk.policies.get`,
`authz.sdk.policies.context`, `authz.sdk.policies.replace`, and
`authz.sdk.authorization.check`. The generated Studio UI client uses these
routes through the shared access adapter. The authenticated BFF forwards these
exact paths with the verified bearer credential from the session cookie.

The Studio adapter continues to enforce host namespace and JWT checks.
The `access.get`, `access.context`, and `access.replace` POST operations under
`/v1/studio/sdk/` remain thin compatibility wrappers around the shared handlers. The separate
`access.list` operation remains Studio-owned because catalog visibility is
namespace-aware.

Policy editing requires `manageAccess`; reading a policy requires `viewAccess`.
Runtime decisions use the shared authorization statuses: 401 for missing or
invalid identity, 403 for denial, and 503 when the decision service is
unavailable. Entity-bounded decisions carry typed, bounded SQL context for the
host runtime to bind.

This migration exposes the listed policy and authorization-check operations.
It does not mount shared gate, create, or catalog endpoints. The Authz policy
schema is initialized from canonical fresh DDL; it does not rename or convert
legacy policy-head tables.

## Host configuration

Configure `studio-api` in authenticated mode with `-access-issuer`,
`-access-audience`, and either `-access-public-key` or `-access-cert-url`.
An optional `-access-user-info-url` supplies verified authority facts. The
session credential must satisfy both Studio authentication and the access
provider's issuer/audience checks; configuring another issuer does not exchange
tokens. Client-supplied roles and development identity headers are not grants.

Published component execution uses the runtime's existing `Access` configuration.
Policies are resolved by exact tenant, kind, ID and version on each request.
Missing policies deny; enabling Authz does not import legacy ownership grants.
See [runtime authentication and scope binding](../../runtime/host/AUTHENTICATION.md)
for the typed access-context input required to enforce entity-bounded decisions.
