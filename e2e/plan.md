# Datly Studio Endly e2e replacement

## Scope and acceptance

Replace the legacy MySQL/unit-test wrapper with real Endly directory-template
regressions against an isolated SQLite Studio catalog and SQLite consumer data.
Use the actual linked Studio native MCP contracts, the namespace runtime MCP
listener, and Endly WebDriver for browser authoring. An action completing with no
assertions is not a passing test. Do not use the developer's running Studio data.

Author workflow files as native YAML mappings and sequences, with two-space
indentation. Do not serialize entire workflows or request objects as JSON inside
`.yaml` files. Keep JSON only where the tested protocol or an actual JSON file
requires it. Browser command expressions remain ordinary YAML string values.

Browser component authoring must use the visual builder controls, including
views, columns, relations and predicates. Loading or pasting DQL does not satisfy
UI component-authoring coverage.

### Complex component revision acceptance (verified focused run)

Use a dedicated namespace and source tables. Create the namespace and component
through the UI, then configure three subviews and four predicates through visual
builder controls. Validate and publish revision 1 through Studio MCP; call the
namespace MCP and assert the returned columns, nested rows and predicate effects.
Reopen the component through the UI, remove a column from a view and add
predicates, then validate and publish the next revision through Studio MCP.
Call the namespace MCP again with inputs covering the original and added
predicates; assert that the removed column is absent and that results reflect the
new revision. Do not use DQL import, DQL editing or backend builder mutations to
substitute for these UI steps.

The reference layout is `viant-internal/platform/e2e`: root setup/build/test tasks,
`regression/regression.yaml`, contiguous numbered `cases/<group>/NNN_name`
directories, `tag: $pathMatch`, `subPath: cases/<group>/${index}_*`, case data and
shared `default/test.yaml` fallback. Case selectors must be discovered through
Endly `listInstances` on the root workflow before execution.

## Services and boundaries

* DSUnit registers SQLite, applies consumer DDL, loads dictionary/case rows,
  resolves sequence-backed foreign keys, and verifies persisted effects. Only
  fixture initialization may create Studio schema; tested authoring uses MCP/UI.
* Endly MCP control owns one reusable session, workflow loading, task/instance
  discovery, execution, operation events, assertions and focused reruns.
* `mcp/runner:listTools` and `mcp/runner:call` exercise the real Studio MCP server.
  Synthetic JWTs are ephemeral secret files; workflows never print tokens.
  Native HTTP runner may test protocol errors and skills/resources protocol
  methods not yet exposed by mcp/runner. Extend Endly with bounded native actions
  when the existing transport cannot express an acceptance case.
* Endly WebDriver owns browser operations, waiting, assertions, screenshots and
  failure artifacts. No Playwright script or direct API request may perform a
  mutation whose case claims browser coverage.
* The native authoring listener, namespace runtime and browser origin use separate
  loopback ports. Namespace B must have an independently discovered MCP endpoint.
* A Go fixture links an example predicate package. Startup receives only its
  canonical package string; `STUDIO_PREDICATE_PACKAGES` scans linked handler types.
  DQL uses the typed predicate and consumer rows prove filtering actually occurs.

## Fixture model

Consumer tables: customers (tenant, name, active), orders (customer FK, amount,
status), and order_items (order FK, product, quantity). Include two tenants,
multiple customers, empty relationships, inactive rows, and differing totals.
Use synthetic owner/editor/viewer/runner/outsider identities, an expired token,
and a token from the wrong issuer. All case-created names include the case tag.
Mocks return explicit allow/deny, role/exposure facts, timeout and malformed
responses. Mocked permission decisions must not replace the application checks.

## Regression groups

