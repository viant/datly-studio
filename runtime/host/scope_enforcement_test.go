package host

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk/access"
	accessstore "github.com/viant/datly-studio/store/sql/access"
	mcpschema "github.com/viant/mcp-protocol/schema"
	_ "modernc.org/sqlite"
)

const (
	scopeTestIssuer   = "https://access.example"
	scopeTestAudience = "runtime"
	scopeTestTenant   = "one"
)

// scopedTasksDQL declares a deployment-bound scope parameter. Only the trusted
// "scope" provider can populate ProjectIDs; no transport kind serves it.
const scopedTasksDQL = `#package('example.com/runtime/scoped')
#setting($_ = $connector('source'))
#setting($_ = $route('/tasks','GET'))
#setting($_ = $mcp('tasks.list','List tasks'))
#define($_ = $ProjectIDs<[]int>(scope/project).Required().WithPredicate(0,'in','t','project_id'))
#define($_ = $Tasks<[]*Task>(output/view))
SELECT tasks.*, type(tasks,'Task')
FROM (SELECT t.id, t.project_id, t.name FROM tasks t ${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")} ORDER BY t.id) tasks`

// unscopedRecordsDQL has no scope parameter; a bounded policy on it must deny
// because there is nothing typed to bind the authorized IDs into.
const unscopedRecordsDQL = `#package('example.com/runtime/records')
#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#define($_ = $Records<[]*Record>(output/view))
SELECT records.*, type(records,'Record')
FROM (SELECT t.id, t.name FROM tasks t ORDER BY t.id) records`

type scopedHost struct {
	service  *Service
	httpAddr string
	mcpAddr  string
	key      *rsa.PrivateKey
	store    *accessstore.Store
}

// newScopedHost publishes the supplied reports against a SQLite source with four
// tasks in four projects and starts real HTTP and MCP listeners. Policies are
// provisioned per report before Start.
func newScopedHost(t *testing.T, bindings []ScopeBinding, reports map[string]string, policies map[string]access.Policy) (*scopedHost, error) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/runtime\n\ngo 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sourceDSN := "file:" + filepath.Join(root, "source.db")
	source, err := sql.Open("sqlite", sourceDSN)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = source.Exec(`CREATE TABLE tasks(id INTEGER PRIMARY KEY, project_id INTEGER NOT NULL, name TEXT NOT NULL);
INSERT INTO tasks VALUES(1,101,'alpha-101'),(2,102,'beta-102'),(3,103,'gamma-103'),(4,104,'delta-104')`); err != nil {
		t.Fatal(err)
	}
	_ = source.Close()
	studioDSN := "file:" + filepath.Join(root, "studio.db")
	studio, err := sql.Open("sqlite", studioDSN)
	if err != nil {
		t.Fatal(err)
	}
	if err = schema.ApplySQLite(ctx, studio, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err = studio.Exec(`INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at) VALUES('source','sqlite',?,'owner','active','{}',1,?,?)`, sourceDSN, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err = studio.Exec(`INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at) VALUES(1,'gen:1','active',?,'{}','owner',?,?)`, len(reports), now, now); err != nil {
		t.Fatal(err)
	}
	for reportID, dql := range reports {
		scope := "example.com/runtime/" + reportID
		if _, err = studio.Exec(`INSERT INTO reports(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES(?,?,?,'owner','active','source',?,?,1,?,?)`, reportID, reportID, reportID, scope, reportID, now, now); err != nil {
			t.Fatal(err)
		}
		if _, err = studio.Exec(`INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES(?,1,'published','dql',?,?,'{}','studio.v1','hash','{}','valid','v1','studio.v1',1,'owner',?)`, reportID, dql, dql, now); err != nil {
			t.Fatal(err)
		}
		if _, err = studio.Exec(`INSERT INTO report_publications(report_id,active_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at) VALUES(?,1,1,1,'active','gen:1','hash','owner',?,?)`, reportID, now, now); err != nil {
			t.Fatal(err)
		}
	}
	_ = studio.Close()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(root, "access.pem")
	if err = os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded}), 0o600); err != nil {
		t.Fatal(err)
	}
	config := Config{
		HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"},
		Authentication: Authentication{DefaultMode: "public"},
		Studio:         Studio{Driver: "sqlite", DSN: studioDSN}, Admin: Admin{Token: "test-token"}, RootDir: root,
		Access: &ResourceAccessConfig{Tenant: scopeTestTenant, Issuer: scopeTestIssuer, Audience: scopeTestAudience, PublicKeyFile: keyFile, ScopeBindings: bindings},
	}
	service, err := New(ctx, config)
	if err != nil {
		return nil, err
	}
	store := service.resourceAccess.Store.(*accessstore.Store)
	for reportID, policy := range policies {
		if _, err = store.Provision(ctx, access.Document{Resource: access.Resource{Kind: "component", ID: reportID, Version: "1", Tenant: scopeTestTenant}, Policies: map[string]access.Policy{"execute": policy}}, "bootstrap"); err != nil {
			t.Fatal(err)
		}
	}
	if err = service.Start(ctx); err != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = service.Close(closeCtx)
		cancel()
		return nil, err
	}
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Close(closeCtx)
	})
	httpAddr, mcpAddr := service.Addresses()
	return &scopedHost{service: service, httpAddr: httpAddr, mcpAddr: mcpAddr, key: key, store: store}, nil
}

