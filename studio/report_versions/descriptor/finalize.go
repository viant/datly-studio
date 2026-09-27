package descriptor

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	xresponse "github.com/viant/xdatly/response"
)

// Finalize projects the authorized version descriptor into the SDK DTO.
func (output *VersionDescriptorOutput) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*VersionDescriptorInput]()).(*VersionDescriptorInput)
	if !ok || input == nil {
		return fmt.Errorf("version descriptor bound input is unavailable")
	}
	if output == nil || output.Item == nil {
		return &xresponse.Error{Code: 404, Cause: errors.New("report version not found")}
	}
	if output.Item.ReportId != input.ReportId || output.Item.VersionNo != input.VersionNo {
		return fmt.Errorf("version descriptor returned a mismatched row")
	}
	output.Component = output.Item.ComponentSpecJson
	output.Types = output.Item.TypeManifestJson
	output.Resources = output.Item.ResourceManifestJson
	return nil
}
