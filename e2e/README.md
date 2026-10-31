# Datly Studio Endly acceptance suite

The replacement suite starts at `e2e/run.yaml`. Its acceptance matrix and remaining
work are tracked in [plan.md](plan.md). Legacy `local/` workflows are retained
until the replacement matrix passes; they are not the new entry point.

Workflow sources use native YAML mappings and lists, not JSON documents with a
`.yaml` extension. Keep request arguments structured as YAML; WebDriver command
expressions are string entries in a YAML list.

## Build and run locally

Install the pinned public Endly revision using the [root build instructions](../README.md#build-and-run-e2e). No sibling checkout is required.

With the prerequisites below installed, run from this directory:

```sh
endly
```

The default run executes build, SQLite setup, MCP tests, permission tests and
WebDriver tests sequentially, followed by fixture shutdown. It stops any previous
fixture before building and assigns a fresh timestamp run ID automatically.
No database deletion or manual run ID is needed for repeat runs. Build only with
`endly -t=build`.

## MCP orchestration

Run Endly's MCP server with this directory as its working directory. Load
`run.yaml`, list tasks and directory-template instances, then use `runWorkflow`.
Omit `params.runId` to get a fresh run directory automatically. If you supply an
explicit `params.runId`, use a new value for setup: preparation refuses to overwrite
an existing catalog. All generated credentials, databases and browser evidence
stay under `.data/e2e/<runId>`.

- Initial build: `tasks: build`.
- MCP group: `tasks: stop,setup,test`, with discovered `testMCP_*` tag IDs.
- Permissions: `tasks: stop,setup,test`, with discovered `testPermissions_*` tag IDs.
- Browser group: `tasks: stop,setup,setupBrowser,test`, with discovered `testWebDriver_*` tag IDs.
- Complete replacement suite: `tasks: stop,build,setup,setupBrowser,test`.
- Individual cases: use `tasks: test` with discovered `tagIds` after setup.
- Cleanup: `tasks: stop`, then close the Endly session to close its WebDriver.

Requirements: Go 1.25.8 or newer (automatic Go toolchain downloads enabled),
Node.js 22.12 or newer with npm, Python 3, a working local Google Chrome installation,
and Endly at `9ed0e3e271c3d` or newer. Oracle and SQLite use pure-Go drivers; a C compiler is not required for the
standard binaries. Run browsers as a regular user, not root. macOS and Linux are the
supported fixture hosts; Windows is not yet supported by its POSIX shell steps.
Network access is required for public Go modules, npm packages, the pinned public
Forge archive, and Google's ChromeDriver download. The workflow runs `npm ci`;
no sibling repository, preinstalled node_modules, Docker, database server, login,
or pre-existing credentials are required. Endly pins Gosh's published strict-completion
implementation; build/readiness commands opt in. WebDriver setup downloads the
official ChromeDriver matching the installed Chrome build into `.data/e2e/webdriver`
and starts it on port 4445, without replacing a system driver.

DSUnit registers SQLite, creates the consumer schema and loads the dictionary
fixtures. Business mutations run through the native Studio MCP tools or the
browser. DSUnit checks persisted effects. The fixture uses signed synthetic
identities, a real native Studio runtime, isolated namespace MCP listeners,
and a trusted browser proxy. It does not implement substitute business handlers.

The example predicate package is `github.com/viant/datly-studio/e2e/predicates`.
The fixture links it and configures `STUDIO_PREDICATE_PACKAGES` using the package
string; the DQL case verifies discovery and filtered execution.

The replacement is still under validation. An authored case or a successful
zero-assertion operation is not acceptance evidence. See [results.md](results.md)
for observed runs and outstanding failures.

The browser fixture exposes `/e2e/login` on its loopback origin for switching
between the allowlisted synthetic IdP users. Credentials remain on the fixture
server; the page selects a test identity, and the backend still verifies its JWT
and mock user-info response. This route is not linked into production mains.
Case `testWebDriver_ui_namespace_admin` creates resources as owner, manages
namespaces as `namespace-admin`, verifies read-only controls for the differently
cased role, and proves namespace administration does not grant component MCP
execution. This case owns its namespace and consumer table.

## Linked codec regression

`regression/cases/mcp/015_package_codec/test.yaml` owns the codec lifecycle case.
The fixture links `e2e/codecs` and approves its synthetic `Tenant` codec using
`host.RegisterCodecPackage`. The case imports DQL through the SDK, validates it,
normalizes the tenant through its linked codec before binding the linked
`CustomerScope` predicate, publishes the revision and invokes it through namespace MCP. It checks exact normalized preview rows and published row values, rejects invalid
codec input in both preview and published execution, verifies the codec is absent
from the authorization-predicate picker, and rejects registering it as a predicate. Consumer rows are
hydrated during shared DSUnit setup; the case imports its own reader afterward.
Run it through the same directory-template MCP group as the predicate cases.
For a focused run after building the fixture, use path selection so unrelated
permission/browser groups are skipped:

```sh
endly -selector-mode=path -r=run -t=cleanupPrevious,setup,test -i=testMCP_package_codec
```

## Namespace composition regression

`regression/cases/mcp/016_namespace_composition/test.yaml` creates its namespace
after fixture startup, imports and publishes its own reader, then calls the linked
`composition.count` handler through namespace MCP. Before publication, preview verifies that the host dependency resolver receives
the selected namespace. The handler invokes that generation's reader in memory
and checks four synthetic consumer rows. The case
also checks anonymous denial and that a second, generic namespace cannot invoke
the linked handler. The fixture resolves the extension from the persisted
namespace identity; no namespace IDs are configured before startup. All consumer
rows are hydrated by shared DSUnit setup.

```sh
endly -selector-mode=path -r=run -t=cleanupPrevious,setup,test -i=testMCP_namespace_composition
```

Build the fixture first (`endly -r=run -t=build`) after changing linked Go code.
This is a generic component-composition test; application reporting acceptance
belongs to the embedding application's E2E suite.

## Component creation and parameter authoring

`testWebDriver_component_create` starts in **Components → New component** and
checks immediate navigation to Root SQL. It creates an internal reader, exercises
input and output parameter add/edit/remove, renames the `output/view` holder,
and adds `output/status` and `output/metrics`. Result-column editing is checked
separately, including an exact public name and removal from the JSON response.
The case checks real preview data, validates the draft, and verifies catalog
persistence. Source kind and source location are labeled **Kind** and
**Kind source** in the parameter forms.

`webdriver/016_ui_input_predicates` is the dedicated UI-only input/predicate lifecycle test.
It creates its namespace, connector, component, and inputs through controls and Root SQL;
it never imports or pastes DQL. It uses MCP-backed view/dimension/predicate lookups,
then edits selectors, expressions, handlers, and input names, and removes predicates and inputs.
Read-only MCP preview assertions verify exact returned IDs after each change.

The same case covers MCP codec discovery, required `JwtClaim` with its absolute claims type,
401 error metadata, a custom codec selected by absolute Go package/type path, invalid codec input,
error-setting edits, codec removal, and denial of codec discovery to an unauthorized principal.

## Dedicated discovery and namespace-manager UI acceptance

```sh
cd e2e
endly -r=management
```

`management.yaml` builds the real fixture and production UI, uses isolated
catalog/source databases, and runs two dedicated cases:

- `management/discovery.yaml`: MCP product/schema metadata, explicit UI schema
  loading, editable project/schema lookup lists, persisted activation/probe
  preservation, and table/column isolation across SQLite attachments.
- `management/namespace-managers.yaml`: UI owner delegation, manager updates to
  namespace settings and read grants, another namespace remaining read-only,
  UI revocation, and server-side denial after revocation. The fixture starts
  with the global namespace administrator role disabled.

This suite uses HTTP/MCP/IdP ports 19780/19781/19782/19790/19795 and ChromeDriver
4446, separately from the default suite and POC. It does not stop other fixtures.
Use a fresh run ID for setup. After setup, rerun only `discovery` or
`namespaceManagers` in the same Endly session; stop with `-t=stop` using that
session's run parameters if a failure interrupted the pipeline.

## Namespace package settings and component resource opt-in

`component-options.yaml` is the dedicated isolated real-UI/native-MCP suite.
It uses separate ports 21780/21781/21782/21790/21795/21796 and ChromeDriver 4449.
It verifies package string-list add/remove and persisted arrays, default-off scope,
namespace-admin-only add/change/removal, denial to owners and delegated managers,
component opt-in and opt-out through the catalog, and rejection of publication
when its namespace starts requiring a declared authorization predicate.

The external BigQuery API is a loopback test fixture; Studio's normal native
reader and actual BigQuery driver perform the work. Counters prove zero dry-run
jobs for opt-out, and a dry run with no business query when the limit is exceeded.
The fixture has no real credentials or customer data.

Run through Endly MCP: load `component-options.yaml`, run `build`, then
`setup,browser,componentOptions,stop`. Supply `appPath` (checkout root), a fresh
`runId`, and `goWork` (`off` to use published pins, or an explicit workspace for
qualifying local dependency commits). Local source qualification does not upgrade
a remote deployment. The dedicated acceptance run passed 76 assertions using the
local dependency workspace; release binding checks also have native Go coverage.
