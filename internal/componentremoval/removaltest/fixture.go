package removaltest

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/internal/datatest"
	_ "github.com/viant/datly-studio/internal/dependencylink"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/datly/standalone"
	"github.com/viant/datly/standalone/config"
)

// Fixture runs the linked native host against an isolated schema-28 database.
type Fixture struct {
	T                         testing.TB
	DB                        *sql.DB
	Server                    *standalone.Server
	Namespace, OtherNamespace string
	Addresses                 []string
	Token                     func(string, ...string) string
	Context                   func(context.Context, string, ...string) context.Context
}

const Now = "2026-11-14 00:00:00"

func New(ctx context.Context, t testing.TB) *Fixture {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "studio.db")
	dsn := "file:" + dbPath + "?cache=shared&_pragma=foreign_keys(1)"
	db := datatest.OpenSQLite(t, "removal", "studio")
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", dbPath); err != nil {
		t.Fatal(err)
	}
	connections, err := connector.Open(ctx, []connector.Config{{Name: "studio", Driver: "sqlite", DSN: dsn}}, "studio")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connections.Close() })
	db = connections.SQL.DB
	f := &Fixture{T: t, DB: db, Namespace: namespaceaccess.ID("alice", "alpha"), OtherNamespace: namespaceaccess.ID("alice", "beta")}
	f.Hydrate(ctx,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": Now, "updated_at": Now}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{
			{"namespace_id": f.Namespace, "owner_id": "alice", "name": "alpha", "title": "Alpha", "visibility": "public", "status": "active", "created_at": Now, "updated_at": Now},
			{"namespace_id": f.OtherNamespace, "owner_id": "alice", "name": "beta", "title": "Beta", "status": "active", "created_at": Now, "updated_at": Now},
		}})
	return startHost(ctx, t, f, "sqlite", dsn)
}

