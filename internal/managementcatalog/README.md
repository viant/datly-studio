# Portable management windows

Management resources use the existing Studio SDK MCP operations. The producer
publishes format-3 Forge windows, separately fingerprinted datasource resources,
a tool-only canonical administrative connection, and the native workspace style.
`studio/catalog.ConfigureManagementResources` lets an embedding application add
its own binary-owned definitions without a second transport or authorization
stack. Catalog discovery and each operation retain their existing server guards.

The target architecture is one Forge window and JavaScript behavior per screen,
consumed by both the native Studio shell and remote hosts. The current native JSX
screens have not yet all been consolidated; the portable windows below must not
be treated as native parity until both hosts render the same artifact.

The window sources are `windows/<screen>/shared/main.yaml` and `main.js`.
The native Connectors and Namespaces routes read these same published resources
and render them through Forge. They do not maintain separate JSX screens or action
implementations. Connector payload normalization lives in
`connectors/shared/behavior.js` and is embedded in the shared action bundle.
Other screens still require consolidation; do not count a portable replica as
completion of the single-source migration.
Actions contain presentation state and exact SDK input mappings; validation,
ownership, source revisions, probes and publication decisions stay in the SDK.
Unported navigation entries are disabled instead of launching another app.

| Screen | Current implementation |
| --- | --- |
| Connectors | Shared native/remote Forge window: server search and pagination, create/edit, YAML/SCY references, probe, activate/disable and confirmed deletion. Activation and disabling use `editableCollection` and its shared `MutationCommand` executor. |
| Namespaces | Shared native/remote Forge window: server search, create/edit/delete, viewer state, visibility/roles and MCP exposure/port. Port zero retains automatic allocation. |
| Components | Catalog/metadata; reader version selection, creation and cloning; retained source tree and code editor; DQL/file saves with source revisions; validation, preview and publication/rollback. Native structural dialogs are still pending; the structural-operation JSON control invokes the existing typed builder. |
| Overview, Schema Browser, Security, Runtime, Skills | Pending Forge ports. Their native URL actions are disabled in the published management application. |

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
