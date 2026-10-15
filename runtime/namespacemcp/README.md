# Namespace MCP listeners

`New` and `NewShared` retain the Datly catalog/source integration.
`NewHTTP` and `NewSharedHTTP` accept a host-owned MCP HTTP handler instead,
reusing the same port reservation, namespace routing, rebind, drain and shutdown
lifecycle. The HTTP factory receives the reserved endpoint. It must supply an
already namespace-scoped handler with its own authentication and authorization.
The manager does not infer identity or add business policy.

Namespace IDs are canonical 64-character lowercase hexadecimal IDs. Port zero
uses an OS-allocated loopback listener with `NewHTTP`, or the shared listener
with `NewSharedHTTP`. A positive namespace port overrides the shared policy
with a dedicated listener. Shared paths are `/namespaces/<id>/mcp`; handlers
receive `/mcp`. Credentials and protocol bodies are preserved.

`RebindWithCommit` constructs and reserves a candidate before invoking a
caller-owned state commit. If that commit fails, the candidate is released and
the previous endpoint remains active. The callback also runs when the endpoint
is unchanged. `StopWithCommit` commits withdrawal before removing its endpoint;
a rejected commit leaves it active. This supports reviewed/CAS persistence
without closing a port between validation and activation.

Commit callbacks run under the manager's control lock. Keep them bounded to
finalizing already prepared state; perform discovery and validation beforehand,
and never call manager methods from a callback. A listener-drain error after a
successful callback does not undo committed state. Rebind returns the active
new endpoint in that case; callers must report the drain failure separately.

Handlers may implement the same optional `Close`, `Reload`, `RuntimeStatus` and
`RuntimeConnection` hooks used by Datly sources. Factory, reservation or commit
failure cleans up a constructed candidate. No storage, reporting or application
identity types are required by the HTTP factory API.
