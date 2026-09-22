# Datly Studio on Datly 1.0

Status: redesign in progress; all canonical Studio schema tables have Datly 1.0 component coverage
Audience: Datly, Datly Studio, Forge, and platform engineers
Primary decision: rebuild the unfinished Studio integration on Datly 1.0 public APIs; do not port the legacy repository/view runtime architecture.

## 1. Business objective

Datly Studio is the governed control plane for creating and operating database-backed Datly components. It is not a second Datly compiler and it is not merely a SQL editor.

Studio must let users:

- register, test, activate, and retire database connectors
- create a report/component from SQL, DQL, or structured edits
- inspect views, relations, parameters, predicates, selectors, and fields
- validate drafts with source-linked diagnostics
- preview a draft without changing the published runtime
- version and review changes
- publish components as HTTP routes and explicitly named MCP tools
- enable cube endpoints and optional cube MCP tools
- enable bounded cube composition and optional compose MCP tools
- upload and publish versioned MCP skills and supporting resources
- atomically replace the active Datly application generation
- audit ownership, permissions, publication, and rollback
- manage the experience through a UI built with `github.com/viant/forge`

The term **report** is the Studio product object. Every executable report version compiles to one Datly 1.0 `spec.Component`. Report/cube settings are optional component facets, not a separate execution engine.

Runtime listeners are deliberately separate: the Studio SDK/BFF uses 8080,
linked Studio control-plane Datly components use 8081, dynamic reader HTTP uses
8082, and dynamic reader MCP has the dedicated 8091 listener. The browser sends
only its opaque BFF cookie; the BFF expands that session to the server-held JWT
when proxying dynamic HTTP or MCP traffic.

## 2. Non-negotiable boundaries

### 2.1 Datly owns execution semantics

Datly 1.0 owns:

- DQL parsing and normalization
- canonical component/view/parameter metadata
- SQL/view/relation compilation
- input and output contracts
- predicates, selectors, codecs, partitions, and caches
- reader plans and reader execution
- route execution
- HTTP, MCP, OpenAPI, and report adapters
- atomic application generation publication

Studio must not implement parallel reader, relation, binder, predicate, or SQL-building behavior.

### 2.2 Studio owns governance

Studio owns:

- report identity and lifecycle
- connector metadata and secret references
- drafts, immutable versions, review, publication, and rollback
- ACLs and ownership
- semantic editing commands
- persisted authored sources and compiled snapshots
- orchestration of validate, preview, and publish
- stable UI-facing descriptors and diagnostics

### 2.3 Forge uses only the Studio SDK

Every operation initiated by Forge must go through `github.com/viant/datly-studio/sdk`.

Forge must not:

- import Datly packages
- import Studio stores or control services
- access the Studio database directly
- construct `spec.Component` mutations directly
- call undocumented/ad hoc endpoints
- receive live readers, database handles, `reflect.Type`, or runtime registrations

The SDK is the contract boundary for all UI operations. HTTP is one SDK transport; tests may use an in-process transport implementing the same interfaces.

```text
Forge UI
  -> datly-studio/sdk
  -> SDK transport (HTTP or in-process)
  -> Studio control services
  -> Datly 1.0 public APIs
```

## 3. Current code findings

The current repository is a useful product prototype but is coupled to pre-1.0 Datly internals.

### 3.1 Capabilities worth retaining

The original prototype modeled:

- connectors and connector validation
- reports/components and versions
- draft validation and preview
- publication records
- optimistic concurrency through `etag`
- SQL and DQL modes
- input/output field editing
- view SQL patching
- relation add/patch/delete operations
- route, package, connector, and MCP metadata editing
- compile diagnostics
- reload coordination

The canonical runtime database path is now `schema/schema.ddl`. The
`cmd/studio-migrate` commands `init-studio`, `up`, `down-to`, `version`, and
`seed-sample` apply and inspect that snapshot only; the sample fixture writes
the canonical `reports`, `report_versions`, `report_views`, parameter,
predicate, cube, MCP, runtime-generation, publication, and ACL tables. The
SQL-store migration service delegates to the same canonical snapshot; there
is no second catalog DDL. The old control/store prototype has been retired;
runtime and UI-facing paths consume canonical SDK DTOs.

### 3.2 Legacy integrations removed

The repository originally contained the following integrations with packages removed by Datly 1.0. They were deleted during the redesign reset:

- `control/service/compiler_service.go`
  - `repository/shape`
  - `repository/shape/compile`
  - `repository/shape/load`
  - `repository/shape/plan`
  - `repository/shape/transcriber`
  - `view.Resource`
- `runtime/datlyruntime/runtime.go`
  - `repository.Service`
  - `repository.Component`
  - `service/session`
  - `service/operator`
  - old gateway/router types
- `runtime/datlyruntime/preview_gateway.go`
  - legacy DQL bootstrap and repository construction
- `runtime/connectors/datly.go`
  - `view.Connector` and `view.Resource`
- `control/service/codegen_publish_branch.go`
  - legacy shape transcriber/codegen APIs
- `runtime/reload/datly_gateway.go`
  - legacy source reload contract

