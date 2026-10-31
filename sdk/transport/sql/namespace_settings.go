package sqltransport

import (
	"context"
	"errors"
	"github.com/viant/datly-studio/internal/namespacevisibility"
	"github.com/viant/datly-studio/internal/settingstore"
	"github.com/viant/datly-studio/sdk"
	"strings"

	xhandler "github.com/viant/xdatly/handler"
)

func (t *Transport) namespaceSettings(ctx context.Context, operation string, input, output any) error {
	var in struct {
		Name  string                           `json:"name"`
		Input sdk.UpdateNamespaceSettingsInput `json:"input"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.Name) == "" {
		return invalid(errors.New("namespace name is required"))
	}
	namespace, err := t.namespaceValue(ctx, in.Name)
	if err != nil {
		return err
	}
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "verified Studio identity is required"}
	}
	// Recheck current persisted role grants, independent of global edit authority.
	canEdit := namespacevisibility.IsAdministrator(ctx, principal.Subject)
	if operation == sdk.OperationNamespaceSettingsUpdate {
		if !canEdit {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "namespace settings require the namespace administrator role"}
		}
		if in.Input.ETag <= 0 {
			return invalid(errors.New("positive namespace etag is required"))
		}
		if namespace.ETag != in.Input.ETag {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "namespace etag does not match"}
		}
		if err = t.Predicates.ValidateSecuritySettings(in.Input.Settings.Security); err != nil {
			return invalid(err)
		}

		values, err := settingstore.Patch(&in.Input.Settings, in.Input.RemoveKeys)
		if err != nil {
			return invalid(err)
		}
		if len(values) == 0 {
			return invalid(errors.New("supply at least one setting"))
		}
		if err = settingstore.Write(ctx, t.DB, namespace.OwnerID, namespace.Name, in.Input.ETag, principal.Subject, values); err != nil {
			var conflict *xhandler.Conflict
			if errors.As(err, &conflict) {
				return &sdk.Error{Code: sdk.ErrorConflict, Message: "namespace etag does not match", Cause: err}
			}
			return err
		}
		namespace.ETag++
	}
	value, err := settingstore.Read(ctx, t.DB, namespace.OwnerID, namespace.Name)
	if err != nil {
		return err
	}
	return assign(output, &sdk.NamespaceSettingsResult{CanEdit: canEdit, Settings: value, ETag: namespace.ETag})
}
