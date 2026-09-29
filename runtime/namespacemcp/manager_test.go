package namespacemcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	mcpserver "github.com/viant/datly/mcp/server"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/authorization"
	"github.com/viant/mcp-protocol/schema"
	protocolserver "github.com/viant/mcp-protocol/server"
)

type testService struct{ registry *protocolserver.Registry }

func (s *testService) Registry() *protocolserver.Registry { return s.registry }
func (*testService) Authorization() *authorization.Policy { return nil }
func (*testService) ReadResource(context.Context, *schema.ReadResourceRequest) (*schema.ReadResourceResult, *jsonrpc.Error) {
	return nil, jsonrpc.NewInvalidParamsError("unknown resource", nil)
}

func factory(_ context.Context, e Endpoint) (mcpserver.Config, error) {
	registry := protocolserver.NewRegistry()
	registry.RegisterTool(&protocolserver.ToolEntry{Metadata: schema.Tool{Name: "read_" + e.NamespaceID[:8]}, Handler: func(context.Context, *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
		return &schema.CallToolResult{StructuredContent: map[string]any{"namespaceId": e.NamespaceID}}, nil
	}})
	return mcpserver.Config{Service: &testService{registry}, Implementation: schema.Implementation{Name: "namespace-test", Version: "1.0"}}, nil
}
func call(t *testing.T, e Endpoint, method string, arguments map[string]any) []byte {
	t.Helper()
	params := map[string]any{"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": schema.LatestProtocolVersion, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}}
	for key, value := range arguments {
		params[key] = value
	}
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, e.URL(), bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", schema.LatestProtocolVersion)
	request.Header.Set("Mcp-Method", method)
	if name, ok := arguments["name"].(string); ok {
		request.Header.Set("Mcp-Name", name)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 200 {
		t.Fatalf("MCP response status=%d err=%v", response.StatusCode, err)
	}
	return body
}

