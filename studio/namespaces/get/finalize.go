package get

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"reflect"

	xresponse "github.com/viant/xdatly/response"
)

// Finalize projects the authorized row to the direct namespaces.get DTO.
func (output *NamespaceGetOutput) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*NamespaceGetInput]()).(*NamespaceGetInput)
	if !ok || input == nil {
		return fmt.Errorf("namespace get bound input is unavailable")
	}
	if output == nil || output.Item == nil {
		return &xresponse.Error{Code: 404, Cause: errors.New("namespace not found")}
	}
	namespace := output.Item
	if namespace.Name != input.Name || namespace.OwnerId == "" {
		return fmt.Errorf("namespace get returned a mismatched row")
	}
	output.NamespaceID = namespace.NamespaceId
	if output.NamespaceID == "" {
		output.NamespaceID = namespaceaccess.ID(namespace.OwnerId, namespace.Name)
	}
	output.Visibility = namespace.Visibility
	if output.Visibility == "" {
		output.Visibility = namespaceaccess.Private
	}
	output.AllowedRoles = []string{}
	if namespace.AllowedRolesJson != nil {
		if err := json.Unmarshal([]byte(*namespace.AllowedRolesJson), &output.AllowedRoles); err != nil {
			return fmt.Errorf("invalid namespace roles")
		}
	}
	output.MCPEnabled = namespace.McpEnabled
	output.MCPPort = namespace.McpPort
	output.CanManage = input.Jwt != nil && input.Jwt.Subject == namespace.OwnerId
	output.OwnerId = namespace.OwnerId
	output.ResponseName = namespace.Name
	output.Title = namespace.Title
	if namespace.Description != nil {
		output.Description = *namespace.Description
	}
	output.Status = namespace.Status
	output.Etag = namespace.Etag
	output.CreatedAt = namespace.CreatedAt
	output.UpdatedAt = namespace.UpdatedAt
	return nil
}
