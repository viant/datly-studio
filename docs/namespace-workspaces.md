# Namespace workspaces: implementation status

Namespaces are private by default. Their owner manages visibility and assigned
viewer roles. Public visibility does not grant management or bypass resource ACL.
DQL, components, skills and MCP exposure belong to a namespace; owning two
namespaces does not combine them. Connectors may be global infrastructure shared
across namespaces. Connector ownership is not implicitly inherited from the
currently selected namespace, and connector access does not grant resource access.
Datly Studio remains independent and reporting-free.

## Current acceptance summary

The published Studio main revision now includes authz integration, namespace
selection, private/public visibility, separate namespace MCP listeners, and
cross-version ACL protection. UI-created Alpha/Beta publications run together on
ports 18891/18892; forecasting and skills pass on private port 18893. The latest
full forecasting matrix passed 262 inputs / 135 custom handler types.

Completion is still unproven. Real Viant IDP forecasting permission modeling and
successful scoped UI preview are open. Mandatory resource selection, remaining
namespace persistence/query scopes and complete
multi-window coordination also remain open. Later sections record incremental
implementation evidence; none substitutes for these acceptance gates.

## Implemented foundations

- Schema version 17 adds namespace identity and namespace visibility/MCP metadata.
  Global connectors are excluded from namespace ownership migration.
- SDK namespace DTOs and native create/update components carry these settings.
- Namespace authorization helpers separate visibility from owner management.
- Legacy component namespace keys missing from the namespace catalog are preserved
  as private namespaces. Stable IDs propagate from components to their versions.
- Native namespace get/list responses project namespace ID, private-default
  visibility, assigned roles, and MCP settings. Invalid stored role JSON fails.
- OpenAPI and generated UI clients include the namespace metadata.

The role verifier uses the existing `STUDIO_ACCESS_ISSUER`,
`STUDIO_ACCESS_AUDIENCE`, `STUDIO_ACCESS_PUBLIC_KEY_FILE`, and optional
`STUDIO_ACCESS_USER_INFO_URL` settings. With no role verifier configured, native
namespace discovery grants owner/public visibility only. Partial or invalid
configuration fails closed. These native endpoints still require a verified JWT;
public namespace visibility is not an anonymous-authentication mode.

The migration regression test verifies two namespaces for the same owner, private
initial visibility, child ownership, and repeat migration. These checks establish
migration behavior, not complete request isolation.

## Verified directory access

- Native namespace get/list predicates now enforce owner/public visibility and
  assigned roles from the server-configured authz provider. Native HTTP/MCP tests
  cover exact role names, role revocation, and request-role forgery rejection.
  The standalone Datly host already supplies the connector capability; hand-built
  test registrations must supply it explicitly as well.
- The SDK directory and namespace access readers use the same visibility criteria
  as the native endpoints. Component ACL does not grant private namespace access.
  Edit/publish namespace management remains owner-only. Signed-role SDK tests
  cover get/list, role revocation, public access, subject mismatch, and denied role
  management.

## Selected namespace request context

The SDK HTTP gateway accepts one `X-Studio-Namespace` ID after authentication.
The SQL transport resolves that ID through the authorized Datly namespace reader
and binds server-read namespace metadata for the request. A requested ID supplies
selection only; it does not supply ownership or permission. Empty, malformed,
inaccessible and archived selections fail without falling back to a workspace.

When selected, component lists apply namespace owner/name filters in SQL before
pagination. Detail and child authoring operations reject out-of-scope component
IDs even when the caller owns both namespaces. Component creation inherits the
selection; contradictory namespace inputs and moves out of the selected workspace
are rejected before mutation. The ACL resource catalog also uses this selection.
Global connectors and namespace management are control-plane operations and
remain available when a workspace selection becomes unavailable.

This is an incremental SDK boundary. Selection is not yet mandatory for all
resource operations, and most native HTTP/MCP resource routes do not yet bind it.
The UI selector, scoped predicates and all persistence writes are still required.

## Native component catalog selection

Native `components.get` and `components.list` bind optional namespace selection
from the `X-Studio-Namespace` HTTP header. Their MCP contracts expose the same
selection as `namespaceId`. The component catalog predicate joins the component's
owner/name namespace to the selected ID and checks active status and namespace
visibility in SQL alongside the resource ACL.

Native HTTP/MCP tests verify same-owner cross-namespace denial, private namespace
protection despite component ACL, malformed/empty HTTP selection rejection, and
filtering before pagination. Source DQL declarations and linked Go inputs carry
the header parameter; browser and generated SDK contracts are regenerated.

