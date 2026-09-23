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
