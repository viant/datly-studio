package get

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	xresponse "github.com/viant/xdatly/response"
)

// Finalize exposes the SDK's direct publication DTO and gives missing or
// inaccessible reports the same not-found response.
func (output *PublicationGetOutput) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*PublicationGetInput]()).(*PublicationGetInput)
	if !ok || input == nil {
		return fmt.Errorf("publication get bound input is unavailable")
	}
	if output == nil || output.Item == nil {
		return &xresponse.Error{Code: 404, Cause: errors.New("publication not found")}
	}
	item := output.Item
	if item.ReportId != input.ReportId {
		return fmt.Errorf("publication get returned a mismatched row")
	}
	output.ResponseReportId = item.ReportId
	output.ActiveVersionNo = item.ActiveVersionNo
	output.DesiredVersionNo = item.DesiredVersionNo
	output.DesiredGeneration = item.DesiredGeneration
	output.ActiveGeneration = item.ActiveGeneration
	output.Status = item.PublicationStatus
	if item.RuntimeRevision != nil {
		output.RuntimeRevision = *item.RuntimeRevision
	}
	if item.SpecHash != nil {
		output.SpecHash = *item.SpecHash
	}
	output.PublishedAt = item.PublishedAt
	return nil
}
