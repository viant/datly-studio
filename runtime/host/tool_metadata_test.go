package host

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	access "github.com/viant/authz"
	"github.com/viant/mcp-protocol/schema"
)

func TestPublishedMCPToolAttestsTenantReportAndVersion(t *testing.T) {
	policy := access.Policy{Mode: "protected", Rule: &access.Rule{Kind: "role", Value: "reader"}, EntityType: "project"}
	host, err := newScopedHost(t, map[string]string{"tasks": scopedTasksDQL}, map[string]access.Policy{"tasks": policy})
	if err != nil {
		t.Fatal(err)
	}
	client := host.token(t, "alice", []access.Entity{{Type: "project", ID: "101"}})
	encoded, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": schema.MethodToolsList,
		"params": map[string]any{"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": schema.LatestProtocolVersion, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}}})
	list := func(token string) []byte {
		t.Helper()
		request, err := http.NewRequest(http.MethodPost, "http://"+host.mcpAddr+"/mcp", bytes.NewReader(encoded))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json, text/event-stream")
		request.Header.Set(schema.HeaderProtocolVersion, schema.LatestProtocolVersion)
		request.Header.Set(schema.HeaderMethod, schema.MethodToolsList)
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("catalog status=%d", response.StatusCode)
		}
		return body
	}
	if body := list(client); strings.Contains(string(body), `"viant.datly/source"`) {
		t.Fatal("execute permission disclosed protected metadata without discover/describe permission")
	}
	resource := access.Resource{Kind: "component", ID: "tasks", Version: "1", Tenant: scopeTestTenant}
	document, err := host.store.Get(context.Background(), resource)
	if err != nil {
		t.Fatal(err)
	}
	metadataPolicy := access.Policy{Mode: "protected", Rule: &access.Rule{Kind: "role", Value: "reader"}}
	document.Policies["discover"] = metadataPolicy
	document.Policies["describe"] = metadataPolicy
	if _, err = host.store.Replace(context.Background(), document, document.Revision, "catalog-test"); err != nil {
		t.Fatal(err)
	}
	if body := list(""); strings.Contains(string(body), `"viant.datly/source"`) {
		t.Fatal("anonymous caller received protected metadata")
	}
	body := list(client)
	for _, expected := range []string{`"viant.datly/source"`, `"reportId":"tasks"`, `"tenant":"` + scopeTestTenant + `"`, `"version":1`} {
		if !strings.Contains(string(body), expected) {
			t.Fatalf("published MCP tool lacks %s: body=%s", expected, body)
		}
	}
}
