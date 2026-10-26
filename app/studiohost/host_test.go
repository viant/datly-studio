package studiohost_test

import (
	"bufio"
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
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/app/hostconfig"
	"github.com/viant/datly-studio/app/studioapi"
	"github.com/viant/datly-studio/app/studiohost"
	runtimehost "github.com/viant/datly-studio/runtime/host"
	nativeauth "github.com/viant/datly/runtime/auth"
	mcpschema "github.com/viant/mcp-protocol/schema"
	"github.com/viant/scy/auth/jwt"
	xcodec "github.com/viant/xdatly/codec"
)

// Child isolation is for immutable native provider registration, not deployment:
// the child itself serves all application listeners in one process.
func TestOneProcessHost(t *testing.T) {
	if os.Getenv("STUDIO_UNIFIED_HOST_TEST") == "" && os.Getenv("STUDIO_UNIFIED_BINARY") == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestOneProcessHost$", "-test.v")
		cmd.Env = append(os.Environ(), "STUDIO_UNIFIED_HOST_TEST=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		t.Log(string(out))
		return
	}
	dir := t.TempDir()
	dsn := filepath.Join(dir, "fresh-store", "storage.db")
	db, err := sql.Open("sqlite", "file:"+dsn+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPath := filepath.Join(dir, "public.pem")
	if err := os.WriteFile(publicPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}), 0600); err != nil {
		t.Fatal(err)
	}
	document := map[string]any{"BaseDir": dir, "Endpoint": map[string]any{"Address": "127.0.0.1:0"}, "MCP": map[string]any{"Address": "127.0.0.1:0"}, "NamespaceMCP": map[string]any{"Address": "127.0.0.1:0"}, "Store": map[string]any{"Driver": "sqlite", "DSN": dsn}, "JWTValidator": map[string]any{"RSA": []any{map[string]any{"URL": publicPath}}}, "JWTClaims": map[string]any{"Issuer": "https://synthetic.example", "Audience": "studio", "RequireSubject": true}}
	// Connection export must never decrypt or read its SCY reference. This
	// deliberately absent file is unrelated to the browser's login client.
	document["Audit"] = map[string]any{"ViewerRole": "E2E_AUDIT_VIEWER"}
	document["NamespaceConnectionExport"] = map[string]any{"Provider": "synthetic", "ClientConfigURL": "file:///unreadable-synthetic-namespace-client.enc|blowfish://env/UNSET_SYNTHETIC_NAMESPACE_KEY", "Scopes": []string{"openid"}}
	document["NamespaceOAuthProviders"] = map[string]any{"synthetic": map[string]any{"Issuer": "https://synthetic.example", "Audience": "studio", "CertURL": "file://" + publicPath}}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	config, err := hostconfig.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	bearer, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"iss": "https://synthetic.example", "aud": "studio", "sub": "synthetic-owner", "exp": time.Now().Add(time.Minute).Unix(), "tenant": "studio", "user_id": 1, "scope": "studio:read",
	}).SignedString(private)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	applicationPID := os.Getpid()
	ready := make(chan studiohost.Endpoints, 1)
	done := make(chan error, 1)
	if binary := os.Getenv("STUDIO_UNIFIED_BINARY"); binary != "" {
		ui := filepath.Join(dir, "ui")
		if err := os.Mkdir(ui, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ui, "index.html"), []byte("<html>synthetic unified Studio</html>"), 0600); err != nil {
			t.Fatal(err)
		}
		document["StaticRoot"] = ui
		raw, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "studio.json")
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		command := exec.Command(binary, "--config", path)
		var diagnostics bytes.Buffer
		command.Stderr = &diagnostics
		stdout, err := command.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		applicationPID = command.Process.Pid
		go func() { <-ctx.Done(); _ = command.Process.Signal(os.Interrupt) }()
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				if !strings.HasPrefix(line, "Studio ready: ") {
					continue
				}
				var e studiohost.Endpoints
				values := map[string]string{}
				for _, item := range strings.Fields(strings.TrimPrefix(line, "Studio ready: ")) {
					key, value, ok := strings.Cut(item, "=")
					if ok {
						values[key] = value
					}
				}
				pid, err := strconv.Atoi(values["pid"])
				if err != nil || pid != command.Process.Pid {
					cancel()
					return
				}
				e.HTTP = values["HTTP"]
				e.AdminMCP = values["adminMCP"]
				e.NamespaceMCP = values["namespaceMCP"]
				ready <- e
			}
		}()
		go func() {
			err := command.Wait()
			if err != nil {
				err = fmt.Errorf("application: %w: %s", err, diagnostics.String())
			}
			done <- err
		}()
	} else {
		go func() {
			done <- studiohost.Serve(ctx, config, studiohost.Options{Diagnostics: os.Stderr, Browser: studioapi.Options{StaticFS: fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html>synthetic unified Studio</html>")}}}, ConfigureRuntime: func(runtime *runtimehost.Config) error {
				if runtime.CodecFactory == nil {
					return fmt.Errorf("namespace readers have no deployment JWT codec factory")
				}
				codec, err := runtime.CodecFactory.New(&xcodec.Config{Body: nativeauth.JwtClaim, SourceType: reflect.TypeFor[string](), DestinationType: reflect.TypeFor[*jwt.Claims]()})
				if err != nil {
					return err
				}
				if _, err := codec.Value(ctx, bearer); err != nil {
					return fmt.Errorf("reader rejected valid configured credential: %w", err)
				}
				for _, invalid := range []struct {
					issuer, audience, subject string
					expiry                    time.Time
				}{
					{"https://wrong.example", "studio", "synthetic-owner", time.Now().Add(time.Minute)},
					{"https://synthetic.example", "wrong", "synthetic-owner", time.Now().Add(time.Minute)},
					{"https://synthetic.example", "studio", "", time.Now().Add(time.Minute)},
					{"https://synthetic.example", "studio", "synthetic-owner", time.Now().Add(-time.Minute)},
				} {
					token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{"iss": invalid.issuer, "aud": invalid.audience, "sub": invalid.subject, "exp": invalid.expiry.Unix()}).SignedString(private)
					if err != nil {
						return err
					}
					if _, err := codec.Value(ctx, token); err == nil {
						return fmt.Errorf("reader accepted credential outside the deployment claim policy")
					}
				}
				if _, err := codec.Value(ctx, bearer+"x"); err == nil {
					return fmt.Errorf("reader accepted invalid JWT signature")
				}
				return nil
			}, Ready: func(e studiohost.Endpoints) { ready <- e }})
		}()
	}

	finished := false
	t.Cleanup(func() {
		cancel()
		if !finished {
			select {
			case <-done:
			case <-time.After(15 * time.Second):
				t.Error("application did not stop after failed test")
			}
		}
	})
	var endpoints studiohost.Endpoints
	select {
	case endpoints = <-ready:
	case err := <-done:
		finished = true
		t.Fatalf("startup failed: %v", err)
	case <-ctx.Done():
		t.Fatal("startup timed out")
	}
	if _, err := db.Exec(`INSERT INTO namespaces(namespace_id,name,title,owner_id,status,etag,created_at,updated_at) VALUES(?, 'synthetic','Synthetic','synthetic-owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get("http://" + endpoints.HTTP + "/")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(page), "synthetic unified Studio") {
		t.Fatalf("browser: %d %s", res.StatusCode, page)
	}
	response, err := client.Get("http://" + endpoints.HTTP + "/studio-config.json")
	if err != nil {
		t.Fatal(err)
	}
	var browserConfig map[string]any
	err = json.NewDecoder(response.Body).Decode(&browserConfig)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || browserConfig["mode"] != "authenticated" || browserConfig["apiBaseURL"] != "http://"+endpoints.HTTP {
		t.Fatalf("derived browser config: %v", err)
	}
	// Standard MCP initialization and native tools/list both use the same binary.
	for _, body := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"synthetic","version":"1"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"studio.sdk.namespaces.list","arguments":{}}}`,
	} {
		req, _ := http.NewRequest("POST", "http://"+endpoints.AdminMCP+"/mcp", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", "Bearer "+bearer)
		var message struct{ Method string }
		_ = json.Unmarshal([]byte(body), &message)
		req.Header.Set(mcpschema.HeaderMethod, message.Method)
		if message.Method == "tools/call" {
			req.Header.Set(mcpschema.HeaderName, "studio.sdk.namespaces.list")
		}
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		payload, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatalf("admin MCP: %d %s", response.StatusCode, payload)
		}
		if message.Method == "tools/call" && !strings.Contains(string(payload), "Synthetic") {
			t.Fatalf("authenticated native namespace result: %s", payload)
		}
		if message.Method == "tools/list" && !strings.Contains(string(payload), "studio.sdk.namespaces.list") {
			t.Fatalf("missing native tools: %s", payload)
		}
	}
	wrongIssuer, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{"iss": "https://other.example", "aud": "studio", "sub": "synthetic-owner", "exp": time.Now().Add(time.Minute).Unix(), "tenant": "studio", "user_id": 1}).SignedString(private)
	if err != nil {
		t.Fatal(err)
	}
	for _, credential := range []string{"", "forged", wrongIssuer} {
		req, _ := http.NewRequest("POST", "http://"+endpoints.AdminMCP+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"studio.sdk.namespaces.list","arguments":{}}}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set(mcpschema.HeaderMethod, "tools/call")
		req.Header.Set(mcpschema.HeaderName, "studio.sdk.namespaces.list")
		if credential != "" {
			req.Header.Set("Authorization", "Bearer "+credential)
		}
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		payload, _ := io.ReadAll(response.Body)
		response.Body.Close()
		denied := response.StatusCode == 401 || response.StatusCode == 403 || strings.Contains(string(payload), `"status":401`) || strings.Contains(string(payload), `"status":403`) || strings.Contains(string(payload), `"code":-32001`)
		if !denied {
			t.Fatalf("invalid native MCP credential was not denied: %d %s", response.StatusCode, payload)
		}
	}
	// The deployment owns one application process, with no child services.
	processContext, processStop := context.WithTimeout(context.Background(), 5*time.Second)
	processSnapshot, processErr := exec.CommandContext(processContext, "ps", "-A", "-o", "pid=,ppid=").Output()
	processStop()
	if processErr != nil {
		t.Fatalf("inspect application process tree: %v", processErr)
	}
	for _, line := range strings.Split(string(processSnapshot), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		ppid, _ := strconv.Atoi(fields[1])
		if os.Getenv("STUDIO_UNIFIED_BINARY") != "" && ppid == applicationPID {
			t.Fatalf("application spawned a child service: %s", line)
		}
	}
	// Unknown namespaces remain unavailable; sharing a port is not authorization.
	res, err = client.Get("http://" + endpoints.NamespaceMCP + "/namespaces/" + strings.Repeat("b", 64) + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 404 {
		t.Fatalf("unknown namespace returned %d", res.StatusCode)
	}

	// Changes in persisted namespace metadata are discovered without restart.
	if _, err := db.Exec("UPDATE namespaces SET mcp_enabled=1,mcp_port=0 WHERE namespace_id=?", strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	namespaceURL := "http://" + endpoints.NamespaceMCP + "/namespaces/" + strings.Repeat("a", 64) + "/mcp"
	discoveryDeadline := time.Now().Add(10 * time.Second)
	discovered := false
	for time.Now().Before(discoveryDeadline) {
		body := `{"jsonrpc":"2.0","id":10,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"synthetic","version":"1"}}}`
		req, _ := http.NewRequest("POST", namespaceURL, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", "Bearer "+bearer)
		req.Header.Set(mcpschema.HeaderMethod, "initialize")
		response, err := client.Do(req)
		if err == nil {
			payload, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode == 200 && strings.Contains(string(payload), "protocolVersion") {
				discovered = true
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !discovered {
		t.Fatal("database-enabled namespace did not become available on the shared listener")
	}
	// The ordinary post-login flow discovers connection metadata without
	// calling the namespace-administration get tool.
	metadata := map[string]any{"io.modelcontextprotocol/protocolVersion": mcpschema.LatestProtocolVersion, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}
	callNative := func(method string, params map[string]any) []byte {
		params["_meta"] = metadata
		raw, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 15, "method": method, "params": params})
		if err != nil {
			t.Fatal(err)
		}
		request, _ := http.NewRequest("POST", "http://"+endpoints.AdminMCP+"/mcp", bytes.NewReader(raw))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Authorization", "Bearer "+bearer)
		request.Header.Set(mcpschema.HeaderMethod, method)
		request.Header.Set("Mcp-Protocol-Version", mcpschema.LatestProtocolVersion)
		if method == "resources/read" {
			request.Header.Set(mcpschema.HeaderName, params["uri"].(string))
		} else if method == "tools/call" {
			request.Header.Set(mcpschema.HeaderName, params["name"].(string))
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		if response.StatusCode != 200 {
			t.Fatalf("native discovery %s status=%d body=%s", method, response.StatusCode, body)
		}
		return body
	}

	// Authoring failures must remain useful MCP errors rather than null payloads.
	for _, probe := range []struct {
		name      string
		arguments map[string]any
		message   string
	}{
		{"studio.sdk.components.create", map[string]any{"slug": "invalid_slug", "title": "Demo", "defaultConnectorName": "source"}, "slug must use lowercase letters"},
		{"studio.sdk.components.get", map[string]any{"id": "missing-report"}, "report not found"},
		{"studio.sdk.namespaces.get", map[string]any{"name": "missing-namespace"}, "namespace not found"},
	} {
		body := callNative("tools/call", map[string]any{"name": probe.name, "arguments": probe.arguments})
		if !bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(probe.message)) || bytes.Contains(body, []byte(`"structuredContent":null`)) {
			t.Fatalf("authoring error %s: %s", probe.name, body)
		}
	}

	// Built-in authoring guidance is a native skill, independent of any
	// component publication or namespace-owned skill.
	var skillPage struct {
		Result mcpschema.ListSkillsResult
		Error  any
	}
	listedSkill := callNative("skills/list", map[string]any{})
	if err := json.Unmarshal(listedSkill, &skillPage); err != nil || skillPage.Error != nil {
		t.Fatalf("native management skills: %s %v", listedSkill, err)
	}
	skillURI := ""
	for _, item := range skillPage.Result.Skills {
		if item.Frontmatter["name"] == "datly-studio-authoring" {
			skillURI = item.Uri
		}
	}
	if skillURI == "" {
		t.Fatalf("built-in skill absent: %s", listedSkill)
	}
	var skillResult struct {
		Result mcpschema.GetSkillResult
		Error  any
	}
	gotSkill := callNative("skills/get", map[string]any{"uri": skillURI})
	if err := json.Unmarshal(gotSkill, &skillResult); err != nil || skillResult.Error != nil {
		t.Fatalf("native skill get: %s %v", gotSkill, err)
	}
	base := strings.TrimSuffix(skillURI, "SKILL.md")
	for _, path := range []string{"SKILL.md", "references/workflow.md", "references/worked-example.md", "references/cubes-and-dictionaries.md", "references/dql.md", "references/dql.ebnf.txt", "references/builder-operations.schema.json", "references/report-consumers.md"} {
		found := false
		for _, file := range skillResult.Result.Skill.Resources.Files {
			found = found || file.Uri == base+path
		}
		if !found {
			t.Fatalf("native skill resource missing: %s", path)
		}
	}
	for _, file := range skillResult.Result.Skill.Resources.Files {
		var readResult struct {
			Result mcpschema.ReadResourceResult
			Error  any
		}
		read := callNative("resources/read", map[string]any{"uri": file.Uri})
		if err := json.Unmarshal(read, &readResult); err != nil || readResult.Error != nil || len(readResult.Result.Contents) != 1 || readResult.Result.Contents[0].Text == "" {
			t.Fatalf("native skill reference read %s: %s %v", file.Uri, read, err)
		}
	}

	verifiedBearer := bearer
	bearer = "invalid-resource-credential"
	for _, probe := range []struct {
		method string
		params map[string]any
	}{
		{"skills/list", map[string]any{}},
		{"skills/get", map[string]any{"uri": skillURI}},
		{"resources/read", map[string]any{"uri": skillURI}},
	} {
		denied := callNative(probe.method, probe.params)
		if !bytes.Contains(denied, []byte(`"error":`)) {
			t.Fatalf("unverified credential read management skill via %s: %s", probe.method, denied)
		}
	}
	bearer = verifiedBearer

	// Agents can discover the live namespace catalog through their existing admin
	// connection, including after publication changes its business tool set.
	catalogTools := callNative("tools/call", map[string]any{"name": "studio.sdk.catalog.tools.list", "arguments": map[string]any{"namespaceId": strings.Repeat("a", 64)}})
	if bytes.Contains(catalogTools, []byte(`"isError":true`)) || !bytes.Contains(catalogTools, []byte(`"tools":`)) {
		t.Fatalf("live namespace tool discovery: %s", catalogTools)
	}
	deniedTools := callNative("tools/call", map[string]any{"name": "studio.sdk.catalog.tools.list", "arguments": map[string]any{"namespaceId": strings.Repeat("b", 64)}})
	if !bytes.Contains(deniedTools, []byte(`"isError":true`)) && !bytes.Contains(deniedTools, []byte(`"error":`)) {
		t.Fatalf("unknown namespace tool discovery permitted: %s", deniedTools)
	}

	// Real admin MCP controls use namespace ownership, independently of audit viewing.
	for _, enabled := range []bool{false, true} {
		body := callNative("tools/call", map[string]any{"name": "studio.sdk.audit.configure", "arguments": map[string]any{"namespaceId": strings.Repeat("a", 64), "enabled": enabled}})
		if bytes.Contains(body, []byte(`"isError":true`)) || !bytes.Contains(body, []byte(`"canManage":true`)) {
			t.Fatalf("audit control: %s", body)
		}
		var persisted bool
		if err := db.QueryRow("SELECT audit_enabled FROM namespaces WHERE namespace_id=?", strings.Repeat("a", 64)).Scan(&persisted); err != nil || persisted != enabled {
			t.Fatalf("audit setting persistence: %v %v", persisted, err)
		}
	}
	deniedAudit := callNative("tools/call", map[string]any{"name": "studio.sdk.audit.list", "arguments": map[string]any{"namespaceId": strings.Repeat("a", 64)}})
	if !bytes.Contains(deniedAudit, []byte(`"isError":true`)) && !bytes.Contains(deniedAudit, []byte(`"error":`)) {
		t.Fatalf("manager read audit without viewer role: %s", deniedAudit)
	}
	auditDeadline := time.Now().Add(5 * time.Second)
	for {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM tool_call_audit WHERE tool_name='studio.sdk.namespaces.list' AND jwt_sub='synthetic-owner' AND response_compression='gzip' AND response_original_bytes IS NULL").Scan(&count)
		if err == nil && count > 0 {
			break
		}
		if time.Now().After(auditDeadline) {
			t.Fatalf("automatic admin MCP audit missing: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	ownerBearer := bearer
	bearer, err = jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{"iss": "https://synthetic.example", "aud": "studio", "sub": "synthetic-owner", "exp": time.Now().Add(time.Minute).Unix(), "tenant": "studio", "user_id": 1, "roles": []string{"E2E_AUDIT_VIEWER"}}).SignedString(private)
	if err != nil {
		t.Fatal(err)
	}
	csvResult := callNative("tools/call", map[string]any{"name": "studio.sdk.audit.export", "arguments": map[string]any{"namespaceId": strings.Repeat("a", 64), "filter": map[string]any{"toolName": "audit.configuration"}}})
	if bytes.Contains(csvResult, []byte(`"isError":true`)) || !bytes.Contains(csvResult, []byte(`"mediaType":"text/csv; charset=utf-8"`)) || !bytes.Contains(csvResult, []byte(`"archive":`)) {
		t.Fatalf("CSV export MCP: %s", csvResult)
	}
	bearer = ownerBearer
	deniedCSV := callNative("tools/call", map[string]any{"name": "studio.sdk.audit.export", "arguments": map[string]any{"namespaceId": strings.Repeat("a", 64)}})
	if !bytes.Contains(deniedCSV, []byte(`"isError":true`)) && !bytes.Contains(deniedCSV, []byte(`"error":`)) {
		t.Fatalf("CSV export bypassed viewer permission: %s", deniedCSV)
	}
	metadata["io.viant/resources/filter"] = map[string]any{"namespace": "ns://" + strings.Repeat("a", 64), "kind": "mcp", "limit": 1}
	connectionPage := callNative("resources/list", map[string]any{})
	var listed struct {
		Result struct {
			Resources []struct {
				URI string `json:"uri"`
			}
		}
	}
	if err := json.Unmarshal(connectionPage, &listed); err != nil || len(listed.Result.Resources) != 1 {
		t.Fatalf("native MCP reference missing: %s", connectionPage)
	}
	delete(metadata, "io.viant/resources/filter")
	uri := listed.Result.Resources[0].URI
	read := callNative("resources/read", map[string]any{"uri": uri})
	var opened struct {
		Result struct{ Contents []struct{ Text string } }
	}
	if err := json.Unmarshal(read, &opened); err != nil || len(opened.Result.Contents) != 1 {
		t.Fatalf("native MCP reference unreadable: %s", read)
	}
	var connection struct {
		Name      string
		Transport struct{ URL string }
		Auth      struct {
			WorkspaceTokenReuse string
			PassUserToken       *bool
			InlineProvider      struct {
				Clients map[string]struct{ ConfigURL string }
			}
		}
	}
	if err := json.Unmarshal([]byte(opened.Result.Contents[0].Text), &connection); err != nil {
		t.Fatal(err)
	}
	if connection.Name != "synthetic" || connection.Transport.URL != namespaceURL[:len(namespaceURL)-len("/mcp")]+"/oauth/synthetic/mcp" || connection.Auth.WorkspaceTokenReuse != "never" || connection.Auth.InlineProvider.Clients["agently"].ConfigURL != "file:///unreadable-synthetic-namespace-client.enc|blowfish://env/UNSET_SYNTHETIC_NAMESPACE_KEY" {
		t.Fatalf("incorrect namespace connection metadata: %s", opened.Result.Contents[0].Text)
	}
	tool := callNative("tools/call", map[string]any{"name": "resource/get", "arguments": map[string]any{"uri": uri}})
	var toolWire struct {
		Result struct {
			StructuredContent json.RawMessage
			IsError           bool
		}
	}
	if err := json.Unmarshal(tool, &toolWire); err != nil || toolWire.Result.IsError {
		t.Fatalf("native MCP reference tool failed: %s", tool)
	}
	var toolJSON, readJSON any
	_ = json.Unmarshal(toolWire.Result.StructuredContent, &toolJSON)
	_ = json.Unmarshal([]byte(opened.Result.Contents[0].Text), &readJSON)
	toolBytes, _ := json.Marshal(toolJSON)
	readBytes, _ := json.Marshal(readJSON)
	if !bytes.Equal(toolBytes, readBytes) {
		t.Fatal("standard and tool MCP connection definitions differ")
	}
	if _, err := db.Exec("UPDATE namespaces SET mcp_enabled=0 WHERE namespace_id=?", strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	stopped := false
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		response, err := client.Get(namespaceURL)
		if err == nil {
			response.Body.Close()
			if response.StatusCode == 404 {
				stopped = true
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !stopped {
		t.Fatal("database-disabled namespace remained available")
	}
	cancel()
	select {
	case err := <-done:
		finished = true
		if err != nil {
			t.Fatalf("shutdown: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("host failed to drain")
	}
	for _, address := range []string{endpoints.HTTP, endpoints.AdminMCP, endpoints.NamespaceMCP} {
		connection, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			connection.Close()
			t.Fatalf("owned listener remains open: %s", address)
		}
	}
	fmt.Println("one process served HTTP, admin MCP and namespace MCP; all listeners drained")
}
