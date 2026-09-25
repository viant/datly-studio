package get

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/sdk"
	xresponse "github.com/viant/xdatly/response"
)

// Finalize gives the direct Datly reader the same single-item contract as
// reports.get. Missing and inaccessible components share the 404 response.
func (output *ReportGetOutput) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*ReportGetInput]()).(*ReportGetInput)
	if !ok || input == nil {
		return fmt.Errorf("report get bound input is unavailable")
	}
	if output == nil || output.Item == nil {
		return &xresponse.Error{Code: 404, Cause: errors.New("report not found")}
	}
	if output.Item.Id != input.Id || output.Item.OwnerId == "" {
		return fmt.Errorf("report get returned a mismatched row")
	}
	report := output.Item
	output.ResponseId = report.Id
	output.Namespace = report.Namespace
	output.Slug = report.Slug
	output.Title = report.Title
	if report.Description != nil {
		output.Description = *report.Description
	}
	output.OwnerId = report.OwnerId
	output.OwnerPackage = sdk.OwnerPackageSegment(report.OwnerId)
	output.Status = report.Status
	output.DefaultConnectorName = report.DefaultConnectorName
	output.ComponentScope = report.ComponentScope
	output.ComponentName = report.ComponentName
	output.CurrentDraftVersion = report.CurrentDraftVersion
	output.Etag = report.Etag
	output.CreatedAt = report.CreatedAt
	output.UpdatedAt = report.UpdatedAt
	report.OwnerPackage = output.OwnerPackage
	return nil
}
