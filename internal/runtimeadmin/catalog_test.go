package runtimeadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk"
)

func TestReadCatalogPreservesNamespaceAndCallerAuthority(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		t.Run(fmt.Sprint(streaming), func(t *testing.T) {
			id := strings.Repeat("a", 64)
			var origin string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/_studio/status" {
					if r.Header.Get("X-Studio-Runtime-Token") != "deployment" || r.Header.Get("Authorization") != "" || r.URL.Query().Get("namespaceId") != id {
						t.Error("incorrect admin authority or namespace")
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"namespaceId": id, "status": "ready", "mcpUrl": origin + "/namespaces/" + id + "/mcp"})
					return
				}
				if r.URL.Path != "/namespaces/"+id+"/mcp" || r.Header.Get("Authorization") != "Bearer caller" || r.Header.Get("X-Studio-Runtime-Token") != "" {
					t.Error("incorrect MCP authority or namespace")
				}
				var input struct {
					Method string         `json:"method"`
					Params map[string]any `json:"params"`
				}
				if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
					t.Error(err)
				}
				if input.Method != "skills/list" || input.Params["cursor"] != "opaque-page" || r.Header.Get("Mcp-Method") != input.Method {
					t.Error("catalog request changed")
				}
				body := `{"jsonrpc":"2.0","id":1,"result":{"skills":[]}}`
				if streaming {
					w.Header().Set("Content-Type", "text/event-stream")
					body = "event: message\ndata: " + body + "\n\n"
				}
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			origin = server.URL
			client, err := New(origin, "deployment", server.Client())
			if err != nil {
				t.Fatal(err)
			}
			cursor := "opaque-page"
			result, err := client.ReadCatalog(sdk.WithNamespaceSelection(context.Background(), id), "caller", "skills/list", "", &cursor)
			if err != nil || string(result) != `{"skills":[]}` {
				t.Fatalf("catalog=%s error=%v", result, err)
			}
		})
	}
}

func TestReadCatalogRejectsUnscopedAndNonReadOperations(t *testing.T) {
	client, err := New("http://127.0.0.1:1", "deployment", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sdk.WithNamespaceSelection(context.Background(), strings.Repeat("a", 64))
	for _, method := range []string{"tools/call", "initialize", "resources/subscribe", "unknown"} {
		if _, err := client.ReadCatalog(ctx, "caller", method, "", nil); err == nil || !strings.Contains(err.Error(), "unsupported read-only") {
			t.Fatalf("method %s: %v", method, err)
		}
	}
	if _, err := client.ReadCatalog(context.Background(), "caller", "skills/list", "", nil); err == nil || !strings.Contains(err.Error(), "namespace selection") {
		t.Fatalf("unscoped request: %v", err)
	}
}
