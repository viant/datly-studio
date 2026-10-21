package create

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// NamespaceResponse is the stable successful wire shape of the writer result.
// Input fields remain optional where the server supplies their values.
type NamespaceResponse struct {
	CanManage    bool      `json:"canManage"`
	NamespaceID  string    `json:"namespaceId"`
	Visibility   string    `json:"visibility"`
	AllowedRoles []string  `json:"allowedRoles"`
	ManagerRoles []string  `json:"managerRoles"`
	MCPEnabled   bool      `json:"mcpEnabled"`
	MCPPort      *int      `json:"mcpPort,omitempty"`
	OwnerID      string    `json:"ownerId"`
	Name         string    `json:"name"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	Status       string    `json:"status"`
	ETag         int64     `json:"etag"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (*NamespaceCreateOutput) JSONWireType() reflect.Type {
	return reflect.TypeFor[NamespaceResponse]()
}

// MarshalJSON keeps the mutation writer's canonical Data field as the SDK's
// direct namespace DTO instead of adding a transport-only {data: ...} wrapper.
func (output *NamespaceCreateOutput) MarshalJSON() ([]byte, error) {
	if output == nil || output.Data == nil {
		return []byte("null"), nil
	}
	row := output.Data
	if row.Etag == nil || row.CreatedAt == nil || row.UpdatedAt == nil {
		return nil, fmt.Errorf("created namespace response is incomplete")
	}
	roles := append([]string{}, row.AllowedRoles...)
	result := NamespaceResponse{CanManage: true, NamespaceID: row.NamespaceId, Visibility: row.Visibility, AllowedRoles: roles, ManagerRoles: append([]string{}, row.ManagerRoles...), MCPEnabled: row.McpEnabled, MCPPort: row.McpPort, OwnerID: row.OwnerId, Name: row.Name, Title: row.Title,
		Status: row.Status, ETag: *row.Etag, CreatedAt: *row.CreatedAt, UpdatedAt: *row.UpdatedAt}
	if row.Description != nil {
		result.Description = *row.Description
	}
	return json.Marshal(result)
}
