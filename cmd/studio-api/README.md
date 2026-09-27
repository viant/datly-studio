# Authenticated extension proxy

`studio-api` can expose one trusted extension backend through its existing BFF
session boundary. The route is disabled unless both flags are set in
`-mode authenticated`:

```sh
go run ./cmd/studio-api -mode authenticated \
  -extension-backend-url https://extension.example.com \
  -extension-upstream-prefix /api/widgets
```

A browser request to `/v1/studio/extensions/api/widgets/items/123` is forwarded
to `https://extension.example.com/api/widgets/items/123`. The configured URL
must be an HTTP(S) origin without credentials, a path, query, or fragment. The
prefix must be a canonical, non-root absolute path. Only that path and its
descendants are forwarded; paths with traversal or encoded separators are
rejected. Choose a prefix that contains only the intended extension API.

The browser sends its HttpOnly Studio session cookie to the BFF. The BFF
forwards the server-held bearer token and its existing allowlisted request
headers, including `X-Request-ID`, but never forwards browser cookies or
caller-supplied authorization. This proxy is unavailable in development mode.

## Optional same-origin static UI

`-static-root /absolute/path/to/dist` serves a built, public UI directory from
the Studio API origin. The directory must exist and be absolute. Only GET/HEAD
regular files under that root are served; `/` resolves to `index.html`.
Directory listings, hidden paths, traversal, encoded separators, and symlinks
escaping the root are rejected. SDK, session, runtime, and extension routes keep
their existing handlers and take precedence over the static fallback.

The build directory must contain only public assets and public browser
configuration. In authenticated mode, the BFF can optionally own the browser
login route as well:

```sh
studio-api -mode authenticated \
  -login-auth-url https://idp.example/authorize \
  -login-token-url https://idp.example/token \
  -login-client-id studio-web \
  -login-redirect-url https://studio.example/v1/studio/auth/callback
```

Set `-allowed-origin` to the same public origin as `-login-redirect-url`.
The login endpoints must share one trusted origin. Add
`-login-client-secret-file /absolute/private/path` only for a confidential
client; its contents are never placed in browser config or a command argument.
The registered redirect URI must match exactly. The BFF uses authorization
code + S256 PKCE, an encrypted five-minute state cookie, a fixed `/` return,
and the existing JWT verifier/issuer/audience binding before storing an
opaque HttpOnly session. The provider must return a signed JWT access token
accepted by that verifier, not only an ID token or opaque access token. TLS
termination and the callback route must be reachable at the public origin.
Opaque session persistence uses generated, server-only Datly v1 components;
these packages are not included in the public `datly.yaml` runtime package
list. Cleanup deletes expired rows in bounded batches at startup, then every
minute by default (`-session-prune-interval 0` disables background cleanup).
SIGINT/SIGTERM gracefully stop the HTTP server and session pruner before the
Datly session runtime and database close.
Without these login flags, deployments may provide their own login route and
establish the existing BFF session through `/v1/studio/auth/session`.

## Direct Datly identity-token mode

`-login-id-token` changes the authenticated host into an auth-only broker.
Its login, callback, token and logout routes are typed Datly custom handlers
served by Datly's HTTP adapter; they are not registered as MCP tools.
Studio UI must be served from the broker's public origin (the configured
`-allowed-origin` and `-login-redirect-url` origin). After authorization code
and S256 PKCE, the broker verifies the provider's signed OIDC `id_token`, stores
that token and the provider's refresh token encrypted in `bff_sessions`, and
sets an HttpOnly, Secure cookie containing only a random session identifier.
The database key is a hash of that identifier. The browser calls
`POST /v1/studio/auth/token` with its cookie to obtain an ID token in memory;
the endpoint returns neither an access nor a refresh token. It refreshes the
ID token server-side before expiry. `DELETE /v1/studio/auth/session` logs out.
Schema version 16 adds a per-session refresh lease. The private Datly lease
component atomically admits one server instance to call the refresh
endpoint and commits the encrypted replacement token set only while that
instance still owns the lease. Other instances wait briefly for the committed
replacement ID token rather than reusing the old refresh token. Providers
may rotate, preserve, or omit the refresh token in a refresh response; in the
latter two cases Studio retains the stored refresh token. Expired leases can be
reclaimed; a stale worker cannot commit after ownership changes. No lease
route or lease data is exposed over public HTTP or MCP.

Use the existing backend OAuth flags and add `-login-id-token`; set
`-jwt-audience` to the OAuth client ID because this provider's ID token is
audienced to that client. Configure a confidential OAuth client with
`-login-client-secret-file`; the secret stays on the backend. The
provider must return a new `id_token` on refresh. The browser configuration
uses `authentication.mode: identity-token`, `apiBaseURL` for static Datly,
and `mcpBaseURL` for dynamic MCP. Both Datly origins need CORS that permits
the Studio UI origin and `Authorization`, but no credentialed Datly requests.
Because Datly validates these JWTs without calling the auth broker on each
request, sign-out revokes the refresh session but cannot revoke a previously
copied ID token. The broker accepts ID tokens with at most 60 minutes remaining
lifetime on login and refresh. The browser policy now allows the provider's
one-hour default (`ID_TOKEN_TTL_SECONDS=3600`); tokens with more than 60
minutes remaining are rejected. The UI keeps the token only in page memory
and clears it after confirmed logout.
For the static Datly host, configure `CORS.AllowOrigins` with the Studio UI
origin, `AllowMethods` with `POST` and `OPTIONS`, `AllowHeaders` with
`Content-Type` and `Authorization`, and `AllowCredentials: false`. The dynamic
host accepts the equivalent policy under `MCP.CORS`; include
`Mcp-Protocol-Version` and `Mcp-Method` in its allowed headers. The example
browser config assumes the UI and auth broker are both at
`https://studio.example.com`, while the static and dynamic Datly hosts are
separate origins.

In this mode, the host does not mount BFF SDK, runtime, static MCP, or dynamic
MCP proxy routes, nor the legacy `/auth/me` and `/auth/session` exchange
routes. The UI sends `Authorization: Bearer <id_token>` directly to Datly.
Deployments must make their Datly verifier and authorization policy accept
only the intended issuer and OAuth client audience; a signing-key check alone
is not an audience check.
The sample static `datly.yaml` now sets `JWTClaims` with both values and a
required subject. Set `STUDIO_PREDICATE_PACKAGES` to a comma-separated list of
trusted, already-linked predicate package paths on static, authoring, and
dynamic hosts when embedding additional handlers. The same allowlist is used
by the native predicate catalog and Reader Builder; it does not register
types, and unlinked packages fail closed.

For generic resource-policy ACLs, `-access-user-info-url` (or
`STUDIO_ACCESS_USER_INFO_URL`) optionally resolves current roles and feature
exposures from a trusted HTTPS user-info endpoint; loopback HTTP is allowed
for local tests. The existing `-access-issuer`, `-access-audience`, and
`-access-public-key` validate the ID token first. The endpoint must return
`{"status":"ok","info":{"uid":"subject","userId":1,"accountId":21,
"roles":[],"features":[]}}`; the user and account IDs must match the
signed token. The account ID becomes the ACL tenant. No entity IDs are
inferred. Without this option, the dedicated ACL-token provider remains the
default. A runtime serving one tenant must set its `Access.Tenant` to that
account ID; mismatches deny access.