Native `versions.get` and `versions.list` now carry the same HTTP header/MCP
`namespaceId` selection. A shared authorization mixin combines resource permission
with namespace visibility. Tests verify metadata/source denial for the wrong
namespace and private namespace denial despite a component ACL, while retaining
existing source redaction. The version-edit predicate also supports the boundary
when a native request supplies selection; mutation route coverage is incomplete.

Native DQL import and version apply now expose namespace selection through
HTTP/MCP and enforce it through the edit guard before source mutation. Integration
tests verify successful in-scope imports/edits and unchanged persisted versions
following cross-namespace denials. The private run-access reader also checks
namespace visibility independently of metadata-view permission; native preview
and view-test tools expose selection and reject the wrong workspace before query
execution. A run-only caller remains able to execute inside a visible namespace.

Archive import, builder commands, resource snapshots and all file/folder/skill
mutation tool contracts now accept namespace selection. Archive and builder
handlers use the namespace-aware edit guard; resource writes do so before their
storage mutation. HTTP/MCP integration tests verify archive and file-write
rejection without new persisted records, builder denial, and successful scoped
resource operations through direct and BFF MCP. Relation and cube-composition
query-test contracts also carry selection into the shared run guard.

Inspection, download, descriptor and DQL-export contracts now carry selection;
their metadata/source guards remain responsible for resource permission as well.

Selection is still optional during this unfinished migration. Other authoring and
resource mutations, policy and publication paths need complete route coverage
before the selector or full namespace isolation can be accepted.

## UI request safety

The namespace settings dialog exposes private/public visibility, assigned viewer
roles, MCP enablement and a port (zero assigns an available port). Native and SDK
directory responses include server-computed `canManage`; viewers receive a View
action and no Delete action, and the dialog is read-only. Pending role text is
included on Save, closing is blocked while saving, and conflict reload refreshes
both ownership and management permission. Delete confirmation cannot close or
repeat its request while deletion is running. Fourteen rendered dialog/catalog
tests pass. GPT-6 Sol granted scoped UX approval for the settings dialog and
catalog actions; complete workspace coordination and forecasting acceptance remain
unverified.

The SDK runtime host status now includes `namespaceId` and `mcpUrl`, preserving
the endpoint returned by the deployment-configured admin service. The admin client
rejects a different namespace and malformed endpoints. This makes assigned ports
discoverable. In identity-token mode, browser tools/skills discovery resolves the
selected namespace's ready endpoint and sends the bearer token with cookies
omitted. A missing or mismatched endpoint cannot fall back to the global catalog;
late catalog responses are discarded after a switch. Session-authenticated browser
discovery stays on the BFF origin. The BFF resolves the namespace endpoint through
an authorized SDK runtime-status lookup and supplies the server-held bearer token;
opaque cookies and client-supplied bearer/admin tokens never reach the runtime.
Malformed selections, inaccessible namespaces and failed endpoint resolution do
not fall back to the global catalog. Namespace headers also survive the static
SDK proxy. An idle namespace can expose its MCP endpoint before a component is
published; runtime status probes the host even when no active generation exists.

`StudioAPI.setNamespace(id)` pins resource requests to `X-Studio-Namespace`.
Namespace management and global connector requests stay unscoped. Each resource
request captures a namespace revision; a late response is discarded after any
switch, including A → B → A. Token refresh cannot resend an old request after a
switch, and a stale 401 cannot expire the newly selected workspace view.

The app header now provides one current-namespace selector. It loads the visible
active directory with pagination, restores selection per user/API origin, and
applies selection before resource requests. Lost visibility blocks resource
requests rather than falling back to an unscoped catalog. Directory management
and global connectors remain available for recovery. New-component authoring is
pinned to the current namespace and the redundant component namespace filter is
removed.

Switching clears component/overview/runtime results, resets component pagination,
and remounts the runtime catalog so the previous namespace's tool and skill lists
do not remain cached in the new workspace.

Selection synchronizes through browser storage events. Open editors and dialogs
hold a short renewable window lease; switching is refused while any window holds
one. Unexpected external changes block requests and preserve the editor until it
is closed. The guard is currently conservative: an open editor locks switching
even when it has no unsaved edits. Storage failure/background suspension and
simultaneous editor-open/switch races still need stronger coordination acceptance;
the lease is a UI guard, never an authorization mechanism. The browser fixture
verified two-window synchronization and blocked/resumed switching around a dialog.
The full UI suite passes (94 client tests and 121 rendered tests). Two additional
retry regressions pass in the targeted 13-test selector/catalog run. Directory
errors expose Retry namespaces, which retains the intended selection, and stale
directory reads cannot overwrite a later selection. GPT-6 Sol granted scoped UX
approval for the selector layout and retry flow.

This UI flow does not establish server isolation for native routes that have not
yet bound namespace selection. Live forecasting acceptance remains a separate
gate; scoped selector approval does not establish full resource isolation.

## Remaining acceptance gates

