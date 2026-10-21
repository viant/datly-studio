package audit

import (
	"context"
	"database/sql"
	"errors"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/schema"
	_ "modernc.org/sqlite"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type syntheticProvider struct {
	subject string
	roles   []string
}

func (p *syntheticProvider) Resolve(context.Context) (authz.Facts, error) {
	return authz.Facts{Subject: p.subject, Roles: p.roles, Issuer: "e2e", Tenant: "e2e", ValidUntil: time.Now().Add(time.Minute)}, nil
}
func TestScopedAuditControls(t *testing.T) {
	db, e := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "audit.db")+"?_pragma=busy_timeout(5000)&_txlock=immediate")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = schema.ApplySQLite(context.Background(), db, "studio"); e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{
		`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,allowed_roles_json,created_at,updated_at) VALUES('ns','owner','alpha','Alpha','active','{"view":["E2E_VIEWER"],"manage":["E2E_MANAGER"]}',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,created_at,updated_at) VALUES('other','stranger','beta','Beta','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO components(id,namespace_id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,created_at,updated_at) VALUES('component','ns','alpha','reader','Reader','owner','active','db','example','Reader',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO component_acl(report_id,subject_type,subject_id,can_edit) VALUES('component','role','E2E_TOOL_MANAGER',TRUE)`,
		`INSERT INTO component_publications(report_id,active_version_no,desired_generation,active_generation,publication_status,spec_hash,published_by,published_at) VALUES('component',1,1,1,'active','hash','owner',CURRENT_TIMESTAMP)`,
		`INSERT INTO component_mcp_exposures(report_id,version_no,exposure_id,route_id,route_method,route_path,kind,name) VALUES('component',1,'exposure','route','GET','/read','tool','example.read')`,
	} {
		if _, e = db.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	p := &syntheticProvider{subject: "manager", roles: []string{"E2E_MANAGER"}}
	s := &Service{DB: db, Provider: p, ViewerRole: "E2E_AUDIT_VIEWER"}
	ctx := context.Background()
	scope := Scope{NamespaceID: "ns"}
	status, e := s.Configure(ctx, scope, false)
	if e != nil || status.Enabled || !status.CanManage || status.CanView {
		t.Fatalf("manager configure: %+v %v", status, e)
	}
	if _, e = s.List(ctx, Filter{NamespaceID: "ns"}); !errors.Is(e, ErrForbidden) {
		t.Fatal("manager must not read audit", e)
	}
	_, e = s.Configure(ctx, scope, true)
	if e != nil {
		t.Fatal(e)
	}
	p.subject = "viewer"
	p.roles = []string{"E2E_VIEWER", "E2E_AUDIT_VIEWER"}
	if _, e = s.Configure(ctx, scope, false); !errors.Is(e, ErrForbidden) {
		t.Fatal("audit viewer changed settings", e)
	}
	if _, e = s.List(ctx, Filter{NamespaceID: "ns"}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.List(ctx, Filter{NamespaceID: "other"}); !errors.Is(e, ErrForbidden) {
		t.Fatal("cross-namespace audit disclosure", e)
	}
	p.subject = "tool-manager"
	p.roles = []string{"E2E_TOOL_MANAGER"}
	if _, e = s.Configure(ctx, Scope{NamespaceID: "ns", ComponentID: "component"}, false); e != nil {
		t.Fatal("tool manager", e)
	}
	request := httptest.NewRequest("POST", "/mcp", nil)
	request.Header.Set("X-Studio-Namespace", "ns")
	identity := Identity{}
	if s.Policy(request, "example.read", &identity) || identity.ComponentID != "component" {
		t.Fatal("component opt out was ignored", identity)
	}
	if _, e = s.Configure(ctx, scope, false); !errors.Is(e, ErrForbidden) {
		t.Fatal("tool manager changed namespace", e)
	}
	_, e = s.Configure(ctx, Scope{NamespaceID: "ns", ComponentID: "component"}, true)
	if e != nil {
		t.Fatal(e)
	}
	toolScope := Scope{NamespaceID: "ns", ToolName: "example.read"}
	_, e = s.Configure(ctx, toolScope, false)
	if e != nil {
		t.Fatal(e)
	}
	if s.Policy(request, "example.read", &Identity{}) {
		t.Fatal("tool opt out was ignored")
	}
	// Reconstruct the controller to prove settings survive restart.
	s = &Service{DB: db, Provider: p, ViewerRole: "E2E_AUDIT_VIEWER"}
	status, e = s.Status(ctx, toolScope)
	if e != nil || status.Enabled {
		t.Fatal("settings not persisted", status, e)
	}
	p.roles = nil
	if _, e = s.Configure(ctx, toolScope, true); !errors.Is(e, ErrForbidden) {
		t.Fatal("revoked role changed settings", e)
	}
	p.subject = "owner"
	if _, e = s.Configure(ctx, Scope{NamespaceID: "other", ComponentID: "component"}, false); !errors.Is(e, ErrForbidden) {
		t.Fatal("cross-namespace component update", e)
	}
}
func TestCaptureLimitAndDynamicToggle(t *testing.T) {
	records := make(chan Record, 4)
	l := New(Config{Enabled: true, Sanitize: RedactJSON}, testSink{records})
	payload := `{"result":{"password":"private","content":[{"text":"{\"token\":\"private\"}"}],"value":"` + strings.Repeat("x", 40000) + `"}}`
	handler := l.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(payload)) }))
	invoke := func() {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("POST", "/mcp", strings.NewReader(`{"method":"tools/call","params":{"name":"test"}}`)))
		if response.Body.String() != payload {
			t.Fatal("capture changed actual response")
		}
	}
	invoke()
	l.SetEnabled(false)
	invoke()
	l.SetEnabled(true)
	invoke()
	if e := l.Close(context.Background()); e != nil {
		t.Fatal(e)
	}
	if len(records) != 2 {
		t.Fatal("toggle was ignored", len(records))
	}
	for len(records) > 0 {
		record := <-records
		body := decodeBody(record.ResponseBody, record.ResponseCompression, ResponseLimit+1)
		if record.ResponseCompression != "gzip" || len(body) != ResponseLimit || !record.ResponseTruncated || strings.Contains(body, "private") {
			t.Fatal("cap, compression or redaction failed")
		}
	}
}
func TestOversizedRequestStillAudited(t *testing.T) {
	records := make(chan Record, 1)
	l := New(Config{Enabled: true, MaxBodyBytes: 64, Sanitize: RedactJSON}, testSink{records})
	request := `{"method":"tools/call","params":{"name":"large","arguments":{"value":"` + strings.Repeat("x", 1000) + `"}}}`
	handler := l.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"result":{}}`)) }))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/mcp", strings.NewReader(request)))
	l.Close(context.Background())
	if len(records) != 1 {
		t.Fatal("oversized call was not audited")
	}
}

func TestAuditFiltersAndDateRange(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "audit.db")+"?_pragma=busy_timeout(5000)&_txlock=immediate")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	start := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for _, r := range []Record{
		{Identity: Identity{Subject: "alice"}, ToolName: "one", StartedAt: start, Status: "OK"},
		{Identity: Identity{Subject: "bob"}, ToolName: "one", StartedAt: start, Status: "OK"},
		{Identity: Identity{Subject: "alice"}, ToolName: "two", StartedAt: start, Status: "OK"},
		{Identity: Identity{Subject: "alice"}, ToolName: "one", StartedAt: start.Add(time.Hour), Status: "OK"},
	} {
		r.RequestCompression = "none"
		r.ResponseCompression = "none"
		if err = (&SQLSink{DB: db}).Write(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	s := &Service{DB: db, Provider: &syntheticProvider{subject: "auditor", roles: []string{"E2E_AUDIT_VIEWER"}}, ViewerRole: "E2E_AUDIT_VIEWER"}
	filter := Filter{Subject: "alice", ToolName: "one", Status: "OK", From: start.Format(time.RFC3339), To: start.Add(time.Minute).Format(time.RFC3339)}
	page, err := s.List(ctx, filter)
	if err != nil || len(page.Items) != 1 || page.Items[0].Subject != "alice" || page.Items[0].ToolName != "one" {
		t.Fatalf("filters: %+v %v", page, err)
	}
	filter.From = start.Add(time.Hour).Format(time.RFC3339)
	if _, err = s.List(ctx, filter); !errors.Is(err, ErrInvalid) {
		t.Fatal("reversed date range accepted", err)
	}
	var populated int
	if err = db.QueryRow("SELECT COUNT(*) FROM tool_call_audit WHERE response_original_bytes IS NOT NULL").Scan(&populated); err != nil || populated != 0 {
		t.Fatal("reserved column was written", populated, err)
	}
}

func TestAuditEnrichmentDoesNotBlockResponse(t *testing.T) {
	gate := make(chan struct{})
	entered := make(chan struct{})
	records := make(chan Record, 1)
	logger := New(Config{Enabled: true, Sanitize: RedactJSON, Identity: func(*http.Request) Identity { close(entered); <-gate; return Identity{Subject: "verified"} }}, testSink{records})
	defer logger.Close(context.Background())
	completed := make(chan struct{})
	handler := logger.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"result":{"ready":true}}`)) }))
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/mcp", strings.NewReader(`{"method":"tools/call","params":{"name":"test"}}`)))
		close(completed)
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		close(gate)
		t.Fatal("worker did not start")
	}
	select {
	case <-completed:
	case <-time.After(time.Second):
		close(gate)
		t.Fatal("response waited for audit identity lookup")
	}
	close(gate)
}
