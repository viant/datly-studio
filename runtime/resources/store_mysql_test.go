package resources

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/schema"
	_ "github.com/viant/sqlx/metadata/product/mysql"
)

func TestResourceSnapshotMySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_RESOURCE_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set STUDIO_RESOURCE_MYSQL_DSN to an empty disposable MySQL database")
	}
	ctx := context.Background()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = schema.Initialize(ctx, db, "mysql"); err != nil {
		t.Fatal(err)
	}
	f := &snapshotFixture{t: t, db: db}
	now := time.Now().UTC()
	f.exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,mcp_enabled,etag,created_at,updated_at) VALUES(?,'owner','general','Fixture','active',TRUE,1,?,?)`, strings.Repeat("a", 64), now, now)
	f.exec(`INSERT INTO connectors(name,driver,owner_id,status,options_json,etag,created_at,updated_at) VALUES('main','mysql','owner','active','{}',1,?,?)`, now, now)
	f.version("first", 1)
	f.version("first", 2)
	f.version("second", 1)
	f.file("first", 1, "old", "old_docs", "old.txt", "old")
	f.file("first", 2, "guide", "first_docs", "guide/SKILL.md", "---\nname: guide\ndescription: Synthetic guide\n---\nInstructions.")
	f.file("second", 1, "other", "second_docs", "other.txt", "other")
	f.folder("first", 2, "guide", "first_docs", "guide", "skill://example-guide/", 0)
	f.skill("first", 2, "guide", "guide", ".", 0)
	loaded, err := Load(ctx, db, []Version{{ReportID: "first", VersionNo: 2}, {ReportID: "second", VersionNo: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.ByVersion) != 2 || len(loaded.Folders) != 1 || len(loaded.Folders[0].Skills) != 1 {
		t.Fatalf("incomplete snapshot: %+v", loaded)
	}
	if _, err = loaded.ByVersion[Version{ReportID: "first", VersionNo: 2}].ReadFile("old.txt"); err == nil {
		t.Fatal("unselected revision leaked")
	}
	if body, err := loaded.ByVersion[Version{ReportID: "second", VersionNo: 1}].ReadFile("other.txt"); err != nil || string(body) != "other" {
		t.Fatalf("second reader: %q %v", body, err)
	}
}
