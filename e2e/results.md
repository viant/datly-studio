# Execution evidence

Date: 2026-10-13. Replacement suite validation is in progress.

## Published dependencies

- DSUnit: `c522731b506b333e76b891bab3eaf9480edfac82` (origin/master).
- Gosh: `35150777a1dc5cea718af2a034a3852df98d60a1` (origin/main).
- Endly: `80a3c216c60df6347eec03bdd9c9676a451aaf68` (origin/master),
  pinning Gosh `v0.3.1-0.20261013193746-35150777a1dc`.

All pushes were performed through IntelliJ and verified against remote-tracking
refs. Gosh package tests and focused race tests pass. Endly's focused exec/process
package and race tests pass. The full Endly test suite is not claimed green.

## Studio operations

Endly MCP session `9d2016d7-483a-464c-9d97-477b44d8abeb` uses the published Endly
build on loopback port 4989. SQLite fixture directory: `.data/e2e/endly-08`.

- `34ad7202-eea0-4be5-abab-824c60575780`: MCP group; 134 passed assertions,
  one failed assertion; stopped at builder edit payload. Corrected the authoredDql
  payload and replaced the empty-skill assertion with a DSUnit row count.
- `731d2fe1-ad85-49d7-95a6-8aa0699b4557`: predicate catalog and dual namespace
  publication/invocation; 33 passed, two failed assertions. Both runtime calls
  succeeded. Corrected draft compile-status expectations and changed isolation
  proof to assert each namespace's exact tool catalog.
- `b87c4d60-b9ac-46f9-ac64-e53a7c1be320`: permissions; 22 passed, zero failed
  assertions before runner grant rejected missing prerequisite view capability.
  Corrected the grant; a full rerun is required.
- `1b743418-f5ed-4653-a57f-e3350f74f3ce`: WebDriver blocked before assertions by
  installed Chrome 155 versus ChromeDriver 149. Added fixture-local matching-driver
  preparation; browser validation is still pending.

Production fixes uncovered so far: namespace create response omitted namespaceId
and governance fields; connector recreation with a tombstoned reserved name
returned an internal error instead of conflict. Focused tests for both fixes pass.

The full tool coverage manifest, remaining WebDriver lifecycle cases, mock-based
permission failures, standalone namespace-owned skill provider integration, and
per-exposure revisions remain unfinished. Legacy workflows have not yet been
removed. No complete-suite success is claimed.

## Later verified results

- `573f7cfa-9074-43a1-9d09-183bc1cf2c68` in session
  `4dcb144c-338d-4edb-bac4-39235c7f2277`: **succeeded, 69 passed assertions,
  zero failed**. Selected predicate catalog lifecycle, two namespace runtime
  publication/invocation/isolation, outsider rejection, and ACL role/revocation
  cases. Fixture: `.data/e2e/endly-09`.
- `aca706d0-e52b-47f1-a206-498c6d211a90`: browser namespace creation passed;
  component creation exposed missing aria-label lookup in Endly getByLabel.
  Added a parser regression and extended Endly's live browser flow; both pass,
  including the live test against ChromeDriver 155 on port 4445.
- Endly WebDriver fix published as `321c0c61fe9b324fa47b872115a667ef6ea3ac17` on top of `80a3c216`; origin/master matches.

- `508f5c2a-4b9e-4e1c-b0a4-09f5c0a2c538`: fresh full authored matrix stopped
  at resource-folder write, 119 assertions passed and zero failed. Subsequent
  independent SQLite reads reported `database is locked`. The fixture now uses
  WAL plus a five-second busy timeout on its modernc SQLite connections; this
  configuration change still requires a fresh validation run.

## Passing authored matrix

Operation `999914af-0ce5-45c7-977c-8648b4f0a7c7`, session
`442741ab-45e4-441d-ae81-abccd55e6da3`, fixture `.data/e2e/endly-12`:
**succeeded, 225 passed assertions, zero failed**, tasks
`stop,setup,setupBrowser,test`. This executes all currently authored cases:
11 MCP cases, two permission cases, and two WebDriver cases (namespace and
component creation). It uses Endly `321c0c61` and the published Gosh dependency.

The corrected SQLite configuration passed this combined run. Browser component
creation returns to the catalog; the test verifies the catalog row and DSUnit
checks its title, namespace, slug, and connector. The manifest currently maps
53 of 83 discovered tools to authored cases. This is a passing intermediate
matrix, not completion of the larger plan: the remaining 30 tools and requested
browser lifecycle/namespace-skill/exposure cases still need implementation.

## Plain CLI rerun verification

After fixing the default entry point, ran `/usr/local/bin/endly` from `e2e`
with no arguments twice in succession on 2026-10-13:

| Run directory | Result | Exit code | Duration |
| --- | --- | --- | --- |
| `20261013202825000` | SUCCESS, 16/16 checks | 0 | 17.996 seconds |
| `20261013202848000` | SUCCESS, 16/16 checks | 0 | 18.689 seconds |

Both runs built the fixture and UI, seeded SQLite, ran the 11 MCP, two permission
and two browser cases sequentially, and stopped the fixture. The CLI's 16 checks
include the startup health check plus the 15 regression cases.