- Stamp and scope all namespace-owned resource writes and queries, including
  predicates, DQL/SQL files, skills, publication and runtime state. Empty namespace
  defaults are migration foundations and must not become authorization fallback.
- Complete global-selector acceptance for every window/dialog, including dirty
  editor state, storage failures, background suspension and switch/open races.
- Require and bind namespace selection in all resource routes, including native
  HTTP/MCP components, and scope predicates, policies and runtime metadata.
- Broaden the verified independent MCP listener/generation scenarios to complete
  deployment acceptance; UI-published Alpha/Beta and forecasting now run together.
- Test cross-namespace denial for a caller owning both namespaces, role/public
  visibility, revoked visibility, duplicate resource names, and MCP isolation.
- Complete end-to-end forecasting execution/publication UX review. Namespace
  settings, selection, first-draft import and cube selection have scoped approvals.

Do not treat the schema or SDK fields as proof that namespace isolation is ready.
No live database upgrade or runtime deployment was performed for these foundations.

## Resource mutation ownership

Generated file, folder and skill-root mutation writers now persist `namespace_id`.
The shared SDK/native resource workflow derives it from the server-read component
owner/name key and rejects a contradictory selected namespace before mutation.
It never accepts workspace ownership from the public resource DTO. Lifecycle
hooks require canonical ownership, prevent changing a nonempty stored namespace,
and allow trusted assignment to repair an empty legacy value. Update/delete
operations preserve this invariant as well as existing source-revision checks.

SDK and native HTTP/MCP integration tests inspect persisted ownership on all three
resource types; lifecycle tests reject ownership changes. All three writer DQLs
also pass regeneration checks. Redundant older link-sync anchors were removed
after the current generator emitted their discovery anchors directly.

This covers explicit resource mutations. Other writers have separate graphs and
still need ownership coverage. Named Bindly
resource claims also retain their older global uniqueness rules; namespace-local
duplicate names have not been accepted yet. These are remaining isolation work,
not implied by the mutation tests.

## DQL import ownership

The generated DQL/archive import graph now persists namespace ownership on both
the version and embedded files. Its relation carries report ID, version number
and namespace ID together. SDK imports resolve the server-read component key;
native imports derive the same immutable key from the authorized component.
The import hook fills file ownership from the version and validates the result.
Conflicting explicitly supplied file ownership is rejected before the SDK invokes
the writer, so parent-link normalization cannot silently repair that conflict.

SDK storage tests verify exact ownership alongside source bytes, binary markers,
hashes and audit fields. Native HTTP/MCP integration tests verify the persisted
version and imported files, and existing cross-namespace denial/rollback tests
continue to pass.

## Scoped preview controls and view-test dependencies

Reader Builder no longer disables execution solely because a component declares
a server-owned access context. Execute capability still controls the UI; the
backend verifies the caller and provider, narrows entity scope, and rejects
client-supplied context values. UI tests exercise an enabled scoped request and
surface a server denial, and retain disabled controls when execute is denied.

Connected view testing exposed a nil-component panic during DQL extraction.
View testing now inspects with the version's SQL resources, registered types and
available connectors, handles missing authority as a typed error, uses the view's
SQL instead of invalid offsets into embedded DQL, and selects its named connector.
Connector discovery likewise receives version resources and types. Regressions
cover embedded sources, named connector discovery and missing authority.

The local helper API must run from the Studio checkout because its compiler uses
the process project root. Starting it from the private helper module produced a
package-authority error; the corrected launcher reaches the expected scoped
authorization denial instead. A configured trusted publisher-grant source remains
necessary for successful Viant IDP execution.

## Legacy generation ownership and version navigation

Schema 18 assigns an empty legacy generation namespace only when publication
references account for every recorded component, all referenced components exist
with ownership, and every component belongs to one namespace. Mixed namespaces,
incomplete references, orphaned components and unreferenced history remain
unassigned. Existing assigned ownership is preserved. Migration tests cover these
cases and repeat execution. The saved forecasting test copy's generation 25 now
appears inside its selected namespace after normal API initialization.

Runtime Open now passes the displayed version number to Reader Builder, so opening
a published component does not silently fall back to an editable draft. Sidebar
labels are keyboard buttons; rendered tests verify keyboard navigation and the
runtime version handoff. Full identity-backed forecasting preview remains a
separate acceptance gate.

## Component ownership

Component creation persists the canonical namespace ID derived from verified
ownership and the resolved workspace name. Native HTTP/MCP creation accepts a
namespace selection, resolves it through the owned active namespace directory,
inherits its name when omitted, and rejects contradictory names. SDK/native tests
inspect the parent component's persisted ownership.

