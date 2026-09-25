package reader

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

func (input *NamespaceQueryInput) Init(context.Context) error {
	if input == nil {
		return fmt.Errorf("namespace list input is required")
	}
	input.Query = strings.TrimSpace(input.Query)
	if input.Has != nil {
		input.Has.Query = input.Has.Query && input.Query != ""
		input.Has.Status = input.Has.Status && input.Status != ""
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	input.SetLimit(limit)
	if input.Offset < 0 {
		input.SetOffset(0)
	}
	return nil
}

func (output *NamespaceQueryOutput) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*NamespaceQueryInput]()).(*NamespaceQueryInput)
	if !ok || input == nil {
		return fmt.Errorf("namespace list bound input is unavailable")
	}
	output.PageLimit = input.Limit
	output.PageOffset = input.Offset
	if output.Items == nil {
		output.Items = []*NamespaceRecord{}
	}
	return nil
}
