package clone_test

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/authz"
	policystore "github.com/viant/authz/datly/store/sql"
	"github.com/viant/datly-studio/internal/datatest"
	_ "github.com/viant/datly-studio/internal/dependencylink"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/versionclone"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/store/sql/accesscatalog"
	clone "github.com/viant/datly-studio/studio/report_versions/clone"
	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/datly/standalone"
	"github.com/viant/datly/standalone/config"
)

func TestNativeClonePreservesPolicyAndRejectsStaleSource(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "studio.db")
	dsn := "file:" + dbPath + "?cache=shared"
	db := datatest.OpenSQLite(t, "clone_policy", "studio")
	// Copy a canonical empty fixture into the file-backed database consumed by
	// the real standalone host; subsequent fixture mutations use its shared handle.
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", dbPath); err != nil {
		t.Fatal(err)
	}
	connections, err := connector.Open(ctx, []connector.Config{{Name: "studio", Driver: "sqlite", DSN: dsn}}, "studio")
	if err != nil {
		t.Fatal(err)
	}
	defer connections.Close()
	db = connections.SQL.DB
	now := "2026-09-29 00:00:00"
	namespace := namespaceaccess.ID("alice", "alpha")
	markdown := "---\nname: clone-test\ndescription: Synthetic clone acceptance\n---\nUse the permitted tool.\n"
	digest := sha256.Sum256([]byte(markdown))
	fileID, folderID, skillID := strings.Repeat("f", 64), strings.Repeat("d", 64), strings.Repeat("e", 64)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": now, "updated_at": now}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"namespace_id": namespace, "owner_id": "alice", "name": "alpha", "title": "Alpha", "status": "active", "created_at": now, "updated_at": now}, {"namespace_id": namespaceaccess.ID("alice", "beta"), "owner_id": "alice", "name": "beta", "title": "Beta", "status": "active", "created_at": now, "updated_at": now}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{"id": "r1", "namespace_id": namespace, "namespace": "alpha", "slug": "source", "title": "Source", "owner_id": "alice", "status": "active", "default_connector_name": "main", "component_scope": "reader/source", "component_name": "source", "created_at": now, "updated_at": now}}},
		datatest.Table{Name: "report_versions", Rows: []datatest.Row{{"namespace_id": namespace, "report_id": "r1", "version_no": 1, "state": "published", "authoring_mode": "dql", "authored_dql": "#package('reader/source')\nSELECT 1 AS id", "generated_dql": "#package('reader/source')\nSELECT 1 AS id", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "source", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": now}}},
		datatest.Table{Name: "report_resource_files", Rows: []datatest.Row{{"namespace_id": namespace, "report_id": "r1", "version_no": 1, "resource_id": fileID, "namespace": "alice.bundle", "resource_path": "skills/clone/SKILL.md", "media_type": "text/markdown", "content": []byte(markdown), "content_size": len(markdown), "content_sha256": hex.EncodeToString(digest[:]), "is_binary": false, "created_at": now}}},
		datatest.Table{Name: "report_resource_folders", Rows: []datatest.Row{{"namespace_id": namespace, "report_id": "r1", "version_no": 1, "folder_id": folderID, "namespace": "alice.bundle", "root_path": "skills/clone", "uri_prefix": "skill://clone/", "ordinal": 0}}},
		datatest.Table{Name: "report_skill_roots", Rows: []datatest.Row{{"namespace_id": namespace, "report_id": "r1", "version_no": 1, "skill_id": skillID, "folder_id": folderID, "skill_root": ".", "ordinal": 0}}},
	); err != nil {
		t.Fatal(err)
	}
	policy := &policystore.Store{DB: db}
	defer policy.Close(ctx)
	source := authz.Document{Resource: authz.Resource{Kind: "component", ID: "r1", Version: "1", Tenant: "tenant"}, Policies: map[string]authz.Policy{"execute": {Mode: "protected", EntityType: "project", Rule: &authz.Rule{Kind: "all", Rules: []authz.Rule{{Kind: "role", Value: "reader"}, {Kind: "exposure", Value: "FEATURE"}}}}}}
	if _, err := policy.Provision(ctx, source, "fixture"); err != nil {
		t.Fatal(err)
	}
	skillPolicy := authz.Document{Resource: authz.Resource{Kind: "skill", ID: skillID, Version: "1", Tenant: "tenant"}, Policies: map[string]authz.Policy{"retrieve": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}}
	if _, err := policy.Provision(ctx, skillPolicy, "fixture"); err != nil {
		t.Fatal(err)
	}
	jwt := datatest.NewJWTFixture(t)
	keyPath := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(keyPath, jwt.PublicKeyPEM(t), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUDIO_ACCESS_ISSUER", "clone-test")
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
	defer jwks.Close()
	root, err := filepath.Abs("../../..")
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
	cfg.Connectors = []connector.Config{{Name: "studio", Driver: "sqlite", DSN: dsn}, {Name: "authz", AliasOf: "studio"}}
	cfg.JWTValidator.CertURL = jwks.URL
	cfg.JWTClaims.Issuer = "clone-test"
	cfg.JWTClaims.Audience = "studio"
	server, err := standalone.New(ctx, standalone.Options{Config: cfg, RequireLinked: true})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Shutdown(ctx)
	serveCtx, stop := context.WithCancel(ctx)
	defer stop()
	served := make(chan error, 1)
	go func() { served <- server.Serve(serveCtx, io.Discard) }()
	readyCtx, cancelReady := context.WithTimeout(ctx, 5*time.Second)
	addresses, err := server.WaitReady(readyCtx)
	cancelReady()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := versionclone.Snapshot(ctx, &accesscatalog.Store{DB: db, Invoker: server}, &policystore.Store{DB: db, Invoker: server}, "r1", namespace, 1); err != nil {
		t.Fatalf("source policy snapshot: %v", err)
	}
	bearer := jwt.BearerWithClaimsAndKeyID(t, jwtv5.MapClaims{"sub": "alice", "iss": "clone-test", "aud": "studio", "tenant": "tenant", "exp": time.Now().Add(time.Hour).Unix()}, "fixture")
	call := func(revision int64) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"reportId": "r1", "versionNo": 1, "expectedSourceRevision": revision})
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/versions.clone", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearer)
		req.Header.Set("X-Studio-Namespace", namespace)
		result := httptest.NewRecorder()
		server.ServeHTTP(result, req)
		return result
	}
	stale := call(2)
	if stale.Code != 409 {
		t.Fatalf("stale clone: %d %s", stale.Code, stale.Body.String())
	}
	selected := namespace
	namespace = namespaceaccess.ID("alice", "beta")
	foreign := call(1)
	namespace = selected
	if foreign.Code != 403 && foreign.Code != 404 {
		t.Fatalf("same owner cloned across namespaces: %d %s", foreign.Code, foreign.Body.String())
	}
	response := call(1)
	if response.Code != 200 {
		t.Fatalf("clone: %d %s", response.Code, response.Body.String())
	}
	copied, err := policy.Get(ctx, authz.Resource{Kind: "component", ID: "r1", Version: "2", Tenant: "tenant"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(source.Policies, copied.Policies) {
		t.Fatal("clone changed source policy restrictions")
	}
	facts := authz.Facts{Subject: "reader", Tenant: "tenant", Issuer: "verified", ValidUntil: time.Now().Add(time.Hour), Roles: []string{"reader"}, Exposures: []string{"FEATURE"}, EntityGroups: authz.EntityGroups{"project": []authz.EntityID{"101"}}}
	request := authz.Request{Resource: copied.Resource, Action: "execute"}
	decision, err := authz.Evaluate(request, copied.Policies, facts, time.Now())
	if err != nil || !decision.Bounded || len(decision.Entities) != 1 {
		t.Fatalf("cloned scope not retained: %+v %v", decision, err)
	}
	for _, missing := range []string{"role", "feature", "entity"} {
		denied := facts
		attempt := request
		switch missing {
		case "role":
			denied.Roles = []string{"other"}
		case "feature":
			denied.Exposures = []string{"OTHER"}
		case "entity":
			entities := []authz.Entity{{Type: "project", ID: "202"}}
			attempt.Selection = &entities
		}
		if _, err := authz.Evaluate(attempt, copied.Policies, denied, time.Now()); err == nil {
			t.Fatalf("clone broadened %s permission", missing)
		}
	}
	copiedSkill, err := policy.Get(ctx, authz.Resource{Kind: "skill", ID: skillID, Version: "2", Tenant: "tenant"})
	if err != nil || !reflect.DeepEqual(copiedSkill.Policies, skillPolicy.Policies) {
		t.Fatalf("explicit skill policy was not preserved: %v", err)
	}
	for _, table := range []string{"report_resource_files", "report_resource_folders", "report_skill_roots"} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE report_id='r1' AND version_no=2").Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s copy count=%d error=%v", table, count, err)
		}
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM report_versions WHERE report_id='r1'").Scan(&count); err != nil || count != 2 {
		t.Fatalf("unexpected clone version count: %d %v", count, err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER reject_clone_policy BEFORE INSERT ON resource_policy_heads WHEN NEW.resource_version='3' AND NEW.resource_kind='skill' BEGIN SELECT RAISE(ABORT,'injected late policy failure'); END`); err != nil {
		t.Fatal(err)
	}
	failure := call(1)
	if failure.Code != 409 {
		t.Fatalf("injected policy failure was not reached: %d %s", failure.Code, failure.Body.String())
	}
	for _, table := range []string{"report_versions", "report_resource_files", "report_resource_folders", "report_skill_roots"} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE report_id='r1' AND version_no=3").Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial clone survived in %s: %d %v", table, count, err)
		}
	}
	for _, table := range []string{"resource_policy_heads", "resource_policy_revisions"} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE resource_version='3'").Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial policy survived in %s: %d %v", table, count, err)
		}
	}
	if _, err := db.ExecContext(ctx, "DROP TRIGGER reject_clone_policy"); err != nil {
		t.Fatal(err)
	}
	parsed, err := jwtv5.Parse(strings.TrimPrefix(bearer, "Bearer "), func(*jwtv5.Token) (any, error) { return key, nil })
	if err != nil || !parsed.Valid {
		t.Fatal("signed SDK fixture did not verify")
	}
	client, err := sdk.NewClient(&clone.Transport{Invoker: server})
	if err != nil {
		t.Fatal(err)
	}
	clientCtx := sdk.WithNamespaceSelection(sdk.WithVerifiedCredential(sdk.WithPrincipal(ctx, sdk.Principal{Subject: "alice"}), sdk.VerifiedCredential{Bearer: bearer, Claims: parsed.Claims}), namespace)
	third, err := client.Versions().Clone(clientCtx, "r1", sdk.CloneVersionInput{VersionNo: 1, ExpectedSourceRevision: 1})
	if err != nil || third.VersionNo != 3 {
		t.Fatalf("in-process SDK clone failed: %+v %v", third, err)
	}
	thirdPolicy, err := policy.Get(ctx, authz.Resource{Kind: "component", ID: "r1", Version: "3", Tenant: "tenant"})
	if err != nil || !reflect.DeepEqual(thirdPolicy.Policies, source.Policies) {
		t.Fatalf("SDK clone policy mismatch: %v", err)
	}
	if len(addresses) != 2 {
		t.Fatalf("native MCP listener missing: %v", addresses)
	}
	mcpBody, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "studio.sdk.versions.clone", "arguments": map[string]any{"namespaceId": namespace, "reportId": "r1", "versionNo": 1, "expectedSourceRevision": 1}}})
	mcpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+addresses[1]+"/mcp", bytes.NewReader(mcpBody))
	if err != nil {
		t.Fatal(err)
	}
	mcpRequest.Header.Set("Content-Type", "application/json")
	mcpRequest.Header.Set("Accept", "application/json, text/event-stream")
	mcpRequest.Header.Set("Authorization", bearer)
	mcpResponse, err := http.DefaultClient.Do(mcpRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer mcpResponse.Body.Close()
	responseBytes, err := io.ReadAll(mcpResponse.Body)
	if err != nil {
		t.Fatal(err)
	}
	var rpc struct {
		Error  json.RawMessage `json:"error"`
		Result struct {
			IsError bool `json:"isError"`
			Data    struct {
				VersionNo int `json:"versionNo"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err = json.Unmarshal(responseBytes, &rpc); err != nil {
		t.Fatal(err)
	}
	if len(rpc.Error) > 0 || rpc.Result.IsError || rpc.Result.Data.VersionNo != 4 {
		t.Fatalf("MCP clone failed: %s", responseBytes)
	}
	fourthPolicy, err := policy.Get(ctx, authz.Resource{Kind: "component", ID: "r1", Version: "4", Tenant: "tenant"})
	if err != nil || !reflect.DeepEqual(fourthPolicy.Policies, source.Policies) {
		t.Fatalf("MCP clone policy mismatch: %v", err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE resource_policy_heads SET revision=999 WHERE resource_kind='component' AND resource_id='r1' AND resource_version='1'"); err != nil {
		t.Fatal(err)
	}
	corrupt := call(1)
	if corrupt.Code != 409 {
		t.Fatalf("corrupt source policy was not rejected: %d %s", corrupt.Code, corrupt.Body.String())
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM report_versions WHERE report_id='r1'").Scan(&count); err != nil || count != 4 {
		t.Fatalf("corrupt source created a partial version: %d %v", count, err)
	}
}
