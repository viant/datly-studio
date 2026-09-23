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

# Typed entity scope for generic resource access

With `Access` configured, an `execute` policy that sets `EntityType` yields an
entity-bounded decision. The runtime never filters rows after the fact and never
exposes scope as metadata: it binds the authorized IDs into a server-owned typed
input that the compiled query consumes. The component DQL declares the input
with the reserved `scope` source kind, which no HTTP or MCP transport provider
serves, and the deployment confirms the binding per published version:

```dql
#define($_ = $ProjectIDs<[]int>(scope/project).Required().WithPredicate(0,'in','t','project_id'))
```

```yaml
Access:
  Tenant: example
  Issuer: https://identity.example
  Audience: studio-access
  PublicKeyFile: /configured/access-public.pem
  ScopeBindings:
    - Component: tasks      # Studio report ID
      Version: "1"
      EntityType: project
      Parameter: ProjectIDs
```

HTTP execution and MCP tool calls share one authorization path. For a bounded
decision the runtime converts every authorized `project` ID to the compiled Go
element type (`string` or any integer kind, canonical decimal only, so `"007"`
never aliases `7`) and binds the slice with authority over transport input;
query, header, body or MCP arguments named like the parameter are ignored.

Enforcement fails closed. A bounded decision on a component without a binding,
an unbounded or public decision on a scoped component, an entity type other than
the bound one, or an ID that does not convert all return 403 before the query
runs. A generation fails to load when a component declares a `scope` parameter
without a deployment binding, a binding names a parameter or entity type the
component does not declare, the parameter is not `Required()`, its compiled type
is not a slice of string or integer, or the runtime is not configured with
`Access`. Bindings apply to the exact published version they name; stale or
future versions are ignored. Policy and scope are re-evaluated on every request.