The original failure came from a fixed default run ID (`studio`). Plain CLI also
expanded all root task names, causing the old group shortcuts to rerun the suite.
The root now generates a fresh run ID and has one regression entry point. Group
selection uses discovered tag IDs, so the default does not execute duplicate cases.
Logs: `/tmp/datly-studio-plain-endly-first.log` and
`/tmp/datly-studio-plain-endly-second.log`.

## Clean dependency validation

Copied tracked source and the in-progress suite to a fresh temporary checkout
layout with no sibling Forge checkout, node_modules, built UI, fixture binary,
or databases, and ran plain `endly` with a fresh npm cache. Result: **SUCCESS,
16/16, exit 0**, 58.619 seconds. Go's module cache was reused; this does not claim
an empty Go module cache. Log: `/tmp/datly-studio-clean-checkout.log`.

Forge now resolves to public commit `0597fac893974348cff3f5f3228a3a472b2f3276`
through the npm lockfile, and the e2e build runs `npm ci` automatically.

## Pure-Go SQLite probe correction

The UI connector probe exposed a transitive CGO-disabled sqlite3 stub. The probe
previously selected sqlite3 even when the requested driver was sqlite. The
regression reproduced the exact stub error; honoring the requested sqlite driver
fixed it. The running CGO-disabled fixture then returned `status: passed` for
connector ui_feature through the real native connector-test MCP tool.

Endly DSUnit now registers the SQLite dialect under sqlite (and Oracle under
oracle); the fixture requests sqlite explicitly. DSUnit package tests pass with
CGO_ENABLED=0. Earlier passing e2e counts do not cover the newly added UI policy
cases, which are still being validated.

## 2026-10-13 visual authoring and policy follow-up

- Converted 19 workflow documents from JSON syntax to native YAML; conversion
  verified decoded values and declaration order.
- Replaced DQL import in the feature/role/combined browser scenario with Schema
  Browser table selection, Add as root view, component creation and MCP settings.
- Initial policy creation now succeeds through the generated browser client.
- Feature case passed with actual returned IDs 10/20, one visible tool for an
  admitted user, zero tools and protocol denial for a rejected user. Full mixed
  sequential run remains under investigation; do not infer complete coverage.
- Added independent UI policy-edit case 006, including reopening revision 1,
  switching exposure, saving revision 2, admitting operations and revoking sales.
  This new case is authored but not yet verified end to end.
- Endly dialog-checkbox animation regression passed in live Chrome; fix committed
  as 78d39dc0 and installed in /usr/local/bin/endly (CGO_ENABLED=0). Not yet pushed.
- Initial policy/connectivity/OpenAPI Go tests pass with CGO_ENABLED=0. Resource
  editor and namespace lifecycle unit/render tests pass. Namespace admin exact-role
  helper and focused namespace authorizer/SDK tests pass.
- Wildcard condition/action resolution and explicit deny added to local viant/authz;
  full root-module race suite passes. Dependency publication/pinning and real
  wildcard UI/MCP acceptance remain outstanding.

### Verified four-case UI policy run

Endly MCP session `2379eb80-82c7-4c61-9025-4931b0ee7ef4`, operation
`fba59475-6cf7-4a3f-bcac-e61ede01c2f2`, isolated run `ui-authz-19` completed
with **120 passed assertions and zero failures**. Selected cases:
`testWebDriver_ui_feature`, `testWebDriver_ui_role`, `testWebDriver_ui_both`,
`testWebDriver_ui_policy_edit`. All namespace, connector, reader and policy
mutations used UI controls; publication and final catalog/invocation checks used
MCP. Reader creation selected a table through Schema Browser, with no DQL import.
The policy-edit case reopened revision 1, saved revision 2, admitted operations
and revoked sales without republishing the component.

This proves these four cases only. Entity scope, wildcard rules, management
policy separation, namespace admin UI, skills, cubes, complex revision editing,
Agently configuration resources and complete clean-checkout coverage remain open.

### Complex component revision case

`testWebDriver_ui_complex_revision`, run `ui-complex-05`, Endly operation
`7793ce99-8067-43ae-a499-3e868527d0f7` completed with **95 passed assertions,
zero failures**. The UI selects four dedicated tables, creates one root and three
related subviews, and adds four inputs/predicates. Revision 1 is published through
MCP; each predicate is called separately and all three nested views are checked.
The UI clones revision 2, hides the contact detail field from output, adds Region
and ContactLabel predicates (separate root/child groups), validates, and confirms
the cloned policy. MCP republishes revision 2 and verifies the new input schema,
filtered nested data, and absence of the removed output field. Original and new
predicates are also called together. No DQL import or backend builder mutation
performs the authored changes.

### Namespace connection resource implementation

Added `mcp://ns/<name>` to namespace runtime resource publication. Focused Go
integration tests read and parse the actual MCP resource into Agently MCPClient
for separate and shared listeners, reject foreign namespace URIs, and reject
reads after namespace visibility becomes private. OAuth export tests verify
provider endpoints, scoped discovery URLs, SCY encrypted-resource references and
absence of inline credentials. Public mode exports a bare connection. Endly
resource/skill protocol actions are implemented and their race tests pass; the
Endly UI acceptance case for this export remains to be added.

### Published dependencies

