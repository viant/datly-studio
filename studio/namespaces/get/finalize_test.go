package get

import (
	"context"
	"encoding/json"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"reflect"
	"testing"
)

func TestFinalizeCarriesNamespaceSettings(t *testing.T) {
	roles := `["forecast_reader"]`
	port := 8591
	output := &NamespaceGetOutput{Item: &NamespaceRecord{OwnerId: "alice", Name: "forecasting", Visibility: "private", AllowedRolesJson: &roles, McpEnabled: true, McpPort: &port}}
	ctx := context.WithValue(context.Background(), reflect.TypeFor[*NamespaceGetInput](), &NamespaceGetInput{Name: "forecasting"})
	if err := output.Finalize(ctx); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		NamespaceID string   `json:"namespaceId"`
		Visibility  string   `json:"visibility"`
		Roles       []string `json:"allowedRoles"`
		MCPEnabled  bool     `json:"mcpEnabled"`
		MCPPort     int      `json:"mcpPort"`
	}
	if err = json.Unmarshal(payload, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.NamespaceID != namespaceaccess.ID("alice", "forecasting") || wire.Visibility != "private" || !reflect.DeepEqual(wire.Roles, []string{"forecast_reader"}) || !wire.MCPEnabled || wire.MCPPort != port {
		t.Fatalf("incomplete namespace settings: %s", payload)
	}
	roles = `invalid`
	if err := output.Finalize(ctx); err == nil {
		t.Fatal("malformed assigned roles accepted")
	}
}
