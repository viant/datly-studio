# Portable management windows

Management resources use the existing Studio SDK MCP operations. The producer
publishes format-3 Forge windows, separately fingerprinted datasource resources,
a tool-only canonical administrative connection, and the native workspace style.
`studio/catalog.ConfigureManagementResources` lets an embedding application add
its own binary-owned definitions without a second transport or authorization
stack. Catalog discovery and each operation retain their existing server guards.

The Studio shell and remote hosts consume the same Forge window metadata and
shared widgets. Native embedding exports are thin compatibility adapters. The
shell retains authentication transport, navigation and download capabilities.

The window sources are `windows/<screen>/shared/main.yaml` and `main.js`.
The native Connectors and Namespaces routes read these same published resources
and render them through Forge. They do not maintain separate JSX screens or action
implementations. Connector payload normalization lives in
`connectors/shared/behavior.js` and is embedded in the shared action bundle.
Actions contain presentation state and exact SDK input mappings; validation,
ownership, source revisions, probes and publication decisions stay in the SDK.
Navigation entries use the local metadata host instead of launching another app.

| Screen | Current implementation |
| --- | --- |
| Connectors | Shared native/remote Forge window: server search and pagination, create/edit, YAML/SCY references, probe, activate/disable and confirmed deletion. Revision-guarded activation and disabling are available on each row. |
| Namespaces | Shared native/remote Forge window: server search, create/edit/delete, viewer state, visibility/roles and MCP exposure/port. Port zero retains automatic allocation. |
| Components | Catalog/metadata; reader version selection, creation and cloning; retained source tree and code editor; DQL/file saves with source revisions; validation, preview and publication/rollback. The expert source editor also offers structural-operation JSON through the typed builder; ordinary authoring uses the Reader Builder widgets. |
| Overview, Runtime, Components catalog | Native routes consume the metadata sources described below. |
| Explorer | Shared native/remote metadata: connector/schema/table discovery, SQL editor, columns/result tabs, root creation and CAS-guarded cross-connector subviews. |
| Skills | Metadata namespace drafts/files, revision history, CAS, publication, and component-owned inventory. The optional host renderer is a named Forge host-content slot. |
| Security, Reader Builder | Shared Forge widgets composed by metadata, including policy review, SQL/graph authoring and protected dialogs. |

Verification:

```
node internal/managementcatalog/actions.test.mjs
go test ./internal/managementcatalog ./studio/catalog ./runtime/managementwindow
cd ui
npm exec -- vitest run src/ManagementWindows.render.test.jsx
```

Rendered tests mount the published Forge dependency, change real CodeMirror
text, and assert operation inputs through its datasource transport. They use
synthetic records; they are not a live deployment or production persistence test.
Native operation transport and persistence tests remain under `studio/` and
`sdk/transport/sql/`. No production business writes are needed for these tests.

## Authoring and parity rules

Prefer Forge workflow primitives (`draftForm`, `queryToolbar`,
`editableCollection`, `mutationCommand`, `statusWorkflow`) before adding action
JavaScript. Command validation, confirmation, pending/indeterminate state,
duplicate-submit prevention and post-write refresh belong to Forge's shared
executor. Keep only application-specific payload preparation and orchestration
in shared actions. Do not recreate a primitive inside Studio or maintain a native
screen beside a portable lookalike.

Verify commands through an actual `WindowContent` render with the published
resource shape. A mocked command context can hide missing datasource mounts.
Writer contexts must be mounted lazily without executing on page open. Test
single dispatch, exact SDK inputs, conflict/error behavior, and refresh, then
compare native and remote hosts using the same resource and styles. Rendering
or API tests alone are not proof of visual parity.

## Native consolidation progress

The native shell now renders the published Overview, Connectors, Namespaces,
Runtime, Explorer, Skills, and Components catalog (`window://datastudio/component-catalog`) windows.
The expert `components` resource remains a separate Forge version/source editor.
Input, output and predicate authoring dialogs also use metadata windows. Reader
Builder, Security, policy review and the preview harness now use shared Forge
widgets composed by metadata. Native embedding exports are compatibility wrappers;
they do not maintain a second screen renderer. Authentication loading, retry and
sign-in states use a bundled local metadata window, so opening the sign-in screen
does not require an authenticated resource read.

Catalog tables use Forge's `table.presentation: catalog`; summary lists use
`workspaceSummary`. The metadata owns field mappings, row commands, lookup
sources and SDK arguments. Browser download and navigation are host capabilities.
Creation and import dialogs hold the namespace/edit guard until closed, and
`closeDisabledWhen` blocks dismissal while saving.

Runtime discovery declares an `mcp_protocol` datasource with `service: runtime`
and `method: tools/list`. This is a protocol operation, not a `tools/call` name.
The native transport resolves it through the signed-in, namespace-scoped runtime
MCP adapter. It rejects other services or methods and never takes a URL or token
from window metadata. Other hosts can provide `invokeManagementProtocol` for this
same explicit binding; this capability must be installed before offering the
Runtime window. Ordinary SDK datasources retain `kind: mcp_tool`.

## Migration verification

Dedicated Endly WebDriver cases exercise the real UI and verify persistence:

- `015_ui_skills`: namespace/component skills, supporting files, access policies,
  schema creation and UI publication; 77 assertions passed in the migration run.
- `016_ui_input_predicates`: input, predicate and codec creation, modification and
  deletion, including MCP-backed choices; 113 assertions passed.
- `017_ui_output_contracts`: component creation, output contracts, SQL result
  fields and preview; 49 assertions passed.

These cases do not establish that the complete WebDriver suite passes. The older
shared scenario helpers under `e2e/regression/default/` still contain selectors
from the former native screens and need consolidation with the dedicated cases.
Rendered tests and a production build pass against the pinned published Forge
package. The full Go suite also has independently reproduced baseline failures
in distributed browser identity refresh and warmup recovery; the catalog,
namespace skill, Studio API and Datly bootstrap suites pass.