Configuration updates can no longer change a component's workspace. Previously
they changed the grouping name without transferring version/resource ownership.
The SDK and native endpoints reject this with an explicit ownership error, and
the private configuration writer also prevents the change. Tests verify unchanged
ownership and etag after denial. An intentional transfer would require a dedicated
transaction covering the whole tree; ordinary metadata edits do not implement it.

## Version creation and draft-copy ownership

The generated version-insert writer now persists canonical namespace ownership
for SQL, DQL and structured creation. SDK creation resolves the component from
storage; native creation derives ownership from the authorized component.
`versions.create` exposes namespace selection in its HTTP/MCP contract, and
integration tests reject another selected namespace without allocating a version.

A real SQLite/Datly SDK test exercises the operations used by the UI draft-copy
flow: copy a published version's DQL, files, folder and skill into a draft, retain
folder/skill links, verify ownership on every copied row, preserve the published
source and its revision, and reject a copy into another selected namespace. The
existing rendered client test covers the UI helper's ordering and revision chain.
This combination is server/client coverage, not a live browser acceptance run.
Other component settings and insert graphs still need full namespace coverage.

## SQLite publication setup

Applying the Studio SQLite schema also creates SQLX's allocation ledger before
publication starts a transaction. Raw schema users previously lacked that ledger
once the SQLite SQLX product was registered. Publication/generation tests now pass
with the registered SQLite allocator. The allocation ledger is infrastructure and
has no namespace ownership or domain audit requirements.

## Current verification

- `go test ./...` passes for the current Studio worktree.
- Native namespace tests exercise HTTP and MCP role-filtered discovery.
- SDK tests exercise signed-role get/list and reject role-based management and
  mismatched credential subjects.
- Publication/generation tests pass with registered SQLite sequence allocation.
- The private forecasting custom runtime builds with the updated Studio dependency
  and its linked authorization predicates. No running runtime was restarted.

These checks do not verify the remaining global-selector, resource-scoping,
listener-isolation, or complete forecasting UX gates.

## Integration sequencing

The static/BFF MCP integration test now waits for the exact asynchronous warmup
handle to reach a terminal state before probing connectors on the same SQLite
file. It polls the warmup status through MCP, checks the returned run ID, and
bounds the wait. The recurring secret-backed connector-probe failure did not
reproduce across three consecutive integration runs after this sequencing change.
This result does not establish a Datly concurrency defect or prove arbitrary
concurrent workloads.

## Namespace MCP listener manager

`runtime/namespacemcp.Manager` creates independent Datly 1.0 streamable MCP HTTP
servers on loopback ports. A host factory receives the namespace ID and reserved
address, and supplies the scoped catalog/source plus its authorization policy.
Port zero allocates a free port. Identical starts are idempotent. `Rebind` reserves and constructs a replacement
before switching endpoints and draining the old listener. Occupied ports fail before catalog build,
and a failed build releases its port. Stop/close drain the HTTP servers.

Race-enabled tests exercise real Datly MCP servers and protocol calls: distinct
ports, separate tool catalogs, foreign-tool rejection, independent shutdown,
occupied-port failure, idempotency, and independent namespace construction.

A reconciler now loads persisted namespace rows through the private linked Datly
reader and starts, stops or rebinds listeners from enabled/status/port settings.
It reads a complete snapshot before changes and preserves existing listeners when
snapshot loading or replacement fails. Race-enabled database/protocol tests cover
independent enable/disable/archive behavior and port conflicts.

The runtime command and private forecasting launcher now wire this reconciler
through NamespaceFactory with scoped definitions, namespace visibility gates and
independent generation reloads. UI-created Alpha/Beta publications and the
forecasting endpoint have been exercised concurrently. The listener manager
alone does not enforce ACL; the configured host factory supplies those gates.

Port-change tests verify that an occupied port or failed catalog construction
preserves the existing endpoint, a successful change closes the old port, and
another namespace's endpoint remains available. Rebind replaces the listener;
namespace-specific generation reload is implemented separately by the host
source and authenticated admin API.

## Namespace-scoped runtime definition loads

Dynamic runtime configuration now accepts `NamespaceID`. With it set, active and
candidate publication readers apply the namespace ID in SQL, joining immutable
component owner/name to active, undeleted namespace rows. Global connectors remain
shared. The resulting definition list drives component and per-component resource
construction, preventing another namespace's definitions from entering the build.

Runtime tests verify independent active/candidate definition sets for two
namespaces owned by the same principal. Empty `NamespaceID` retains the existing
single-runtime deployment mode during the incomplete migration; it must not be
used for a namespace listener. Namespace visibility authorization and independent
runtime generations/reloads remain separate required gates.

## Runtime namespace visibility gate

