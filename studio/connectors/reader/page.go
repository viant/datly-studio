package reader

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// Init applies the Studio SDK catalog bounds before Datly executes the view.
func (input *Input) Init(context.Context) error {
	if input == nil {
		return fmt.Errorf("connector list input is required")
	}
	input.Query = strings.TrimSpace(input.Query)
	if input.Has != nil {
		input.Has.Query = input.Has.Query && input.Query != ""
		input.Has.Status = input.Has.Status && input.Status != ""
		input.Has.OwnerID = input.Has.OwnerID && input.OwnerID != ""
		input.Has.Driver = input.Has.Driver && input.Driver != ""
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

func (output *Output) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*Input]()).(*Input)
	if !ok || input == nil {
		return fmt.Errorf("connector list bound input is unavailable")
	}
	output.PageLimit = input.Limit
	output.PageOffset = input.Offset
	if output.Items == nil {
		output.Items = []*Connector{}
	}
	return nil
}
