package host

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"github.com/viant/authz"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	accessstore "github.com/viant/authz/datly/store/sql"
	"github.com/viant/datly-studio/schema"
	mcpschema "github.com/viant/mcp-protocol/schema"
	mcpprotocol "github.com/viant/mcp/server"
	_ "modernc.org/sqlite"
)

func TestDynamicHostServesPublishedHTTPAndDedicatedMCP(t *testing.T) {
	t.Run("legacy", func(t *testing.T) { testDynamicHost(t, false, false) })
	t.Run("generic-access", func(t *testing.T) { testDynamicHost(t, true, false) })
	t.Run("identity-token", func(t *testing.T) { testDynamicHost(t, false, true) })
}

func testDynamicHost(t *testing.T, generic, identityRequired bool) {
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
	defer source.Close()
	if _, err = source.Exec(`CREATE TABLE records(id INTEGER PRIMARY KEY,name TEXT); INSERT INTO records VALUES(1,'ready')`); err != nil {
		t.Fatal(err)
	}
	lookupDSN := "file:" + filepath.Join(root, "lookup.db")
	lookup, err := sql.Open("sqlite", lookupDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer lookup.Close()
	if _, err = lookup.Exec(`CREATE TABLE labels(id INTEGER PRIMARY KEY,label TEXT); INSERT INTO labels VALUES(1,'primary')`); err != nil {
		t.Fatal(err)
	}
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
	if _, err = studio.Exec(`INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at) VALUES('lookup','sqlite',?,'owner','active','{}',1,?,?)`, lookupDSN, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err = studio.Exec(`INSERT INTO components(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('records','records','Records','owner','active','source','example.com/runtime/read','records',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	dql := `#package('example.com/runtime/read')
#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#setting($_ = $mcp('records.read','Read records'))
#define($_ = $Records<[]*Record>(output/view))
SELECT records.*, labels.*, type(records,'Record'), type(labels,'Label'), use_connector(labels,'lookup')
FROM (SELECT id,name FROM records) records
JOIN (SELECT id,label FROM labels) labels ON labels.id=records.id`
	if _, err = studio.Exec(`INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('records',1,'published','dql',?,?,'{}','studio.v1','hash','{}','valid','v1','studio.v1',1,'owner',?)`, dql, dql, now); err != nil {
		t.Fatal(err)
	}
	if _, err = studio.Exec(`INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at) VALUES(1,'records:1','active',1,'{}','owner',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err = studio.Exec(`INSERT INTO report_publications(report_id,active_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at) VALUES('records',1,1,1,'active','records:1','hash','owner',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	_ = studio.Close()
	runtimeConfig := Config{HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"}, Authentication: Authentication{DefaultMode: "public"}, Studio: Studio{Driver: "sqlite", DSN: studioDSN}, Admin: Admin{Token: "test-token"}, RootDir: root}
	noCredentials := false
	runtimeConfig.MCP.CORS = &mcpprotocol.Cors{AllowOrigins: []string{"https://studio.example.com"},
		AllowMethods: []string{"POST", "OPTIONS"}, AllowHeaders: []string{"Content-Type", "Authorization", "Mcp-Protocol-Version", "Mcp-Method"},
		AllowCredentials: &noCredentials}
	runtimeConfig.HTTP.CORS = &mcpprotocol.Cors{AllowOrigins: []string{"https://studio.example.com"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"}, AllowHeaders: []string{"Content-Type", "Authorization"},
		AllowCredentials: &noCredentials}
	var identityKey *rsa.PrivateKey
	if identityRequired {
		identityKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		keyID := "studio-identity-test"
		jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			n := base64.RawURLEncoding.EncodeToString(identityKey.PublicKey.N.Bytes())
			e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(identityKey.PublicKey.E)).Bytes())
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kty": "RSA", "kid": keyID, "use": "sig", "alg": "RS256", "n": n, "e": e}}})
		}))
		defer jwks.Close()
		runtimeConfig.Authentication = Authentication{DefaultMode: "required", CertURL: jwks.URL,
			Issuer: "https://identity.example", Audience: "studio-web"}
	}
	if generic {
		key, e := rsa.GenerateKey(rand.Reader, 2048)
		if e != nil {
			t.Fatal(e)
		}
		encoded, e := x509.MarshalPKIXPublicKey(&key.PublicKey)
		if e != nil {
			t.Fatal(e)
		}
		keyFile := filepath.Join(root, "access.pem")
		if e = os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded}), 0600); e != nil {
			t.Fatal(e)
		}
		runtimeConfig.Access = &ResourceAccessConfig{Tenant: "*", Issuer: "https://access.example", Audience: "runtime", PublicKeyFile: keyFile}
	}
	service, err := New(ctx, runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	if generic {
		_, err = service.resourceAccess.Store.(*accessstore.Store).Provision(ctx, authz.Document{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: "*"}, Policies: map[string]authz.Policy{"execute": {Mode: "public"}}}, "bootstrap")
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = service.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Close(closeCtx)
	})
	httpAddress, mcpAddress := service.Addresses()
	if httpAddress == "" || mcpAddress == "" || httpAddress == mcpAddress {
		t.Fatalf("addresses=%q,%q", httpAddress, mcpAddress)
	}
	if identityRequired {
		verifyPublishedIdentityRuntime(t, httpAddress, mcpAddress, identityKey)
		return
	}
	preflight, err := http.NewRequest(http.MethodOptions, "http://"+mcpAddress+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	preflight.Header.Set("Origin", "https://studio.example.com")
	preflight.Header.Set("Access-Control-Request-Method", "POST")
	preflight.Header.Set("Access-Control-Request-Headers", "authorization,mcp-protocol-version,mcp-method")
	preflightResponse, err := http.DefaultClient.Do(preflight)
	if err != nil {
		t.Fatal(err)
	}
	preflightResponse.Body.Close()
	if preflightResponse.StatusCode != http.StatusNoContent || preflightResponse.Header.Get("Access-Control-Allow-Origin") != "https://studio.example.com" ||
		preflightResponse.Header.Get("Access-Control-Allow-Credentials") == "true" {
		t.Fatalf("direct MCP preflight status=%d headers=%v", preflightResponse.StatusCode, preflightResponse.Header)
	}
	for _, change := range []func(*http.Request){
		func(r *http.Request) { r.Header.Set("Origin", "https://other.example.com") },
		func(r *http.Request) { r.Header.Set("Access-Control-Request-Method", "DELETE") },
		func(r *http.Request) { r.Header.Set("Access-Control-Request-Headers", "x-admin-token") },
	} {
		denied, err := http.NewRequest(http.MethodOptions, "http://"+mcpAddress+"/mcp", nil)
		if err != nil {
			t.Fatal(err)
		}
		denied.Header = preflight.Header.Clone()
		change(denied)
		result, err := http.DefaultClient.Do(denied)
		if err != nil {
			t.Fatal(err)
		}
		result.Body.Close()
		if result.StatusCode != http.StatusForbidden || result.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("unsafe MCP preflight status=%d headers=%v", result.StatusCode, result.Header)
		}
	}
	response, err := http.Get("http://" + httpAddress + "/records")
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(payload), "ready") || !strings.Contains(string(payload), "primary") {
		t.Fatalf("status=%d body=%s", response.StatusCode, payload)
	}
	mutationDQL := strings.Replace(dql, "$route('/records','GET')", "$route('/records','PATCH')", 1)
	if mutationDQL == dql {
		t.Fatal("mutation route fixture was not changed")
	}
	verificationDB, openErr := sql.Open("sqlite", studioDSN)
	if openErr != nil {
		t.Fatal(openErr)
	}
	defer verificationDB.Close()
	if _, err = verificationDB.ExecContext(ctx, `UPDATE report_versions SET generated_dql=?, authored_dql=? WHERE report_id='records' AND version_no=1`, mutationDQL, mutationDQL); err != nil {
		t.Fatal(err)
	}
	if reloadErr := service.Reload(ctx, 0); reloadErr == nil || !strings.Contains(reloadErr.Error(), "dynamic components support readers only") {
		t.Fatalf("mutation-route reload error=%v", reloadErr)
	}
	if service.manager.Revision() != 1 {
		t.Fatalf("rejected mutation changed serving revision to %d", service.manager.Revision())
	}
	stillServing, serveErr := http.Get("http://" + httpAddress + "/records")
	if serveErr != nil {
		t.Fatal(serveErr)
	}
	stillPayload, _ := io.ReadAll(stillServing.Body)
	stillServing.Body.Close()
	if stillServing.StatusCode != http.StatusOK || !strings.Contains(string(stillPayload), "ready") {
		t.Fatalf("rejected mutation disrupted reader: status=%d body=%s", stillServing.StatusCode, stillPayload)
	}
	if _, err = verificationDB.ExecContext(ctx, `UPDATE report_versions SET generated_dql=?, authored_dql=? WHERE report_id='records' AND version_no=1`, dql, dql); err != nil {
		t.Fatal(err)
	}
	mcpResponse, err := http.Get("http://" + mcpAddress + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	mcpResponse.Body.Close()
	if mcpResponse.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("MCP status=%d", mcpResponse.StatusCode)
	}
	statusRequest, err := http.NewRequest(http.MethodGet, "http://"+httpAddress+"/_studio/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	statusRequest.Header.Set("X-Studio-Runtime-Token", "test-token")
	statusResponse, err := http.DefaultClient.Do(statusRequest)
	if err != nil {
		t.Fatal(err)
	}
	statusPayload, _ := io.ReadAll(statusResponse.Body)
	statusResponse.Body.Close()
	if statusResponse.StatusCode != http.StatusOK || !strings.Contains(string(statusPayload), `"status":"ready"`) || !strings.Contains(string(statusPayload), `"revision":1`) {
		t.Fatalf("runtime status=%d body=%s", statusResponse.StatusCode, statusPayload)
	}
	admin, err := http.NewRequest(http.MethodPost, "http://"+httpAddress+"/_studio/reload", bytes.NewBufferString(`{"generation":2}`))
	if err != nil {
		t.Fatal(err)
	}
	admin.Header.Set("X-Studio-Runtime-Token", "test-token")
	adminResponse, err := http.DefaultClient.Do(admin)
	if err != nil {
		t.Fatal(err)
	}
	adminResponse.Body.Close()
	if adminResponse.StatusCode != http.StatusOK {
		t.Fatalf("admin reload status=%d", adminResponse.StatusCode)
	}
	unauthorized, err := http.Post("http://"+httpAddress+"/_studio/reload", "application/json", bytes.NewBufferString(`{"generation":3}`))
	if err != nil {
		t.Fatal(err)
	}
	unauthorized.Body.Close()
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("admin auth status=%d", unauthorized.StatusCode)
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	if err = service.Close(closeCtx); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	service, err = New(ctx, runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Start(ctx); err != nil {
		t.Fatal(err)
	}
	restartedHTTP, restartedMCP := service.Addresses()
	if restartedHTTP == "" || restartedMCP == "" || restartedHTTP == restartedMCP {
		t.Fatalf("restarted addresses=%q,%q", restartedHTTP, restartedMCP)
	}
	restartedResponse, err := http.Get("http://" + restartedHTTP + "/records")
	if err != nil {
		t.Fatal(err)
	}
	restartedPayload, _ := io.ReadAll(restartedResponse.Body)
	restartedResponse.Body.Close()
	if restartedResponse.StatusCode != http.StatusOK || !strings.Contains(string(restartedPayload), "ready") || !strings.Contains(string(restartedPayload), "primary") {
		t.Fatalf("restarted status=%d body=%s", restartedResponse.StatusCode, restartedPayload)
	}
	if generic {
		store := service.resourceAccess.Store.(*accessstore.Store)
		r := authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: "*"}
		doc, e := store.Get(ctx, r)
		if e != nil {
			t.Fatal(e)
		}
		doc.Policies["execute"] = authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
		if _, e = store.Replace(ctx, doc, doc.Revision, "revoke-public"); e != nil {
			t.Fatal(e)
		}
		denied, e := http.Get("http://" + restartedHTTP + "/records?roles=reader&subject=owner")
		if e != nil {
			t.Fatal(e)
		}
		body, _ := io.ReadAll(denied.Body)
		denied.Body.Close()
		if denied.StatusCode != http.StatusForbidden || strings.Contains(string(body), "ready") {
			t.Fatalf("runtime policy bypass: %d %s", denied.StatusCode, body)
		}
	}
}

func verifyPublishedIdentityRuntime(t *testing.T, httpAddress, mcpAddress string, key *rsa.PrivateKey) {
	t.Helper()
	tools, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": mcpschema.MethodToolsList,
		"params": map[string]any{"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": mcpschema.LatestProtocolVersion,
			"io.modelcontextprotocol/clientCapabilities": map[string]any{}}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []struct {
		name, url, method string
		body              []byte
	}{
		{"HTTP", "http://" + httpAddress + "/records", http.MethodGet, nil},
		{"MCP", "http://" + mcpAddress + "/mcp", http.MethodPost, tools},
	} {
		t.Run("missing bearer "+target.name, func(t *testing.T) {
			request, err := http.NewRequest(target.method, target.url, bytes.NewReader(target.body))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Origin", "https://studio.example.com")
			if target.name == "MCP" {
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Accept", "application/json, text/event-stream")
				request.Header.Set(mcpschema.HeaderProtocolVersion, mcpschema.LatestProtocolVersion)
				request.Header.Set(mcpschema.HeaderMethod, mcpschema.MethodToolsList)
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			payload, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode != http.StatusUnauthorized ||
				response.Header.Get("Access-Control-Allow-Origin") != "https://studio.example.com" ||
				response.Header.Get("Access-Control-Allow-Credentials") == "true" ||
				len(response.Header.Values("Set-Cookie")) != 0 || strings.Contains(string(payload), "ready") {
				t.Fatalf("missing bearer %s status=%d headers=%v body=%s", target.name, response.StatusCode, response.Header, payload)
			}
		})
	}
	for _, candidate := range []struct {
		name, issuer, audience, subject string
		authorized                      bool
	}{
		{"Studio ID token", "https://identity.example", "studio-web", "owner", true},
		{"other issuer", "https://other.example", "studio-web", "owner", false},
		{"other audience", "https://identity.example", "other-client", "owner", false},
		{"missing subject", "https://identity.example", "studio-web", "", false},
	} {
		t.Run(candidate.name, func(t *testing.T) {
			claims := jwtv5.MapClaims{"iss": candidate.issuer, "aud": candidate.audience, "sub": candidate.subject,
				"exp": time.Now().Add(time.Hour).Unix()}
			token := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, claims)
			token.Header["kid"] = "studio-identity-test"
			signed, err := token.SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			for _, target := range []struct {
				name, url, method string
				body              []byte
			}{
				{"HTTP", "http://" + httpAddress + "/records", http.MethodGet, nil},
				{"MCP", "http://" + mcpAddress + "/mcp", http.MethodPost, tools},
			} {
				t.Run(target.name, func(t *testing.T) {
					request, err := http.NewRequest(target.method, target.url, bytes.NewReader(target.body))
					if err != nil {
						t.Fatal(err)
					}
					request.Header.Set("Authorization", "Bearer "+signed)
					request.Header.Set("Origin", "https://studio.example.com")
					if target.name == "MCP" {
						request.Header.Set("Content-Type", "application/json")
						request.Header.Set("Accept", "application/json, text/event-stream")
						request.Header.Set(mcpschema.HeaderProtocolVersion, mcpschema.LatestProtocolVersion)
						request.Header.Set(mcpschema.HeaderMethod, mcpschema.MethodToolsList)
					}
					response, err := http.DefaultClient.Do(request)
					if err != nil {
						t.Fatal(err)
					}
					payload, _ := io.ReadAll(response.Body)
					response.Body.Close()
					want := http.StatusUnauthorized
					if candidate.authorized {
						want = http.StatusOK
					}
					if response.StatusCode != want || response.Header.Get("Access-Control-Allow-Origin") != "https://studio.example.com" {
						t.Fatalf("status=%d want=%d origin=%q body=%s", response.StatusCode, want,
							response.Header.Get("Access-Control-Allow-Origin"), payload)
					}
					if target.name == "MCP" && len(response.Header.Values("Set-Cookie")) != 0 {
						t.Fatalf("direct MCP response unexpectedly set cookies: %v", response.Header.Values("Set-Cookie"))
					}
					if candidate.authorized && target.name == "HTTP" && !strings.Contains(string(payload), "ready") {
						t.Fatalf("published reader did not execute: %s", payload)
					}
					if candidate.authorized && target.name == "MCP" && !strings.Contains(string(payload), "records.read") {
						t.Fatalf("published reader tool was not listed: %s", payload)
					}
				})
			}
		})
	}
	valid := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, jwtv5.MapClaims{"iss": "https://identity.example",
		"aud": "studio-web", "sub": "owner", "exp": time.Now().Add(time.Hour).Unix()})
	valid.Header["kid"] = "studio-identity-test"
	signed, err := valid.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	call, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 2, "method": mcpschema.MethodToolsCall,
		"params": map[string]any{"name": "records.read", "arguments": map[string]any{},
			"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": mcpschema.LatestProtocolVersion,
				"io.modelcontextprotocol/clientCapabilities": map[string]any{}}}})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, "http://"+mcpAddress+"/mcp", bytes.NewReader(call))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+signed)
	request.Header.Set("Origin", "https://studio.example.com")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set(mcpschema.HeaderProtocolVersion, mcpschema.LatestProtocolVersion)
	request.Header.Set(mcpschema.HeaderMethod, mcpschema.MethodToolsCall)
	request.Header.Set("Mcp-Name", "records.read")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Access-Control-Allow-Origin") != "https://studio.example.com" ||
		!strings.Contains(string(payload), "ready") {
		t.Fatalf("published ID-token MCP call status=%d origin=%q body=%s", response.StatusCode,
			response.Header.Get("Access-Control-Allow-Origin"), payload)
	}
	if len(response.Header.Values("Set-Cookie")) != 0 {
		t.Fatalf("direct MCP tool unexpectedly set cookies: %v", response.Header.Values("Set-Cookie"))
	}
}