Endly commits through `1bdad2abd5d03c0d4bfa43709a0c6b41915e1374` were pushed
through IntelliJ to `origin/master`; the installed `/usr/local/bin/endly` was
rebuilt at that revision with CGO disabled. The native MCP resource/skill actions
pass race tests. Binary screenshot results pass JSON round-trip tests and a live
Chrome PNG decode test.

Authz wildcard commit `83350c91a2e00a6cd3a6b9b96a1d97948d11b97c` was pushed
through IntelliJ to `origin/schema-ddl-export`. Studio now pins public module
`v0.0.0-20261013220326-83350c91a2e0`; no local replacement is used.

### Wildcard UI policies

`testWebDriver_ui_wildcard_exposure` and `testWebDriver_ui_wildcard_role`, isolated
run `ui-wildcard-01`, Endly operation `b47b4ae4-c23d-470a-a5bd-5ffdb33e798e`,
completed with **59 passed assertions, zero failures** against the published
Authz dependency. Policies are created through UI with action `*` and condition
value `*`. Nonempty exposure/role facts permit real namespace MCP calls; empty
facts hide and deny the tool. The exposure case then edits the UI policy to add
an explicit execute deny and verifies discovery remains inherited while direct
execution is denied. Each case has its own namespace and consumer table.

### Complete current matrix through the default CLI entry point

Plain `cd e2e && endly` completed successfully: **23/23 checks** (22 selected
use cases plus fixture health), **165.217 seconds**, isolated directory
`.data/e2e/20261013223720000`. Log: `/tmp/datly-studio-current-full-suite.log`.
This run rebuilt the fixture/UI, ran all MCP, permission and browser directory
instances sequentially, and stopped the fixture. The build command was then
strengthened to explicitly set `CGO_ENABLED=0`; its repeat run is a separate gate.
The full requested acceptance matrix is still incomplete despite this current
matrix being green.

### Pure-Go default-run repeat

The first CGO-disabled repeat exposed a missing UI wait after connector creation:
subsequent actions could target the covered catalog while the modal was still
open. Added explicit modal-closure assertions after connector/component/policy
saves. Re-running plain `endly` then completed **23/23 checks in 154.327 seconds**,
using `.data/e2e/20261013224458000`; log
`/tmp/datly-studio-purego-full-suite-2.log` confirms the fixture build ran with
`CGO_ENABLED=0 GOWORK=off`. Runtime host, namespace manager, namespace handlers
and namespace visibility Go regression packages also pass with CGO disabled.

### UI-authored entity scope

`testWebDriver_ui_entity_scope`, run `ui-entity-02`, Endly operation
`5927361d-a651-4821-b4f9-aa37037ad40c` completed with **40 passed assertions and
zero failures**. Namespace, connector, table reader, mandatory row binding and
role/entity policy are authored through UI controls. The builder applies the
server-owned Auth binding, required ID codec and group-99 predicate atomically.
MCP publication and calls prove XYZ + customer-10 reads row 10 only; forged Auth,
AuthorizedEntityIDs and scope arguments cannot broaden the result. Customer 20,
customer 10 without XYZ, and XYZ without an entity grant are each hidden from
catalog discovery and denied direct invocation.

### Namespace administrator native write checks

Native namespace update/delete now resolve the optional exact admin role from
verified provider facts, preserving the original owner and optimistic revision
checks. The combined production-component bootstrap regression passes with a
foreign-owner namespace: wrong-case role denied, matching role updates, `get`
reports `canManage`, and delete succeeds. Invocation-scoped admin decisions are
private, expiring, subject/configuration/credential-bound and covered by tests.
The real UI admin/non-admin scenario remains outstanding; these Go checks do not
replace that acceptance case.

### Full matrix including entity-scope UI

Plain `cd e2e && endly` passed **24/24 checks in 142.700 seconds** with
`CGO_ENABLED=0 GOWORK=off`, isolated run `.data/e2e/20261013230916000`.
Log: `/tmp/datly-studio-full-entity-suite.log`. This includes the entity case's
explicit zero public input properties assertion. The fixture for this run was
built before the subsequent namespace-admin write changes; those changes have
separate native bootstrap/race validation above and still need UI acceptance.

### Namespace administrator UI acceptance

`testWebDriver_ui_namespace_admin`, run `ui-admin-02`, Endly operation
`710930ff-80d8-41f8-aa2b-2e73011b343b`, passed **50 assertions, zero failures**.
The case creates its namespace, dedicated consumer-table component and policy
through UI, publishes through MCP, and changes synthetic identities using the
fixture login page. The exact configured admin role edits another owner's
namespace and deletes an empty namespace created by that owner. The differently
cased role receives read-only UI. Owner identity and etag are checked after the
edit. Real namespace MCP calls prove the admin role does not grant consumption
while the policy-authorized reader still receives both expected rows.

### Refreshed native tool inventory

Endly `mcp/runner:listTools` operation
`022a397b-e5de-4782-8d46-d856bd937c0f` returned **84 unique tools**, no next cursor.
The manifest now includes `authz.sdk.policies.create`. Inventory evidence alone
is not lifecycle coverage: entries without observed MCP case calls remain pending.

### Sequential policy-review timing follow-up

The first full run with case 011 passed 24/25 checks but stopped before saving its
initial policy. Failure HTML showed the unchanged review dialog, enabled Save,
no server error and policy revision 0. Added an explicit wait for Blueprint's
review-dialog enter transition to finish before Save in UI policy cases, then
started a full repeat. This repeat must pass before claiming the combined matrix.