Namespace-pinned runtime builds now check current namespace visibility before
HTTP/MCP component execution and before tool/resource/skill catalog authorization
and retrieval. This boundary remains independent of resource ACL. Private
namespaces allow their verified owner or assigned roles from the configured facts
provider; public component mode cannot bypass a private namespace. Active status
and role changes are re-read through the private Datly namespace component.

Tests cover private anonymous denial, verified owner access, assigned-role access,
role revocation, public visibility, archived denial, and catalog/resource callback
protection. Runtime/listener integration tests pass. End-to-end listener-factory
wiring with real credentials and independent generation/reload behavior remains
unverified; namespace isolation is not yet complete.

## Runtime-backed listener factory

`host.NamespaceFactory` now creates a namespace-pinned runtime per reserved MCP
endpoint, loads its active definitions, applies existing credential/discovery
middleware, and supplies its Datly snapshot source to the listener manager. Each
listener can reload its own runtime without moving either endpoint's port. Runtime
ownership follows listener ownership: drain/stop closes its runtime, and failed
transport construction closes the newly created source.

Integration tests exercise two persisted namespaces through actual Datly MCP
servers, verify distinct catalogs, reload one without moving either port, and
verify that a private visibility change removes that namespace's catalog while
leaving the other available. Race-enabled cleanup tests verify runtime closure on
shutdown and transport-build failure.

The runtime startup command still needs reconciliation wiring and namespace-aware
publication administration. Independent publication/generation routing, UI
controls and final forecasting acceptance remain unfinished.

## Runtime command startup

`cmd/studio-runtime -conf datly-runtime.yaml -namespace-mcp` now starts persisted
namespace MCP listeners without starting a global unscoped MCP listener. It polls
namespace settings every two seconds by default; `-namespace-reconcile-interval`
changes that deployment interval. Resource connector/predicate/DQL authoring
continues through the Studio UI and SDK. Namespace-group polling is cancellation-
aware and closes its listeners and database on shutdown.

Tests cover runtime factory/catalog isolation, listener lifecycles and polling
shutdown. The running instance was not changed to this mode. Namespace-aware
publication administration, UI controls, required selection across remaining
routes and final forecasting acceptance remain required.

## Namespace reload administration

Namespace-group mode now binds its deployment-configured HTTP address to a
private `POST /_studio/reload` handler. It requires the deployment token plus a
canonical `namespaceId` and nonnegative generation; unscoped reload, unknown
fields, trailing JSON and unknown runtime IDs are rejected. It reloads only the
selected namespace manager, retaining both listener ports.

The deployment-owned runtime admin client includes namespace selection when its
SDK context supplies one. Integration tests cover unauthorized/missing/unknown
namespace requests and successful scoped reload. Publication handlers still need
to retain that context through their detached lifecycle phases, and persistence
must isolate publication/generation changes between namespaces.

## Scoped publication lifecycle context

Native publish/rollback/unpublish tool contracts now carry namespace selection.
The detached publication lifecycle retains that selection and the verified bearer
credential rather than only the subject. The runtime admin client consequently
can address a namespace-specific reload. Namespace membership is checked before
resource publication permission without adding an unintended metadata-view grant.

Native HTTP/MCP tests verify cross-namespace publication rejection before
publication state creation or runtime reload, and successful publication with
correct selection. Shared generation records and other-publication repointing
still need namespace persistence isolation; these request guards do not prove
complete publication isolation or enable the final UI flow.

## Publication pointer isolation

The private other-active-publications reader now receives namespace selection and
filters publication rows through component owner/name and canonical namespace ID.
Activation repointing consequently changes only other publications in the
selected namespace. A transaction test with two namespaces owned by one principal
verifies the other namespace's active generation stays unchanged and verifies
rollback restoration of the selected namespace's pointer. Reader generation and
SDK SQL tests pass.

Runtime generation ownership, retirement, staged-generation recovery and status
queries remain shared and need complete namespace isolation before publishing
across namespaces can be accepted.

## Generation ownership and retirement

New runtime-generation inserts now stamp the selected namespace ID through the
Datly generation writer. The other-active-generation reader selects only that
namespace for retirement, including the separate empty-ID single-runtime mode.
A transaction test creates active generations for A and B, activates a new A
generation, and verifies that A's previous generation retires while B remains
active. It also verifies stored ownership and rollback restoration.

SDK SQL and generation-reader/writer transcription tests pass. Staged-generation
recovery, runtime status and other generation lifecycle queries still need complete
namespace filtering; existing empty-ID generation records require an explicit
migration/transition plan before namespace publication is accepted.

## Staged-generation recovery isolation

Generation catalog reads now select the current namespace, and the private staged
publication reader filters publication ownership through its component namespace.
Expired-stage recovery therefore considers only that namespace's builds and
publications. A test verifies that B's fresh build does not block recovery of A's
expired build, that B remains building, and that rollback restores both states.
SDK SQL and the affected DQL-generation tests pass.

