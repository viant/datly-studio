package sqltransport

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
)

func TestDownloadIncludesMoreThanOneHundredResourceFiles(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	connector, err := client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "source", Driver: "sqlite", DSNTemplate: "file:source.db"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE connectors SET status='active' WHERE name=?`, connector.Name); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(ctx, sdk.CreateReportInput{Slug: "many-files", Title: "Many files", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := client.Versions().LoadDQL(ctx, report.ID, sdk.LoadDQLInput{DQL: "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	const count = 137
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("assets/file-%03d.txt", i)
		content := []byte(fmt.Sprintf("contents-%03d", i))
		_, err := tx.ExecContext(ctx, `INSERT INTO report_resource_files
 (report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at)
 VALUES(?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`, report.ID, loaded.Version.VersionNo,
			fmt.Sprintf("%064x", i+1), "assets", name, content, len(content), fmt.Sprintf("%064x", i+1), false)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var storedFiles int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM report_resource_files WHERE report_id=? AND version_no=?`,
		report.ID, loaded.Version.VersionNo).Scan(&storedFiles); err != nil {
		t.Fatal(err)
	}
	download, err := client.Versions().Download(ctx, report.ID, loaded.Version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	if download.Filename != "component-v1.zip" || download.MediaType != "application/zip" || len(download.Files) != storedFiles+1 {
		t.Fatalf("download metadata: filename=%q media=%q files=%d", download.Filename, download.MediaType, len(download.Files))
	}
	bundle, err := sdk.ReadDQLArchive(bytes.NewReader(download.Archive), "zip")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Files) != storedFiles+1 || string(bundle.Files[download.EntryDQL]) != "SELECT 1" {
		t.Fatalf("archive entries=%d entry=%q", len(bundle.Files), download.EntryDQL)
	}
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("assets/file-%03d.txt", i)
		want := fmt.Sprintf("contents-%03d", i)
		if got := string(bundle.Files[name]); got != want {
			t.Fatalf("%s=%q, want %q", name, got, want)
		}
	}
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{name: "duplicate", path: "assets/file-000.txt", want: "duplicate resource path"},
		{name: "unsafe", path: "../escape.txt", want: "unsafe resource path"},
	} {
		t.Run(test.name, func(t *testing.T) {
			resourceID := fmt.Sprintf("%064x", 1000+len(test.name))
			_, err := db.ExecContext(ctx, `INSERT INTO report_resource_files
 (report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at)
 VALUES(?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`, report.ID, loaded.Version.VersionNo,
				resourceID, "assets", test.path, []byte("bad"), 3, resourceID, false)
			if err != nil {
				t.Fatal(err)
			}
			defer db.ExecContext(ctx, `DELETE FROM report_resource_files WHERE report_id=? AND version_no=? AND resource_id=?`,
				report.ID, loaded.Version.VersionNo, resourceID)
			_, err = client.Versions().Download(ctx, report.ID, loaded.Version.VersionNo)
			var sdkErr *sdk.Error
			if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorInvalidArgument || !strings.Contains(sdkErr.Message, test.want) {
				t.Fatalf("download error=%v, want %q invalid argument", err, test.want)
			}
		})
	}
}