An isolated build of the pre-reset Studio against Datly branch `origin/v1` failed
because these package paths no longer exist. The replacement connector slice now
uses Datly 1.0 DQL, generated Go shapes, package bootstrap, embedded SQL resources,
typed reader/writer execution, and the local v1 module replacement. The obsolete
`/v1/api/components` and `/v1/api/connectors` HTTP surface has been removed;
Forge and other UI callers use the Studio SDK.

### 3.3 Verified Datly 1.0 static data-model slices

The current connector implementation is generated from:

- `dql/studio/connectors/reader/connector.dql`
- `dql/studio/connectors/reader/sql/read.sql`
- `dql/studio/connectors/writer/connector.dql`
- `dql/studio/connectors/writer/sql/patch.sql`

Report and immutable report-version metadata are generated from the equivalent
`dql/studio/reports/...` and `dql/studio/report_versions/...` reader/writer
packages. ReportVersion uses composite `(report_id, version_no)` identity and
`source_revision` optimistic concurrency. Its reader exposes collection and
`ByVersionNo` routes plus filters/selectors; physical deletion is intentionally
absent because version lifecycle is represented by state transitions.

Generated component `init()` functions are empty. The user-owned
`internal/dependencylink` package blank-imports each selected package and also has an
empty `init()`. Bootstrap scans only package names listed in `datly.yaml` and
discovers linked holder types through collision-free reflection anchors. No
generated registry, `link_gen.go`, checksum, or package sidecar is persisted.
`x.Registry` remains reserved for genuinely dynamic types.

Generated holders expose `EmbedFS() *embed.FS` and `EmbedNamespace() string`.
Standard writers use Datly's shared metadata-driven writer rather than generated
component-private mutation programs. Reader predicates use `Has` suppliedness;
writer tests cover full insert validation, sparse update validation delegated to
govalidator, explicit null/zero behavior, concurrency failure, and deletion only
where the model declares a delete marker. JSON database columns use SQLX
`enc=JSON` with generated `json.RawMessage` fields.

### 3.5 SDK boundary and shared component fixtures

`github.com/viant/datly-studio/sdk` now defines the Forge-facing client,
connector/report/version/preview/publication/runtime services, canonical DTOs,
stable operation names, typed errors, and a transport-neutral invocation
contract. A package-level guard rejects Datly, control, store, and `database/sql`
imports from SDK implementation files. `sdk/transport/sql` now provides the
canonical database-backed transport for connector and report CRUD/status
operations, but requires an injected authorization decision for every SDK
operation so it cannot bypass the protected Datly routes. Version create/list/get, validation, DQL export, and descriptor
operations and optimistic `versions.apply` structured edits are also backed by
the canonical `report_versions` table. Preview,
publication, and runtime lifecycle operations are backed by
`report_publications` and `runtime_generations`; publication writes the active
generation and publication row atomically. Preview is routed through an
injected Datly execution adapter, so the SQL transport does not reimplement a
query engine. Forge must never bypass the SDK.

Generated component tests use `internal/datatest`. Its `HydrationPhase` consumes
ordered JSON tables, preserves JSON numeric authority, validates table and
column names plus required columns against the live SQLite schema, normalizes
nested JSON for SQLX, and inserts the dataset in one transaction. Query
expectations can use the matching JSON row decoder through `AssertRowsJSON`.

The shared DQL contract test compiles and generates every static reader/writer
package from the canonical schema inventory. It covers connectors, reports,
versions, views/fields, parameters/predicates, cube configuration, MCP
exposures, resources, skills, runtime generations, publications, and ACL.
Focused SQLite runtime scenarios cover the components with mutation semantics;
the contract test covers generation for system-owned rows.

## 4. Datly 1.0 code findings

The reviewed implementation is the local `github.com/viant/datly` branch `origin/v1`.

### 4.1 Canonical model

Datly 1.0 exposes:

- `spec.Component`
- `spec.Route`
- `spec.Parameter`
- `spec.Predicate`
- `spec.View`
- `spec.Relation`
- `spec.Selector`
- `spec.Column`
- `spec.Settings`

References:

- `origin/v1:spec/component.go`
- `origin/v1:spec/state.go`
- `origin/v1:spec/view.go`
- `origin/v1:spec/selector.go`
- `origin/v1:spec/column.go`
- `origin/v1:spec/component_clone.go`
- `origin/v1:spec/view_clone.go`

`spec.Component.Clone()` creates an isolated component graph, including routes, parameters, settings, root view, independent views, and nested relations. This is suitable for copy-on-write draft editing.

Predicates are attached to parameters:

```go
type Parameter struct {
    Name          string
    Declaration   DeclarationKind
    Source        BindSource
    TypeExpr      string
    Required      *bool
    Activation    *RouteActivation
    Predicates    []*Predicate
    QuerySelector *QuerySelectorBinding
    Codec         *Codec
    EmitOutput    bool
}

type Predicate struct {
    Group           int
    Name            string
    Args            []string
    ApplyWhenAbsent bool
}
```

Parameter identity must use `spec.Parameter.Identity()`, not name alone.

### 4.2 DQL compilation

The authored-source entry point is:

