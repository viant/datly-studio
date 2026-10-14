package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"time"

	access "github.com/viant/authz"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/namespaceskill"
)

func TestNamespaceSkillPublishedSnapshotWithoutComponents(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "namespace_skill_publication", "studio")
	first, err := namespaceskill.Compile(ctx, "ns-a", "guide", 1, namespaceskill.Bundle{Content: "---\nname: guide\ndescription: Published description\n---\nPublished content."})
	if err != nil {
		t.Fatal(err)
	}
	second, err := namespaceskill.Compile(ctx, "ns-a", "guide", 2, namespaceskill.Bundle{Content: "---\nname: guide\ndescription: Draft description\n---\nDraft content."})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, ns := range []string{"ns-a", "ns-b"} {
		_, err = db.Exec(`INSERT INTO namespace_skills(namespace_id,skill_id,name,title,description,latest_revision,published_revision,status,etag,created_at,created_by,updated_at,updated_by) VALUES(?,?,?,?,?,2,1,'active',3,?,?,?,?)`, ns, "guide", "guide", "Draft title", "Draft description", now, "owner", now, "owner")
		if err != nil {
			t.Fatal(err)
		}
		for i, snapshot := range []*namespaceskill.Snapshot{first, second} {
			_, err = db.Exec(`INSERT INTO namespace_skill_revisions(namespace_id,skill_id,revision,bundle_json,content_sha256,created_at,created_by,updated_at,updated_by) VALUES(?,?,?,?,?,?,?,?,?)`, ns, "guide", i+1, string(snapshot.JSON), snapshot.SHA256, now, "owner", now, "owner")
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	service := &Service{studio: db, config: Config{NamespaceID: "ns-a"}}
	folders, uris, err := service.namespaceSkillFolders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || len(uris) != 1 || folders[0].Description != "Published description" {
		t.Fatalf("draft metadata leaked: %+v %v", folders, uris)
	}
	content, err := fs.ReadFile(folders[0].FS, "SKILL.md")
	if err != nil || string(content) != first.Bundle.Content {
		t.Fatalf("published content: %q %v", content, err)
	}
	// With no component rows, independent skills still publish; other namespaces
	// and drafts never join the listener's inventory.
	var components int
	if err = db.QueryRow(`SELECT COUNT(*) FROM components`).Scan(&components); err != nil || components != 0 {
		t.Fatalf("unexpected components: %d %v", components, err)
	}
	if _, err = db.Exec(`UPDATE namespace_skills SET published_revision=2 WHERE namespace_id='ns-a'`); err != nil {
		t.Fatal(err)
	}
	next, _, err := service.namespaceSkillFolders(ctx)
	if err != nil || len(next) != 1 || next[0].Description != "Draft description" {
		t.Fatalf("publish second revision: %v", err)
	}
	retained, _ := fs.ReadFile(folders[0].FS, "SKILL.md")
	if string(retained) != first.Bundle.Content {
		t.Fatal("published snapshot mutated")
	}
	if _, err = db.Exec(`UPDATE namespace_skill_revisions SET bundle_json='{}' WHERE namespace_id='ns-a' AND revision=2`); err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.namespaceSkillFolders(ctx); err == nil {
		t.Fatal("corrupt publication accepted")
	}
	if _, err = db.Exec(`UPDATE namespace_skills SET published_revision=0 WHERE namespace_id='ns-a'`); err != nil {
		t.Fatal(err)
	}
	if remaining, _, err := service.namespaceSkillFolders(ctx); err != nil || len(remaining) != 0 {
		t.Fatalf("unpublish: %v", err)
	}
	service.config.NamespaceID = ""
	if remaining, _, err := service.namespaceSkillFolders(ctx); err != nil || len(remaining) != 0 {
		t.Fatalf("root runtime aggregated namespaces: %v", err)
	}
}

func TestNamespaceSkillAuthorizerExactInventoryAndExplicitPolicy(t *testing.T) {
	uri := "skill://ns-a/guide/1/guide/SKILL.md"
	denied := errors.New("component denied")
	service := &Service{}
	inventory := map[string]bool{uri: true}
	authorize := service.namespaceSkillAuthorizer(inventory, func(context.Context, string, string) error { return denied })
	inventory[uri+"/private"] = true
	for _, action := range []string{"discover", "describe", "retrieve"} {
		if err := authorize(context.Background(), uri, action); err != nil {
			t.Fatal(err)
		}
	}
	for _, other := range []string{uri + "/private", "skill://ns-b/guide/1/guide/SKILL.md", uri + "?other=1"} {
		if err := authorize(context.Background(), other, "retrieve"); !errors.Is(err, denied) {
			t.Fatalf("unowned URI permitted: %s %v", other, err)
		}
	}
	resource := access.Resource{Kind: "skill", ID: "guide", Version: "1"}
	service.config.Access = &ResourceAccessConfig{ResourceBindings: map[string]access.Resource{"skill://ns-a/": resource}}
	service.resourceAccess = &access.Service{Store: resourceStore{documents: map[access.Resource]access.Document{resource: {Resource: resource, Revision: 1, Policies: map[string]access.Policy{"retrieve": {Mode: "protected", Rule: &access.Rule{Kind: "role", Value: "other"}}}}}}, Provider: runtimeFacts{}}
	if err := authorize(context.Background(), uri, "retrieve"); err == nil {
		t.Fatal("namespace ownership bypassed explicit binding")
	}
}

func TestNamespaceSkillCatalogPagination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db := datatest.OpenSQLite(t, "namespace_skill_pages", "studio")
	for i := 0; i < 105; i++ {
		name := fmt.Sprintf("guide-%03d", i)
		snapshot, err := namespaceskill.Compile(ctx, "ns-pages", name, 1, namespaceskill.Bundle{Content: "---\nname: " + name + "\ndescription: Paginated guide\n---\nGuide."})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(`INSERT INTO namespace_skills(namespace_id,skill_id,name,title,description,latest_revision,published_revision,status,etag,created_at,created_by,updated_at,updated_by) VALUES('ns-pages',?,?,?,'Paginated guide',1,1,'active',2,CURRENT_TIMESTAMP,'owner',CURRENT_TIMESTAMP,'owner')`, name, name, name); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(`INSERT INTO namespace_skill_revisions(namespace_id,skill_id,revision,bundle_json,content_sha256,created_at,created_by,updated_at,updated_by) VALUES('ns-pages',?,1,?,?,CURRENT_TIMESTAMP,'owner',CURRENT_TIMESTAMP,'owner')`, name, string(snapshot.JSON), snapshot.SHA256); err != nil {
			t.Fatal(err)
		}
	}
	service := &Service{studio: db, config: Config{NamespaceID: "ns-pages"}}
	folders, uris, err := service.namespaceSkillFolders(ctx)
	if err != nil || len(folders) != 105 || len(uris) != 105 {
		t.Fatalf("pagination: folders=%d uris=%d err=%v", len(folders), len(uris), err)
	}
}