func TestIndependentDatlyMCPPortsAndCatalogs(t *testing.T) {
	manager, err := New(factory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := manager.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	aID, bID := strings.Repeat("a", 64), strings.Repeat("b", 64)
	a, err := manager.Start(context.Background(), aID, 0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := manager.Start(context.Background(), bID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.Port == b.Port || a.NamespaceID != aID || b.NamespaceID != bID {
		t.Fatalf("namespace endpoints collided: %+v %+v", a, b)
	}
	aCatalog, bCatalog := call(t, a, "tools/list", nil), call(t, b, "tools/list", nil)
	if !bytes.Contains(aCatalog, []byte("read_aaaaaaaa")) || bytes.Contains(aCatalog, []byte("read_bbbbbbbb")) || !bytes.Contains(bCatalog, []byte("read_bbbbbbbb")) || bytes.Contains(bCatalog, []byte("read_aaaaaaaa")) {
		t.Fatal("namespace catalogs crossed listeners")
	}
	result := call(t, a, "tools/call", map[string]any{"name": "read_aaaaaaaa", "arguments": map[string]any{}})
	if !bytes.Contains(result, []byte(aID)) || bytes.Contains(result, []byte(bID)) {
		t.Fatal("tool executed with another namespace")
	}
	foreign := call(t, a, "tools/call", map[string]any{"name": "read_bbbbbbbb", "arguments": map[string]any{}})
	if !bytes.Contains(foreign, []byte("error")) {
		t.Fatal("foreign namespace tool was executable")
	}
	same, err := manager.Start(context.Background(), aID, 0)
	if err != nil || same != a {
		t.Fatal("identical start was not idempotent")
	}
	if _, err := manager.Start(context.Background(), bID, a.Port); err == nil {
		t.Fatal("port change silently replaced listener")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Stop(ctx, aID); err != nil {
		t.Fatal(err)
	}
	if _, ok := manager.Get(aID); ok {
		t.Fatal("stopped namespace reported ready")
	}
	if !bytes.Contains(call(t, b, "tools/list", nil), []byte("read_bbbbbbbb")) {
		t.Fatal("stopping A affected B")
	}
}

func TestPortConflictAndFailedBuildDoNotLoseOtherListeners(t *testing.T) {
	var builds atomic.Int32
	manager, _ := New(func(ctx context.Context, e Endpoint) (mcpserver.Config, error) { builds.Add(1); return factory(ctx, e) })
	defer manager.Close(context.Background())
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	id := strings.Repeat("c", 64)
	if _, err := manager.Start(context.Background(), id, occupied.Addr().(*net.TCPAddr).Port); err == nil {
		t.Fatal("occupied port accepted")
	}
	if builds.Load() != 0 {
		t.Fatal("factory ran before port reservation")
	}
	if _, ok := manager.Get(id); ok {
		t.Fatal("failed listener reported ready")
	}
	failing, _ := New(func(context.Context, Endpoint) (mcpserver.Config, error) {
		return mcpserver.Config{}, errors.New("build failed")
	})
	if _, err := failing.Start(context.Background(), id, 0); err == nil {
		t.Fatal("failed catalog build accepted")
	}
	if _, ok := failing.Get(id); ok {
		t.Fatal("failed build reported ready")
	}
	if err := failing.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := failing.Start(context.Background(), id, 0); err == nil {
		t.Fatal("closed manager restarted")
	}
}

func TestSlowNamespaceBuildDoesNotBlockAnotherNamespace(t *testing.T) {
	aID, bID := strings.Repeat("d", 64), strings.Repeat("e", 64)
	entered, release := make(chan struct{}), make(chan struct{})
	manager, _ := New(func(ctx context.Context, e Endpoint) (mcpserver.Config, error) {
		if e.NamespaceID == aID {
			close(entered)
			<-release
		}
		return factory(ctx, e)
	})
	defer manager.Close(context.Background())
	done := make(chan error, 1)
	go func() { _, err := manager.Start(context.Background(), aID, 0); done <- err }()
	<-entered
	bDone := make(chan error, 1)
	go func() { _, err := manager.Start(context.Background(), bID, 0); bDone <- err }()
	select {
	case err := <-bDone:
		if err != nil {
			t.Error(err)
		}
	case <-time.After(time.Second):
		t.Error("namespace B blocked behind A build")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRebindPreservesOldListenerOnFailureAndMovesOnlyOneNamespace(t *testing.T) {
	var fail atomic.Bool
	manager, _ := New(func(ctx context.Context, e Endpoint) (mcpserver.Config, error) {
		if fail.Load() {
			return mcpserver.Config{}, errors.New("catalog unavailable")
		}
		return factory(ctx, e)
	})
	defer manager.Close(context.Background())
	aID, bID := strings.Repeat("f", 64), strings.Repeat("1", 64)
	a, err := manager.Start(context.Background(), aID, 0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := manager.Start(context.Background(), bID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Rebind(context.Background(), aID, b.Port); err == nil {
		t.Fatal("occupied port replacement accepted")
	}
	current, ok := manager.Get(aID)
	if !ok || current != a {
		t.Fatal("port conflict removed existing endpoint")
	}
	fail.Store(true)
	if _, err := manager.Rebind(context.Background(), aID, 0); err == nil {
		t.Fatal("failed catalog replacement accepted")
	}
	fail.Store(false)
	if !bytes.Contains(call(t, a, "tools/list", nil), []byte("read_ffffffff")) {
		t.Fatal("failed build stopped existing endpoint")
	}
	replacement, err := manager.Rebind(context.Background(), aID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.Port == a.Port || replacement.NamespaceID != aID {
		t.Fatal("rebind did not change port")
	}
	if !bytes.Contains(call(t, replacement, "tools/list", nil), []byte("read_ffffffff")) {
		t.Fatal("replacement catalog unavailable")
	}
	if !bytes.Contains(call(t, b, "tools/list", nil), []byte("read_11111111")) {
		t.Fatal("rebind affected another namespace")
	}
	connection, err := net.DialTimeout("tcp", a.Address, 100*time.Millisecond)
	if err == nil {
		connection.Close()
		t.Fatal("old port remained open")
	}
}

type ownedSource struct {
	service *testService
	closes  *atomic.Int32
}

func (s *ownedSource) Pin(ctx context.Context) (context.Context, mcpserver.ServerService, error) {
	return ctx, s.service, nil
}
func (s *ownedSource) Close(context.Context) error { s.closes.Add(1); return nil }
func TestListenerOwnsRuntimeSourceCleanup(t *testing.T) {
	var closes atomic.Int32
	manager, _ := New(func(ctx context.Context, e Endpoint) (mcpserver.Config, error) {
		config, err := factory(ctx, e)
		if err != nil {
			return config, err
		}
		config.Source = &ownedSource{service: config.Service.(*testService), closes: &closes}
		config.Service = nil
		return config, nil
	})
	id := strings.Repeat("2", 64)
	if _, err := manager.Start(context.Background(), id, 0); err != nil {
		t.Fatal(err)
	}
	if err := manager.Stop(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if closes.Load() != 1 {
		t.Fatal("runtime source was not closed after listener drain")
	}
	if err := manager.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if closes.Load() != 1 {
		t.Fatal("runtime source closed more than once")
	}
	var failedCloses atomic.Int32
	invalid, _ := New(func(context.Context, Endpoint) (mcpserver.Config, error) {
		return mcpserver.Config{Source: &ownedSource{closes: &failedCloses}, Transport: mcpserver.TransportConfig{StreamableURI: "invalid"}}, nil
	})
	// Invalid transport configuration must close the already-built runtime source.
	defer invalid.Close(context.Background())
	if _, err := invalid.Start(context.Background(), id, 0); err == nil {
		t.Fatal("invalid source accepted")
	}
	if failedCloses.Load() != 1 {
		t.Fatal("failed transport construction leaked its source")
	}
}