```go
compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
    Scope:         scope,
    Name:          name,
    Path:          sourcePath,
    Text:          dql,
    Connector:     connector,
    Resources:     resources,
    Types:         types,
    ColumnRefiner: refiner,
})
```

References:

- `origin/v1:transcribe/compiler.go`
- `origin/v1:transcribe/source_model.go`
- `origin/v1:transcribe/diagnostic.go`
- `origin/v1:transcribe/column/refiner.go`
- `origin/v1:transcribe/compile/reader.go`

`transcribe.Result` includes the component, prepared SQL, source map, declarations, type context/resolver, generated type references, view bindings, handler assets, and diagnostics.

### 4.3 Reader build pipeline

The reader is a staged build, not only `sql/builder`:

```text
DQL/SQL
  -> transcribe.Compiler
  -> spec.Component and spec.View graph
  -> bootstrap.BuildArtifact
  -> immutable reader plan
  -> Artifact.ReaderCompilation().NewExecution
  -> exec.Reader
```

Public composition:

```go
artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{
    Component:       compiled.Component,
    InputType:       inputType,
    OutputType:      outputType,
    DirectViewField: directViewField,
    Types:           types,
    Resources:       resources,
    CodecFactory:    codecFactory,
})

reader, err := artifact.ReaderCompilation().NewExecution(
    bootstrap.ReaderRuntimeConfig{
        SQL:                      sqlComponent,
        ReadCaches:               caches,
        CacheSettings:            cacheSettings,
        RelationFetchConcurrency: relationConcurrency,
    },
)
```

References:

- `origin/v1:bootstrap/artifact.go`
- `origin/v1:bootstrap/artifact_builder.go`
- `origin/v1:bootstrap/reader.go`
- `origin/v1:bootstrap/registration.go`
- `origin/v1:sql/reader/compiler/compiler.go`
- `origin/v1:sql/reader/plan.go`
- `origin/v1:sql/reader/execution.go`
- `origin/v1:exec/reader.go`

Studio should not import reader compiler, collector, row-codec, or reader-service internals. `bootstrap` is the composition boundary.

### 4.4 Atomic application publication

An artifact becomes a runtime registration through:

```go
registration, err := artifact.Registration(registry.RegisteredComponent{
    Reader: reader,
})
```

Datly provides complete generation publication:

```go
manager, err := application.New(seedTypes, options...)

err = manager.Reload(ctx, application.Request{
    Revision: revision,
    Compile: func(ctx context.Context, types *typecatalog.Catalog) (*application.Build, error) {
        return &application.Build{
            Components: registrations,
            Types:      stagedTypes,
            Resources:  resources,
            HTTP:       httpConfig,
            MCP:        mcpConfig,
            Version:    version,
        }, nil
    },
})
```

References:

- `origin/v1:application/generation.go`
- `origin/v1:application/http.go`
- `origin/v1:runtime/runtime_facade.go`
- `origin/v1:runtime/registry/registered_component.go`

`application.Manager` stages the full generation, rejects stale revisions, validates runtime/HTTP/MCP composition, and publishes once. It should replace Studio's custom route/MCP snapshot coordinator.

### 4.5 Connector API

Datly 1.0 uses `bootstrap/connector` and `sql.SQLComponent`, not `view.Resource`:

```go
set, err := connector.Open(ctx, configs, defaultName)
readerConfig.SQL = set.SQL
```

References:

- `origin/v1:bootstrap/connector/config.go`
- `origin/v1:sql/component.go`

Studio Reader Builder receives the complete active connector-name catalog that
is visible to the current principal. Dynamic preview then derives the names it
actually needs from Datly's component/view bindings, opens those Studio
connectors, supplies all handles to `column.Connections`, and registers each
exact name with one `sql.SQLComponent`. A missing or disabled connector is an
explicit unavailable error; runtime execution must never substitute the report
default or maintain a second Studio-only connector alias map. The Studio
binaries link MySQL, PostgreSQL, SQLite, and Viant BigQuery database/sql drivers.

Connector DSN templates are server-only material. SDK and Forge DTOs expose a
`dsnConfigured` boolean, never the stored DSN value. The linked Datly connector
reader likewise projects no `dsn_template` column, and the writer uses Datly's
`output_exclude('Data.DsnTemplate')` wire policy so a valid mutation input is
not echoed as connection material. Connector editing accepts a
replacement DSN but leaves the existing value blank and undisclosed. Changing a
driver, DSN, secret reference, or provider options resets the connector to
`draft`, clears the old probe evidence, and requires a successful probe before
reactivation; description-only edits retain its lifecycle state.

Dynamic component package identity is also server authority. Derive its owner
segment only from the verified JWT `sub`—never `username`, `email`, or a display
claim—by taking text before `@`, lowercasing, and removing every non-alphanumeric
character. New readers use
`github.com/viant/datly-studio/dynamic/<owner>/<report-id>/reader`; callers cannot
write `component_scope` or `component_name`. Existing DQL is rehomed only with
the Datly reader builder's typed, source-preserving `setPackage` operation.

