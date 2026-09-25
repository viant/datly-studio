# Auth-context mock (local testing only)

This command serves fixed synthetic auth contexts for testing generic remote
handlers. It is not an identity provider or a production authorization service.
It listens only on a literal loopback IP and keeps no credentials, files, or
database. Requests name a synthetic principal and tenant explicitly; they do
not authenticate a user.

From the Datly Studio repository root:

```sh
go run ./cmd/auth-context-mock
```

Default address: `127.0.0.1:8098`. `-address 127.0.0.1:0` selects a free port
for local tests. `go run ./cmd/auth-context-mock -h` lists the command flags.

HTTP example:

```sh
curl -X POST http://127.0.0.1:8098/context \
  -H 'Content-Type: application/json' \
  -d '{"principalId":"alice","tenant":"demo","resource":"example","action":"read","correlationId":"request-1"}'
```

The MCP streamable endpoint is `http://127.0.0.1:8098/mcp`; call the
`auth_context` tool with the same JSON arguments. Both transports use the same
fixture. A successful response has `context.userId`, `context.tenant`,
`context.roles`, `context.exposures`, `context.allowedEntities`, and
`context.validUntil`; `correlationId` is echoed when supplied. MCP returns the
same envelope in `structuredContent` and a JSON text content item.

To test header passthrough, start a separate mock instance with
`go run ./cmd/auth-context-mock -address 127.0.0.1:8099 -require-authorization`.
This opt-in mode requires the exact synthetic header `Authorization: Bearer
mock-alice` or `Authorization: Bearer mock-bob` on **every** HTTP and MCP
transport request, including MCP initialization and tool discovery. The
requested `principalId` must match the header's fixture principal. There is no
OAuth exchange or automatic refresh.

```sh
curl -X POST http://127.0.0.1:8099/context \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer mock-alice' \
  -d '{"principalId":"alice","tenant":"demo"}'
```

For an MCP client, configure the streamable endpoint
`http://127.0.0.1:8099/mcp`, send `Authorization: Bearer mock-alice` as an
outgoing header on initialization, discovery and every `auth_context` call,
then call `auth_context` with `{"principalId":"alice","tenant":"demo"}`.
To test a nonstandard header, start with `-authorization-header
X-Mock-Authorization` and map the client's outgoing header option to that
name. Supplying only the default `Authorization` header then returns 401.

The only tenant is `demo`. `alice` has project IDs `[101,102]`; `bob` has
`[103]`. Unknown principals or tenants fail. Optional `-scenario` values are
`normal` (default), `deny`, `malformed` (invalid expiry), and `delay`. Delay is
bounded to 0–2 seconds with `-delay` (default `250ms`). Press Ctrl-C for
graceful shutdown.