Runtime status, full generation mutation guards, and migration/transition of
existing empty-ID generations still need completion before namespace publication
is accepted. UI controls and final forecasting acceptance remain unfinished.

## Generation mutation guards

The Datly generation-state writer now binds namespace selection into its previous-
row lookup and lifecycle validation. A foreign generation ID has no eligible
previous row and is rejected; namespace ownership cannot be rewritten through a
state mutation. Tests verify rejected cross-namespace and ownership-change writes
leave both generation states and owners unchanged. SDK SQL, writer lifecycle and
DQL-generation checks pass.

Runtime status filtering, empty-ID generation transition, required namespace
selection across every resource route, the global namespace UI/draft guards and
final forecasting/skill UX acceptance remain open. No live runtime was redeployed.

## Namespace runtime status

Active-generation status reads now filter generation ownership. Reader catalog
queries accept namespace selection and filter component ownership before returning
runtime readers, exposures and resources. The native runtime status contract
retains verified credentials and namespace selection into the SDK transport.
A test verifies A and B retain separate active generations and report counts,
including after the higher-numbered B generation activates. SDK, native bootstrap
and affected DQL-generation tests pass.

Namespace-group host readiness/status administration remains unfinished, along
with legacy generation transition, mandatory route selection, global UI/draft
coordination and final forecasting/skill UX acceptance.

## Grouped host readiness/status

Namespace-group administration now exposes token-protected
`GET /_studio/status?namespaceId=...`. It reports that namespace's runtime revision,
authentication mode and MCP URL, requiring one canonical ID and an existing
runtime source. The admin client sends its SDK namespace selection and rejects a
status response for another namespace. Factory integration tests cover separate
endpoint identities plus unauthorized/missing/unknown selection failures.
Runtime/admin tests and listener race regressions pass.

Legacy generation transition, mandatory selection, global namespace UI/draft
coordination, and final forecasting/skill UX acceptance remain unfinished.

## Native warmup selection

The native warmup start and list contracts now expose `namespaceId` for MCP and
`X-Studio-Namespace` for HTTP. Shared publication and warmup-read predicates add
namespace selection to resource permission checks. The generated warmup-get
contract carries the same selection. HTTP regression tests reject another
workspace owned by the same caller and malformed selection before list expiry
recovery; MCP list rejects cross-workspace requests too. This is selected-request
coverage, not mandatory selection or complete warmup storage ownership coverage.

## Cube picker UX refinement

Dimensions and measures are hidden initially. Preview or Preview settings opens
selection, Close restores focus to the invoking control, and component refresh
closes the picker. Twenty-five targeted rendered checks pass, including keyboard
opening/closing and focus restoration. The running forecasting draft v12 verifies
this flow. GPT-6 Sol granted scoped visibility, discoverability and keyboard
approval; this does not establish full Studio UX or authenticated execution.

Namespace switching now refuses storage enumeration or persistence failure. The
existing selection and request scope remain intact, and the error directs the
user to allow browser storage and select the namespace again. The directory
reload action is hidden for this error because it cannot retry a failed switch. Regression checks cover both failure
points and successful retry after recovery. This closes the storage-failure
switching gap; background suspension and simultaneous editor/switch coordination
remain separate acceptance work.

## Warmup storage ownership

New warmup runs now persist `namespace_id` through the transcribed private
writer. SDK creation derives it from the stored component; native creation uses
owner/name metadata returned by its authorized component guard. New rows require
canonical namespace ownership, and a supplied ownership change on an existing
run is rejected before writing. SQLite checks verify persisted ownership and
unchanged ownership/status after a rejected move. Updates of legacy empty rows
remain possible for system expiry recovery; this is not an access fallback.
This does not complete all namespace-owned table coverage.

## Validation and warmup native-route acceptance

Version validation now exposes namespace selection in its native HTTP/MCP
contract and carries it into the validation context. The linked-host integration
rejects a different workspace owned by the same caller for validation and warmup
start/list/get through both protocols. Denials do not change validated-at evidence
or add warmup rows. Authorized selected-namespace validation succeeds through HTTP
and MCP. The same integration inspects native warmup creation and confirms its
persisted namespace ID. Selection remains optional; these checks do not establish
mandatory isolation for every resource endpoint.

## Native component ACL selection

Native ACL list/upsert/delete now expose namespace selection for HTTP and MCP.
Listing applies the namespace boundary alongside its owner-only access rule.
Mutation handlers first invoke the namespace-aware component publication guard,
then perform the existing owner-only ACL lookup. An empty filtered grant list is
never used as proof that a cross-namespace grant may be created.