`datly.yaml` is launched by this repository's `cmd/datly` binary, not an
unmodified standalone Datly executable. The Studio binary links the reflection
dependency package and the SQLite, MySQL, PostgreSQL, and BigQuery drivers
needed by the declared connector catalog. On 2026-09-18, the project binary
started the configured control-plane host on `127.0.0.1:8081`; an unauthenticated
request to `/v1/studio/connectors` returned 401, proving the linked component
routes and JWT boundary were active.

`reports.namespace` is a separate business catalog filter (`general` by default),
not part of package identity. Explicit base-reader MCP names must begin with the
same owner segment and must not duplicate another reader's tool name. This is
validated server-side before the revision is persisted; UI suggestions are not
authority.

Reader delegation also uses the verified JWT `sub` as its only principal type.
Studio does not expose role delegation until there is a verified, deployment-owned
role-claim mapping shared by the SDK, Datly predicates, and dynamic runtime.

Version validation is runtime-contract validation, not a nonempty-source flag.
The Studio-owned validator opens every required active named connector, invokes
Datly `RuntimeContracts`, builds the typed artifact, and initializes its reader
without executing the business query. It persists `valid` or `invalid` plus safe
diagnostics for the exact source revision. Publication must continue to require
that validated revision and must not infer readiness from static inspection.

Publication stages a `runtime_generations` row as `building` and a report
publication as `pending`. Studio requests `POST /_studio/reload` from the
dedicated loopback dynamic host using its configured server token. The host
compiles all active and pending reports with named connector bindings and swaps
the Datly application manager atomically. Only a confirmed reload promotes the
generation/publications to `active`; a failure records diagnostics and restores
the preceding publication state. Browser requests carry only an optional release
note and source revision; the server derives the publisher from verified JWT
identity.

Dynamic-host restart recovery is verified against persisted state: the host
test publishes a reader that uses two named connectors, serves its nested
result, shuts down the application manager, creates a fresh host against the
same Studio database, and serves the same route again. Startup rebuilds the
active generation and does not depend on an in-memory component registry.

Datly 1.0 commit `ab47118d` adds a generation-owned `application.Build.Shutdown`
contract. Failed stages close their external resources before `Reload` returns;
published stages close them only after the generation is retired and all
admitted requests release their leases. Studio binds every opened dynamic
connector handle to that callback, so reloads no longer retain retired database
connections until process shutdown.

Versioned `report_resource_files` are materialized into immutable Bindly stores
for both preview validation and the dynamic host. Each report receives a scoped
default resource filesystem while named namespaces remain generation-wide and
owner-reserved. `report_resource_folders` become explicit Datly MCP folder
plans, and `report_skill_roots` become explicit skills only after their root
contains `SKILL.md`. Validation invokes Datly's native folder/skill compiler;
publication reuses the same plans in the atomically reloaded dynamic host.

Unpublish stages a new generation while marking only the target publication
`unpublishing`; the host source excludes that report, its MCP tools, and its
published resource/skill folders. A successful reload advances active readers to
the new generation. A failed reload restores the prior publication snapshot and
records a failed generation.

Rollback is a publication of an existing, validated immutable version through
the same staged activation pipeline. The selected version becomes `published`;
the previously published version becomes `superseded`. Rollback never performs
source replacement or replays authoring mutations.

Connector sets own opened database handles. Generation swapping must define when old handles close after in-flight requests drain.

### 4.6 DQL export limitation

`transcribe/dql.Serializer.Export` retains original source when present. Reconstruction from spec currently supports only a narrow route/connector/root-SQL case and reports limitations for parameters, independent views, relations, selectors, partitions, MCP, handlers, and richer settings.

Reference:

- `origin/v1:transcribe/dql/serialize.go`

Structured edits cannot promise full editable DQL round-trip until Datly provides complete deterministic export, or Studio keeps structured and DQL authoring modes explicitly separate.

### 4.7 MCP tools, cubes, composition, and skills

Datly 1.0 represents a named route tool with `spec.MCPExposure{Kind: "tool", Name: ...}` on the route. Tool names are explicit protocol identities and must be validated for uniqueness across the staged application generation.

Cube behavior is controlled by `spec.ReportSettings`:

- `Enabled` derives the cube component from an eligible groupable GET component
- `MCPTool` controls exposure of the derived cube tool
- `Compose.Enabled` derives the `/compose` component
- `Compose.MCPTool` controls compose-tool exposure
- `Compose.MaxCubes`, `MaxLimit`, and `TimeoutMs` bound request-local composition

Datly currently derives cube tool names from component identity: `<Component>Cube` and `<Component>CubeCompose`. If Studio must customize those derived names independently, Datly needs an explicit naming extension; Studio must not rename registry tools after compilation.

Skills use `spec.Settings.MCPFolders`. Each `spec.ResourceFolder` declares a resource namespace, relative root, absolute URI prefix, and explicit skill roots. Resource files must be registered in the shared Bindly resource store before MCP publication. Datly validates paths, `SKILL.md` metadata, URI/name consistency, duplicate roots, and attempts to escape the published root.

References:

- `origin/v1:spec/mcp.go`
- `origin/v1:spec/component.go`
- `origin/v1:spec/cube_compose.go`
- `origin/v1:spec/resource_folder.go`
- `origin/v1:report/project.go`
- `origin/v1:report/compose_compile.go`
- `origin/v1:mcp/resource/skills.go`
- `origin/v1:mcp/resource/skills_test.go`

