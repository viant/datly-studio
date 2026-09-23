# Datly Studio UI

This is the app-owned React shell composed with Forge. It never talks to SQL,
Datly DQL, generated component packages, or a legacy control API. Every request
uses an SDK operation name and SDK DTO through `StudioAPI`.

`public/studio-config.json` is local-development configuration only. It may use
an explicit development subject and only a loopback API URL. The server must
accept `X-Studio-Development-Subject` only in its explicitly configured local
development mode.

For an authenticated deployment, deploy `config/authenticated.example.json`
with real public issuer/client/audience values as `public/studio-config.json`.
Do not place an OAuth client secret or bearer token in this file. The host gives
the UI an access-token function through `window.studioAccessToken`; the Studio
server verifies that bearer before adapting it to the SDK authorization layer.

Run `npm install`, `npm test`, and `npm run dev` from this directory.

## ACL UX review window

Open **Security → Permissions → Open ACL UX review**, or visit
`/acl-review.html` on the UI server. The production UI build includes this page.
It uses the same `ResourceAccessEditor` as live permission management, inside a
real iframe viewport (1200, 768 or 390 pixels).

Choose component or skill and inspect editable, read-only, denied, unavailable,
loading, empty and revision-conflict states. In the conflict state, edit and save
to trigger recovery. Reset restores the fixture. All identities and resources are
synthetic; saves stay in memory and issue no API requests. Checklist selections
are temporary reviewer notes, not persisted approvals or automated test results.