| Group | Cases and assertions |
|---|---|
| discovery | Native tools/list pagination, input/output schema inventory, no duplicate tool names; committed coverage manifest must match discovered tools |
| connectors | create/get/list/update/probe, schema/table discovery, test SQL, activate/disable/delete; failed probe and stale etag do not mutate valid configuration |
| namespaces | create/get/list/update/delete; owner/private/public/role visibility; invalid ID, archived selection and cross-namespace access fail closed |
| components | create/get/list/update/archive; version create/get/list, DQL/archive import/export/download, builder/inspect/descriptor/apply, exact revision CAS |
| execution | validate, preview, view/relation/compose tests, warmup submission/get/list; consumer row IDs and aggregates verified |
| publication | publish/get/events/status, edit clone and republish, stale publication conflict, rollback and unpublish; call published tool in the same case |
| isolation | create namespace B and dedicated component, publish, discover its endpoint, invoke it; A tools/resources/skills never appear in B and vice versa |
| predicates | package-string scan exposes example handler, non-handler ignored, unlinked package rejected, DQL execution filters rows by typed predicate |
| resources | file/folder/skill create/update/get/delete, root confinement, duplicate names, invalid SKILL.md, immutable publication snapshots |
| permissions | ACL list/upsert/delete, access list/context/get/replace, authz policy get/context/replace and authorization.check; owner/editor/viewer/runner/outsider, revocation, forged claims, outages, stale policy revision |
| exposure revisions | separate cases own their namespaces, components and published revisions for each exposure; no simultaneous per-user revision routing is required |
| webdriver | independent cases: create component; create/update; create/archive; create/publish and invoke MCP; component skill; namespace skill/publish; separate exposure cases and revisions |

Every tool in `coverage.json` needs a case mapping and an observed call. Read,
mutation, deny/error, and persistence assertions are distinguished; calling all
tools with invalid empty arguments does not count as lifecycle coverage.

## Case contract

Each case is independently runnable after declared setup. It creates its own
connector/namespace/component through MCP in the MCP group. Browser cases create
their namespace, connector, component and policies through UI; MCP is used for
read-only assertions, publication and actual namespace calls. Record returned IDs and
revisions in case-local state. Assert tool `IsError`, relevant structured output,
and DSUnit effects. Negative cases assert denial plus unchanged rows. Publication
cases query the live endpoint before passing, not only the catalog record.
Use bounded polling for runtime reconciliation; do not hide failures with sleeps.

## Capability gaps to close or report explicitly

The current public Studio API has component archival via components.update, not a
hard-delete component tool. Test archival and describe it accurately. Independent
namespace skill storage/revisions and runtime catalog loading are implemented;
the public authoring API and publication orchestration now pass a focused native
MCP lifecycle case. The default UI and combined two-skill case remain outstanding. An optional embedding callback is not completion of this
requirement. Add real namespace ownership and verify native skills/list and
skills/get alongside a component-owned skill; never substitute a hidden component
or skip the namespace-owned case. No private implementation or business fixture
is copied upstream.

The current Endly MCP server starts in another project's cwd. Workflows must load
relative resources from their source directory while preserving instance/default
fallback. Verify this with a focused regression before using root templates.

## Execution and evidence

1. Inspect services and selected Datly v1 contracts; load root and enumerate cases.
2. Build/start isolated fixture and seed consumer data through DSUnit.
3. Run one MCP lifecycle, then its whole group; fix failures at their source.
4. Run permission/isolation/publication/resource groups and every coverage entry.
5. Run each browser case through Endly WebDriver, then all browser cases together.
6. Record exact revisions, Endly session/operation IDs, selected case IDs, passed,
   failed and skipped counts, assertion totals, and failure artifact paths.
7. Stop only fixture-owned processes. A blocked or zero-assertion case is not green.

Status: implementation in progress. Execution evidence is recorded separately in
`results.md`; this plan does not claim the acceptance matrix has passed.

## Additional acceptance requirements

Each authored-component case owns a namespace and component-specific source tables.
Namespace creation, connector setup, component authoring and policy creation/edits
use the UI. Publication uses Studio MCP; the final steps discover and invoke that
namespace's MCP as allowed and denied users. Each case may contain many sequential
steps and repeated calls. The new mock follows `/v1/api/user/info` with roles,
features and entityPermissions, but decisions remain in `viant/authz`.

