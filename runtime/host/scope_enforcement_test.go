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

// scopedTasksDQL binds the server-owned access-context component natively and
// derives the SQL-bound project IDs from its typed output. Component-kind and
// param-kind inputs are runtime-protected: no query, body, header or MCP
// argument can supply or override them.
const scopedTasksDQL = `#package('example.com/runtime/scoped')
#import('studioaccess','github.com/viant/datly-studio/runtime/accesscontext')
#setting($_ = $connector('source'))
#setting($_ = $route('/tasks','GET'))
#setting($_ = $mcp('tasks.list','List tasks'))
#define($_ = $Auth<*studioaccess.Output>(component/GET:/_studio/access/context/tasks/project).Required())
#define($_ = $ProjectIDs<[]string,[]int>(param/Auth.Scope.IDs).WithCodec('EntityIDs').Required().WithPredicate(0,'in','t','project_id'))
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
// provisioned per report before Start. No scope configuration exists: a
// component's DQL alone declares whether it binds its access context.
// narrowingDecisions is an injected trusted decision provider that keeps only
// the listed IDs of the local decision, like a remote policy service would.
type narrowingDecisions struct {
	keep  map[string]bool
	calls int
}

func (d *narrowingDecisions) Evaluate(_ context.Context, _ access.Request, _ access.Document, facts access.Facts) (access.Decision, error) {
	d.calls++
	flat, err := facts.FlatEntities()
	if err != nil {
		return access.Decision{}, err
	}
	decision := access.Decision{Bounded: true}
	for _, entity := range flat {
		if d.keep[entity.ID] {
			decision.Entities = append(decision.Entities, entity)
		}
	}
	if len(decision.Entities) == 0 {
		return access.Decision{}, access.ErrDenied
	}
	return decision, nil
}

func newScopedHost(t *testing.T, reports map[string]string, policies map[string]access.Policy, decisions ...access.DecisionProvider) (*scopedHost, error) {
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
		Access: &ResourceAccessConfig{Tenant: scopeTestTenant, Issuer: scopeTestIssuer, Audience: scopeTestAudience, PublicKeyFile: keyFile},
	}
	if len(decisions) > 0 {
		config.DecisionProvider = decisions[0]
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

func (h *scopedHost) expiredToken(t *testing.T, subject string, entities []access.Entity) string {
	t.Helper()
	now := time.Now()
	claims := jwtlib.MapClaims{
		"iss": scopeTestIssuer, "aud": scopeTestAudience, "sub": subject,
		"exp": now.Add(-time.Minute).Unix(), "iat": now.Add(-2 * time.Minute).Unix(),
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
		malicious := "/tasks?ProjectIDs=104&projectIds=104&Auth.Scope.IDs=104&Auth=%7B%22scope%22%3A%7B%22ids%22%3A%5B%22104%22%5D%7D%7D&scope=104&ProjectIDs=103&_criteria=1%3D1"
		status, body := host.get(t, malicious, alice, map[string]string{"ProjectIDs": "104", "Auth": `{"scope":{"ids":["104"]}}`, "X-Scope": "104"})
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
		_, body := host.callTool(t, alice, "tasks.list", map[string]any{"ProjectIDs": []string{"104", "103"}, "projectIds": []int{104}, "Auth": map[string]any{"scope": map[string]any{"ids": []string{"104"}}, "context": map[string]any{"roles": []string{"admin"}, "allowedEntities": map[string]any{"project": []string{"104"}}}}, "scope": map[string]any{"project": []int{104}}})
		assertRows(t, body, nil, []string{"gamma-103", "delta-104"})
	})
	t.Run("expired credential denies", func(t *testing.T) {
		expired := host.expiredToken(t, "alice", []access.Entity{{Type: "project", ID: "101"}})
		status, body := host.get(t, "/tasks", expired, nil)
		if status != http.StatusForbidden {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, nil, all)
		_, body = host.callTool(t, expired, "tasks.list", nil)
		assertRows(t, body, nil, all)
	})
	t.Run("principal without allowed IDs denies", func(t *testing.T) {
		none := host.token(t, "dave", nil)
		status, body := host.get(t, "/tasks", none, nil)
		if status != http.StatusForbidden {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, nil, all)
		empty := host.token(t, "erin", []access.Entity{})
		status, body = host.get(t, "/tasks", empty, nil)
		if status != http.StatusForbidden {
			t.Fatalf("status=%d body=%s", status, body)
		}
		assertRows(t, body, nil, all)
	})
	t.Run("access context route is not directly executable", func(t *testing.T) {
		status, body := host.get(t, "/_studio/access/context/tasks/project", alice, nil)
		if status != http.StatusForbidden || strings.Contains(body, "101") {
			t.Fatalf("status=%d body=%s", status, body)
		}
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
	t.Run("bounded decision on a component that does not bind its context denies", func(t *testing.T) {
		status, body := host.get(t, "/records", alice, nil)
		if status != http.StatusForbidden || !strings.Contains(body, "bind its access context") {
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
		if status != http.StatusForbidden {
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

// TestScopedComponentHonorsTrustedRemoteNarrowing proves the bound access
// context carries the local/remote intersection, not the raw facts: alice's
// facts grant projects 101 and 102, the injected trusted decision keeps only
// 102, and both the SQL scope and the canonical context see 102 alone.
func TestScopedComponentHonorsTrustedRemoteNarrowing(t *testing.T) {
	reader := &access.Rule{Kind: "role", Value: "reader"}
	scoped := access.Policy{Mode: "protected", Rule: reader, EntityType: "project"}
	decisions := &narrowingDecisions{keep: map[string]bool{"102": true, "103": true}}
	host, err := newScopedHost(t, map[string]string{"tasks": scopedTasksDQL}, map[string]access.Policy{"tasks": scoped}, decisions)
	if err != nil {
		t.Fatal(err)
	}
	alice := host.token(t, "alice", []access.Entity{{Type: "project", ID: "101"}, {Type: "project", ID: "102"}})
	status, body := host.get(t, "/tasks", alice, nil)
	if status != http.StatusOK {
		t.Fatalf("status=%d body=%s", status, body)
	}
	assertRows(t, body, []string{"beta-102"}, []string{"alpha-101", "gamma-103", "delta-104"})
	status, body = host.get(t, "/tasks?ProjectIDs=101&Auth.Scope.IDs=101", alice, nil)
	if status != http.StatusOK {
		t.Fatalf("status=%d body=%s", status, body)
	}
	assertRows(t, body, []string{"beta-102"}, []string{"alpha-101", "gamma-103", "delta-104"})
	_, body = host.callTool(t, alice, "tasks.list", nil)
	assertRows(t, body, []string{"beta-102"}, []string{"alpha-101", "gamma-103", "delta-104"})
	// Forged arguments are either rejected as unknown or ignored; never applied.
	_, body = host.callTool(t, alice, "tasks.list", map[string]any{"ProjectIDs": []string{"101"}, "Auth": map[string]any{"scope": map[string]any{"ids": []string{"101"}}}})
	assertRows(t, body, nil, []string{"alpha-101", "gamma-103", "delta-104"})
	// The execute hook and the bound context component each evaluate the
	// intersected decision once per request; the remote provider is never
	// bypassed.
	if decisions.calls < 3 {
		t.Fatalf("remote decision provider consulted %d times", decisions.calls)
	}
	// A remote decision outside the local facts cannot widen either.
	decisions.keep = map[string]bool{"104": true}
	status, body = host.get(t, "/tasks", alice, nil)
	if status != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", status, body)
	}
	assertRows(t, body, nil, []string{"alpha-101", "beta-102", "gamma-103", "delta-104"})
}

func TestScopedGenerationRejectsForeignAccessContext(t *testing.T) {
	// A component may bind only its own access context; naming another
	// component's context would read that resource's decision.
	reader := &access.Rule{Kind: "role", Value: "reader"}
	scoped := access.Policy{Mode: "protected", Rule: reader, EntityType: "project"}
	foreign := strings.Replace(scopedTasksDQL, "/_studio/access/context/tasks/project", "/_studio/access/context/records/project", 1)
	host, err := newScopedHost(t, map[string]string{"tasks": foreign, "records": unscopedRecordsDQL}, map[string]access.Policy{"tasks": scoped, "records": scoped})
	if err == nil {
		status, body := host.get(t, "/tasks", "", nil)
		t.Fatalf("generation started binding a foreign access context (GET /tasks -> %d %s)", status, body)
	}
	if !strings.Contains(err.Error(), "may bind only its own") {
		t.Fatalf("want foreign context rejection, got %v", err)
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
		t.Fatalf("legacy runtime served a component binding an access context: %v", err)
	}
}
