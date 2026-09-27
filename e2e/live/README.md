# Live reader acceptance

This read-only smoke checks one existing deployment. It never signs in, stores
credentials, publishes a component, or changes Studio data. Supply a short-lived
ID token for a user allowed to list SDK namespaces and execute one already
published reader. Keep the token out of shell arguments, files, and logs.

Set these variables to deployment-owned HTTPS URLs (loopback HTTP is accepted
only for local checks):

```sh
export STUDIO_LIVE_UI_ORIGIN=https://studio.example.com
export STUDIO_LIVE_SDK_URL=https://sdk.example.com/v1/studio/sdk/namespaces.list
export STUDIO_LIVE_STATIC_MCP_URL=https://sdk.example.com/mcp
export STUDIO_LIVE_READER_URL=https://reader.example.com/records
export STUDIO_LIVE_DYNAMIC_MCP_URL=https://reader.example.com/mcp
export STUDIO_LIVE_READER_TOOL=records.list
export STUDIO_LIVE_READER_ARGS='{}'
export STUDIO_LIVE_EXPECTED_TEXT='"knownRowId":123'
```

Paste the raw ID token only when prompted, then run:

```sh
read -rs STUDIO_LIVE_TOKEN
printf '%s' "$STUDIO_LIVE_TOKEN" | node scripts/verify-live-reader.mjs
unset STUDIO_LIVE_TOKEN
```

The command checks 401 without a bearer on both the SDK and published reader,
exact noncredentialed CORS and no `Set-Cookie` on denials and successful bearer
responses, static SDK MCP discovery and a read-only SDK tool invocation, and
the same known reader value over dynamic HTTP and MCP. It outputs only a small
pass summary; failures do not print the token or response bodies.

This is not a substitute for deployed OAuth callback/refresh/logout and
denied-user exercises, spoken screen-reader verification, or actual 200%
browser zoom. Record those observations and an independent reviewer verdict
in `ui/UX_REVIEW.md` before production UX acceptance.