### Focused retry after the policy-review wait

At the user's request, the broad repeat was stopped and only
`testWebDriver_ui_namespace_admin` was selected through Endly MCP. Run
`ui-admin-03`, operation `ee68d832-e663-4bc1-90ea-a36aa4a78fde`, completed with
**51 passed assertions and zero failures**. This includes the new transition
readiness assertion and all final namespace MCP permission/data checks. The full
matrix after this wait change has not yet been revalidated; the interrupted
broad repeat is not a passing full run.

### UI-authored combined custom predicate

`testWebDriver_ui_custom_predicate`, run `ui-custom-predicate-02`, Endly operation
`e3ed8e9f-4b79-453b-a533-dcf144b12f1b`, passed **54 assertions, zero failures**.
UI controls create the namespace, SQLite connector, table-backed component,
source SQL alias, input parameter, one linked custom predicate and role policy.
Studio MCP publishes the version and real namespace MCP calls prove:

- `eligible`: EXISTS paid order AND NOT EXISTS active exclusion returns only 10.
- `exists`: 10 and 30; duplicate paid orders do not duplicate customer rows.
- `not_exists`: 20 and 40, including unpaid-only and no-order customers.
- Repeated `eligible` calls still return only 10; absent optional input returns
  all four rows; invalid mode errors; the unauthorized user cannot discover or
  invoke the tool.

The example's SQLite unit test also verifies the combined condition. It remains
linked by standard mains and optional for reusable app packages. No DQL import
or backend builder mutation substitutes for browser authoring in this case.

### Cube authoring gaps found and fixed (acceptance still in progress)

Focused case 013 exposed two separate issues: inline view SQL mutations returned
structural inspection without refreshed compiled columns, and SQLite could not
infer an aggregate output type from its zero-row discovery query. Reader Builder
now refreshes exact-revision column metadata after view/column edits. The column
editor supports declaring an existing SQL output alias and Go type when discovery
has no columns, delegating validation to Datly's column-contract operation.
A read-only compiler probe proved `total` declared as `int64` resolves all three
aggregate columns; both relevant UI test files pass **30 tests**. Case 013 was
updated to use these controls, but its successful end-to-end result is not yet
established. An independently started terminal Endly run currently owns the
shared fixture ports; no competing fixture was started.

### UI-authored cube acceptance

`testWebDriver_ui_cube`, run `ui-cube-04`, operation
`53d8a460-91b9-4833-94cf-099fb92e5975`, passed **60 assertions, zero failures**.
UI creates the dedicated namespace/connector/component, aggregate view SQL,
explicit total type, dimension and measure roles, optional filters, cube exposure
and access policy. Studio MCP publishes, then namespace MCP verifies source rows,
region totals (east 30, west 20), channel totals (store 35, web 15), east-only
channel totals, repeat-request stability and direct denied cube invocation.
The settings fix adds root groupability atomically with Cube enablement; without
it the previous UI saved cube settings but published only the source reader.

Endly commit `524d393f7862ad80956c4895bc644045d579968a` was pushed through IntelliJ
to `origin/master`. It retries pending controlled-checkbox state within the
existing bounded element wait, re-reading state before any further click.
Targeted race tests and live Chrome `TestLiveBrowserFlow` pass, including a
synthetic deferred checkbox and binary PNG validation. The focused cube run
used this runtime fix; build instructions now pin its published revision.

The full Endly WebDriver package also passes `go test -race` after the checkbox
change. `/usr/local/bin/endly` was rebuilt and installed atomically;
`go version -m` verifies `CGO_ENABLED=0` and revision
`524d393f7862ad80956c4895bc644045d579968a`.

### Cross-database cube and composition UI acceptance

Case `testWebDriver_ui_cube_compose`, run `ui-compose-03`, operation
`ffc1ff4d-b5e8-4d66-94f2-8b62da30acee`, passed **105 assertions, zero failures**.
DSUnit initializes two separate SQLite files. UI creates both connectors, the
namespace/component, aggregate SQL and typed columns, an optional one-to-one
lookup on the dictionary connector, cube/compose settings and role policy.
The Schema Browser now lists components within the namespace independently of
the selected source connector and emits one atomic add-view/connector binding.

Source-cube calls verify east/west dictionary labels, a null lookup for north,
region and channel grouping, filtering and repeat stability. Compose calls verify
web/store comparisons, an inherited east filter, repeat isolation, the two-frame
budget, rejection of arbitrary table SQL, and denied caller execution. The UI
Composition lab and native `studio.sdk.versions.test_compose` also return the
expected aggregate. The lab now sends the actual native frame contract; wrapper
SQL selects dimensions/measures rather than ignored frame properties.

This proves dictionary enrichment on the source cube and scalar composition
results. Including dictionary labels inside composed results remains an open
user preference, not a capability claimed by this test. Full-suite and clean
checkout gates remain pending after these changes.

The user confirmed **dictionary on source cube only**. Case 014’s 105-assertion
run therefore satisfies the requested enrichment scope; adding dictionary
labels to composed scalar results is not an outstanding requirement.


## Independent namespace skill runtime — partial implementation

