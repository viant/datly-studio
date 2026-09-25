package host

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/scy/auth/jwt"
	_ "modernc.org/sqlite"
)

func TestAuthorizeRunRequiresOwnerOrCanRun(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('reader','general','reader','Reader','owner','active','main','example.com/reader','reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_run) VALUES('reader','user','runner',TRUE,TRUE);
INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_run) VALUES('reader','user','viewer',TRUE,FALSE);`)
	if err != nil {
		t.Fatal(err)
	}
	store, err := newRunAccessStore(db)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(context.Background())
	service := &Service{studio: db, runAccessStore: store, config: Config{Authentication: Authentication{DefaultMode: "required"}}}
	for _, subject := range []string{"owner", "runner"} {
		claims := &jwt.Claims{}
		claims.Subject = subject
		ctx := context.WithValue(context.Background(), verifiedClaimsKey{}, claims)
		if err = service.authorizeRun(ctx, "reader"); err != nil {
			t.Fatalf("subject %s: %v", subject, err)
		}
	}
	claims := &jwt.Claims{}
	claims.Subject = "viewer"
	ctx := context.WithValue(context.Background(), verifiedClaimsKey{}, claims)
	if err = service.authorizeRun(ctx, "reader"); err == nil {
		t.Fatal("view-only subject was allowed to execute the reader")
	}
	for _, subject := range []string{"runner", "viewer"} {
		if _, err = db.Exec(`UPDATE report_acl SET subject_type='role' WHERE report_id='reader' AND subject_id=?`, subject); err != nil {
			t.Fatal(err)
		}
	}
	claims.Subject = "runner"
	if err = service.authorizeRun(ctx, "reader"); err == nil {
		t.Fatal("role ACL row was treated as a direct user grant")
	}
	claims.Subject = "owner"
	if _, err = db.Exec(`UPDATE reports SET deleted_at=CURRENT_TIMESTAMP WHERE id='reader'`); err != nil {
		t.Fatal(err)
	}
	if err = service.authorizeRun(ctx, "reader"); err == nil {
		t.Fatal("deleted report remained executable by its owner")
	}
	if err = service.authorizeRun(context.Background(), "reader"); err == nil {
		t.Fatal("missing verified claims were allowed")
	}
}

func TestReportForResourceURIUsesLongestOwnedPrefix(t *testing.T) {
	prefixes := map[string]string{
		"docs://team/":        "team",
		"docs://team/orders/": "orders",
	}
	if actual := reportForResourceURI(prefixes, "docs://team/orders/guide.md"); actual != "orders" {
		t.Fatalf("owner=%q", actual)
	}
	if actual := reportForResourceURI(prefixes, "docs://public/guide.md"); actual != "" {
		t.Fatalf("unexpected owner=%q", actual)
	}
}

func TestDefinitionsSeparateActiveAndCandidateVersions(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := "2026-09-20 12:00:00"
	_, err = db.Exec(`
INSERT INTO connectors(name,driver,dsn_template,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','file:test.db','owner','active',1,?,?);
INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,?,?);
INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('reader','general','reader','Reader','owner','active','main','example.com/reader','reader',1,?,?);
INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('reader',1,'published','dql','SELECT 1','{}','1','one','{}','valid','v1','v1',1,'owner',?);
INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('reader',2,'validated','dql','SELECT 2','{}','1','two','{}','valid','v1','v1',2,'owner',?);
INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at) VALUES(1,'one','active',1,'{}','owner',?,?);
INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at) VALUES(2,'two','building',0,'{}','owner',?);
INSERT INTO report_publications(report_id,active_version_no,desired_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at) VALUES('reader',1,2,2,1,'pending','two','two','owner',?,?);`,
		now, now, now, now, now, now, now, now, now, now, now, now)
	if err != nil {
		t.Fatal(err)
	}
	store, err := newPublishedDefinitionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(ctx)
	service := &Service{studio: db, definitionStore: store}
	assertVersion := func(candidate *int64, want int) {
		t.Helper()
		items, queryErr := service.definitions(ctx, candidate)
		if queryErr != nil || len(items) != 1 || items[0].versionNo != want {
			t.Fatalf("candidate=%v definitions=%+v err=%v want version %d", candidate, items, queryErr, want)
		}
	}
	assertVersion(nil, 1)
	candidate := int64(2)
	assertVersion(&candidate, 2)
	previous := int64(1)
	assertVersion(&previous, 1)
	if _, err = db.Exec(`UPDATE report_publications SET desired_version_no=NULL,publication_status='unpublishing' WHERE report_id='reader'`); err != nil {
		t.Fatal(err)
	}
	items, err := service.definitions(ctx, &candidate)
	if err != nil || len(items) != 0 {
		t.Fatalf("unpublish candidate definitions=%+v err=%v", items, err)
	}
	assertVersion(nil, 1)
}