func (h *scopedHost) token(t *testing.T, subject string, entities []access.Entity) string {
	t.Helper()
	now := time.Now()
	claims := jwtlib.MapClaims{
		"iss": scopeTestIssuer, "aud": scopeTestAudience, "sub": subject,
		"exp": now.Add(5 * time.Minute).Unix(), "iat": now.Add(-5 * time.Second).Unix(),
		"tenant": scopeTestTenant, "roles": []string{"reader"}, "allowedEntities": entities,
	}
	signed, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims).SignedString(h.key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func (h *scopedHost) get(t *testing.T, path, token string, headers map[string]string) (int, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, "http://"+h.httpAddr+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

func (h *scopedHost) callTool(t *testing.T, token, name string, arguments map[string]any) (int, string) {
	t.Helper()
	if arguments == nil {
		arguments = map[string]any{}
	}
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": mcpschema.MethodToolsCall, "params": map[string]any{"name": name, "arguments": arguments, "_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": mcpschema.LatestProtocolVersion, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}}})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, "http://"+h.mcpAddr+"/mcp", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set(mcpschema.HeaderProtocolVersion, mcpschema.LatestProtocolVersion)
	request.Header.Set(mcpschema.HeaderMethod, mcpschema.MethodToolsCall)
	request.Header.Set("Mcp-Name", name)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

func (h *scopedHost) replacePolicy(t *testing.T, reportID string, policy access.Policy) {
	t.Helper()
	ctx := context.Background()
	resource := access.Resource{Kind: "component", ID: reportID, Version: "1", Tenant: scopeTestTenant}
	doc, err := h.store.Get(ctx, resource)
	if err != nil {
		t.Fatal(err)
	}
	doc.Policies["execute"] = policy
	if _, err = h.store.Replace(ctx, doc, doc.Revision, "test"); err != nil {
		t.Fatal(err)
	}
}

func assertRows(t *testing.T, body string, present, absent []string) {
	t.Helper()
	for _, name := range present {
		if !strings.Contains(body, name) {
			t.Fatalf("expected %q in %s", name, body)
		}
	}
	for _, name := range absent {
		if strings.Contains(body, name) {
			t.Fatalf("unauthorized %q leaked in %s", name, body)
		}
	}
}

func TestScopedComponentBindsAuthorizedEntitiesIntoCompiledQuery(t *testing.T) {
	reader := &access.Rule{Kind: "role", Value: "reader"}
	scoped := access.Policy{Mode: "protected", Rule: reader, EntityType: "project"}
	host, err := newScopedHost(t,
		[]ScopeBinding{{Component: "tasks", Version: "1", EntityType: "project", Parameter: "ProjectIDs"}},
		map[string]string{"tasks": scopedTasksDQL, "records": unscopedRecordsDQL},
		map[string]access.Policy{"tasks": scoped, "records": scoped},
	)
	if err != nil {
		t.Fatal(err)
	}
	alice := host.token(t, "alice", []access.Entity{{Type: "project", ID: "101"}, {Type: "project", ID: "102"}})
	bob := host.token(t, "bob", []access.Entity{{Type: "project", ID: "103"}})
	all := []string{"alpha-101", "beta-102", "gamma-103", "delta-104"}

	t.Run("HTTP returns only the caller's projects", func(t *testing.T) {
		status, body := host.get(t, "/tasks", alice, nil)
		if status != http.StatusOK {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, []string{"alpha-101", "beta-102"}, []string{"gamma-103", "delta-104"})
	})
	t.Run("distinct principals do not share scope", func(t *testing.T) {
		status, body := host.get(t, "/tasks", bob, nil)
		if status != http.StatusOK {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, []string{"gamma-103"}, []string{"alpha-101", "beta-102", "delta-104"})
	})
	t.Run("HTTP query and header inputs cannot widen scope", func(t *testing.T) {
		malicious := "/tasks?ProjectIDs=104&projectIds=104&project=104&scope=104&ProjectIDs=103&_criteria=1%3D1"
		status, body := host.get(t, malicious, alice, map[string]string{"ProjectIDs": "104", "project": "104", "scope": "104", "X-Scope": "104"})
		if status != http.StatusOK {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, []string{"alpha-101", "beta-102"}, []string{"gamma-103", "delta-104"})
	})
	t.Run("MCP tool call uses the same enforcement path", func(t *testing.T) {
		status, body := host.callTool(t, alice, "tasks.list", nil)
		if status != http.StatusOK || strings.Contains(body, `"isError":true`) {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, []string{"alpha-101", "beta-102"}, []string{"gamma-103", "delta-104"})
		status, body = host.callTool(t, bob, "tasks.list", nil)
		if status != http.StatusOK || strings.Contains(body, `"isError":true`) {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, []string{"gamma-103"}, []string{"alpha-101", "beta-102", "delta-104"})
	})
	t.Run("MCP arguments cannot widen scope", func(t *testing.T) {
		_, body := host.callTool(t, alice, "tasks.list", map[string]any{"ProjectIDs": []int{104, 103}, "projectIds": []int{104}, "project": 104, "scope": map[string]any{"project": []int{104}}})
		assertRows(t, body, nil, []string{"gamma-103", "delta-104"})
	})
	t.Run("missing credential denies before execution", func(t *testing.T) {
		status, body := host.get(t, "/tasks", "", nil)
		if status != http.StatusForbidden {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, nil, all)
		status, body = host.callTool(t, "", "tasks.list", nil)
		if status == http.StatusOK && !strings.Contains(body, "denied") && !strings.Contains(body, "error") {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, nil, all)
	})
	t.Run("bounded decision without a scope binding denies", func(t *testing.T) {
		status, body := host.get(t, "/records", alice, nil)
		if status != http.StatusForbidden || !strings.Contains(body, "typed runtime scope binding is required") {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, nil, all)
	})
	t.Run("entity dimension other than the bound one denies", func(t *testing.T) {
		host.replacePolicy(t, "tasks", access.Policy{Mode: "protected", Rule: reader, EntityType: "account"})
		accountHolder := host.token(t, "carol", []access.Entity{{Type: "account", ID: "101"}})
		status, body := host.get(t, "/tasks", accountHolder, nil)
		if status != http.StatusForbidden {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, nil, all)
		_, body = host.callTool(t, accountHolder, "tasks.list", nil)
		assertRows(t, body, nil, all)
		host.replacePolicy(t, "tasks", scoped)
	})
	t.Run("non canonical or non numeric IDs deny", func(t *testing.T) {
		for _, id := range []string{"abc", "007", "+101", " 101", "101.0", "1 OR 1=1"} {
			status, body := host.get(t, "/tasks", host.token(t, "mallory", []access.Entity{{Type: "project", ID: id}}), nil)
			if status != http.StatusForbidden {
				t.Fatalf("id %q status=%d body=%s", id, status, body)
			}
			assertRows(t, body, nil, all)
		}
	})
	t.Run("unbounded decision on a scoped component denies", func(t *testing.T) {
		host.replacePolicy(t, "tasks", access.Policy{Mode: "protected", Rule: reader})
		status, body := host.get(t, "/tasks", alice, nil)
		if status != http.StatusForbidden || !strings.Contains(body, "entity-bounded") {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, nil, all)
		_, body = host.callTool(t, alice, "tasks.list", nil)
		assertRows(t, body, nil, all)
		host.replacePolicy(t, "tasks", access.Policy{Mode: "public"})
		status, body = host.get(t, "/tasks", "", nil)
		if status != http.StatusForbidden {
			t.Fatalf("public policy ran a scoped component: status=%d body=%s", status, body)
		}
		assertRows(t, body, nil, all)
		host.replacePolicy(t, "tasks", scoped)
	})
	t.Run("scope is re-evaluated per request", func(t *testing.T) {
		status, body := host.get(t, "/tasks", alice, nil)
		if status != http.StatusOK {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, []string{"alpha-101", "beta-102"}, []string{"gamma-103", "delta-104"})
		widened := host.token(t, "alice", []access.Entity{{Type: "project", ID: "104"}})
		status, body = host.get(t, "/tasks", widened, nil)
		if status != http.StatusOK {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, []string{"delta-104"}, []string{"alpha-101", "beta-102", "gamma-103"})
	})
}

func TestScopedGenerationRejectsUndeclaredOrMismatchedBindings(t *testing.T) {
	reader := &access.Rule{Kind: "role", Value: "reader"}
	scoped := access.Policy{Mode: "protected", Rule: reader, EntityType: "project"}
	for name, tc := range map[string]struct {
		bindings []ScopeBinding
		reports  map[string]string
		want     string
	}{
		"scope parameter without deployment binding": {reports: map[string]string{"tasks": scopedTasksDQL}, want: "without a deployment scope binding"},
		"binding for a component without scope parameter": {
			bindings: []ScopeBinding{{Component: "records", Version: "1", EntityType: "project", Parameter: "Records"}},
			reports:  map[string]string{"records": unscopedRecordsDQL}, want: "has no scope parameter",
		},
		"binding names the wrong parameter": {
			bindings: []ScopeBinding{{Component: "tasks", Version: "1", EntityType: "project", Parameter: "Tasks"}},
			reports:  map[string]string{"tasks": scopedTasksDQL}, want: "names parameter",
		},
		"binding names the wrong entity type": {
			bindings: []ScopeBinding{{Component: "tasks", Version: "1", EntityType: "account", Parameter: "ProjectIDs"}},
			reports:  map[string]string{"tasks": scopedTasksDQL}, want: "does not match parameter scope/project",
		},
		"binding for a stale version": {
			bindings: []ScopeBinding{{Component: "tasks", Version: "2", EntityType: "project", Parameter: "ProjectIDs"}},
			reports:  map[string]string{"tasks": scopedTasksDQL}, want: "without a deployment scope binding",
		},
		"optional scope parameter": {
			bindings: []ScopeBinding{{Component: "tasks", Version: "1", EntityType: "project", Parameter: "ProjectIDs"}},
			reports:  map[string]string{"tasks": strings.Replace(scopedTasksDQL, ".Required()", ".Optional()", 1)}, want: "must be declared Required()",
		},
		"non integer scope type": {
			bindings: []ScopeBinding{{Component: "tasks", Version: "1", EntityType: "project", Parameter: "ProjectIDs"}},
			reports:  map[string]string{"tasks": strings.Replace(scopedTasksDQL, "<[]int>", "<[]float64>", 1)}, want: "must be a slice of string or integer",
		},
	} {
		t.Run(name, func(t *testing.T) {
			policies := map[string]access.Policy{}
			for reportID := range tc.reports {
				policies[reportID] = scoped
			}
			host, err := newScopedHost(t, tc.bindings, tc.reports, policies)
			if err == nil {
				status, body := host.get(t, "/tasks", "", nil)
				t.Fatalf("generation started with an undefined scope contract (GET /tasks -> %d %s)", status, body)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestScopedComponentIsRejectedWithoutGenericAccess(t *testing.T) {
	// Legacy authentication has no entity-bounded decisions, so a component that
	// declares a scope parameter cannot be served at all.
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/runtime\n\ngo 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sourceDSN := "file:" + filepath.Join(root, "source.db")
	source, err := sql.Open("sqlite", sourceDSN)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = source.Exec(`CREATE TABLE tasks(id INTEGER PRIMARY KEY, project_id INTEGER NOT NULL, name TEXT NOT NULL); INSERT INTO tasks VALUES(1,101,'alpha-101')`); err != nil {
		t.Fatal(err)
	}
	_ = source.Close()
	studioDSN := "file:" + filepath.Join(root, "studio.db")
	studio, err := sql.Open("sqlite", studioDSN)
	if err != nil {
		t.Fatal(err)
	}
	if err = schema.ApplySQLite(ctx, studio, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at) VALUES('source','sqlite',?,'owner','active','{}',1,?,?)`, []any{sourceDSN, now, now}},
		{`INSERT INTO reports(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('tasks','tasks','Tasks','owner','active','source','example.com/runtime/scoped','tasks',1,?,?)`, []any{now, now}},
		{`INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('tasks',1,'published','dql',?,?,'{}','studio.v1','hash','{}','valid','v1','studio.v1',1,'owner',?)`, []any{scopedTasksDQL, scopedTasksDQL, now}},
		{`INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at) VALUES(1,'gen:1','active',1,'{}','owner',?,?)`, []any{now, now}},
		{`INSERT INTO report_publications(report_id,active_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at) VALUES('tasks',1,1,1,'active','gen:1','hash','owner',?,?)`, []any{now, now}},
	}
	for _, statement := range statements {
		if _, err = studio.Exec(statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	_ = studio.Close()
	service, err := New(ctx, Config{HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"}, Authentication: Authentication{DefaultMode: "public"}, Studio: Studio{Driver: "sqlite", DSN: studioDSN}, Admin: Admin{Token: "test-token"}, RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Close(closeCtx)
	})
	err = service.Start(ctx)
	if err == nil || !strings.Contains(err.Error(), "requires generic resource access") {
		t.Fatalf("legacy runtime served a scoped component: %v", err)
	}
}
