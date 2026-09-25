package reader

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/sdk"
)

// Init normalizes the public page size before Datly applies its selector.
// The same bounds are used by the existing Studio SDK transport.
func (input *Input) Init(context.Context) error {
	if input == nil {
		return fmt.Errorf("report list input is required")
	}
	if input.Has != nil {
		// The existing SDK treats empty optional filters as absent even when a
		// caller sends the JSON property explicitly.
		input.Has.Query = input.Has.Query && input.Query != ""
		input.Has.Namespace = input.Has.Namespace && input.Namespace != ""
		input.Has.Status = input.Has.Status && input.Status != ""
		input.Has.OwnerId = input.Has.OwnerId && input.OwnerId != ""
		input.Has.ConnectorName = input.Has.ConnectorName && input.ConnectorName != ""
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	input.SetLimit(limit)
	return nil
}

// Finalize projects the server-owned package name and page evidence from the
// bound input. The browser cannot supply either derived report identity.
func (output *Output) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*Input]()).(*Input)
	if !ok || input == nil {
		return fmt.Errorf("report list bound input is unavailable")
	}
	output.PageLimit = input.Limit
	output.PageOffset = input.Offset
	if output.Items == nil {
		output.Items = []*Report{}
	}
	for _, report := range output.Items {
		if report == nil || report.OwnerId == "" {
			return fmt.Errorf("report list returned a row without owner")
		}
		report.OwnerPackage = sdk.OwnerPackageSegment(report.OwnerId)
	}
	return nil
}
