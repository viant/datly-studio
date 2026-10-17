# Shared host configuration

This package belongs to public Datly Studio. AI Studio and other embedding
applications reuse it; it contains no application report registry or AR settings.

The public `cmd/studio` launcher consumes this shared configuration and starts
one application process. HTTP, admin MCP, and namespace MCP have separate public
listeners. Internal native HTTP and reload administration use process-owned
loopback addresses and credentials. Applications can embed `app/studiohost.Serve`.

Connected acceptance covers a compiled binary, synthetic SQLite storage, admin
MCP discovery and an authenticated tool call, database-driven namespace enable/
disable, and shutdown of all public listeners. Broader browser authoring and
publication workflows, MySQL combined-host acceptance, and AI Studio adoption
remain outstanding.

```yaml
Endpoint:
  Address: 127.0.0.1:8080
MCP:
  Address: 127.0.0.1:8081
NamespaceMCP:
  Address: 127.0.0.1:8082
Connectors:
  - Driver: mysql
    DSN: '${Username}:${Password}@tcp(localhost:3306)/studio?parseTime=true'
    Secret:
      URL: file:///config/database.enc
      Key: blowfish://default
JWTValidator:
  CertURL: https://auth.example.com/v1/api/oauth2/certs
JWTClaims:
  Issuer: https://auth.example.com
  Audience: studio
  RequireSubject: true
```

`Endpoint` is the public web/API listener. `MCP` serves administrative and authoring
operations. `NamespaceMCP` is the default shared listener for published namespaces.
All three belong to one process by default. Separately deployed services remain an
explicit opt-in; they must not be the normal documentation or E2E startup path.

The inline connector is only the Studio storage connection, shared by host
services. Application/report data connectors are stored in Studio metadata and
resolved through the normal authorized connector lifecycle. Define exactly one
storage connector; its name defaults to `studio`, so the example needs no name
or selector. `OpenCatalog` uses Datly to resolve SCY secrets and open the pool.
The parent closes the pool after all dependent services stop.

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

## Optional compatibility: existing connector files

The normal deployment keeps the storage connector inline in the main config.
A separate file is optional compatibility support, not a requirement or default.

`hostconfig.Load(ctx, location)` reads YAML or JSON from a local path or an AFS
URL, then delegates the native portion to Datly's existing loader. It retains the
original document URL for relative references and writes no intermediate config
files. For a Platform-style connector file, the shared document can contain:

```yaml
Connector: studio
DependencyURL: connector.yaml
```

`connector.yaml` uses Datly's existing `Connectors` document, including `DSN`,
`Secret.URL`, `Secret.Key`, and pool settings. Dependency loading remains Datly's
responsibility; the shared loader does not invent a credential format or decrypt
secrets itself. `Decode` alone only parses the root document; use `Load` when
relative dependency files must be resolved. Listener addresses currently must be
literal values in the shared document.

## Running one application

```sh
go build -o studio ./cmd/studio
./studio --config studio.yaml
```

The storage schema must be provisioned before namespace runtime discovery. MySQL
uses operator-managed migrations; existing standalone SQLite initialization is
retained by the browser host. The host does not create application/report connector
definitions in deployment configuration.

`PublicURL` is optional for a proxy deployment; otherwise the browser origin is
derived from the HTTP listener. `Tenant` defaults to `studio` for resource policy
lookups. `StaticRoot` optionally points to built public UI assets; embedding apps
can instead supply `Browser.StaticFS` without a directory setting. `SessionKey`
optionally supplies a stable base64-encoded 32-byte key as a plain SCY secret.
Without it, a random process-local key makes existing browser sessions expire on
restart. These are advanced settings, not additional required service wiring.

The browser's `/studio-config.json` is derived from the same public origin and
uses authenticated BFF mode; there is no separate deployment-side browser config.
Browser OAuth login wiring is not yet exposed by the combined launcher and remains
an adoption gate before the full browser workflow can move to it.

The host links the standard authoring components by default, adds internal
`studio`/`authz` aliases for the single storage definition, and supplies the same
identity authority to native authoring, HTTP APIs, and namespace execution.
Application extensions remain process-owned Go bindings. Namespace definitions,
ports overridden per namespace, published components and data connectors come
from database metadata. The default namespace listener is shared.

Run the standalone binary regression from `e2e/`:

```sh
endly -r=local/unified
```

That workflow builds `cmd/studio` and launches precisely that application binary
with a single temporary config, invented identities, and a temporary database.
The checks cover the reported PID, absence of child application services, HTTP
assets, native MCP initialization/list/call, unknown namespace rejection,
database-driven namespace enable/disable without restart, and listener shutdown.
The regression is included in the main Endly workflow. The retained broader E2E
fixture still has its previous custom assembly and requires migration to this
shared host before full-suite adoption can be claimed.