Required independent UI cases: feature-only, role-only, role plus feature, customer
10 permission plus role XYZ, and a separate existing-policy modification flow.
Add a cube case and cube-compose with a fact table and an outer dictionary in a
different database, each with multiple calls. Add one linked custom handler
covering both EXISTS and NOT EXISTS. These additions are not represented by the
earlier 16/16 success result.

Namespace MCP transport supports both separate ports (default) and a shared port
with namespace paths. In shared mode a positive namespace port requests a dedicated
override. Tests must use discovered URLs and maintain namespace isolation in either
mode. Pure-Go Oracle and SQLite drivers must build with CGO disabled.

Add a combined skill acceptance case with its own namespace: create one component
skill and one independent namespace skill through UI, publish, and use native MCP
skills/list to assert exactly two entries. Call skills/get for each and compare
identity, revision and document content. This requires real namespace-owned skill
support; do not emulate that ownership using a hidden component.

## Policy refinements requested during implementation

- Keep component management and published consumption policies separate. Management
  actions are view, edit (all component changes), remove, publish and manage
  permissions. Do not add a duplicate "change component" permission.
- Exposing a business MCP tool and invoking it use the same consumption rule by
  default, enforced independently at catalog discovery and execution. The UI must
  apply the rule to both without requiring repeated manual configuration.
- Support `*` as an action fallback within each policy domain. Explicit action
  rules, including an explicit deny, take precedence. Removing an override
  restores fallback inheritance.
- Support `*` within subject, role, exposure and typed entity conditions. A role
  or exposure wildcard requires a nonempty verified value; a customer wildcard
  must not expand access beyond granted customer IDs. Test both allowed and
  empty/mismatched facts using real namespace MCP discovery and invocation.
- Optional `STUDIO_NAMESPACE_ADMIN_ROLE` grants namespace management only when
  it exactly matches a verified IdP role. Empty configuration preserves existing
  ownership checks. Test unset, matching, wrong-case, absent, expired and forged
  roles. Include UI namespace management by the admin and denial to a non-admin.
- The new wildcard evaluator lives in `viant/authz`; publish/pin the dependency
  before claiming standalone Studio e2e support. No committed sibling replace or
  hidden Go workspace dependency is acceptable.

## Agently namespace connection resource

Expose the literal resource URI `mcp://ns/<namespace-name>` as a namespace MCP
connection definition, not a namespace/component reference. Return JSON using the actual Agently MCPClient contract
matching Agently's MCPClient/viant MCP ClientOptions contract, including the active
shared or dedicated endpoint and OAuth/protected-resource metadata configuration.
Use SCY secret-resource references for credential material. Verify resource
visibility, namespace-name resolution, exact YAML decoding into Agently config,
OAuth configuration and actual client connectivity. Do not expose access or
refresh tokens or encryption keys as plaintext in resource content. Also support a bare connection without authentication when the namespace runtime
allows it; never describe a protected namespace as unauthenticated.

## Next implementation checkpoints

- Entity-scope UI acceptance is implemented and passed (40 assertions before
  the explicit input-schema assertion). Reader Builder atomically adds the
  component's own fully qualified access-context binding, required entity IDs,
  and mandatory group-99 predicate. The mock IdP supplies role `XYZ` and typed
  customer grants. Real MCP calls prove customer-10 bounds and reject missing
  roles/grants; forged arguments cannot broaden scope. Run the entire matrix
  again with the new case and assert protected bindings stay out of tool input.
- Namespace admin helpers, native update/delete, and management projections pass
  the production-component bootstrap test and real UI acceptance (case 011,
  50 assertions). The fixture uses allowlisted synthetic login identities. The
  full default suite including this case is the next regression gate.
- Component management policy separation (view/edit/remove/publish/manage
  permissions) still requires implementation; current component grants remain
  the legacy capability model. Preserve separation from consumption policies.