The linked-host regression verifies same-owner cross-namespace list redaction and
upsert/delete denial through both protocols, inspects unchanged existing grants
and absent rejected additions, and verifies authorized selected-namespace creation.
The ACL reader DQL now explicitly shapes its identity columns, preserving the
existing nullable Go contract across regeneration. These checks cover component
ACL endpoints; generic resource policy and remaining route coverage are separate.

## Publication read selection

Native publication-get and publication-event-list contracts now carry namespace
selection over HTTP and MCP. Publication status keeps its existing resource-view
permission; event history keeps its current-owner rule. Both add the selected
workspace boundary. Protocol checks verify same-owner cross-workspace redaction
and correct-workspace results. Event DQL explicitly shapes the stored identity,
status and timestamp columns so regeneration preserves the existing output types.
Selection is still optional, so this is not complete workspace isolation.

## Two namespace UI publication check

In an isolated catalog copy, namespace.alpha and namespace.beta were created as
private namespaces through the UI with MCP ports 18891 and 18892. Synthetic
MySQL literal readers (markers 101 and 202) were imported, validated and published
through the UI. Both publications are active together in separate owned
generations, 26 and 27; the same custom runtime process listens on both ports.
Publishing Beta did not retire Alpha.

This exposed two gaps. The private custom launcher lacked namespace mode; it now
uses the public namespace group API. A new component's empty builder had no DQL
entry action; Load DQL now opens an import fixed to that component and creates the
first draft. Targeted rendered tests and both actual UI imports verify the flow.

The initial check found empty catalogs for new readers without policy documents.
This is now resolved by artifact-scoped public defaults for discover, describe
and execute. The defaults apply only to the exact component IDs/versions compiled
into that artifact and retain namespace and deployment tenant verification.
Explicit documents remain authoritative, including missing actions and protected
rules. Unknown resources, database failures and policy administration never
receive a default. The generic authz store now distinguishes a missing policy
head from a damaged head whose revision is absent; the latter denies access.

After rebuilding with the published authz integrity fix, independent concurrent
MCP clients discover and execute Alpha and Beta on their separate ports. Each
catalog contains only its own tool, returns the expected marker, rejects the
other namespace's tool, and hides/denies private access without a token. The
private verification script retains these checks. These use signed synthetic
owner identities, not a claim of real Viant IDP forecasting grants.

Both exact-version UI previews returned their respective markers (101 and 202)
after both publications became active. The initial named-tool calls were denied as well as discovery; the artifact
public-default correction now passes both protocol paths. GPT-6 Sol granted scoped approval to the new empty-builder Load DQL
entry and fixed-resource import flow. It did not approve full MCP acceptance.

## Forecasting and skill regression on a namespace endpoint

The isolated forecasting namespace remains private with no viewer-role additions.
Its MCP endpoint was enabled through the UI on port 18893. The latest custom
runtime revalidated the UI-authored published v11 graph: seven dictionary
relations, composite country/region matching, nullable missing dictionaries,
representative predicate totals, publisher 127/147 isolation, role/exposure
discovery and execution denial, and client scope override rejection all pass.
Native skill discovery/get, MCP resource retrieval and compatibility discovery
also pass their anonymous, role and exposure denials. Negative fixture tokens
use the namespace owner subject, so they test the explicit resource policy after
passing namespace visibility. Alpha and Beta checks still pass concurrently.

These are synthetic signed-owner identities and synthetic viant-e2e data. Real
Viant IDP publisher grants, successful scoped UI preview and complete namespace
route/persistence acceptance remain open; no full-goal completion is claimed.

## Cross-version policy safety

Implicit public defaults now require that the component has no policy head in
any version for the configured tenant. A new version cannot erase an earlier
explicit policy by omitting its version-specific document. Such a version denies
runtime discovery/execution until its policy is configured. The presence check
is a private generated Datly reader; query failures deny access. Tests distinguish
an explicitly configured component from a never-configured one and cover lookup
failure. No version policy is silently copied or inferred from role claims.

## Native component update selection

The component-update contract now exposes namespace selection over HTTP and MCP
and uses the existing namespace-aware edit guard. Linked-host tests reject
same-owner updates from the wrong workspace, inspect unchanged title/revision,
and verify updates in the selected workspace. This completes that route's
selection contract; mandatory selection and remaining persistence scopes are
still separate acceptance work.

## Component ACL storage ownership

The private generated ACL writer now persists `namespace_id`. SDK writes derive
ownership from the stored component; native upsert/delete obtain it from the
namespace-aware component guard before the owner-only ACL workflow. Caller
payloads never supply ownership. Lifecycle checks require canonical ownership
and reject changes to an existing nonempty namespace. Persistence regressions
inspect native and SDK-created grants, and mutation tests reject missing or moved
ownership. This covers component ACL records, not generic policy-head ownership.

