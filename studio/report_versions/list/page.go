package list

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	xresponse "github.com/viant/xdatly/response"
)

func (input *VersionListInput) Init(context.Context) error {
	if input == nil || strings.TrimSpace(input.ReportId) == "" {
		return &xresponse.Error{Code: 400, Cause: errors.New("reportId is required")}
	}
	if input.Input.Limit <= 0 {
		input.Input.Limit = 50
	}
	if input.Input.Limit > 500 {
		input.Input.Limit = 500
	}
	if input.Input.Offset < 0 {
		input.Input.Offset = 0
	}
	input.SetLimit(input.Input.Limit)
	input.SetOffset(input.Input.Offset)
	return nil
}

func (output *VersionListOutput) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*VersionListInput]()).(*VersionListInput)
	if !ok || input == nil {
		return fmt.Errorf("version list bound input is unavailable")
	}
	output.PageLimit = input.Input.Limit
	output.PageOffset = input.Input.Offset
	if output.Items == nil {
		output.Items = []*Version{}
	}
	for _, item := range output.Items {
		if item == nil || item.ReportId != input.ReportId || item.VersionNo <= 0 {
			return fmt.Errorf("version list returned a mismatched row")
		}
	}
	return nil
}
