package migrate

import (
	"context"
	"testing"

	"github.com/viant/datly-studio/schema"
)

func TestNamespaceSkillUpgradePreservesExistingCatalog(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,status,etag,created_at,updated_at) VALUES('ns-a','owner','a','Existing namespace','active',7,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP); DROP TABLE namespace_skill_revisions; DROP TABLE namespace_skills`); err != nil {
		t.Fatal(err)
	}
	if err := schema.SetSQLiteVersion(ctx, db, 22); err != nil {
		t.Fatal(err)
	}
	service, _ := New()
	for i := 0; i < 2; i++ {
		if err := service.Up(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	version, err := service.CurrentVersion(ctx, db)
	if err != nil || version != schema.CanonicalVersion {
		t.Fatalf("version=%d err=%v", version, err)
	}
	var title string
	var etag int
	if err := db.QueryRowContext(ctx, `SELECT title,etag FROM namespaces WHERE namespace_id='ns-a'`).Scan(&title, &etag); err != nil || title != "Existing namespace" || etag != 7 {
		t.Fatalf("namespace changed: title=%s etag=%d err=%v", title, etag, err)
	}
	for _, ns := range []string{"ns-a", "ns-b"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO namespace_skills(namespace_id,skill_id,name,title,description,latest_revision,created_at,created_by,updated_at,updated_by) VALUES(?,?,'guide','Guide','Namespace guide',1,CURRENT_TIMESTAMP,'owner',CURRENT_TIMESTAMP,'owner')`, ns, ns+"-skill"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO namespace_skill_revisions(namespace_id,skill_id,revision,bundle_json,content_sha256,created_at,created_by,updated_at,updated_by) VALUES('ns-a','ns-a-skill',1,'{}','digest',CURRENT_TIMESTAMP,'owner',CURRENT_TIMESTAMP,'owner')`); err != nil {
		t.Fatal(err)
	}
	var components int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM components`).Scan(&components); err != nil || components != 0 {
		t.Fatalf("namespace skills required a component: count=%d err=%v", components, err)
	}
	for _, statement := range []string{
		`UPDATE namespace_skills SET published_revision=2 WHERE namespace_id='ns-a'`,
		`INSERT INTO namespace_skill_revisions(namespace_id,skill_id,revision,bundle_json,content_sha256,created_at,created_by,updated_at,updated_by) VALUES('ns-b','ns-a-skill',1,'{}','other',CURRENT_TIMESTAMP,'owner',CURRENT_TIMESTAMP,'owner')`,
		`INSERT INTO namespace_skill_revisions(namespace_id,skill_id,revision,bundle_json,content_sha256,created_at,created_by,updated_at,updated_by) VALUES('ns-a','ns-a-skill',1,'{}','other',CURRENT_TIMESTAMP,'owner',CURRENT_TIMESTAMP,'owner')`,
	} {
		if _, err := db.ExecContext(ctx, statement); err == nil {
			t.Fatalf("accepted invalid skill state: %s", statement)
		}
	}
}
