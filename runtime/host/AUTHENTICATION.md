# Separate authoring and execution identities

Studio's login provider governs authors and authoring ACLs. A public execution
runtime does not disable those ACLs. Studio development identities cannot manage
ACLs.

The runtime YAML can declare independent JWT identity providers and policies
keyed by Studio component report ID:

```yaml
Authentication:
  PublicMCPURL: https://mcp.example
  DefaultMode: public
  Providers:
    partners:
      CertURL: https://partners.example/jwks.json
      Issuer: https://partners.example
      Audience: reporting-mcp
  Components:
    vendor-spend:
      Provider: partners
      Scopes: [reports.read]
    public-catalog:
      Public: true
```

Explicit policies apply to HTTP execution, MCP tools and component resources.
They require a verified token from the selected provider with the configured
issuer, audience, subject and scopes. Runtime client subjects do not need Studio
author accounts or authoring ACL entries. Components absent from this map retain
the existing runtime default and its authorization behavior. Policies are
deployment-owned and loaded at runtime startup.

When generic `Access` is not configured and the default mode requires a Studio
identity, the legacy owner/`report_acl.can_run` fallback is read through a
server-only Datly v1 component. It requires a verified subject, considers only
direct user grants, and denies deleted reports. That reader is not registered
on the public HTTP or MCP gateways. With `Access` configured, the generic
resource policy path remains authoritative instead.

With PublicMCPURL configured, each provider has a dedicated MCP endpoint:
`https://mcp.example/oauth/partners/mcp`. Its protected-resource metadata is
available at `/.well-known/oauth-protected-resource/oauth/partners/mcp`.
Unauthenticated endpoint requests return 401 with a `WWW-Authenticate` header
pointing to that metadata. The metadata advertises the provider's issuer; the
MCP host discovers the authorization server and acquires its token there.
The provider-specific endpoint accepts only that provider's validated tokens.

The identity provider remains responsible for OAuth client registration,
authorization-code/PKCE flows, consent and issuing access tokens. Studio never
stores OAuth client secrets in component metadata.

# Entity scope through the native access-context component

With `Access` configured, an `execute` policy that sets `EntityType` yields an
entity-bounded decision. The runtime never filters rows after the fact and never
carries scope through transport-specific state or a scope-binding registry: a
published component binds its server-owned access-context component as an
ordinary Datly component dependency and derives the SQL-bound IDs from that
typed output. The DQL alone declares the binding:

```dql
#import('studioaccess','github.com/viant/datly-studio/runtime/accesscontext')
#define($_ = $Auth<*studioaccess.Output>(component/GET:/_studio/access/context/tasks/project).Required())
#define($_ = $ProjectIDs<[]string,[]int>(param/Auth.Scope.IDs).WithCodec('EntityIDs').Required().WithPredicate(0,'in','t','project_id'))
```

The explicit `EntityIDs` codec converts canonical decimal IDs to the declared
integer slice and rejects malformed or out-of-range values before SQL runs.
For opaque string identifiers, declare a `[]string` output instead; entity
dimension names never determine the value type.

For every (published component, entity dimension) a DQL declares, the runtime
registers one context component in the same generation at
`/_studio/access/context/<reportId>/<entityType>`, so both the resource whose
decision is served and the dimension the SQL applies are explicit in the DQL and
owned by the server. The handler has no client-bindable input: it resolves the
verified caller credential the host already attached to the invocation,
evaluates the `execute` policy for exactly that published component (local
policy intersected with any injected `DecisionProvider`), and returns
`Output{Context, Scope}`. `Context.AllowedEntities` is the canonical
`allowedEntities` map (entity type -> IDs); `Scope.IDs` is the narrowed decision
for the declared dimension and is never broader than it — a decision of another
dimension denies rather than being applied to the wrong column. `component` and
`param` kinds are runtime-protected, so no query, header, body or MCP argument
can supply or override `Auth` or `ProjectIDs`.

Published version numbers belong to Studio's catalog and deployment selection;
the host resolves the selected version server-side when it evaluates the policy.
Datly's component execution key acquires no version or lineage concept.
Published MCP tools also carry `ToolSourceMetaKey` (`viant.datly/source`) metadata
with the catalog-owned tenant, report ID and exact active version. Remote report
clients may compare that identity with an independently configured ACL resource
to detect version drift. The metadata itself grants no access; the runtime still
evaluates the current component policy on every tool call.

HTTP execution and MCP tool calls share one authorization path and one binding.
Enforcement fails closed: a bounded decision on a component that does not bind
its context, an unbounded or public decision on a component that does, a
missing, expired or unverifiable credential, a principal with no IDs for the
policy's entity type, and a direct request to the context route all return 403
before the query runs. A generation fails to load when a component binds another
component's context or binds a context without `Access` configured. Policy and
scope are re-evaluated on every request.
