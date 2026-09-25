package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	resourcestore "github.com/viant/datly-studio/sdk/transport/sql/internal/resources"
	_ "modernc.org/sqlite"
)

func TestResourceSnapshotReaderIsUnboundedAndOrdered(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	const count = 137
	for i := count - 1; i >= 0; i-- {
		id := fmt.Sprintf("%064x", i+1)
		name := fmt.Sprintf("assets/file-%03d.txt", i)
		content := []byte(fmt.Sprintf("value-%03d", i))
		var media any
		if i > 0 {
			media = "text/plain"
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO report_resource_files
			(report_id,version_no,resource_id,namespace,resource_path,media_type,content,content_size,content_sha256,is_binary,created_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?)`, "r1", 2, id, "assets", name, media, content, len(content), id, false, now); err != nil {
			t.Fatal(err)
		}
	}
	for _, folder := range []struct {
		id, root string
		ordinal  int
	}{{"b", "second", 2}, {"a", "first", 1}} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO report_resource_folders
			(report_id,version_no,folder_id,namespace,root_path,uri_prefix,ordinal)
			VALUES(?,?,?,?,?,?,?)`, "r1", 2, folder.id, "assets", folder.root, "skill://"+folder.root+"/", folder.ordinal); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO report_skill_roots
			(report_id,version_no,skill_id,folder_id,skill_root,ordinal)
			VALUES(?,?,?,?,?,?)`, "r1", 2, folder.id, folder.id, folder.root, folder.ordinal); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	result := &sdk.ResourceSnapshot{}
	if err := resourcestore.ReadSnapshot(ctx, db, "r1", 2, result); err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != count || len(result.Folders) != 2 || len(result.Skills) != 2 {
		t.Fatalf("snapshot file/folder/skill counts=%d/%d/%d", len(result.Files), len(result.Folders), len(result.Skills))
	}
	if result.Files[0].ResourcePath != "assets/file-000.txt" || result.Files[0].Content != "value-000" ||
		result.Files[0].MediaType != "" || result.Files[count-1].ResourcePath != "assets/file-136.txt" ||
		result.Folders[0].FolderID != "a" || result.Skills[0].SkillID != "a" {
		t.Fatalf("snapshot ordering/content: first=%+v last=%+v folders=%+v skills=%+v",
			result.Files[0], result.Files[count-1], result.Folders, result.Skills)
	}
}
