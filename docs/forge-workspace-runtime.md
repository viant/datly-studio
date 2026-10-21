# Studio workspace runtime

Studio uses Forge's `WorkspaceRuntimeSurface` from `forge/components` for
lazy first visits, retained panels, visibility/inert handling, presentation and
lifecycle. The sidebar remains Studio's navigation adapter; the shared tab strip
is hidden in this layout. ResourceWindow still loads each canonical window and
Studio supplies authenticated transport, downloads, navigation admission and
unsaved-edit guards.

Visited windows retain their state when navigating within a namespace. Forge's
`PersistentWindowContent` detaches inactive content from the document, preventing
repeated form IDs from redirecting labels or automation into hidden forms. Changing
namespace replaces the whole workspace surface so retained data and drafts do
not cross that boundary. Builder identities include the component and version.

The unified runtime is currently supplied by the local sibling Forge checkout:
`ui/package.json` declares `file:../../forge`. `ui/.npmrc` uses `install-links=true`
so npm installs a package snapshot and its dependencies, rather than resolving a
second React tree through a symlink. Run `npm ci` in `ui` after updating Forge.
The Vite configuration keeps React and signals on the host's module identities.
This is deliberately a local workspace dependency, not a published Forge pin.
Replace it with a verified published revision once the shared runtime is released.
