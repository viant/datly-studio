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
Connector validation and save construction live only in `connectors/shared/behavior.js`:
the native dialog imports that module, and publication embeds the exact same
function in the Forge action bundle. This is an intermediate consolidation step,
not completion of the shared-window migration.
Actions contain presentation state and exact SDK input mappings; validation,
ownership, source revisions, probes and publication decisions stay in the SDK.
Unported navigation entries are disabled instead of launching another app.

| Screen | Current implementation |
| --- | --- |
| Connectors | List, create, edit, YAML/SCY references, test and activate. Blank edits preserve saved connection fields; removal is explicit. |
| Namespaces | List, create, edit, visibility/roles and MCP exposure/port. Port zero retains the existing automatic allocation contract. |
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
