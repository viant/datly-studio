package sqltransport

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	imported "github.com/viant/datly-studio/studio/report_versions/store_import"
	xhandler "github.com/viant/xdatly/handler"
)

type importFixture struct {
	ctx       context.Context
	db        *sql.DB
	transport *Transport
	client    sdk.Client
	report    *sdk.Report
	now       time.Time
	bump      atomic.Bool
}

// newImportFixture prepares a report whose owner can import DQL. Now is fixed
// so audit timestamps are asserted exactly; when bump is set the clock call
// inside the import simulates a concurrent report change.
func newImportFixture(t *testing.T) *importFixture {
	return newImportFixtureDSN(t, "")
}

// newImportFixtureDSN accepts the DSN query the deployment uses, such as the
// Studio API's shared-cache mode, so lock behavior is exercised as deployed.
func newImportFixtureDSN(t *testing.T, query string) *importFixture {
	t.Helper()
	fixture := &importFixture{ctx: sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"}),
		now: time.Date(2026, 9, 24, 12, 30, 0, 0, time.UTC)}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db")+query)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = schema.ApplySQLite(fixture.ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	fixture.db = db
	fixture.transport = &Transport{DB: db, Authorizer: allowAuthorizer{}, Now: func() time.Time {
		if fixture.bump.Load() && fixture.report != nil {
			if _, err := db.Exec(`UPDATE reports SET etag=etag+1 WHERE id=?`, fixture.report.ID); err != nil {
				t.Error(err)
			}
		}
		return fixture.now
	}}
	client, err := sdk.NewClient(fixture.transport)
	if err != nil {
		t.Fatal(err)
	}
	fixture.client = client
	connector, err := client.Connectors().Create(fixture.ctx, sdk.CreateConnectorInput{Name: "source", Driver: "sqlite", DSNTemplate: "file:source.db"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name=?`, connector.Name); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(fixture.ctx, sdk.CreateReportInput{Slug: "imported", Title: "Imported", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	fixture.report = report
	return fixture
}

func zipArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func TestDQLImportPersistsExactRowsThroughStoreComponents(t *testing.T) {
	f := newImportFixture(t)
	binary := []byte{0xff, 0xfe, 0x00, 0x01, 0x80}
	files := map[string][]byte{"main.dql": []byte("SELECT 1"), "assets/logo.bin": binary, "sql/q.sql": []byte("SELECT 1 AS q")}
	loaded, err := f.client.Versions().LoadArchive(f.ctx, f.report.ID, sdk.LoadArchiveInput{Archive: zipArchive(t, files), Format: "zip", EntryDQL: "main.dql", Notes: " keep spacing "})
	if err != nil {
		t.Fatal(err)
	}
	version := loaded.Version
	if version.VersionNo != 1 || version.State != "draft" || version.AuthoringMode != "dql" || version.CompileStatus != "pending" ||
		version.AuthoredDQL != "SELECT 1" || version.GeneratedDQL != "SELECT 1" || len(loaded.Files) != 3 {
		t.Fatalf("version=%+v files=%v", version, loaded.Files)
	}
	var specFormat, datlyVersion, compilerVersion, createdBy, spec, manifest, notes string
	var sourceRevision int64
	var createdAt time.Time
	err = f.db.QueryRow(`SELECT spec_format_version,datly_version,compiler_version,created_by,component_spec_json,type_manifest_json,notes,source_revision,created_at FROM report_versions WHERE report_id=? AND version_no=1`, f.report.ID).
		Scan(&specFormat, &datlyVersion, &compilerVersion, &createdBy, &spec, &manifest, &notes, &sourceRevision, &createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if specFormat != "studio.v1" || datlyVersion != "v1" || compilerVersion != "studio.v1" || createdBy != "owner" || spec != "{}" || manifest != "{}" ||
		notes != " keep spacing " || sourceRevision != 1 || !createdAt.Equal(f.now) {
		t.Fatalf("version row: %q %q %q %q %q %q %q %d %s", specFormat, datlyVersion, compilerVersion, createdBy, spec, manifest, notes, sourceRevision, createdAt)
	}
	rows, err := f.db.Query(`SELECT resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at FROM report_resource_files WHERE report_id=? AND version_no=1 ORDER BY resource_path`, f.report.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	reportDigest := sha256.Sum256([]byte(f.report.ID))
	wantNamespace := fmt.Sprintf("%s.imports.%x", f.report.OwnerPackage, reportDigest[:8])
	count := 0
	for rows.Next() {
		var resourceID, namespace, path, sha string
		var content []byte
		var size int64
		var isBinary bool
		var fileCreatedAt time.Time
		if err = rows.Scan(&resourceID, &namespace, &path, &content, &size, &sha, &isBinary, &fileCreatedAt); err != nil {
			t.Fatal(err)
		}
		want := files[path]
		if want == nil || !bytes.Equal(content, want) || resourceID != digest([]byte(path)) || sha != digest(want) || size != int64(len(want)) ||
			namespace != wantNamespace || isBinary != (path == "assets/logo.bin") || !fileCreatedAt.Equal(f.now) {
			t.Fatalf("resource row %s: id=%s ns=%s size=%d sha=%s binary=%v at=%s", path, resourceID, namespace, size, sha, isBinary, fileCreatedAt)
		}
		count++
	}
	if count != 3 {
		t.Fatalf("resource rows=%d", count)
	}
	current, err := f.client.Reports().Get(f.ctx, f.report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.ETag != f.report.ETag+1 || current.CurrentDraftVersion == nil || *current.CurrentDraftVersion != 1 || !current.UpdatedAt.Equal(f.now) {
		t.Fatalf("report after import: etag=%d (was %d) draft=%v updated=%s", current.ETag, f.report.ETag, current.CurrentDraftVersion, current.UpdatedAt)
	}
	download, err := f.client.Versions().Download(f.ctx, f.report.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := sdk.ReadDQLArchive(bytes.NewReader(download.Archive), "zip")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bundle.Files["assets/logo.bin"], binary) || string(bundle.Files["sql/q.sql"]) != "SELECT 1 AS q" {
		t.Fatalf("round trip lost binary or text resources: %v", bundle.Entries)
	}
	second, err := f.client.Versions().LoadDQL(f.ctx, f.report.ID, sdk.LoadDQLInput{DQL: "SELECT 2"})
	if err != nil {
		t.Fatal(err)
	}
	var secondNotes sql.NullString
	if err = f.db.QueryRow(`SELECT notes FROM report_versions WHERE report_id=? AND version_no=?`, f.report.ID, second.Version.VersionNo).Scan(&secondNotes); err != nil {
		t.Fatal(err)
	}
	if second.Version.VersionNo != 2 || secondNotes.Valid {
		t.Fatalf("second import version=%d notes=%v", second.Version.VersionNo, secondNotes)
	}
	current, err = f.client.Reports().Get(f.ctx, f.report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.ETag != f.report.ETag+2 || *current.CurrentDraftVersion != 2 {
		t.Fatalf("report after second import: etag=%d draft=%v", current.ETag, current.CurrentDraftVersion)
	}
}

func TestDQLImportRollsBackWhenReportChangesConcurrently(t *testing.T) {
	f := newImportFixture(t)
	f.bump.Store(true)
	_, err := f.client.Versions().LoadDQL(f.ctx, f.report.ID, sdk.LoadDQLInput{DQL: "SELECT 1"})
	f.bump.Store(false)
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict || !strings.Contains(sdkErr.Message, "modified concurrently") {
		t.Fatalf("stale report etag error=%v", err)
	}
	var versions, resources int
	if err = f.db.QueryRow(`SELECT COUNT(*) FROM report_versions WHERE report_id=?`, f.report.ID).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if err = f.db.QueryRow(`SELECT COUNT(*) FROM report_resource_files WHERE report_id=?`, f.report.ID).Scan(&resources); err != nil {
		t.Fatal(err)
	}
	if versions != 0 || resources != 0 {
		t.Fatalf("rolled back import left versions=%d resources=%d", versions, resources)
	}
	current, err := f.client.Reports().Get(f.ctx, f.report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.CurrentDraftVersion != nil || current.ETag != f.report.ETag+1 {
		t.Fatalf("report after rolled back import: draft=%v etag=%d", current.CurrentDraftVersion, current.ETag)
	}
	loaded, err := f.client.Versions().LoadDQL(f.ctx, f.report.ID, sdk.LoadDQLInput{DQL: "SELECT 1"})
	if err != nil || loaded.Version.VersionNo != 1 {
		t.Fatalf("import after conflict: %+v %v", loaded, err)
	}
}

func TestDQLImportWriterRejectsDuplicateVersionAllocation(t *testing.T) {
	f := newImportFixture(t)
	if _, err := f.client.Versions().LoadDQL(f.ctx, f.report.ID, sdk.LoadDQLInput{DQL: "SELECT 1"}); err != nil {
		t.Fatal(err)
	}
	next, err := f.transport.nextVersionNo(f.ctx, f.report.ID)
	if err != nil || next != 2 {
		t.Fatalf("next version=%d err=%v", next, err)
	}
	tx, err := f.db.BeginTx(f.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	row := importedVersionRow(f.report.ID, 1, "SELECT 9", "hash-9", "", "owner", f.now)
	row.File = []*imported.ImportedResourceFile{importedResourceFileRow("ns", "main.dql", []byte("SELECT 9"), f.now)}
	row.Has.File = true
	err = f.transport.writeImportedVersion(f.ctx, tx, row)
	var sdkErr *sdk.Error
	if err == nil || !errors.As(classify(err, "report version", f.report.ID), &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
		t.Fatalf("duplicate version error=%v", err)
	}
}

func TestDQLImportWriterDeniesInconsistentInputBeforeWriting(t *testing.T) {
	f := newImportFixture(t)
	for _, test := range []struct {
		name string
		edit func(row *imported.ImportedVersion)
		want string
	}{
		{name: "digest", edit: func(row *imported.ImportedVersion) { row.File[0].SetContentSha256("0") }, want: "content digest"},
		{name: "identity", edit: func(row *imported.ImportedVersion) { row.File[0].SetResourceId("0") }, want: "SHA-256 of its path"},
		{name: "unsafe path", edit: func(row *imported.ImportedVersion) { row.File[0].SetResourcePath("../escape.dql") }, want: "unsafe resource path"},
		{name: "duplicate path", edit: func(row *imported.ImportedVersion) {
			row.File = append(row.File, importedResourceFileRow("ns", "main.dql", []byte("SELECT 1"), f.now))
		}, want: "duplicate resource path"},
		{name: "source not retained", edit: func(row *imported.ImportedVersion) { row.File[0].SetContent([]byte("SELECT 2")) }, want: "retained as a resource file"},
		{name: "no files", edit: func(row *imported.ImportedVersion) { row.File = nil }, want: "at least one resource file"},
		{name: "not a draft", edit: func(row *imported.ImportedVersion) { row.SetState("published") }, want: "must be drafts"},
		{name: "generated drift", edit: func(row *imported.ImportedVersion) { drift := "SELECT 3"; row.SetGeneratedDql(&drift) }, want: "generated DQL must equal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx, err := f.db.BeginTx(f.ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			row := importedVersionRow(f.report.ID, 1, "SELECT 1", "hash-1", "", "owner", f.now)
			row.File = []*imported.ImportedResourceFile{importedResourceFileRow("ns", "main.dql", []byte("SELECT 1"), f.now)}
			row.Has.File = true
			test.edit(row)
			err = f.transport.writeImportedVersion(f.ctx, tx, row)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want %q", err, test.want)
			}
			var count int
			if err = tx.QueryRow(`SELECT COUNT(*) FROM report_versions WHERE report_id=?`, f.report.ID).Scan(&count); err != nil || count != 0 {
				t.Fatalf("denied import wrote versions=%d err=%v", count, err)
			}
		})
	}
	tx, err := f.db.BeginTx(f.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	stale := *f.report
	stale.ETag = f.report.ETag + 5
	err = f.transport.writeDraftPointer(f.ctx, tx, draftPointerRow(&stale, 1, f.now))
	var conflict *xhandler.Conflict
	if !errors.As(err, &conflict) {
		t.Fatalf("stale pointer error=%v", err)
	}
	if err = f.transport.writeDraftPointer(f.ctx, tx, draftPointerRow(&sdk.Report{ID: "missing", ETag: 1}, 1, f.now)); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("missing report error=%v", err)
	}
}

// The Studio API opens its database with SQLite shared-cache mode. The draft
// pointer writer reads the report through a pooled connection while the import
// transaction already holds version and file writes, so that mode must neither
// deadlock nor report a table lock.
func TestDQLImportCommitsAndRollsBackUnderSharedCacheDSN(t *testing.T) {
	f := newImportFixtureDSN(t, "?cache=shared")
	loaded, err := f.client.Versions().LoadArchive(f.ctx, f.report.ID, sdk.LoadArchiveInput{Archive: zipArchive(t, map[string][]byte{"main.dql": []byte("SELECT 1"), "sql/q.sql": []byte("SELECT 2")}), Format: "zip", EntryDQL: "main.dql"})
	if err != nil || loaded.Version.VersionNo != 1 || len(loaded.Files) != 2 {
		t.Fatalf("shared-cache import: %+v %v", loaded, err)
	}
	if _, err = f.db.Exec(`CREATE TRIGGER reject_import BEFORE INSERT ON report_resource_files BEGIN SELECT RAISE(ABORT,'test import failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err = f.client.Versions().LoadDQL(f.ctx, f.report.ID, sdk.LoadDQLInput{DQL: "SELECT 3"}); err == nil {
		t.Fatal("expected resource failure")
	}
	var versions int
	if err = f.db.QueryRow(`SELECT COUNT(*) FROM report_versions WHERE report_id=?`, f.report.ID).Scan(&versions); err != nil || versions != 1 {
		t.Fatalf("versions after rollback=%d err=%v", versions, err)
	}
	current, err := f.client.Reports().Get(f.ctx, f.report.ID)
	if err != nil || current.CurrentDraftVersion == nil || *current.CurrentDraftVersion != 1 || current.ETag != f.report.ETag+1 {
		t.Fatalf("report after rollback: %+v %v", current, err)
	}
}
