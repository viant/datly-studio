package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	dexec "github.com/viant/datly/exec"
	_ "modernc.org/sqlite"
)

type cacheInvalidatorStub struct {
	calls              int
	report, path, view string
	version            int
}

func (s *cacheInvalidatorStub) InvalidatePublishedCacheView(_ context.Context, report string, version int, path, view string) ([]dexec.CacheInvalidation, error) {
	s.calls++
	s.report, s.version, s.path, s.view = report, version, path, view
	return []dexec.CacheInvalidation{{View: view, Scope: "all", Generation: "view-generation-2"}}, nil
}

func TestCacheInvalidateUsesExistingPublishAuthorityAndExactVersion(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	_, err = db.Exec(`
INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,?,?);
INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,?,?);
INSERT INTO components(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('reader','general','reader','Reader','owner','active','main','example.com/reader','reader',1,?,?);
INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('reader',1,'validated','dql','SELECT 1','{}','1','hash','{}','valid','v1','v1',3,'owner',?);`, now, now, now, now, now, now, now)
	if err != nil {
		t.Fatal(err)
	}
	stub := &cacheInvalidatorStub{}
	allowed := false
	transport := &Transport{DB: db, CacheInvalidator: stub, Authorizer: AuthorizerFunc(func(_ context.Context, request AuthorizationRequest) error {
		if request.Permission != "publish" || request.ReportID != "reader" {
			t.Fatalf("wrong authorization boundary: %+v", request)
		}
		if !allowed {
			return errors.New("publisher denied")
		}
		return nil
	})}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Versions().InvalidateCacheView(ctx, "reader", 1, "/records", "records"); err == nil || stub.calls != 0 {
		t.Fatalf("denied publisher reached cache invalidator: err=%v calls=%d", err, stub.calls)
	}
	allowed = true
	if _, err = client.Versions().InvalidateCacheView(ctx, "reader", 2, "/records", "records"); err == nil || stub.calls != 0 {
		t.Fatalf("nonexistent version reached cache invalidator: err=%v calls=%d", err, stub.calls)
	}
	result, err := client.Versions().InvalidateCacheView(ctx, "reader", 1, "/records", "records")
	if err != nil || result == nil || result.Generation != "view-generation-2" || result.View != "records" || stub.calls != 1 || stub.report != "reader" || stub.version != 1 || stub.path != "/records" || stub.view != "records" {
		t.Fatalf("exact control request result=%+v err=%v stub=%+v", result, err, stub)
	}
}