## 5. Authoring and source-of-truth model

Each report version is an immutable envelope containing authorship and executable semantics.

### 5.1 SQL mode

The user owns SQL plus Studio metadata. Studio composes a Datly component from:

- SQL
- route and connector
- parameters and predicates
- selectors
- report/cube settings
- MCP exposure

The compiled `spec.Component` is the executable snapshot.

### 5.2 DQL mode

The user owns full DQL source.

- original source bytes are retained
- source edits recompile the whole component
- structured edits are disabled unless source can be updated without semantic loss
- compiled spec is persisted for deterministic execution and inspection

### 5.3 Structured mode

The user edits component semantics through SDK commands.

- each command clones and updates the spec
- Datly validates/builds the result
- generated DQL is a review/export artifact
- export limitations are visible

### 5.4 Persisted version artifacts

Persist:

- authored SQL and/or DQL
- canonical `spec.Component` JSON
- spec format version and hash
- type and resource manifests
- component descriptor for UI
- diagnostics
- generated DQL plus export limitations
- compiler and Datly version
- publication/audit metadata

Never persist live `reflect.Type`, DB handles, readers, caches, or runtime registrations.

## 6. Proposed architecture

```text
Forge windows/actions
  -> datly-studio/sdk interfaces and DTOs
  -> SDK HTTP/in-process transport
  -> control services
       +-- connector service/store
       +-- report/version service/store
       +-- semantic command service
       +-- preview service
       +-- publication service
  -> Datly adapter
       +-- transcribe.Compiler
       +-- bootstrap.BuildArtifact
       +-- CompiledReader.NewExecution
       +-- Artifact.Registration
       +-- application.Manager.Reload
```

Suggested packages:

```text
sdk/
  client.go
  connector.go
  report.go
  version.go
  command.go
  preview.go
  publication.go
  types/
  transport/http/

sdk/transport/sql/
studio/
store/sql/
datlyadapter/compile/
datlyadapter/types/
datlyadapter/preview/
datlyadapter/publish/
ui/forge/
```

## 7. Studio SDK contract

The SDK is a first-class product API, not a thin collection of raw URLs.

```go
type Client interface {
    Connectors() ConnectorService
    Reports() ReportService
    Versions() VersionService
    Preview() PreviewService
    Publications() PublicationService
    Runtime() RuntimeService
}
```

### 7.1 Connector SDK

```go
type ConnectorService interface {
    Create(ctx context.Context, input CreateConnectorInput) (*Connector, error)
    Get(ctx context.Context, name string) (*Connector, error)
    List(ctx context.Context, input ListConnectorsInput) (*ConnectorPage, error)
    Update(ctx context.Context, name string, input UpdateConnectorInput) (*Connector, error)
    Test(ctx context.Context, name string) (*ConnectorTestResult, error)
    Activate(ctx context.Context, name string) (*Connector, error)
    Disable(ctx context.Context, name string) (*Connector, error)
    Delete(ctx context.Context, name string) error
}
```

### 7.2 Report/version SDK

```go
type ReportService interface {
    Create(ctx context.Context, input CreateReportInput) (*Report, error)
    Get(ctx context.Context, id string) (*Report, error)
    List(ctx context.Context, input ListReportsInput) (*ReportPage, error)
    Update(ctx context.Context, id string, input UpdateReportInput) (*Report, error)
}

type VersionService interface {
    Create(ctx context.Context, reportID string, input CreateVersionInput) (*ReportVersion, error)
    Get(ctx context.Context, reportID string, version int) (*ReportVersion, error)
    List(ctx context.Context, reportID string) (*VersionPage, error)
    Apply(ctx context.Context, reportID string, version int, command EditCommand) (*EditResult, error)
    Validate(ctx context.Context, reportID string, version int) (*ValidationResult, error)
    Descriptor(ctx context.Context, reportID string, version int) (*ComponentDescriptor, error)
    ExportDQL(ctx context.Context, reportID string, version int) (*DQLExport, error)
}
```

### 7.3 Preview/publication SDK

```go
type PreviewService interface {
    Execute(ctx context.Context, reportID string, version int, input PreviewInput) (*PreviewResult, error)
}

type PublicationService interface {
    Publish(ctx context.Context, reportID string, version int, input PublishInput) (*Publication, error)
    Unpublish(ctx context.Context, reportID string, input UnpublishInput) (*Publication, error)
    Rollback(ctx context.Context, reportID string, version int, input PublishInput) (*Publication, error)
}
```

SDK rules:

- all mutation inputs carry expected `etag` or draft revision
- errors use typed codes suitable for Forge forms and notifications
- pagination, filtering, and sorting are SDK types
- diagnostics are stable SDK DTOs, not raw Datly error strings
- secrets are write-only and never returned resolved
- SDK versioning is independent from internal package layout

## 8. Semantic editing commands

Forge issues intent-based SDK commands.

Component commands:

