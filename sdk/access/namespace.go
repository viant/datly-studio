package access

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/viant/authz"
)

// CheckNamespaceResource verifies persisted workspace ownership independently
// of permission administration grants. A skill belongs to its owning component.
// Tenant/action authorization remains the authz service's responsibility.
func CheckNamespaceResource(ctx context.Context, namespaceID *string, resource authz.Resource, source catalogStore, canView func(context.Context, string) bool) error {
	if namespaceID == nil {
		return nil
	}
	decoded, err := hex.DecodeString(*namespaceID)
	if err != nil || len(decoded) != 32 || *namespaceID != strings.ToLower(*namespaceID) || source == nil || canView == nil {
		return authz.ErrDenied
	}
	if resource.Kind != "component" && resource.Kind != "skill" {
		return authz.ErrDenied
	}
	for offset := 0; ; offset += 100 {
		if err := ctx.Err(); err != nil {
			return err
		}
		rows, err := source.Catalog(ctx, 100, offset)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if row == nil {
				return authz.ErrDenied
			}
			if row.Kind != resource.Kind || row.ID != resource.ID || row.Version != resource.Version {
				continue
			}
			if row.NamespaceID != *namespaceID || row.ComponentID == "" || !canView(ctx, row.ComponentID) {
				return authz.ErrDenied
			}
			return nil
		}
		if len(rows) < 100 {
			return authz.ErrDenied
		}
	}
}
