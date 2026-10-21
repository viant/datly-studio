# MCP tool-call audit

The unified Studio host enables auditing by default on both admin and namespace
MCP listeners. `tools/call` requests are captured; discovery protocol messages
such as `initialize` and `tools/list` are not tool executions.

The canonical schema includes `tool_call_audit`, namespace/component
`audit_enabled` flags (default true), and `tool_audit_settings`. Existing stores
must explicitly apply the version 24 additive migration under
`schema/migrations/` after backing up. Startup verifies existing stores and does
not silently alter them.

## Permissions and controls

- `studio.sdk.audit.status` reads the selected namespace, component, or tool's
  configured and effective capture state.
- `studio.sdk.audit.configure` accepts `namespaceId`, optional `componentId` or
  `toolName`, and `enabled`. Namespace owners and delegated manager roles can
  manage their namespace and its components/tools. Component owners or component
  ACL editors/publishers can manage that component and its tools. There is no
  public global switch. Grants are re-evaluated on every invocation.
- `studio.sdk.audit.export` generates CSV on the backend using the same predicates
  and permissions, across keyset pages. It exports metadata (no captured bodies),
  uses Datly v1’s native output plan backed by Structology CSV, and escapes
  spreadsheet formula cells. Exports cap at
  10,000 rows / 8 MiB and return `truncated` plus `nextId` for continuation.
  The Audit window’s **Export CSV** button downloads the returned file.
- `studio.sdk.audit.list` requires the independent exact role configured as
  `Audit.ViewerRole`, default `AUDIT_VIEWER`, plus access to the selected namespace.
  Management alone does not grant payload access. Audit viewing alone does not
  grant permission to change capture. An unscoped request reads only unscoped
  admin events, never other namespaces.

The Forge **Audit log** window provides tool, caller, status and date/time range filters (local-time pickers, UTC on the API),
keyset pagination, an event detail dialog and a namespace capture checkbox.
The Components catalog has a **Component audit settings** action with an
**Audit enabled** checkbox. Per-tool settings are also exposed through MCP.

Effective capture requires the namespace flag, component flag (for an active
published tool) and tool override all to be enabled. An absent override means
enabled. These settings persist immediately and do not require republishing or
restarting. Control changes create an `audit.configuration` event even when they
disable their scope. Storage failures preserve capture instead of disabling it.

## Payloads and operational limits

Captured JSON/SSE is sanitized before storage, including JSON nested in MCP
text content. Password, secret, token and authorization fields are redacted.
Nonempty captured bodies are gzip-compressed by default. Response previews are
capped at **32 KiB before compression**; `response_truncated` identifies capped
or omitted responses. The actual response sent to the client is unchanged.
`response_original_bytes` remains nullable for compatibility and is intentionally
neither written nor read by the application.

The raw capture buffer defaults to 1 MiB. Larger responses retain event metadata
and omit the body, avoiding unsafe redaction of partial JSON. Oversized MCP
requests retain metadata (with a placeholder name if the transport supplies no
tool name). Bodies that cannot be safely parsed are omitted. Application-specific
sensitive fields may require a stronger host-supplied sanitizer.

The request path only captures bounded bytes and enqueues a record. Verified
identity lookup, scope-setting lookup, redaction, gzip and the **generated Datly
v1 POST writer** run on the worker. The private DQL writer omits
`response_original_bytes` entirely. Datly owns the native auto-increment identity
and transaction; the adapter does not issue INSERT SQL. Audit list filters are
native DQL predicates (user/tool/status equality, inclusive date bounds and
keyset pagination), with mandatory namespace scope outside optional predicates.

The asynchronous queue has 256 slots and a five-second write deadline. It is
best effort, not a durable delivery queue. The host emits diagnostics on full
queues or write failures; the audit window reports their process-lifetime
counters. Stop admission and drain the host before closing the logger/database.
Verified subject identity comes from the host provider; namespace runtime scope
comes from the listener rather than a caller-supplied namespace. IP is the socket
peer; forwarded headers are not trusted.

## Verification

`e2e/audit.yaml` exercises real TCP namespace capture, compression, redaction,
scoped permissions, dynamic settings, payload caps and unified-host admin MCP.
`e2e/management.yaml` includes browser acceptance for separate synthetic manager
and audit-viewer roles, persisted UI toggles and server-side denials. It uses its
own fixture database and IdP, with global namespace administration disabled.
