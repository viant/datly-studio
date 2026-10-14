package create

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCreatedNamespacePublishesIdentityAndMCPConfiguration(t *testing.T) {
	now := time.Now().UTC()
	etag := int64(1)
	port := 18891
	row := &NamespaceRecord{NamespaceId: "stable-namespace-id", OwnerId: "owner", Name: "alpha", Title: "Alpha", Visibility: "private", McpEnabled: true, McpPort: &port, AllowedRoles: []string{"viewer"}, Status: "active", Etag: &etag, CreatedAt: &now, UpdatedAt: &now}
	payload, err := json.Marshal(&NamespaceCreateOutput{Data: row})
	if err != nil {
		t.Fatal(err)
	}
	var result NamespaceResponse
	if err = json.Unmarshal(payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.NamespaceID != row.NamespaceId || !result.CanManage || result.Visibility != "private" || !result.MCPEnabled || result.MCPPort == nil || *result.MCPPort != port || len(result.AllowedRoles) != 1 || result.AllowedRoles[0] != "viewer" {
		t.Fatalf("incomplete namespace response: %s", payload)
	}
	row.AllowedRoles = nil
	payload, err = json.Marshal(&NamespaceCreateOutput{Data: row})
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err = json.Unmarshal(payload, &object); err != nil {
		t.Fatal(err)
	}
	if roles, ok := object["allowedRoles"].([]any); !ok || len(roles) != 0 {
		t.Fatalf("empty roles must be an array: %s", payload)
	}
}
