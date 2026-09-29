package sqltransport

import (
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/sdk"
	"testing"
)

// Exercise the server operations used by cloneReaderDraft with real Datly
// writers and SQLite, preserving published resources and their version links.
func TestDraftCopyPreservesNamespaceAndPublishedSource(t *testing.T) {
	f := newImportFixture(t)
	id := namespaceaccess.ID(f.report.OwnerID, f.report.Namespace)
	ctx := sdk.WithNamespaceSelection(f.ctx, id)
	other, err := f.client.Namespaces().Create(f.ctx, sdk.CreateNamespaceInput{Name: "other", Title: "Other"})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := f.client.Versions().LoadArchive(ctx, f.report.ID, sdk.LoadArchiveInput{Archive: zipArchive(t, map[string][]byte{
		"main.dql":       []byte("SELECT 1"),
		"guide/SKILL.md": []byte("---\nname: copy-guide\ndescription: Fixture guide\n---\nUse this guide."),
	}), Format: "zip", EntryDQL: "main.dql"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := f.client.Resources().Get(ctx, f.report.ID, loaded.Version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	named := snapshot.Files[0].Namespace
	snapshot, err = f.client.Resources().UpsertFolder(ctx, sdk.ResourceFolder{ReportID: f.report.ID, VersionNo: loaded.Version.VersionNo, Namespace: named, RootPath: "guide", URIPrefix: "skill://copy-guide/", ExpectedSourceRevision: snapshot.Version.SourceRevision})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err = f.client.Resources().UpsertSkill(ctx, sdk.SkillRoot{ReportID: f.report.ID, VersionNo: loaded.Version.VersionNo, FolderID: snapshot.Folders[0].FolderID, SkillRoot: ".", ExpectedSourceRevision: snapshot.Version.SourceRevision})
	if err != nil {
		t.Fatal(err)
	}
	sourceRevision := snapshot.Version.SourceRevision
	if _, err = f.db.Exec(`UPDATE report_versions SET state='published' WHERE report_id=? AND version_no=?`, f.report.ID, loaded.Version.VersionNo); err != nil {
		t.Fatal(err)
	}
	source, err := f.client.Versions().Get(ctx, f.report.ID, loaded.Version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := f.client.Versions().Create(ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: source.AuthoringMode, AuthoredDQL: source.AuthoredDQL, ComponentSpec: source.ComponentSpec, Notes: "Draft copy"})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range snapshot.Files {
		copy := *file
		copy.VersionNo, copy.ExpectedSourceRevision = draft.VersionNo, draft.SourceRevision
		result, err := f.client.Resources().UpsertFile(ctx, copy)
		if err != nil {
			t.Fatal(err)
		}
		draft = result.Version
	}
	for _, folder := range snapshot.Folders {
		copy := *folder
		copy.VersionNo, copy.ExpectedSourceRevision = draft.VersionNo, draft.SourceRevision
		result, err := f.client.Resources().UpsertFolder(ctx, copy)
		if err != nil {
			t.Fatal(err)
		}
		draft = result.Version
	}
	for _, skill := range snapshot.Skills {
		copy := *skill
		copy.VersionNo, copy.ExpectedSourceRevision = draft.VersionNo, draft.SourceRevision
		result, err := f.client.Resources().UpsertSkill(ctx, copy)
		if err != nil {
			t.Fatal(err)
		}
		draft = result.Version
	}
	cloned, err := f.client.Resources().Get(ctx, f.report.ID, draft.VersionNo)
	if err != nil || len(cloned.Files) != 2 || len(cloned.Folders) != 1 || len(cloned.Skills) != 1 || cloned.Skills[0].FolderID != cloned.Folders[0].FolderID {
		t.Fatalf("copied graph=%+v err=%v", cloned, err)
	}
	for _, table := range []string{"report_versions", "report_resource_files", "report_resource_folders", "report_skill_roots"} {
		var wrong int
		if err := f.db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE report_id=? AND namespace_id<>?", f.report.ID, id).Scan(&wrong); err != nil || wrong != 0 {
			t.Fatalf("%s wrong namespace=%d err=%v", table, wrong, err)
		}
	}
	original, err := f.client.Versions().Get(ctx, f.report.ID, source.VersionNo)
	if err != nil || original.State != "published" || original.SourceRevision != sourceRevision || original.AuthoredDQL != source.AuthoredDQL {
		t.Fatalf("published source changed=%+v err=%v", original, err)
	}
	if _, err := f.client.Versions().Create(sdk.WithNamespaceSelection(f.ctx, other.NamespaceID), f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: source.AuthoredDQL}); err == nil {
		t.Fatal("copy created in another selected namespace")
	}
	var count int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM report_versions WHERE report_id=?`, f.report.ID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("copy denial changed versions=%d err=%v", count, err)
	}
}
