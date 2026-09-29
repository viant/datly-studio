# Draft policy continuity: identified gap

Creating an editable draft currently copies DQL, column metadata, files, folders
and skill roots. It does not preserve the source version's explicit access
policies. A protected published component can therefore produce a valid draft
that its authorized author cannot preview. Exact-version enforcement correctly
denies the new version; relaxing it would erase protection.

The signed local forecasting acceptance establishes this gap. The component
run-access reader returns one matching row. The same verified role, feature and
publisher facts authorize bounded execution of published v11 and deny draft v12.
The catalog has a v11 policy head and no v12 head. This is separate from the still
unresolved real Viant publisher-grant source.

## Required implementation

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
back. Studio's native configuration uses this alias. The existing browser-side
multi-operation clone still does not establish atomicity for the real clone.

## Acceptance

- Native HTTP/MCP and in-process SDK clone the exact same source policy snapshot.
- Authorized preview of the new draft retains bounded entity scope; wrong role,
  missing feature, wrong entity and client scope overrides remain denied.
- A source outside the selected namespace is rejected, even for the same owner.
- Source policies stay immutable; target policy changes require manageAccess.
- Injected writer failures roll back the version and all policy/history rows.
- UI cloning communicates policy continuity and partial/error outcomes clearly;
  GPT-6 Sol reviews the actual successful preview flow.

This implementation remains outstanding. It is not replaced by test-only policy
provisioning or by treating an absent target policy as public.
