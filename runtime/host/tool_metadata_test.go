package host

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk/access"
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
	request, err := http.NewRequest(http.MethodPost, "http://"+host.mcpAddr+"/mcp", bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set(schema.HeaderProtocolVersion, schema.LatestProtocolVersion)
	request.Header.Set(schema.HeaderMethod, schema.MethodToolsList)
	request.Header.Set("Authorization", "Bearer "+client)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	for _, expected := range []string{`"viant.datly/source"`, `"reportId":"tasks"`, `"tenant":"` + scopeTestTenant + `"`, `"version":1`} {
		if response.StatusCode != http.StatusOK || !strings.Contains(string(body), expected) {
			t.Fatalf("published MCP tool lacks %s: status=%d body=%s", expected, response.StatusCode, body)
		}
	}
}
