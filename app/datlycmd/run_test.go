package datlycmd_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	datlyconfig "github.com/viant/datly/standalone/config"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/app/datlycmd"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/schema"
	datlyexec "github.com/viant/datly/exec"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	mcpschema "github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

type fixtureProvider struct{ calls atomic.Int32 }

func (p *fixtureProvider) Resolve(context.Context) (authz.Facts, error) {
	p.calls.Add(1)
	return authz.Facts{Subject: "owner", Tenant: "tenant", Issuer: "test", ValidUntil: time.Now().Add(time.Hour)}, nil
}

type startupOutput struct {
	sync.Mutex
	data string
}

func (o *startupOutput) Write(b []byte) (int, error) {
	o.Lock()
	defer o.Unlock()
	o.data += string(b)
	return len(b), nil
}
func (o *startupOutput) String() string { o.Lock(); defer o.Unlock(); return o.data }

func TestEmbeddingStartup(t *testing.T) {
	mode := os.Getenv("STUDIO_EMBEDDING_TEST")
	if mode == "" {
		for _, name := range []string{"native", "native-artifact", "native-embedded", "missing", "duplicate", "late", "existing"} {
			t.Run(name, func(t *testing.T) {
				c := exec.Command(os.Args[0], "-test.run=^TestEmbeddingStartup$", "-test.v")
				c.Env = append(os.Environ(), "STUDIO_EMBEDDING_TEST="+name)
				out, err := c.CombinedOutput()
				if err != nil {
					t.Fatalf("%v\n%s", err, out)
				}
				t.Log(string(out))
			})
		}
		return
	}
	t.Setenv("STUDIO_ACCESS_USER_INFO_URL", "https://fixture.invalid/userinfo")
	p := &fixtureProvider{}
	factory := func(_ context.Context, c accessprovider.Config) (authz.Provider, error) {
		if c.UserInfoURL != "https://fixture.invalid/userinfo" {
			t.Errorf("operator config=%+v", c)
		}
		return p, nil
	}
	var diagnostics bytes.Buffer
	switch mode {
	case "late":
		accessprovider.FromEnvironment(context.Background())
		if code := datlycmd.RunWithOptions(context.Background(), nil, io.Discard, &diagnostics, datlycmd.Options{AccessProviderFactory: factory}); code != 1 || !strings.Contains(diagnostics.String(), "resolved before startup") {
			t.Fatalf("code=%d: %s", code, &diagnostics)
		}
		return
	case "duplicate", "existing":
		if err := accessprovider.RegisterEnvironmentFactory(factory); err != nil {
			t.Fatal(err)
		}
		if mode == "existing" {
			if code := datlycmd.Run(context.Background(), nil, io.Discard, &diagnostics); code != 2 {
				t.Fatal(code)
			}
			got, err := accessprovider.FromEnvironment(context.Background())
			if err != nil || got != p {
				t.Fatalf("provider=%v err=%v", got, err)
			}
			return
		}
		if code := datlycmd.RunWithOptions(context.Background(), nil, io.Discard, &diagnostics, datlycmd.Options{AccessProviderFactory: factory}); code != 1 || !strings.Contains(diagnostics.String(), "already registered") {
			t.Fatalf("code=%d: %s", code, &diagnostics)
		}
		return
	}
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "studio.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO namespaces(namespace_id,name,title,owner_id,status,etag,created_at,updated_at) VALUES('fixture-id','fixture','Fixture','owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	token, err := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, jwtv5.MapClaims{"sub": "owner", "user_id": 1, "scope": "studio:read", "exp": time.Now().Add(time.Hour).Unix()}).SignedString(private)
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(dir, "key.pem")
	if err = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}), 0600); err != nil {
		t.Fatal(err)
	}
	// The original linked auth reader and namespace visibility predicate invoke the
	// injected provider while loading the original SDK namespace catalog.
	cfg := map[string]any{"BaseDir": dir, "Endpoint": map[string]any{"Address": "127.0.0.1:0"}, "Connector": "studio", "Connectors": []any{map[string]any{"Name": "studio", "Driver": "sqlite", "DSN": dbPath}, map[string]any{"Name": "authz", "AliasOf": "studio"}}, "JWTValidator": map[string]any{"RSA": []any{map[string]any{"URL": keyPath}}}, "GoBootstrap": map[string]any{"EagerComponents": true, "Packages": []string{"github.com/viant/datly-studio/studio/auth/reader", "github.com/viant/datly-studio/studio/namespaces/reader", "github.com/viant/datly-studio/studio/authorization"}}}
	raw, _ := json.Marshal(cfg)
	if mode == "native-artifact" || mode == "native-embedded" {
		cfg["MCP"] = map[string]any{"Address": "127.0.0.1:0", "Authorization": map[string]any{"Global": map[string]any{"RequiredScopes": []string{"studio:read"}, "ProtectedResourceMetadata": map[string]any{"resource": "https://fixture.invalid/studio"}}}}
		raw, _ = json.Marshal(cfg)
	}
	conf := filepath.Join(dir, "config.json")
	os.WriteFile(conf, raw, 0600)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	out, errs := &startupOutput{}, &startupOutput{}
	done := make(chan int, 1)
	options := datlycmd.Options{}
	if mode != "missing" {
		options.AccessProviderFactory = factory
	}
	if mode == "native-artifact" || mode == "native-embedded" {
		binary, err := os.Open(os.Args[0])
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.New()
		_, err = io.Copy(h, binary)
		binary.Close()
		if err != nil {
			t.Fatal(err)
		}
		options.LinkedArtifact = &datlyexec.LinkedArtifact{Revision: "owned-studio-release", ContentFingerprint: hex.EncodeToString(h.Sum(nil))}
	}
	go func() {
		if mode != "native-embedded" {
			done <- datlycmd.RunWithOptions(ctx, []string{"run", "-conf", conf}, out, errs, options)
			return
		}
		resolved, err := (datlyconfig.Loader{}).Load(ctx, conf)
		if err == nil {
			server, buildErr := datlycmd.New(ctx, resolved, options, errs)
			err = buildErr
			if err == nil {
				err = server.Serve(ctx, out)
			}
		}
		if err != nil {
			fmt.Fprintln(errs, err)
			done <- 1
			return
		}
		done <- 0
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("native command did not stop")
		}
	})
	address := ""
	for address == "" {
		s := out.String()
		if i := strings.Index(s, "HTTP listening on "); i >= 0 {
			address = strings.TrimSpace(strings.Split(s[i+len("HTTP listening on "):], "\n")[0])
			break
		}
		select {
		case code := <-done:
			done <- code
			t.Fatalf("startup code=%d: %s", code, errs.String())
		case <-ctx.Done():
			t.Fatalf("startup: %s", errs.String())
		case <-time.After(20 * time.Millisecond):
		}
	}
	req, _ := http.NewRequest("POST", "http://"+address+"/v1/studio/sdk/namespaces.list", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if mode == "missing" {
		if res.StatusCode != 403 || p.calls.Load() != 0 {
			t.Fatalf("missing adapter native HTTP=%d: %s", res.StatusCode, body)
		}
		t.Logf("missing adapter fails closed in original native reader HTTP=%d: %s", res.StatusCode, body)
		return
	}
	if res.StatusCode != 200 || !strings.Contains(string(body), "fixture") {
		t.Fatalf("native reader HTTP=%d: %s", res.StatusCode, body)
	}
	if p.calls.Load() == 0 {
		t.Fatal("original native runtime did not invoke injected provider")
	}
	t.Logf("original native reader HTTP=%d factory provider Resolve calls=%d body=%s", res.StatusCode, p.calls.Load(), body)
	if mode == "native-artifact" || mode == "native-embedded" {
		mcpAddress := ""
		for mcpAddress == "" {
			if i := strings.Index(out.String(), "MCP listening on "); i >= 0 {
				mcpAddress = strings.TrimSpace(strings.Split(out.String()[i+len("MCP listening on "):], "\n")[0])
				break
			}
			select {
			case <-ctx.Done():
				t.Fatal("MCP startup timeout")
			case <-time.After(20 * time.Millisecond):
			}
		}
		transport, err := streamable.New(ctx, "http://"+mcpAddress+"/mcp", streamable.WithHTTPClient(&http.Client{Transport: bearerTransport{token: token}}), streamable.WithStateless(), streamable.WithProtocolVersion(mcpschema.LatestProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
			var request struct {
				Method string
				Params struct{ Name string }
			}
			if err := json.Unmarshal(body, &request); err != nil {
				return err
			}
			headers.Set(mcpschema.HeaderMethod, request.Method)
			if request.Method == mcpschema.MethodToolsCall {
				headers.Set(mcpschema.HeaderName, request.Params.Name)
			}
			return nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		client := mcpclient.New("original-studio-artifact", "1", transport, mcpclient.WithProtocolVersion(mcpschema.LatestProtocolVersion))
		defer client.Close()
		if _, err = client.Initialize(ctx); err != nil {
			t.Fatal(err)
		}
		listing, err := client.ListTools(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, entry := range listing.Tools {
			if entry.Name == "studio.sdk.namespaces.list" {
				found = true
			}
		}
		if !found {
			t.Fatal("original Studio namespace tool missing")
		}
		result, err := client.CallTool(ctx, &mcpschema.CallToolRequestParams{Name: "studio.sdk.namespaces.list", Arguments: map[string]interface{}{}})
		if err != nil {
			t.Fatal(err)
		}
		payload, _ := json.Marshal(result.StructuredContent)
		if result.IsError != nil && *result.IsError || !strings.Contains(string(payload), "fixture") {
			t.Fatalf("ordinary original namespace reader did not return fixture: %s", payload)
		}
		before := p.calls.Load()
		if before == 0 {
			t.Fatal("ordinary MCP did not use configured native authorization provider")
		}
	}
}

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	copy := request.Clone(request.Context())
	copy.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(copy)
}
