package migrate

import (
	"context"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/schema"
	"strings"
	"testing"
)

func TestComponentSlugMigrationPreservesIDsChildrenAndForeignKeys(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	db.SetMaxOpenConns(1)
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DROP TABLE components"); err != nil {
		t.Fatal(err)
	}
	ddl, err := schema.SQLiteTableStatements("components")
	if err != nil {
		t.Fatal(err)
	}
	ddl = strings.Replace(ddl, "UNIQUE (namespace_id, slug)", "UNIQUE (slug)", 1)
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	a, b := namespaceaccess.ID("owner", "alpha"), namespaceaccess.ID("owner", "beta")
	for _, query := range []string{
		`CREATE TABLE schema_version(version INTEGER NOT NULL)`,
		`INSERT INTO schema_version VALUES(19)`,
		`INSERT INTO connectors(name,driver,owner_id,status,created_at,updated_at) VALUES('shared','sqlite','owner','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO namespaces(owner_id,name,title,status,created_at,updated_at) VALUES('owner','alpha','Alpha','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),('owner','beta','Beta','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
	} {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	for _, entry := range []struct{ id, name string }{{a, "alpha"}, {b, "beta"}} {
		if _, err := db.ExecContext(ctx, `UPDATE namespaces SET namespace_id=? WHERE name=?`, entry.id, entry.name); err != nil {
			t.Fatal(err)
		}
	}
	insert := `INSERT INTO components(namespace_id,id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,created_at,updated_at) VALUES(?,? ,?,'reader','Reader','owner','draft','shared',?,'reader',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`
	if _, err := db.ExecContext(ctx, insert, a, "a", "alpha", "scope/a"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO component_acl(namespace_id,report_id,subject_type,subject_id,can_view) VALUES(?,'a','user','viewer',1)`, a); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}
	service, _ := New()
	if err := service.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	var state int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&state); err != nil || state != 1 {
		t.Fatalf("foreign keys not restored: %d %v", state, err)
	}
	if _, err := db.ExecContext(ctx, insert, b, "b", "beta", "scope/b"); err != nil {
		t.Fatalf("namespace-local slug rejected: %v", err)
	}
	if _, err := db.ExecContext(ctx, insert, a, "another", "alpha", "scope/another"); err == nil {
		t.Fatal("same-workspace duplicate slug accepted")
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_acl WHERE report_id='a'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("child lost in migration: %d %v", count, err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM components WHERE id='a'"); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_acl WHERE report_id='a'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("foreign key target changed: %d %v", count, err)
	}
	if err := service.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
}