- `PatchRoute`
- `PatchConnector`
- `PatchMCPExposure`
- `PatchReportSettings`
- `PatchOutputSettings`
- `PatchTypeContext`
- `AddMCPExposure`
- `PatchMCPExposure`
- `DeleteMCPExposure`
- `EnableCube`
- `DisableCube`
- `PatchCubeConfig`
- `EnableCubeCompose`
- `DisableCubeCompose`
- `PatchCubeComposeLimits`
- `AddResourceFolder`
- `DeleteResourceFolder`
- `UploadResourceFile`
- `DeleteResourceFile`
- `AddSkillRoot`
- `DeleteSkillRoot`

Parameter commands:

- `AddParameter`
- `PatchParameter`
- `DeleteParameter`
- `AddPredicate`
- `PatchPredicate`
- `DeletePredicate`
- `PatchQuerySelectorBinding`
- `PatchCodec`

View commands:

- `AddRootViewFromSQL`
- `ReplaceRootViewSQL`
- `AddIndependentViewFromSQL`
- `PatchView`
- `DeleteIndependentView`
- `AddRelationFromSQL`
- `PatchRelation`
- `DeleteRelation`
- `PatchColumns`
- `PatchSelector`
- `PatchPartitioning`
- `PatchBatchPolicy`

Every command follows:

```text
load version
  -> check expected revision
  -> clone component
  -> apply command
  -> validate/build with Datly
  -> persist spec, descriptor, diagnostics, and derived indexes
  -> increment revision
```

Published versions are immutable.

Generic invariant-preserving edit helpers may be added to Datly if they benefit other consumers. Studio retains workflow, versioning, and authorization.

## 9. Dynamic type prerequisite

This is the highest-risk technical prerequisite.

`bootstrap.ArtifactInput` expects concrete `InputType` and `OutputType`. Reader compilation uses these for bindings, row codecs, output holders, and relation collectors.

Studio reports often have no precompiled Go contract. Required pipeline:

```text
SQL/DQL
  -> column discovery and canonical view graph
  -> input/output/view descriptors
  -> runtime or generated Go types
  -> typecatalog registration
  -> bootstrap.BuildArtifact
```

Preferred order:

1. reuse Datly transcribe discovery/generation
2. reuse `typecatalog.Catalog` and `github.com/viant/x`
3. generate deterministic contracts for published versions when needed
4. use runtime synthetic types only if they preserve all reader behavior
5. never add a Studio-only row mapper

### 9.1 Dynamic product boundary: readers only

The first dynamic Studio product is a **reader only**. Studio persists a
versioned reader definition (connector, source SQL/DQL, views, predicates,
selectors, output layout, and exposure choices), then materializes one
deterministic Datly reader with `datly transcribe get`. It does not generate a
PATCH/POST/PUT component, a writer package, mutation presence markers, writer
lifecycle hooks, or Studio-managed writes to the connected vendor database.

The dynamic package identity derives from the immutable report/version owner
identity (not a mutable username). It is required in DQL through `#package` so
Datly has stable type authority and so generated reader artifacts can be
reloaded safely. Forge edits the SDK reader definition; it never authors
arbitrary Go source or a separate browser-side query model.

Required proof:

- DQL-only component
- query/path inputs
- root list output
- one-to-many relation
- selector and predicate
- SQLite execution
- preview through public APIs
- registration through `application.Manager`
- restart from persisted artifacts

## 10. Connector-first implementation

Connector persistence and lifecycle is the first implementation milestone because compile, column discovery, preview, and publication all depend on it.

### 10.1 Connector state model

- `draft`: metadata may be edited; not available to runtime builds
- `active`: validated and eligible for compile/preview/publication
- `disabled`: retained but cannot be used for new builds
- `deleted`: soft-deleted tombstone

Activation requires a successful connection test. Disabling an in-use connector must report dependent active reports and require an explicit policy decision.

### 10.2 Authorization

Every static Studio component declares `Jwt<string,*jwt.Claims>` from the
`Authorization` header, verifies it with Datly's `JwtClaim` codec, returns 401
for a missing or invalid bearer token, and keeps authentication separate from
row authorization. Report-scoped readers attach a typed authorization handler
to `ReportId` in predicate group 3; global runtime reads attach the same policy
to the auth-context component dependency. Connector/report catalog readers use
their explicit auth-context SQL. The application-owned `studio/authorization`
handlers scope rows through `reports.owner_id` and `report_acl`; reads require
`can_view`, edits require `can_edit`, and publication/runtime operations require
`can_publish`. Writers enforce the same policy through entity lifecycle hooks
or public `Input.Init` owner validation. A `datly.yaml` reflection test fails if
any configured component has JWT authentication without one of these
authorization boundaries.

`datly.yaml` configures the same JWKS verifier endpoint used by Platform's
OAuth deployment. Platform E2E uses an encrypted RSA public key and signed
bearing tokens for isolated testing; Studio tests use the equivalent ephemeral
RSA verifier fixture. The connected Datly v1 standalone MCP configuration
accepts MCP authorization policy metadata but does not expose Platform's legacy
`OAuth2ConfigURL`/`IssuerURL` configuration fields. Deployments that require
OAuth-protected MCP transport must provide that bridge at the host layer until
the v1 standalone configuration exposes the equivalent settings.

### 10.3 Connector schema

