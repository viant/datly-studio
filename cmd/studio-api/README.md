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
