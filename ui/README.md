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
