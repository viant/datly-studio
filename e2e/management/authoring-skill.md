# Administrative MCP authoring acceptance

Run against an explicitly authorized private test namespace with an existing
component, a BigQuery fact connector and a MySQL dictionary connector. Do not
change business database data or DDL. Give a fresh agent the concrete fact table,
requested dimensions/additive measures, two dictionary table/key/name mappings,
namespace and known bounded date/entity scope. Do not give it the finished DQL
or tool payloads: it must discover those from the management MCP skill.

The published `datly-studio-authoring` skill supplies the complete example and
native-version-checked builder operation schema. The agent should explain the
existing component without editing it, create one new reader/cube, discover its
columns, validate its exact revision, release it, inspect active publication and
runtime, and use `studio.sdk.catalog.tools.list/call` to discover and execute the
new cube through the same administrative connection. Test full-key and reduced
key grouping, and preserve fact rows when dictionary labels are absent.

Capture Claude's stream-json transcript and API/MCP debug logs locally with
private file permissions and bearer redaction. Run the acceptance gate:

```sh
python3 e2e/scripts/check_authoring_transcript.py /private/path/transcript.jsonl
```

This fixture's known test scope is date `2026-10-01`, advertiser `108508`, and
at most 10 returned rows; change the checker with the fixture when changing that
scope. A missing final answer, tool error, missing lifecycle stage, corrective inline DQL edits, publication
revision mismatch, absent live catalog tool or failed/empty/unscoped invocation
fails the gate. This checks actual tool evidence, not the model's claim that it
completed the task. The agent must read an advertised skill using `resources/read`;
`skills/get` is optional when `skills/list` already returned its resource inventory.
Review fact/lookup SQL and reduced-grouping output as well;
metadata and transcript checks alone do not prove analytical semantics.

Deterministic transport coverage lives in `app/studiohost/host_test.go` (native
skill inventory/reference reads, useful authoring errors, live namespace tool
discovery and denied unknown namespaces), `internal/runtimeadmin/catalog_test.go`
(caller credential forwarding and scoped invocation), and
`studio/connectors/reader/connector_test.go` (small projected connector catalog).
The real BigQuery/MySQL agent run is a separate live acceptance check, not a CI
fixture that silently depends on a developer's credentials.
