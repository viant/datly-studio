# Docker reader scale exercise

This fixture follows the MySQL 8 setup in Studio's `e2e/local/system.yaml` and
the older Datly `origin/master` e2e `VENDOR`/`PRODUCT` schema and related-view
DQL (`001_one_to_many/vendor_list.dql`, `006_tree/user_tree.sql`). It tests the
Datly 1.0 Studio workflow with a five-level, 20-view reader and a 60-column
table. It is local development data only.

Start a fresh MySQL container on the e2e port and load the fixture:

```sh
docker run -d --name datly_studio_mysql -p 127.0.0.1:43306:3306 \
  -e MYSQL_ROOT_PASSWORD=dev mysql:8.4
docker exec -i -e MYSQL_PWD=dev datly_studio_mysql mysql -uroot \
  < e2e/testdata/scale_reader_mysql.sql
```

The fixture creates `reporting_e2e.STUDIO_NODE`, 20 named SQL views, and
`STUDIO_WIDE_60` with exactly 60 columns including `ID`. There are two rows
per view. The view graph has levels 1–5 with counts 1, 4, 5, 5, and 5.

For an isolated Studio catalog, initialize `.data/scale-studio.db` and start
the SDK, runtime, and UI in separate terminals from the repository root:

```sh
mkdir -p .data
GOWORK=off go run ./cmd/studio-migrate -command init-studio \
  -dsn 'file:.data/scale-studio.db?cache=shared'
GOWORK=off go run ./cmd/studio-api -address 127.0.0.1:8080 \
  -dsn 'file:.data/scale-studio.db?cache=shared' -subject scale-review
GOWORK=off go run ./cmd/studio-runtime -conf e2e/local/scale-runtime.yaml
cd ui && npm run dev
```

The default browser config names another development subject. For a local
visual pass, temporarily set `ui/public/studio-config.json`'s development
subject to `scale-review`, then restore it afterward. With both servers ready,
seed the catalog and verify every published dynamic MCP variant:

```sh
node e2e/testdata/seed_scale_studio.mjs
node e2e/testdata/verify_scale_mcp.mjs
GOWORK=off go run ./e2e/scale/verify_bff_mcp.go
```

The seed creates two reader components through Studio SDK Reader Builder
commands and a grouped probe that exposes reader, Cube, and Cube Compose MCP
tools. It adds optional root/row ID predicates to the hierarchy and wide
readers, validates and publishes them through the SDK. The verifier calls all
five live tools and checks hierarchy nesting, 60-column output, grouped rows,
and composed rows. It also checks that MCP advertises typed `RootID` and
`RowID` inputs and that each filter selects exactly one row while preserving
the deepest child and all 60 columns. Those MCP argument names are the DQL
field names; the HTTP query names are `rootId` and `rowId`. The seed needs a
fresh catalog because report slugs are unique.
The BFF check wraps the already-running MCP listener with an opaque local
session, rejects a cookie-less request, and reruns those same five calls
through the authenticated proxy. It does not require a deployed identity
provider; it verifies local proxy and tool-routing behavior only.

The full seed-plus-verifier sequence passed again on 2026-09-25 against a
newly initialized SQLite catalog and the Docker database, not only against
the original manually authored catalog.
The five-tool authenticated BFF check also passed against the published
Docker fixture on 2026-09-25.
On 2026-09-26, the current dynamic runtime was started against the retained
catalog and Docker fixture. Direct MCP and authenticated BFF MCP each listed
and invoked all five tools successfully, returning two hierarchy roots,
60 columns in the wide reader, two Cube rows, and two Cube Compose rows.
Later on 2026-09-26, a new `scale-filter-studio.db` catalog and
`scale-filter-runtime.yaml` on ports 8084/8093 passed the updated seed and
both direct and authenticated BFF verification, including the two live
filtered reader calls. A first call using the HTTP query name (`rootId`) was
correctly rejected by MCP; its schema publishes `RootID` instead.
The retained Docker-backed catalog was started again after the native warmup
route work on 2026-09-26. All five dynamic tools passed direct and
authenticated BFF MCP invocation again, including one-root hierarchy and
one-row, full-60-column filters.
To repeat that isolated filtered run after loading the MySQL fixture, migrate
`.data/scale-filter-studio.db`, run `cmd/studio-runtime` with
`e2e/local/scale-filter-runtime.yaml`, and run `cmd/studio-api` on port 8083
with the same database plus `-dynamic-http-url http://127.0.0.1:8084`,
`-dynamic-mcp-url http://127.0.0.1:8093`, and
`-dynamic-admin-token datly-studio-local-runtime`. Then use
`STUDIO_SCALE_API_URL=http://127.0.0.1:8083` for the seed,
`STUDIO_SCALE_MCP_URL=http://127.0.0.1:8093/mcp` for the direct verifier, and
`STUDIO_SCALE_RUNTIME_MCP_URL=http://127.0.0.1:8093` for the BFF verifier.

Observed on 2026-09-25: Studio's schema catalog found all 20 SQL views and all
60 physical columns. `versions.inspect` reported 20 component views spanning
five levels, and runtime validation plus exact-version preview passed against
the MySQL connector. In the browser, the compact Views block exposed a
searchable lineage catalog; searching `STUDIO_VIEW_20` returned one level-5
view and opened its five fields. The wide view showed 60 fields in three
20-row pages. A metadata fallback now fills the Output catalog for `SELECT *`
views and places the primary key first; Output showed 100 columns across the
20-view graph and 60 columns for the wide reader.

Static SDK MCP invocations use verified JWT test fixtures. Run the native
route packages and their HTTP/MCP/OpenAPI tests with `GOWORK=off go test
./studio/...`; `internal/dependencylink` also checks that selected MCP
declarations match linked Datly components.
For the native connector catalog against this Docker fixture, set
`STUDIO_SCALE_MYSQL_DSN` to the fixture DSN and run
`GOWORK=off go test ./cmd/datly -run TestSelectedStudioStaticComponentsBootstrapTogether -count=1`.
That opt-in branch probes the Docker MySQL connector, executes a bounded
transient SQL test, verifies all 20 SQL
views and all 60 wide-table columns through signed static HTTP, direct SDK
MCP, and authenticated BFF MCP. It also previews the two complete 60-column
rows through the native exact-version SDK route and validates that wide reader
through the native runtime-contract SDK route. It also runs a transient
60-column view test through signed HTTP and both SDK MCP paths. Without
the variable, the same test uses an isolated SQLite catalog and requires no
Docker service.
