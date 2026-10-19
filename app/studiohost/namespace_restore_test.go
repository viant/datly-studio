package studiohost_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/app/hostconfig"
	"github.com/viant/datly-studio/app/studioapi"
	"github.com/viant/datly-studio/app/studiohost"
	runtimehost "github.com/viant/datly-studio/runtime/host"
)

func TestInitialNamespaceRestoreIsolation(t *testing.T) {
	mode := os.Getenv("STUDIO_RESTORE_TEST")
	if mode == "" {
		for _, mode := range []string{"namespace-failure", "global-failure"} {
			t.Run(mode, func(t *testing.T) {
				cmd := exec.Command(os.Args[0], "-test.run=^TestInitialNamespaceRestoreIsolation$", "-test.v")
				cmd.Env = append(os.Environ(), "STUDIO_RESTORE_TEST="+mode)
				out, err := cmd.CombinedOutput()
				require.NoError(t, err, string(out))
			})
		}
		return
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	root := t.TempDir()
	publicPath := filepath.Join(root, "public.pem")
	require.NoError(t, os.WriteFile(publicPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}), 0600))
	doc := map[string]any{"BaseDir": root, "Endpoint": map[string]any{"Address": "127.0.0.1:0"}, "MCP": map[string]any{"Address": "127.0.0.1:0"}, "NamespaceMCP": map[string]any{"Address": "127.0.0.1:0"}, "Store": map[string]any{"Driver": "sqlite", "DSN": filepath.Join(root, "studio.db")}, "Tenant": "studio", "JWTValidator": map[string]any{"RSA": []any{map[string]any{"URL": publicPath}}}}
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	config, err := hostconfig.Decode(raw)
	require.NoError(t, err)
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer occupied.Close()
	_, portText, err := net.SplitHostPort(occupied.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	badID, healthyID := strings.Repeat("a", 64), strings.Repeat("b", 64)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan studiohost.Endpoints, 1)
	done := make(chan error, 1)
	var held atomic.Pointer[runtimehost.NamespaceGroup]
	options := studiohost.Options{Diagnostics: io.Discard, Browser: studioapi.Options{StaticFS: fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("healthy control plane")}}}, ConfigureBrowser: func(_ *studioapi.Options, _ studiohost.Endpoints, group *runtimehost.NamespaceGroup) error {
		held.Store(group)
		if mode == "global-failure" {
			return group.DB.Close()
		}
		for _, entry := range []struct {
			id, name string
			port     int
		}{{badID, "broken", port}, {healthyID, "healthy", 0}} {
			_, err := group.DB.Exec(`INSERT INTO namespaces(namespace_id,name,title,owner_id,status,mcp_enabled,mcp_port,etag,created_at,updated_at) VALUES(?,?,?,'owner','active',1,?,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, entry.id, entry.name, entry.name, entry.port)
			if err != nil {
				return err
			}
		}
		return nil
	}, Ready: func(endpoints studiohost.Endpoints) { ready <- endpoints }}
	go func() { done <- studiohost.Serve(ctx, config, options) }()
	if mode == "global-failure" {
		select {
		case err := <-done:
			require.Error(t, err)
		case <-ready:
			t.Fatal("global snapshot failure incorrectly declared ready")
		case <-time.After(20 * time.Second):
			t.Fatal("global startup failure hung")
		}
		return
	}
	var endpoints studiohost.Endpoints
	select {
	case endpoints = <-ready:
	case err := <-done:
		t.Fatalf("one failed namespace terminated the host: %v", err)
	case <-time.After(20 * time.Second):
		t.Fatal("host readiness timeout")
	}
	defer func() { cancel(); require.NoError(t, <-done) }()
	client := &http.Client{Timeout: 5 * time.Second}
	checkUI := func() {
		response, err := client.Get("http://" + endpoints.HTTP + "/studio-config.json")
		require.NoError(t, err)
		response.Body.Close()
		require.Equal(t, http.StatusOK, response.StatusCode)
	}
	checkUI()
	bearer, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{"iss": "https://signed.example", "sub": "owner", "account_id": 42, "exp": time.Now().Add(time.Minute).Unix()}).SignedString(key)
	require.NoError(t, err)
	checkMCP := func(endpoint string) {
		body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"restore-test","version":"1"}}}`
		request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Mcp-Method", "initialize")
		request.Header.Set("Authorization", "Bearer "+bearer)
		response, err := client.Do(request)
		require.NoError(t, err)
		defer response.Body.Close()
		raw, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode, string(raw))
		require.Contains(t, string(raw), "protocolVersion")
	}
	checkMCP("http://" + endpoints.AdminMCP + "/mcp")
	group := held.Load()
	require.NotNil(t, group)
	_, healthy := group.Listeners.Get(healthyID)
	require.True(t, healthy, "healthy namespace must restore independently")
	_, bad := group.Listeners.Get(badID)
	require.False(t, bad)
	checkMCP("http://" + endpoints.NamespaceMCP + "/namespaces/" + healthyID + "/mcp")
	require.NoError(t, occupied.Close())
	// The unchanged reconciler retries the persisted namespace after its local
	// condition is repaired; no enabling flag, ACL or cache fallback is changed.
	_, err = group.Reconcile(ctx)
	require.NoError(t, err)
	endpoint, bad := group.Listeners.Get(badID)
	require.True(t, bad)
	checkMCP(endpoint.URL())
	checkUI()
	checkMCP("http://" + endpoints.AdminMCP + "/mcp")
	var enabled int
	require.NoError(t, group.DB.QueryRow(`SELECT mcp_enabled FROM namespaces WHERE namespace_id=?`, badID).Scan(&enabled))
	require.Equal(t, 1, enabled)
}
