package get

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	xresponse "github.com/viant/xdatly/response"
)

// Finalize projects the generated reader row into the public connector DTO.
// The row carries configuration flags but no DSN or secret material.
func (output *ConnectorGetOutput) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*ConnectorGetInput]()).(*ConnectorGetInput)
	if !ok || input == nil {
		return fmt.Errorf("connector get bound input is unavailable")
	}
	if output == nil || output.Item == nil {
		return &xresponse.Error{Code: 404, Cause: errors.New("connector not found")}
	}
	connector := output.Item
	if connector.Name != input.Name || connector.OwnerId == "" {
		return fmt.Errorf("connector get returned a mismatched row")
	}
	output.ResponseName = connector.Name
	output.Driver = connector.Driver
	output.DsnConfigured = connector.DsnConfigured
	output.SecretConfigured = connector.SecretConfigured
	if connector.Description != nil {
		output.Description = *connector.Description
	}
	output.OwnerId = connector.OwnerId
	output.Status = connector.Status
	output.Options = append(output.Options[:0], connector.OptionsJson...)
	if connector.LastTestStatus != nil {
		output.LastTestStatus = *connector.LastTestStatus
	}
	if connector.LastTestErrorCode != nil {
		output.LastTestErrorCode = *connector.LastTestErrorCode
	}
	output.LastTestedAt = connector.LastTestedAt
	output.Etag = connector.Etag
	output.CreatedAt = connector.CreatedAt
	output.UpdatedAt = connector.UpdatedAt
	return nil
}