// NewMySQL requires an empty database in an exclusively owned disposable
// instance configured with ANSI_QUOTES. It does not alter an existing store.
func NewMySQL(ctx context.Context, t testing.TB, dsn string) *Fixture {
	t.Helper()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(8)
	var tables int
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Fatal("MySQL removal fixture requires an empty owned database")
	}
	if err = schema.Initialize(ctx, db, "mysql"); err != nil {
		t.Fatal(err)
	}
	f := &Fixture{T: t, DB: db, Namespace: namespaceaccess.ID("alice", "alpha"), OtherNamespace: namespaceaccess.ID("alice", "beta")}
	f.Exec(ctx, `INSERT INTO connectors(name,driver,owner_id,status,created_at,updated_at) VALUES('main','mysql','alice','active',?,?)`, Now, Now)
	for _, name := range []string{"alpha", "beta"} {
		f.Exec(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,visibility,status,created_at,updated_at) VALUES(?,'alice',?,?,'public','active',?,?)`, namespaceaccess.ID("alice", name), name, name, Now, Now)
	}
	return startHost(ctx, t, f, "mysql", dsn)
}

func startHost(ctx context.Context, t testing.TB, f *Fixture, driver, dsn string) *Fixture {
	t.Helper()
	jwt := datatest.NewJWTFixture(t)
	keyPath := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(keyPath, jwt.PublicKeyPEM(t), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUDIO_ACCESS_ISSUER", "removal-test")
	t.Setenv("STUDIO_ACCESS_AUDIENCE", "studio")
	t.Setenv("STUDIO_ACCESS_PUBLIC_KEY_FILE", keyPath)
	t.Setenv("STUDIO_ACCESS_USER_INFO_URL", "")
	block, _ := pem.Decode(jwt.PublicKeyPEM(t))
	public, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	key := public.(*rsa.PublicKey)
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "fixture", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
	}))
	t.Cleanup(jwks.Close)
	_, source, _, _ := runtime.Caller(0)
	root, err := filepath.Abs(filepath.Join(filepath.Dir(source), "../../.."))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := (config.Loader{}).Load(ctx, filepath.Join(root, "datly.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.BaseDir = root
	cfg.Endpoint.Address = "127.0.0.1:0"
	cfg.MCP.Address = "127.0.0.1:0"
	cfg.Connectors = []connector.Config{{Name: "studio", Driver: driver, DSN: dsn}, {Name: "authz", AliasOf: "studio"}}
	cfg.JWTValidator.CertURL = jwks.URL
	cfg.JWTClaims.Issuer = "removal-test"
	cfg.JWTClaims.Audience = "studio"
	server, err := standalone.New(ctx, standalone.Options{Config: cfg, RequireLinked: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Shutdown(ctx) })
	serveCtx, stop := context.WithCancel(ctx)
	t.Cleanup(stop)
	served := make(chan error, 1)
	go func() { served <- server.Serve(serveCtx, io.Discard) }()
	readyCtx, cancelReady := context.WithTimeout(ctx, 5*time.Second)
	addresses, err := server.WaitReady(readyCtx)
	cancelReady()
	if err != nil {
		t.Fatal(err)
	}

	f.Server, f.Addresses = server, addresses
	f.Token = func(subject string, roles ...string) string {
		return jwt.BearerWithClaimsAndKeyID(t, jwtv5.MapClaims{"sub": subject, "roles": roles, "iss": "removal-test", "aud": "studio", "tenant": "tenant", "exp": time.Now().Add(time.Hour).Unix()}, "fixture")
	}
	f.Context = func(ctx context.Context, subject string, roles ...string) context.Context {
		bearer := f.Token(subject, roles...)
		parsed, err := jwtv5.Parse(strings.TrimPrefix(bearer, "Bearer "), func(*jwtv5.Token) (any, error) { return key, nil })
		if err != nil || !parsed.Valid {
			t.Fatalf("verify SDK credential: %v", err)
		}
		return sdk.WithNamespaceSelection(sdk.WithVerifiedCredential(sdk.WithPrincipal(ctx, sdk.Principal{Subject: subject}), sdk.VerifiedCredential{Bearer: bearer, Claims: parsed.Claims}), f.Namespace)
	}
	return f
}

func (f *Fixture) Hydrate(ctx context.Context, tables ...datatest.Table) {
	f.T.Helper()
	if err := datatest.Hydrate(ctx, f.DB, tables...); err != nil {
		f.T.Fatal(err)
	}
}
func (f *Fixture) Exec(ctx context.Context, query string, args ...any) {
	f.T.Helper()
	if _, err := f.DB.ExecContext(ctx, query, args...); err != nil {
		f.T.Fatal(err)
	}
}
func (f *Fixture) Count(ctx context.Context, query string, want int, args ...any) {
	f.T.Helper()
	var got int
	if err := f.DB.QueryRowContext(ctx, query, args...).Scan(&got); err != nil || got != want {
		f.T.Fatalf("%s: got=%d want=%d err=%v", query, got, want, err)
	}
}
func (f *Fixture) Seed(ctx context.Context, id string, versions int) {
	f.T.Helper()
	var draft any
	if versions > 0 {
		draft = versions
	}
	f.Hydrate(ctx, datatest.Table{Name: "components", Rows: []datatest.Row{{"id": id, "namespace_id": f.Namespace, "namespace": "alpha", "slug": id, "title": id, "owner_id": "alice", "status": "active", "default_connector_name": "main", "component_scope": "reader/" + id, "component_name": id, "current_draft_version": draft, "last_version_no": versions, "created_at": Now, "updated_at": Now}}})
	for v := 1; v <= versions; v++ {
		f.Hydrate(ctx, datatest.Table{Name: "component_versions", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": id, "version_no": v, "state": "draft", "authoring_mode": "dql", "authored_dql": "SELECT 1 AS id", "generated_dql": "SELECT 1 AS id", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": fmt.Sprintf("%064d", v), "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 7, "created_by": "alice", "created_at": Now}}})
	}
}
func Component(id string, etag int64) map[string]any {
	return map[string]any{"id": id, "input": map[string]any{"etag": etag}}
}
func Version(id string, version int, revision int64) map[string]any {
	return map[string]any{"reportId": id, "versionNo": version, "expectedSourceRevision": revision}
}
func (f *Fixture) HTTP(ctx context.Context, operation, subject, namespace string, body any, want int, roles ...string) []byte {
	f.T.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		f.T.Fatal(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+f.Addresses[0]+"/v1/studio/sdk/"+operation, bytes.NewReader(raw))
	if err != nil {
		f.T.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", f.Token(subject, roles...))
	req.Header.Set("X-Studio-Namespace", namespace)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		f.T.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		f.T.Fatal(err)
	}
	if response.StatusCode != want {
		f.T.Fatalf("%s status=%d want=%d: %s", operation, response.StatusCode, want, data)
	}
	return data
}
func (f *Fixture) Result(raw []byte, id string, version, deleted int) {
	f.T.Helper()
	var result sdk.DeletionResult
	if err := json.Unmarshal(raw, &result); err != nil || result.ComponentID != id || result.VersionNo != version || result.DeletedVersions != deleted {
		f.T.Fatalf("deletion receipt=%s err=%v", raw, err)
	}
}
func (f *Fixture) MCP(ctx context.Context, operation string, arguments map[string]any) []byte {
	f.T.Helper()
	arguments["namespaceId"] = f.Namespace
	raw, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "studio.sdk." + operation, "arguments": arguments}})
	if err != nil {
		f.T.Fatal(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+f.Addresses[1]+"/mcp", bytes.NewReader(raw))
	if err != nil {
		f.T.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", f.Token("alice"))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		f.T.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		f.T.Fatal(err)
	}
	var rpc struct {
		Error  json.RawMessage `json:"error"`
		Result struct {
			IsError bool            `json:"isError"`
			Data    json.RawMessage `json:"structuredContent"`
		} `json:"result"`
	}
	if err = json.Unmarshal(data, &rpc); err != nil || response.StatusCode != 200 || len(rpc.Error) > 0 || rpc.Result.IsError || len(rpc.Result.Data) == 0 {
		f.T.Fatalf("MCP %s: %s err=%v", operation, data, err)
	}
	return rpc.Result.Data
}

// Warmup inserts owned evidence; production deletion must remove terminal work
// in the graph before its version, and reject accepted/running work.
func (f *Fixture) Warmup(ctx context.Context, id string, version int, status string) {
	f.Hydrate(ctx, datatest.Table{Name: "component_warmup_runs", Rows: []datatest.Row{{"namespace_id": f.Namespace, "run_id": fmt.Sprintf("%s-%d-%s", id, version, status), "report_id": id, "version_no": version, "source_revision": 7, "spec_hash": fmt.Sprintf("%064d", version), "plan_key": strings.Repeat("c", 64), "status": status, "requested_by": "alice", "target_json": "{}", "requested_at": Now}}})
}

// Graph seeds real FK-owned children for cascade and rollback assertions.
func (f *Fixture) Graph(ctx context.Context, id string, version int) {
	f.Hydrate(ctx,
		datatest.Table{Name: "component_views", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": id, "version_no": version, "view_id": "view", "view_identity": "view", "name": "root", "role": "root", "source_kind": "sql", "source_sql": "SELECT 1 AS id"}}},
		datatest.Table{Name: "component_fields", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": id, "version_no": version, "view_id": "view", "field_name": "id", "ordinal": 0}}},
		datatest.Table{Name: "component_parameters", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": id, "version_no": version, "parameter_id": "parameter", "parameter_identity": "parameter", "name": "ID", "source_kind": "query", "ordinal": 0}}},
		datatest.Table{Name: "component_predicates", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": id, "version_no": version, "parameter_id": "parameter", "predicate_index": 0, "name": "equal"}}},
		datatest.Table{Name: "component_resource_files", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": id, "version_no": version, "resource_id": "file", "namespace": "fixture.bundle", "resource_path": "guide/SKILL.md", "content": []byte("guide"), "content_size": 5, "content_sha256": strings.Repeat("f", 64), "created_at": Now}}},
		datatest.Table{Name: "component_resource_folders", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": id, "version_no": version, "folder_id": "folder", "namespace": "fixture.bundle", "root_path": "guide", "uri_prefix": "skill://guide/"}}},
		datatest.Table{Name: "component_skill_roots", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": id, "version_no": version, "skill_id": "skill", "folder_id": "folder", "skill_root": "."}}})
}

// RetainedBinding has no active channel: retention alone must fence deletion.
func (f *Fixture) RetainedBinding(ctx context.Context, id string, revision string) {
	raw, err := json.Marshal([]map[string]any{{"componentKind": "dynamic", "componentId": id, "componentRevision": revision}})
	if err != nil {
		f.T.Fatal(err)
	}
	f.Hydrate(ctx,
		datatest.Table{Name: "artifact_release_bundles", Rows: []datatest.Row{{"scope_id": f.Namespace, "tenant_id": "tenant", "account_id": "account", "workspace_id": "workspace", "namespace": "alpha", "bundle_hash": id, "bundle_json": "{}", "created_at": Now, "created_by": "alice", "updated_at": Now, "updated_by": "alice"}}},
		datatest.Table{Name: "artifact_release_bundle_bindings", Rows: []datatest.Row{{"scope_id": f.Namespace, "bundle_hash": id, "bindings_json": string(raw), "bindings_hash": strings.Repeat("b", 64), "created_at": Now, "created_by": "alice", "updated_at": Now, "updated_by": "alice"}}})
}