Canonical schema version 23 adds namespace skill heads and immutable revisions
without a component dependency. Generated Datly writers enforce expected etags,
append-only history, checksum validation and atomic rollback. Generated readers
use native query selectors for pagination. Published metadata comes from the
immutable bundle, so a later draft cannot alter the published title/description
or files.

The runtime now loads published namespace skills into the namespace's MCP
generation. Tests exercise a resource-only namespace on both separate and shared
listeners with actual `skills/list`, `skills/get` and `resources/read` requests,
cross-namespace denial and removal after unpublishing/reload. Removing the final
skill withdraws the optional skills extension. Exact per-file ownership retains
explicit resource policies and cannot grant access to a component folder through
a matching prefix. A 105-skill test verifies pagination across the 100-row page.

Bundle tests cover immutable snapshots, canonical ordering/digests, binary files,
path confinement, collisions, malformed metadata, inventory limits and cancelled
requests. Schema/migration, generated storage and runtime host package tests pass
with `CGO_ENABLED=0 GOWORK=off`.

This is backend/runtime verification, not the requested UI/Endly acceptance.
The public namespace-skill API, commit/reload publication orchestration, default
UI editor and combined component-plus-namespace `skills/list = 2` case remain
outstanding. No new full Endly suite or clean-checkout result is claimed.


## Namespace skill management and namespace pagination

Case `testMCP_namespace_skills`, run `namespace-skills-03`, operation
`b94a6c0e-bdfd-426f-b9ab-e836ec20f42f`, passed **56 assertions, zero failures**.
All eight native `studio.sdk.namespace_skills` operations are exercised: list,
get, create, update, revisions, publish, unpublish and delete. The case creates
its own namespace with MCP disabled, authors an independent skill, verifies zero
components, enables the listener, publishes two immutable revisions, and calls
native `skills/list`, `skills/get` and `resources/read`. It checks outsider denial,
stale edits, retained published bytes while a draft changes, expired resource
URIs after republishing/unpublishing, archive guards and cleanup.

The first run found that namespace-scoped `runtime.status` incorrectly required
component publication authority. It now uses verified namespace visibility;
unscoped runtime status keeps its global authorization requirement. A retry in
the same Endly session reused the old mock IdP listener and responses; the final
run used a fresh session and fresh fixture data. Repeated setup within a retained
MCP session still needs explicit mock-listener reset; `listen` is idempotent and
does not replace its data automatically.

Case `testMCP_namespace_lifecycle`, operation
`cba0d0f4-fc04-41c3-ac2a-860a97fe3adb`, passed **23 assertions, zero failures**
against that same fixture. Native namespace listing is paginated with limit and
offset; tests check both pages, the empty final page, and an unauthorized user's
empty page. Focused reader tests verify authorization before pagination, default
limit 50, maximum 500 and offset handling.

The native management catalog now contains **92 tools**, verified through
`mcp/runner:listTools` operation `fa9f5b0f-8be0-4a6a-a659-b840e21458c3`.
The coverage manifest records the eight new skill tools and verified namespace
lifecycle methods. Publication unit tests additionally verify confirmed commits
before runtime reload and restoration of the prior publication on activation
failure. Go/TypeScript clients were regenerated; 65 StudioAPI client tests pass.

The default namespace-skill UI and combined UI-authored component/namespace
`skills/list = 2` acceptance case subsequently passed as recorded below. These focused results do
not complete the full suite, clean-checkout gate or remaining authorization/tool
coverage requirements.


The connection-resource contract was subsequently clarified to **JSON**, not
YAML. `mcp://ns/<namespace-name>` now serializes JSON and round-trips through
`agently-core/protocol/mcp/config.MCPClient`. Focused tests pass for bare and
OAuth/SCY configurations and actual resource reads on shared/separate listeners.
This does not yet implement the newly specified two-tool root discovery catalog:
`namespaces/list` and `namespaces/get`, with paginated accessible virtual resource
references and native resource reads for the JSON definitions.

## Root/admin catalog preparation (not final acceptance)

Datly's authored MCP tool-name validator rejected slash names even though native
skill bridges use them. Commit `6d083dcb9628a6463dd62e285350c311d906194c` allows
hierarchical names with nonempty, non-dot segments; MCP compiler/service tests
pass. It was pushed through IntelliJ to `origin/v1`. Studio now pins public
`v1.1.1-0.20261014005443-6d083dcb9628`, with no local replacement.

The protected runtime connection API and client validate exact namespace scope
and return the deployed Agently JSON definition. Tests pass for separate/shared
listeners, deployment-token guards, wrong namespace identities and malformed
configurations. Shared-path runtime status validation now permits only the
selected `/namespaces/<id>/mcp` path, fixing the earlier `/mcp`-only check.
Namespace-owned skill URIs now use the namespace as authority:
`skill://<namespace-id>/<skill-id>/<revision>/<name>/SKILL.md`.

The admin discovery handlers in `studio/catalog` are provisional, unlinked
work. The agreed shared extension is owned by the `Unified auth - latest` chat;
Studio will adopt its actual API instead of retaining duplicate contracts.
The final five-tool admin discovery Endly acceptance remains outstanding;
these preparatory tests do not prove it.