On 2026-09-29 the full forecasting runtime matrix passed against namespace MCP
port 18893: 262 input cases across 135 custom handler types, including built-in
predicates and explicit-false cases, with zero failures. It ran against the
current custom binary and published v11 graph using a signed synthetic namespace
owner, exercising live MCP schema binding and execution without changing the
predicate inventory or underlying test data. This strengthens fixture acceptance;
it does not supply real Viant IDP publisher permissions.

## Workspace-local resource namespace claims

Canonical schema version 19 keys named resource claims by workspace ID and
resource namespace together. Claim writes derive ownership from their stored
component, and the foreign-usage reader compares only components in the same
workspace. The SQLite upgrade rebuilds the claim table atomically, preserves
ownership/audit fields, and rejects orphaned or inconsistent claims. Canonical
MySQL DDL uses the same composite key.

SDK tests create the same owner.docs/shared.sql resource in Alpha and Beta at
once, verify content isolation and cross-workspace denial, then release Alpha
without affecting Beta. Same-workspace name conflicts remain enforced. Component
slug/ID identity is still global per owner; this change does not claim that
component slugs can be duplicated across workspaces.

The isolated UI catalog was upgraded to schema 19 through custom API startup.
Using the UI, draft v2 in Alpha and Beta now both contain awitas.docs/shared.sql,
with distinct marker SQL and distinct workspace ownership. Both saves succeeded;
the published v1 generations remain unchanged. The original Studio catalog was
not upgraded. SDK tests separately verify claim release and cross-workspace
read denial. Live MySQL upgrade acceptance remains unverified; canonical MySQL
creation uses the composite key.

## Workspace-local component slugs

Canonical schema version 20 scopes component slug uniqueness to workspace ID.
Component IDs remain independent server-generated identities. The SQLite upgrade
preserves component IDs/columns and child foreign-key targets while changing the
old global uniqueness constraint; it restores foreign-key state on its pinned
connection. Tests create the same reader slug in Alpha and Beta, reject a duplicate
within one workspace, and verify child ACL preservation/cascade after migration.
Older namespace foundations finish at version 19 before this separate upgrade.

The isolated UI catalog was upgraded to schema 20 through custom API startup.
The UI created shared-reader in both Alpha and Beta; persisted IDs and namespace
IDs are different, and each selected catalog displays only its own component.
The original Studio catalog remains untouched. The full Go suite and custom API/
runtime builds pass. Canonical MySQL DDL is updated; live MySQL migration is not
claimed by the SQLite acceptance check.

## Resuming a Studio window

Focus, page restoration and visibility restoration recheck the shared namespace
selection and renew any open-editor lease. If another window changed selection,
the resumed window blocks resource requests and preserves its open editor until
it closes; then it loads the new workspace. Storage failures block requests and
offer retry. A missing or invalid shared selection keeps the namespace selector
usable so the user can recover without reloading the application.

Fourteen rendered hook regressions cover these recovery paths; the full UI suite
passes 135 rendered tests plus contract tests. GPT-6 Sol granted scoped approval
for recovery and editor preservation. This does not establish complete concurrent
window coordination: expiring leases and simultaneous editor-open/switch races
still require broader acceptance.

## Permission resource discovery scope

Native `access.list` accepts the same optional namespace header/MCP argument as
component discovery. Its internal candidate reader includes persisted component
namespace ownership, including the owner component for skills. Candidates from
another workspace are discarded before component visibility checks, policy
inspection fallback, grouping or pagination. A policy administration grant cannot
bypass this workspace boundary. The SQL SDK catalog uses the same independent
scope filter.

Native HTTP/MCP regression tests use two private namespaces owned by the same
principal and verify a one-item page contains only the selected component.
Catalog regressions additionally verify that a policy inspection grant cannot
reveal either a foreign component or skill or affect pagination. Typed component
invocation retains namespace selection in the component reader predicate and
rejects conflicting header/input selection.

## Policy administration scope

Native policy get/context/replace operations now carry optional namespace
selection through HTTP and MCP. Before invoking authz policy operations, Studio
resolves the exact component/skill version from persisted catalog ownership and
checks its owning component's namespace visibility. Unknown resources, versions,
invalid selections, and foreign ownership fail closed. Existing policy actions,
tenant checks, predefined role validation and revision CAS remain independent.

The authenticated SDK access wrapper applies the same check to policy operations
and catalog discovery; it cannot bypass the inner SQL transport's selected
workspace by handling these operations directly. Native HTTP/MCP tests reject all
three management operations against the other namespace even with explicit policy
grants for the same principal in both workspaces. Native local policy read and
editor context succeed. SDK tests retain local
policy reads and reject foreign components/skills before component or policy
access. Selection is still optional on these routes; mandatory selection across
resource APIs remains an acceptance gate.
