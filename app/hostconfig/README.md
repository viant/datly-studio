# Shared host configuration

This package belongs to public Datly Studio. AI Studio and other embedding
applications reuse it; it contains no application report registry or AR settings.

Status: configuration decoding and listener validation are implemented and tested.
The combined single-process launcher and its connected E2E acceptance are still in
progress. This example is not yet a working replacement for existing launch commands.

```yaml
Endpoint:
  Address: 127.0.0.1:8080
MCP:
  Address: 127.0.0.1:8081
NamespaceMCP:
  Address: 127.0.0.1:8082
Connector: studio
Connectors:
  - Name: studio
    Driver: mysql
    DSN: '${Username}:${Password}@tcp(localhost:3306)/studio?parseTime=true'
    Secret:
      URL: file:///config/database.enc
      Key: blowfish://default
JWTValidator:
  CertURL: https://auth.example.com/v1/api/oauth2/certs
```

`Endpoint` is the public web/API listener. `MCP` serves administrative and authoring
operations. `NamespaceMCP` is the default shared listener for published namespaces.
All three belong to one process by default. Separately deployed services remain an
explicit opt-in; they must not be the normal documentation or E2E startup path.

`Connector`, `Connectors`, `JWTValidator`, and the other embedded fields retain
Datly's exact Go types. The host must reuse Datly's source loading, connector
initialization and SCY resolution; decoding this document does not load secrets,
open database connections, resolve source paths, or establish authorization.
The DSN placeholders match fields in the referenced secret. There is no separate
`dsnSecret` abstraction. Connection pool settings and aliases retain Datly semantics.

For an RSA resource instead of certificate discovery, use:

```yaml
JWTValidator:
  RSA:
    - URL: file:///config/public.enc
      Key: blowfish://default
```

JWT signature verification is distinct from issuer/audience claim policy, trusted
user-info facts, and browser OAuth login. The unified runtime must retain all
applicable checks and reuse existing implementations. A successful config decode
is not evidence that an authenticated deployment is ready.

Namespaces, publications, active revisions and application metadata remain in the
database. `Namespaces` and `Discovery` are intentionally not configuration fields.
Application extensions are linked through trusted Go APIs; authored metadata does
not load executable plugins.

`Decode` accepts a single YAML or JSON object up to 1 MiB. It rejects duplicate YAML
mapping keys, unknown fields and multiple documents. Errors do not echo values
that could contain secrets. Listener validation delegates to Datly's existing
`Endpoint` and `MCP` validation.

Before changing the default E2E launcher, verify one application PID owns the
web/API, admin MCP and default namespace MCP listeners; test readiness, startup
failure cleanup, cancellation, authenticated authoring, generic namespace
publication and discovery. AI Studio must run its AR suite through the same host
and configuration contract, adding only its private extensions.

## In-process building blocks

- `studioapi.Serve(ctx, args, options)` owns the browser/API host until cancellation
  and returns errors to its parent. It does not modify global CLI state.
- `datlycmd.New(ctx, nativeConfig, options, diagnostics)` constructs the native
  authoring server from resolved Datly configuration. The parent uses its `Serve`,
  `WaitReady`, and `Shutdown` methods. There is no subprocess or command parsing.
- Native authoring middleware and provider setup are shared with the existing CLI;
  embedders must register the process-wide provider factory only once.

The native constructor is exercised through real loopback HTTP and MCP requests
with a synthetic SQLite catalog, generated RSA key, injected identity authority,
and linked artifact metadata (`TestEmbeddingStartup/native-embedded`). These tests
prove the constructor path, not the complete combined host or Endly deployment.
