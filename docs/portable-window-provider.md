# Authorized portable window provider

The stock `studio-runtime -conf FILE` host accepts an optional `Forge` block in
its normal configuration. It serves operator-owned windows through
`forgeWindowCatalog`, `forgeWindowDefinition`, `forgeDatasourceFetch`, and native
MCP `resources/list` / `resources/read` on `/forge/mcp`. The ordinary Datly
component MCP endpoint remains `/mcp`.

Set operator-owned `Forge.ProviderIdentity` to also advertise ordinary
`namespaces/list`, `namespaces/get`, `windows/list`, `windows/get`, and
`windows/datasource` tools on that same endpoint. Namespaces and canonical URIs
come exclusively from the existing `Forge.Windows` mappings. This is an
authoritative window provider that Core's shared resource gateway can discover;
it does not export an aggregated gateway or a report catalog. Omit the identity
to retain only the existing portable tools.

The ordinary provider preloads complete `window.bundle` definitions and exact
component bindings at startup, without a caller or authorization grants. It
retains only authored content and compiled native metadata read plans; every
read uses fresh namespace/role authorization and current database metadata.
Configured files, traced imports, DQL, resources, connector/source identity,
and linked artifact declarations are checked before serving. Changes deny the
existing snapshot until restart, rather than recompiling or switching a pin.
The existing native datasource executor retains its independent authorization,
typed inputs, predicates, exact component checks and buffered postchecks.

`windows/get` returns a separate runtime `executionProof`. A provider-owned
`windows/datasource` call must forward it along with the exact resource pin,
datasource ID and untrusted inputs. Its signature covers the original provider
lease, actor binding, URI, candidate and component-bearing window variant;
clients cannot extend the deadline or use another actor/provider's pin. A host
may further narrow its own lease and must retain its independent target proof.
Proofs are never part of authored definition bytes or an ACL grant. By default
the signing key is process-owned and changes on restart; replicated hosts can
inject a shared `Forge.ExecutionProof` signer/verifier through trusted Go
configuration. No signing keys or connector credentials are returned to clients.

`Forge.Windows` supplies a canonical `window://namespace/name`, a definition
path relative to `RootDir`, and a map of datasource IDs to `ComponentReference`
values. Each reference declares `kind` (`dynamic` or `linked`), `id`, an exact
`revision`, HTTP `method`, and `route`. The definition file supplies the Forge
window and portable datasource presentation contracts. Routes, connectors and
credentials never come from datasource request inputs.

`Forge.Policies` must authorize discover, describe and execute on each exact
window resource (`kind=window`, `id=namespace/name`, configured access tenant,
`version=working`). `Forge.Selections` must explicitly select that working
candidate. An unversioned file has one mutable working candidate; there is no
invented history or implicit latest/publication selection. A configured verified
account principal resolver is required. Generic signed JWT facts work in the
public executable; private user-info payloads require a private embedding host.

The source bundle fingerprints both the window bytes and its resolved native
component bindings. Definitions and fetches use Forge's shared
`identity.ResourceResolver`, retain the selected resource/content identity, and
recheck namespace, identity, window policy and source content before data release.
A file or component change invalidates an existing pin rather than switching it.

Use `ResolveComponentBinding(ctx, reference)` to resolve an exact executable
source and `ExecuteComponentJSON(ctx, reference, binding, inputs)` to dispatch it.
Dynamic revisions are canonical positive version numbers and use their validated
or published source and versioned resources. An approved older version is
compiled into its own native runtime even while another version is active.
Linked sources are supplied by the embedding application with an explicit
artifact revision/fingerprint and a fresh native Datly build. Both paths retain
native typed bindings, authorization and predicates. The old route-only
`ExecutePublishedJSON` helper rejects calls without a revision binding.

`ModulePath` supplies virtual package authority for dynamic execution in a
source-free deployment. It uses Datly's native `RuntimeContractsInModule` path;
no deployment `go.mod` or source checkout is required when it is configured.
Application packages and their required types/resources must still be linked.

A programmatic `Config.ForgeProvider` remains available for hosts with another
trusted resource source or revision policy. It cannot be combined with the stock
`Forge` block. Datly Studio supplies window transport and component execution;
report authoring, report storage and report orchestration remain outside Studio.

The private AI Studio `cmd/studio` embedding registers its explicit Viant
provider before static SDK/native handlers start. Public Studio's
`runtime/accessprovider` startup factory and trusted request context keep those
paths on the same provider without a private dependency.

Connected tests use local SQLite fixtures, signed credentials and real loopback
HTTP/MCP. They cover stock catalog/definitions/fetch/resources, invalid signature,
expiry and role denial, selected older versions, DQL/resource/window drift,
linked artifact drift and source-free execution:

```sh
go test ./runtime/host ./runtime/resources ./runtime/accessprovider
go test ./studio/resource_policy/access ./app/studioapi
```

Host startup binds both HTTP/MCP transports before serving either. A failed
second bind releases the first listener, and repeated `Start` calls reject
without replacing live listeners. Local integration tests currently require the
reviewed local module workspace; release pins and deployment remain separate.
