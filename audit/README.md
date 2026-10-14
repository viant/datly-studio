# Optional MCP tool-call audit

Auditing is disabled when `runtime/host.Config.ToolCallAudit` is nil. The
embedding host explicitly provisions `schema/tool_call_audit.sql` for MySQL
or `schema/sqlite/tool_call_audit.sql` for SQLite, then supplies a logger:

```go
logger := audit.New(audit.Config{
    Enabled: true,
    QueueSize: 256,
    CompressionThreshold: 4096,
    MaxBodyBytes: 1 << 20,
    Sanitize: audit.RedactJSON,
}, audit.SQLSink{DB: auditDB})
config.ToolCallAudit = logger
```

Stop HTTP admission and drain the runtime before calling `logger.Close(ctx)`;
close the audit database only after that succeeds. A cancelled drain leaves
the worker running; retry the drain before releasing its database.

The queue never waits for persistence. Full queues drop records; configure
`OnDrop` and `OnError` to observe loss. These callbacks must be quick and must
not block. Inserts have a five-second timeout. Compression and redaction run
on the worker. Request and response bytes are bounded separately; oversized
requests bypass audit capture, and oversized responses retain metadata only.
This is best-effort audit capture, not a durable delivery queue.

`RedactJSON` handles JSON and MCP SSE JSON data frames, redacting keys that
contain password, secret, token, or authorization. Payload-specific secrets
require a host-supplied sanitizer. Nil sanitization omits bodies. Bodies and
sizes describe sanitized content; response delivery remains unchanged.

Verified identity comes from `audit.WithIdentity` or a host-owned `Identity`
callback. Never decode an unverified bearer token into audit identity. The
runtime's direct JWT authentication supplies subject and numeric user ID.
Other provider integrations must supply their verified identity explicitly.
The active OpenTelemetry span overrides supplied trace/span values when present.
Client IP is the socket peer; forwarded headers are not trusted automatically.

The internal DQL insert source is under
`dql/studio/tool_call_audit/store_write`; it is not exposed as a public MCP tool.
The current asynchronous SQL sink performs the insert; DQL generation remains
separate from runtime capture.

Run the focused Endly workflow `e2e/audit.yaml`. It invokes a real Datly reader
through its TCP MCP endpoint, gates the SQLite sink to prove nonblocking
response delivery, then verifies persisted identity, IP, compressed/redacted
request and response bodies, and disabled capture with no inserted rows.
