package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	resourceaccess "github.com/viant/authz"
	accessstore "github.com/viant/authz/component/store/sql"
	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/warmupprojection"
	studiopreview "github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension"
	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/datly/standalone"
	standaloneconfig "github.com/viant/datly/standalone/config"
	mcpschema "github.com/viant/mcp-protocol/schema"
	"github.com/viant/scy"
	scyjwt "github.com/viant/scy/auth/jwt"
	"github.com/viant/scy/auth/jwt/verifier"
)

type staticSessionVerifier struct{}

func (staticSessionVerifier) VerifyClaims(context.Context, string) (*scyjwt.Claims, error) {
	claims := &scyjwt.Claims{}
	claims.Subject = "alice"
	return claims, nil
}

// Focused SDK tests compile each endpoint in isolation. This test also boots
// the exact configured static package set as one generation, catching route,
// factory, type-link and MCP-name collisions before deployment.
func TestSelectedStudioStaticComponentsBootstrapTogether(t *testing.T) {
	ctx := context.Background()
	t.Setenv("STUDIO_PREDICATE_PACKAGES", "")
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	configuration, err := (standaloneconfig.Loader{}).Load(ctx, filepath.Join(root, "datly.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	dsn := "file:" + filepath.Join(t.TempDir(), "studio.db") + "?cache=shared"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	if err = schema.EnsureSQLiteSequenceLedger(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	configuration.Connectors = []connector.Config{{Name: "studio", Driver: "sqlite", DSN: dsn}, {Name: "authz", Driver: "sqlite", DSN: dsn}}
	configuration.Endpoint.Address = "127.0.0.1:0"
	configuration.Endpoint.Port = 0
	if configuration.MCP == nil {
		t.Fatal("selected static Studio configuration has no SDK MCP listener")
	}
	configuration.MCP.Address = "127.0.0.1:0"
	configuration.MCP.Port = nil
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	configuration.JWTValidator = &verifier.Config{RSA: []*scy.Resource{{URL: "studio-static-bootstrap-test", Data: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public})}}}
	server, err := standalone.New(ctx, standalone.Options{Config: configuration, RequireLinked: true})
	if err != nil {
		t.Fatalf("construct selected static host: %v", err)
	}
	serveCtx, stop := context.WithCancel(ctx)
	served := make(chan error, 1)
	go func() { served <- server.Serve(serveCtx, nil) }()
	defer func() {
		stop()
		if serveErr := <-served; serveErr != nil {
			t.Errorf("static host serve/shutdown: %v", serveErr)
		}
	}()
	readyCtx, readyCancel := context.WithTimeout(ctx, 30*time.Second)
	addresses, err := server.WaitReady(readyCtx)
	readyCancel()
	if err != nil || len(addresses) != 2 {
		t.Fatalf("static HTTP/MCP listeners=%v err=%v", addresses, err)
	}
	preflight := httptest.NewRequest(http.MethodOptions, "/v1/studio/sdk/namespaces.list", nil)
	preflight.Header.Set("Origin", "https://studio.example.com")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	preflight.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	preflightResponse := httptest.NewRecorder()
	server.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusNoContent || preflightResponse.Header().Get("Access-Control-Allow-Origin") != "https://studio.example.com" ||
		preflightResponse.Header().Get("Access-Control-Allow-Credentials") == "true" {
		t.Fatalf("direct browser preflight status=%d headers=%v", preflightResponse.Code, preflightResponse.Header())
	}
	unauthenticated := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/namespaces.list", strings.NewReader(`{}`))
	unauthenticated.Header.Set("Content-Type", "application/json")
	unauthenticated.Header.Set("Origin", "https://studio.example.com")
	unauthenticatedResponse := httptest.NewRecorder()
	server.ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized ||
		unauthenticatedResponse.Header().Get("Access-Control-Allow-Origin") != "https://studio.example.com" ||
		unauthenticatedResponse.Header().Get("Access-Control-Allow-Credentials") == "true" ||
		len(unauthenticatedResponse.Header().Values("Set-Cookie")) != 0 {
		t.Fatalf("direct browser denial status=%d headers=%v", unauthenticatedResponse.Code, unauthenticatedResponse.Header())
	}
	metadata, err := server.Metadata(ctx)
	if err != nil || metadata == nil || metadata.Revision != 1 || len(metadata.Components) == 0 {
		t.Fatalf("published static metadata=%+v err=%v", metadata, err)
	}
	for _, path := range []string{
		"/_studio/report-store/catalog", "/_studio/report-version-store/touch",
		"/_authz/policy-store/read", "/_authz/policy-store/write",
		"/_studio/resource-file-store/write", "/_studio/resource-folder-store/write",
		"/_studio/skill-root-store/write", "/_studio/resource-namespace-claim-store/write",
		"/_studio/resource-namespace-store/presence", "/_studio/resource-namespace-store/usage",
	} {
		found := false
		for _, component := range metadata.Components {
			for _, route := range component.Routes {
				if route == nil || route.Path != path {
					continue
				}
				found = true
				if !route.Internal || len(route.MCP) != 0 {
					t.Errorf("private resource route %s is externally exposed", path)
				}
			}
		}
		if !found {
			t.Errorf("private resource route %s is not linked", path)
		}
	}
	contract, err := os.ReadFile(filepath.Join(root, "sdk", "openapi", "studio.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]json.RawMessage `json:"paths"`
	}
	if err = json.Unmarshal(contract, &document); err != nil {
		t.Fatal(err)
	}
	for path := range document.Paths {
		matches := 0
		for _, component := range metadata.Components {
			for _, route := range component.Routes {
				if route == nil || route.Method != http.MethodPost || route.Path != path || route.Internal {
					continue
				}
				matches++
				wantTool := sdkToolForPath(path)
				foundTool := false
				for _, exposure := range route.MCP {
					foundTool = foundTool || exposure != nil && exposure.Name == wantTool
				}
				if !foundTool {
					t.Errorf("static route %s is missing MCP tool %s", path, wantTool)
				}
			}
		}
		if matches != 1 {
			t.Errorf("static route %s has %d selected owners, want 1", path, matches)
		}
	}
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"sub": "alice", "user_id": 1, "iss": "https://idp.viantinc.com", "aud": "datly-studio-web", "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	request := func(path, body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		return req
	}
	created := httptest.NewRecorder()
	server.ServeHTTP(created, request("/v1/studio/sdk/namespaces.create", `{"name":"production.audit","title":"Production Audit"}`))
	if created.Code != http.StatusOK || !strings.Contains(created.Body.String(), `"name":"production.audit"`) {
		t.Fatalf("combined static create status=%d body=%s", created.Code, created.Body.String())
	}
	listed := httptest.NewRecorder()
	server.ServeHTTP(listed, request("/v1/studio/sdk/namespaces.list", `{}`))
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"name":"production.audit"`) {
		t.Fatalf("combined static list status=%d body=%s", listed.Code, listed.Body.String())
	}
	updated := httptest.NewRecorder()
	server.ServeHTTP(updated, request("/v1/studio/sdk/namespaces.update", `{"name":"production.audit","input":{"title":"Production Ready","etag":1}}`))
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), `"title":"Production Ready"`) || !strings.Contains(updated.Body.String(), `"etag":2`) {
		t.Fatalf("combined static namespace update status=%d body=%s", updated.Code, updated.Body.String())
	}
	stale := httptest.NewRecorder()
	server.ServeHTTP(stale, request("/v1/studio/sdk/namespaces.update", `{"name":"production.audit","input":{"title":"Stale","etag":1}}`))
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale namespace update status=%d body=%s", stale.Code, stale.Body.String())
	}
	bobToken, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"sub": "bob", "user_id": 2, "iss": "https://idp.viantinc.com", "aud": "datly-studio-web", "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	bobRequest := request("/v1/studio/sdk/namespaces.update", `{"name":"production.audit","input":{"title":"Stolen","etag":2}}`)
	bobRequest.Header.Set("Authorization", "Bearer "+bobToken)
	bob := httptest.NewRecorder()
	server.ServeHTTP(bob, bobRequest)
	if bob.Code != http.StatusForbidden {
		t.Fatalf("non-owner namespace update status=%d body=%s", bob.Code, bob.Body.String())
	}
	invalidNamespace := httptest.NewRecorder()
	server.ServeHTTP(invalidNamespace, request("/v1/studio/sdk/namespaces.update", `{"name":"production.audit","input":{"title":"","etag":2}}`))
	if invalidNamespace.Code != http.StatusBadRequest {
		t.Fatalf("empty namespace title status=%d body=%s", invalidNamespace.Code, invalidNamespace.Body.String())
	}
	store, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err = store.ExecContext(ctx, `INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at)
		VALUES('main','sqlite','alice','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ExecContext(ctx, `UPDATE connectors SET dsn_template=? WHERE name='main'`, dsn); err != nil {
		t.Fatal(err)
	}
	secretPath := filepath.Join(t.TempDir(), "connector-secret.txt")
	if err = os.WriteFile(secretPath, []byte(dsn), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ExecContext(ctx, `INSERT INTO connectors(name,driver,secret_ref,owner_id,status,etag,created_at,updated_at)
		VALUES('secret_catalog','sqlite',?,'alice','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, secretPath); err != nil {
		t.Fatal(err)
	}
	secretSchemas := httptest.NewRecorder()
	server.ServeHTTP(secretSchemas, request("/v1/studio/sdk/connectors.schemas", `{"name":"secret_catalog","input":{}}`))
	if secretSchemas.Code != http.StatusOK || !strings.Contains(secretSchemas.Body.String(), `"items"`) ||
		strings.Contains(secretSchemas.Body.String(), dsn) || strings.Contains(secretSchemas.Body.String(), secretPath) {
		t.Fatalf("static secret-backed schema catalog status=%d", secretSchemas.Code)
	}
	secretProbe := httptest.NewRecorder()
	server.ServeHTTP(secretProbe, request("/v1/studio/sdk/connectors.test", `{"name":"secret_catalog"}`))
	if secretProbe.Code != http.StatusOK || !strings.Contains(secretProbe.Body.String(), `"status":"passed"`) ||
		strings.Contains(secretProbe.Body.String(), dsn) || strings.Contains(secretProbe.Body.String(), secretPath) {
		t.Fatalf("static secret-backed connector probe status=%d", secretProbe.Code)
	}
	for _, name := range []string{"probe_http", "probe_mcp", "probe_bff"} {
		if _, err = store.ExecContext(ctx, `INSERT INTO connectors(name,driver,dsn_template,owner_id,status,etag,created_at,updated_at)
			VALUES(?,'sqlite',?,'alice','draft',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, name, dsn); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = store.ExecContext(ctx, `INSERT INTO connectors(name,driver,dsn_template,owner_id,status,etag,created_at,updated_at)
		VALUES('probe_failed','sqlite','file:/nonexistent-studio-probe/catalog.db?mode=ro','alice','draft',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	probeHTTP := httptest.NewRecorder()
	server.ServeHTTP(probeHTTP, request("/v1/studio/sdk/connectors.test", `{"name":"probe_http"}`))
	if probeHTTP.Code != http.StatusOK || !strings.Contains(probeHTTP.Body.String(), `"status":"passed"`) || strings.Contains(probeHTTP.Body.String(), dsn) {
		t.Fatalf("native connector probe status=%d body=%s", probeHTTP.Code, probeHTTP.Body.String())
	}
	var probeStatus string
	var probeETag int64
	if err = store.QueryRowContext(ctx, `SELECT last_test_status,etag FROM connectors WHERE name='probe_http'`).Scan(&probeStatus, &probeETag); err != nil || probeStatus != "passed" || probeETag != 1 {
		t.Fatalf("native connector probe persistence status=%q etag=%d err=%v", probeStatus, probeETag, err)
	}
	failedProbe := httptest.NewRecorder()
	server.ServeHTTP(failedProbe, request("/v1/studio/sdk/connectors.test", `{"name":"probe_failed"}`))
	if failedProbe.Code != http.StatusOK || !strings.Contains(failedProbe.Body.String(), `"status":"failed"`) ||
		!strings.Contains(failedProbe.Body.String(), `"errorCode":"connectivity_failed"`) || strings.Contains(failedProbe.Body.String(), "nonexistent-studio-probe") {
		t.Fatalf("failed connector probe status=%d body=%s", failedProbe.Code, failedProbe.Body.String())
	}
	bobProbeRequest := request("/v1/studio/sdk/connectors.test", `{"name":"probe_http"}`)
	bobProbeRequest.Header.Set("Authorization", "Bearer "+bobToken)
	bobProbe := httptest.NewRecorder()
	server.ServeHTTP(bobProbe, bobProbeRequest)
	if bobProbe.Code != http.StatusForbidden {
		t.Fatalf("non-editor connector probe status=%d body=%s", bobProbe.Code, bobProbe.Body.String())
	}
	schemas := httptest.NewRecorder()
	server.ServeHTTP(schemas, request("/v1/studio/sdk/connectors.schemas", `{"name":"main","input":{}}`))
	if schemas.Code != http.StatusOK || !strings.Contains(schemas.Body.String(), `"items"`) || strings.Contains(schemas.Body.String(), dsn) {
		t.Fatalf("native connector schemas status=%d body=%s", schemas.Code, schemas.Body.String())
	}
	tables := httptest.NewRecorder()
	server.ServeHTTP(tables, request("/v1/studio/sdk/connectors.tables", `{"name":"main","input":{"query":"components","limit":5}}`))
	if tables.Code != http.StatusOK || !strings.Contains(tables.Body.String(), `"name":"components"`) ||
		!strings.Contains(tables.Body.String(), `"limit":5`) || strings.Contains(tables.Body.String(), dsn) {
		t.Fatalf("native connector tables status=%d body=%s", tables.Code, tables.Body.String())
	}
	tableDetail := httptest.NewRecorder()
	server.ServeHTTP(tableDetail, request("/v1/studio/sdk/connectors.table", `{"name":"main","input":{"table":"components"}}`))
	if tableDetail.Code != http.StatusOK || !strings.Contains(tableDetail.Body.String(), `"name":"components"`) ||
		!strings.Contains(tableDetail.Body.String(), `"name":"slug"`) || strings.Contains(tableDetail.Body.String(), dsn) {
		t.Fatalf("native connector table detail status=%d body=%s", tableDetail.Code, tableDetail.Body.String())
	}
	missingTable := httptest.NewRecorder()
	server.ServeHTTP(missingTable, request("/v1/studio/sdk/connectors.table", `{"name":"main","input":{}}`))
	if missingTable.Code != http.StatusBadRequest {
		t.Fatalf("missing connector table status=%d body=%s", missingTable.Code, missingTable.Body.String())
	}
	bobSchemasRequest := request("/v1/studio/sdk/connectors.schemas", `{"name":"main","input":{}}`)
	bobSchemasRequest.Header.Set("Authorization", "Bearer "+bobToken)
	bobSchemas := httptest.NewRecorder()
	server.ServeHTTP(bobSchemas, bobSchemasRequest)
	if bobSchemas.Code == http.StatusOK || strings.Contains(bobSchemas.Body.String(), dsn) {
		t.Fatalf("non-viewer connector schemas status=%d body=%s", bobSchemas.Code, bobSchemas.Body.String())
	}
	for _, operation := range []string{"tables", "table"} {
		body := `{"name":"main","input":{"query":"components"}}`
		if operation == "table" {
			body = `{"name":"main","input":{"table":"components"}}`
		}
		denied := request("/v1/studio/sdk/connectors."+operation, body)
		denied.Header.Set("Authorization", "Bearer "+bobToken)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, denied)
		if response.Code == http.StatusOK || strings.Contains(response.Body.String(), dsn) {
			t.Fatalf("non-viewer connector %s status=%d body=%s", operation, response.Code, response.Body.String())
		}
	}
	dockerScaleDSN := os.Getenv("STUDIO_SCALE_MYSQL_DSN")
	if dockerScaleDSN != "" {
		if _, err = store.ExecContext(ctx, `INSERT INTO connectors(name,driver,dsn_template,owner_id,status,etag,created_at,updated_at)
			VALUES('scale_mysql','mysql',?,'alice','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, dockerScaleDSN); err != nil {
			t.Fatal(err)
		}
		mysqlProbe := httptest.NewRecorder()
		server.ServeHTTP(mysqlProbe, request("/v1/studio/sdk/connectors.test", `{"name":"scale_mysql"}`))
		if mysqlProbe.Code != http.StatusOK || !strings.Contains(mysqlProbe.Body.String(), `"status":"passed"`) || strings.Contains(mysqlProbe.Body.String(), dockerScaleDSN) {
			t.Fatalf("Docker native connector probe status=%d", mysqlProbe.Code)
		}
		mysqlSQL := httptest.NewRecorder()
		server.ServeHTTP(mysqlSQL, request("/v1/studio/sdk/connectors.test_sql", `{"name":"scale_mysql","input":{"sql":"SELECT ID,FIELD_59 FROM STUDIO_WIDE_60 ORDER BY ID","limit":1}}`))
		if mysqlSQL.Code != http.StatusOK || !strings.Contains(mysqlSQL.Body.String(), "omega") || strings.Contains(mysqlSQL.Body.String(), dockerScaleDSN) {
			t.Fatalf("Docker native transient SQL test status=%d", mysqlSQL.Code)
		}
		if _, err = store.ExecContext(ctx, `INSERT INTO components(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
			VALUES('preview-wide','production.audit','preview-wide','Preview Wide','alice','draft','scale_mysql','github.com/viant/datly-studio/dynamic/preview_wide','reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
			t.Fatal(err)
		}
		wideDQL := `#package('github.com/viant/datly-studio/dynamic/preview_wide')
#setting($_ = $connector('scale_mysql'))
#setting($_ = $route('/v1/studio/readers/preview-wide','GET'))
#define($_ = $Rows<[]*Row>(output/view))
SELECT wide.*, type(wide,'Row') FROM (SELECT * FROM STUDIO_WIDE_60) wide`
		if _, err = store.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
			VALUES('preview-wide',1,'draft','dql',?,?,'{}','studio.v1','wide-hash','{}','valid','v1','studio.v1',1,'alice',CURRENT_TIMESTAMP)`, wideDQL, wideDQL); err != nil {
			t.Fatal(err)
		}
		wideValidation := httptest.NewRecorder()
		server.ServeHTTP(wideValidation, request("/v1/studio/sdk/versions.validate", `{"reportId":"preview-wide","versionNo":1,"expectedSourceRevision":1}`))
		if wideValidation.Code != http.StatusOK || !strings.Contains(wideValidation.Body.String(), `"valid":true`) || strings.Contains(wideValidation.Body.String(), dockerScaleDSN) {
			t.Fatalf("Docker native wide validation status=%d", wideValidation.Code)
		}
		widePreview := httptest.NewRecorder()
		server.ServeHTTP(widePreview, request("/v1/studio/sdk/preview.execute", `{"reportId":"preview-wide","versionNo":1,"input":{"limit":10}}`))
		var wideOutput struct {
			Data map[string][]map[string]any `json:"data"`
		}
		wideColumns := 0
		decodeErr := json.Unmarshal(widePreview.Body.Bytes(), &wideOutput)
		if len(wideOutput.Data["Rows"]) != 0 {
			wideColumns = len(wideOutput.Data["Rows"][0])
		}
		if widePreview.Code != http.StatusOK || decodeErr != nil ||
			len(wideOutput.Data["Rows"]) != 2 || wideColumns != 60 || strings.Contains(widePreview.Body.String(), dockerScaleDSN) {
			t.Fatalf("Docker native wide preview status=%d rows=%d firstColumns=%d err=%v", widePreview.Code, len(wideOutput.Data["Rows"]), wideColumns, decodeErr)
		}
		wideView := httptest.NewRecorder()
		server.ServeHTTP(wideView, request("/v1/studio/sdk/versions.test_view", `{"reportId":"preview-wide","versionNo":1,"view":"wide","input":{"limit":10}}`))
		if wideView.Code != http.StatusOK || !strings.Contains(wideView.Body.String(), "omega") || strings.Contains(wideView.Body.String(), dockerScaleDSN) {
			t.Fatalf("Docker native wide view test status=%d", wideView.Code)
		}
		views := httptest.NewRecorder()
		server.ServeHTTP(views, request("/v1/studio/sdk/connectors.tables", `{"name":"scale_mysql","input":{"query":"STUDIO_VIEW_","limit":100}}`))
		var viewPage struct {
			Items []struct {
				Name string `json:"name"`
			} `json:"items"`
		}
		if views.Code != http.StatusOK || json.Unmarshal(views.Body.Bytes(), &viewPage) != nil || len(viewPage.Items) != 20 || strings.Contains(views.Body.String(), dockerScaleDSN) {
			t.Fatalf("Docker native view catalog status=%d views=%d", views.Code, len(viewPage.Items))
		}
		wide := httptest.NewRecorder()
		server.ServeHTTP(wide, request("/v1/studio/sdk/connectors.table", `{"name":"scale_mysql","input":{"table":"STUDIO_WIDE_60"}}`))
		var wideTable struct {
			Columns []struct {
				Name string `json:"name"`
			} `json:"columns"`
		}
		if wide.Code != http.StatusOK || json.Unmarshal(wide.Body.Bytes(), &wideTable) != nil || len(wideTable.Columns) != 60 || strings.Contains(wide.Body.String(), dockerScaleDSN) {
			t.Fatalf("Docker native wide detail status=%d columns=%d", wide.Code, len(wideTable.Columns))
		}
	}
	for _, name := range []string{"disable_http", "disable_mcp", "disable_bff", "delegated", "delete_http", "delete_mcp", "delete_bff", "update_http", "update_mcp", "update_bff"} {
		if _, err = store.ExecContext(ctx, `INSERT INTO connectors(name,driver,dsn_template,secret_ref,owner_id,status,options_json,etag,created_at,updated_at)
			VALUES(?,'mysql','private-dsn','vault://private','alice','active','{}',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = store.ExecContext(ctx, `INSERT INTO connectors(name,driver,dsn_template,secret_ref,owner_id,status,options_json,last_test_status,last_tested_at,etag,created_at,updated_at)
		VALUES('update_description','mysql','private-dsn','vault://private','alice','active','{}','passed',CURRENT_TIMESTAMP,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ExecContext(ctx, `UPDATE connectors SET last_test_status='passed', last_tested_at=CURRENT_TIMESTAMP WHERE name='update_http'`); err != nil {
		t.Fatal(err)
	}
	forbiddenUpdate := request("/v1/studio/sdk/connectors.update", `{"name":"update_http","input":{"description":"Stolen","etag":1}}`)
	forbiddenUpdate.Header.Set("Authorization", "Bearer "+bobToken)
	deniedUpdate := httptest.NewRecorder()
	server.ServeHTTP(deniedUpdate, forbiddenUpdate)
	if deniedUpdate.Code != http.StatusForbidden {
		t.Fatalf("non-editor connector update status=%d body=%s", deniedUpdate.Code, deniedUpdate.Body.String())
	}
	rotated := httptest.NewRecorder()
	server.ServeHTTP(rotated, request("/v1/studio/sdk/connectors.update", `{"name":"update_http","input":{"dsnTemplate":"new-private-dsn","etag":1}}`))
	if rotated.Code != http.StatusOK || !strings.Contains(rotated.Body.String(), `"status":"draft"`) ||
		!strings.Contains(rotated.Body.String(), `"dsnConfigured":true`) || !strings.Contains(rotated.Body.String(), `"etag":2`) ||
		strings.Contains(rotated.Body.String(), "new-private-dsn") || strings.Contains(rotated.Body.String(), "vault://private") ||
		strings.Contains(rotated.Body.String(), `"lastTestStatus":"passed"`) {
		t.Fatalf("native connector config rotation status=%d body=%s", rotated.Code, rotated.Body.String())
	}
	staleUpdate := httptest.NewRecorder()
	server.ServeHTTP(staleUpdate, request("/v1/studio/sdk/connectors.update", `{"name":"update_http","input":{"description":"Stale","etag":1}}`))
	if staleUpdate.Code != http.StatusConflict {
		t.Fatalf("stale connector update status=%d body=%s", staleUpdate.Code, staleUpdate.Body.String())
	}
	descriptionOnly := httptest.NewRecorder()
	server.ServeHTTP(descriptionOnly, request("/v1/studio/sdk/connectors.update", `{"name":"update_description","input":{"description":"Reviewed","etag":1}}`))
	if descriptionOnly.Code != http.StatusOK || !strings.Contains(descriptionOnly.Body.String(), `"status":"active"`) ||
		!strings.Contains(descriptionOnly.Body.String(), `"lastTestStatus":"passed"`) || strings.Contains(descriptionOnly.Body.String(), "private-dsn") {
		t.Fatalf("description-only connector update status=%d body=%s", descriptionOnly.Code, descriptionOnly.Body.String())
	}
	for _, name := range []string{"activate_http", "activate_mcp", "activate_bff", "activate_untested"} {
		probeStatus := "passed"
		if name == "activate_untested" {
			probeStatus = "failed"
		}
		if _, err = store.ExecContext(ctx, `INSERT INTO connectors(name,driver,dsn_template,secret_ref,owner_id,status,options_json,last_test_status,last_tested_at,etag,created_at,updated_at)
			VALUES(?,'mysql','private-dsn','vault://private','alice','draft','{}',?,CURRENT_TIMESTAMP,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, name, probeStatus); err != nil {
			t.Fatal(err)
		}
	}
	untested := httptest.NewRecorder()
	server.ServeHTTP(untested, request("/v1/studio/sdk/connectors.activate", `{"name":"activate_untested","etag":1}`))
	if untested.Code != http.StatusBadRequest || !strings.Contains(untested.Body.String(), "connectivity test") {
		t.Fatalf("untested connector activation status=%d body=%s", untested.Code, untested.Body.String())
	}
	forbiddenActivate := request("/v1/studio/sdk/connectors.activate", `{"name":"activate_http","etag":1}`)
	forbiddenActivate.Header.Set("Authorization", "Bearer "+bobToken)
	deniedActivate := httptest.NewRecorder()
	server.ServeHTTP(deniedActivate, forbiddenActivate)
	if deniedActivate.Code != http.StatusForbidden {
		t.Fatalf("non-owner connector activation status=%d body=%s", deniedActivate.Code, deniedActivate.Body.String())
	}
	activated := httptest.NewRecorder()
	server.ServeHTTP(activated, request("/v1/studio/sdk/connectors.activate", `{"name":"activate_http","etag":1}`))
	if activated.Code != http.StatusOK || !strings.Contains(activated.Body.String(), `"status":"active"`) ||
		!strings.Contains(activated.Body.String(), `"etag":2`) || strings.Contains(activated.Body.String(), "private-dsn") ||
		strings.Contains(activated.Body.String(), "vault://private") {
		t.Fatalf("native connector activation status=%d body=%s", activated.Code, activated.Body.String())
	}
	staleActivate := httptest.NewRecorder()
	server.ServeHTTP(staleActivate, request("/v1/studio/sdk/connectors.activate", `{"name":"activate_http","etag":1}`))
	if staleActivate.Code != http.StatusConflict {
		t.Fatalf("stale connector activation status=%d body=%s", staleActivate.Code, staleActivate.Body.String())
	}
	forbiddenDisable := request("/v1/studio/sdk/connectors.disable", `{"name":"disable_http","etag":1}`)
	forbiddenDisable.Header.Set("Authorization", "Bearer "+bobToken)
	deniedDisable := httptest.NewRecorder()
	server.ServeHTTP(deniedDisable, forbiddenDisable)
	if deniedDisable.Code != http.StatusForbidden {
		t.Fatalf("non-owner connector disable status=%d body=%s", deniedDisable.Code, deniedDisable.Body.String())
	}
	disabled := httptest.NewRecorder()
	server.ServeHTTP(disabled, request("/v1/studio/sdk/connectors.disable", `{"name":"disable_http","etag":1}`))
	if disabled.Code != http.StatusOK || !strings.Contains(disabled.Body.String(), `"status":"disabled"`) ||
		!strings.Contains(disabled.Body.String(), `"dsnConfigured":true`) || !strings.Contains(disabled.Body.String(), `"secretConfigured":true`) ||
		strings.Contains(disabled.Body.String(), "private-dsn") || strings.Contains(disabled.Body.String(), "vault://private") {
		t.Fatalf("native connector disable status=%d body=%s", disabled.Code, disabled.Body.String())
	}
	staleDisable := httptest.NewRecorder()
	server.ServeHTTP(staleDisable, request("/v1/studio/sdk/connectors.disable", `{"name":"disable_http","etag":1}`))
	if staleDisable.Code != http.StatusConflict {
		t.Fatalf("stale connector disable status=%d body=%s", staleDisable.Code, staleDisable.Body.String())
	}
	forbiddenDelete := request("/v1/studio/sdk/connectors.delete", `{"name":"delete_http","etag":1}`)
	forbiddenDelete.Header.Set("Authorization", "Bearer "+bobToken)
	deniedDelete := httptest.NewRecorder()
	server.ServeHTTP(deniedDelete, forbiddenDelete)
	if deniedDelete.Code != http.StatusForbidden {
		t.Fatalf("non-editor connector delete status=%d body=%s", deniedDelete.Code, deniedDelete.Body.String())
	}
	staleConnectorDelete := httptest.NewRecorder()
	server.ServeHTTP(staleConnectorDelete, request("/v1/studio/sdk/connectors.delete", `{"name":"delete_http","etag":9}`))
	if staleConnectorDelete.Code != http.StatusNotFound {
		t.Fatalf("stale connector delete status=%d body=%s", staleConnectorDelete.Code, staleConnectorDelete.Body.String())
	}
	deletedConnector := httptest.NewRecorder()
	server.ServeHTTP(deletedConnector, request("/v1/studio/sdk/connectors.delete", `{"name":"delete_http","etag":1}`))
	if deletedConnector.Code != http.StatusNoContent {
		t.Fatalf("native connector delete status=%d body=%s", deletedConnector.Code, deletedConnector.Body.String())
	}
	forged := httptest.NewRecorder()
	server.ServeHTTP(forged, request("/v1/studio/sdk/components.create", `{"slug":"forged","title":"Forged","defaultConnectorName":"main","ownerId":"mallory"}`))
	if forged.Code != http.StatusForbidden {
		t.Fatalf("forged report owner status=%d body=%s", forged.Code, forged.Body.String())
	}
	reportCreated := httptest.NewRecorder()
	server.ServeHTTP(reportCreated, request("/v1/studio/sdk/components.create", `{"slug":"native-first","title":"Native First","namespace":"production.audit","defaultConnectorName":"main","componentScope":"attacker","componentName":"attacker"}`))
	var reportCount int
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM components WHERE slug='native-first' AND owner_id='alice' AND namespace='production.audit'`).Scan(&reportCount); err != nil {
		t.Fatal(err)
	}
	if reportCreated.Code != http.StatusOK || !strings.Contains(reportCreated.Body.String(), `"slug":"native-first"`) ||
		!strings.Contains(reportCreated.Body.String(), `"componentName":"reader"`) || strings.Contains(reportCreated.Body.String(), `"attacker"`) {
		t.Fatalf("combined static report create status=%d body=%s persisted=%d", reportCreated.Code, reportCreated.Body.String(), reportCount)
	}
	if reportCount != 1 {
		t.Fatalf("native report persistence count=%d", reportCount)
	}
	idleRuntime := httptest.NewRecorder()
	server.ServeHTTP(idleRuntime, request("/v1/studio/sdk/runtime.status", `{}`))
	if idleRuntime.Code != http.StatusOK || !strings.Contains(idleRuntime.Body.String(), `"status":"idle"`) {
		t.Fatalf("native runtime status=%d body=%s", idleRuntime.Code, idleRuntime.Body.String())
	}
	deniedRuntime := request("/v1/studio/sdk/runtime.status", `{}`)
	deniedRuntime.Header.Set("Authorization", "Bearer "+bobToken)
	deniedRuntimeResponse := httptest.NewRecorder()
	server.ServeHTTP(deniedRuntimeResponse, deniedRuntime)
	if deniedRuntimeResponse.Code != http.StatusForbidden {
		t.Fatalf("non-publisher runtime status=%d body=%s", deniedRuntimeResponse.Code, deniedRuntimeResponse.Body.String())
	}
	testedSQL := httptest.NewRecorder()
	server.ServeHTTP(testedSQL, request("/v1/studio/sdk/connectors.test_sql", `{"name":"main","input":{"sql":"SELECT id,slug FROM components ORDER BY slug","limit":1}}`))
	if testedSQL.Code != http.StatusOK || !strings.Contains(testedSQL.Body.String(), `"native-first"`) || strings.Contains(testedSQL.Body.String(), dsn) {
		t.Fatalf("native transient SQL test status=%d body=%s", testedSQL.Code, testedSQL.Body.String())
	}
	invalidSQL := httptest.NewRecorder()
	server.ServeHTTP(invalidSQL, request("/v1/studio/sdk/connectors.test_sql", `{"name":"main","input":{"sql":""}}`))
	if invalidSQL.Code != http.StatusBadRequest {
		t.Fatalf("empty SQL test status=%d body=%s", invalidSQL.Code, invalidSQL.Body.String())
	}
	draftSQL := httptest.NewRecorder()
	server.ServeHTTP(draftSQL, request("/v1/studio/sdk/connectors.test_sql", `{"name":"probe_http","input":{"sql":"SELECT 1"}}`))
	if draftSQL.Code != http.StatusBadRequest {
		t.Fatalf("draft connector SQL test status=%d body=%s", draftSQL.Code, draftSQL.Body.String())
	}
	bobSQLRequest := request("/v1/studio/sdk/connectors.test_sql", `{"name":"main","input":{"sql":"SELECT 1"}}`)
	bobSQLRequest.Header.Set("Authorization", "Bearer "+bobToken)
	bobSQL := httptest.NewRecorder()
	server.ServeHTTP(bobSQL, bobSQLRequest)
	if bobSQL.Code != http.StatusForbidden {
		t.Fatalf("non-editor SQL test status=%d body=%s", bobSQL.Code, bobSQL.Body.String())
	}
	predicateTypes := httptest.NewRecorder()
	server.ServeHTTP(predicateTypes, request("/v1/studio/sdk/authorization_predicates.types", `{}`))
	if predicateTypes.Code != http.StatusOK || !strings.Contains(predicateTypes.Body.String(), `"typeName":"ReportRead"`) {
		t.Fatalf("publisher predicate types status=%d body=%s", predicateTypes.Code, predicateTypes.Body.String())
	}
	deniedPredicateTypes := request("/v1/studio/sdk/authorization_predicates.types", `{}`)
	deniedPredicateTypes.Header.Set("Authorization", "Bearer "+bobToken)
	deniedPredicateTypesResponse := httptest.NewRecorder()
	server.ServeHTTP(deniedPredicateTypesResponse, deniedPredicateTypes)
	if deniedPredicateTypesResponse.Code != http.StatusForbidden || strings.Contains(deniedPredicateTypesResponse.Body.String(), "ReportRead") {
		t.Fatalf("non-publisher predicate types status=%d body=%s", deniedPredicateTypesResponse.Code, deniedPredicateTypesResponse.Body.String())
	}
	unauthenticatedPredicateTypes := httptest.NewRecorder()
	server.ServeHTTP(unauthenticatedPredicateTypes, httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/authorization_predicates.types", strings.NewReader(`{}`)))
	if unauthenticatedPredicateTypes.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated predicate types status=%d body=%s", unauthenticatedPredicateTypes.Code, unauthenticatedPredicateTypes.Body.String())
	}
	for _, row := range []struct{ name, title, packagePath, typeName, status, scope string }{
		{"studio.alpha.read", "Alpha access", "github.com/viant/datly-studio/studio/authorization", "ReportRead", "active", `{"alias":"r","columns":["tenant_id"]}`},
		{"studio.beta.read", "Beta access", "example.com/missing", "NotLinked", "disabled", ""},
	} {
		if _, err = store.ExecContext(ctx, `INSERT INTO authorization_predicates(name,title,package_path,type_name,sql_scope_json,owner_id,status,etag,created_at,updated_at)
			VALUES(?,?,?,?,?,'alice',?,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, row.name, row.title, row.packagePath, row.typeName, row.scope, row.status); err != nil {
			t.Fatal(err)
		}
	}
	predicateGet := httptest.NewRecorder()
	server.ServeHTTP(predicateGet, request("/v1/studio/sdk/authorization_predicates.get", `{"name":"studio.alpha.read"}`))
	if predicateGet.Code != http.StatusOK || !strings.Contains(predicateGet.Body.String(), `"name":"studio.alpha.read"`) ||
		!strings.Contains(predicateGet.Body.String(), `"linked":true`) || !strings.Contains(predicateGet.Body.String(), `"alias":"r"`) ||
		!strings.Contains(predicateGet.Body.String(), `"columns":["tenant_id"]`) {
		t.Fatalf("publisher predicate get status=%d body=%s", predicateGet.Code, predicateGet.Body.String())
	}
	missingPredicate := httptest.NewRecorder()
	server.ServeHTTP(missingPredicate, request("/v1/studio/sdk/authorization_predicates.get", `{"name":"studio.missing.read"}`))
	if missingPredicate.Code != http.StatusNotFound {
		t.Fatalf("missing predicate get status=%d body=%s", missingPredicate.Code, missingPredicate.Body.String())
	}
	predicateList := httptest.NewRecorder()
	server.ServeHTTP(predicateList, request("/v1/studio/sdk/authorization_predicates.list", `{"query":"ACCESS","status":"active","limit":1,"offset":0}`))
	if predicateList.Code != http.StatusOK || !strings.Contains(predicateList.Body.String(), `"name":"studio.alpha.read"`) ||
		strings.Contains(predicateList.Body.String(), `"name":"studio.beta.read"`) || !strings.Contains(predicateList.Body.String(), `"limit":1`) {
		t.Fatalf("filtered predicate list status=%d body=%s", predicateList.Code, predicateList.Body.String())
	}
	createdPredicate := httptest.NewRecorder()
	server.ServeHTTP(createdPredicate, request("/v1/studio/sdk/authorization_predicates.create", `{"name":"studio.gamma.read","title":"Gamma access","packagePath":"github.com/viant/datly-studio/studio/authorization","typeName":"ReportEdit","alias":"g","columns":["tenant_id"]}`))
	if createdPredicate.Code != http.StatusOK || !strings.Contains(createdPredicate.Body.String(), `"name":"studio.gamma.read"`) ||
		!strings.Contains(createdPredicate.Body.String(), `"linked":true`) || !strings.Contains(createdPredicate.Body.String(), `"ownerId":"alice"`) {
		t.Fatalf("native predicate create status=%d body=%s", createdPredicate.Code, createdPredicate.Body.String())
	}
	var createdPredicateOwner, createdPredicateScope string
	if err = store.QueryRowContext(ctx, `SELECT owner_id,sql_scope_json FROM authorization_predicates WHERE name='studio.gamma.read'`).Scan(&createdPredicateOwner, &createdPredicateScope); err != nil || createdPredicateOwner != "alice" || createdPredicateScope != `{"alias":"g","columns":["tenant_id"]}` {
		t.Fatalf("native predicate persistence owner=%q scope=%q err=%v", createdPredicateOwner, createdPredicateScope, err)
	}
	duplicatePredicate := httptest.NewRecorder()
	server.ServeHTTP(duplicatePredicate, request("/v1/studio/sdk/authorization_predicates.create", `{"name":"studio.gamma.read","title":"Duplicate","packagePath":"github.com/viant/datly-studio/studio/authorization","typeName":"ReportEdit"}`))
	if duplicatePredicate.Code != http.StatusConflict {
		t.Fatalf("duplicate predicate create status=%d body=%s", duplicatePredicate.Code, duplicatePredicate.Body.String())
	}
	unlinkedPredicate := httptest.NewRecorder()
	server.ServeHTTP(unlinkedPredicate, request("/v1/studio/sdk/authorization_predicates.create", `{"name":"studio.unknown.read","title":"Unknown","packagePath":"example.com/missing","typeName":"NotLinked"}`))
	if unlinkedPredicate.Code != http.StatusBadRequest {
		t.Fatalf("unlinked predicate create status=%d body=%s", unlinkedPredicate.Code, unlinkedPredicate.Body.String())
	}
	updatedPredicate := httptest.NewRecorder()
	server.ServeHTTP(updatedPredicate, request("/v1/studio/sdk/authorization_predicates.update", `{"name":"studio.gamma.read","input":{"title":"Gamma revised","alias":"","columns":[],"etag":1}}`))
	if updatedPredicate.Code != http.StatusOK || !strings.Contains(updatedPredicate.Body.String(), `"title":"Gamma revised"`) ||
		!strings.Contains(updatedPredicate.Body.String(), `"etag":2`) || strings.Contains(updatedPredicate.Body.String(), `"alias":"g"`) {
		t.Fatalf("native predicate update status=%d body=%s", updatedPredicate.Code, updatedPredicate.Body.String())
	}
	stalePredicateUpdate := httptest.NewRecorder()
	server.ServeHTTP(stalePredicateUpdate, request("/v1/studio/sdk/authorization_predicates.update", `{"name":"studio.gamma.read","input":{"title":"Stale","etag":1}}`))
	if stalePredicateUpdate.Code != http.StatusConflict {
		t.Fatalf("stale predicate update status=%d body=%s", stalePredicateUpdate.Code, stalePredicateUpdate.Body.String())
	}
	invalidPredicateUpdate := httptest.NewRecorder()
	server.ServeHTTP(invalidPredicateUpdate, request("/v1/studio/sdk/authorization_predicates.update", `{"name":"studio.gamma.read","input":{"title":"","etag":2}}`))
	if invalidPredicateUpdate.Code != http.StatusBadRequest {
		t.Fatalf("invalid predicate update status=%d body=%s", invalidPredicateUpdate.Code, invalidPredicateUpdate.Body.String())
	}
	stalePredicateDelete := httptest.NewRecorder()
	server.ServeHTTP(stalePredicateDelete, request("/v1/studio/sdk/authorization_predicates.delete", `{"name":"studio.gamma.read","etag":1}`))
	if stalePredicateDelete.Code != http.StatusConflict {
		t.Fatalf("stale predicate delete status=%d body=%s", stalePredicateDelete.Code, stalePredicateDelete.Body.String())
	}
	deletedPredicate := httptest.NewRecorder()
	server.ServeHTTP(deletedPredicate, request("/v1/studio/sdk/authorization_predicates.delete", `{"name":"studio.gamma.read","etag":2}`))
	if deletedPredicate.Code != http.StatusNoContent {
		t.Fatalf("native predicate delete status=%d body=%s", deletedPredicate.Code, deletedPredicate.Body.String())
	}
	var deletedPredicateStatus string
	var deletedPredicateETag int
	var deletedPredicateAt sql.NullTime
	if err = store.QueryRowContext(ctx, `SELECT status,etag,deleted_at FROM authorization_predicates WHERE name='studio.gamma.read'`).Scan(&deletedPredicateStatus, &deletedPredicateETag, &deletedPredicateAt); err != nil || deletedPredicateStatus != "disabled" || deletedPredicateETag != 3 || !deletedPredicateAt.Valid {
		t.Fatalf("predicate soft delete status=%q etag=%d deleted=%v err=%v", deletedPredicateStatus, deletedPredicateETag, deletedPredicateAt, err)
	}
	_ = extension.LinkedType
	const extensionPath = "github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension"
	t.Setenv("STUDIO_PREDICATE_PACKAGES", extensionPath)
	extendedTypes := httptest.NewRecorder()
	server.ServeHTTP(extendedTypes, request("/v1/studio/sdk/authorization_predicates.types", `{}`))
	if extendedTypes.Code != http.StatusOK || !strings.Contains(extendedTypes.Body.String(), `"typeName":"ReaderScope"`) || strings.Contains(extendedTypes.Body.String(), `"typeName":"NotAPredicate"`) {
		t.Fatalf("linked extension types status=%d body=%s", extendedTypes.Code, extendedTypes.Body.String())
	}
	extendedCreate := httptest.NewRecorder()
	server.ServeHTTP(extendedCreate, request("/v1/studio/sdk/authorization_predicates.create", fmt.Sprintf(`{"name":"studio.extension.read","title":"Extension access","packagePath":%q,"typeName":"ReaderScope"}`, extensionPath)))
	if extendedCreate.Code != http.StatusOK || !strings.Contains(extendedCreate.Body.String(), `"linked":true`) {
		t.Fatalf("linked extension create status=%d body=%s", extendedCreate.Code, extendedCreate.Body.String())
	}
	t.Setenv("STUDIO_PREDICATE_PACKAGES", "")
	extendedGet := httptest.NewRecorder()
	server.ServeHTTP(extendedGet, request("/v1/studio/sdk/authorization_predicates.get", `{"name":"studio.extension.read"}`))
	if extendedGet.Code != http.StatusOK || !strings.Contains(extendedGet.Body.String(), `"linked":false`) {
		t.Fatalf("removed extension package remains linked status=%d body=%s", extendedGet.Code, extendedGet.Body.String())
	}
	if _, err = store.ExecContext(ctx, `DELETE FROM authorization_predicates WHERE name='studio.extension.read'`); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"get", "list", "create", "update", "delete"} {
		body := `{"name":"studio.alpha.read"}`
		if operation == "create" {
			body = `{"name":"studio.denied.read","title":"Denied","packagePath":"github.com/viant/datly-studio/studio/authorization","typeName":"ReportRead"}`
		} else if operation == "update" {
			body = `{"name":"studio.alpha.read","input":{"title":"Denied","etag":1}}`
		} else if operation == "delete" {
			body = `{"name":"studio.alpha.read","etag":1}`
		}
		deniedRequest := request("/v1/studio/sdk/authorization_predicates."+operation, body)
		deniedRequest.Header.Set("Authorization", "Bearer "+bobToken)
		deniedResponse := httptest.NewRecorder()
		server.ServeHTTP(deniedResponse, deniedRequest)
		if deniedResponse.Code != http.StatusForbidden || strings.Contains(deniedResponse.Body.String(), "Alpha access") {
			t.Fatalf("non-publisher predicate %s status=%d body=%s", operation, deniedResponse.Code, deniedResponse.Body.String())
		}
	}
	var nativeReport struct {
		Title string `json:"title"`
		ID    string `json:"id"`
	}
	if err = json.Unmarshal(reportCreated.Body.Bytes(), &nativeReport); err != nil || nativeReport.ID == "" {
		t.Fatalf("native report identity=%+v err=%v", nativeReport, err)
	}
	const previewReportID = "preview-fixture"
	if _, err = store.ExecContext(ctx, `INSERT INTO components(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
		VALUES(?, 'production.audit', 'preview-fixture', 'Preview Fixture', 'alice', 'draft', 'main', 'github.com/viant/datly-studio/dynamic/preview_fixture', 'reader', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, previewReportID); err != nil {
		t.Fatal(err)
	}
	previewDQL := `#package('github.com/viant/datly-studio/dynamic/preview_fixture')
#setting($_ = $connector('main'))
#setting($_ = $route('/v1/studio/readers/preview-fixture','GET'))
#define($_ = $Rows<[]*Row>(output/view))
SELECT rows.*, type(rows,'Row') FROM (SELECT id,slug FROM components WHERE id='preview-fixture') rows`
	if _, err = store.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
		VALUES(?,1,'draft','dql',?,?,'{}','studio.v1','preview-hash','{}','valid','v1','studio.v1',1,'alice',CURRENT_TIMESTAMP)`, previewReportID, previewDQL, previewDQL); err != nil {
		t.Fatal(err)
	}
	previewCheck, previewErr := (studiopreview.Dynamic{StudioDB: store, ModulePath: "github.com/viant/datly-studio"}).Execute(
		sdk.WithPrincipal(ctx, sdk.Principal{Subject: "alice"}), previewReportID, 1, sdk.PreviewInput{Limit: 10})
	if previewErr != nil || previewCheck == nil {
		t.Fatalf("exact-version preview engine failed: %v", previewErr)
	}
	validatedVersion := httptest.NewRecorder()
	selectedValidationRequest := request("/v1/studio/sdk/versions.validate", `{"reportId":"preview-fixture","versionNo":1,"expectedSourceRevision":1}`)
	selectedValidationRequest.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	server.ServeHTTP(validatedVersion, selectedValidationRequest)
	if validatedVersion.Code != http.StatusOK || !strings.Contains(validatedVersion.Body.String(), `"valid":true`) ||
		!strings.Contains(validatedVersion.Body.String(), `"compileStatus":"valid"`) {
		t.Fatalf("native version validation status=%d body=%s", validatedVersion.Code, validatedVersion.Body.String())
	}
	var validatedStatus string
	var validatedAt sql.NullTime
	if err = store.QueryRowContext(ctx, `SELECT compile_status,validated_at FROM component_versions WHERE report_id=? AND version_no=1`, previewReportID).Scan(&validatedStatus, &validatedAt); err != nil || validatedStatus != "valid" || !validatedAt.Valid {
		t.Fatalf("native version validation persistence status=%q at=%v err=%v", validatedStatus, validatedAt, err)
	}
	staleValidation := httptest.NewRecorder()
	server.ServeHTTP(staleValidation, request("/v1/studio/sdk/versions.validate", `{"reportId":"preview-fixture","versionNo":1,"expectedSourceRevision":2}`))
	if staleValidation.Code != http.StatusConflict {
		t.Fatalf("stale version validation status=%d body=%s", staleValidation.Code, staleValidation.Body.String())
	}
	warmupHTTP := httptest.NewRecorder()
	server.ServeHTTP(warmupHTTP, request("/v1/studio/sdk/versions.warmup", `{"reportId":"preview-fixture","versionNo":1}`))
	var acceptedWarmup sdk.WarmupRun
	if warmupHTTP.Code != http.StatusOK || json.Unmarshal(warmupHTTP.Body.Bytes(), &acceptedWarmup) != nil ||
		acceptedWarmup.RunID == "" || acceptedWarmup.ReportID != previewReportID || acceptedWarmup.Status != "accepted" || acceptedWarmup.RequestedBy != "alice" {
		var recordedStatus string
		var recordedUpdated sql.NullTime
		recordedErr := store.QueryRowContext(ctx, `SELECT status,updated_at FROM component_warmup_runs WHERE report_id=? ORDER BY requested_at DESC LIMIT 1`, previewReportID).Scan(&recordedStatus, &recordedUpdated)
		t.Fatalf("native warmup acceptance status=%d run=%+v recorded=%q updated=%v err=%v body=%s", warmupHTTP.Code, acceptedWarmup, recordedStatus, recordedUpdated, recordedErr, warmupHTTP.Body.String())
	}
	var warmupNamespace string
	if err = store.QueryRowContext(ctx, "SELECT namespace_id FROM component_warmup_runs WHERE run_id=?", acceptedWarmup.RunID).Scan(&warmupNamespace); err != nil || warmupNamespace != namespaceaccess.ID("alice", "production.audit") {
		t.Fatalf("native warmup namespace=%q err=%v", warmupNamespace, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	var terminalWarmupStatus string
	for time.Now().Before(deadline) {
		if err = store.QueryRowContext(ctx, `SELECT status FROM component_warmup_runs WHERE run_id=?`, acceptedWarmup.RunID).Scan(&terminalWarmupStatus); err != nil {
			t.Fatal(err)
		}
		if terminalWarmupStatus == "failed" || terminalWarmupStatus == "completed" || terminalWarmupStatus == "partial" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if terminalWarmupStatus != "failed" {
		t.Fatalf("uncached warmup terminal status=%q, want failed", terminalWarmupStatus)
	}
	cachedDQL := fmt.Sprintf(`#package('github.com/viant/datly-studio/dynamic/preview_fixture')
#setting($_ = $connector('main'))
#setting($_ = $route('/v1/studio/readers/preview-fixture-cache','GET'))
#setting($_ = $cache(true, '1m').WithLocation('%s'))
#setting($_ = $cache_warmup(''))
#define($_ = $Rows<[]*Row>(output/view))
SELECT rows.*, type(rows,'Row') FROM (SELECT id,slug FROM components WHERE id='preview-fixture') rows`, filepath.ToSlash(filepath.Join(t.TempDir(), "warmup-cache")))
	if _, err = store.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
		VALUES(?,3,'draft','dql',?,?,'{}','studio.v1','cache-hash','{}','valid','v1','studio.v1',1,'alice',CURRENT_TIMESTAMP)`, previewReportID, cachedDQL, cachedDQL); err != nil {
		t.Fatal(err)
	}
	cachedWarmup := httptest.NewRecorder()
	server.ServeHTTP(cachedWarmup, request("/v1/studio/sdk/versions.warmup", `{"reportId":"preview-fixture","versionNo":3}`))
	var acceptedCached sdk.WarmupRun
	if cachedWarmup.Code != http.StatusOK || json.Unmarshal(cachedWarmup.Body.Bytes(), &acceptedCached) != nil || acceptedCached.Status != "accepted" {
		t.Fatalf("cached warmup acceptance status=%d body=%s", cachedWarmup.Code, cachedWarmup.Body.String())
	}
	deadline = time.Now().Add(5 * time.Second)
	var completedCached sdk.WarmupRun
	for time.Now().Before(deadline) {
		if err = store.QueryRowContext(ctx, `SELECT status,planned_cases,completed_cases,entries FROM component_warmup_runs WHERE run_id=?`, acceptedCached.RunID).
			Scan(&completedCached.Status, &completedCached.PlannedCases, &completedCached.CompletedCases, &completedCached.Entries); err != nil {
			t.Fatal(err)
		}
		if completedCached.Status == "completed" || completedCached.Status == "failed" || completedCached.Status == "partial" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if completedCached.Status != "completed" || completedCached.PlannedCases < 1 || completedCached.CompletedCases != completedCached.PlannedCases {
		t.Fatalf("cached warmup terminal result=%+v", completedCached)
	}
	// The caller owns this namespace too; ownership must not bypass selection.
	wrongNamespaceID := namespaceaccess.ID("alice", "namespace_guard_other")
	if _, err = store.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,status,etag,created_at,updated_at) VALUES(?,'alice','namespace_guard_other','Other','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, wrongNamespaceID); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"preview.execute", "versions.test_view"} {
		payload := `{"reportId":"preview-fixture","versionNo":1,"view":"rows","input":{"limit":10}}`
		blocked := request("/v1/studio/sdk/"+operation, payload)
		blocked.Header.Set("X-Studio-Namespace", wrongNamespaceID)
		blockedResponse := httptest.NewRecorder()
		server.ServeHTTP(blockedResponse, blocked)
		if blockedResponse.Code < 400 || blockedResponse.Code >= 500 || strings.Contains(blockedResponse.Body.String(), `"returnedRows":1`) {
			t.Fatalf("cross-namespace run %s status=%d body=%s", operation, blockedResponse.Code, blockedResponse.Body.String())
		}
	}

	var beforeWarmupCount int
	if err = store.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_warmup_runs WHERE report_id=?", previewReportID).Scan(&beforeWarmupCount); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct{ operation, payload string }{
		{"versions.validate", `{"reportId":"preview-fixture","versionNo":1,"expectedSourceRevision":1}`},
		{"versions.warmup", `{"reportId":"preview-fixture","versionNo":1}`},
		{"versions.warmup_list", `{"reportId":"preview-fixture","versionNo":1,"input":{"limit":10}}`},
		{"versions.warmup_get", `{"reportId":"preview-fixture","runId":"` + acceptedWarmup.RunID + `"}`},
	} {
		blocked := request("/v1/studio/sdk/"+check.operation, check.payload)
		blocked.Header.Set("X-Studio-Namespace", wrongNamespaceID)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, blocked)
		if response.Code < 400 || response.Code >= 500 {
			t.Fatalf("cross-namespace %s status=%d body=%s", check.operation, response.Code, response.Body.String())
		}
	}
	var afterWarmupCount int
	var unchangedValidation sql.NullTime
	if err = store.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_warmup_runs WHERE report_id=?", previewReportID).Scan(&afterWarmupCount); err != nil || afterWarmupCount != beforeWarmupCount {
		t.Fatalf("denial created warmup runs: before=%d after=%d err=%v", beforeWarmupCount, afterWarmupCount, err)
	}
	if err = store.QueryRowContext(ctx, "SELECT validated_at FROM component_versions WHERE report_id=? AND version_no=1", previewReportID).Scan(&unchangedValidation); err != nil || !unchangedValidation.Time.Equal(validatedAt.Time) {
		t.Fatalf("denial changed validation evidence: %v", err)
	}
	previewHTTP := httptest.NewRecorder()
	previewHTTPRequest := request("/v1/studio/sdk/preview.execute", `{"reportId":"preview-fixture","versionNo":1,"input":{"limit":10}}`)
	previewHTTPRequest.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	server.ServeHTTP(previewHTTP, previewHTTPRequest)
	if previewHTTP.Code != http.StatusOK || !strings.Contains(previewHTTP.Body.String(), "preview-fixture") ||
		!strings.Contains(previewHTTP.Body.String(), `"returnedRows":1`) || strings.Contains(previewHTTP.Body.String(), dsn) {
		t.Fatalf("native exact-version preview status=%d body=%s", previewHTTP.Code, previewHTTP.Body.String())
	}
	viewHTTP := httptest.NewRecorder()
	viewHTTPRequest := request("/v1/studio/sdk/versions.test_view", `{"reportId":"preview-fixture","versionNo":1,"view":"rows","input":{"limit":10}}`)
	viewHTTPRequest.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	server.ServeHTTP(viewHTTP, viewHTTPRequest)
	if viewHTTP.Code != http.StatusOK || !strings.Contains(viewHTTP.Body.String(), `"view":"rows"`) ||
		!strings.Contains(viewHTTP.Body.String(), "preview-fixture") || strings.Contains(viewHTTP.Body.String(), dsn) {
		t.Fatalf("native root view test status=%d body=%s", viewHTTP.Code, viewHTTP.Body.String())
	}
	bobPreviewRequest := request("/v1/studio/sdk/preview.execute", `{"reportId":"preview-fixture","versionNo":1,"input":{"limit":10}}`)
	bobPreviewRequest.Header.Set("Authorization", "Bearer "+bobToken)
	bobPreview := httptest.NewRecorder()
	server.ServeHTTP(bobPreview, bobPreviewRequest)
	if bobPreview.Code != http.StatusForbidden {
		t.Fatalf("non-runner preview status=%d body=%s", bobPreview.Code, bobPreview.Body.String())
	}
	if _, err = store.ExecContext(ctx, `INSERT INTO component_acl(report_id,subject_type,subject_id,can_view,can_run,etag)
		VALUES(?,'user','bob',1,1,1)`, previewReportID); err != nil {
		t.Fatal(err)
	}
	bobPreviewRequest = request("/v1/studio/sdk/preview.execute", `{"reportId":"preview-fixture","versionNo":1,"input":{"limit":10}}`)
	bobPreviewRequest.Header.Set("Authorization", "Bearer "+bobToken)
	bobPreview = httptest.NewRecorder()
	server.ServeHTTP(bobPreview, bobPreviewRequest)
	if bobPreview.Code != http.StatusOK || !strings.Contains(bobPreview.Body.String(), "preview-fixture") {
		t.Fatalf("delegated runner preview status=%d body=%s", bobPreview.Code, bobPreview.Body.String())
	}

	// Run permission is separate from metadata visibility within a visible namespace.
	if _, err = store.ExecContext(ctx, `UPDATE component_acl SET can_view=0 WHERE report_id=? AND subject_id='bob'`, previewReportID); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ExecContext(ctx, `UPDATE namespaces SET visibility='public' WHERE owner_id='alice' AND name='production.audit'`); err != nil {
		t.Fatal(err)
	}
	runOnlyRequest := request("/v1/studio/sdk/preview.execute", `{"reportId":"preview-fixture","versionNo":1,"input":{"limit":10}}`)
	runOnlyRequest.Header.Set("Authorization", "Bearer "+bobToken)
	runOnlyRequest.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	runOnlyResponse := httptest.NewRecorder()
	server.ServeHTTP(runOnlyResponse, runOnlyRequest)
	if runOnlyResponse.Code != http.StatusOK || !strings.Contains(runOnlyResponse.Body.String(), `"returnedRows":1`) {
		t.Fatalf("visible namespace run-only access status=%d body=%s", runOnlyResponse.Code, runOnlyResponse.Body.String())
	}
	if _, err = store.ExecContext(ctx, `UPDATE component_acl SET can_view=1 WHERE report_id=? AND subject_id='bob'`, previewReportID); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ExecContext(ctx, `UPDATE namespaces SET visibility='private' WHERE owner_id='alice' AND name='production.audit'`); err != nil {
		t.Fatal(err)
	}
	bobValidation := request("/v1/studio/sdk/versions.validate", `{"reportId":"preview-fixture","versionNo":1,"expectedSourceRevision":1}`)
	bobValidation.Header.Set("Authorization", "Bearer "+bobToken)
	bobValidationResponse := httptest.NewRecorder()
	server.ServeHTTP(bobValidationResponse, bobValidation)
	if bobValidationResponse.Code != http.StatusForbidden {
		t.Fatalf("run-only version validation status=%d body=%s", bobValidationResponse.Code, bobValidationResponse.Body.String())
	}
	bobWarmup := request("/v1/studio/sdk/versions.warmup", `{"reportId":"preview-fixture","versionNo":1}`)
	bobWarmup.Header.Set("Authorization", "Bearer "+bobToken)
	bobWarmupResponse := httptest.NewRecorder()
	server.ServeHTTP(bobWarmupResponse, bobWarmup)
	if bobWarmupResponse.Code != http.StatusNotFound {
		t.Fatalf("run-only warmup status=%d body=%s", bobWarmupResponse.Code, bobWarmupResponse.Body.String())
	}
	editorToken, signErr := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"sub": "editor", "user_id": 3, "iss": "https://idp.viantinc.com", "aud": "datly-studio-web", "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(key)
	if signErr != nil {
		t.Fatal(signErr)
	}
	if _, err = store.ExecContext(ctx, `INSERT INTO component_acl(report_id,subject_type,subject_id,can_view,can_edit,can_use_dql,etag)
		VALUES(?,'user','editor',1,1,0,1)`, previewReportID); err != nil {
		t.Fatal(err)
	}
	invalidDQL := `#package('github.com/viant/datly-studio/dynamic/preview_fixture')
#setting($_ = $connector('main'))
#setting($_ = $route('/v1/studio/readers/preview-fixture','GET'))
#define($_ = $Rows<[]*Row>(output/view))
SELECT rows.*, type(rows,'Row') FROM (SELECT FROM components) rows`
	if _, err = store.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
		VALUES(?,2,'draft','dql',?,?,'{}','studio.v1','invalid-hash','{}','pending','v1','studio.v1',1,'alice',CURRENT_TIMESTAMP)`, previewReportID, invalidDQL, invalidDQL); err != nil {
		t.Fatal(err)
	}
	delegatedValidationRequest := request("/v1/studio/sdk/versions.validate", `{"reportId":"preview-fixture","versionNo":2,"expectedSourceRevision":1}`)
	delegatedValidationRequest.Header.Set("Authorization", "Bearer "+editorToken)
	delegatedValidation := httptest.NewRecorder()
	server.ServeHTTP(delegatedValidation, delegatedValidationRequest)
	if delegatedValidation.Code != http.StatusOK || !strings.Contains(delegatedValidation.Body.String(), `"valid":false`) ||
		!strings.Contains(delegatedValidation.Body.String(), `"compileStatus":"invalid"`) ||
		strings.Contains(delegatedValidation.Body.String(), "SELECT FROM") || strings.Contains(delegatedValidation.Body.String(), `"authoredDql"`) ||
		strings.Contains(delegatedValidation.Body.String(), `"generatedDql"`) {
		t.Fatalf("delegated redacted validation status=%d body=%s", delegatedValidation.Code, delegatedValidation.Body.String())
	}
	dedupePlan := warmupprojection.PlanKey(&sdk.ReportVersion{ReportID: previewReportID, VersionNo: 2, SourceRevision: 1, SpecHash: "invalid-hash"})
	dedupeKey := fmt.Sprintf("%s:%d:%s", previewReportID, 2, dedupePlan)
	if _, err = store.ExecContext(ctx, `INSERT INTO component_warmup_runs(run_id,report_id,version_no,source_revision,spec_hash,plan_key,active_key,status,requested_by,target_json,requested_at,created_at,created_by,updated_at,updated_by)
		VALUES('w-dedupe',?,2,1,'invalid-hash',?,?,'accepted','alice','{}',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'alice',CURRENT_TIMESTAMP,'alice')`, previewReportID, dedupePlan, dedupeKey); err != nil {
		t.Fatal(err)
	}
	deduplicatedWarmup := httptest.NewRecorder()
	server.ServeHTTP(deduplicatedWarmup, request("/v1/studio/sdk/versions.warmup", `{"reportId":"preview-fixture","versionNo":2}`))
	if deduplicatedWarmup.Code != http.StatusOK || !strings.Contains(deduplicatedWarmup.Body.String(), `"runId":"w-dedupe"`) ||
		!strings.Contains(deduplicatedWarmup.Body.String(), `"status":"accepted"`) {
		t.Fatalf("native warmup active-key reuse status=%d body=%s", deduplicatedWarmup.Code, deduplicatedWarmup.Body.String())
	}
	if _, err = store.ExecContext(ctx, `INSERT INTO components(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
		VALUES('relation-fixture','production.audit','relation-fixture','Relation Fixture','alice','draft','main','github.com/viant/datly-studio/dynamic/relation_fixture','reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	relationDQL := `#package('github.com/viant/datly-studio/dynamic/relation_fixture')
#setting($_ = $connector('main'))
#setting($_ = $route('/v1/studio/readers/relation-fixture','GET'))
#define($_ = $Rows<[]*ReportRow>(output/view))
SELECT reports.*, versions.*, type(reports,'ReportRow'), type(versions,'VersionRow')
FROM (SELECT id,slug FROM components WHERE id='preview-fixture') reports
LEFT JOIN (SELECT report_id,version_no FROM component_versions WHERE report_id='preview-fixture' AND version_no<=2) versions
ON versions.report_id=reports.id`
	if _, err = store.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
		VALUES('relation-fixture',1,'draft','dql',?,?,'{}','studio.v1','relation-hash','{}','valid','v1','studio.v1',1,'alice',CURRENT_TIMESTAMP)`, relationDQL, relationDQL); err != nil {
		t.Fatal(err)
	}
	relationHTTP := httptest.NewRecorder()
	server.ServeHTTP(relationHTTP, request("/v1/studio/sdk/versions.test_relation", `{"reportId":"relation-fixture","versionNo":1,"relation":"versions","input":{"limit":10}}`))
	if relationHTTP.Code != http.StatusOK || !strings.Contains(relationHTTP.Body.String(), `"parentRows":1`) ||
		!strings.Contains(relationHTTP.Body.String(), `"attachedChildren":2`) || strings.Contains(relationHTTP.Body.String(), dsn) {
		t.Fatalf("native relation test status=%d body=%s", relationHTTP.Code, relationHTTP.Body.String())
	}
	if _, err = store.ExecContext(ctx, `INSERT INTO components(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
		VALUES('compose-fixture','production.audit','compose-fixture','Compose Fixture','alice','draft','main','github.com/viant/datly-studio/dynamic/compose_fixture','reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	composeDQL := `#package('github.com/viant/datly-studio/dynamic/compose_fixture')
#setting($_ = $connector('main'))
#setting($_ = $route('/v1/studio/readers/compose-fixture','GET'))
#setting($_ = $cube())
#setting($_ = $cubeCompose(true, false, 4, 20, 5000))
#define($_ = $Rows<[]*Summary>(output/view))
SELECT summary.*, groupable(summary), tag(summary.status, 'groupable:"true"'),
       CAST(summary.product_count AS float64), type(summary,'Summary')
FROM (SELECT status,COUNT(*) AS product_count FROM components GROUP BY status) summary`
	if _, err = store.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
		VALUES('compose-fixture',1,'draft','dql',?,?,'{}','studio.v1','compose-hash','{}','valid','v1','studio.v1',1,'alice',CURRENT_TIMESTAMP)`, composeDQL, composeDQL); err != nil {
		t.Fatal(err)
	}
	composeHTTP := httptest.NewRecorder()
	server.ServeHTTP(composeHTTP, request("/v1/studio/sdk/versions.test_compose", `{"reportId":"compose-fixture","versionNo":1,"input":{"cubes":[{"dimensions":{"status":true},"measures":{"productCount":true},"filters":{}}],"sql":"SELECT t1.status FROM $CubeSQL1 AS t1"}}`))
	if composeHTTP.Code != http.StatusOK || !strings.Contains(composeHTTP.Body.String(), `"data"`) || strings.Contains(composeHTTP.Body.String(), dsn) {
		t.Fatalf("native cube compose test status=%d body=%s", composeHTTP.Code, composeHTTP.Body.String())
	}
	missingPreview := httptest.NewRecorder()
	server.ServeHTTP(missingPreview, request("/v1/studio/sdk/preview.execute", `{"reportId":"preview-fixture","versionNo":99}`))
	if missingPreview.Code != http.StatusNotFound {
		t.Fatalf("missing version preview status=%d body=%s", missingPreview.Code, missingPreview.Body.String())
	}
	for _, subject := range []string{"grant_http", "grant_mcp", "grant_bff"} {
		if _, err = store.ExecContext(ctx, `INSERT INTO component_acl(report_id,subject_type,subject_id,can_view,etag)
			VALUES(?,'user',?,1,1)`, nativeReport.ID, subject); err != nil {
			t.Fatal(err)
		}
	}
	for _, check := range []struct{ operation, payload string }{
		{"acl.upsert", `{"reportId":"` + nativeReport.ID + `","subjectType":"user","subjectId":"ns_denied_http","canView":true}`},
		{"acl.delete", `{"reportId":"` + nativeReport.ID + `","subjectType":"user","subjectId":"grant_http","etag":1}`},
	} {
		blocked := request("/v1/studio/sdk/"+check.operation, check.payload)
		blocked.Header.Set("X-Studio-Namespace", wrongNamespaceID)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, blocked)
		if response.Code != http.StatusForbidden {
			t.Fatalf("cross-namespace %s status=%d body=%s", check.operation, response.Code, response.Body.String())
		}
	}
	deniedACLList := request("/v1/studio/sdk/acl.list", `{"reportId":"`+nativeReport.ID+`"}`)
	deniedACLList.Header.Set("X-Studio-Namespace", wrongNamespaceID)
	deniedACLResponse := httptest.NewRecorder()
	server.ServeHTTP(deniedACLResponse, deniedACLList)
	if deniedACLResponse.Code != http.StatusOK || strings.Contains(deniedACLResponse.Body.String(), "grant_http") {
		t.Fatalf("cross-namespace ACL list status=%d body=%s", deniedACLResponse.Code, deniedACLResponse.Body.String())
	}
	var untouchedACLCount int
	if err = store.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_acl WHERE report_id=? AND subject_id='grant_http' AND etag=1", nativeReport.ID).Scan(&untouchedACLCount); err != nil || untouchedACLCount != 1 {
		t.Fatalf("cross-namespace delete changed grant: count=%d err=%v", untouchedACLCount, err)
	}
	if err = store.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_acl WHERE report_id=? AND subject_id='ns_denied_http'", nativeReport.ID).Scan(&untouchedACLCount); err != nil || untouchedACLCount != 0 {
		t.Fatalf("cross-namespace upsert added grant: count=%d err=%v", untouchedACLCount, err)
	}
	invalidACLGrant := httptest.NewRecorder()
	server.ServeHTTP(invalidACLGrant, request("/v1/studio/sdk/acl.upsert", `{"reportId":"`+nativeReport.ID+`","subjectType":"user","subjectId":"invalid","canEdit":true}`))
	if invalidACLGrant.Code != http.StatusBadRequest {
		t.Fatalf("invalid ACL capabilities status=%d body=%s", invalidACLGrant.Code, invalidACLGrant.Body.String())
	}
	forbiddenACLUpsertRequest := request("/v1/studio/sdk/acl.upsert", `{"reportId":"`+nativeReport.ID+`","subjectType":"user","subjectId":"forged","canView":true}`)
	forbiddenACLUpsertRequest.Header.Set("Authorization", "Bearer "+bobToken)
	forbiddenACLUpsert := httptest.NewRecorder()
	server.ServeHTTP(forbiddenACLUpsert, forbiddenACLUpsertRequest)
	if forbiddenACLUpsert.Code != http.StatusForbidden {
		t.Fatalf("non-owner ACL upsert status=%d body=%s", forbiddenACLUpsert.Code, forbiddenACLUpsert.Body.String())
	}
	createdACL := httptest.NewRecorder()
	selectedACLCreate := request("/v1/studio/sdk/acl.upsert", `{"reportId":"`+nativeReport.ID+`","subjectType":"user","subjectId":"upsert_http","canView":true,"canEdit":true}`)
	selectedACLCreate.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	server.ServeHTTP(createdACL, selectedACLCreate)
	if createdACL.Code != http.StatusOK || !strings.Contains(createdACL.Body.String(), `"etag":1`) ||
		!strings.Contains(createdACL.Body.String(), `"canEdit":true`) {
		t.Fatalf("native ACL creation status=%d body=%s", createdACL.Code, createdACL.Body.String())
	}
	var nativeACLNamespace string
	if err = store.QueryRowContext(ctx, "SELECT namespace_id FROM component_acl WHERE report_id=? AND subject_id='upsert_http'", nativeReport.ID).Scan(&nativeACLNamespace); err != nil || nativeACLNamespace != namespaceaccess.ID("alice", "production.audit") {
		t.Fatalf("native ACL ownership=%q err=%v", nativeACLNamespace, err)
	}
	updatedACL := httptest.NewRecorder()
	server.ServeHTTP(updatedACL, request("/v1/studio/sdk/acl.upsert", `{"reportId":"`+nativeReport.ID+`","subjectType":"user","subjectId":"upsert_http","canView":true,"canRun":true,"canEdit":true,"etag":1}`))
	if updatedACL.Code != http.StatusOK || !strings.Contains(updatedACL.Body.String(), `"etag":2`) ||
		!strings.Contains(updatedACL.Body.String(), `"canRun":true`) {
		t.Fatalf("native ACL update status=%d body=%s", updatedACL.Code, updatedACL.Body.String())
	}
	staleACLUpsert := httptest.NewRecorder()
	server.ServeHTTP(staleACLUpsert, request("/v1/studio/sdk/acl.upsert", `{"reportId":"`+nativeReport.ID+`","subjectType":"user","subjectId":"upsert_http","canView":true,"etag":1}`))
	if staleACLUpsert.Code != http.StatusConflict || !strings.Contains(staleACLUpsert.Body.String(), `"currentEtag":2`) {
		t.Fatalf("stale ACL upsert status=%d body=%s", staleACLUpsert.Code, staleACLUpsert.Body.String())
	}
	deniedACLDeleteRequest := request("/v1/studio/sdk/acl.delete", `{"reportId":"`+nativeReport.ID+`","subjectType":"user","subjectId":"grant_http","etag":1}`)
	deniedACLDeleteRequest.Header.Set("Authorization", "Bearer "+bobToken)
	deniedACLDelete := httptest.NewRecorder()
	server.ServeHTTP(deniedACLDelete, deniedACLDeleteRequest)
	if deniedACLDelete.Code != http.StatusForbidden {
		t.Fatalf("non-owner ACL delete status=%d body=%s", deniedACLDelete.Code, deniedACLDelete.Body.String())
	}
	staleACLDelete := httptest.NewRecorder()
	server.ServeHTTP(staleACLDelete, request("/v1/studio/sdk/acl.delete", `{"reportId":"`+nativeReport.ID+`","subjectType":"user","subjectId":"grant_http","etag":2}`))
	if staleACLDelete.Code != http.StatusConflict || !strings.Contains(staleACLDelete.Body.String(), `"currentEtag":1`) {
		t.Fatalf("stale ACL delete status=%d body=%s", staleACLDelete.Code, staleACLDelete.Body.String())
	}
	deletedACL := httptest.NewRecorder()
	server.ServeHTTP(deletedACL, request("/v1/studio/sdk/acl.delete", `{"reportId":"`+nativeReport.ID+`","subjectType":"user","subjectId":"grant_http","etag":1}`))
	if deletedACL.Code != http.StatusNoContent {
		t.Fatalf("native ACL delete status=%d body=%s", deletedACL.Code, deletedACL.Body.String())
	}
	missingACLDelete := httptest.NewRecorder()
	server.ServeHTTP(missingACLDelete, request("/v1/studio/sdk/acl.delete", `{"reportId":"`+nativeReport.ID+`","subjectType":"user","subjectId":"grant_http","etag":1}`))
	if missingACLDelete.Code != http.StatusNotFound {
		t.Fatalf("missing ACL delete status=%d body=%s", missingACLDelete.Code, missingACLDelete.Body.String())
	}
	forgedReportUpdate := httptest.NewRecorder()
	server.ServeHTTP(forgedReportUpdate, request("/v1/studio/sdk/components.update", `{"id":"`+nativeReport.ID+`","input":{"componentScope":"attacker","etag":1}}`))
	if forgedReportUpdate.Code != http.StatusForbidden {
		t.Fatalf("forged report identity update status=%d body=%s", forgedReportUpdate.Code, forgedReportUpdate.Body.String())
	}
	bobReportUpdate := request("/v1/studio/sdk/components.update", `{"id":"`+nativeReport.ID+`","input":{"title":"Stolen","etag":1}}`)
	bobReportUpdate.Header.Set("Authorization", "Bearer "+bobToken)
	deniedReportUpdate := httptest.NewRecorder()
	server.ServeHTTP(deniedReportUpdate, bobReportUpdate)
	if deniedReportUpdate.Code != http.StatusForbidden {
		t.Fatalf("non-editor report update status=%d body=%s", deniedReportUpdate.Code, deniedReportUpdate.Body.String())
	}
	wrongUpdate := request("/v1/studio/sdk/components.update", `{"id":"`+nativeReport.ID+`","input":{"title":"Wrong workspace","etag":1}}`)
	wrongUpdate.Header.Set("X-Studio-Namespace", wrongNamespaceID)
	wrongUpdateResponse := httptest.NewRecorder()
	server.ServeHTTP(wrongUpdateResponse, wrongUpdate)
	if wrongUpdateResponse.Code != http.StatusForbidden {
		t.Fatalf("cross-namespace component update status=%d body=%s", wrongUpdateResponse.Code, wrongUpdateResponse.Body.String())
	}
	var unchangedTitle string
	var unchangedETag int
	if err = store.QueryRowContext(ctx, "SELECT title,etag FROM components WHERE id=?", nativeReport.ID).Scan(&unchangedTitle, &unchangedETag); err != nil || unchangedTitle != nativeReport.Title || unchangedETag != 1 {
		t.Fatalf("denial changed component: %q/%d err=%v", unchangedTitle, unchangedETag, err)
	}
	updatedReport := httptest.NewRecorder()
	selectedUpdate := request("/v1/studio/sdk/components.update", `{"id":"`+nativeReport.ID+`","input":{"title":"Native Updated","etag":1}}`)
	selectedUpdate.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	server.ServeHTTP(updatedReport, selectedUpdate)
	if updatedReport.Code != http.StatusOK || !strings.Contains(updatedReport.Body.String(), `"title":"Native Updated"`) ||
		!strings.Contains(updatedReport.Body.String(), `"etag":2`) {
		t.Fatalf("native report update status=%d body=%s", updatedReport.Code, updatedReport.Body.String())
	}
	moveNamespaceResponse := httptest.NewRecorder()
	server.ServeHTTP(moveNamespaceResponse, request("/v1/studio/sdk/components.update", `{"id":"`+nativeReport.ID+`","input":{"namespace":"namespace_guard_other","etag":2}}`))
	if moveNamespaceResponse.Code != http.StatusForbidden {
		t.Fatalf("namespace transfer through metadata status=%d body=%s", moveNamespaceResponse.Code, moveNamespaceResponse.Body.String())
	}
	var namespaceAfterMove string
	if err := store.QueryRowContext(ctx, `SELECT namespace_id FROM components WHERE id=?`, nativeReport.ID).Scan(&namespaceAfterMove); err != nil || namespaceAfterMove != namespaceaccess.ID("alice", "production.audit") {
		t.Fatalf("rejected namespace move changed ownership=%q err=%v", namespaceAfterMove, err)
	}
	staleReportUpdate := httptest.NewRecorder()
	server.ServeHTTP(staleReportUpdate, request("/v1/studio/sdk/components.update", `{"id":"`+nativeReport.ID+`","input":{"title":"Stale","etag":1}}`))
	if staleReportUpdate.Code != http.StatusConflict {
		t.Fatalf("stale report update status=%d body=%s", staleReportUpdate.Code, staleReportUpdate.Body.String())
	}
	invalidReportNamespace := httptest.NewRecorder()
	server.ServeHTTP(invalidReportNamespace, request("/v1/studio/sdk/components.update", `{"id":"`+nativeReport.ID+`","input":{"namespace":"Bad Space","etag":2}}`))
	if invalidReportNamespace.Code != http.StatusBadRequest {
		t.Fatalf("invalid report namespace status=%d body=%s", invalidReportNamespace.Code, invalidReportNamespace.Body.String())
	}
	forgedVersionActor := httptest.NewRecorder()
	server.ServeHTTP(forgedVersionActor, request("/v1/studio/sdk/versions.create", `{"reportId":"`+nativeReport.ID+`","input":{"authoringMode":"dql","authoredDql":"SELECT 1","createdBy":"bob"}}`))
	if forgedVersionActor.Code != http.StatusForbidden {
		t.Fatalf("forged version actor status=%d body=%s", forgedVersionActor.Code, forgedVersionActor.Body.String())
	}
	invalidVersionMode := httptest.NewRecorder()
	server.ServeHTTP(invalidVersionMode, request("/v1/studio/sdk/versions.create", `{"reportId":"`+nativeReport.ID+`","input":{"authoringMode":"invalid"}}`))
	if invalidVersionMode.Code != http.StatusBadRequest {
		t.Fatalf("invalid version mode status=%d body=%s", invalidVersionMode.Code, invalidVersionMode.Body.String())
	}
	createdVersion := httptest.NewRecorder()
	wrongVersionRequest := request("/v1/studio/sdk/versions.create", `{"reportId":"`+nativeReport.ID+`","input":{"authoringMode":"dql","authoredDql":"SELECT forbidden"}}`)
	wrongVersionRequest.Header.Set("X-Studio-Namespace", wrongNamespaceID)
	wrongVersionResponse := httptest.NewRecorder()
	server.ServeHTTP(wrongVersionResponse, wrongVersionRequest)
	if wrongVersionResponse.Code != http.StatusForbidden {
		t.Fatalf("cross-namespace version create status=%d body=%s", wrongVersionResponse.Code, wrongVersionResponse.Body.String())
	}
	server.ServeHTTP(createdVersion, request("/v1/studio/sdk/versions.create", `{"reportId":"`+nativeReport.ID+`","input":{"authoringMode":"dql","authoredDql":"SELECT 1","componentSpec":{}}}`))
	if createdVersion.Code != http.StatusOK || !strings.Contains(createdVersion.Body.String(), `"versionNo":1`) ||
		!strings.Contains(createdVersion.Body.String(), `"authoredDql":"SELECT 1"`) || !strings.Contains(createdVersion.Body.String(), `"sourceRevision":1`) {
		t.Fatalf("native version create status=%d body=%s", createdVersion.Code, createdVersion.Body.String())
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE report_id=? AND version_no=1 AND created_by='alice' AND compile_status='pending'`, nativeReport.ID).Scan(&reportCount); err != nil || reportCount != 1 {
		t.Fatalf("native version persistence count=%d err=%v", reportCount, err)
	}
	var createdVersionNamespace, createdVersionOwner, createdVersionWorkspace string
	if err := store.QueryRowContext(ctx, `SELECT v.namespace_id,c.owner_id,c.namespace FROM component_versions v JOIN components c ON c.id=v.report_id WHERE v.report_id=? AND v.version_no=1`, nativeReport.ID).Scan(&createdVersionNamespace, &createdVersionOwner, &createdVersionWorkspace); err != nil || createdVersionNamespace != namespaceaccess.ID(createdVersionOwner, createdVersionWorkspace) {
		t.Fatalf("native created version namespace=%q err=%v", createdVersionNamespace, err)
	}
	ownerInspection := httptest.NewRecorder()
	server.ServeHTTP(ownerInspection, request("/v1/studio/sdk/versions.inspect", `{"reportId":"`+nativeReport.ID+`","versionNo":1}`))
	if ownerInspection.Code != http.StatusOK || !strings.Contains(ownerInspection.Body.String(), `"dql":"SELECT 1"`) ||
		!strings.Contains(ownerInspection.Body.String(), `"canUseDql":true`) {
		t.Fatalf("owner version inspect status=%d body=%s", ownerInspection.Code, ownerInspection.Body.String())
	}
	structureOnly := httptest.NewRecorder()
	server.ServeHTTP(structureOnly, request("/v1/studio/sdk/versions.inspect", `{"reportId":"`+nativeReport.ID+`","versionNo":1,"discoverColumns":false}`))
	if structureOnly.Code != http.StatusOK || !strings.Contains(structureOnly.Body.String(), `"canUseDql":true`) {
		t.Fatalf("structure-only version inspect status=%d body=%s", structureOnly.Code, structureOnly.Body.String())
	}
	loadReportResponse := httptest.NewRecorder()
	wrongCreate := request("/v1/studio/sdk/components.create", `{"slug":"ns-create-denied","title":"Denied","namespace":"production.audit","defaultConnectorName":"main"}`)
	wrongCreate.Header.Set("X-Studio-Namespace", wrongNamespaceID)
	wrongCreateResult := httptest.NewRecorder()
	server.ServeHTTP(wrongCreateResult, wrongCreate)
	if wrongCreateResult.Code != http.StatusForbidden {
		t.Fatalf("contradictory namespace create status=%d body=%s", wrongCreateResult.Code, wrongCreateResult.Body.String())
	}
	selectedCreate := request("/v1/studio/sdk/components.create", `{"slug":"dql-load","title":"DQL Load","defaultConnectorName":"main"}`)
	selectedCreate.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	server.ServeHTTP(loadReportResponse, selectedCreate)
	if loadReportResponse.Code != http.StatusOK {
		t.Fatalf("create DQL load report status=%d body=%s", loadReportResponse.Code, loadReportResponse.Body.String())
	}
	var loadReport struct {
		ID string `json:"id"`
	}
	if err = json.Unmarshal(loadReportResponse.Body.Bytes(), &loadReport); err != nil || loadReport.ID == "" {
		t.Fatalf("DQL load report identity=%+v err=%v", loadReport, err)
	}
	var componentWorkspace string
	if err := store.QueryRowContext(ctx, `SELECT namespace_id FROM components WHERE id=?`, loadReport.ID).Scan(&componentWorkspace); err != nil || componentWorkspace != namespaceaccess.ID("alice", "production.audit") {
		t.Fatalf("component namespace ownership=%q err=%v", componentWorkspace, err)
	}

	blockedImport := request("/v1/studio/sdk/versions.load_dql", `{"reportId":"`+loadReport.ID+`","input":{"dql":"SELECT wrong_namespace"}}`)
	blockedImport.Header.Set("X-Studio-Namespace", wrongNamespaceID)
	blockedImportResponse := httptest.NewRecorder()
	server.ServeHTTP(blockedImportResponse, blockedImport)
	if blockedImportResponse.Code < 400 || blockedImportResponse.Code >= 500 {
		t.Fatalf("cross-namespace DQL import status=%d body=%s", blockedImportResponse.Code, blockedImportResponse.Body.String())
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE report_id=?`, loadReport.ID).Scan(&reportCount); err != nil || reportCount != 0 {
		t.Fatalf("denied namespace import wrote versions: count=%d err=%v", reportCount, err)
	}
	deniedDQLLoad := request("/v1/studio/sdk/versions.load_dql", `{"reportId":"`+loadReport.ID+`","input":{"dql":"SELECT 1"}}`)
	deniedDQLLoad.Header.Set("Authorization", "Bearer "+bobToken)
	deniedDQLLoadResponse := httptest.NewRecorder()
	server.ServeHTTP(deniedDQLLoadResponse, deniedDQLLoad)
	if deniedDQLLoadResponse.Code != http.StatusForbidden {
		t.Fatalf("non-editor DQL load status=%d body=%s", deniedDQLLoadResponse.Code, deniedDQLLoadResponse.Body.String())
	}
	invalidDQLLoad := httptest.NewRecorder()
	server.ServeHTTP(invalidDQLLoad, request("/v1/studio/sdk/versions.load_dql", `{"reportId":"`+loadReport.ID+`","input":{"dql":" "}}`))
	if invalidDQLLoad.Code != http.StatusBadRequest {
		t.Fatalf("invalid DQL load status=%d body=%s", invalidDQLLoad.Code, invalidDQLLoad.Body.String())
	}
	loadedDQL := httptest.NewRecorder()
	loadedDQLRequest := request("/v1/studio/sdk/versions.load_dql", `{"reportId":"`+loadReport.ID+`","input":{"dql":"SELECT 1","notes":"first import"}}`)
	loadedDQLRequest.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	server.ServeHTTP(loadedDQL, loadedDQLRequest)
	if loadedDQL.Code != http.StatusOK || !strings.Contains(loadedDQL.Body.String(), `"entryDql":"main.dql"`) ||
		!strings.Contains(loadedDQL.Body.String(), `"versionNo":1`) || !strings.Contains(loadedDQL.Body.String(), `"authoredDql":"SELECT 1"`) {
		t.Fatalf("native DQL load status=%d body=%s", loadedDQL.Code, loadedDQL.Body.String())
	}
	var importedWorkspace string
	if err := store.QueryRowContext(ctx, `SELECT namespace_id FROM component_versions WHERE report_id=? AND version_no=1`, loadReport.ID).Scan(&importedWorkspace); err != nil || importedWorkspace != namespaceaccess.ID("alice", "production.audit") {
		t.Fatalf("native imported version namespace=%q err=%v", importedWorkspace, err)
	}
	var mismatchedImportFiles int
	if err := store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_resource_files WHERE report_id=? AND version_no=1 AND namespace_id<>?`, loadReport.ID, importedWorkspace).Scan(&mismatchedImportFiles); err != nil || mismatchedImportFiles != 0 {
		t.Fatalf("native imported files with wrong namespace=%d err=%v", mismatchedImportFiles, err)
	}
	var loadVersionCount, loadFileCount, loadDraft, loadETag int
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE report_id=? AND version_no=1`, loadReport.ID).Scan(&loadVersionCount); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_resource_files WHERE report_id=? AND version_no=1 AND resource_path='main.dql'`, loadReport.ID).Scan(&loadFileCount); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRowContext(ctx, `SELECT current_draft_version,etag FROM components WHERE id=?`, loadReport.ID).Scan(&loadDraft, &loadETag); err != nil ||
		loadVersionCount != 1 || loadFileCount != 1 || loadDraft != 1 || loadETag != 2 {
		t.Fatalf("DQL import persisted version=%d file=%d draft=%d etag=%d err=%v", loadVersionCount, loadFileCount, loadDraft, loadETag, err)
	}

	blockedEdit := request("/v1/studio/sdk/versions.apply", `{"reportId":"`+loadReport.ID+`","versionNo":1,"command":{"kind":"set_dql","expectedSourceRevision":1,"payload":{"authoredDql":"SELECT wrong_namespace"}}}`)
	blockedEdit.Header.Set("X-Studio-Namespace", wrongNamespaceID)
	blockedEditResponse := httptest.NewRecorder()
	server.ServeHTTP(blockedEditResponse, blockedEdit)
	if blockedEditResponse.Code < 400 || blockedEditResponse.Code >= 500 {
		t.Fatalf("cross-namespace version edit status=%d body=%s", blockedEditResponse.Code, blockedEditResponse.Body.String())
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE report_id=? AND version_no=1 AND source_revision=1 AND authored_dql='SELECT 1'`, loadReport.ID).Scan(&reportCount); err != nil || reportCount != 1 {
		t.Fatalf("denied namespace edit changed source: count=%d err=%v", reportCount, err)
	}
	deniedVersionEdit := request("/v1/studio/sdk/versions.apply", `{"reportId":"`+loadReport.ID+`","versionNo":1,"command":{"kind":"set_dql","expectedSourceRevision":1,"payload":{"authoredDql":"SELECT stolen"}}}`)
	deniedVersionEdit.Header.Set("Authorization", "Bearer "+bobToken)
	deniedVersionEditResponse := httptest.NewRecorder()
	server.ServeHTTP(deniedVersionEditResponse, deniedVersionEdit)
	if deniedVersionEditResponse.Code != http.StatusForbidden {
		t.Fatalf("non-editor version apply status=%d body=%s", deniedVersionEditResponse.Code, deniedVersionEditResponse.Body.String())
	}
	invalidVersionEdit := httptest.NewRecorder()
	server.ServeHTTP(invalidVersionEdit, request("/v1/studio/sdk/versions.apply", `{"reportId":"`+loadReport.ID+`","versionNo":1,"command":{"kind":"unknown","expectedSourceRevision":1,"payload":{}}}`))
	if invalidVersionEdit.Code != http.StatusBadRequest {
		t.Fatalf("invalid version edit status=%d body=%s", invalidVersionEdit.Code, invalidVersionEdit.Body.String())
	}
	ownerVersionEdit := httptest.NewRecorder()
	ownerVersionEditRequest := request("/v1/studio/sdk/versions.apply", `{"reportId":"`+loadReport.ID+`","versionNo":1,"command":{"kind":"set_dql","expectedSourceRevision":1,"payload":{"authoredDql":"SELECT edited"}}}`)
	ownerVersionEditRequest.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	server.ServeHTTP(ownerVersionEdit, ownerVersionEditRequest)
	if ownerVersionEdit.Code != http.StatusOK || !strings.Contains(ownerVersionEdit.Body.String(), `"sourceRevision":2`) ||
		!strings.Contains(ownerVersionEdit.Body.String(), `"authoredDql":"SELECT edited"`) {
		t.Fatalf("native version apply status=%d body=%s", ownerVersionEdit.Code, ownerVersionEdit.Body.String())
	}
	staleVersionEdit := httptest.NewRecorder()
	server.ServeHTTP(staleVersionEdit, request("/v1/studio/sdk/versions.apply", `{"reportId":"`+loadReport.ID+`","versionNo":1,"command":{"kind":"set_dql","expectedSourceRevision":1,"payload":{"authoredDql":"SELECT stale"}}}`))
	if staleVersionEdit.Code != http.StatusConflict {
		t.Fatalf("stale version apply status=%d body=%s", staleVersionEdit.Code, staleVersionEdit.Body.String())
	}
	archiveReportResponse := httptest.NewRecorder()
	server.ServeHTTP(archiveReportResponse, request("/v1/studio/sdk/components.create", `{"slug":"archive-load","title":"Archive Load","namespace":"production.audit","defaultConnectorName":"main"}`))
	if archiveReportResponse.Code != http.StatusOK {
		t.Fatalf("create archive load report status=%d body=%s", archiveReportResponse.Code, archiveReportResponse.Body.String())
	}
	var archiveReport struct {
		ID string `json:"id"`
	}
	if err = json.Unmarshal(archiveReportResponse.Body.Bytes(), &archiveReport); err != nil || archiveReport.ID == "" {
		t.Fatalf("archive load report identity=%+v err=%v", archiveReport, err)
	}
	var archiveBuffer bytes.Buffer
	archiveWriter := zip.NewWriter(&archiveBuffer)
	for _, file := range []struct{ name, content string }{
		{"main.dql", "SELECT 1"}, {"other.dql", "SELECT 2"}, {"sql/dependency.sql", "SELECT 3"},
	} {
		member, createErr := archiveWriter.Create(file.name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := member.Write([]byte(file.content)); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err = archiveWriter.Close(); err != nil {
		t.Fatal(err)
	}
	archiveBytes := archiveBuffer.Bytes()
	archiveRequest := func(entry string) string {
		payload, marshalErr := json.Marshal(map[string]any{"reportId": archiveReport.ID, "input": map[string]any{
			"archive": archiveBytes, "format": "zip", "entryDql": entry,
		}})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		return string(payload)
	}
	deniedArchivePayload, err := json.Marshal(map[string]any{"reportId": archiveReport.ID, "input": map[string]any{
		"archive": []byte("not-a-zip"), "format": "zip",
	}})
	if err != nil {
		t.Fatal(err)
	}
	deniedArchiveRequest := request("/v1/studio/sdk/versions.load_archive", string(deniedArchivePayload))
	deniedArchiveRequest.Header.Set("Authorization", "Bearer "+bobToken)
	deniedArchive := httptest.NewRecorder()
	server.ServeHTTP(deniedArchive, deniedArchiveRequest)
	if deniedArchive.Code != http.StatusForbidden {
		t.Fatalf("unauthorized malformed archive status=%d body=%s", deniedArchive.Code, deniedArchive.Body.String())
	}
	ambiguousArchive := httptest.NewRecorder()
	server.ServeHTTP(ambiguousArchive, request("/v1/studio/sdk/versions.load_archive", archiveRequest("")))
	if ambiguousArchive.Code != http.StatusBadRequest {
		t.Fatalf("ambiguous archive entry status=%d body=%s", ambiguousArchive.Code, ambiguousArchive.Body.String())
	}
	blockedArchiveRequest := request("/v1/studio/sdk/versions.load_archive", archiveRequest("main.dql"))
	blockedArchiveRequest.Header.Set("X-Studio-Namespace", wrongNamespaceID)
	blockedArchiveResponse := httptest.NewRecorder()
	server.ServeHTTP(blockedArchiveResponse, blockedArchiveRequest)
	if blockedArchiveResponse.Code < 400 || blockedArchiveResponse.Code >= 500 {
		t.Fatalf("cross-namespace archive status=%d body=%s", blockedArchiveResponse.Code, blockedArchiveResponse.Body.String())
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE report_id=?`, archiveReport.ID).Scan(&reportCount); err != nil || reportCount != 0 {
		t.Fatalf("denied archive created versions: count=%d err=%v", reportCount, err)
	}

	loadedArchive := httptest.NewRecorder()
	loadedArchiveRequest := request("/v1/studio/sdk/versions.load_archive", archiveRequest("main.dql"))
	loadedArchiveRequest.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	server.ServeHTTP(loadedArchive, loadedArchiveRequest)
	if loadedArchive.Code != http.StatusOK || !strings.Contains(loadedArchive.Body.String(), `"entryDql":"main.dql"`) ||
		!strings.Contains(loadedArchive.Body.String(), `"versionNo":1`) || !strings.Contains(loadedArchive.Body.String(), `"other.dql"`) ||
		!strings.Contains(loadedArchive.Body.String(), `"sql/dependency.sql"`) {
		t.Fatalf("native archive load status=%d body=%s", loadedArchive.Code, loadedArchive.Body.String())
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_resource_files WHERE report_id=? AND version_no=1`, archiveReport.ID).Scan(&loadFileCount); err != nil || loadFileCount != 3 {
		t.Fatalf("archive resource count=%d err=%v", loadFileCount, err)
	}
	if err = store.QueryRowContext(ctx, `SELECT current_draft_version,etag FROM components WHERE id=?`, archiveReport.ID).Scan(&loadDraft, &loadETag); err != nil || loadDraft != 1 || loadETag != 2 {
		t.Fatalf("archive draft pointer=%d etag=%d err=%v", loadDraft, loadETag, err)
	}
	referencedConnector := httptest.NewRecorder()
	server.ServeHTTP(referencedConnector, request("/v1/studio/sdk/connectors.delete", `{"name":"main","etag":1}`))
	if referencedConnector.Code != http.StatusConflict || !strings.Contains(referencedConnector.Body.String(), "referenced") {
		t.Fatalf("referenced connector delete status=%d body=%s", referencedConnector.Code, referencedConnector.Body.String())
	}
	delegatedReport := httptest.NewRecorder()
	server.ServeHTTP(delegatedReport, request("/v1/studio/sdk/components.create", `{"slug":"delegated-reader","title":"Delegated Reader","namespace":"production.audit","defaultConnectorName":"delegated"}`))
	if delegatedReport.Code != http.StatusOK {
		t.Fatalf("create delegated connector report status=%d body=%s", delegatedReport.Code, delegatedReport.Body.String())
	}
	var delegatedBody struct {
		ID string `json:"id"`
	}
	if err = json.Unmarshal(delegatedReport.Body.Bytes(), &delegatedBody); err != nil || delegatedBody.ID == "" {
		t.Fatalf("delegated report identity=%+v err=%v", delegatedBody, err)
	}
	if _, err = store.ExecContext(ctx, `INSERT INTO component_acl(report_id,subject_type,subject_id,can_view,can_edit)
		VALUES(?,'user','bob',1,1)`, delegatedBody.ID); err != nil {
		t.Fatal(err)
	}
	delegatedRequest := request("/v1/studio/sdk/connectors.disable", `{"name":"delegated","etag":1}`)
	delegatedRequest.Header.Set("Authorization", "Bearer "+bobToken)
	delegatedDisable := httptest.NewRecorder()
	server.ServeHTTP(delegatedDisable, delegatedRequest)
	if delegatedDisable.Code != http.StatusOK || !strings.Contains(delegatedDisable.Body.String(), `"status":"disabled"`) ||
		strings.Contains(delegatedDisable.Body.String(), "private-dsn") {
		t.Fatalf("delegated editor disable status=%d body=%s", delegatedDisable.Code, delegatedDisable.Body.String())
	}
	unversionedConnectorChange := httptest.NewRecorder()
	server.ServeHTTP(unversionedConnectorChange, request("/v1/studio/sdk/components.update", `{"id":"`+delegatedBody.ID+`","input":{"defaultConnectorName":"activate_http","etag":1}}`))
	if unversionedConnectorChange.Code != http.StatusOK ||
		!strings.Contains(unversionedConnectorChange.Body.String(), `"defaultConnectorName":"activate_http"`) ||
		!strings.Contains(unversionedConnectorChange.Body.String(), `"etag":2`) {
		t.Fatalf("unversioned connector change status=%d body=%s", unversionedConnectorChange.Code, unversionedConnectorChange.Body.String())
	}
	delegatedVersionRequest := request("/v1/studio/sdk/versions.create", `{"reportId":"`+delegatedBody.ID+`","input":{"authoringMode":"dql","authoredDql":"SELECT private_value"}}`)
	delegatedVersionRequest.Header.Set("Authorization", "Bearer "+bobToken)
	delegatedVersion := httptest.NewRecorder()
	server.ServeHTTP(delegatedVersion, delegatedVersionRequest)
	if delegatedVersion.Code != http.StatusOK || !strings.Contains(delegatedVersion.Body.String(), `"createdBy":"bob"`) ||
		strings.Contains(delegatedVersion.Body.String(), "private_value") || strings.Contains(delegatedVersion.Body.String(), `"authoredDql"`) ||
		strings.Contains(delegatedVersion.Body.String(), `"generatedDql"`) {
		t.Fatalf("delegated editor version create status=%d body=%s", delegatedVersion.Code, delegatedVersion.Body.String())
	}
	var delegatedSource string
	if err = store.QueryRowContext(ctx, `SELECT authored_dql FROM component_versions WHERE report_id=? AND version_no=1`, delegatedBody.ID).Scan(&delegatedSource); err != nil || delegatedSource != "SELECT private_value" {
		t.Fatalf("delegated version stored source=%q err=%v", delegatedSource, err)
	}
	viewerInspectRequest := request("/v1/studio/sdk/versions.inspect", `{"reportId":"`+delegatedBody.ID+`","versionNo":1}`)
	viewerInspectRequest.Header.Set("Authorization", "Bearer "+bobToken)
	viewerInspection := httptest.NewRecorder()
	server.ServeHTTP(viewerInspection, viewerInspectRequest)
	if viewerInspection.Code != http.StatusOK || !strings.Contains(viewerInspection.Body.String(), `"canUseDql":false`) ||
		strings.Contains(viewerInspection.Body.String(), "private_value") || strings.Contains(viewerInspection.Body.String(), `"authoredDql"`) ||
		strings.Contains(viewerInspection.Body.String(), `"generatedDql"`) {
		t.Fatalf("delegated version inspect status=%d body=%s", viewerInspection.Code, viewerInspection.Body.String())
	}
	delegatedDQLLoad := request("/v1/studio/sdk/versions.load_dql", `{"reportId":"`+delegatedBody.ID+`","input":{"dql":"SELECT delegated_import"}}`)
	delegatedDQLLoad.Header.Set("Authorization", "Bearer "+bobToken)
	delegatedDQLLoadResponse := httptest.NewRecorder()
	server.ServeHTTP(delegatedDQLLoadResponse, delegatedDQLLoad)
	if delegatedDQLLoadResponse.Code != http.StatusOK || !strings.Contains(delegatedDQLLoadResponse.Body.String(), `"versionNo":2`) ||
		strings.Contains(delegatedDQLLoadResponse.Body.String(), "delegated_import") || strings.Contains(delegatedDQLLoadResponse.Body.String(), `"authoredDql"`) {
		t.Fatalf("delegated DQL import status=%d body=%s", delegatedDQLLoadResponse.Code, delegatedDQLLoadResponse.Body.String())
	}
	delegatedArchivePayload, err := json.Marshal(map[string]any{"reportId": delegatedBody.ID, "input": map[string]any{
		"archive": archiveBytes, "format": "zip", "entryDql": "main.dql",
	}})
	if err != nil {
		t.Fatal(err)
	}
	delegatedArchiveRequest := request("/v1/studio/sdk/versions.load_archive", string(delegatedArchivePayload))
	delegatedArchiveRequest.Header.Set("Authorization", "Bearer "+bobToken)
	delegatedArchive := httptest.NewRecorder()
	server.ServeHTTP(delegatedArchive, delegatedArchiveRequest)
	if delegatedArchive.Code != http.StatusOK || !strings.Contains(delegatedArchive.Body.String(), `"versionNo":3`) ||
		strings.Contains(delegatedArchive.Body.String(), `"authoredDql"`) || strings.Contains(delegatedArchive.Body.String(), `"generatedDql"`) {
		t.Fatalf("delegated archive import status=%d body=%s", delegatedArchive.Code, delegatedArchive.Body.String())
	}
	delegatedEditRequest := request("/v1/studio/sdk/versions.apply", `{"reportId":"`+delegatedBody.ID+`","versionNo":3,"command":{"kind":"set_dql","expectedSourceRevision":1,"payload":{"authoredDql":"SELECT delegated_edited"}}}`)
	delegatedEditRequest.Header.Set("Authorization", "Bearer "+bobToken)
	delegatedEdit := httptest.NewRecorder()
	server.ServeHTTP(delegatedEdit, delegatedEditRequest)
	if delegatedEdit.Code != http.StatusOK || !strings.Contains(delegatedEdit.Body.String(), `"sourceRevision":2`) ||
		strings.Contains(delegatedEdit.Body.String(), "delegated_edited") || strings.Contains(delegatedEdit.Body.String(), `"authoredDql"`) {
		t.Fatalf("delegated version apply status=%d body=%s", delegatedEdit.Code, delegatedEdit.Body.String())
	}
	usedNamespace := httptest.NewRecorder()
	server.ServeHTTP(usedNamespace, request("/v1/studio/sdk/namespaces.delete", `{"name":"production.audit","etag":2}`))
	if usedNamespace.Code != http.StatusConflict || !strings.Contains(usedNamespace.Body.String(), "referenced") {
		t.Fatalf("referenced namespace delete status=%d body=%s", usedNamespace.Code, usedNamespace.Body.String())
	}
	unusedCreated := httptest.NewRecorder()
	server.ServeHTTP(unusedCreated, request("/v1/studio/sdk/namespaces.create", `{"name":"unused","title":"Unused"}`))
	if unusedCreated.Code != http.StatusOK {
		t.Fatalf("create unused namespace status=%d body=%s", unusedCreated.Code, unusedCreated.Body.String())
	}
	bobDeleteRequest := request("/v1/studio/sdk/namespaces.delete", `{"name":"unused","etag":1}`)
	bobDeleteRequest.Header.Set("Authorization", "Bearer "+bobToken)
	bobDelete := httptest.NewRecorder()
	server.ServeHTTP(bobDelete, bobDeleteRequest)
	if bobDelete.Code != http.StatusForbidden {
		t.Fatalf("non-owner namespace delete status=%d body=%s", bobDelete.Code, bobDelete.Body.String())
	}
	staleDelete := httptest.NewRecorder()
	server.ServeHTTP(staleDelete, request("/v1/studio/sdk/namespaces.delete", `{"name":"unused","etag":2}`))
	if staleDelete.Code != http.StatusConflict {
		t.Fatalf("stale namespace delete status=%d body=%s", staleDelete.Code, staleDelete.Body.String())
	}
	deleted := httptest.NewRecorder()
	server.ServeHTTP(deleted, request("/v1/studio/sdk/namespaces.delete", `{"name":"unused","etag":1}`))
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("native namespace delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM namespaces WHERE owner_id='alice' AND name='unused' AND deleted_at IS NOT NULL AND status='archived'`).Scan(&reportCount); err != nil || reportCount != 1 {
		t.Fatalf("archived namespace count=%d err=%v", reportCount, err)
	}
	for _, name := range []string{"mcp_trash", "bff_trash"} {
		created := httptest.NewRecorder()
		server.ServeHTTP(created, request("/v1/studio/sdk/namespaces.create", `{"name":"`+name+`","title":"Disposable"}`))
		if created.Code != http.StatusOK {
			t.Fatalf("create %s status=%d body=%s", name, created.Code, created.Body.String())
		}
	}
	duplicate := httptest.NewRecorder()
	server.ServeHTTP(duplicate, request("/v1/studio/sdk/components.create", `{"slug":"native-first","title":"Duplicate","namespace":"production.audit","defaultConnectorName":"main"}`))
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate native report status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
	unauthenticatedReport := httptest.NewRecorder()
	server.ServeHTTP(unauthenticatedReport, httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/components.create", strings.NewReader(`{"slug":"anonymous","title":"Anonymous","defaultConnectorName":"main"}`)))
	if unauthenticatedReport.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated native report status=%d body=%s", unauthenticatedReport.Code, unauthenticatedReport.Body.String())
	}
	missingNamespace := httptest.NewRecorder()
	server.ServeHTTP(missingNamespace, request("/v1/studio/sdk/components.create", `{"slug":"missing-space","title":"Missing","namespace":"unknown","defaultConnectorName":"main"}`))
	if missingNamespace.Code != http.StatusNotFound {
		t.Fatalf("missing report namespace status=%d body=%s", missingNamespace.Code, missingNamespace.Body.String())
	}
	if _, err = store.ExecContext(ctx, `INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at)
		VALUES('inactive','sqlite','alice','draft',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	inactive := httptest.NewRecorder()
	server.ServeHTTP(inactive, request("/v1/studio/sdk/components.create", `{"slug":"inactive-source","title":"Inactive","defaultConnectorName":"inactive"}`))
	if inactive.Code != http.StatusBadRequest {
		t.Fatalf("inactive report connector status=%d body=%s", inactive.Code, inactive.Body.String())
	}
	internal := httptest.NewRecorder()
	server.ServeHTTP(internal, httptest.NewRequest(http.MethodGet, "/_studio/resource-snapshot/files?reportId=r1&versionNo=1", nil))
	if internal.Code == http.StatusOK {
		t.Fatalf("internal child route became public: status=%d body=%s", internal.Code, internal.Body.String())
	}
	for _, child := range []struct{ method, path string }{
		{http.MethodPatch, "/_studio/report-acl-store"},
		{http.MethodGet, "/_studio/connector-store/access?name=main&subject=alice&permission=edit"},
		{http.MethodGet, "/_studio/connector-store/catalog?name=main&subject=alice&scoped=false&pageLimit=1&pageOffset=0"},
		{http.MethodPatch, "/_studio/connector-store/config"},
		{http.MethodGet, "/_studio/connector-store/usage?name=main"},
		{http.MethodPatch, "/_studio/connector-store/status"},
		{http.MethodGet, "/_studio/namespace-store/read?ownerId=alice&name=general"},
		{http.MethodGet, "/_studio/namespace-store/usage?ownerId=alice&name=general"},
		{http.MethodPatch, "/_studio/namespace-store/write"},
		{http.MethodPost, "/_studio/reports/edit-guard"},
		{http.MethodGet, "/_studio/report-capabilities?reportId=missing&subject=alice"},
		{http.MethodGet, "/_studio/report-store/global-access?subject=alice"},
		{http.MethodGet, "/_studio/report-run-access?reportId=preview-fixture&subject=alice"},
		{http.MethodGet, "/_studio/authorization-predicate-store?limit=1&offset=0"},
		{http.MethodPost, "/_studio/authorization-predicate-store/insert"},
		{http.MethodPatch, "/_studio/authorization-predicate-store"},
		{http.MethodPatch, "/_studio/report-store/config"},
		{http.MethodGet, "/_studio/report-version-store/head?reportId=missing"},
		{http.MethodGet, "/_studio/report-version-store/catalog?reportId=missing"},
		{http.MethodPatch, "/_studio/report-version-store/validation"},
		{http.MethodPost, "/_studio/report-version-store/insert"},
		{http.MethodPost, "/_studio/report-version-store/import"},
		{http.MethodPatch, "/_studio/report-store/draft-pointer"},
		{http.MethodPost, "/_studio/report-store/insert"},
	} {
		denied := httptest.NewRecorder()
		server.ServeHTTP(denied, httptest.NewRequest(child.method, child.path, nil))
		if denied.Code == http.StatusOK {
			t.Fatalf("internal report create child %s %s became public", child.method, child.path)
		}
	}
	mcpCall := func(endpoint, bearer string, cookie *http.Cookie, method, name string, arguments map[string]any) (int, []byte) {
		t.Helper()
		params := map[string]any{"_meta": map[string]any{
			"io.modelcontextprotocol/protocolVersion":    mcpschema.LatestProtocolVersion,
			"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		}}
		if name != "" {
			params["name"], params["arguments"] = name, arguments
		}
		payload, marshalErr := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		req, requestErr := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		if bearer != "" {
			req.Header.Set("Authorization", bearer)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		req.Header.Set("Mcp-Protocol-Version", mcpschema.LatestProtocolVersion)
		req.Header.Set("Mcp-Method", method)
		if name != "" {
			req.Header.Set("Mcp-Name", name)
		}
		res, callErr := http.DefaultClient.Do(req)
		if callErr != nil {
			t.Fatal(callErr)
		}
		defer res.Body.Close()
		body, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			t.Fatal(readErr)
		}
		return res.StatusCode, body
	}
	staticMCP := "http://" + addresses[1] + "/mcp"
	runtimeMCPStatus, runtimeMCPBody := mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.runtime.status", map[string]any{})
	if runtimeMCPStatus != http.StatusOK || bytes.Contains(runtimeMCPBody, []byte(`"isError":true`)) || !bytes.Contains(runtimeMCPBody, []byte(`"status":"idle"`)) {
		t.Fatalf("static MCP runtime status=%d body=%s", runtimeMCPStatus, runtimeMCPBody)
	}
	status, body := mcpCall(staticMCP, "Bearer "+token, nil, "tools/list", "", nil)
	if status != http.StatusOK {
		t.Fatalf("static MCP tools/list status=%d body=%s", status, body)
	}
	var catalog struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err = json.Unmarshal(body, &catalog); err != nil {
		t.Fatal(err)
	}
	available := map[string]bool{}
	for _, tool := range catalog.Result.Tools {
		if available[tool.Name] {
			t.Errorf("static MCP listener listed duplicate tool %s", tool.Name)
		}
		available[tool.Name] = true
	}
	expectedSDKTools := map[string]bool{}
	for path := range document.Paths {
		name := sdkToolForPath(path)
		expectedSDKTools[name] = true
		if !available[name] {
			t.Errorf("static MCP listener did not list %s", name)
		}
	}
	for name := range available {
		if (strings.HasPrefix(name, "studio.sdk.") || strings.HasPrefix(name, "authz.sdk.")) && !expectedSDKTools[name] {
			t.Errorf("static MCP listener exposed undeclared SDK tool %s", name)
		}
	}

	for _, toolName := range []string{"studio.sdk.preview.execute", "studio.sdk.versions.test_view"} {
		arguments := map[string]any{"namespaceId": wrongNamespaceID, "reportId": previewReportID, "versionNo": 1, "input": map[string]any{"limit": 10}}
		if toolName == "studio.sdk.versions.test_view" {
			arguments["view"] = "rows"
		}
		status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", toolName, arguments)
		if status != http.StatusOK || !bytes.Contains(body, []byte(`"isError":true`)) || bytes.Contains(body, []byte(`"returnedRows":1`)) {
			t.Fatalf("cross-namespace MCP run %s status=%d body=%s", toolName, status, body)
		}
	}
	for _, check := range []struct {
		tool string
		args map[string]any
	}{
		{"studio.sdk.versions.validate", map[string]any{"reportId": previewReportID, "versionNo": 1, "expectedSourceRevision": 1}},
		{"studio.sdk.versions.warmup", map[string]any{"reportId": previewReportID, "versionNo": 1}},
		{"studio.sdk.versions.warmup_list", map[string]any{"reportId": previewReportID, "versionNo": 1, "input": map[string]any{"limit": 10}}},
		{"studio.sdk.versions.warmup_get", map[string]any{"reportId": previewReportID, "runId": acceptedWarmup.RunID}},
	} {
		check.args["namespaceId"] = wrongNamespaceID
		status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", check.tool, check.args)
		if status != http.StatusOK || !bytes.Contains(body, []byte(`"isError":true`)) {
			t.Fatalf("cross-namespace MCP %s status=%d body=%s", check.tool, status, body)
		}
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.preview.execute", map[string]any{"namespaceId": namespaceaccess.ID("alice", "production.audit"), "reportId": previewReportID, "versionNo": 1, "input": map[string]any{"limit": 10}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("preview-fixture")) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("static MCP exact-version preview status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.test_view", map[string]any{"namespaceId": namespaceaccess.ID("alice", "production.audit"), "reportId": previewReportID, "versionNo": 1, "view": "rows", "input": map[string]any{"limit": 10}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("preview-fixture")) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("static MCP view test status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.test_relation", map[string]any{"namespaceId": namespaceaccess.ID("alice", "production.audit"), "reportId": "relation-fixture", "versionNo": 1, "relation": "versions", "input": map[string]any{"limit": 10}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"attachedChildren":2`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("static MCP relation test status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.test_compose", map[string]any{"namespaceId": namespaceaccess.ID("alice", "production.audit"), "reportId": "compose-fixture", "versionNo": 1, "input": map[string]any{"cubes": []any{map[string]any{"dimensions": map[string]any{"status": true}, "measures": map[string]any{"productCount": true}, "filters": map[string]any{}}}, "sql": "SELECT t1.status FROM $CubeSQL1 AS t1"}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"data"`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("static MCP compose test status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.validate", map[string]any{"namespaceId": namespaceaccess.ID("alice", "production.audit"), "reportId": previewReportID, "versionNo": 1, "expectedSourceRevision": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"valid":true`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("static MCP version validation status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.warmup", map[string]any{"reportId": previewReportID, "versionNo": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"runId":"w`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("static MCP durable warmup status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.schemas", map[string]any{"name": "main", "input": map[string]any{}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"items"`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("static MCP connector schemas status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.test", map[string]any{"name": "probe_mcp"})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"passed"`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("static MCP connector probe status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.test_sql", map[string]any{"name": "main", "input": map[string]any{"sql": "SELECT slug FROM components WHERE slug='native-first'", "limit": 1}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("native-first")) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("static MCP transient SQL test status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.schemas", map[string]any{"name": "secret_catalog", "input": map[string]any{}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"items"`)) || bytes.Contains(body, []byte(dsn)) || bytes.Contains(body, []byte(secretPath)) {
		t.Fatalf("static MCP secret-backed schema catalog status=%d", status)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.tables", map[string]any{"name": "main", "input": map[string]any{"query": "components", "limit": 5}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"components"`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("static MCP connector tables status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.table", map[string]any{"name": "main", "input": map[string]any{"table": "components"}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"slug"`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("static MCP connector table detail status=%d body=%s", status, body)
	}
	if dockerScaleDSN != "" {
		status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.test_view", map[string]any{"reportId": "preview-wide", "versionNo": 1, "view": "wide", "input": map[string]any{"limit": 10}})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("omega")) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker static MCP wide view status=%d", status)
		}
		status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.validate", map[string]any{"reportId": "preview-wide", "versionNo": 1, "expectedSourceRevision": 1})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"valid":true`)) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker static MCP wide validation status=%d", status)
		}
		status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.preview.execute", map[string]any{"reportId": "preview-wide", "versionNo": 1, "input": map[string]any{"limit": 10}})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("omega")) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker static MCP wide preview status=%d", status)
		}
		status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.test_sql", map[string]any{"name": "scale_mysql", "input": map[string]any{"sql": "SELECT ID,FIELD_59 FROM STUDIO_WIDE_60 ORDER BY ID", "limit": 1}})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("omega")) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker static MCP transient SQL test status=%d", status)
		}
		status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.test", map[string]any{"name": "scale_mysql"})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"passed"`)) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker static MCP connector probe status=%d", status)
		}
		status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.tables", map[string]any{"name": "scale_mysql", "input": map[string]any{"query": "STUDIO_VIEW_", "limit": 100}})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"STUDIO_VIEW_20"`)) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker static MCP view catalog status=%d", status)
		}
		status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.table", map[string]any{"name": "scale_mysql", "input": map[string]any{"table": "STUDIO_WIDE_60"}})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"FIELD_59"`)) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker static MCP wide detail status=%d", status)
		}
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.authorization_predicates.types", map[string]any{})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"typeName":"ReportRead"`)) {
		t.Fatalf("static MCP predicate types status=%d body=%s", status, body)
	}
	t.Setenv("STUDIO_PREDICATE_PACKAGES", extensionPath)
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.authorization_predicates.types", map[string]any{})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"typeName":"ReaderScope"`)) {
		t.Fatalf("static MCP extension predicate types status=%d body=%s", status, body)
	}
	t.Setenv("STUDIO_PREDICATE_PACKAGES", "")
	status, body = mcpCall(staticMCP, "Bearer "+bobToken, nil, "tools/call", "studio.sdk.authorization_predicates.types", map[string]any{})
	if status == http.StatusOK && !bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("non-publisher MCP predicate types status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.authorization_predicates.get", map[string]any{"name": "studio.alpha.read"})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"studio.alpha.read"`)) || !bytes.Contains(body, []byte(`"linked":true`)) {
		t.Fatalf("static MCP predicate get status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.authorization_predicates.list", map[string]any{"limit": 1, "offset": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"studio.beta.read"`)) || !bytes.Contains(body, []byte(`"linked":false`)) {
		t.Fatalf("static MCP predicate list status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.authorization_predicates.create", map[string]any{
		"name": "studio.delta.read", "title": "Delta access", "packagePath": "github.com/viant/datly-studio/studio/authorization", "typeName": "ReportPublish",
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"studio.delta.read"`)) {
		t.Fatalf("static MCP predicate create status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.authorization_predicates.update", map[string]any{
		"name": "studio.delta.read", "input": map[string]any{"title": "Delta revised", "etag": 1},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"title":"Delta revised"`)) || !bytes.Contains(body, []byte(`"etag":2`)) {
		t.Fatalf("static MCP predicate update status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.components.create", map[string]any{
		"namespaceId": wrongNamespaceID, "namespace": "production.audit", "slug": "ns-mcp-create-denied", "title": "Denied", "defaultConnectorName": "main",
	})
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("contradictory MCP namespace create status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.components.create", map[string]any{
		"slug": "mcp-default", "title": "MCP Default", "defaultConnectorName": "main",
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"slug":"mcp-default"`)) {
		t.Fatalf("static MCP report create status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM components WHERE slug='mcp-default' AND owner_id='alice' AND namespace='general'`).Scan(&reportCount); err != nil || reportCount != 1 {
		t.Fatalf("MCP default namespace report count=%d err=%v", reportCount, err)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.components.update", map[string]any{"namespaceId": wrongNamespaceID, "id": nativeReport.ID, "input": map[string]any{"title": "MCP wrong workspace", "etag": 2}})
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("MCP cross-namespace component update status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, "SELECT title,etag FROM components WHERE id=?", nativeReport.ID).Scan(&unchangedTitle, &unchangedETag); err != nil || unchangedTitle != "Native Updated" || unchangedETag != 2 {
		t.Fatalf("MCP denial changed component: %q/%d err=%v", unchangedTitle, unchangedETag, err)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.components.update", map[string]any{
		"namespaceId": namespaceaccess.ID("alice", "production.audit"), "id": nativeReport.ID, "input": map[string]any{"title": "MCP Updated", "etag": 2},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"title":"MCP Updated"`)) {
		t.Fatalf("static MCP report update status=%d body=%s", status, body)
	}
	for _, check := range []struct {
		tool string
		args map[string]any
	}{
		{"studio.sdk.acl.upsert", map[string]any{"reportId": nativeReport.ID, "subjectType": "user", "subjectId": "ns_denied_mcp", "canView": true}},
		{"studio.sdk.acl.delete", map[string]any{"reportId": nativeReport.ID, "subjectType": "user", "subjectId": "grant_mcp", "etag": 1}},
	} {
		check.args["namespaceId"] = wrongNamespaceID
		status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", check.tool, check.args)
		if status != http.StatusOK || !bytes.Contains(body, []byte(`"isError":true`)) {
			t.Fatalf("cross-namespace %s status=%d body=%s", check.tool, status, body)
		}
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.acl.list", map[string]any{"namespaceId": wrongNamespaceID, "reportId": nativeReport.ID})
	if status != http.StatusOK || bytes.Contains(body, []byte("grant_mcp")) {
		t.Fatalf("cross-namespace MCP ACL disclosure: status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_acl WHERE report_id=? AND subject_id='grant_mcp' AND etag=1", nativeReport.ID).Scan(&untouchedACLCount); err != nil || untouchedACLCount != 1 {
		t.Fatalf("MCP denial deleted grant: count=%d err=%v", untouchedACLCount, err)
	}
	if err = store.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_acl WHERE report_id=? AND subject_id='ns_denied_mcp'", nativeReport.ID).Scan(&untouchedACLCount); err != nil || untouchedACLCount != 0 {
		t.Fatalf("MCP denial created grant: count=%d err=%v", untouchedACLCount, err)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.acl.upsert", map[string]any{
		"namespaceId": namespaceaccess.ID("alice", "production.audit"), "reportId": nativeReport.ID, "subjectType": "user", "subjectId": "upsert_mcp", "canView": true,
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"subjectId":"upsert_mcp"`)) {
		t.Fatalf("static MCP ACL upsert status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.acl.delete", map[string]any{
		"reportId": nativeReport.ID, "subjectType": "user", "subjectId": "grant_mcp", "etag": 1,
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("static MCP ACL delete status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.create", map[string]any{
		"namespaceId": wrongNamespaceID, "reportId": nativeReport.ID, "input": map[string]any{"authoringMode": "dql", "authoredDql": "SELECT wrong_namespace"},
	})
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("cross-namespace MCP version create status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.create", map[string]any{
		"reportId": nativeReport.ID, "input": map[string]any{"authoringMode": "sql", "authoredSql": "SELECT 2"},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"versionNo":2`)) {
		t.Fatalf("static MCP version create status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.inspect", map[string]any{
		"reportId": nativeReport.ID, "versionNo": 1,
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"canUseDql":true`)) ||
		!bytes.Contains(body, []byte(`"dql":"SELECT 1"`)) {
		t.Fatalf("static MCP owner inspect status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+bobToken, nil, "tools/call", "studio.sdk.versions.inspect", map[string]any{
		"reportId": delegatedBody.ID, "versionNo": 1,
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"canUseDql":false`)) ||
		bytes.Contains(body, []byte("private_value")) {
		t.Fatalf("static MCP delegated inspect status=%d body=%s", status, body)
	}

	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.load_dql", map[string]any{"namespaceId": wrongNamespaceID, "reportId": loadReport.ID, "input": map[string]any{"dql": "SELECT wrong_namespace"}})
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("cross-namespace MCP import status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE report_id=?`, loadReport.ID).Scan(&reportCount); err != nil || reportCount != 1 {
		t.Fatalf("denied MCP import wrote versions: count=%d err=%v", reportCount, err)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.apply", map[string]any{"namespaceId": wrongNamespaceID, "reportId": loadReport.ID, "versionNo": 1, "command": map[string]any{"kind": "set_dql", "expectedSourceRevision": 2, "payload": map[string]any{"authoredDql": "SELECT wrong_namespace"}}})
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("cross-namespace MCP edit status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE report_id=? AND version_no=1 AND source_revision=2 AND authored_dql='SELECT edited'`, loadReport.ID).Scan(&reportCount); err != nil || reportCount != 1 {
		t.Fatalf("denied MCP edit changed source: count=%d err=%v", reportCount, err)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.load_dql", map[string]any{
		"reportId": loadReport.ID, "input": map[string]any{"dql": "SELECT 2"},
		"namespaceId": namespaceaccess.ID("alice", "production.audit"),
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"versionNo":2`)) ||
		!bytes.Contains(body, []byte(`"entryDql":"main.dql"`)) {
		t.Fatalf("static MCP DQL load status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.apply", map[string]any{
		"namespaceId": namespaceaccess.ID("alice", "production.audit"),
		"reportId":    loadReport.ID, "versionNo": 1, "command": map[string]any{
			"kind": "set_sql", "expectedSourceRevision": 2, "payload": map[string]any{"authoredSql": "SELECT 3"},
		},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"sourceRevision":3`)) ||
		!bytes.Contains(body, []byte(`"authoredSql":"SELECT 3"`)) {
		t.Fatalf("static MCP version apply status=%d body=%s", status, body)
	}

	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.load_archive", map[string]any{"namespaceId": wrongNamespaceID, "reportId": archiveReport.ID, "input": map[string]any{"archive": archiveBytes, "format": "zip", "entryDql": "other.dql"}})
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("cross-namespace MCP archive status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE report_id=?`, archiveReport.ID).Scan(&reportCount); err != nil || reportCount != 1 {
		t.Fatalf("denied MCP archive wrote versions: count=%d err=%v", reportCount, err)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.load_archive", map[string]any{
		"namespaceId": namespaceaccess.ID("alice", "production.audit"), "reportId": archiveReport.ID, "input": map[string]any{"archive": archiveBytes, "format": "zip", "entryDql": "other.dql"},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"versionNo":2`)) ||
		!bytes.Contains(body, []byte(`"entryDql":"other.dql"`)) {
		t.Fatalf("static MCP archive load status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM namespaces WHERE owner_id='alice' AND name='general' AND status='active'`).Scan(&reportCount); err != nil || reportCount != 1 {
		t.Fatalf("MCP default namespace count=%d err=%v", reportCount, err)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.namespaces.update", map[string]any{
		"name": "general", "input": map[string]any{"title": "General Readers", "etag": 1},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"title":"General Readers"`)) {
		t.Fatalf("static MCP namespace update status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.disable", map[string]any{"name": "disable_mcp", "etag": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"disabled"`)) ||
		bytes.Contains(body, []byte("private-dsn")) || bytes.Contains(body, []byte("vault://private")) {
		t.Fatalf("static MCP connector disable status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.activate", map[string]any{"name": "activate_mcp", "etag": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"active"`)) ||
		bytes.Contains(body, []byte("private-dsn")) || bytes.Contains(body, []byte("vault://private")) {
		t.Fatalf("static MCP connector activation status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.update", map[string]any{
		"name": "update_mcp", "input": map[string]any{"secretRef": "vault://rotated", "etag": 1},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"draft"`)) ||
		bytes.Contains(body, []byte("vault://rotated")) || bytes.Contains(body, []byte("private-dsn")) {
		t.Fatalf("static MCP connector update status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.connectors.delete", map[string]any{"name": "delete_mcp", "etag": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || bytes.Contains(body, []byte("private-dsn")) {
		t.Fatalf("static MCP connector delete status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.namespaces.delete", map[string]any{"name": "mcp_trash", "etag": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("static MCP namespace delete status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM namespaces WHERE owner_id='alice' AND name='mcp_trash' AND deleted_at IS NOT NULL`).Scan(&reportCount); err != nil || reportCount != 1 {
		t.Fatalf("MCP archived namespace count=%d err=%v", reportCount, err)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.namespaces.list", map[string]any{})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("production.audit")) {
		t.Fatalf("static MCP namespace tool status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "", nil, "tools/call", "studio.sdk.namespaces.list", map[string]any{})
	if bytes.Contains(body, []byte("production.audit")) || status == http.StatusOK && !bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("unauthenticated static MCP tool status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer invalid", nil, "tools/call", "studio.sdk.namespaces.list", map[string]any{})
	if bytes.Contains(body, []byte("production.audit")) || status == http.StatusOK && !bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("invalid static MCP bearer status=%d body=%s", status, body)
	}
	sessions, err := bffauth.New(bffauth.Config{}, staticSessionVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	id, _, _, err := sessions.Exchange(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	target, err := url.Parse("http://" + addresses[1])
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := sessions.Proxy(target, "/v1/studio/sdk-mcp")
	if err != nil {
		t.Fatal(err)
	}
	bff := httptest.NewServer(proxy)
	defer bff.Close()
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/list", "", nil)
	if status != http.StatusOK {
		t.Fatalf("BFF-proxied static MCP tools/list status=%d body=%s", status, body)
	}
	var proxiedCatalog struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err = json.Unmarshal(body, &proxiedCatalog); err != nil {
		t.Fatal(err)
	}
	proxiedTools := map[string]bool{}
	for _, tool := range proxiedCatalog.Result.Tools {
		proxiedTools[tool.Name] = true
	}
	for path := range document.Paths {
		name := sdkToolForPath(path)
		if !proxiedTools[name] {
			t.Errorf("authenticated BFF MCP catalog did not list %s", name)
		}
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", nil, "tools/list", "", nil)
	if status == http.StatusOK {
		t.Fatalf("cookie-less BFF MCP catalog was exposed: %s", body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.namespaces.list", map[string]any{})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("production.audit")) {
		t.Fatalf("BFF-proxied static MCP tool status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.runtime.status", map[string]any{})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"idle"`)) {
		t.Fatalf("BFF-proxied runtime status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.components.create", map[string]any{
		"slug": "bff-report", "title": "BFF Report", "defaultConnectorName": "main",
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"slug":"bff-report"`)) {
		t.Fatalf("BFF-proxied report create status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.components.update", map[string]any{
		"id": nativeReport.ID, "input": map[string]any{"description": "BFF reviewed", "etag": 3},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"description":"BFF reviewed"`)) {
		t.Fatalf("BFF-proxied report update status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.acl.upsert", map[string]any{
		"reportId": nativeReport.ID, "subjectType": "user", "subjectId": "upsert_bff", "canView": true, "canPublish": true,
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"subjectId":"upsert_bff"`)) {
		t.Fatalf("BFF-proxied ACL upsert status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_acl WHERE report_id=? AND
		(subject_id='upsert_http' AND can_view=1 AND can_run=1 AND can_edit=1 AND etag=2 OR
		 subject_id='upsert_mcp' AND can_view=1 AND etag=1 OR
		 subject_id='upsert_bff' AND can_view=1 AND can_publish=1 AND etag=1)`, nativeReport.ID).Scan(&reportCount); err != nil || reportCount != 3 {
		t.Fatalf("ACL upsert persistence count=%d err=%v", reportCount, err)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.acl.delete", map[string]any{
		"reportId": nativeReport.ID, "subjectType": "user", "subjectId": "grant_bff", "etag": 1,
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("BFF-proxied ACL delete status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_acl WHERE report_id=? AND subject_id IN ('grant_http','grant_mcp','grant_bff')`, nativeReport.ID).Scan(&reportCount); err != nil || reportCount != 0 {
		t.Fatalf("ACL deletion persistence count=%d err=%v", reportCount, err)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.authorization_predicates.types", map[string]any{})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"typeName":"ReportRead"`)) {
		t.Fatalf("BFF-proxied predicate types status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.authorization_predicates.list", map[string]any{"query": "Beta", "limit": 50})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"studio.beta.read"`)) || bytes.Contains(body, []byte(`"name":"studio.alpha.read"`)) {
		t.Fatalf("BFF-proxied predicate list status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.authorization_predicates.get", map[string]any{"name": "studio.alpha.read"})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"studio.alpha.read"`)) {
		t.Fatalf("BFF-proxied predicate get status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.schemas", map[string]any{"name": "main", "input": map[string]any{}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"items"`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("BFF-proxied connector schemas status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.preview.execute", map[string]any{"reportId": previewReportID, "versionNo": 1, "input": map[string]any{"limit": 10}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("preview-fixture")) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("BFF-proxied exact-version preview status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.test_view", map[string]any{"reportId": previewReportID, "versionNo": 1, "view": "rows", "input": map[string]any{"limit": 10}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("preview-fixture")) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("BFF-proxied view test status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.test_relation", map[string]any{"reportId": "relation-fixture", "versionNo": 1, "relation": "versions", "input": map[string]any{"limit": 10}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"attachedChildren":2`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("BFF-proxied relation test status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.test_compose", map[string]any{"reportId": "compose-fixture", "versionNo": 1, "input": map[string]any{"cubes": []any{map[string]any{"dimensions": map[string]any{"status": true}, "measures": map[string]any{"productCount": true}, "filters": map[string]any{}}}, "sql": "SELECT t1.status FROM $CubeSQL1 AS t1"}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"data"`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("BFF-proxied compose test status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.validate", map[string]any{"reportId": previewReportID, "versionNo": 1, "expectedSourceRevision": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"valid":true`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("BFF-proxied version validation status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.warmup", map[string]any{"reportId": previewReportID, "versionNo": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"runId":"w`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("BFF-proxied durable warmup status=%d body=%s", status, body)
	}

	// Warmup is asynchronous. Await its exact handle before a connector probe on
	// the same SQLite file, instead of interleaving unrelated readiness checks.
	var bffWarmup struct {
		Result struct {
			StructuredContent sdk.WarmupRun `json:"structuredContent"`
		} `json:"result"`
	}
	if err = json.Unmarshal(body, &bffWarmup); err != nil || bffWarmup.Result.StructuredContent.RunID == "" {
		t.Fatalf("BFF warmup handle missing: decode error=%v", err)
	}
	deadline = time.Now().Add(5 * time.Second)
	var bffWarmupTerminal string
	for time.Now().Before(deadline) {
		var warmupStatus int
		warmupStatus, warmupBody := mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.warmup_get", map[string]any{"reportId": previewReportID, "runId": bffWarmup.Result.StructuredContent.RunID})
		var snapshot struct {
			Result struct {
				StructuredContent sdk.WarmupRun `json:"structuredContent"`
			} `json:"result"`
		}
		if warmupStatus != http.StatusOK || json.Unmarshal(warmupBody, &snapshot) != nil || snapshot.Result.StructuredContent.RunID != bffWarmup.Result.StructuredContent.RunID {
			t.Fatal("BFF warmup status did not match its handle")
		}
		bffWarmupTerminal = snapshot.Result.StructuredContent.Status
		if bffWarmupTerminal == "completed" || bffWarmupTerminal == "failed" || bffWarmupTerminal == "partial" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if bffWarmupTerminal != "completed" && bffWarmupTerminal != "failed" && bffWarmupTerminal != "partial" {
		t.Fatalf("BFF warmup not terminal: %s", bffWarmupTerminal)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.test", map[string]any{"name": "probe_bff"})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"passed"`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("BFF-proxied connector probe status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.test_sql", map[string]any{"name": "main", "input": map[string]any{"sql": "SELECT slug FROM components WHERE slug='native-first'", "limit": 1}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("native-first")) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("BFF-proxied transient SQL test status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.test", map[string]any{"name": "secret_catalog"})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"passed"`)) || bytes.Contains(body, []byte(dsn)) || bytes.Contains(body, []byte(secretPath)) {
		t.Fatalf("BFF MCP secret-backed connector probe status=%d toolError=%t passed=%t leakedDSN=%t leakedSecretReference=%t", status, bytes.Contains(body, []byte(`"isError":true`)), bytes.Contains(body, []byte(`"status":"passed"`)), bytes.Contains(body, []byte(dsn)), bytes.Contains(body, []byte(secretPath)))
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.tables", map[string]any{"name": "main", "input": map[string]any{"query": "components", "limit": 5}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"components"`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("BFF-proxied connector tables status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.table", map[string]any{"name": "main", "input": map[string]any{"table": "components"}})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"slug"`)) || bytes.Contains(body, []byte(dsn)) {
		t.Fatalf("BFF-proxied connector table detail status=%d body=%s", status, body)
	}
	if dockerScaleDSN != "" {
		status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.test_view", map[string]any{"reportId": "preview-wide", "versionNo": 1, "view": "wide", "input": map[string]any{"limit": 10}})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("omega")) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker BFF MCP wide view status=%d", status)
		}
		status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.validate", map[string]any{"reportId": "preview-wide", "versionNo": 1, "expectedSourceRevision": 1})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"valid":true`)) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker BFF MCP wide validation status=%d", status)
		}
		status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.preview.execute", map[string]any{"reportId": "preview-wide", "versionNo": 1, "input": map[string]any{"limit": 10}})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("omega")) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker BFF MCP wide preview status=%d", status)
		}
		status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.test_sql", map[string]any{"name": "scale_mysql", "input": map[string]any{"sql": "SELECT ID,FIELD_59 FROM STUDIO_WIDE_60 ORDER BY ID", "limit": 1}})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte("omega")) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker BFF MCP transient SQL test status=%d", status)
		}
		status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.test", map[string]any{"name": "scale_mysql"})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"passed"`)) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker BFF MCP connector probe status=%d", status)
		}
		status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.tables", map[string]any{"name": "scale_mysql", "input": map[string]any{"query": "STUDIO_VIEW_", "limit": 100}})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"STUDIO_VIEW_20"`)) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker BFF MCP view catalog status=%d", status)
		}
		status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.table", map[string]any{"name": "scale_mysql", "input": map[string]any{"table": "STUDIO_WIDE_60"}})
		if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"FIELD_59"`)) || bytes.Contains(body, []byte(dockerScaleDSN)) {
			t.Fatalf("Docker BFF MCP wide detail status=%d", status)
		}
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.authorization_predicates.create", map[string]any{
		"name": "studio.epsilon.read", "title": "Epsilon access", "packagePath": "github.com/viant/datly-studio/studio/authorization", "typeName": "ReportVersionRead",
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"name":"studio.epsilon.read"`)) {
		t.Fatalf("BFF-proxied predicate create status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.authorization_predicates.delete", map[string]any{
		"name": "studio.epsilon.read", "etag": 1,
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("BFF-proxied predicate delete status=%d body=%s", status, body)
	}
	var epsilonStatus string
	if err = store.QueryRowContext(ctx, `SELECT status FROM authorization_predicates WHERE name='studio.epsilon.read'`).Scan(&epsilonStatus); err != nil || epsilonStatus != "disabled" {
		t.Fatalf("BFF predicate deletion status=%q err=%v", epsilonStatus, err)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.create", map[string]any{
		"reportId": nativeReport.ID, "input": map[string]any{"authoringMode": "structured", "componentSpec": map[string]any{}},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"versionNo":3`)) {
		t.Fatalf("BFF-proxied version create status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.inspect", map[string]any{
		"reportId": nativeReport.ID, "versionNo": 1,
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"canUseDql":true`)) ||
		!bytes.Contains(body, []byte(`"dql":"SELECT 1"`)) {
		t.Fatalf("BFF-proxied version inspect status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.load_dql", map[string]any{
		"reportId": loadReport.ID, "input": map[string]any{"dql": "SELECT 3"},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"versionNo":3`)) ||
		!bytes.Contains(body, []byte(`"entryDql":"main.dql"`)) {
		t.Fatalf("BFF-proxied DQL load status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.apply", map[string]any{
		"reportId": loadReport.ID, "versionNo": 1, "command": map[string]any{
			"kind": "replace_spec", "expectedSourceRevision": 3, "payload": map[string]any{"name": "Updated"},
		},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"sourceRevision":4`)) ||
		!bytes.Contains(body, []byte(`"name":"Updated"`)) {
		t.Fatalf("BFF-proxied version apply status=%d body=%s", status, body)
	}
	var editedSourceRevision int64
	var editedSQL, editedSpec string
	if err = store.QueryRowContext(ctx, `SELECT source_revision,authored_sql,component_spec_json FROM component_versions WHERE report_id=? AND version_no=1`, loadReport.ID).
		Scan(&editedSourceRevision, &editedSQL, &editedSpec); err != nil || editedSourceRevision != 4 || editedSQL != "SELECT 3" || editedSpec != `{"name":"Updated"}` {
		t.Fatalf("MCP version edits persisted revision=%d SQL=%q spec=%q err=%v", editedSourceRevision, editedSQL, editedSpec, err)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.versions.load_archive", map[string]any{
		"reportId": archiveReport.ID, "input": map[string]any{"archive": archiveBytes, "format": "zip", "entryDql": "main.dql"},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"versionNo":3`)) ||
		!bytes.Contains(body, []byte(`"entryDql":"main.dql"`)) {
		t.Fatalf("BFF-proxied archive load status=%d body=%s", status, body)
	}
	var archiveVersionCount, archiveFileCount, archiveDraft, archiveETag int
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE report_id=?`, archiveReport.ID).Scan(&archiveVersionCount); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_resource_files WHERE report_id=?`, archiveReport.ID).Scan(&archiveFileCount); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRowContext(ctx, `SELECT current_draft_version,etag FROM components WHERE id=?`, archiveReport.ID).Scan(&archiveDraft, &archiveETag); err != nil ||
		archiveVersionCount != 3 || archiveFileCount != 9 || archiveDraft != 3 || archiveETag != 4 {
		t.Fatalf("MCP archive imports persisted version=%d file=%d draft=%d etag=%d err=%v", archiveVersionCount, archiveFileCount, archiveDraft, archiveETag, err)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE report_id=?`, loadReport.ID).Scan(&loadVersionCount); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_resource_files WHERE report_id=?`, loadReport.ID).Scan(&loadFileCount); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRowContext(ctx, `SELECT current_draft_version,etag FROM components WHERE id=?`, loadReport.ID).Scan(&loadDraft, &loadETag); err != nil ||
		loadVersionCount != 3 || loadFileCount != 3 || loadDraft != 3 || loadETag != 4 {
		t.Fatalf("MCP DQL imports persisted version=%d file=%d draft=%d etag=%d err=%v", loadVersionCount, loadFileCount, loadDraft, loadETag, err)
	}
	if _, err = store.ExecContext(ctx, `CREATE TRIGGER reject_native_draft BEFORE UPDATE OF current_draft_version ON components
		BEGIN SELECT RAISE(ABORT,'test pointer failure'); END`); err != nil {
		t.Fatal(err)
	}
	rejectedImport := httptest.NewRecorder()
	server.ServeHTTP(rejectedImport, request("/v1/studio/sdk/versions.load_dql", `{"reportId":"`+loadReport.ID+`","input":{"dql":"SELECT rejected"}}`))
	if rejectedImport.Code == http.StatusOK {
		t.Fatalf("native DQL import ignored draft-pointer failure: %s", rejectedImport.Body.String())
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE report_id=?`, loadReport.ID).Scan(&loadVersionCount); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_resource_files WHERE report_id=?`, loadReport.ID).Scan(&loadFileCount); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRowContext(ctx, `SELECT current_draft_version,etag FROM components WHERE id=?`, loadReport.ID).Scan(&loadDraft, &loadETag); err != nil ||
		loadVersionCount != 3 || loadFileCount != 3 || loadDraft != 3 || loadETag != 4 {
		t.Fatalf("failed native import was not atomic: version=%d file=%d draft=%d etag=%d err=%v", loadVersionCount, loadFileCount, loadDraft, loadETag, err)
	}
	if _, err = store.ExecContext(ctx, `DROP TRIGGER reject_native_draft`); err != nil {
		t.Fatal(err)
	}
	var updatedTitle, updatedDescription string
	var updatedETag int64
	if err = store.QueryRowContext(ctx, `SELECT title,description,etag FROM components WHERE id=?`, nativeReport.ID).Scan(&updatedTitle, &updatedDescription, &updatedETag); err != nil ||
		updatedTitle != "MCP Updated" || updatedDescription != "BFF reviewed" || updatedETag != 4 {
		t.Fatalf("report update persistence title=%q description=%q etag=%d err=%v", updatedTitle, updatedDescription, updatedETag, err)
	}
	versionedConnectorChange := httptest.NewRecorder()
	server.ServeHTTP(versionedConnectorChange, request("/v1/studio/sdk/components.update", `{"id":"`+nativeReport.ID+`","input":{"defaultConnectorName":"activate_http","etag":4}}`))
	if versionedConnectorChange.Code != http.StatusConflict || !strings.Contains(versionedConnectorChange.Body.String(), "versioned reader contract") {
		t.Fatalf("versioned connector change status=%d body=%s", versionedConnectorChange.Code, versionedConnectorChange.Body.String())
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.namespaces.update", map[string]any{
		"name": "production.audit", "input": map[string]any{"description": "Reviewed", "etag": 2},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"description":"Reviewed"`)) {
		t.Fatalf("BFF-proxied namespace update status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.disable", map[string]any{"name": "disable_bff", "etag": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"disabled"`)) ||
		bytes.Contains(body, []byte("private-dsn")) || bytes.Contains(body, []byte("vault://private")) {
		t.Fatalf("BFF-proxied connector disable status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.activate", map[string]any{"name": "activate_bff", "etag": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"active"`)) ||
		bytes.Contains(body, []byte("private-dsn")) || bytes.Contains(body, []byte("vault://private")) {
		t.Fatalf("BFF-proxied connector activation status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.update", map[string]any{
		"name": "update_bff", "input": map[string]any{"description": "BFF reviewed", "etag": 1},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"active"`)) ||
		!bytes.Contains(body, []byte(`"description":"BFF reviewed"`)) || bytes.Contains(body, []byte("private-dsn")) {
		t.Fatalf("BFF-proxied connector update status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM connectors WHERE
		(name='update_http' AND dsn_template='new-private-dsn' AND status='draft' AND last_test_status IS NULL AND last_tested_at IS NULL AND etag=2)
		OR (name='update_mcp' AND secret_ref='vault://rotated' AND status='draft' AND etag=2)
		OR (name='update_bff' AND description='BFF reviewed' AND status='active' AND etag=2)
		OR (name='update_description' AND description='Reviewed' AND status='active' AND last_test_status='passed' AND etag=2)`).Scan(&reportCount); err != nil || reportCount != 4 {
		t.Fatalf("updated connector persistence count=%d err=%v", reportCount, err)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.connectors.delete", map[string]any{"name": "delete_bff", "etag": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || bytes.Contains(body, []byte("private-dsn")) {
		t.Fatalf("BFF-proxied connector delete status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM connectors WHERE name IN ('delete_http','delete_mcp','delete_bff')
		AND status='deleted' AND etag=2 AND deleted_at IS NOT NULL`).Scan(&reportCount); err != nil || reportCount != 3 {
		t.Fatalf("deleted connector persistence count=%d err=%v", reportCount, err)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM connectors WHERE name IN ('disable_http','disable_mcp','disable_bff')
		AND status='disabled' AND etag=2 AND dsn_template='private-dsn' AND secret_ref='vault://private'`).Scan(&reportCount); err != nil || reportCount != 3 {
		t.Fatalf("disabled connector persistence count=%d err=%v", reportCount, err)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM connectors WHERE name IN ('activate_http','activate_mcp','activate_bff')
		AND status='active' AND etag=2 AND dsn_template='private-dsn' AND secret_ref='vault://private'`).Scan(&reportCount); err != nil || reportCount != 3 {
		t.Fatalf("activated connector persistence count=%d err=%v", reportCount, err)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.namespaces.delete", map[string]any{"name": "bff_trash", "etag": 1})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("BFF-proxied namespace delete status=%d body=%s", status, body)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM namespaces WHERE owner_id='alice' AND name='bff_trash' AND deleted_at IS NOT NULL`).Scan(&reportCount); err != nil || reportCount != 1 {
		t.Fatalf("BFF archived namespace count=%d err=%v", reportCount, err)
	}
	resourceCall := func(operation, body string) sdk.ResourceSnapshot {
		t.Helper()
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request("/v1/studio/sdk/resources."+operation, body))
		var snapshot sdk.ResourceSnapshot
		if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &snapshot) != nil || snapshot.Version == nil {
			t.Fatalf("native resource %s status=%d body=%s", operation, response.Code, response.Body.String())
		}
		return snapshot
	}
	const resourceNamespace = "alice.native_resources"
	fileSnapshot := resourceCall("upsert_file", `{"reportId":"preview-fixture","versionNo":3,"namespace":"`+resourceNamespace+`","resourcePath":"guide/readme.md","content":"public guide","expectedSourceRevision":1}`)
	if len(fileSnapshot.Files) != 1 || fileSnapshot.Files[0].ResourceID == "" || fileSnapshot.Version.SourceRevision != 2 {
		t.Fatalf("native file upsert snapshot=%+v", fileSnapshot)
	}
	folderSnapshot := resourceCall("upsert_folder", `{"reportId":"preview-fixture","versionNo":3,"namespace":"`+resourceNamespace+`","rootPath":"guide","uriPrefix":"skill://native-guide/","expectedSourceRevision":2}`)
	if len(folderSnapshot.Folders) != 1 || folderSnapshot.Folders[0].FolderID == "" || folderSnapshot.Version.SourceRevision != 3 {
		t.Fatalf("native folder upsert snapshot=%+v", folderSnapshot)
	}
	skillFileSnapshot := resourceCall("upsert_file", `{"reportId":"preview-fixture","versionNo":3,"namespace":"`+resourceNamespace+`","resourcePath":"guide/SKILL.md","content":"# Native Skill","expectedSourceRevision":3}`)
	if len(skillFileSnapshot.Files) != 2 || skillFileSnapshot.Version.SourceRevision != 4 {
		t.Fatalf("native skill file upsert snapshot=%+v", skillFileSnapshot)
	}
	var skillFileID string
	for _, file := range skillFileSnapshot.Files {
		if file.ResourcePath == "guide/SKILL.md" {
			skillFileID = file.ResourceID
		}
	}
	if skillFileID == "" {
		t.Fatalf("native skill file ID missing: %+v", skillFileSnapshot.Files)
	}
	skillSnapshot := resourceCall("upsert_skill", `{"reportId":"preview-fixture","versionNo":3,"folderId":"`+folderSnapshot.Folders[0].FolderID+`","skillRoot":".","expectedSourceRevision":4}`)
	if len(skillSnapshot.Skills) != 1 || skillSnapshot.Skills[0].SkillID == "" || skillSnapshot.Version.SourceRevision != 5 {
		t.Fatalf("native skill upsert snapshot=%+v", skillSnapshot)
	}
	var resourceOwner, resourceWorkspace string
	if err := store.QueryRowContext(ctx, `SELECT owner_id,namespace FROM components WHERE id='preview-fixture'`).Scan(&resourceOwner, &resourceWorkspace); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"component_resource_files", "component_resource_folders", "component_skill_roots"} {
		var owned, wrong int
		if err := store.QueryRowContext(ctx, "SELECT COUNT(*), SUM(CASE WHEN namespace_id<>? THEN 1 ELSE 0 END) FROM "+table+" WHERE report_id='preview-fixture'",
			namespaceaccess.ID(resourceOwner, resourceWorkspace)).Scan(&owned, &wrong); err != nil || owned == 0 || wrong != 0 {
			t.Fatalf("native %s namespace ownership total=%d wrong=%d err=%v", table, owned, wrong, err)
		}
	}
	deletedSkill := resourceCall("delete_skill", `{"reportId":"preview-fixture","versionNo":3,"skillId":"`+skillSnapshot.Skills[0].SkillID+`","expectedSourceRevision":5}`)
	if len(deletedSkill.Skills) != 0 || deletedSkill.Version.SourceRevision != 6 {
		t.Fatalf("native skill delete snapshot=%+v", deletedSkill)
	}
	deletedFolder := resourceCall("delete_folder", `{"reportId":"preview-fixture","versionNo":3,"folderId":"`+folderSnapshot.Folders[0].FolderID+`","expectedSourceRevision":6}`)
	if len(deletedFolder.Folders) != 0 || deletedFolder.Version.SourceRevision != 7 {
		t.Fatalf("native folder delete snapshot=%+v", deletedFolder)
	}
	deletedFile := resourceCall("delete_file", `{"reportId":"preview-fixture","versionNo":3,"resourceId":"`+fileSnapshot.Files[0].ResourceID+`","expectedSourceRevision":7}`)
	if len(deletedFile.Files) != 1 || deletedFile.Version.SourceRevision != 8 {
		t.Fatalf("native file delete snapshot=%+v", deletedFile)
	}
	deletedSkillFile := resourceCall("delete_file", `{"reportId":"preview-fixture","versionNo":3,"resourceId":"`+skillFileID+`","expectedSourceRevision":8}`)
	if len(deletedSkillFile.Files) != 0 || deletedSkillFile.Version.SourceRevision != 9 {
		t.Fatalf("native skill file delete snapshot=%+v", deletedSkillFile)
	}
	blockedResourceRequest := request("/v1/studio/sdk/resources.upsert_file", `{"reportId":"preview-fixture","versionNo":3,"namespace":"`+resourceNamespace+`","resourceId":"ns-blocked","resourcePath":"blocked.md","content":"blocked","expectedSourceRevision":9}`)
	blockedResourceRequest.Header.Set("X-Studio-Namespace", wrongNamespaceID)
	blockedResourceResponse := httptest.NewRecorder()
	server.ServeHTTP(blockedResourceResponse, blockedResourceRequest)
	if blockedResourceResponse.Code < 400 || blockedResourceResponse.Code >= 500 {
		t.Fatalf("cross-namespace resource status=%d body=%s", blockedResourceResponse.Code, blockedResourceResponse.Body.String())
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_resource_files WHERE resource_id='ns-blocked'`).Scan(&reportCount); err != nil || reportCount != 0 {
		t.Fatalf("denied resource persisted: count=%d err=%v", reportCount, err)
	}

	invalidResource := httptest.NewRecorder()
	server.ServeHTTP(invalidResource, request("/v1/studio/sdk/resources.upsert_file", `{"reportId":"preview-fixture","versionNo":3,"namespace":"`+resourceNamespace+`","resourcePath":"../escape.md","content":"invalid","expectedSourceRevision":9}`))
	if invalidResource.Code != http.StatusBadRequest {
		t.Fatalf("native invalid resource status=%d body=%s", invalidResource.Code, invalidResource.Body.String())
	}
	var sourceRevision int64
	if err = store.QueryRowContext(ctx, `SELECT source_revision FROM component_versions WHERE report_id='preview-fixture' AND version_no=3`).Scan(&sourceRevision); err != nil || sourceRevision != 9 {
		t.Fatalf("native resource rollback revision=%d err=%v", sourceRevision, err)
	}
	mcpResourceCall := func(viaBFF bool, operation string, arguments map[string]any, wantRevision int64) {
		t.Helper()
		endpoint, bearer := staticMCP, "Bearer "+token
		var cookie *http.Cookie
		if viaBFF {
			endpoint, bearer = bff.URL+"/v1/studio/sdk-mcp/mcp", ""
			cookie = &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}
		}
		arguments["namespaceId"] = namespaceaccess.ID("alice", "production.audit")
		callStatus, callBody := mcpCall(endpoint, bearer, cookie, "tools/call", "studio.sdk.resources."+operation, arguments)
		if callStatus != http.StatusOK || bytes.Contains(callBody, []byte(`"isError":true`)) {
			t.Fatalf("native MCP resource %s viaBFF=%t status=%d body=%s", operation, viaBFF, callStatus, callBody)
		}
		var actual int64
		if err := store.QueryRowContext(ctx, `SELECT source_revision FROM component_versions WHERE report_id='preview-fixture' AND version_no=3`).Scan(&actual); err != nil || actual != wantRevision {
			t.Fatalf("native MCP resource %s revision=%d want=%d err=%v", operation, actual, wantRevision, err)
		}
	}

	blockedResourceStatus, blockedResourceBody := mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.resources.upsert_file", map[string]any{"namespaceId": wrongNamespaceID, "reportId": "preview-fixture", "versionNo": 3, "resourceId": "ns-mcp-blocked", "namespace": resourceNamespace, "resourcePath": "blocked.md", "content": "blocked", "expectedSourceRevision": 9})
	if blockedResourceStatus != http.StatusOK || !bytes.Contains(blockedResourceBody, []byte(`"isError":true`)) {
		t.Fatalf("cross-namespace MCP file status=%d body=%s", blockedResourceStatus, blockedResourceBody)
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_resource_files WHERE resource_id='ns-mcp-blocked'`).Scan(&reportCount); err != nil || reportCount != 0 {
		t.Fatalf("denied MCP file persisted: count=%d err=%v", reportCount, err)
	}
	mcpResourceCall(false, "upsert_file", map[string]any{"reportId": "preview-fixture", "versionNo": 3, "resourceId": "rf-mcp", "namespace": resourceNamespace, "resourcePath": "guide/readme.md", "content": "MCP guide", "expectedSourceRevision": 9}, 10)
	mcpResourceCall(true, "delete_file", map[string]any{"reportId": "preview-fixture", "versionNo": 3, "resourceId": "rf-mcp", "expectedSourceRevision": 10}, 11)
	mcpResourceCall(false, "upsert_folder", map[string]any{"reportId": "preview-fixture", "versionNo": 3, "folderId": "rd-mcp", "namespace": resourceNamespace, "rootPath": "guide", "uriPrefix": "skill://mcp-guide/", "expectedSourceRevision": 11}, 12)
	mcpResourceCall(true, "upsert_file", map[string]any{"reportId": "preview-fixture", "versionNo": 3, "resourceId": "rf-mcp-skill", "namespace": resourceNamespace, "resourcePath": "guide/SKILL.md", "content": "# MCP Skill", "expectedSourceRevision": 12}, 13)
	mcpResourceCall(false, "upsert_skill", map[string]any{"reportId": "preview-fixture", "versionNo": 3, "skillId": "sk-mcp", "folderId": "rd-mcp", "skillRoot": ".", "expectedSourceRevision": 13}, 14)
	mcpResourceCall(true, "delete_skill", map[string]any{"reportId": "preview-fixture", "versionNo": 3, "skillId": "sk-mcp", "expectedSourceRevision": 14}, 15)
	mcpResourceCall(false, "delete_folder", map[string]any{"reportId": "preview-fixture", "versionNo": 3, "folderId": "rd-mcp", "expectedSourceRevision": 15}, 16)
	mcpResourceCall(true, "delete_file", map[string]any{"reportId": "preview-fixture", "versionNo": 3, "resourceId": "rf-mcp-skill", "expectedSourceRevision": 16}, 17)
	blockedBuilderRequest := request("/v1/studio/sdk/versions.builder", `{"reportId":"preview-fixture","versionNo":3,"command":{"expectedSourceRevision":17,"operation":{"type":"inspect"}}}`)
	blockedBuilderRequest.Header.Set("X-Studio-Namespace", wrongNamespaceID)
	blockedBuilderResponse := httptest.NewRecorder()
	server.ServeHTTP(blockedBuilderResponse, blockedBuilderRequest)
	if blockedBuilderResponse.Code < 400 || blockedBuilderResponse.Code >= 500 {
		t.Fatalf("cross-namespace builder status=%d body=%s", blockedBuilderResponse.Code, blockedBuilderResponse.Body.String())
	}

	blockedBuilderStatus, blockedBuilderBody := mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.versions.builder", map[string]any{"namespaceId": wrongNamespaceID, "reportId": "preview-fixture", "versionNo": 3, "command": map[string]any{"expectedSourceRevision": 17, "operation": map[string]any{"type": "inspect"}}})
	if blockedBuilderStatus != http.StatusOK || !bytes.Contains(blockedBuilderBody, []byte(`"isError":true`)) {
		t.Fatalf("cross-namespace MCP builder status=%d body=%s", blockedBuilderStatus, blockedBuilderBody)
	}

	builderInspect := httptest.NewRecorder()
	builderInspectRequest := request("/v1/studio/sdk/versions.builder", `{"reportId":"preview-fixture","versionNo":3,"command":{"expectedSourceRevision":17,"operation":{"type":"inspect"}}}`)
	builderInspectRequest.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	server.ServeHTTP(builderInspect, builderInspectRequest)
	if builderInspect.Code != http.StatusOK || !bytes.Contains(builderInspect.Body.Bytes(), []byte(`"inspection"`)) || bytes.Contains(builderInspect.Body.Bytes(), []byte(dsn)) {
		t.Fatalf("native builder inspect status=%d body=%s", builderInspect.Code, builderInspect.Body.String())
	}
	var reportEtag int64
	if err = store.QueryRowContext(ctx, `SELECT etag FROM components WHERE id='preview-fixture'`).Scan(&reportEtag); err != nil {
		t.Fatal(err)
	}
	builderChange := httptest.NewRecorder()
	server.ServeHTTP(builderChange, request("/v1/studio/sdk/versions.builder", `{"reportId":"preview-fixture","versionNo":3,"command":{"expectedSourceRevision":17,"operation":{"type":"setPackage","package":{"expected":"github.com/viant/datly-studio/dynamic/preview_fixture","path":"client-untrusted"}}}}`))
	var changed sdk.ReaderBuilderResult
	if builderChange.Code != http.StatusOK || json.Unmarshal(builderChange.Body.Bytes(), &changed) != nil || !changed.Applied || changed.Inspection == nil || changed.Inspection.Version == nil || changed.Inspection.Version.SourceRevision != 18 {
		t.Fatalf("native builder mutation status=%d body=%s", builderChange.Code, builderChange.Body.String())
	}
	var persistedRevision, persistedEtag int64
	if err = store.QueryRowContext(ctx, `SELECT source_revision FROM component_versions WHERE report_id='preview-fixture' AND version_no=3`).Scan(&persistedRevision); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRowContext(ctx, `SELECT etag FROM components WHERE id='preview-fixture'`).Scan(&persistedEtag); err != nil {
		t.Fatal(err)
	}
	if persistedRevision != 18 || persistedEtag != reportEtag+1 {
		t.Fatalf("native builder persistence revision=%d etag=%d wantEtag=%d", persistedRevision, persistedEtag, reportEtag+1)
	}
	staleBuilder := httptest.NewRecorder()
	server.ServeHTTP(staleBuilder, request("/v1/studio/sdk/versions.builder", `{"reportId":"preview-fixture","versionNo":3,"command":{"expectedSourceRevision":17,"operation":{"type":"inspect"}}}`))
	if staleBuilder.Code != http.StatusConflict || !bytes.Contains(staleBuilder.Body.Bytes(), []byte(`"field":"sourceRevision"`)) {
		t.Fatalf("stale native builder status=%d body=%s", staleBuilder.Code, staleBuilder.Body.String())
	}
	for _, viaBFF := range []bool{false, true} {
		endpoint, bearer := staticMCP, "Bearer "+token
		var cookie *http.Cookie
		if viaBFF {
			endpoint, bearer = bff.URL+"/v1/studio/sdk-mcp/mcp", ""
			cookie = &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}
		}
		callStatus, callBody := mcpCall(endpoint, bearer, cookie, "tools/call", "studio.sdk.versions.builder", map[string]any{
			"reportId": "preview-fixture", "versionNo": 3,
			"command": map[string]any{"expectedSourceRevision": 18, "operation": map[string]any{"type": "inspect"}},
		})
		if callStatus != http.StatusOK || bytes.Contains(callBody, []byte(`"isError":true`)) || !bytes.Contains(callBody, []byte(`"inspection"`)) {
			t.Fatalf("native builder MCP viaBFF=%t status=%d body=%s", viaBFF, callStatus, callBody)
		}
	}
	var reloadedGenerations []int64
	var rejectReload atomic.Bool
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Studio-Runtime-Token") != "static-test-admin-token" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /_studio/status":
			_, _ = w.Write([]byte(`{"authenticationMode":"authenticated","status":"ready","revision":42}`))
		case "POST /_studio/reload":
			if rejectReload.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			var body struct {
				Generation int64 `json:"generation"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Generation <= 42 {
				t.Errorf("invalid reload request=%+v err=%v", body, err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			reloadedGenerations = append(reloadedGenerations, body.Generation)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer admin.Close()
	t.Setenv("STUDIO_DYNAMIC_HTTP_URL", admin.URL)
	t.Setenv("STUDIO_RUNTIME_ADMIN_TOKEN", "static-test-admin-token")
	if _, err = store.ExecContext(ctx, `INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at)
		VALUES(42,'runtime-status-test','active',0,'{}','alice',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	liveRuntime := httptest.NewRecorder()
	server.ServeHTTP(liveRuntime, request("/v1/studio/sdk/runtime.status", `{}`))
	if liveRuntime.Code != http.StatusOK || !bytes.Contains(liveRuntime.Body.Bytes(), []byte(`"activeGeneration":42`)) ||
		!bytes.Contains(liveRuntime.Body.Bytes(), []byte(`"host":{"authenticationMode":"authenticated","status":"ready","revision":42`)) {
		t.Fatalf("native live runtime status=%d body=%s", liveRuntime.Code, liveRuntime.Body.String())
	}
	var publicationRevision int64
	if _, err = store.ExecContext(ctx, `UPDATE component_versions SET spec_hash=? WHERE report_id=? AND version_no=1`, strings.Repeat("a", 64), previewReportID); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRowContext(ctx, `SELECT source_revision FROM component_versions WHERE report_id=? AND version_no=1`, previewReportID).Scan(&publicationRevision); err != nil {
		t.Fatal(err)
	}
	blockedPublish := request("/v1/studio/sdk/publications.publish", fmt.Sprintf(`{"reportId":%q,"versionNo":1,"input":{"expectedSourceRevision":%d}}`, previewReportID, publicationRevision))
	blockedPublish.Header.Set("X-Studio-Namespace", wrongNamespaceID)
	blockedPublishResponse := httptest.NewRecorder()
	server.ServeHTTP(blockedPublishResponse, blockedPublish)
	if blockedPublishResponse.Code != http.StatusForbidden || len(reloadedGenerations) != 0 {
		t.Fatalf("cross-namespace publish status=%d reloads=%d", blockedPublishResponse.Code, len(reloadedGenerations))
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_publications WHERE report_id=?`, previewReportID).Scan(&reportCount); err != nil || reportCount != 0 {
		t.Fatalf("denied publication created state: count=%d err=%v", reportCount, err)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.publications.publish", map[string]any{"namespaceId": wrongNamespaceID, "reportId": previewReportID, "versionNo": 1, "input": map[string]any{"expectedSourceRevision": publicationRevision}})
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"isError":true`)) || len(reloadedGenerations) != 0 {
		t.Fatalf("cross-namespace MCP publish status=%d reloads=%d", status, len(reloadedGenerations))
	}

	publish := httptest.NewRecorder()
	publishRequest := request("/v1/studio/sdk/publications.publish", fmt.Sprintf(`{"reportId":%q,"versionNo":1,"input":{"expectedSourceRevision":%d}}`, previewReportID, publicationRevision))
	publishRequest.Header.Set("X-Studio-Namespace", namespaceaccess.ID("alice", "production.audit"))
	server.ServeHTTP(publish, publishRequest)
	if publish.Code != http.StatusOK || !bytes.Contains(publish.Body.Bytes(), []byte(`"status":"active"`)) || len(reloadedGenerations) != 1 {
		t.Fatalf("native publish status=%d body=%s reloads=%v", publish.Code, publish.Body.String(), reloadedGenerations)
	}
	deniedPublish := request("/v1/studio/sdk/publications.publish", fmt.Sprintf(`{"reportId":%q,"versionNo":1,"input":{"expectedSourceRevision":%d}}`, previewReportID, publicationRevision))
	deniedPublish.Header.Set("Authorization", "Bearer "+bobToken)
	deniedPublishResponse := httptest.NewRecorder()
	server.ServeHTTP(deniedPublishResponse, deniedPublish)
	if deniedPublishResponse.Code != http.StatusForbidden || len(reloadedGenerations) != 1 {
		t.Fatalf("non-publisher publication status=%d body=%s reloads=%v", deniedPublishResponse.Code, deniedPublishResponse.Body.String(), reloadedGenerations)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.publications.rollback", map[string]any{
		"namespaceId": namespaceaccess.ID("alice", "production.audit"), "reportId": previewReportID, "versionNo": 1, "input": map[string]any{"expectedSourceRevision": publicationRevision},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"active"`)) || len(reloadedGenerations) != 2 {
		t.Fatalf("native rollback MCP status=%d body=%s reloads=%v", status, body, reloadedGenerations)
	}
	var activeGeneration int64
	if err = store.QueryRowContext(ctx, `SELECT active_generation FROM component_publications WHERE report_id=?`, previewReportID).Scan(&activeGeneration); err != nil || activeGeneration != reloadedGenerations[1] {
		t.Fatalf("rollback active generation=%d err=%v reloads=%v", activeGeneration, err, reloadedGenerations)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.publications.unpublish", map[string]any{
		"reportId": previewReportID, "input": map[string]any{"expectedActiveGeneration": activeGeneration},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"unpublished"`)) || len(reloadedGenerations) != 3 {
		t.Fatalf("native BFF unpublish MCP status=%d body=%s reloads=%v", status, body, reloadedGenerations)
	}
	status, body = mcpCall(staticMCP, "Bearer "+token, nil, "tools/call", "studio.sdk.publications.publish", map[string]any{
		"reportId": previewReportID, "versionNo": 1, "input": map[string]any{"expectedSourceRevision": publicationRevision},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"active"`)) || len(reloadedGenerations) != 4 {
		t.Fatalf("native publish MCP status=%d body=%s reloads=%v", status, body, reloadedGenerations)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: id}, "tools/call", "studio.sdk.publications.rollback", map[string]any{
		"reportId": previewReportID, "versionNo": 1, "input": map[string]any{"expectedSourceRevision": publicationRevision},
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"status":"active"`)) || len(reloadedGenerations) != 5 {
		t.Fatalf("native BFF rollback MCP status=%d body=%s reloads=%v", status, body, reloadedGenerations)
	}
	if err = store.QueryRowContext(ctx, `SELECT active_generation FROM component_publications WHERE report_id=?`, previewReportID).Scan(&activeGeneration); err != nil || activeGeneration != reloadedGenerations[4] {
		t.Fatalf("BFF rollback active generation=%d err=%v reloads=%v", activeGeneration, err, reloadedGenerations)
	}
	unpublishHTTP := httptest.NewRecorder()
	server.ServeHTTP(unpublishHTTP, request("/v1/studio/sdk/publications.unpublish", fmt.Sprintf(`{"reportId":%q,"input":{"expectedActiveGeneration":%d}}`, previewReportID, activeGeneration)))
	if unpublishHTTP.Code != http.StatusOK || !bytes.Contains(unpublishHTTP.Body.Bytes(), []byte(`"status":"unpublished"`)) || len(reloadedGenerations) != 6 {
		t.Fatalf("native unpublish HTTP status=%d body=%s reloads=%v", unpublishHTTP.Code, unpublishHTTP.Body.String(), reloadedGenerations)
	}
	stalePublication := httptest.NewRecorder()
	server.ServeHTTP(stalePublication, request("/v1/studio/sdk/publications.publish", fmt.Sprintf(`{"reportId":%q,"versionNo":1,"input":{"expectedSourceRevision":%d}}`, previewReportID, publicationRevision+1)))
	if stalePublication.Code != http.StatusConflict || len(reloadedGenerations) != 6 {
		t.Fatalf("stale native publication status=%d body=%s reloads=%v", stalePublication.Code, stalePublication.Body.String(), reloadedGenerations)
	}
	rejectReload.Store(true)
	failedReload := httptest.NewRecorder()
	server.ServeHTTP(failedReload, request("/v1/studio/sdk/publications.publish", fmt.Sprintf(`{"reportId":%q,"versionNo":1,"input":{"expectedSourceRevision":%d}}`, previewReportID, publicationRevision)))
	if failedReload.Code != http.StatusServiceUnavailable {
		t.Fatalf("failed reload publication status=%d body=%s", failedReload.Code, failedReload.Body.String())
	}
	if err = store.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_publications WHERE report_id=? AND publication_status='active'`, previewReportID).Scan(&reportCount); err != nil || reportCount != 0 {
		t.Fatalf("failed reload left an active publication count=%d err=%v", reportCount, err)
	}
	aclPublicKey := filepath.Join(t.TempDir(), "acl-public.pem")
	if err = os.WriteFile(aclPublicKey, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUDIO_ACCESS_ISSUER", "https://idp.viantinc.com")
	t.Setenv("STUDIO_ACCESS_AUDIENCE", "datly-studio-web")
	t.Setenv("STUDIO_ACCESS_PUBLIC_KEY_FILE", aclPublicKey)
	aclResource := resourceaccess.Resource{Kind: "component", ID: previewReportID, Tenant: "one", Version: "1"}
	policyStore := &accessstore.Store{DB: store}
	if _, err = policyStore.Provision(ctx, resourceaccess.Document{Resource: aclResource, Policies: map[string]resourceaccess.Policy{
		"viewAccess":   {Mode: "protected", Rule: &resourceaccess.Rule{Kind: "subject", Value: "alice"}},
		"manageAccess": {Mode: "protected", Rule: &resourceaccess.Rule{Kind: "subject", Value: "alice"}},
	}}, "bootstrap"); err != nil {
		t.Fatal(err)
	}
	defer policyStore.Close(ctx)
	aclToken := func(subject string) string {
		t.Helper()
		value, signErr := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
			"sub": subject, "iss": "https://idp.viantinc.com", "aud": "datly-studio-web", "tenant": "one",
			"roles": []string{"publisher"}, "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Add(-time.Minute).Unix(),
		}).SignedString(key)
		if signErr != nil {
			t.Fatal(signErr)
		}
		return value
	}
	aliceACLToken := aclToken("alice")
	aclRequest := func(path, body, bearer string) *http.Request {
		req := request(path, body)
		req.Header.Set("Authorization", "Bearer "+bearer)
		return req
	}
	resourceBody := fmt.Sprintf(`{"kind":"component","id":%q,"tenant":"one","version":"1"}`, previewReportID)
	accessGet := httptest.NewRecorder()
	server.ServeHTTP(accessGet, aclRequest("/v1/studio/sdk/access.get", resourceBody, aliceACLToken))
	if accessGet.Code != http.StatusOK || !bytes.Contains(accessGet.Body.Bytes(), []byte(`"revision":1`)) || !bytes.Contains(accessGet.Body.Bytes(), []byte(`"manageAccess"`)) {
		t.Fatalf("native access get status=%d body=%s", accessGet.Code, accessGet.Body.String())
	}
	accessContext := httptest.NewRecorder()
	server.ServeHTTP(accessContext, aclRequest("/v1/studio/sdk/access.context", resourceBody, aliceACLToken))
	if accessContext.Code != http.StatusOK || !bytes.Contains(accessContext.Body.Bytes(), []byte(`"canManage":true`)) {
		t.Fatalf("native access context status=%d body=%s", accessContext.Code, accessContext.Body.String())
	}
	// Studio does not interpret a remote identity service's response schema.
	// An endpoint without a host-owned adapter is invalid configuration on
	// both HTTP and MCP, including for an otherwise verified resource owner.
	t.Setenv("STUDIO_ACCESS_USER_INFO_URL", "https://userinfo.example.test/identity")
	missingAdapter := httptest.NewRecorder()
	server.ServeHTTP(missingAdapter, aclRequest("/v1/studio/sdk/access.get", resourceBody, aliceACLToken))
	if missingAdapter.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing identity adapter HTTP status=%d body=%s", missingAdapter.Code, missingAdapter.Body.String())
	}
	status, body = mcpCall(staticMCP, "Bearer "+aliceACLToken, nil, "tools/call", "studio.sdk.access.get", map[string]any{
		"kind": "component", "id": aclResource.ID, "tenant": aclResource.Tenant, "version": aclResource.Version,
	})
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"isError":true`)) {
		t.Fatalf("missing identity adapter MCP status=%d body=%s", status, body)
	}
	t.Setenv("STUDIO_ACCESS_USER_INFO_URL", "")
	t.Setenv("STUDIO_ACCESS_ISSUER", "https://idp.viantinc.com")
	t.Setenv("STUDIO_ACCESS_AUDIENCE", "datly-studio-web")
	deniedAccess := httptest.NewRecorder()
	server.ServeHTTP(deniedAccess, aclRequest("/v1/studio/sdk/access.get", resourceBody, aclToken("bob")))
	if deniedAccess.Code != http.StatusForbidden {
		t.Fatalf("non-owner access get status=%d body=%s", deniedAccess.Code, deniedAccess.Body.String())
	}
	wrongAudienceToken, signErr := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"sub": "alice", "iss": "https://idp.viantinc.com", "aud": "other-audience", "tenant": "one",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Add(-time.Minute).Unix(),
	}).SignedString(key)
	if signErr != nil {
		t.Fatal(signErr)
	}
	wrongAudience := httptest.NewRecorder()
	server.ServeHTTP(wrongAudience, aclRequest("/v1/studio/sdk/access.get", resourceBody, wrongAudienceToken))
	if wrongAudience.Code != http.StatusUnauthorized {
		t.Fatalf("wrong ACL audience status=%d body=%s", wrongAudience.Code, wrongAudience.Body.String())
	}
	accessReplaceBody := fmt.Sprintf(`{"resource":%s,"revision":1,"policies":{"viewAccess":{"mode":"protected","rule":{"kind":"subject","value":"alice"}},"manageAccess":{"mode":"protected","rule":{"kind":"subject","value":"alice"}}}}`, resourceBody)
	accessReplace := httptest.NewRecorder()
	server.ServeHTTP(accessReplace, aclRequest("/v1/studio/sdk/access.replace", accessReplaceBody, aliceACLToken))
	if accessReplace.Code != http.StatusOK || !bytes.Contains(accessReplace.Body.Bytes(), []byte(`"revision":2`)) {
		t.Fatalf("native access replace status=%d body=%s", accessReplace.Code, accessReplace.Body.String())
	}
	staleAccess := httptest.NewRecorder()
	server.ServeHTTP(staleAccess, aclRequest("/v1/studio/sdk/access.replace", accessReplaceBody, aliceACLToken))
	if staleAccess.Code != http.StatusConflict {
		t.Fatalf("stale native access replace status=%d body=%s", staleAccess.Code, staleAccess.Body.String())
	}
	status, body = mcpCall(staticMCP, "Bearer "+aliceACLToken, nil, "tools/call", "studio.sdk.access.get", map[string]any{
		"kind": "component", "id": previewReportID, "tenant": "one", "version": "1",
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"revision":2`)) {
		t.Fatalf("native access MCP get status=%d body=%s", status, body)
	}
	aclSession, _, _, err := sessions.Exchange(ctx, aliceACLToken)
	if err != nil {
		t.Fatal(err)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: aclSession}, "tools/call", "studio.sdk.access.context", map[string]any{
		"kind": "component", "id": previewReportID, "tenant": "one", "version": "1",
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"canManage":true`)) {
		t.Fatalf("BFF access MCP context status=%d body=%s", status, body)
	}
	aclMCPResource := map[string]any{"kind": "component", "id": previewReportID, "tenant": "one", "version": "1"}
	aclMCPPolicies := map[string]any{
		"viewAccess":   map[string]any{"mode": "protected", "rule": map[string]any{"kind": "subject", "value": "alice"}},
		"manageAccess": map[string]any{"mode": "protected", "rule": map[string]any{"kind": "subject", "value": "alice"}},
	}
	status, body = mcpCall(staticMCP, "Bearer "+aliceACLToken, nil, "tools/call", "studio.sdk.access.replace", map[string]any{
		"resource": aclMCPResource, "revision": 2, "policies": aclMCPPolicies,
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"revision":3`)) {
		t.Fatalf("native access MCP replace status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: aclSession}, "tools/call", "studio.sdk.access.replace", map[string]any{
		"resource": aclMCPResource, "revision": 3, "policies": aclMCPPolicies,
	})
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"revision":4`)) {
		t.Fatalf("BFF access MCP replace status=%d body=%s", status, body)
	}
	status, body = mcpCall(bff.URL+"/v1/studio/sdk-mcp/mcp", "", &http.Cookie{Name: bffauth.DefaultCookieName, Value: aclSession}, "tools/call", "studio.sdk.access.get", aclMCPResource)
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"revision":4`)) {
		t.Fatalf("BFF access MCP get status=%d body=%s", status, body)
	}
	status, body = mcpCall(staticMCP, "Bearer "+aliceACLToken, nil, "tools/call", "studio.sdk.access.context", aclMCPResource)
	if status != http.StatusOK || bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"canManage":true`)) {
		t.Fatalf("native access MCP context status=%d body=%s", status, body)
	}
}

func sdkToolForPath(path string) string {
	if strings.HasPrefix(path, "/v1/authz/sdk/") {
		return "authz.sdk." + strings.TrimPrefix(path, "/v1/authz/sdk/")
	}
	return "studio.sdk." + strings.TrimPrefix(path, "/v1/studio/sdk/")
}