The executable schema is maintained only in [`schema/schema.ddl`](schema/schema.ddl). MySQL Endly setup uses that file directly, while SQLite unit tests derive their normalized DDL from it through `schema.ApplySQLite`.

Schema rules:

- `component_spec_json` is the canonical executable snapshot
- authored SQL/DQL is retained and never overwritten by generated text
- view/field/parameter/predicate rows are derived searchable indexes
- MCP exposures mirror route-level `spec.MCPExposure` metadata
- cube rows mirror `spec.ReportSettings` and normalized compose limits
- resource files, folders, and skill roots rebuild `spec.Settings.MCPFolders` and the generation resource store
- derived rows can be rebuilt from the spec snapshot
- published versions are immutable
- publication points to one immutable version
- runtime generation activation is recorded separately from publication intent
- resolved secrets never enter report/version/generation rows

## 12. Preview and publication

### 12.1 Preview

Preview is isolated from the active manager.

Possible implementations:

- direct reader preview for fast data execution
- ephemeral runtime/application preview when route binding, HTTP output, auth, MCP, or OpenAPI behavior matters

Preview resources are closed after execution and never registered into the active generation.

### 12.2 Publication

Publication flow:

1. verify ACL and immutable version
2. perform strict compile
3. write pending publication intent
4. load all active publication intents
5. open required connector generation
6. compile all active components
7. return one `application.Build`
8. call `application.Manager.Reload` with monotonic revision
9. mark generation and publications active after successful swap
10. preserve previous generation if staging fails

Old connector/resources are retired only after Datly/application confirms request lifetime safety.

## 13. Forge UI

Forge is the UI framework; the Studio SDK is its only operation boundary.

Useful Forge primitives:

- `DraftFormSpec`
- editable resource tables
- `TreeEditorSpec`
- `WizardSpec`
- `StatusWorkflow`
- `HistoryDiffSpec`
- `PermissionBoundarySpec`
- `DataStateBoundarySpec`
- responsive grids and notifications

References:

- `github.com/viant/forge/backend/types/presentation_primitives.go`
- `github.com/viant/forge/backend/types/workflow_primitives.go`
- `github.com/viant/forge/backend/types/model.go`

Initial windows:

- Connector catalog/editor/test workflow
- Report catalog
- Report editor with Overview, SQL/DQL, Views, Parameters, Predicates, Selectors, Cube, MCP, Skills, Preview, Diagnostics, Versions, and Permissions tabs
- View/relation tree editor
- Generated input preview form and result table/tree
- Version diff and publish/rollback workflow

Forge receives SDK DTOs. Raw spec JSON may be shown only in an advanced inspector.

### 13.1 Initial Studio shell and identity modes

`ui/` is the application-owned Forge composition. It begins with the live
Reports and Connectors catalog surfaces and deliberately calls only the public
SDK operation names (`reports.list`, `connectors.list`) through `StudioAPI`.
The browser does not know about SQL, DQL, generated packages, or the removed
control implementation.

The checked-in public configuration has two mutually exclusive modes:

- `development`: requires a loopback API URL and an explicit development
  subject. The client sends that subject in `X-Studio-Development-Subject`.
  A server may honor it only when its own deployment configuration explicitly
  enables local development mode.
- `authenticated`: requires public OIDC issuer, client ID, audience, and
  scopes. The browser obtains a bearer from the host token provider and sends
  it as `Authorization`. It never contains a client secret, JWT, or a
  development identity.

The Studio API host remains the authentication trust boundary: it validates the
bearer before adapting its principal to the SDK authorizer. Client-side mode is
UX configuration, never authorization.

`sdk/httptransport.Gateway` is the shared HTTP adapter for this boundary. It
accepts only `POST /v1/studio/sdk/<operation>`, has a closed operation/output
catalog, and invokes `sdk.Transport` using JSON SDK DTOs. Its authenticated
mode takes a host verifier callback; the existing Datly JWT validator is the
appropriate verifier authority when this is mounted with the Datly runtime. A
verifier must place a nonempty verified `sdk.Principal` in request context;
otherwise the gateway rejects the request. The SQL SDK transport applies that
principal to report and connector catalog reads using the same owner-or-ACL
(`can_view`) rule as the generated Datly readers. Mutation permission remains
the responsibility of the configured SDK authorizer.

## 14. Datly 1.0 prerequisites and extensions

### 14.1 Must be proven

1. stable Datly v1 branch/tag and dependency policy
2. DQL-only reader compilation through public APIs
3. dynamic input/output/view type materialization and restart recovery
4. nested relation execution through public build path
5. source-linked diagnostics
6. connector ownership across generation swaps
7. complete reload through `application.Manager`
8. OpenAPI/MCP metadata from registrations
9. serialized `spec.Component` compatibility/version policy
10. explicit DQL round-trip policy

### 14.2 Candidate Datly extensions

Add only after a proof identifies a real gap.

#### Reader build facade

A facade may safely compose transcribe, types, artifact, execution, registration, and descriptor creation without exposing reader internals.

#### Dynamic contract builder

May be required to construct deterministic contracts for DQL-only components without permanent Go code. It must support stable type identity, relations, nullability, typecatalog registration, preview/publish parity, and restart-safe manifests.

