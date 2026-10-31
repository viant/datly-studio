package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	primitive "github.com/viant/forge/backend/provider/contract"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/forge/backend/types"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	"github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

func TestStockClosureReferencesRejectsUnknownAndMissingMembers(t *testing.T) {
	owner := identity.ResourceURI{Kind: "window", Namespace: "team", Name: "overview"}
	entry := stockWindowEntry{uri: owner, config: ForgeWindow{Components: map[string]ComponentReference{"records": {Kind: "dynamic", ID: "records", Revision: "1"}}}}
	ref := primitive.DataSourceReference{ProviderIdentity: "fixture", Resource: identity.ResourceRef{URI: stockDatasourceURI(owner, "records").String(), Revision: identity.WorkingCandidate}}
	manifest := &types.WindowResourceEnvelope{Variants: map[string]types.WindowResourceVariant{"default": {DataSourceResources: map[string]primitive.DataSourceReference{"records": ref}}}}
	if got, err := stockClosureReferences(manifest, entry, "fixture"); err != nil || len(got) != 1 {
		t.Fatalf("valid closure references rejected: len=%d err=%v", len(got), err)
	}
	for name, mutate := range map[string]func(){
		"unknown id": func() { manifest.Variants["default"].DataSourceResources["unknown"] = ref },
		"wrong uri": func() {
			bad := ref
			bad.Resource.URI = "datasource://team/other"
			manifest.Variants["default"].DataSourceResources["records"] = bad
		},
		"wrong provider": func() {
			bad := ref
			bad.ProviderIdentity = "other"
			manifest.Variants["default"].DataSourceResources["records"] = bad
		},
		"wrong revision": func() {
			bad := ref
			bad.Resource.Revision = "2"
			manifest.Variants["default"].DataSourceResources["records"] = bad
		},
		"missing ref": func() { delete(manifest.Variants["default"].DataSourceResources, "records") },
	} {
		manifest.Variants["default"] = types.WindowResourceVariant{DataSourceResources: map[string]primitive.DataSourceReference{"records": ref}}
		mutate()
		if _, err := stockClosureReferences(manifest, entry, "fixture"); !errors.Is(err, identity.ErrResourceDenied) && !errors.Is(err, identity.ErrResourceStale) {
			t.Errorf("%s: expected fail-closed reference error, got %v", name, err)
		}
	}
}

