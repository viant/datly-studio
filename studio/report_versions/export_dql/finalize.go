package export_dql

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	xresponse "github.com/viant/xdatly/response"
)

// Finalize converts the authorized source row into the direct SDK export DTO.
func (output *VersionExportOutput) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*VersionExportInput]()).(*VersionExportInput)
	if !ok || input == nil {
		return fmt.Errorf("version export bound input is unavailable")
	}
	if output == nil || output.Item == nil {
		return &xresponse.Error{Code: 404, Cause: errors.New("report version not found")}
	}
	if output.Item.ReportId != input.ReportId || output.Item.VersionNo != input.VersionNo {
		return fmt.Errorf("version export returned a mismatched row")
	}
	output.Dql = output.Item.Dql
	output.Complete = output.Dql != ""
	return nil
}
