# Connector quick-test diagnostic regression

Run against an authenticated local Studio instance with the updated binary:

```sh
endly -r=e2e/connectors/quick-test tokenFile=/absolute/private/id-token.file connectorName=connector-quick-test-unique
```

`studioMCP` defaults to `http://127.0.0.1:18981/mcp`. Use a unique connector name
for parallel runs. The workflow checks that the synthetic secret file is absent,
creates a disposable MySQL connector, invokes the native MCP connectivity test,
asserts the actionable `secret_not_found` response and persisted probe status,
then deletes the connector. Secret resolution fails before any database connection.
The probe has a 30-second deadline. It never reads a production credential or
runs a business SQL query. If interrupted, resume the read/cleanup tasks before
reusing the same connector name.

The rendered UI regression is in `ui/src/ConnectorDialog.render.test.jsx` and
checks that Save & test saves the current DSN before probing, keeps the dialog
open with diagnostic details, clears editable credential fields, and refreshes
its optimistic revision before the next save.