- Namespace MCP connection resources are implemented on namespace listeners and
  verified by real protocol Go tests. Add the Endly UI-authored acceptance cases
  and verify SCY encrypted credential resolution, not only YAML shape.
- Add independent namespace-owned skill storage/publication and its combined
  component-skill case. Cube UI case 013 passes 60 assertions; cross-database
  case 014 passes 105 (UI setup, source-cube dictionary enrichment, scalar
  composition). The user confirmed dictionary enrichment is required on the source cube only;
  scalar composed results satisfy this case. The linked combined EXISTS/NOT EXISTS predicate passes case 012
  with 54 assertions; full-matrix verification remains outstanding.

Exposure clarification: the user confirmed separate cases and revisions, not simultaneous published-version selection by user exposure.

## Design discussion: root MCP and workspace namespace discovery

The root defaults to an in-memory MCP representing the workspace namespace; it
requires no network listener or root port. It exposes the same namespace
catalog contract through in-process MCP transport. External roots are optional
and admitted only when they explicitly advertise and implement our versioned
namespace-discovery contract. Ordinary MCP support or a coincidentally named
resource is insufficient. The exact capability identifier and schema remain to
be defined and tested jointly with the Agently client.

The root uses `namespaces/list` for paginated namespace discovery and
`namespaces/get` for paginated authorized virtual `mcp://...` references. It serves
Agently JSON connection definitions through native
`resources/read`. Agently needs opt-in namespace connection expansion; its
existing MCPClient configuration does not yet support child expansion. Maximum
expansion depth is one, enforced by the client. Children cannot expand further.
Discover definitions first and connect lazily. Each child retains its endpoint,
OAuth resource/audience and SCY credential references, or explicit bare mode.
Both shared-path and dedicated-port endpoints work with the in-memory root.
Discovery does not grant execution rights or forward root credentials blindly.

Proposed acceptance, if this design is adopted, would cover the in-memory
workspace root, a compatible external root,
rejection of an unsupported or incompatible external root, allowed/denied
namespace catalogs, stale/revoked entries, duplicate definitions, rejected
recursive expansion, and actual child MCP calls under multiple users. This
section records brainstorming only; root expansion is not implemented or treated
as an approved implementation requirement for the current e2e work.

Custom predicate clarification: one handler must emit an EXISTS subquery on one
related table AND a NOT EXISTS subquery on a different table in the same
condition. Switching EXISTS/NOT EXISTS against only one table is insufficient.
Case 012 uses paid orders and active exclusions, with positive, negative,
both-table and no-order rows, and repeats real MCP calls after UI authoring.


Root discovery contract clarified: the root MCP should expose only
`namespaces/list` and `namespaces/get` as tools. Both are paginated;
`namespaces/get` returns only accessible `mcp://...` resource references in the
selected namespace. Window discovery is not a prerequisite. These virtual resources resolve through native `resources/read` (`resource:get`
in the client) to JSON MCP connection definitions compatible with agently-core; do not replace existing Studio management
or deployed component tools with root discovery aliases. Client-side recursive
expansion remains separate from this root catalog contract.


## Final UI-authored catalog acceptance

The final case must create its own namespace through WebDriver, create a component
and a component-owned skill through the UI, create an independent namespace-owned
skill through the UI, and publish both. Use its own component table and revisions.
Do not substitute MCP creation, DQL import, or a hidden component for this setup.

Call the admin MCP after setup. It retains all existing management tools and adds
the following discovery/read tools:

1. `namespaces/list`: verify bounded pages, a second page, no duplicates and an
   empty final page. Unauthorized namespaces must not consume page slots.
2. `namespaces/get`: verify a paginated list of accessible virtual `mcp://...`
   resource references for the selected namespace, including an empty final page.