The broad native bootstrap check exposed a Datly 1.0 call-policy change after the
upgrade: nested writes now default to buffered execution. The resource-file row
was persisted when the root committed, but the same operation read its response
before flushing and returned empty files plus the previous version revision.
Resource mutation handlers and the dependent Builder/clone/archive workflows now
explicitly declare imperative child calls, preserving one managed transaction
while making writes visible to the following read. The complete native bootstrap
test passes, including HTTP/MCP/BFF resource lifecycle and invalid-input rollback.
No raw SQL replacement, independent commit, or legacy Datly API was introduced.

## Combined UI skill ownership acceptance

Case `webdriver/015_ui_skills`, run `ui-skills-01`, operation
`9497aab9-29ee-4959-88fa-60bd94998603`: **66 assertions passed, 0 failed**.
Endly operation status was rechecked as `succeeded`. The UI creates a dedicated
namespace, an independent skill with supporting content, a connector, a component
and its component-owned skill and role policy. MCP publication is followed by
actual namespace MCP calls: the allowed caller lists two skills and retrieves
both manifests/resources; the denied caller sees only the namespace skill and
cannot invoke the component or retrieve its skill.

The native shared-contract integration also verifies that a principal with no
visible namespaces receives exactly `namespaces: []` without `nextCursor` or a
tool error. `go test ./studio/catalog ./studio/namespace_skills/authoring -count=1`
passes using the explicit temporary shared-protocol workspace. This is not yet
a released dependency or the final root/admin Endly acceptance.

The same native MCP integration now covers 515 visible namespaces, crossing the
private reader's 500-row batch and public 32-entry pages. It checks exact coverage,
stable ordering, descriptions, filtering before pagination, and invalid,
cross-visibility and stale snapshot cursor rejection.

Provisional discovery now consults the active runtime connection export before
advertising an MCP resource. A stopped listener (admin 404) yields an empty
resource page and unavailable `resource/get`; restarting restores the reference,
and configuration changes are returned on the next retrieval. Admin 503 remains
an error. These lifecycle checks use native MCP handlers with a controlled admin
HTTP fixture; they do not yet prove production change notifications or the full
Endly lifecycle. Catalog, namespace-skill authoring, and runtime-admin packages
pass together using the explicit shared-protocol workspace.

## Latest Datly parameter-direction fix

Studio now pins public Datly `v1.1.1-0.20261014013852-e3f9f8a9cf1e`.
The local Datly `v1` checkout was fast-forwarded to the same commit. Its source
comment explains that input and output parameters can deliberately share a name;
type resolution must preserve their direction instead of letting declaration
order replace one type with the other. No GitHub commit discussion comments were
present when checked.