func TestStockWindowClosureTenComponentsLocalProcessing(t *testing.T) {
	f := newStockOAuthFixture(t)
	uri := "window://team/many"
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	components := map[string]ComponentReference{}
	var authored strings.Builder
	authored.WriteString("window:\n  view:\n    content:\n      id: many\n      title: Many\ndataSources:\n")
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("records%d", i)
		components[id] = ref
		fmt.Fprintf(&authored, "  %s:\n    id: %s\n    selectors: {data: Records}\n    cardinality: many\n    parameters: []\n", id, id)
	}
	rule := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	bearer := f.token(t, f.private, "alice", []string{"reader"}, time.Now().Add(time.Minute))
	testDynamicHost(t, true, false, dynamicHostExtension{configure: func(config *Config) {
		config.Access = &ResourceAccessConfig{Tenant: "*", Provider: f.provider}
		config.Forge = &ForgeConfig{ProviderIdentity: "studio-fixture", Windows: []ForgeWindow{{URI: uri, DefinitionPath: "many-window.yaml", Components: components, PublishDataSources: true}}, Policies: []authz.Document{{Resource: authz.Resource{Kind: "window", ID: "team/many", Version: identity.WorkingCandidate, Tenant: "*"}, Revision: 1, Policies: map[string]authz.Policy{"discover": rule, "describe": rule, "execute": rule}}}, Selections: []authz.SelectionDocument{{Resource: authz.ResourceFamily{Kind: "window", ID: "team/many", Tenant: "*"}, Revision: 1, DefaultVersion: identity.WorkingCandidate}}}
		if err := os.WriteFile(filepath.Join(config.RootDir, "many-window.yaml"), []byte(authored.String()), 0600); err != nil {
			t.Fatal(err)
		}
	}, verify: func(host *Service, address string) {
		store, err := authz.NewStaticStore([]authz.Document{{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: "*"}, Revision: 1, Policies: map[string]authz.Policy{"describe": rule, "execute": rule}}})
		if err != nil {
			t.Fatal(err)
		}
		host.resourceAccess.Store = store
		ctx := oauth.WithBearer(context.Background(), bearer)
		handler := &stockResourceHandler{host: host}
		requested := identity.ResourceURI{Kind: "bundle", Namespace: "team", Name: "many"}
		compiledBefore := host.windowPrimitiveSource.closureCompiles.Load()
		if compiledBefore != 1 {
			t.Fatalf("expected one startup closure compilation, got %d", compiledBefore)
		}
		var sum time.Duration
		for i := 0; i < 5; i++ {
			started := time.Now()
			raw, err := handler.stockWindowClosure(ctx, requested)
			if err != nil {
				t.Fatal(err)
			}
			sum += time.Since(started)
			var closure stockWindowClosure
			if err := json.Unmarshal(raw, &closure); err != nil || !closure.Complete || len(closure.Entries) != 11 {
				t.Fatalf("incomplete closure: entries=%d err=%v", len(closure.Entries), err)
			}
		}
		if got := host.windowPrimitiveSource.closureCompiles.Load(); got != compiledBefore {
			t.Fatalf("read recompiled closure: before=%d after=%d", compiledBefore, got)
		}
		t.Logf("ten-component authorized provider processing mean over 5: %s", sum/5)
		var token atomic.Value
		token.Store(bearer)
		transport, err := streamable.New(ctx, "http://"+address+"/forge/mcp", streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: &token}}), streamable.WithStateless(), streamable.WithProtocolVersion(schema.LatestProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
			var message struct {
				Method string
				Params struct{ URI string }
			}
			if err := json.Unmarshal(body, &message); err != nil {
				return err
			}
			headers.Set(schema.HeaderMethod, message.Method)
			if message.Method == schema.MethodResourcesRead {
				headers.Set(schema.HeaderName, message.Params.URI)
			}
			return nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		client := mcpclient.New("closure-perf", "1", transport, mcpclient.WithProtocolVersion(schema.LatestProtocolVersion))
		defer client.Close()
		if _, err := client.Initialize(ctx); err != nil {
			t.Fatal(err)
		}
		// These are injected scenario costs, not measured TLS/RTT. The provider
		// leg is a real authorized MCP resources/read on the prepared session.
		modeled := make([]time.Duration, 10)
		for i := range modeled {
			started := time.Now()
			time.Sleep(400 * time.Millisecond)
			time.Sleep(75 * time.Millisecond)
			read, err := client.ReadResource(ctx, &schema.ReadResourceRequestParams{Uri: requested.String()})
			if err != nil || len(read.Contents) != 1 {
				t.Fatalf("bundle MCP read %d failed: %v", i, err)
			}
			modeled[i] = time.Since(started)
		}
		sort.Slice(modeled, func(i, j int) bool { return modeled[i] < modeled[j] })
		t.Logf("ten-component prepared-session MCP bundle + injected 400ms startup + 75ms RTT, n=10 p50=%s max=%s (synthetic delays, no row fetch)", modeled[len(modeled)/2], modeled[len(modeled)-1])
		oldSnapshot := host.windowPrimitiveSource.snapshot
		oldClosure, err := handler.stockWindowClosure(ctx, requested)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(host.config.RootDir, "many-window.yaml"), []byte("window: [invalid]"), 0600); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.WriteFile(filepath.Join(host.config.RootDir, "many-window.yaml"), []byte(authored.String()), 0600); err != nil {
				t.Error(err)
			}
		}()
		if err := host.windowPrimitiveSource.preload(); err == nil {
			t.Fatal("invalid candidate unexpectedly activated")
		}
		if host.windowPrimitiveSource.snapshot != oldSnapshot {
			t.Fatal("failed candidate replaced the prior snapshot")
		}
		retained, err := handler.stockWindowClosure(ctx, requested)
		if err != nil || string(retained) != string(oldClosure) {
			t.Fatalf("failed candidate changed prior valid closure: %v", err)
		}
	}})
}
