package sqltransport

import (
	"context"
	"encoding/hex"
	"github.com/viant/datly-studio/sdk"
	"strings"
)

type resolvedNamespaceKey struct{}

// resolveNamespace binds only server-read namespace metadata. A browser ID never
// grants access or supplies namespace ownership.
func (t *Transport) resolveNamespace(ctx context.Context) (context.Context, error) {
	id, present := sdk.NamespaceSelectionFromContext(ctx)
	if !present {
		return ctx, nil
	}
	decoded, err := hex.DecodeString(id)
	if err != nil || len(decoded) != 32 || id != strings.ToLower(id) {
		return nil, invalidNamespaceSelection()
	}
	if _, ok := sdk.PrincipalFromContext(ctx); !ok {
		return nil, invalidNamespaceSelection()
	}
	rows, err := t.readNamespaces(ctx, "", "", "", id, 2, 0)
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorForbidden, Message: "Namespace access is unavailable", Cause: err}
	}
	if len(rows) != 1 || rows[0] == nil || rows[0].NamespaceID != id || rows[0].Status != "active" {
		return nil, &sdk.Error{Code: sdk.ErrorForbidden, Message: "Selected namespace is unavailable"}
	}
	return context.WithValue(ctx, resolvedNamespaceKey{}, *rows[0]), nil
}
func selectedNamespace(ctx context.Context) (sdk.Namespace, bool) {
	row, ok := ctx.Value(resolvedNamespaceKey{}).(sdk.Namespace)
	return row, ok
}
func invalidNamespaceSelection() error {
	return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "A valid namespace selection is required"}
}