Both new focused Datly tests pass: runtime input/output type resolution and
factory generation with actual generated component invocation (including the
fixture's race test). Studio's `cmd/datly`, clone, catalog and namespace-skill
authoring tests pass against the new pin; builder, archive and resource mutation
packages compile, with their integrated behavior covered by the bootstrap tests.
The provisional catalog still uses the explicit shared-protocol workspace.

The 29-case Endly operation `33dc088f-f8d0-4a2b-8fa9-b954d50f7459`, session
`ef2d9c04-acbf-4fe8-9d12-d64b17fe1d06`, run `full-current-01`, was already built
before this dependency update. It remains a baseline run on `6d083dcb9628`, not
latest-pin acceptance; final status is pending.

That baseline operation subsequently **succeeded: 960 assertions passed, 0
failed**, with all 29 discovered cases selected. Its full terminal operation
record is `/tmp/studio-full-current-01.json`. This does not replace the required
latest-pin repeat or clean-checkout gate.

Endly commit `9ed0e3e271c3db226abe4f050c550e54913c52f7` adds explicit
`http/endpoint:listen` `replace: true`, retaining default idempotent behavior.
HTTP endpoint race tests cover replacement, replay reset, discarded old matches,
changed rotation, failed-load preservation, host-change rejection and cleanup.
The commit was pushed through IntelliJ and the pure-Go binary installed at
`/usr/local/bin/endly`; Git HEAD and origin/master match. Studio setup now opts
into replacement so repeated setup in a retained MCP session reloads the current
IdP fixtures. The already completed baseline used the preceding Endly sidecar.

## Provisional configured management catalog

Native MCP integration now verifies a configured `ns://dstudio` namespace and
renaming it to `ns://operations`, all-kind/window/datasource filtering, no root
self-connection reference, and retrieval of the complete namespace-management
window and its reusable datasource. The definitions use the shared window v3
reference contract and datasource v1 envelope; the datasource calls the existing
`studio.sdk.namespaces.list` tool with offset/limit paging and no result cache.
Native YAML is the authored source. Complete JSON resources are canonicalized
before hashing so object-map transport roundtrips preserve dependency hashes.

Catalog, namespace-skill authoring and management-definition package tests pass.
The optional host settings are `STUDIO_MANAGEMENT_NAMESPACE`,
`STUDIO_MANAGEMENT_PROVIDER_ID`, and `STUDIO_MANAGEMENT_MCP_SERVICE`.
This catalog is still unlinked production work: only the first window is present,
consumer rendering, remaining windows/skills and final Endly coverage are pending.

Latest-version full run is active: `latest-full-01`, session
`96ac978c-4671-4377-9d38-18d6ad790e9f`, operation
`83794da1-d159-4cac-aaae-11fc6c2954d5`. It builds Datly `e3f9f8a9cf1e` and uses
Endly `9ed0e3e2` on a separate MCP sidecar at port 4999. This operation completed
successfully: **960 assertions passed, 0 failed**. The full terminal record is
`/tmp/studio-latest-full-01.json`. A fresh-fixture repeat, `latest-full-02`, is
starting in the same retained session to exercise mock recording replacement as
well as suite repeatability. The provisional management catalog is not yet linked
into these Endly builds; its native tests remain separate evidence.

The retained-session repeat `latest-full-02`, operation
`5bdfbf26-0555-4a35-a95f-5ceb1d23c61d`, also **succeeded: 960 assertions passed,
0 failed**. The terminal record is `/tmp/studio-latest-full-02.json`. Both runs
built from the current working checkout and reinstalled UI dependencies. This
proves the 29-case suite's repeatability with new fixture/token data in one Endly
session; it does not prove the remaining clean-checkout or new-catalog gates.

New native case `mcp/013_publication_lifecycle` is authored for publication get,
second-version invocation, rollback, stale/unauthorized unpublish and withdrawal.
Its first focused run failed on a fixture slug containing an underscore; the
slug was corrected. A second run passed the lifecycle assertions but needed to
capture the expected JSON-RPC method-not-found response after withdrawal.
The final fresh-fixture run `publication-cycle-03`, operation
`956f479d-1b63-43fd-9b1d-4ae81275e218`, **succeeded: 54 assertions passed, 0
failed**. It owns its namespace, connector, SQLite database and table. Calls to
the actual namespace MCP return two rows for version 1, one row for version 2,
and two rows after rollback. Unauthorized and stale unpublish requests fail;
successful unpublish removes the tool and invocation returns code `-32601`.
Publication get/rollback/unpublish are now marked verified in the coverage map.
The suite now has 30 cases; a combined run including the new case remains due.

The connector lifecycle case now tests actual `connectors.test_sql` results with
a one-row limit, rejects an outsider, and rejects SQL on a disabled connector.
Operation `7034303b-af54-4dba-a410-057e75b0d1b3` succeeded with **24 assertions,
0 failures**. The publication lifecycle case additionally downloads an archive,
imports it as a new draft, validates that draft, and rejects an outsider's import.
Fresh run `publication-cycle-04`, operation
`74694282-0de6-4e8b-ba4a-2ae50b4ca83f`, succeeded with **62 assertions, 0 failures**.
The two additional tools are marked verified in the coverage map.

Studio subsequently pinned public Datly
`v1.1.1-0.20261014014912-85a3c4f9c9e9`, which avoids expanding unused SQL template
input roots while preserving binding checks. Datly SQL-template race tests and
Studio bootstrap, runtime preview, catalog and namespace-skill authoring tests
pass. Expanded 30-case full run `latest-full-03`, operation
`32a9405d-d5c4-4f63-8a27-0a5e21f3181d`, is active in session
`96ac978c-4671-4377-9d38-18d6ad790e9f`; final status is pending.

That expanded 30-case run completed successfully: **1,026 assertions, 0
failures**. Separate case `mcp/014_warmup_lifecycle`, operation
`e3842aeb-ebf0-4c0a-ae85-80d605740fe7`, passed **50 assertions** for durable
warmup acceptance/completion/history, unauthorized rejection, and repeated
published namespace MCP calls with an authored AFS cache. The suite now has 31
cases; its combined acceptance is still pending.

Predicate identity display is verified by focused MCP + UI run
`predicate-identity-01`, operation `d2245c22-c916-4f22-9e3b-5a146de491fa`:
**71 assertions passed, 0 failed**. The MCP catalog exposes `packagePath` and
`typeName`; the Security UI now visibly lists those fields for linked types and
shows full identity in saved entries and the handler picker. A render test checks
same-named types from distinct packages. Screenshot:
`.data/e2e/predicate-identity-01/artifacts/predicate-catalog.png`.

The publication lifecycle now also calls the direct metadata readers using
their advertised `ReportId`/`VersionNo` input names. Run `metadata-readers-03`,
operation `58671e8f-5070-4bea-b5d7-204157e0b88f`, passed **83 assertions**.
Version list/exact version/publication return the expected identities and state;
version ordering is explicitly requested. Views, parameters and exposures are
empty for this DQL-import fixture, matching both stored table counts and the
reader SQL. DQL import retains source/descriptors without populating these
relational metadata tables. Their nonempty coverage remains outstanding in the
coverage manifest; the passing empty checks are not claimed as full coverage.

Resource lifecycle operation `3b98a1a3-6ae4-48e6-8272-2c42153c71cb` passed
**41 assertions**. Files, folders and skills created through their public MCP
management tools are read back through the direct metadata readers with exact
component/version identities. An outsider receives no file contents. The folder
now uses `skill://<namespace-id>/consumer-guide/`. The three nonempty readers are
marked verified; their rows were not inserted directly as Studio DB fixtures.

## Namespace catalog bridge preparation

`internal/runtimeadmin.Client.ReadCatalog` now resolves a selected namespace's
live endpoint through the protected deployment API, then uses its ordinary MCP
authentication and resource authorization. Only skills/list, skills/get,
resources/list and resources/read are admitted. The deployment token is never
forwarded to MCP; only the caller bearer is used there, with redirects disabled.
Requests require canonical namespace selection and have bounded time/body sizes.

Runtime-admin race tests pass for JSON/SSE results, cursor preservation, credential
separation and rejection of unscoped/non-read methods. Real namespace runtime
tests pass for both separate/shared listeners: published skill list/get/content
read and foreign namespace denial. This is bridge evidence, not yet public
admin/root tool wiring or final end-to-end catalog acceptance.

## Endly RSA signing and user-info separation

`endly-rsa-10` passed **32 assertions, zero failures** for
`testMCP_publish_invoke,testPermissions_outsider` after rebuilding the fixture
against the explicit integration workspace. Endly operation
`562235f0-fdb6-4ff6-bce6-39a1a1295d20` completed successfully. Setup uses
`secret:signJWT`; fixture identity reads SCY's `user_id` and `account_id` claims.
Auth-compatible user-info retains its own `userId`/`accountId` wire contract.

`TestUserInfoOwnsPermissionsForSCYIdentity` passes: signed token permission claims
cannot replace user-info roles/features/grants, and mismatched subject, user ID,
or account ID is denied. Skill publication tests also pass after removing the
ordinary skill-tool alias assumption; native skill methods are not tool names.
This is focused verification, not a full-suite or clean-checkout acceptance run.


## Runtime codec extension — 2026-10-18 UTC

The complete 15-case MCP directory group passed with a fresh SQLite catalog and
shared DSUnit consumer hydration (Endly summary: 16/16 including setup, 20.7s).
The new `015_package_codec` case imported, validated, previewed, published and
invoked a reader using the host-approved synthetic `Tenant` codec. Assertions
checked three normalized preview rows, the exact beta row from namespace MCP,
invalid-input rejection and refusal to register the codec as an authorization
predicate. The existing package-predicate preview now supplies its declared
`Tenant` input name. These are MCP assertions; browser coverage was not run for
this change.

Command: `endly -selector-mode=path -r=run -t=cleanupPrevious,setup,test -i=<all 15 testMCP_* IDs>`.
Local log: `.data/codec-e2e-mcp-group2.log` (ignored).
Pinned-module Go tests passed for `studio/codeccatalog`, `studio/host`,
`runtime/host` and `e2e/cmd/fixture` with `GOWORK=off`.

## Persisted namespace composition — 2026-10-18 UTC

The complete 16-case MCP directory group passed with a fresh SQLite catalog and
shared DSUnit hydration: Endly summary **17/17 including setup, 21.6s**.
`016_namespace_composition` creates its namespace after the fixture starts,
imports and publishes a customer reader, and invokes `composition.count` over
namespace MCP. Its linked handler calls the published reader in memory and
returns four synthetic rows. Anonymous invocation is denied, and a separate
generic namespace does not expose the handler. The extension is selected by
looking up the current namespace ID in the catalog, without configured IDs.

The focused case passed first (5.8s). Commands use the directory-template IDs:
`endly -selector-mode=path -r=run -t=cleanupPrevious,setup,test -i=testMCP_namespace_composition`
and then all 16 `testMCP_*` IDs. Logs are ignored at
`.data/composition-endly-focused3.log` and `.data/composition-endly-mcp-group1.log`.
Pinned-module Go tests also passed for `runtime/host`, `app/studiohost`, and
`e2e/cmd/fixture`. This run covered MCP behavior, not browser flows or application
reporting acceptance.

### Native preview namespace propagation

The same full 16-case MCP group passed again (17/17 with setup, 22.0s) after
adding a pre-publication preview assertion to `016_namespace_composition`.
The linked dependency resolver checks the selected persisted namespace, and the
preview returns four actual rows. `versionrun.Authorized` now retains the optional
transport namespace selection after native run-access authorization, for preview
and exact-version diagnostics. The native run-access integration test verifies
selection retention and still denies a foreign namespace owned by the same user.
Tests passed for `internal/versionrun`, `studio/reports/store_run_access`, and
`runtime/preview`. Log: `.data/preview-namespace-endly-mcp-group1.log` (ignored).

## 2026-10-18: codec preview boundaries

Extended MCP case `015_package_codec` with exact preview rows for normalized
` ALPHA ` input, invalid preview input rejection, and verification that `Tenant`
is absent from the authorization-predicate type picker while `CustomerScope`
remains available. The existing published execution and registration rejection
checks remain in the same directory-owned case.

Focused case passed; all 16 MCP cases then passed with fresh shared DSUnit setup
using explicit `testMCP_<case>` selectors. Evidence: ignored local
`.data/e2e/evidence/codec-preview-regression.log` and
`.data/e2e/evidence/codec-mcp-group-regression2.log`.
The first group attempt selected the unsupported literal `testMCP_*` and ran
no cases; it is not counted as acceptance evidence. Browser cases were not run
for this workflow-only change.

The same case now composes the linked `Tenant` codec with package-scanned
`CustomerScope` on one input. Exact alpha/beta rows prove normalization happens
before predicate binding in preview and published execution. The focused run
passed in 4,373 ms and the full 16-case MCP group passed in 21,925 ms; logs:
`.data/e2e/evidence/codec-predicate-composition.log` and
`.data/e2e/evidence/codec-predicate-mcp-group.log`.
