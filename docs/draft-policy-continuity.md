# Draft policy continuity

The native `versions.clone` operation copies DQL, column metadata, files, folders,
skill roots and explicit source-version access policies in one managed transaction.
The previous browser-side clone omitted policies, producing a valid draft that
its authorized author could not preview. Exact-version enforcement correctly
denied that draft.

The signed local forecasting acceptance establishes this gap. The component
run-access reader returns one matching row. The same verified role, feature and
publisher facts authorize bounded execution of published v11 and deny draft v12.
The catalog has a v11 policy head and no v12 head. This is separate from the still
unresolved real Viant publisher-grant source.

## Implementation

An explicit server-owned draft clone should preserve source-version policies as
immutable initial policies for the new version. The client selects only the
source version of the same authorized component; it cannot supply policy content,
an arbitrary policy resource, tenant or role definitions. The server checks source
visibility, target authoring permission and namespace ownership. Existing role,
feature, entity, management and discovery restrictions remain identical.

Component and explicit skill policies must be retained, including tenant and
version identity. Inherited skill access continues to follow its component.
Missing source policies remain missing; a protected component must not become
public. Corrupt heads, absent referenced revisions and write conflicts abort the
clone. Policy administration still requires its independent grants.

Version creation, policy copying and provenance must use the existing Datly
reader/writer components within one managed transaction. A failed clone must
leave no partial version, policy head, history or resource links. Verify the
shared transaction across the Studio/authz connector aliases before relying on
it. Datly now supports explicit `AliasOf: studio`, giving both names one handle
and transaction identity. Its nested-write regression flushes version, policy
head and history markers and verifies a later parent failure rolls all three
back. Studio's native configuration uses this alias. Both browser entry points
now invoke the single native clone operation.

## Acceptance

- Native HTTP/MCP and in-process SDK clone the exact same source policy snapshot.
- Authorized preview of the new draft retains bounded entity scope; wrong role,
  missing feature, wrong entity and client scope overrides remain denied.
- A source outside the selected namespace is rejected, even for the same owner.
- Source policies stay immutable; target policy changes require manageAccess.
- Injected writer failures roll back the version and all policy/history rows.
- UI cloning communicates policy continuity and partial/error outcomes clearly;
  GPT-6 Sol reviews the actual successful preview flow.

Native integration tests exercise HTTP, MCP and in-process SDK cloning, revision
conflicts, namespace denial, bounded role/feature/entity policy evaluation, corrupt
source heads and injected policy-writer rollback after resource writes. A signed
synthetic UI identity cloned published forecasting v11 into v13 with the policy
preserved and validated revision 9.

Full-flow UX approval remains a separate acceptance gate. Revision 9 exposed a
reader contract issue: selected cube fields were
rejected because source field projection was disabled. Component settings now
offer an explicit field-selection toggle; unrelated saves retain the authored
permission, and the UI states that enabling it also affects the source reader.
This does not change authentication or entity predicates. Synthetic UI revision
10 validated and executed the selected cube. A second issue affected native SDK
preview: an enclosing handler's output frame prevented the preview from publishing
its own selection. The SQLite regression reproduces the extra unselected fields;
isolating the separately encoded child frame fixes it and the live UI now shows
only selected scalar columns.

The repeated-country issue came from cube derivation attaching a composite
country/region dictionary whenever either dimension was selected. That forced the
other join key into SQL projection and grouping. Datly commit `6980bd97` includes
a dictionary only when all its key dimensions are selected, preserving source
predicates and single-key lookup behavior. Studio pins its published module
version. The report, SQL and SQL-builder suites pass; a signed synthetic execution
returns three distinct countries and total avails 133943, matching an independent
query against the fabricated viant-e2e fixture for publisher 127 on 2026-09-27.
Real Viant publisher-grant integration remains unverified.
