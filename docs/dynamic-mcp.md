# Dynamic MCP publication

The native Studio authoring MCP endpoint exposes management operations: connector,
namespace and component creation, source editing, validation and publication.
These are separate from the business tools exposed by published components.

## Listener modes and port defaults

Enable MCP in each namespace's UI settings. Both modes retain a separate runtime
source, catalog and authorization context per namespace; sharing a listener does
not merge catalogs. Separate mode is the backward-compatible namespace default.

| Runtime mode | Namespace MCP port | Endpoint |
| --- | --- | --- |
| `separate` (default) | `0` | OS-assigned dedicated port, `/mcp` |
| `separate` | Positive port | Requested dedicated port, `/mcp` |
| `shared` | `0` | Shared `MCP.Address`, `/namespaces/<namespace-id>/mcp` |
| `shared` | Positive port | Dedicated-port override, `/mcp` |

The sample runtime configuration uses `127.0.0.1:8091` for `MCP.Address`. In
shared mode this is the common listener. Set it to another address/port in YAML;
port zero also allows an OS-assigned shared listener. In separate mode, the
namespace's own port setting determines the listener.

```sh
go run ./cmd/studio-runtime -namespace-mcp -namespace-mcp-mode=separate -conf datly-runtime.yaml
go run ./cmd/studio-runtime -namespace-mcp -namespace-mcp-mode=shared -conf datly-runtime.yaml
```

Or set the mode in the runtime file:

```yaml
NamespaceMCPMode: shared
MCP:
  Address: 127.0.0.1:8091
```

The mode flag overrides YAML and enables namespace mode. The existing
`-namespace-mcp` flag without a mode continues to select separate listeners.
An explicit `NamespaceMCPMode` in YAML also enables namespace mode in the stock
executable. Existing configurations with neither setting retain their original
single-runtime behavior.

In shared mode, `/mcp` is not a combined or fallback catalog: a valid namespace
path is required. Disabled, stopped and unknown namespaces return 404. A failed
port reservation or rebind preserves the namespace's previous working endpoint.
Changing a namespace from a dedicated override back to port `0` returns it to the
shared endpoint. Runtime status returns the actual MCP URL; clients and tests
must discover it rather than constructing ports or paths.

Publishing validates an exact component version and source revision, stages a
runtime generation, and requests activation. A successful authoring request is
not sufficient acceptance evidence: check publication status, discover the
namespace tools, and invoke the authored component at that namespace endpoint.
Each tool/resource remains associated with its exact published source identity.

The runtime uses `viant/authz` for per-user discovery and execution. Users may
see different catalogs at the same endpoint when their verified roles, exposures
or entity permissions differ. Namespace visibility remains an independent gate.
Denied tools must also reject direct calls even if their names are known.

End-to-end cases should own separate namespaces, components, policies and source
tables. A case can contain many sequential UI edits, a publication through MCP,
and multiple calls with different users. Keep those calls within the case's
namespace. See [e2e instructions](../e2e/README.md).

## Agently connection resource

A namespace listener exposes `mcp://ns/<namespace-name>` as JSON suitable for an
Agently MCP definition. Public runtime authentication produces a bare connection.
For protected namespaces, configure an OAuth provider and a SCY client-registration
resource on the runtime:

```yaml
Authentication:
  DefaultMode: required
  Providers:
    employees:
      Issuer: https://identity.example.com
      CertURL: https://identity.example.com/.well-known/jwks.json
      Audience: studio
  ConnectionExport:
    Provider: employees
    ClientConfigURL: file:///var/secrets/studio/oauth-client.enc|blowfish://env/STUDIO_OAUTH_CLIENT_KEY
    Scopes: [reports:read]
```

The names and URLs above are illustrative deployment values. The generated YAML
uses the namespace's active dedicated port or shared path, a provider-specific
OAuth endpoint, and its protected-resource metadata URL. It includes the SCY
reference, not decrypted client credentials or access/refresh tokens. Agently's
configured credential resolver owns OAuth linking and encrypted token storage.
Protected runtimes without `ConnectionExport` do not advertise this resource.
Reading or discovering the resource checks current namespace visibility; a
namespace cannot export another namespace's connection.


Virtual connection definitions are derived from the running namespace, including
its actual dedicated port or shared namespace path. The internal runtime API
requires the deployment token and an exact namespace ID to retrieve that JSON;
public discovery must first verify the caller's namespace visibility. Missing
OAuth export configuration is an error, never an implicit bare connection.
