package reader

import (
	"context"
	"fmt"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/namespaceroles"
	"github.com/viant/datly-studio/internal/namespacevisibility"
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
	output.CanCreate = input.Jwt != nil && namespacevisibility.CanCreate(ctx, input.Jwt.Subject)
	for _, row := range output.Items {
		if row == nil {
			return fmt.Errorf("namespace list returned an empty row")
		}
		row.CanManage = input.Jwt != nil && namespacevisibility.CanManage(ctx, input.Jwt.Subject, row.OwnerId, row.AllowedRolesJson)
		if row.NamespaceId == "" {
			row.NamespaceId = namespaceaccess.ID(row.OwnerId, row.Name)
		}
		if row.Visibility == "" {
			row.Visibility = namespaceaccess.Private
		}
		grants, err := namespaceroles.Decode(row.AllowedRolesJson)
		if err != nil {
			return err
		}
		row.AllowedRoles, row.ManagerRoles = grants.View, grants.Manage
	}
	output.PageLimit = input.Limit
	output.PageOffset = input.Offset
	if output.Items == nil {
		output.Items = []*NamespaceRecord{}
	}
	return nil
}
