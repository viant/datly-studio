# Management skills and embedding hosts

This package supplies the generic Data Studio authoring skill. Keep its examples
independent of an embedding application's business rules, private hosts and data.

Embedding applications can supply additional skill folders through the existing
Datly `Config.MCP.Folders` field before starting the shared host. A folder may use
an embedded `fs.FS`, a stable resource URI prefix, and explicit relative skill roots
in `resource.Folder.Skills`. Declare the entrypoint directories; supporting files
are resources rather than additional skills unless explicitly listed.

The native management bootstrap preserves those folders and appends this package's
built-in folder. No separate tool registry, skill-list implementation, HTTP route,
or per-request filesystem scan is needed. Use distinct folder namespaces and URI
prefixes; do not replace the generic authoring skill with a domain-specific copy.

Keep application-specific guidance in the embedding application's repository.
For example, a host can explain its installed authorization predicate, trusted
input bindings, source conventions and validation requirements. Guidance must
describe capabilities available in that host build and must not imply that skill
discovery grants access to data or publication operations.

Use the existing MCP resource authorization and lifecycle. Qualify the embedding
binary through authenticated skill discovery and resource reads, including its
supporting references. Compilation of a folder alone does not prove that a running
management endpoint exposes it.