3. `resource/get`: resolve an advertised virtual URI and validate its JSON against
   the actual agently-core MCPClient contract. Verify namespace endpoint identity,
   shared/separate listener paths, bare configuration and OAuth/SCY references;
   inline access tokens, refresh tokens and encryption keys must not be exported.
4. `skills/list`: assert exactly two published skills, one namespace-owned and one
   component-owned. Test caller-specific filtering, namespace isolation and
   pagination.
5. `skills/get`: fetch both advertised entries and validate identities, immutable
   revisions, frontmatter and file inventories. Read the advertised SKILL.md
   resources to compare content. Denied callers must not retrieve hidden entries.

Keep the separate root discovery catalog limited to `namespaces/list` and
`namespaces/get` tools; virtual resource resolution uses the resource protocol.
Admin `resource/get` and skill tools must use the same authorization decisions as
native resource/skill methods. No window/list or window/get is a prerequisite.
The final case is not complete until it runs through Endly against the real
admin and namespace MCP services and all assertions pass.


Latest namespace listing contract: `namespaces/list` returns paginated namespace
URI references (`ns://<id>`), not the resources in those namespaces. Skills use
`skill://<ns>/<path-to-skill>`. Namespace content selection uses
`/namespaces/<id>` for all kinds or `/namespaces/<id>/<kind>` for one kind.
The final acceptance must distinguish namespace references, content collection
paths, skill URIs and virtual MCP configuration URIs.


Final clarification: `namespaces/get` accepts `uri: ns://<id>`, optional `kind`,
and pagination (`limit`, `offset`). Omitting `kind` lists all accessible resource
kinds. Apply both authorization and kind filtering before slicing a page. The
`/namespaces/<id>[/<kind>]` forms describe namespace collection paths, not an
alternate opaque identity or a replacement for the `ns://` input contract.

## Shared namespace protocol ownership

The user requested coordination with the `Unified auth - latest` chat. That chat
has agreed to own `github.com/viant/mcp-protocol/extension/namespaces` on an
isolated feature worktree. Studio must consume that shared contract rather than
ship its provisional `studio/catalog` DTOs, URI helpers and cursor implementation
as a second protocol. No concurrent edits to the shared package from this chat.

The agreed outline is `namespaces/list` returning namespace references with
`uri`, `name` and optional `description`; `namespaces/get` accepts `uri`, optional
`kind` and `cursor`, returning resource summaries with descriptions and optional
`nextCursor`. Managing MCP servers filter before pagination. Agently Core does
not infer resource/entity permissions; its gateway can retain kind-specific
compatibility methods over the shared index. Exact shared Go types and package
revision are pending the owning chat's implementation.

## Configured administrative namespace

The admin/management MCP must publish its own configuration-named namespace,
for example `ns://dstudio`. This is a virtual host namespace, separate from
user-authored database namespaces; do not impose Studio's persisted hex-ID
validation on its configured opaque identity.

Expose the internal management visual definitions as portable window resources
(namespaces, connectors, components, skills, policies/security and runtime),
their datasource definitions and authoring skills. Names and IDs come from configuration.
The shared namespace index includes these kinds and descriptions; `resource/get`
resolves definitions. Agently discovers/renders generic resources and does not
infer resource/entity permissions. The management MCP remains authoritative.

Final acceptance must enumerate and resolve these administrative resources under
the configured namespace, verify a changed configured name, render the management
windows through the intended consumer path, and retain actual management tool
and authored namespace MCP invocation checks.

The latest coordinated connection-catalog requirement excludes the main MCP's
own connection from registration. It maintains Agently-compatible configuration
resources for running namespace MCPs only. A persisted enable flag does not prove
that a namespace listener is running. Start, configuration change and stop must
update discovery/retrieval and emit standard MCP resource-change notifications
so the consumer refreshes its existing registry. Test add/update/remove and root
self-exclusion against actual listeners. The administrative visual namespace is
separate from this connection self-exclusion. An empty namespace/resource list is
a valid successful state; admin-service failures must not be converted to empty
catalogs.
