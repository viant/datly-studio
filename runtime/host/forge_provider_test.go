package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	windowprotocol "github.com/viant/agently-core/protocol/window"
	forgeservice "github.com/viant/agently-core/service/primitiveprovider"
	"github.com/viant/forge/backend/types"
	mcpschema "github.com/viant/mcp-protocol/schema"
)

// This POC executes the same published Datly v1 multi-connector component used
// by the runtime HTTP/MCP integration test. No consumer knows either connector.
func TestPortableWindowExecutesPublishedDatlyV1Component(t *testing.T) {
	var runtime *Service
	var component windowprotocol.ComponentBinding
	reference := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	allowed := true
	definition := &windowprotocol.Definition{ContractVersion: 1, DefinitionRevision: "records:1",
		Window: &types.Window{WindowKey: "records", View: types.View{Content: &types.Container{}}},
		DataSources: map[string]*windowprotocol.DataSource{"records": {
			ID: "records", DataSource: types.DataSource{Selectors: &types.Selectors{Data: "Records"}},
			Backend: &windowprotocol.Backend{Kind: "provider", Method: windowprotocol.FetchTool,
				Pinned: map[string]any{"windowKey": "records", "dataSourceId": "records", "definitionRevision": "records:1"}},
		}},
	}
	provider := &forgeservice.PrimitiveProvider{
		Authority: forgeservice.PrimitiveAuthorityFuncs{
			AuthenticateFunc: func(context.Context) (string, error) {
				if !allowed {
					return "", errors.New("revoked")
				}
				return "fixture:owner:account:revision1", nil
			},
			AuthorizeFunc: func(_ context.Context, _, _, key, _ string) error {
				if key != "" && key != "records" {
					return errors.New("denied")
				}
				return nil
			},
		},
		Host: forgeservice.PrimitiveHostFuncs{
			CatalogFunc: func(context.Context, *windowprotocol.CatalogInput) (*windowprotocol.Catalog, error) {
				return &windowprotocol.Catalog{ContractVersion: 1, CatalogRevision: "records:1", Windows: []windowprotocol.WindowSummary{{Key: "records", Title: "Published records"}}}, nil
			},
			DefinitionFunc: func(context.Context, *windowprotocol.DefinitionInput) (*windowprotocol.Definition, error) {
				return definition, nil
			},
			FetchFunc: func(ctx context.Context, in *windowprotocol.FetchInput) (windowprotocol.FetchOutput, error) {
				if in.WindowKey != "records" || in.DataSourceID != "records" {
					return nil, errors.New("unknown datasource")
				}
				return runtime.ExecuteComponentJSON(ctx, reference, component, in.Inputs)
			},
		},
	}
	testDynamicHost(t, false, false, dynamicHostExtension{
		configure: func(config *Config) { config.ForgeProvider = provider },
		verify: func(service *Service, address string) {
			runtime = service
			var err error
			component, err = service.ResolveComponentBinding(context.Background(), reference)
			if err != nil {
				t.Fatal(err)
			}
			call := func(tool string, arguments map[string]any) string {
				t.Helper()
				data, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": arguments, "_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": mcpschema.LatestProtocolVersion, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}}})
				if err != nil {
					t.Fatal(err)
				}
				request, err := http.NewRequest(http.MethodPost, "http://"+address+"/forge/mcp", bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Accept", "application/json, text/event-stream")
				request.Header.Set(mcpschema.HeaderProtocolVersion, mcpschema.LatestProtocolVersion)
				request.Header.Set(mcpschema.HeaderMethod, "tools/call")
				request.Header.Set(mcpschema.HeaderName, tool)
				response, err := http.DefaultClient.Do(request)
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				body, err := io.ReadAll(response.Body)
				if err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != http.StatusOK {
					t.Fatalf("status=%d body=%s", response.StatusCode, body)
				}
				return string(body)
			}
			for _, tool := range []string{windowprotocol.CatalogTool, windowprotocol.DefinitionTool} {
				if body := call(tool, map[string]any{"contractVersion": 1, "windowKey": "records"}); !strings.Contains(body, "records:1") {
					t.Fatalf("%s body=%s", tool, body)
				}
			}
			input := map[string]any{"contractVersion": 1, "windowKey": "records", "dataSourceId": "records", "definitionRevision": "records:1", "inputs": map[string]any{}}
			if body := call(windowprotocol.FetchTool, input); !strings.Contains(body, "ready") || !strings.Contains(body, "primary") {
				t.Fatalf("provider fetch=%s", body)
			}
			allowed = false
			if body := call(windowprotocol.FetchTool, input); strings.Contains(body, "ready") || !strings.Contains(body, "isError") {
				t.Fatalf("revoked fetch=%s", body)
			}
			allowed = true
			input["definitionRevision"] = "stale"
			if body := call(windowprotocol.FetchTool, input); strings.Contains(body, "ready") || !strings.Contains(body, "isError") {
				t.Fatalf("stale fetch=%s", body)
			}
		},
	})
}
