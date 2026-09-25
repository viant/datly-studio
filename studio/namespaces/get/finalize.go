package get

import (
	"context"
	"errors"
	"fmt"
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