#### Component/reader descriptor

Studio needs a supported read-only descriptor containing routes, parameters, predicates, views, relations, columns, selector policies, resolved types, output slots, and MCP/report exposure.

#### Semantic editor/validator

Datly may expose invariant-preserving operations for views, parameters, predicates, and relations, returning cloned spec plus diagnostics. It must not introduce a second model.

#### Complete DQL export

Required only if users must freely switch between structured and DQL editing. Otherwise authoring modes remain explicit and export limitations are displayed.

#### Preview facade

Useful if correct binding/handler invocation cannot be achieved cleanly through current public APIs.

## 15. Reuse and replacement

| Current area | Decision | Reason |
|---|---|---|
| Connector store | Evolve first | Core dependency; adapt to v1 connector config |
| Component/version stores | Evolve | Product lifecycle remains valid |
| Publication store | Extend | Add complete generation state |
| ComponentService lifecycle | Reuse concept | Replace compiler/preview/reload implementation |
| QueryVersionService commands | Reuse concepts | Back with spec edits and SDK |
| QueryDocument | Compatibility only | Incomplete semantic model |
| CompilerService | Replace | Removed shape APIs |
| datlyruntime | Replace | Removed repository/session/operator APIs |
| preview gateway | Replace | Legacy bootstrap |
| connector Datly snapshot | Replace | Use connector set and SQLComponent |
| reload Coordinator | Replace | Use application manager |
| codegen publish branch | Replace | Use v1 generation/type APIs |
| e2e lifecycle tests | Rewrite runtime fixtures | Preserve business behavior |

## 16. Implementation sequence

### Phase 0: connector foundation

- migrate connector schema
- implement connector store/service
- implement connector SDK and HTTP transport
- implement create/list/get/update/test/activate/disable
- map active connectors to `bootstrap/connector.Config`
- verify secret redaction and handle ownership
- build first Forge connector window only through SDK

### Phase 1: vendor dynamic reader

- define the SDK reader-definition contract against an active connector
- author/store reader DQL and resource SQL as versioned report content
- materialize only `transcribe get` reader artifacts
- validate predicates, selector policy, output layout, and authorization before preview
- exercise vendor reader preview through the SDK, then expose optional HTTP/MCP/cube behavior

### Phase 1: Datly capability proof

- compile DQL through `transcribe.Compiler`
- discover columns and materialize contracts
- build artifact and reader
- preview with inputs
- publish through `application.Manager`
- restart from persisted artifacts
- document/implement missing generic Datly APIs

Exit: no Studio-specific reader/runtime is required.

### Phase 2: persistence v2

- add canonical spec/type/descriptor columns
- add view/field/parameter/predicate/cube/generation tables
- enforce immutable publication versions
- migrate/analyze legacy rows

### Phase 3: Datly adapter

- compile adapter
- type materialization adapter
- descriptor adapter
- preview adapter
- application generation builder

### Phase 4: SDK and semantic commands

- report/version SDK
- semantic command SDK
- validation/preview SDK
- publication/runtime SDK
- typed errors and diagnostics

### Phase 5: Forge Studio

- report catalog
- structured editor
- SQL/DQL editor
- diagnostics and preview
- version diff
- publish/rollback

### Phase 6: hardening

- ACL enforcement
- connector rotation
- generation rollback
- concurrent edits
- observability
- load tests and recovery tests

## 17. Acceptance criteria

### Connector foundation

- connector CRUD is available only through SDK
- connection testing does not expose secrets
- only active connectors are eligible for Datly builds
- disabling reports dependent active publications
- connector handles close safely after generation retirement

### Compilation and reader

- SQL, DQL, and structured modes produce canonical spec
- diagnostics have stable code, severity, message, path, and span
- root, independent, related, and derived views execute
- selectors, predicates, pagination, ordering, projection, codecs, and nullability match Datly
- unsupported DQL export is explicit

### Lifecycle

- draft edits do not change active runtime
- preview is isolated
- publication activates a complete generation atomically
- failure leaves the previous generation active
- rollback activates a validated immutable version
- restart reproduces the active generation from DB

### SDK/Forge

- every Forge operation uses `datly-studio/sdk`
- SDK supports optimistic concurrency and typed errors
- users can edit parameters, predicates, views, relations, selectors, and report settings
- diagnostics map to source or structured controls
- preview forms derive from compiled descriptors
- version and generation status are visible

## 18. Open decisions for the proof

1. Can Datly produce restart-safe dynamic contracts without permanent Go code?
2. Is `transcribe/compile.Reader` a stable public API for view-from-SQL commands?
3. What is the serialized spec compatibility contract?
4. Will full spec-to-DQL export be implemented?
5. How are connector handles retired after generation swap?
6. Which reader metadata belongs in a stable descriptor API?
7. When are report/cube-derived components composed?
8. Does preview use direct invocation, ephemeral application, or both?

## 19. Final rule

The redesign is successful when Forge expresses intent through the Studio SDK, Studio persists governance and immutable Datly component versions, and Datly 1.0 exclusively owns compilation and execution semantics. A published database state must deterministically rebuild the same Datly application generation after restart.
